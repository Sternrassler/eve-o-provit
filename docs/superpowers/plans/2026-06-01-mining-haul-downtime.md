# Mining ISK/h incl. Haul-Downtime (Greedy) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rank mining ores by **effective ISK/hour** that folds in the fill→fly-to-sell/reprocess→sell cycle (greedy, one cycle from the current system), picking per ore the single in-region sell hub that maximises net ISK/h.

**Architecture:** Pure cycle math + ore-hold lookup live in `pkg/evedb/mining/cycle.go`. The mining service resolves origin system + the current ship's warp/align, memoises travel times via `navigation.CalculateTravelTime`, groups mineral buy-orders by system, computes the raw cycle (1 leg) and the best refine hub (2 legs), sets new fields and re-sorts by effective ISK/h. Rows whose origin/ship/route can't be resolved keep the gross ISK/h and get the existing `is_estimate` marker (no silent 0-downtime).

**Tech Stack:** Go 1.24, SQLite SDE, Next.js/TS, Flutter/Dart.

**Spec:** `docs/superpowers/specs/2026-06-01-mining-haul-downtime-design.md`. Follow-up (full max-ratio-cycle): **issue #158**.

**Verified facts (do not re-derive):**
- Ore-hold capacity = SDE attribute **1556** (`typeAttrs`); ships without it fall back to `types.capacity` (Hulk 22544: oreHold 11500, capacity 350; Retriever 17478: oreHold 27500; Ibis 601: no 1556, capacity 125).
- `navigation.CalculateTravelTime(db, fromSys, toSys int64, *navigation.NavigationParams, useExactFormula bool) (*navigation.RouteResult, error)` → `RouteResult{TotalSeconds, Jumps, ...}`. `NavigationParams{WarpSpeed *float64, AlignTime *float64, AvoidLowSec bool}`.
- `FittingService.GetShipFitting(ctx, characterID, shipTypeID int, accessToken) (*FittingData, error)`, `fit.Bonuses.WarpSpeedAUS`, `fit.Bonuses.AlignTime`.
- `CharacterLocation.SolarSystemID int64` (from `location.GetCharacterLocation`).
- Mining service already resolves the reprocess station system id (`mining_service.go` ~line 231, `s.names.GetSystemIDForLocation(ctx, bestStationID)`), the active ship type id (`s.location.GetActiveShipTypeID`), per-ore `oreM3h` (hull+crystal), `cmp` (CompareOre), and the per-ore best ore buy `oreLoc`.
- `MarketOrder{IsBuyOrder bool, Price float64, LocationID int64}`.

**Test DB:** `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off` from `backend/`.

---

## File Structure

- `backend/pkg/evedb/mining/cycle.go` (Create) — `OreHoldCapacity` + pure `EffectiveISKPerHour`.
- `backend/pkg/evedb/mining/cycle_test.go` (Create).
- `backend/internal/models/mining.go` (Modify) — 6 new `OreRankRow` fields.
- `backend/internal/services/mining_service.go` (Modify) — provider extension, nav params, travel memo, per-system buy grouping, per-ore raw+refine cycle, fields, sort.
- `backend/internal/services/mining_service_test.go` (Modify) — fakes + assertions.
- `frontend/src/types/trading.ts`, `frontend/src/components/trading/OreRankingTable.tsx`, `frontend/tests/components/OreRankingTable.test.tsx` (Modify).
- `app/lib/api/mining_models.dart`, `app/lib/features/mining/mining_screen.dart`, `app/test/mining_models_test.dart`, `app/test/mining_screen_layout_test.dart` (Modify).

---

## Task 1: Ore-hold capacity + pure cycle math (`cycle.go`)

**Files:**
- Create: `backend/pkg/evedb/mining/cycle.go`
- Test: `backend/pkg/evedb/mining/cycle_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/pkg/evedb/mining/cycle_test.go
package mining

import (
	"math"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

func approxEqCycle(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestOreHoldCapacity(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	// Hulk (22544): ore hold 11500 (attr 1556), not the 350 cargo.
	cap, found, err := OreHoldCapacity(db, 22544)
	if err != nil || !found {
		t.Fatalf("Hulk ore hold: found=%v err=%v", found, err)
	}
	if cap != 11500 {
		t.Errorf("Hulk ore hold: got %v, want 11500", cap)
	}

	// Ibis (601): no ore hold → falls back to types.capacity 125.
	cap, found, err = OreHoldCapacity(db, 601)
	if err != nil || !found {
		t.Fatalf("Ibis capacity: found=%v err=%v", found, err)
	}
	if cap != 125 {
		t.Errorf("Ibis capacity: got %v, want 125", cap)
	}
}

func TestEffectiveISKPerHour(t *testing.T) {
	// oreHold 12000 m³, rate 12000 m³/h → fill = 1 h. netPerM3 = 100 → load = 1.2M ISK.
	// travel 600 s + stop 75 s = 675 s = 0.1875 h. cycle = 1.1875 h.
	// eff = 1_200_000 / 1.1875 = 1_010_526.315...
	eff, cycleMin, fillMin := EffectiveISKPerHour(12000, 12000, 100, 600, 75)
	if !approxEqCycle(eff, 1_200_000.0/1.1875) {
		t.Errorf("eff: got %v, want %v", eff, 1_200_000.0/1.1875)
	}
	if !approxEqCycle(cycleMin, 1.1875*60) {
		t.Errorf("cycleMin: got %v, want %v", cycleMin, 1.1875*60)
	}
	if !approxEqCycle(fillMin, 60) {
		t.Errorf("fillMin: got %v, want 60", fillMin)
	}

	// Same-system (travel 0): cycle = fill + stop only.
	eff0, cycle0, _ := EffectiveISKPerHour(12000, 12000, 100, 0, 75)
	wantCycle0 := (1.0 + 75.0/3600.0)
	if !approxEqCycle(cycle0, wantCycle0*60) {
		t.Errorf("same-system cycleMin: got %v, want %v", cycle0, wantCycle0*60)
	}
	if !approxEqCycle(eff0, 1_200_000.0/wantCycle0) {
		t.Errorf("same-system eff: got %v, want %v", eff0, 1_200_000.0/wantCycle0)
	}

	// Zero rate → zero (can't fill).
	if eff, _, _ := EffectiveISKPerHour(12000, 0, 100, 600, 75); eff != 0 {
		t.Errorf("zero rate eff: got %v, want 0", eff)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run 'TestOreHold|TestEffective' -v`
Expected: FAIL — `undefined: OreHoldCapacity`, `undefined: EffectiveISKPerHour`.

- [ ] **Step 3: Write the implementation**

```go
// backend/pkg/evedb/mining/cycle.go
package mining

import (
	"database/sql"
	"fmt"
)

const attrMiningHoldCapacity = 1556

// OreHoldCapacity returns the m³ a ship fills while mining: the dedicated ore
// hold (attr 1556) if present, otherwise the regular cargo (types.capacity).
// found=false when neither is > 0 (the caller marks the row as an estimate).
func OreHoldCapacity(db *sql.DB, shipTypeID int64) (capM3 float64, found bool, err error) {
	attrs, err := typeAttrs(db, shipTypeID)
	if err != nil {
		return 0, false, err
	}
	if v, ok := attrs[attrMiningHoldCapacity]; ok && v > 0 {
		return v, true, nil
	}
	var capacity sql.NullFloat64
	err = db.QueryRow(`SELECT capacity FROM types WHERE _key = ?`, shipTypeID).Scan(&capacity)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("ship capacity %d: %w", shipTypeID, err)
	}
	if capacity.Valid && capacity.Float64 > 0 {
		return capacity.Float64, true, nil
	}
	return 0, false, nil
}

// EffectiveISKPerHour folds the mining cycle into an hourly rate:
//
//	t_fill  = oreHoldM3 / m3h            (hours to fill the hold)
//	cycle   = t_fill + (travelSecs + stopSecs)/3600
//	eff     = (oreHoldM3 * netPerM3) / cycle
//
// travelSecs is the sum of the one-way leg seconds for the path (no return trip);
// stopSecs is the total dock/action overhead. Returns 0,0,0 when the ship cannot
// fill the hold (m3h<=0 or oreHoldM3<=0).
func EffectiveISKPerHour(oreHoldM3, m3h, netPerM3, travelSecs, stopSecs float64) (effISKPerHour, cycleMinutes, fillMinutes float64) {
	if m3h <= 0 || oreHoldM3 <= 0 {
		return 0, 0, 0
	}
	fillHours := oreHoldM3 / m3h
	cycleHours := fillHours + (travelSecs+stopSecs)/3600.0
	if cycleHours <= 0 {
		return 0, 0, 0
	}
	iskLoad := oreHoldM3 * netPerM3
	return iskLoad / cycleHours, cycleHours * 60.0, fillHours * 60.0
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run 'TestOreHold|TestEffective' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/evedb/mining/cycle.go backend/pkg/evedb/mining/cycle_test.go
git commit -m "feat(mining): ore-hold capacity + effective-ISK/h cycle math"
```

---

## Task 2: Response model fields (`models/mining.go`)

**Files:**
- Modify: `backend/internal/models/mining.go`

- [ ] **Step 1: Add fields to `OreRankRow`** (after the `EstimateReason` field from feature #1)

```go
	// Haul-downtime cycle (effective ISK/h, greedy one cycle from current system):
	EffectiveISKPerHour float64 `json:"effective_isk_per_hour,omitempty"` // sort key; 0 when not resolvable
	LoadVolumeM3        float64 `json:"load_volume_m3,omitempty"`         // ore-hold m³ filled per load
	FillMinutes         float64 `json:"fill_minutes,omitempty"`           // time to fill the hold
	CycleMinutes        float64 `json:"cycle_minutes,omitempty"`          // fill + legs + stops
	RouteJumps          int     `json:"route_jumps,omitempty"`            // jumps over the cycle's legs
	SellSystemName      string  `json:"sell_system_name,omitempty"`       // chosen sell hub (refine) / ore-sell system (raw)
```

- [ ] **Step 2: Build**

Run: `cd backend && GOWORK=off go build ./internal/models/...`
Expected: builds.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/models/mining.go
git commit -m "feat(mining): OreRankRow effective-ISK/h cycle fields"
```

---

## Task 3: Service helpers — nav params, travel memo, per-system buy grouping

**Files:**
- Modify: `backend/internal/services/mining_service.go`

- [ ] **Step 1: Extend `MiningModulesProvider` with the ship fitting accessor**

```go
// MiningModulesProvider exposes the active ship's fitted mining module type ids
// and the ship's fitting bonuses (warp/align) for the haul-downtime cycle.
type MiningModulesProvider interface {
	ActiveShipFittedModuleTypeIDs(ctx context.Context, characterID int, accessToken string) ([]int64, error)
	GetShipFitting(ctx context.Context, characterID, shipTypeID int, accessToken string) (*FittingData, error)
}
```

`*FittingService` already implements `GetShipFitting`, so the container wiring is unchanged.

- [ ] **Step 2: Add a travel-time memo + a per-system best-buy helper at the bottom of the file**

```go
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
```

- [ ] **Step 3: Add the `navigation` import** if not present

Ensure the import block has `"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"`.

- [ ] **Step 4: Build**

Run: `cd backend && GOWORK=off go build ./internal/services/... ./cmd/...`
Expected: builds (FittingService satisfies the extended interface).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/mining_service.go
git commit -m "feat(mining): travel-time memo + per-system best-buy helper"
```

---

## Task 4: Wire the cycle into per-ore ranking + re-sort

**Files:**
- Modify: `backend/internal/services/mining_service.go`

- [ ] **Step 1: Resolve origin system, ship nav params, ore hold, reprocess system once**

After the feature-#1 block that computes `hullMul`/`hullResolved`/`crystalCapable` and before "4. Best reprocessing station", add:

```go
	const oreStopSecs = 75.0 // fixed dock/action overhead per stop (shown in UI)

	// Cycle inputs: origin system, ore-hold capacity, ship warp/align.
	cycleResolved := true
	var originSys int64
	if loc, e := s.location.GetCharacterLocation(ctx, characterID, accessToken); e == nil {
		originSys = loc.SolarSystemID
	} else {
		cycleResolved = false
	}
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
	navParams := &navigation.NavigationParams{AvoidLowSec: req.SecBand == "high"}
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
```

`hullTypeID` is the variable already declared by feature #1 (`hullTypeID, shipErr := s.location.GetActiveShipTypeID(...)`).

- [ ] **Step 2: Capture the reprocess station system id**

In the existing block that resolves the reprocess station system name (the `if sysID, e := s.names.GetSystemIDForLocation(ctx, bestStationID); e == nil { ... }`), hoist the id into an outer variable. Change:

```go
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
```

- [ ] **Step 3: Per ore — compute raw + refine cycles, pick the better path**

In the per-ore loop, after the existing `row := models.OreRankRow{...}` is built (feature #1 already sets gross ISK/h, materials, RawSell, etc.), insert the cycle computation **before** `resp.Rows = append(resp.Rows, row)` and after `row.RawSell` is set:

```go
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
		if rowResolved && reprocessSys != 0 {
			// Travel origin → reprocess (shared by all hubs of this ore).
			o2rSecs, o2rJumps, ok := s.travelSecs(originSys, reprocessSys, navParams, travelMemo)
			if !ok {
				rowResolved = false
			} else {
				// Candidate hubs = systems holding a buy order for any of this ore's minerals.
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
					// Refine value at this hub: CompareOre with per-mineral prices in sysID.
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

		// Pick the path with the higher effective ISK/h; that is the verdict now.
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
			// Could not resolve the cycle → keep gross, mark estimate (no silent 0-downtime).
			row.IsEstimate = true
			if row.EstimateReason == "" {
				row.EstimateReason = "Haul-Downtime nicht berechenbar"
			}
		}
```

Note: `oreLoc`, `oreOK`, `orePrice`, `cmp`, `net`, `oreM3h`, `stationTax`, `salesTaxRate`, `regionID` are all already in scope from feature #1's loop body. `MaterialValue`/`OreCompareInput`/`CompareOre` are the existing ore-compare helpers.

- [ ] **Step 4: Re-sort by effective ISK/h (gross fallback for estimate rows)**

Replace the existing sort closure body. Find:

```go
	sort.SliceStable(resp.Rows, func(i, j int) bool {
		if m3h > 0 {
			return maxFloat(resp.Rows[i].RawISKPerHour, resp.Rows[i].RefineISKPerHour) >
				maxFloat(resp.Rows[j].RawISKPerHour, resp.Rows[j].RefineISKPerHour)
```

and change the ranking metric to prefer effective ISK/h, falling back to gross when a row's effective value is 0 (estimate / no mining):

```go
	sortKey := func(r models.OreRankRow) float64 {
		if r.EffectiveISKPerHour > 0 {
			return r.EffectiveISKPerHour
		}
		return maxFloat(r.RawISKPerHour, r.RefineISKPerHour)
	}
	sort.SliceStable(resp.Rows, func(i, j int) bool {
		if m3h > 0 {
			return sortKey(resp.Rows[i]) > sortKey(resp.Rows[j])
```

Keep the rest of the closure (the `else` branch sorting by per-m³ when `m3h == 0`) unchanged.

- [ ] **Step 5: Build + vet + lint**

Run: `cd backend && GOWORK=off go build ./... && GOWORK=off go vet ./internal/services/... && GOWORK=off golangci-lint run ./internal/services/...`
Expected: builds; `0 issues`. (If golangci flags an ineffectual assignment on `bestEff`, initialise it as shown — it is read in the comparison.)

- [ ] **Step 6: Commit**

```bash
git add backend/internal/services/mining_service.go
git commit -m "feat(mining): rank by effective ISK/h with haul-downtime cycle"
```

---

## Task 5: Service tests (`mining_service_test.go`)

**Files:**
- Modify: `backend/internal/services/mining_service_test.go`

- [ ] **Step 1: Teach the fakes the new methods**

`fakeMiningModules` needs `GetShipFitting`; `fakeMiningLocation` already has `GetCharacterLocation` (returns `&CharacterLocation{}` — give it a system id). Update:

```go
func (f *fakeMiningModules) GetShipFitting(_ context.Context, _, _ int, _ string) (*FittingData, error) {
	return &FittingData{Bonuses: FittingBonuses{WarpSpeedAUS: 3.0, AlignTime: 6.0}}, nil
}
```

And make `fakeMiningLocation.GetCharacterLocation` return a concrete system so origin resolves:

```go
func (f fakeMiningLocation) GetCharacterLocation(_ context.Context, _ int, _ string) (*CharacterLocation, error) {
	return &CharacterLocation{SolarSystemID: 30000142}, nil // Jita
}
```

- [ ] **Step 2: Update the Veldspar test for effective ISK/h**

The Veldspar test fakes market buy orders with `LocationID: 60000123`. For routing to resolve, `fakeMiningNames.GetSystemIDForLocation` already returns `30000142`; the active ship is the Hulk (`22544`, real ore hold 11500). Origin (30000142) == sell/reprocess system (30000142) → travel 0 → cycle = fill + stops. Add assertions after the existing feature-#1 assertions:

```go
	if row.LoadVolumeM3 != 11500 {
		t.Errorf("LoadVolumeM3: got %v, want 11500 (Hulk ore hold)", row.LoadVolumeM3)
	}
	if row.EffectiveISKPerHour <= 0 {
		t.Error("EffectiveISKPerHour should be set (cycle resolvable)")
	}
	// Same-system (origin==sell==30000142): travel 0, so effective < gross only by stop overhead.
	gross := row.MiningM3PerHour * row.RawNetPerM3
	if row.Best == "raw" && row.EffectiveISKPerHour >= gross {
		t.Errorf("effective (%v) must be below gross (%v) due to stop overhead", row.EffectiveISKPerHour, gross)
	}
	if row.IsEstimate {
		t.Errorf("row should be resolvable (not estimate): %+v", *row)
	}
```

Note: the Veldspar `Best` may now be chosen by effective ISK/h. Keep the earlier `row.Best != ""` check; remove any assertion that pins `Best` to `CompareOre.Best` if present (the verdict is now cycle-based).

- [ ] **Step 3: Update the existing estimate test fake**

`TestMiningService_OreRanking_EstimateWhenShipUnknown` uses `fakeMiningLocation{shipErr: ...}` (ship lookup fails). With the cycle code, `hullResolved=false` → `cycleResolved=false` → rows are estimate (already asserted). Confirm the test still passes; the `GetShipFitting` fake is harmless here.

- [ ] **Step 4: Run the mining service tests**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestMiningService -v`
Expected: PASS (Veldspar with effective ISK/h, NoMiningSetup, EstimateWhenShipUnknown).

- [ ] **Step 5: Full backend gate**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... && gofmt -l internal pkg && GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Expected: ok; no gofmt output; `0 issues`.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/services/mining_service_test.go
git commit -m "test(mining): effective-ISK/h cycle coverage"
```

---

## Task 6: Web — types + sort-by-effective + detail fields

**Files:**
- Modify: `frontend/src/types/trading.ts`, `frontend/src/components/trading/OreRankingTable.tsx`, `frontend/tests/components/OreRankingTable.test.tsx`

- [ ] **Step 1: Add fields to `OreRankRow`** (after `estimate_reason?`)

```ts
  effective_isk_per_hour?: number;
  load_volume_m3?: number;
  fill_minutes?: number;
  cycle_minutes?: number;
  route_jumps?: number;
  sell_system_name?: string;
```

- [ ] **Step 2: Write the failing test** (rows arrive pre-sorted by backend; the table shows effective ISK/h + cycle detail)

Add to `OreRankingTable.test.tsx`:

```ts
  it("shows effective ISK/h and cycle detail when present", () => {
    const r = makeRow({
      ore_type_id: 1230, ore_name: "Veldspar", best: "refine",
      effective_isk_per_hour: 950000, raw_isk_per_hour: 500000,
      refine_isk_per_hour: 1100000, cycle_minutes: 12.5, route_jumps: 3,
      sell_system_name: "Amarr",
    });
    render(<OreRankingTable rows={[r]} />);
    expect(screen.getByTestId("ore-effective-isk")).toHaveTextContent(/ISK/);

    fireEvent.click(screen.getByTestId("ore-ranking-row"));
    const detail = screen.getByTestId("ore-ranking-detail");
    expect(within(detail).getByText(/Zyklus/)).toBeInTheDocument();
    expect(within(detail).getByText(/Amarr/)).toBeInTheDocument();
  });
```

- [ ] **Step 3: Run it (fails)**

Run: `cd frontend && npx vitest run tests/components/OreRankingTable.test.tsx`
Expected: FAIL — no `ore-effective-isk`.

- [ ] **Step 4: Render effective ISK/h + cycle detail in `OreRankingTable.tsx`**

In the summary `<tr>`, add an effective-ISK/h cell (reuse `formatISK`). Replace the "Δ ISK/h" cell value or add a column — to keep the column count stable, repurpose the existing right-most cell to show effective ISK/h when present, else the existing delta. Concretely, change the delta cell to:

```tsx
        <td data-testid="ore-effective-isk" className="px-4 py-3 text-right font-medium text-green-600 dark:text-green-400">
          {formatISK(row.effective_isk_per_hour ?? row.delta_isk_per_hour)}
        </td>
```

Update the header label for that column from `Δ ISK/h` to `ISK/h eff.`.

In the detail panel (`isRefine` and raw branches), add a cycle line near the top of each branch:

```tsx
            {row.cycle_minutes != null && (
              <div className="mb-2 text-xs text-muted-foreground">
                Zyklus {row.cycle_minutes.toFixed(1)} min · {row.route_jumps ?? 0} Jumps
                {row.sell_system_name ? ` · Verkauf in ${row.sell_system_name}` : ""}
              </div>
            )}
```

Update the legend `<p>` in `frontend/src/app/mining/page.tsx` to mention effective ISK/h, e.g. append: "ISK/h ist effektiv inkl. Füllen + Flug zum Verkauf/Reprocess (ein Zyklus ab deinem System)."

- [ ] **Step 5: Run tests + build + lint**

Run: `cd frontend && npx vitest run tests/components/OreRankingTable.test.tsx && npm run build && npx eslint src/components/trading/OreRankingTable.tsx src/types/trading.ts src/app/mining/page.tsx`
Expected: tests pass; build ok; eslint clean.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/types/trading.ts frontend/src/components/trading/OreRankingTable.tsx frontend/tests/components/OreRankingTable.test.tsx frontend/src/app/mining/page.tsx
git commit -m "feat(web): show effective ISK/h + cycle detail in ore ranking"
```

---

## Task 7: Flutter — model fields + cycle detail

**Files:**
- Modify: `app/lib/api/mining_models.dart`, `app/lib/features/mining/mining_screen.dart`, `app/test/mining_models_test.dart`, `app/test/mining_screen_layout_test.dart`

- [ ] **Step 1: Add fields to `OreRankRow` (Dart) + parse**

Constructor params + fields:

```dart
    this.effectiveIskPerHour = 0.0,
    this.loadVolumeM3 = 0.0,
    this.fillMinutes = 0.0,
    this.cycleMinutes = 0.0,
    this.routeJumps = 0,
    this.sellSystemName,
```
```dart
  final double effectiveIskPerHour;
  final double loadVolumeM3;
  final double fillMinutes;
  final double cycleMinutes;
  final int routeJumps;
  final String? sellSystemName;
```

In `fromJson`:

```dart
      effectiveIskPerHour: (json['effective_isk_per_hour'] as num?)?.toDouble() ?? 0.0,
      loadVolumeM3: (json['load_volume_m3'] as num?)?.toDouble() ?? 0.0,
      fillMinutes: (json['fill_minutes'] as num?)?.toDouble() ?? 0.0,
      cycleMinutes: (json['cycle_minutes'] as num?)?.toDouble() ?? 0.0,
      routeJumps: (json['route_jumps'] as num?)?.toInt() ?? 0,
      sellSystemName: json['sell_system_name'] as String?,
```

- [ ] **Step 2: Write the failing model test**

Add to `mining_models_test.dart` (inside the `OreRankRow.fromJson` group):

```dart
    test('parses effective-ISK/h cycle fields; defaults to zero', () {
      final r = OreRankRow.fromJson(const {
        'ore_type_id': 1230, 'ore_name': 'Veldspar', 'best': 'refine',
        'effective_isk_per_hour': 950000, 'cycle_minutes': 12.5,
        'route_jumps': 3, 'sell_system_name': 'Amarr', 'load_volume_m3': 11500,
      });
      expect(r.effectiveIskPerHour, closeTo(950000, 0.1));
      expect(r.cycleMinutes, closeTo(12.5, 0.01));
      expect(r.routeJumps, 3);
      expect(r.sellSystemName, 'Amarr');

      final d = OreRankRow.fromJson(const {'ore_type_id': 1, 'ore_name': 'X', 'best': 'raw'});
      expect(d.effectiveIskPerHour, 0.0);
      expect(d.routeJumps, 0);
    });
```

- [ ] **Step 3: Run model tests (fail→pass)**

Run: `cd app && flutter test test/mining_models_test.dart`
Expected: FAIL before Step 1, PASS after.

- [ ] **Step 4: Show effective ISK/h + cycle in `mining_screen.dart`**

In `_OreRankTile`, change the subtitle to lead with effective ISK/h when present, and add a cycle line in the expanded detail. In the subtitle `Text`, prepend:

```dart
            '${row.effectiveIskPerHour > 0 ? 'eff ${fmtIsk(row.effectiveIskPerHour)} · ' : ''}'
            'm³/h ${fmtVolume(row.miningM3PerHour)} · '
```

At the top of `_refineDetail` and `_rawDetail` column children, add (guarded):

```dart
        if (row.cycleMinutes > 0)
          Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: Text(
              'Zyklus ${row.cycleMinutes.toStringAsFixed(1)} min · ${row.routeJumps} Jumps'
              '${row.sellSystemName != null ? ' · Verkauf in ${row.sellSystemName}' : ''}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
```

Update the `_ResultPane` footer note to mention effective ISK/h (append: " ISK/h ist effektiv inkl. Füllen + Flug (ein Zyklus ab deinem System).").

- [ ] **Step 5: Write the failing widget test**

Add `effectiveIskPerHour`, `cycleMinutes`, `routeJumps`, `sellSystemName` to the Veldspar row in `_detailResponse()` and a test:

```dart
  testWidgets('Expanded row shows the cycle line', (tester) async {
    await _pumpScreen(tester, 1280, notifier: _DetailNotifier.new);
    await tester.tap(find.byKey(const ValueKey('mining-ore-expand-1230')));
    await tester.pumpAndSettle();
    expect(find.textContaining('Zyklus'), findsOneWidget);
  });
```

(Set the Veldspar row's `cycleMinutes: 12.5, routeJumps: 3, effectiveIskPerHour: 950000` in `_detailResponse()`.)

- [ ] **Step 6: analyze + test**

Run: `cd app && flutter analyze lib/api/mining_models.dart lib/features/mining/mining_screen.dart test/mining_models_test.dart test/mining_screen_layout_test.dart && flutter test test/mining_models_test.dart test/mining_screen_layout_test.dart`
Expected: no issues; tests pass.

- [ ] **Step 7: Commit**

```bash
git add app/lib/api/mining_models.dart app/lib/features/mining/mining_screen.dart app/test/mining_models_test.dart app/test/mining_screen_layout_test.dart
git commit -m "feat(app): show effective ISK/h + cycle detail in ore ranking"
```

---

## Task 8: Full verification, PR, release, deploy, APK

**Files:** none (process).

- [ ] **Step 1: Full backend gate**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... && gofmt -l internal cmd pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Expected: ok; no gofmt output; `0 issues`.

- [ ] **Step 2: Full frontend gate**

Run: `cd frontend && npx vitest run && npm run build`
Expected: all tests pass; build ok.

- [ ] **Step 3: Full Flutter gate**

Run: `cd app && flutter analyze && flutter test`
Expected: no issues; all tests pass.

- [ ] **Step 4: Push branch + open PR to `main`**

```bash
git push -u origin feat/mining-haul-downtime
gh pr create --base main --title "feat(mining): rank by effective ISK/h (haul-downtime cycle)" --body "<summary: greedy 1-cycle effective ISK/h; raw 1 leg / refine best-hub 2 legs; supersedes feature #1 per-mineral-anywhere for refine; fail-loud estimate; follow-up max-ratio-cycle #158; spec+plan linked>"
```

- [ ] **Step 5: Wait for CI green, then squash-merge + delete branch**

```bash
gh pr checks <PR#> --watch --interval 20
gh pr merge <PR#> --squash --delete-branch
git checkout main && git pull --ff-only
```

- [ ] **Step 6: Release v0.23.0**

Add a `## [Unreleased]` CHANGELOG entry (Added: effective ISK/h with haul-downtime cycle; notes the refine-sell single-hub change + #158 follow-up), commit, then:

```bash
make release-check
make release VERSION=0.23.0
git add CHANGELOG.md && git commit -m "chore(release): v0.23.0"
git tag v0.23.0 && git push origin main && git push origin v0.23.0
```

- [ ] **Step 7: Watch deploy + prod smoke**

```bash
gh run watch <deploy-run-id> --interval 20 --exit-status
curl -s https://eveonline.sternrassler.de/api/v1/version   # → v0.23.0
curl -s -o /dev/null -w "%{http_code}\n" https://eveonline.sternrassler.de/mining  # → 200
```

- [ ] **Step 8: Rebuild + reinstall the Flutter APK on the tablet (Flutter changed)**

```bash
cd app && CID="$(grep '^EVE_MOBILE_CLIENT_ID=' ../deployments/.env | cut -d= -f2- | tr -d '"'\'' \t')" \
  && flutter build apk --release --dart-define=API_BASE_URL=https://eveonline.sternrassler.de --dart-define=EVE_CLIENT_ID="$CID"
adb -s R5GL3433JKE install -r build/app/outputs/flutter-apk/app-release.apk
```

Expected: `Success`. Do not print `$CID`.

---

## Notes for the implementer

- **No silent fallbacks:** effective ISK/h is computed only when origin system, ship warp/align and the needed routes resolve. Otherwise the row keeps gross ISK/h and is flagged `is_estimate` — never a silent 0-downtime.
- **One-way chain:** there is no return leg (you mine again where you sold). Raw = 1 leg (origin→ore-sell), refine = 2 legs (origin→reprocess, reprocess→best-hub).
- **Verdict is now cycle-based:** `Best` = whichever path yields higher effective ISK/h; this can differ from feature #1's per-m³ verdict, and the refine value now reflects a single sell hub (supersedes feature #1's per-mineral-anywhere). Both are intended.
- **Performance:** travel times and order-location→system are memoised per request; candidate hubs are bounded to systems that actually hold the ore's mineral buy-orders. Don't route per ore unmemoised.
