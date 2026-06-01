# Reachability-aware + decoupled haul-downtime — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The mining ranking chooses sell/reprocess destinations that are actually reachable under the player's security choice, computes the raw and refine paths independently, and only marks a row "estimate" when no path is reachable (or hull/crystal is unresolved).

**Architecture:** A `bestReachableBuyOrder` helper picks the highest buy order whose system is routable from the origin (citadels excluded). The reprocess-station list is filtered to reachable stations before `BestStation`. In the per-ore loop, raw and refine are computed independently from reachable destinations; the row resolves if either path resolves and only estimates when neither does. Backend-only — response schema unchanged.

**Tech Stack:** Go 1.24, SQLite SDE.

**Spec:** `docs/superpowers/specs/2026-06-01-mining-reachability-aware-haul-design.md`.

**Verified current code (in `backend/internal/services/mining_service.go`):**
- Before the ore loop, `originSys`, `navParams` (`AvoidLowSec = !req.AllowLowSec`), `travelMemo`, `sysOf` are all in scope. `s.travelSecs(from, to int64, params, travelMemo) (secs float64, jumps int, ok bool)` returns `(0,0,true)` for `from==to` and `false` when unroutable.
- `s.highestBuyOrder(ctx, regionID, typeID int) (price float64, locationID int64, ok bool)` returns the single global-best buy order.
- Reprocess stations: `stations` ([]database.ReprocessStation{StationID, OwnerCorpID, BaseRate, BaseTake}), `stStandings` ([]StationStanding{StationID, BaseRate, BaseTake, Standing}), `best := BestStation(stStandings)`, then `bestStationID`/`baseRate`/`baseTake`/`bestStanding`/`stationTax`, `reprocessSys` via `s.names.GetSystemIDForLocation(bestStationID)`.
- `s.names.GetSystemIDForLocation(ctx, locationID int64) (int64, error)` errors for citadels (≥1e12).
- Per ore: `bestBuyBySystem(ctx, regionID, typeID int, sysOf) (map[int64]systemBuy, error)` groups buy orders by system; `systemBuy{price, locationID}`. `loc := newLocResolver(s.names)`; `loc.resolve(ctx, locationID) models.SellLocation`. `CompareOre(OreCompareInput{...}) {RawNetPerM3, RefineNetPerM3, Best, DeltaPerM3}`. `mining.EffectiveISKPerHour(oreHoldM3, m3h, netPerM3, travelSecs, stopSecs) (eff, cycleMin, fillMin)`. `mining.OreCrystalMultiplierT2`, `hullMul`, `oreM3h`, `oreStopSecs=75`.

**Test DB:** `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off` from `backend/`.

---

## Task 1: Reachable-best-buy helper + reachable reprocess station

**Files:**
- Modify: `backend/internal/services/mining_service.go`

- [ ] **Step 1: Add `bestReachableBuyOrder` (after `highestBuyOrder`)**

```go
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
		if !o.IsBuyOrder || o.Price <= 0 || (ok && o.Price <= price) {
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
		price, locationID, sellSys, secs, jumps, ok = o.Price, o.LocationID, sys, secsTo, jumpsTo, true
	}
	return price, locationID, sellSys, secs, jumps, ok
}
```

- [ ] **Step 2: Filter the reprocess-station list to reachable stations**

Replace the reprocess-station block. The current code is:

```go
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
```

with (resolve each station's system, keep only reachable ones, remember the system + travel):

```go
	stationSys := map[int64]int64{}        // stationID → system (reachable only)
	stationSecs := map[int64]float64{}     // stationID → origin→station one-way secs
	stationJumps := map[int64]int{}        // stationID → origin→station jumps
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
```

- [ ] **Step 3: Set reprocess system/travel from the reachable best station**

Replace the reprocess name/system block. The current code is:

```go
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
```

with (reprocess is reachable by construction; capture its travel):

```go
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
```

- [ ] **Step 4: Build**

Run: `cd backend && GOWORK=off go build ./... 2>&1 | head` — if `go build` errors that `highestBuyOrder` is now unused, it is still used by the loop (Task 2 replaces that). Expect: builds (the new helper + reprocess changes compile; `go vet` on the test file is fine).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/mining_service.go
git commit -m "feat(mining): reachable-best-buy helper + reachable reprocess station"
```

---

## Task 2: Decoupled, reachability-aware per-ore loop

**Files:**
- Modify: `backend/internal/services/mining_service.go`

This replaces the per-ore loop body from the raw-buy line through the end of the haul-downtime block. Keep everything **above** `orePrice, oreLoc, oreOK := s.highestBuyOrder(...)` (net, crystal, oreM3h, hull/crystal estimate) and the loop header unchanged.

- [ ] **Step 1: Replace the raw-buy + materials + CompareOre + row-build + haul-downtime block**

Find the block starting at `orePrice, oreLoc, oreOK := s.highestBuyOrder(ctx, regionID, int(o.TypeID))` and ending at the close of the haul-downtime `} else if oreM3h > 0 { ... }` estimate block (the lines that set `row.IsEstimate`/`row.EstimateReason` from the cycle), and replace the WHOLE thing with:

```go
		// ---- RAW path (reachability-aware): best reachable ore buy order ----
		orePrice, oreLoc, _, oreSecs, oreJumps, oreOK :=
			s.bestReachableBuyOrder(ctx, regionID, int(o.TypeID), originSys, navParams, travelMemo, sysOf)
		rawCmp := CompareOre(OreCompareInput{
			PortionSize: o.PortionSize, OreVolumeM3: o.VolumeM3, OreBuyPrice: orePrice,
			Materials: nil, NetYield: net, StationTake: stationTax, SalesTaxRate: salesTaxRate,
		})
		rawAvailable := oreOK && oreM3h > 0
		var rawEff, rawCycleMin, rawFillMin float64
		if rawAvailable {
			rawEff, rawCycleMin, rawFillMin =
				mining.EffectiveISKPerHour(oreHoldM3, oreM3h, rawCmp.RawNetPerM3, oreSecs, oreStopSecs)
		}

		// ---- REFINE path: reachable reprocess + best reachable hub ----
		var refNet, refEff, refCycleMin, refFillMin float64
		var refJumps int
		var refSellSysName string
		var refBreakdown []models.RefineMaterial
		refineAvailable := false
		if reprocessAvailable && oreM3h > 0 {
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
				bestEff := -1.0
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
					eff, cyc, fil := mining.EffectiveISKPerHour(
						oreHoldM3, oreM3h, hubCmp.RefineNetPerM3, reprocessSecs+hubSecs, 2*oreStopSecs)
					if eff > bestEff {
						bestEff = eff
						refNet = hubCmp.RefineNetPerM3
						refEff, refCycleMin, refFillMin = eff, cyc, fil
						refJumps = reprocessJumps + hubJumps
						if sn, e := s.names.GetSystemName(ctx, sysID); e == nil {
							refSellSysName = sn
						}
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
						refineAvailable = true
					}
				}
			}
		}

		// Skip the ore entirely if neither path is reachable.
		if !rawAvailable && !refineAvailable {
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
			LoadVolumeM3:        oreHoldM3,
			IsEstimate:          oreIsEstimate,
			EstimateReason:      oreEstimateReason,
		}
		if rawAvailable {
			rs := loc.resolve(ctx, oreLoc)
			row.RawSell = &rs
		}
		if refineAvailable {
			row.BestStationID = bestStationID
			row.BestStationName = bestStationName
			row.BestStationSystem = bestStationSystem
		}

		// Verdict + effective ISK/h from the better available path (≥1 is available).
		switch {
		case refineAvailable && (!rawAvailable || refEff >= rawEff):
			row.Best = "refine"
			row.EffectiveISKPerHour, row.CycleMinutes, row.FillMinutes = refEff, refCycleMin, refFillMin
			row.RouteJumps, row.SellSystemName = refJumps, refSellSysName
		default:
			row.Best = "raw"
			row.EffectiveISKPerHour, row.CycleMinutes, row.FillMinutes = rawEff, rawCycleMin, rawFillMin
			row.RouteJumps = oreJumps
			if row.RawSell != nil {
				row.SellSystemName = row.RawSell.SystemName
			}
		}

		row.DeltaISKPerHour = oreM3h * math.Abs(rawCmp.RawNetPerM3-refNet)
		resp.Rows = append(resp.Rows, row)
	}
```

Note: this replaces the OLD `resp.Rows = append(resp.Rows, row)` at the loop end too (it is now the last line above). Make sure there is exactly one `append` and one closing `}` for the `for i := range allOres` loop. The OLD code between the raw-buy line and the old append (materials loop, `cmp`, the old `row :=`, the `if oreOK { row.RawSell }`, the entire `// ---- Haul-downtime cycle ...` block, and the `if rowResolved ... else if oreM3h>0 ...` estimate block) is entirely replaced by the code above.

- [ ] **Step 2: Remove the now-unused `highestBuyOrder`**

`bestReachableBuyOrder` replaces it everywhere. Delete the `highestBuyOrder` method. Confirm no references: `grep -n "highestBuyOrder" backend/internal/services/*.go` → none.

- [ ] **Step 3: Build + gofmt + vet + lint**

Run: `cd backend && GOWORK=off go build ./... && gofmt -l internal/services/ && GOWORK=off go vet ./internal/services/... && GOWORK=off golangci-lint run ./internal/services/...`
Expected: builds; no gofmt output; `0 issues`. (If golangci flags an unused variable like `bestStanding`, ensure it still feeds `ReprocessTax`/`stationTax` as before; that block above the reprocess name change is unchanged.)

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/mining_service.go
git commit -m "feat(mining): decoupled, reachability-aware raw/refine haul-downtime"
```

---

## Task 3: Service tests

**Files:**
- Modify: `backend/internal/services/mining_service_test.go`

- [ ] **Step 1: Update existing tests for the new behavior**

The fakes already place buy orders at `LocationID: 60000123` (an NPC station) and `fakeMiningNames.GetSystemIDForLocation` returns `30000142` (Jita = origin via `fakeMiningLocation`). So the existing orders are in-system → reachable (travel 0). The existing Veldspar / VariantRealName / NoMiningSetup / EstimateWhenShipUnknown tests should still pass — run them and fix only assertions that depended on the removed shared-`rowResolved` behavior. Run:
`SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestMiningService -v`
If `TestMiningService_OreRanking_VariantRealName` or `_Veldspar` fail because the row is now built differently, adjust the field they read (e.g. `OreName`, `RawSell`, `BestStationName`) to match the new build — do not weaken the real-name / no-low-sec-leak assertions.

- [ ] **Step 2: Add a citadel-unreachable test**

Add a fake market whose only buy order sits in a citadel location (≥1e12), and assert the ore estimates with the new reason. Add near the other tests:

```go
func TestMiningService_OreRanking_CitadelUnreachable(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	// Veldspar buy order only in a citadel (≥1e12) → unreachable → no raw/refine path.
	market := &citadelOnlyMarket{price: 15.0}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{17940: 5, 22551: 5}},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}
	loc := fakeMiningLocation{shipTypeID: 22544}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002, AllowLowSec: false})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	// No reachable sell/reprocess (only-citadel orders) → the ore is skipped.
	for i := range resp.Rows {
		if resp.Rows[i].OreTypeID == veldsparTypeID {
			t.Errorf("Veldspar must be skipped when no reachable destination exists: %+v", resp.Rows[i])
		}
	}
}
```

And the fake (add to the fakes section):

```go
// citadelOnlyMarket returns one Veldspar buy order in a citadel (location ≥ 1e12)
// and its Tritanium material in a citadel too — nothing reachable.
type citadelOnlyMarket struct{ price float64 }

func (m *citadelOnlyMarket) GetMarketOrders(_ context.Context, _ int, typeID int) ([]database.MarketOrder, error) {
	if typeID == veldsparTypeID || typeID == tritaniumTypeID {
		return []database.MarketOrder{
			{TypeID: typeID, IsBuyOrder: true, Price: m.price, VolumeRemain: 1_000_000, LocationID: 1_035_000_000_001},
		}, nil
	}
	return []database.MarketOrder{}, nil
}
```

`fakeMiningNames.GetSystemIDForLocation` must error for citadel ids. Confirm its body returns an error for `id >= 1_000_000_000_000`; if it currently always returns `30000142`, change it to:

```go
func (fakeMiningNames) GetSystemIDForLocation(_ context.Context, id int64) (int64, error) {
	if id >= 1_000_000_000_000 {
		return 0, fmt.Errorf("citadel %d not in SDE", id)
	}
	return 30000142, nil
}
```

(Keep the NPC case returning 30000142 so the other tests stay reachable.)

- [ ] **Step 3: Add a decoupling test (raw reachable, refine reprocess unreachable)**

Make the reprocess station a citadel (so refine has no reachable reprocess) while the ore raw buy is at a reachable NPC station; assert the row resolves as raw with `EffectiveISKPerHour > 0` and is NOT estimate:

```go
func TestMiningService_OreRanking_DecoupledRawWhenRefineUnreachable(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	market := &fakeMiningMarket{buyPriceByType: map[int]float64{veldsparTypeID: 15.0, tritaniumTypeID: 5.0}}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{17940: 5, 22551: 5}},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	// Reprocess station is a citadel id → unreachable → refine path unavailable.
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 1_035_000_000_009, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}
	loc := fakeMiningLocation{shipTypeID: 22544}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002, AllowLowSec: false})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	var v *models.OreRankRow
	for i := range resp.Rows {
		if resp.Rows[i].OreTypeID == veldsparTypeID {
			v = &resp.Rows[i]
		}
	}
	if v == nil {
		t.Fatal("Veldspar row missing")
	}
	if v.IsEstimate {
		t.Errorf("raw is reachable → row must NOT be estimate even though refine reprocess is unreachable: %+v", *v)
	}
	if v.Best != "raw" || v.EffectiveISKPerHour <= 0 {
		t.Errorf("expected resolved raw verdict with eff>0, got Best=%q eff=%v", v.Best, v.EffectiveISKPerHour)
	}
}
```

`fmt` is already imported by the test file.

- [ ] **Step 4: Run + full gate**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... 2>&1 | grep -E "FAIL" ; gofmt -l internal pkg ; GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Expected: all three new/updated tests pass; no FAIL; no gofmt output; `0 issues`.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/services/mining_service_test.go
git commit -m "test(mining): reachability + decoupling + citadel-unreachable coverage"
```

---

## Task 4: Verify, PR, release, deploy

**Files:** none (process). **Backend-only — no web/flutter/APK change.**

- [ ] **Step 1: Full backend gate**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... && gofmt -l internal cmd pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Expected: ok; `0 issues`. (Frontend/Flutter unchanged — no need to rebuild, but a quick `cd ../frontend && npm run build` sanity check is fine since the response schema is unchanged.)

- [ ] **Step 2: PR**

```bash
git push -u origin feat/mining-reachability-aware-haul
gh pr create --base main --title "feat(mining): reachability-aware + decoupled haul-downtime" --body "<summary: pick reachable sell/reprocess destinations under the security choice (citadels/low-sec-gated excluded); compute raw/refine independently; estimate only when no path is reachable (or hull/crystal unresolved); fixes all-rows-estimate under High-only; backend-only; spec+plan linked>"
```

- [ ] **Step 3: CI green → squash-merge → update main**

```bash
gh pr checks <PR#> --watch --interval 20
gh pr merge <PR#> --squash --delete-branch
git checkout main && git pull --ff-only
```

- [ ] **Step 4: Release v0.25.2**

Add a `## [Unreleased]` CHANGELOG entry (Fixed: reachability-aware + decoupled haul-downtime; no more all-rows-estimate under High-only when the best sell/reprocess is low-sec-gated or in a citadel). Commit, then:
```bash
make release-check && make release VERSION=0.25.2
git add CHANGELOG.md && git commit -m "chore(release): v0.25.2"
git tag v0.25.2 && git push origin main && git push origin v0.25.2
```

- [ ] **Step 5: Watch deploy + prod smoke**

```bash
gh run watch <deploy-run-id> --interval 20 --exit-status
curl -s https://eveonline.sternrassler.de/api/v1/version   # → v0.25.2
curl -s -o /dev/null -w "%{http_code}\n" https://eveonline.sternrassler.de/mining  # → 200
```

No tablet APK — this change does not touch Flutter.

---

## Notes for the implementer

- **No silent fallbacks:** every path that drops a destination (citadel/unreachable, market fetch error) logs via `s.logger.Warn`. A row only estimates when hull/crystal is unresolved (feature #1) or no path has a reachable destination — with a specific reason.
- **Decoupling:** raw and refine are independent. A failed refine (no reachable reprocess/hub) must not estimate a row whose raw path is reachable, and vice versa.
- **Consistency:** raw uses the best reachable ore buy order; refine uses the best reachable reprocess + hub, and the materials breakdown shows that hub's prices/locations. Gross ISK/h (raw/refine) reflect the same reachable destinations.
- **No reachable path → skip the ore** (`continue`). `bestReachableBuyOrder` already returns the best *reachable* order, so an ore is only skipped when none of its buy orders (and no reprocess station) are routable under the security choice — a rare edge. `is_estimate` is therefore only ever set by the hull/crystal logic (feature #1), never by routing.
