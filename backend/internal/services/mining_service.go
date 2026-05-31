package services

import (
	"context"
	"database/sql"
	"sort"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/mining"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/reprocessing"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// MiningSkillsProvider fetches the character's mining/reprocessing skills + standings.
// Abstracted so MiningService can be unit-tested without ESI (implemented by SkillsService).
type MiningSkillsProvider interface {
	GetMiningReprocessingSkills(ctx context.Context, sdeDB *sql.DB, characterID int, accessToken string) (*MiningReprocessingSkills, error)
	GetCharacterStandings(ctx context.Context, characterID int, accessToken string) (map[int64]float64, error)
}

// MiningModulesProvider exposes the active ship's fitted mining module type ids.
// Subset of FittingServicer, kept narrow for fakeability.
type MiningModulesProvider interface {
	ActiveShipFittedModuleTypeIDs(ctx context.Context, characterID int, accessToken string) ([]int64, error)
}

// ReprocessStationProvider lists a region's NPC reprocessing stations. Abstracted so the
// service can be unit-tested without Postgres/SDE wiring (implemented by *database.SDERepository).
type ReprocessStationProvider interface {
	GetRegionReprocessStations(ctx context.Context, regionID int) ([]database.ReprocessStation, error)
}

// MarketBuyProvider fetches market orders for a (region, type). Subset of *database.MarketRepository.
type MarketBuyProvider interface {
	GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error)
}

// MiningLocationProvider resolves the character's current location (for region fallback).
type MiningLocationProvider interface {
	GetCharacterLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error)
}

// MiningRegionProvider resolves a solar system's region id (for region fallback).
type MiningRegionProvider interface {
	GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error)
}

// MiningService ranks asteroid ores for a region + security band by raw-vs-refine
// net ISK (per m³ and, with a mining fit, per hour). It reuses the reprocessing/mining
// evedb building blocks + the ore-compare service helpers.
type MiningService struct {
	sdeDB      *sql.DB
	stations   ReprocessStationProvider
	marketRepo MarketBuyProvider
	skills     MiningSkillsProvider
	fitting    MiningModulesProvider
	location   MiningLocationProvider
	region     MiningRegionProvider
	logger     *logger.Logger
}

// NewMiningService creates a new mining ore-ranking service.
func NewMiningService(
	sdeDB *sql.DB,
	stations ReprocessStationProvider,
	marketRepo MarketBuyProvider,
	skills MiningSkillsProvider,
	fitting MiningModulesProvider,
	location MiningLocationProvider,
	region MiningRegionProvider,
	logger *logger.Logger,
) *MiningService {
	return &MiningService{
		sdeDB:      sdeDB,
		stations:   stations,
		marketRepo: marketRepo,
		skills:     skills,
		fitting:    fitting,
		location:   location,
		region:     region,
		logger:     logger,
	}
}

// OreRanking computes the raw-vs-refine ore ranking for the request's region + security band.
func (s *MiningService) OreRanking(ctx context.Context, characterID int, accessToken string, req models.OreRankingRequest) (*models.OreRankingResponse, error) {
	// 1. Resolve region (request or current location).
	regionID := req.RegionID
	if regionID <= 0 {
		loc, err := s.location.GetCharacterLocation(ctx, characterID, accessToken)
		if err != nil {
			return nil, err
		}
		r, err := s.region.GetRegionIDForSystem(ctx, loc.SolarSystemID)
		if err != nil {
			return nil, err
		}
		regionID = r
	}

	resp := &models.OreRankingResponse{
		RegionID: regionID,
		SecBand:  req.SecBand,
		Rows:     []models.OreRankRow{},
	}

	// 2. Ore set for the band.
	bandGroups := secBandOreGroups(req.SecBand)
	if len(bandGroups) == 0 {
		return resp, nil
	}
	allOres, err := reprocessing.ListOres(s.sdeDB)
	if err != nil {
		return nil, err
	}

	// 3. Skills + standings + sales tax + mining rate.
	skills, err := s.skills.GetMiningReprocessingSkills(ctx, s.sdeDB, characterID, accessToken)
	if err != nil {
		// Degrade to zero skills (still rank ores) but log loudly.
		if s.logger != nil {
			s.logger.Warn("ore ranking: mining skills unavailable, using zero skills", "error", err)
		}
		skills = &MiningReprocessingSkills{OreProcessing: map[int64]int{}}
	}
	standings, err := s.skills.GetCharacterStandings(ctx, characterID, accessToken)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("ore ranking: standings unavailable, using neutral", "error", err)
		}
		standings = map[int64]float64{}
	}
	salesTaxRate := SalesTaxRate(skills.Accounting)

	moduleIDs, err := s.fitting.ActiveShipFittedModuleTypeIDs(ctx, characterID, accessToken)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("ore ranking: active-ship modules unavailable", "error", err)
		}
		moduleIDs = nil
	}
	m3h, err := mining.MiningRateM3PerHour(s.sdeDB, moduleIDs, mining.MiningSkills{
		Mining:       skills.Mining,
		Astrogeology: skills.Astrogeology,
	}, 1.0)
	if err != nil {
		return nil, err
	}
	if m3h == 0 {
		resp.NoMiningSetup = true
	}

	// 4. Best reprocessing station (lowest tax given the player's owner-corp standing).
	stations, err := s.stations.GetRegionReprocessStations(ctx, regionID)
	if err != nil {
		return nil, err
	}
	stStandings := make([]StationStanding, 0, len(stations))
	for _, st := range stations {
		stStandings = append(stStandings, StationStanding{
			StationID: st.StationID,
			BaseRate:  st.BaseRate,
			BaseTake:  st.BaseTake,
			Standing:  standings[st.OwnerCorpID],
		})
	}
	best := BestStation(stStandings)

	// Base reprocessing rate/take: from the best station, or NPC defaults if none.
	baseRate, baseTake := 0.50, 0.05
	var bestStationID int64
	var bestStanding float64
	if best != nil {
		baseRate, baseTake = best.BaseRate, best.BaseTake
		bestStationID = best.StationID
		bestStanding = best.Standing
	}
	stationTax := ReprocessTax(baseTake, bestStanding)

	// 5. Per ore.
	for i := range allOres {
		if !bandGroups[allOres[i].GroupID] {
			continue
		}
		o, err := reprocessing.GetOre(s.sdeDB, allOres[i].TypeID)
		if err != nil {
			if s.logger != nil {
				s.logger.Warn("ore ranking: get ore failed", "typeID", allOres[i].TypeID, "error", err)
			}
			continue
		}
		if o.VolumeM3 <= 0 || o.PortionSize <= 0 {
			continue
		}

		oreBuy := s.highestBuyPrice(ctx, regionID, int(o.TypeID))

		// Materials: each material's highest buy price. Missing material price → 0
		// (that material contributes nothing to the refine value).
		mats := make([]MaterialValue, 0, len(o.Materials))
		anyMatPrice := false
		for _, m := range o.Materials {
			bp := s.highestBuyPrice(ctx, regionID, int(m.MaterialTypeID))
			if bp > 0 {
				anyMatPrice = true
			}
			mats = append(mats, MaterialValue{Qty: m.Quantity, BuyPrice: bp})
		}

		// Skip the ore entirely if NEITHER raw nor refine has any buy-order value.
		if oreBuy <= 0 && !anyMatPrice {
			continue
		}

		oreProc := skills.OreProcessing[o.GroupID]
		net := reprocessing.NetYield(baseRate, reprocessing.ReprocessingSkills{
			Reprocessing:           skills.Reprocessing,
			ReprocessingEfficiency: skills.ReprocessingEfficiency,
			OreProcessing:          oreProc,
		})

		cmp := CompareOre(OreCompareInput{
			PortionSize:  o.PortionSize,
			OreVolumeM3:  o.VolumeM3,
			OreBuyPrice:  oreBuy,
			Materials:    mats,
			NetYield:     net,
			StationTake:  stationTax,
			SalesTaxRate: salesTaxRate,
		})

		row := models.OreRankRow{
			OreTypeID:        o.TypeID,
			OreName:          o.Name,
			MiningM3PerHour:  m3h,
			RawNetPerM3:      cmp.RawNetPerM3,
			RefineNetPerM3:   cmp.RefineNetPerM3,
			Best:             cmp.Best,
			RawISKPerHour:    m3h * cmp.RawNetPerM3,
			RefineISKPerHour: m3h * cmp.RefineNetPerM3,
			DeltaISKPerHour:  m3h * cmp.DeltaPerM3,
			BestStationID:    bestStationID,
			BestStationTax:   stationTax,
		}
		resp.Rows = append(resp.Rows, row)
	}

	// 6. Sort: by isk/h when mining, else by per-m³.
	sort.SliceStable(resp.Rows, func(i, j int) bool {
		if m3h > 0 {
			return maxFloat(resp.Rows[i].RawISKPerHour, resp.Rows[i].RefineISKPerHour) >
				maxFloat(resp.Rows[j].RawISKPerHour, resp.Rows[j].RefineISKPerHour)
		}
		return maxFloat(resp.Rows[i].RawNetPerM3, resp.Rows[i].RefineNetPerM3) >
			maxFloat(resp.Rows[j].RawNetPerM3, resp.Rows[j].RefineNetPerM3)
	})

	return resp, nil
}

// highestBuyPrice returns the highest buy-order price for (region, type), or 0 if none.
func (s *MiningService) highestBuyPrice(ctx context.Context, regionID, typeID int) float64 {
	orders, err := s.marketRepo.GetMarketOrders(ctx, regionID, typeID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("ore ranking: market orders fetch failed", "typeID", typeID, "error", err)
		}
		return 0
	}
	best := 0.0
	for _, o := range orders {
		if o.IsBuyOrder && o.Price > best {
			best = o.Price
		}
	}
	return best
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
