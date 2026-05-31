# Mining sell & reprocess locations — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add to each Mining ore-ranking row WHERE to act: the best NPC reprocess station (name + system), where to sell the raw ore, and a per-mineral breakdown (mineral · effective qty · buy price · sell location) for the refine path — citadel locations shown as "Player-Structure".

**Architecture:** Backend captures the best buy order's location (not just price), resolves NPC station/system names (citadels → structure flag), and adds the fields to `OreRankRow` + the per-mineral breakdown in `MiningService.OreRanking`. Web + Flutter rows become expandable to show the locations.

**Tech Stack:** Go 1.24 / Fiber, SQLite SDE, Next.js/React/TS (web), Flutter/Dart (`app/`).

Spec: `docs/superpowers/specs/2026-05-31-mining-sell-reprocess-locations-design.md`.

**Verify env:** backend SDE tests use `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db` + `GOWORK=off`; golangci-lint 0 issues. Commits: prefix `PATH="$HOME/.local/bin:$PATH"`; Conventional Commits; NEVER --no-verify. Branch `feat/mining-sell-reprocess-locations` (already created).

**Confirmed interfaces:**
- `database.MarketOrder` has `IsBuyOrder bool`, `Price float64`, `LocationID int64`.
- `*SDERepository`: `GetStationName(ctx, int64) (string, error)` (NPC station-type name; returns `"Station-<id>"` for non-NPC/citadel), `GetSystemName(ctx, int64) (string, error)`, `GetSystemIDForLocation(ctx, int64) (int64, error)`, `GetTypeInfo(ctx, typeID int) (*TypeInfo, error)`.
- `MiningService` uses narrow dep interfaces (`MarketBuyProvider`, `ReprocessStationProvider`, `MiningLocationProvider`, …) — extend `MiningLocationProvider` with the name/system/type methods (all exist on `*SDERepository`).

---

## Task 1: highestBuyOrder — capture price + location

**Files:** `internal/services/mining_service.go`, `internal/services/mining_service_test.go`.

- [ ] **Step 1 — replace `highestBuyPrice`** with one that also returns the order's location:
```go
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
```
Update the two call sites in `OreRanking` (ore price, material price) to use the new signature (keep the existing `oreBuy`/`bp` price usage; you now also have the locationID for Task 4). Until Task 4 uses them, you may temporarily ignore the locationID with `_` — but prefer to land Task 4 right after so nothing is unused.

- [ ] **Step 2 — run** `SDE_DB_PATH=… GOWORK=off go test ./internal/services/ -run Mining` — existing mining tests still pass (the change is internal; the fake market repo already returns orders with a LocationID — set one in the test fixture if not).
- [ ] **Step 3 — commit:** `... -m "refactor(mining): capture buy-order location alongside price"`

## Task 2: resolveSellLocation + SellLocation type

**Files:** `internal/services/mining_location.go` (new) + `internal/services/mining_location_test.go`.

- [ ] **Step 1 — failing test** (in-memory SDE: an NPC station + its system; a citadel id):
```go
func TestResolveSellLocation(t *testing.T) {
	db := openLocSDE(t) // npcStations(60003760→typeID 1531, sys 30000142), types(1531 'Caldari Navy Assembly Plant'), mapSolarSystems(30000142 'Jita')
	repo := database.NewSDERepository(db)
	r := newLocResolver(repo)

	npc := r.resolve(context.Background(), 60003760)
	if npc.IsStructure || npc.StationName == "" || npc.SystemName != "Jita" {
		t.Fatalf("npc: %+v", npc)
	}
	cit := r.resolve(context.Background(), 1_035_000_000_001) // citadel id ≥ 1e12
	if !cit.IsStructure || cit.StationName != "" {
		t.Fatalf("citadel: %+v", cit)
	}
}
```
> Write `openLocSDE` mirroring `sde_reprocess_test.go`'s in-memory-schema style (create `npcStations`, `types`, `mapSolarSystems`; insert the rows). `GetStationName` joins `npcStations.typeID → types.name` (json `$.en`); `GetSystemName` reads `mapSolarSystems` (check the exact column it uses — read `GetSystemName` first).

- [ ] **Step 2 — run, expect fail.**
- [ ] **Step 3 — implement** `mining_location.go`:
```go
package services

import (
	"context"
	"strings"
)

// SellLocation describes where an order sits. Citadels (not in the SDE) carry only IsStructure.
type SellLocation struct {
	StationName string `json:"station_name,omitempty"`
	SystemName  string `json:"system_name,omitempty"`
	IsStructure bool   `json:"is_structure"`
}

// locSDE is the SDE subset the resolver needs.
type locSDE interface {
	GetStationName(ctx context.Context, stationID int64) (string, error)
	GetSystemName(ctx context.Context, systemID int64) (string, error)
	GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error)
}

type locResolver struct {
	sde   locSDE
	cache map[int64]SellLocation
}

func newLocResolver(sde locSDE) *locResolver { return &locResolver{sde: sde, cache: map[int64]SellLocation{}} }

const citadelIDFloor = int64(1_000_000_000_000)

// resolve maps a market-order locationID to a SellLocation. NPC stations → name + system;
// citadels / unresolvable → IsStructure (fail-loud: never a fabricated NPC name).
func (r *locResolver) resolve(ctx context.Context, locationID int64) SellLocation {
	if loc, ok := r.cache[locationID]; ok {
		return loc
	}
	var loc SellLocation
	if locationID >= citadelIDFloor {
		loc = SellLocation{IsStructure: true}
	} else if name, err := r.sde.GetStationName(ctx, locationID); err != nil || strings.HasPrefix(name, "Station-") {
		loc = SellLocation{IsStructure: true}
	} else {
		loc.StationName = name
		if sysID, e := r.sde.GetSystemIDForLocation(ctx, locationID); e == nil {
			if sysName, e2 := r.sde.GetSystemName(ctx, sysID); e2 == nil {
				loc.SystemName = sysName
			}
		}
	}
	r.cache[locationID] = loc
	return loc
}
```

- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit:** `... -m "feat(mining): resolve sell location (NPC station/system; citadel→structure)"`

## Task 3: model additions

**Files:** `internal/models/mining.go`.

- [ ] **Step 1 — add** to `models`:
```go
type RefineMaterial struct {
	MaterialTypeID int64               `json:"material_type_id"`
	MaterialName   string              `json:"material_name"`
	EffectiveQty   int64               `json:"effective_qty"`
	BuyPrice       float64             `json:"buy_price"`
	Sell           services.SellLocation `json:"sell"`
}
```
> `SellLocation` lives in `package services`; to avoid an import cycle (models must not import services), **define `SellLocation` in `package models`** instead (move the type from Task 2 into `models/mining.go`, and have `services` reference `models.SellLocation`). Adjust Task 2's resolver to return `models.SellLocation`. Add to `OreRankRow`:
```go
	BestStationName   string                 `json:"best_station_name,omitempty"`
	BestStationSystem string                 `json:"best_station_system,omitempty"`
	RawSell           *models.SellLocation   `json:"raw_sell,omitempty"`
	Materials         []models.RefineMaterial `json:"materials,omitempty"`
```
(Place `SellLocation`/`RefineMaterial` in `models/mining.go`; `OreRankRow` references them directly since it's already in `models`.)

- [ ] **Step 2 — build** `GOWORK=off go build ./...` (no test yet — wired in Task 4).
- [ ] **Step 3 — commit:** `... -m "feat(mining): response fields for sell + reprocess locations"`

## Task 4: wire locations into OreRanking

**Files:** `internal/services/mining_service.go`, `mining_service_test.go`. Extend `MiningLocationProvider` (the service's SDE dep) with `GetStationName`, `GetSystemName`, `GetSystemIDForLocation`, `GetTypeInfo` if not already present, and update the fakes in `mining_service_test.go`.

- [ ] **Step 1 — in `OreRanking`**, build a `locResolver` once per call from the service's SDE dep. Per ore:
  - ore best-buy: `orePrice, oreLoc, oreOK := s.highestBuyOrder(...)`; if `oreOK`, `row.RawSell = ptr(loc.resolve(ctx, oreLoc))`.
  - per material: `mp, mLoc, mOK := s.highestBuyOrder(...)`; build `models.RefineMaterial{ MaterialTypeID, MaterialName: typeName(ctx, matID), EffectiveQty: int64(math.Floor(float64(m.Quantity)*netYield)), BuyPrice: mp, Sell: loc.resolve(ctx, mLoc) }` (sell zero-value when `!mOK`); append to `row.Materials`.
  - reprocess station: `row.BestStationName = stationName(ctx, best.StationID)`; `row.BestStationSystem`: `GetSystemIDForLocation(best.StationID) → GetSystemName`.
  `typeName` uses `GetTypeInfo(ctx, int(matID)).Name` (guard nil/err → ""). Keep the existing `CompareOre` math unchanged.

- [ ] **Step 2 — extend the service test:** assert a Veldspar row has `RawSell` (with the fake station's name/system or IsStructure), and a `Materials` entry for Tritanium (type 34) with `EffectiveQty == floor(400×netYield)`, its `BuyPrice`, and `Sell`; and that a refine-verdict ore sets `BestStationName`/`BestStationSystem`. Extend the fake SDE/location provider to return canned names.

- [ ] **Step 3 — verify + commit:** `cd backend && gofmt -l internal && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./... && SDE_DB_PATH=… GOWORK=off go test ./...` → all green; `... -m "feat(mining): populate sell + reprocess locations in ore ranking"`.

---

## Task 5: web — types, api, expandable row

**Files:** `frontend/src/types/trading.ts`, `frontend/src/components/trading/OreRankingTable.tsx`, test.

- [ ] **Step 1 — types:** add `SellLocation { station_name?: string; system_name?: string; is_structure: boolean }`, `RefineMaterial { material_type_id: number; material_name: string; effective_qty: number; buy_price: number; sell: SellLocation }`; extend `OreRankRow` with `best_station_name?`, `best_station_system?`, `raw_sell?: SellLocation`, `materials?: RefineMaterial[]`. (`fetchOreRanking` is unchanged — same endpoint.)

- [ ] **Step 2 — `OreRankingTable`:** make each row **expandable** (a click/chevron toggles a detail panel under the row — mirror an existing expandable component if one exists; else a simple `useState` open-set keyed by `ore_type_id`). Detail panel:
  - if `best === "refine"`: show the reprocess station — `formatLocation(best_station_name, best_station_system)` (e.g. "Caldari Navy Assembly Plant — Jita") and a small **materials table**: `material_name · effective_qty · buy_price · formatSell(sell)`.
  - if `best === "raw"`: show "Verkaufen: " + `formatSell(raw_sell)`.
  - `formatSell(loc)`: `loc.is_structure ? "Player-Structure" : [loc.station_name, loc.system_name].filter(Boolean).join(" — ")`.
  Add a "Verkaufs-/Aufbereitungs-Ort" affordance (chevron) on each row.

- [ ] **Step 3 — test** (`tests/components/OreRankingTable.test.tsx`, extend): a refine row, when expanded, shows the reprocess station name + a material row with its sell location; a raw row shows the raw sell location; a row with `is_structure` sell shows "Player-Structure". Mirror existing test style.

- [ ] **Step 4 — verify + commit:** `cd frontend && npx vitest run && rm -rf .next && npm run build && npx eslint <files>`; `... -m "feat(frontend): expandable mining rows with sell + reprocess locations"`.

---

## Task 6: Flutter — models + expandable row

**Files:** `app/lib/api/mining_models.dart`, `app/lib/features/mining/mining_screen.dart` (or the ore_ranking_table widget), tests.

- [ ] **Step 1 — models:** add `SellLocation.fromJson{stationName?, systemName?, isStructure}`, `RefineMaterial.fromJson{materialTypeId, materialName, effectiveQty, buyPrice, sell}`; extend `OreRankRow.fromJson` with `bestStationName?`, `bestStationSystem?`, `rawSell?`, `materials?`.

- [ ] **Step 2 — table:** make each ore row **expandable** (`ExpansionTile` or the existing detail idiom). Expanded content mirrors web: refine → reprocess station (name — system) + per-material list (name · effective qty · buy price · sell location); raw → raw sell location; `isStructure` → "Player-Structure". Add a `_formatSell` helper.

- [ ] **Step 3 — tests:** model fromJson for the new fields; a widget test expanding a refine row shows the station + a material's sell location; a structure sell shows "Player-Structure".

- [ ] **Step 4 — verify + commit:** `cd app && flutter analyze && flutter test`; `... -m "feat(flutter): expandable mining rows with sell + reprocess locations"`.

---

## Task 7: full verification + PR

- [ ] Backend: `gofmt -l internal pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./... && SDE_DB_PATH=… GOWORK=off go test ./...` → green, golangci 0.
- [ ] Web: `npx vitest run && npm run build`. Flutter: `flutter analyze && flutter test`.
- [ ] Push `feat/mining-sell-reprocess-locations`; open PR (base main). After CI green + merge → release `v0.21.0` (gated). No Flutter APK rebuild needed for backend-only? — this touches Flutter too, so **rebuild + reinstall the APK** after release (`flutter build apk --release --dart-define=…` + `adb install -r`).

---

## Self-review notes

- **Spec coverage:** reprocess station name+system (T4), raw sell location (T4), per-mineral breakdown with effective qty + buy price + sell location (T3/T4), citadel→structure (T2), buy-order-location capture without valuation change (T1), web expandable (T5), Flutter expandable (T6), fail-loud unresolvable→structure (T2). All covered.
- **Import-cycle note:** `SellLocation`/`RefineMaterial` live in `package models`; `services` returns `models.SellLocation` — avoids models→services cycle. (Resolved inline in T3.)
- **Verify during execution:** the exact `GetSystemName` source column; whether `MiningLocationProvider` already has these methods (extend + update fakes if not); an existing expandable-row pattern to mirror (web + Flutter).
