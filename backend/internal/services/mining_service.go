package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/mining"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/reprocessing"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// MiningSkillsProvider fetches the character's mining/reprocessing skills + standings.
// Abstracted so MiningService can be unit-tested without ESI (implemented by SkillsService).
type MiningSkillsProvider interface {
	GetMiningReprocessingSkills(ctx context.Context, sdeDB *sql.DB, characterID int, accessToken string) (*MiningReprocessingSkills, error)
	GetCharacterStandings(ctx context.Context, characterID int, accessToken string) (map[int64]float64, error)
}

// MiningModulesProvider exposes the active ship's fitted mining module type ids
// and the ship's fitting bonuses (warp/align) for the haul-downtime cycle.
type MiningModulesProvider interface {
	ActiveShipFittedModuleTypeIDs(ctx context.Context, characterID int, accessToken string) ([]int64, error)
	GetShipFitting(ctx context.Context, characterID, shipTypeID int, accessToken string) (*FittingData, error)
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

// MiningLocationProvider resolves the character's current location (for region fallback)
// and active ship (for the hull mining-yield bonus). Implemented by *CharacterHelper.
type MiningLocationProvider interface {
	GetCharacterLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error)
	GetActiveShipTypeID(ctx context.Context, characterID int, accessToken string) (int, error)
}

// MiningRegionProvider resolves a solar system's region id (for region fallback).
type MiningRegionProvider interface {
	GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error)
}

// MiningNameProvider resolves NPC station/system/type names for the sell + reprocess
// locations. Implemented by *database.SDERepository; kept narrow for fakeability. Includes
// the locSDE methods (used by locResolver) plus GetTypeInfo for mineral names.
type MiningNameProvider interface {
	GetStationName(ctx context.Context, stationID int64) (string, error)
	GetSystemName(ctx context.Context, systemID int64) (string, error)
	GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error)
	GetTypeInfo(ctx context.Context, typeID int) (*database.TypeInfo, error)
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
	names      MiningNameProvider
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
	names MiningNameProvider,
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
		names:      names,
		logger:     logger,
	}
}

// OreRanking computes the raw-vs-refine ore ranking for the request's region + security band.
func (s *MiningService) OreRanking(ctx context.Context, characterID int, accessToken string, req models.OreRankingRequest) (*models.OreRankingResponse, error) {
	// 1. Resolve the current system (origin for ore set + region + routing).
	curLoc, err := s.location.GetCharacterLocation(ctx, characterID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("current location unavailable: %w", err)
	}
	originSys := curLoc.SolarSystemID

	regionID := req.RegionID
	if regionID <= 0 {
		r, err := s.region.GetRegionIDForSystem(ctx, originSys)
		if err != nil {
			return nil, err
		}
		regionID = r
	}

	quarter, sec, err := mining.SystemQuarterAndSec(s.sdeDB, originSys)
	if err != nil {
		return nil, fmt.Errorf("system ore class: %w", err)
	}
	resp := &models.OreRankingResponse{
		RegionID:       regionID,
		SystemSecurity: math.Round(sec*10) / 10,
		Quarter:        quarter,
		Rows:           []models.OreRankRow{},
	}

	// 2. Ore set that actually spawns in this system (deterministic).
	bandGroups := mining.AvailableOreGroups(quarter, sec, req.AllowLowSec)
	if len(bandGroups) == 0 {
		resp.NotAvailableReason = "In diesem System sind mit der aktuellen Auswahl keine abbaubaren Erze verfügbar (Low-Sec? Null-Sec?)."
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

	// Hull ore-mining-yield bonus (role + per-skill-level), applied to every ore.
	// hullResolved starts false: if the active ship can't be resolved it stays false,
	// so rows are marked estimate rather than assuming "no bonus" (fail-loud).
	hullMul := 1.0
	hullResolved := false
	hullTypeID, shipErr := s.location.GetActiveShipTypeID(ctx, characterID, accessToken)
	if shipErr != nil {
		if s.logger != nil {
			s.logger.Warn("ore ranking: active ship unavailable for hull bonus", "error", shipErr)
		}
	} else {
		hm, resolved, err := mining.HullMiningYieldMultiplier(s.sdeDB, int64(hullTypeID), skills.SkillLevels)
		if err != nil {
			return nil, err
		}
		hullMul = hm
		hullResolved = resolved
	}

	// Best-case T2 ore crystals apply only if a crystal-capable miner is fitted.
	crystalCapable, err := mining.CrystalCapable(s.sdeDB, moduleIDs)
	if err != nil {
		return nil, err
	}

	const oreStopSecs = 75.0 // fixed dock/action overhead per stop (shown in UI)

	// Cycle inputs: origin system, ore-hold capacity, ship warp/align.
	cycleResolved := true
	var oreHoldM3 float64
	if hullResolved && hullTypeID != 0 {
		if c, found, e := mining.OreHoldCapacity(s.sdeDB, int64(hullTypeID)); e == nil && found {
			oreHoldM3 = c
		} else {
			cycleResolved = false
		}
	} else {
		cycleResolved = false
	}
	navParams := &navigation.NavigationParams{AvoidLowSec: !req.AllowLowSec}
	if hullResolved && hullTypeID != 0 {
		if fit, e := s.fitting.GetShipFitting(ctx, characterID, hullTypeID, accessToken); e == nil && fit != nil {
			ws, at := fit.Bonuses.WarpSpeedAUS, fit.Bonuses.AlignTime
			navParams.WarpSpeed, navParams.AlignTime = &ws, &at
		} else {
			cycleResolved = false
		}
	} else {
		cycleResolved = false
	}
	travelMemo := map[travelKey]*navigation.RouteResult{}
	sysOf := map[int64]int64{} // order location → system, memoised across ores

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

	// Resolve the chosen reprocess station's name + system, and a request-scoped
	// sell-location resolver (memoizes hub stations across ores).
	loc := newLocResolver(s.names)
	var bestStationName, bestStationSystem string
	var reprocessSys int64
	if bestStationID != 0 {
		if n, e := s.names.GetStationName(ctx, bestStationID); e == nil && !strings.HasPrefix(n, "Station-") {
			bestStationName = n
		}
		if sysID, e := s.names.GetSystemIDForLocation(ctx, bestStationID); e == nil {
			reprocessSys = sysID
			if sn, e2 := s.names.GetSystemName(ctx, sysID); e2 == nil {
				bestStationSystem = sn
			}
		}
	}

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

		// Net reprocessing yield for this ore (also drives each mineral's effective qty).
		net := reprocessing.NetYield(baseRate, reprocessing.ReprocessingSkills{
			Reprocessing:           skills.Reprocessing,
			ReprocessingEfficiency: skills.ReprocessingEfficiency,
			OreProcessing:          skills.OreProcessing[o.GroupID],
		})

		// Per-ore best-case crystal multiplier (1.0 when no crystal-capable miner).
		crystalMul := 1.0
		oreIsEstimate := !hullResolved
		oreEstimateReason := ""
		if !hullResolved {
			oreEstimateReason = "Schiffs-Bonus nicht auflösbar"
		}
		if crystalCapable {
			cm, found, cErr := mining.OreCrystalMultiplierT2(s.sdeDB, o.GroupID)
			if cErr != nil {
				return nil, cErr
			}
			if found {
				crystalMul = cm
			} else {
				// Crystal-capable setup but no crystal for this ore → estimate, never silent 1.0.
				oreIsEstimate = true
				if oreEstimateReason == "" {
					oreEstimateReason = "Kein Crystal für dieses Erz"
				}
			}
		}
		oreM3h := m3h * hullMul * crystalMul

		orePrice, oreLoc, oreOK := s.highestBuyOrder(ctx, regionID, int(o.TypeID))

		// Materials: each material's highest buy order → CompareOre input + per-mineral breakdown.
		mats := make([]MaterialValue, 0, len(o.Materials))
		breakdown := make([]models.RefineMaterial, 0, len(o.Materials))
		anyMatPrice := false
		for _, m := range o.Materials {
			mp, mLoc, mOK := s.highestBuyOrder(ctx, regionID, int(m.MaterialTypeID))
			if mOK && mp > 0 {
				anyMatPrice = true
			}
			mats = append(mats, MaterialValue{Qty: m.Quantity, BuyPrice: mp})
			rm := models.RefineMaterial{
				MaterialTypeID: m.MaterialTypeID,
				MaterialName:   s.typeName(ctx, int(m.MaterialTypeID)),
				EffectiveQty:   int64(math.Floor(float64(m.Quantity) * net)),
				BuyPrice:       mp,
			}
			if mOK {
				rm.Sell = loc.resolve(ctx, mLoc)
			}
			breakdown = append(breakdown, rm)
		}

		// Skip the ore entirely if NEITHER raw nor refine has any buy-order value.
		if !oreOK && !anyMatPrice {
			continue
		}

		cmp := CompareOre(OreCompareInput{
			PortionSize:  o.PortionSize,
			OreVolumeM3:  o.VolumeM3,
			OreBuyPrice:  orePrice,
			Materials:    mats,
			NetYield:     net,
			StationTake:  stationTax,
			SalesTaxRate: salesTaxRate,
		})

		row := models.OreRankRow{
			OreTypeID:           o.TypeID,
			OreName:             allOres[i].Name, // real client name from ListOres (GetOre returns the raw "-Grade" name)
			MiningM3PerHour:     oreM3h,
			RawNetPerM3:         cmp.RawNetPerM3,
			RefineNetPerM3:      cmp.RefineNetPerM3,
			Best:                cmp.Best,
			RawISKPerHour:       oreM3h * cmp.RawNetPerM3,
			RefineISKPerHour:    oreM3h * cmp.RefineNetPerM3,
			DeltaISKPerHour:     oreM3h * cmp.DeltaPerM3,
			BestStationID:       bestStationID,
			BestStationTax:      stationTax,
			BestStationName:     bestStationName,
			BestStationSystem:   bestStationSystem,
			Materials:           breakdown,
			HullYieldMultiplier: hullMul,
			CrystalMultiplier:   crystalMul,
			IsEstimate:          oreIsEstimate,
			EstimateReason:      oreEstimateReason,
		}
		if oreOK {
			rs := loc.resolve(ctx, oreLoc)
			row.RawSell = &rs
		}

		// ---- Haul-downtime cycle (greedy): raw 1 leg, refine best-hub 2 legs ----
		rowResolved := cycleResolved && hullResolved
		var rawEff, rawCycleMin, rawFillMin float64
		var rawJumps int
		var rawSellSys int64
		if rowResolved && oreOK {
			if sid, e := s.names.GetSystemIDForLocation(ctx, oreLoc); e == nil {
				rawSellSys = sid
				if secs, jumps, ok := s.travelSecs(originSys, rawSellSys, navParams, travelMemo); ok {
					rawEff, rawCycleMin, rawFillMin = mining.EffectiveISKPerHour(oreHoldM3, oreM3h, cmp.RawNetPerM3, secs, oreStopSecs)
					rawJumps = jumps
				} else {
					rowResolved = false
				}
			} else {
				rowResolved = false
			}
		}

		var refEff, refCycleMin, refFillMin float64
		var refJumps int
		var refSellSysName string
		// A reprocess station exists but its system didn't resolve → can't route the
		// refine haul; fail-loud rather than silently publishing a raw-only verdict.
		if rowResolved && bestStationID != 0 && reprocessSys == 0 {
			rowResolved = false
		}
		if rowResolved && reprocessSys != 0 {
			o2rSecs, o2rJumps, ok := s.travelSecs(originSys, reprocessSys, navParams, travelMemo)
			if !ok {
				rowResolved = false
			} else {
				bySys := make(map[int64]map[int64]systemBuy, len(o.Materials)) // mineralType → system → buy
				candidates := map[int64]bool{}
				for _, m := range o.Materials {
					g, e := s.bestBuyBySystem(ctx, regionID, int(m.MaterialTypeID), sysOf)
					if e != nil {
						rowResolved = false
						break
					}
					bySys[m.MaterialTypeID] = g
					for sysID := range g {
						candidates[sysID] = true
					}
				}
				bestEff := -1.0
				for sysID := range candidates {
					mats := make([]MaterialValue, 0, len(o.Materials))
					for _, m := range o.Materials {
						price := bySys[m.MaterialTypeID][sysID].price // 0 if absent
						mats = append(mats, MaterialValue{Qty: m.Quantity, BuyPrice: price})
					}
					hubCmp := CompareOre(OreCompareInput{
						PortionSize: o.PortionSize, OreVolumeM3: o.VolumeM3, OreBuyPrice: orePrice,
						Materials: mats, NetYield: net, StationTake: stationTax, SalesTaxRate: salesTaxRate,
					})
					hubSecs, hubJumps, hok := s.travelSecs(reprocessSys, sysID, navParams, travelMemo)
					if !hok {
						continue
					}
					eff, cyc, fil := mining.EffectiveISKPerHour(oreHoldM3, oreM3h, hubCmp.RefineNetPerM3, o2rSecs+hubSecs, 2*oreStopSecs)
					if eff > bestEff {
						bestEff, refEff, refCycleMin, refFillMin = eff, eff, cyc, fil
						refJumps = o2rJumps + hubJumps
						if sn, e := s.names.GetSystemName(ctx, sysID); e == nil {
							refSellSysName = sn
						}
					}
				}
			}
		}

		if rowResolved && (rawEff > 0 || refEff > 0) {
			if refEff > rawEff {
				row.Best = "refine"
				row.EffectiveISKPerHour, row.CycleMinutes, row.FillMinutes = refEff, refCycleMin, refFillMin
				row.RouteJumps, row.SellSystemName = refJumps, refSellSysName
			} else {
				row.Best = "raw"
				row.EffectiveISKPerHour, row.CycleMinutes, row.FillMinutes = rawEff, rawCycleMin, rawFillMin
				row.RouteJumps = rawJumps
				if row.RawSell != nil {
					row.SellSystemName = row.RawSell.SystemName
				}
			}
			row.LoadVolumeM3 = oreHoldM3
		} else if oreM3h > 0 {
			row.IsEstimate = true
			if row.EstimateReason == "" {
				row.EstimateReason = "Haul-Downtime nicht berechenbar"
			}
		}

		resp.Rows = append(resp.Rows, row)
	}

	// 6. Sort: by isk/h when mining, else by per-m³.
	sortKey := func(r models.OreRankRow) float64 {
		if r.EffectiveISKPerHour > 0 {
			return r.EffectiveISKPerHour
		}
		return maxFloat(r.RawISKPerHour, r.RefineISKPerHour)
	}
	sort.SliceStable(resp.Rows, func(i, j int) bool {
		if m3h > 0 {
			return sortKey(resp.Rows[i]) > sortKey(resp.Rows[j])
		}
		return maxFloat(resp.Rows[i].RawNetPerM3, resp.Rows[i].RefineNetPerM3) >
			maxFloat(resp.Rows[j].RawNetPerM3, resp.Rows[j].RefineNetPerM3)
	})

	return resp, nil
}

// highestBuyPrice returns the highest buy-order price for (region, type), or 0 if none.
// highestBuyOrder returns the highest buy-order price and its location for (region, type).
// ok=false when there is no buy order.
func (s *MiningService) highestBuyOrder(ctx context.Context, regionID, typeID int) (price float64, locationID int64, ok bool) {
	orders, err := s.marketRepo.GetMarketOrders(ctx, regionID, typeID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("ore ranking: market orders fetch failed", "typeID", typeID, "error", err)
		}
		return 0, 0, false
	}
	for _, o := range orders {
		if o.IsBuyOrder && (!ok || o.Price > price) {
			price, locationID, ok = o.Price, o.LocationID, true
		}
	}
	return price, locationID, ok
}

// typeName resolves a type's name (e.g. for a refined mineral), "" on failure.
func (s *MiningService) typeName(ctx context.Context, typeID int) string {
	if ti, err := s.names.GetTypeInfo(ctx, typeID); err == nil && ti != nil {
		return ti.Name
	}
	return ""
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// travelKey memoises CalculateTravelTime results within one OreRanking request.
type travelKey struct{ from, to int64 }

// systemBuy is the best buy price for a type in one system, with the order's station.
type systemBuy struct {
	price      float64
	locationID int64
}

// bestBuyBySystem groups a type's region buy-orders by solar system and keeps the
// highest price per system (with its station location). Locations that can't be
// resolved to a system (e.g. citadels) are skipped — they can't anchor a haul leg.
func (s *MiningService) bestBuyBySystem(ctx context.Context, regionID, typeID int, sysOf map[int64]int64) (map[int64]systemBuy, error) {
	orders, err := s.marketRepo.GetMarketOrders(ctx, regionID, typeID)
	if err != nil {
		return nil, err
	}
	out := map[int64]systemBuy{}
	for _, o := range orders {
		if !o.IsBuyOrder || o.Price <= 0 {
			continue
		}
		sysID, ok := sysOf[o.LocationID]
		if !ok {
			id, e := s.names.GetSystemIDForLocation(ctx, o.LocationID)
			if e != nil {
				sysOf[o.LocationID] = 0 // memoise "unresolvable"
				continue
			}
			sysOf[o.LocationID] = id
			sysID = id
		}
		if sysID == 0 {
			continue
		}
		if cur, exists := out[sysID]; !exists || o.Price > cur.price {
			out[sysID] = systemBuy{price: o.Price, locationID: o.LocationID}
		}
	}
	return out, nil
}

// travelSecs returns one-way travel seconds + jumps between two systems, memoised.
// resolved=false when the route can't be computed (the caller marks an estimate).
func (s *MiningService) travelSecs(from, to int64, params *navigation.NavigationParams, memo map[travelKey]*navigation.RouteResult) (secs float64, jumps int, resolved bool) {
	if from == to {
		return 0, 0, true
	}
	key := travelKey{from, to}
	if r, ok := memo[key]; ok {
		if r == nil {
			return 0, 0, false
		}
		return r.TotalSeconds, r.Jumps, true
	}
	r, err := navigation.CalculateTravelTime(s.sdeDB, from, to, params, false)
	if err != nil {
		memo[key] = nil
		return 0, 0, false
	}
	memo[key] = r
	return r.TotalSeconds, r.Jumps, true
}
