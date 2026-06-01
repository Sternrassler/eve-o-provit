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

	// Cycle inputs: origin system, ore-hold capacity, ship warp/align. When the
	// hull/fit can't be resolved, oreHoldM3 stays 0 (→ zero effective ISK/h) and
	// navParams keeps default warp/align — the per-ore haul-downtime degrades
	// rather than gating on a separate resolved flag.
	var oreHoldM3 float64
	if hullResolved && hullTypeID != 0 {
		c, found, e := mining.OreHoldCapacity(s.sdeDB, int64(hullTypeID))
		switch {
		case e != nil:
			if s.logger != nil {
				s.logger.Warn("ore ranking: ore-hold capacity lookup failed", "hullTypeID", hullTypeID, "error", e)
			}
		case !found:
			if s.logger != nil {
				s.logger.Warn("ore ranking: ore-hold capacity unknown for hull", "hullTypeID", hullTypeID)
			}
		default:
			oreHoldM3 = c
		}
	}
	navParams := &navigation.NavigationParams{AvoidLowSec: !req.AllowLowSec}
	if hullResolved && hullTypeID != 0 {
		if fit, e := s.fitting.GetShipFitting(ctx, characterID, hullTypeID, accessToken); e == nil && fit != nil {
			ws, at := fit.Bonuses.WarpSpeedAUS, fit.Bonuses.AlignTime
			navParams.WarpSpeed, navParams.AlignTime = &ws, &at
		} else if e != nil && s.logger != nil {
			s.logger.Warn("ore ranking: ship fitting lookup failed, using default warp/align", "hullTypeID", hullTypeID, "error", e)
		}
	}
	travelMemo := map[travelKey]*navigation.RouteResult{}
	sysOf := map[int64]int64{} // order location → system, memoised across ores

	// 4. Best reprocessing station (lowest tax given the player's owner-corp standing).
	stations, err := s.stations.GetRegionReprocessStations(ctx, regionID)
	if err != nil {
		return nil, err
	}
	stationSys := map[int64]int64{}    // stationID → system (reachable only)
	stationSecs := map[int64]float64{} // stationID → origin→station one-way secs
	stationJumps := map[int64]int{}    // stationID → origin→station jumps
	stStandings := make([]StationStanding, 0, len(stations))
	for _, st := range stations {
		sysID, e := s.names.GetSystemIDForLocation(ctx, st.StationID)
		if e != nil {
			continue // citadel/unresolvable → not reachable
		}
		secs, jumps, reachable := s.travelSecs(originSys, sysID, navParams, travelMemo)
		if !reachable {
			continue
		}
		stationSys[st.StationID] = sysID
		stationSecs[st.StationID] = secs
		stationJumps[st.StationID] = jumps
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
	var reprocessSecs float64
	var reprocessJumps int
	reprocessAvailable := false
	if bestStationID != 0 {
		reprocessSys = stationSys[bestStationID]
		reprocessSecs = stationSecs[bestStationID]
		reprocessJumps = stationJumps[bestStationID]
		reprocessAvailable = true
		if n, e := s.names.GetStationName(ctx, bestStationID); e == nil && !strings.HasPrefix(n, "Station-") {
			bestStationName = n
		}
		if sn, e := s.names.GetSystemName(ctx, reprocessSys); e == nil {
			bestStationSystem = sn
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

		// ---- RAW path (reachability-aware): best reachable ore buy order ----
		orePrice, oreLoc, _, oreSecs, oreJumps, oreOK :=
			s.bestReachableBuyOrder(ctx, regionID, int(o.TypeID), originSys, navParams, travelMemo, sysOf)
		rawCmp := CompareOre(OreCompareInput{
			PortionSize: o.PortionSize, OreVolumeM3: o.VolumeM3, OreBuyPrice: orePrice,
			Materials: nil, NetYield: net, StationTake: stationTax, SalesTaxRate: salesTaxRate,
		})
		// "Reachable" = a sell destination exists under the player's security
		// preference, independent of mining setup: per-m³ rows must still appear with
		// no miner fitted (effective ISK/h then stays 0).
		rawReachable := oreOK
		var rawEff, rawCycleMin, rawFillMin float64
		if rawReachable && oreM3h > 0 {
			rawEff, rawCycleMin, rawFillMin =
				mining.EffectiveISKPerHour(oreHoldM3, oreM3h, rawCmp.RawNetPerM3, oreSecs, oreStopSecs)
		}

		// ---- REFINE path: reachable reprocess + best reachable hub ----
		var refNet, refEff, refCycleMin, refFillMin float64
		var refJumps int
		var refSellSysName string
		var refBreakdown []models.RefineMaterial
		refineReachable := false
		if reprocessAvailable {
			bySys := make(map[int64]map[int64]systemBuy, len(o.Materials))
			candidates := map[int64]bool{}
			ok := true
			for _, m := range o.Materials {
				g, e := s.bestBuyBySystem(ctx, regionID, int(m.MaterialTypeID), sysOf)
				if e != nil {
					if s.logger != nil {
						s.logger.Warn("ore ranking: mineral market fetch failed", "typeID", m.MaterialTypeID, "error", e)
					}
					ok = false
					break
				}
				bySys[m.MaterialTypeID] = g
				for sysID := range g {
					candidates[sysID] = true
				}
			}
			if ok {
				// Pick the best reachable hub by effective ISK/h when mining, else by
				// per-m³ refine net so the per-m³ column is meaningful with no miner.
				bestScore := -math.MaxFloat64
				for sysID := range candidates {
					hubSecs, hubJumps, reachable := s.travelSecs(reprocessSys, sysID, navParams, travelMemo)
					if !reachable {
						continue
					}
					mats := make([]MaterialValue, 0, len(o.Materials))
					for _, m := range o.Materials {
						mats = append(mats, MaterialValue{Qty: m.Quantity, BuyPrice: bySys[m.MaterialTypeID][sysID].price})
					}
					hubCmp := CompareOre(OreCompareInput{
						PortionSize: o.PortionSize, OreVolumeM3: o.VolumeM3, OreBuyPrice: orePrice,
						Materials: mats, NetYield: net, StationTake: stationTax, SalesTaxRate: salesTaxRate,
					})
					var eff, cyc, fil float64
					if oreM3h > 0 {
						eff, cyc, fil = mining.EffectiveISKPerHour(
							oreHoldM3, oreM3h, hubCmp.RefineNetPerM3, reprocessSecs+hubSecs, 2*oreStopSecs)
					}
					// Rank by effective ISK/h only when it's well-defined (ore hold
					// known); otherwise eff is 0 for every hub, so fall back to
					// per-m³ net for a deterministic, meaningful choice.
					score := hubCmp.RefineNetPerM3
					if oreM3h > 0 && oreHoldM3 > 0 {
						score = eff
					}
					if score > bestScore {
						bestScore = score
						refNet = hubCmp.RefineNetPerM3
						refEff, refCycleMin, refFillMin = eff, cyc, fil
						refJumps = reprocessJumps + hubJumps
						if sn, e := s.names.GetSystemName(ctx, sysID); e == nil {
							refSellSysName = sn
						}
						// Reset length, keep backing array; the prior winner's
						// materials in it are intentionally discarded.
						refBreakdown = refBreakdown[:0]
						for _, m := range o.Materials {
							sb := bySys[m.MaterialTypeID][sysID]
							rm := models.RefineMaterial{
								MaterialTypeID: m.MaterialTypeID,
								MaterialName:   s.typeName(ctx, int(m.MaterialTypeID)),
								EffectiveQty:   int64(math.Floor(float64(m.Quantity) * net)),
								BuyPrice:       sb.price,
							}
							if sb.locationID != 0 {
								rm.Sell = loc.resolve(ctx, sb.locationID)
							}
							refBreakdown = append(refBreakdown, rm)
						}
						refineReachable = true
					}
				}
			}
		}

		// Skip the ore entirely if neither path has a reachable destination.
		if !rawReachable && !refineReachable {
			continue
		}

		// ---- Build the row from the available path(s) ----
		row := models.OreRankRow{
			OreTypeID:           o.TypeID,
			OreName:             allOres[i].Name,
			MiningM3PerHour:     oreM3h,
			RawNetPerM3:         rawCmp.RawNetPerM3,
			RefineNetPerM3:      refNet,
			RawISKPerHour:       oreM3h * rawCmp.RawNetPerM3,
			RefineISKPerHour:    oreM3h * refNet,
			BestStationTax:      stationTax,
			Materials:           refBreakdown,
			HullYieldMultiplier: hullMul,
			CrystalMultiplier:   crystalMul,
			IsEstimate:          oreIsEstimate,
			EstimateReason:      oreEstimateReason,
		}
		// Per-cycle load only means something when actually mining; for a no-miner
		// row oreHoldM3 may be a generic-cargo fallback, which would mislead.
		if oreM3h > 0 {
			row.LoadVolumeM3 = oreHoldM3
		}
		if rawReachable {
			rs := loc.resolve(ctx, oreLoc)
			row.RawSell = &rs
		}
		if refineReachable {
			row.BestStationID = bestStationID
			row.BestStationName = bestStationName
			row.BestStationSystem = bestStationSystem
		}
		// Verdict: by effective ISK/h when mining, else by per-m³ refine vs raw net.
		var refineWins bool
		switch {
		case !refineReachable:
			refineWins = false
		case !rawReachable:
			refineWins = true
		case oreM3h > 0:
			refineWins = refEff >= rawEff
		default:
			refineWins = refNet >= rawCmp.RawNetPerM3
		}
		if refineWins {
			row.Best = "refine"
			row.EffectiveISKPerHour, row.CycleMinutes, row.FillMinutes = refEff, refCycleMin, refFillMin
			row.RouteJumps, row.SellSystemName = refJumps, refSellSysName
		} else {
			row.Best = "raw"
			row.EffectiveISKPerHour, row.CycleMinutes, row.FillMinutes = rawEff, rawCycleMin, rawFillMin
			row.RouteJumps = oreJumps
			if row.RawSell != nil {
				row.SellSystemName = row.RawSell.SystemName
			}
		}
		// Delta only means something when both paths are a real choice.
		if rawReachable && refineReachable {
			row.DeltaISKPerHour = oreM3h * math.Abs(rawCmp.RawNetPerM3-refNet)
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

// bestReachableBuyOrder returns the highest buy order for a type whose station's
// system is reachable from origin under the request's security preference (params).
// Citadels (location not resolvable to a system) are skipped — they can't anchor a
// haul leg. Returns the order's price, location, system, one-way travel secs/jumps,
// and ok=false when no reachable buy order exists. Reuses sysOf (location→system)
// and travelMemo for memoisation.
func (s *MiningService) bestReachableBuyOrder(
	ctx context.Context, regionID, typeID int, origin int64,
	params *navigation.NavigationParams, travelMemo map[travelKey]*navigation.RouteResult, sysOf map[int64]int64,
) (price float64, locationID int64, sellSys int64, secs float64, jumps int, ok bool) {
	orders, err := s.marketRepo.GetMarketOrders(ctx, regionID, typeID)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("ore ranking: market orders fetch failed", "typeID", typeID, "error", err)
		}
		return 0, 0, 0, 0, 0, false
	}
	for _, o := range orders {
		// Skip strictly-lower prices; equal prices still get a reachability check so
		// a closer station offering the same price can win the jumps tiebreak below.
		if !o.IsBuyOrder || o.Price <= 0 || (ok && o.Price < price) {
			continue
		}
		sys, known := sysOf[o.LocationID]
		if !known {
			id, e := s.names.GetSystemIDForLocation(ctx, o.LocationID)
			if e != nil {
				sysOf[o.LocationID] = 0 // unresolvable (citadel) → unreachable
				continue
			}
			sysOf[o.LocationID] = id
			sys = id
		}
		if sys == 0 {
			continue
		}
		secsTo, jumpsTo, reachable := s.travelSecs(origin, sys, params, travelMemo)
		if !reachable {
			continue
		}
		// Higher price always wins; on an equal price prefer fewer jumps (less haul).
		if !ok || o.Price > price || jumpsTo < jumps {
			price, locationID, sellSys, secs, jumps, ok = o.Price, o.LocationID, sys, secsTo, jumpsTo, true
		}
	}
	return price, locationID, sellSys, secs, jumps, ok
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
