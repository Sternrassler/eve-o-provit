# Umkreis-Hauling Routes (#45) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** From the character's current region + adjacent regions, find profitable station→station hauling routes (buy cheap in one station, sell dear in another) with an optimal per-trip cargo load under cargo + capital constraints.

**Architecture:** Reuse the per-region order fetch, navigation, cargo-knapsack and character-location services. New: region adjacency from `v_stargate_graph`, a pure cross-region matcher (orders → per-item best buy/sell → routes), and a cargo+capital greedy `HaulingOptimizer`. Surfaced via `POST /api/v1/trading/hauling/routes`, a web `/hauling` page and a Flutter screen.

**Tech Stack:** Go/Fiber (pgx, existing services), Next.js 16 + React Query + Tailwind, Flutter + Riverpod + dio. Spec: `docs/superpowers/specs/2026-05-29-hauling-routes-design.md`.

---

## File Structure

**Backend (`backend/`)**
- Create `internal/models/hauling.go` — `HaulingRequest`, `HaulingRoute`, `HaulingItem`, `HaulingResponse`.
- Create `internal/services/hauling_optimizer.go` — pure cargo+capital greedy fill + unit tests.
- Create `internal/services/hauling_matcher.go` — pure cross-region matcher (orders → routes) + unit tests.
- Create `internal/services/hauling_service.go` — orchestration (location → regions → fetch → match → optimize → navigation → rank).
- Modify `internal/database/sde.go` — add `GetNeighborRegions`. Test: `internal/database/sde_neighbor_integration_test.go`.
- Create `internal/handlers/hauling.go`; Modify `cmd/api/container.go`, `cmd/api/main.go`.

**Web (`frontend/`)** — Create `src/app/hauling/page.tsx`, `src/components/trading/HaulingRouteList.tsx`, `HaulingRouteDetail.tsx`; Modify `src/lib/api-client.ts`, `src/types/trading.ts`, nav. Tests: `tests/components/HaulingRouteList.test.tsx`, `tests/e2e/auth/hauling.spec.ts`.

**Flutter (`app/`)** — Create `lib/api/hauling_models.dart`, `lib/features/trading/hauling_screen.dart`, `lib/features/trading/hauling_providers.dart`; Modify `lib/api/trading_api.dart`, `lib/core/router.dart`. Tests: `test/hauling_models_test.dart`, `test/hauling_screen_layout_test.dart`, extend `test/e2e/*`.

---

## Task 1: Backend models

**Files:** Create `backend/internal/models/hauling.go`

- [ ] **Step 1: Write `hauling.go`**

```go
package models

// HaulingRequest is the body for POST /api/v1/trading/hauling/routes.
type HaulingRequest struct {
	OriginRegionID int     `json:"origin_region_id"` // 0 = use character's current region
	ShipTypeID     int     `json:"ship_type_id"`
	Capital        float64 `json:"capital"`
	AvoidLowSec    bool    `json:"avoid_low_sec"`
	MaxRoutes      int     `json:"max_routes"` // 0 = default 15
} // @name HaulingRequest

// HaulingItem is one item in a route's optimal cargo load.
type HaulingItem struct {
	TypeID      int     `json:"type_id"`
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	BuyPrice    float64 `json:"buy_price"`
	SellPrice   float64 `json:"sell_price"`
	UnitVolume  float64 `json:"unit_volume"`
	Profit      float64 `json:"profit"`        // net profit for this position
	ProfitPerM3 float64 `json:"profit_per_m3"`
} // @name HaulingItem

// HaulingRoute is one buy-station → sell-station haul with its optimal load.
type HaulingRoute struct {
	BuySystemName   string  `json:"buy_system_name"`
	BuyStationName  string  `json:"buy_station_name"`
	BuyStationID    int64   `json:"buy_station_id"`
	SellSystemName  string  `json:"sell_system_name"`
	SellStationName string  `json:"sell_station_name"`
	SellStationID   int64   `json:"sell_station_id"`
	Jumps           int     `json:"jumps"`
	TravelTimeMin   float64 `json:"travel_time_min"`
	SecurityRisk    string  `json:"security_risk"` // "safe" | "caution" | "danger"
	TotalProfit     float64 `json:"total_profit"`
	TotalCapital    float64 `json:"total_capital"`
	TotalVolume     float64 `json:"total_volume"`
	ISKPerHour      float64 `json:"isk_per_hour"`
	Items           []HaulingItem `json:"items"`
} // @name HaulingRoute

// HaulingResponse is the response.
type HaulingResponse struct {
	OriginRegionID int            `json:"origin_region_id"`
	RegionsScanned []int          `json:"regions_scanned"`
	Routes         []HaulingRoute `json:"routes"`
	SkillsApplied  SkillsApplied  `json:"skills_applied"`
} // @name HaulingResponse
```

- [ ] **Step 2:** `cd backend && go build ./...` → ok. Commit: `git add backend/internal/models/hauling.go && git commit -m "feat(hauling): models (#45)"`

---

## Task 2: GetNeighborRegions (SDE) — TDD

**Files:** Modify `backend/internal/database/sde.go`; Test `backend/internal/database/sde_neighbor_integration_test.go`

- [ ] **Step 1: Write integration test** (`//go:build integration`)

```go
//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNeighborRegions_TheForge(t *testing.T) {
	// SDE is read-only SQLite; open it like other SDE integration tests do.
	repo := newSDERepoForTest(t) // see existing SDE integration tests for the helper/open pattern
	ctx := context.Background()
	neighbors, err := repo.GetNeighborRegions(ctx, 10000002) // The Forge
	require.NoError(t, err)
	// The Forge borders 8 regions incl. Lonetrek (10000016), Sinq Laison (10000032),
	// Metropolis (10000042), The Citadel (10000033).
	assert.GreaterOrEqual(t, len(neighbors), 6)
	assert.Contains(t, neighbors, 10000016)
	assert.Contains(t, neighbors, 10000032)
	assert.NotContains(t, neighbors, 10000002) // excludes self
}
```

NOTE: use the same SDE-open pattern the existing SDE integration tests use (look for how other `*_integration_test.go` in `internal/database` obtain an `*SDERepository` against `backend/data/sde/eve-sde.db`). If none exists, open `sql.Open("sqlite3", "file:../../data/sde/eve-sde.db?mode=ro&immutable=1")` and `NewSDERepository(db)`.

- [ ] **Step 2: Run, verify fail** — `go test -tags integration -run TestGetNeighborRegions ./internal/database/` → FAIL (undefined method).

- [ ] **Step 3: Implement `GetNeighborRegions`** in `sde.go`

```go
// GetNeighborRegions returns region IDs adjacent to regionID — regions whose system
// is connected by a stargate to a system in regionID. Excludes regionID itself.
func (r *SDERepository) GetNeighborRegions(ctx context.Context, regionID int) ([]int, error) {
	const q = `
		SELECT DISTINCT s2.regionID
		FROM v_stargate_graph g
		JOIN mapSolarSystems s1 ON g.from_system_id = s1._key
		JOIN mapSolarSystems s2 ON g.to_system_id   = s2._key
		WHERE s1.regionID = ? AND s2.regionID <> ?
		ORDER BY s2.regionID`
	rows, err := r.db.QueryContext(ctx, q, regionID, regionID)
	if err != nil {
		return nil, fmt.Errorf("query neighbor regions: %w", err)
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var rid int
		if err := rows.Scan(&rid); err != nil {
			return nil, fmt.Errorf("scan neighbor region: %w", err)
		}
		out = append(out, rid)
	}
	return out, rows.Err()
}
```

(Use the SDERepository's existing `*sql.DB` field/accessor — match how other methods in `sde.go` query, e.g. `r.db.QueryContext` or the repo's query helper.)

- [ ] **Step 4: Run** — `go test -tags integration -run TestGetNeighborRegions ./internal/database/` → PASS (needs Docker? No — SDE is local SQLite; runs directly).

- [ ] **Step 5: Commit** — `git add backend/internal/database/ && git commit -m "feat(hauling): SDE GetNeighborRegions from stargate graph (#45)"`

---

## Task 3: HaulingOptimizer (cargo + capital greedy) — TDD

**Files:** Create `backend/internal/services/hauling_optimizer.go`, `..._test.go`

- [ ] **Step 1: Write failing tests**

```go
package services

import (
	"math"
	"testing"
)

func happrox(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func hitem(id int, name string, buy, profit, vol float64, qty int) HaulItem {
	return HaulItem{TypeID: id, Name: name, BuyPrice: buy, ProfitPerUnit: profit, UnitVolume: vol, AvailableQty: qty}
}

func TestFillCargo_RespectsCargoCapitalAndQty(t *testing.T) {
	opt := NewHaulingOptimizer()
	// Item A: best profit/m³; B: worse. Cargo 10 m³, capital 1000.
	load := opt.FillCargo([]HaulItem{
		hitem(1, "A", 100, 20, 1, 100), // profit/m³ = 20
		hitem(2, "B", 50, 5, 1, 100),   // profit/m³ = 5
	}, 10, 1000)
	// Cargo (10 m³ → 10 units) binds before capital (1000/100=10) — 10x A.
	if len(load.Items) == 0 || load.Items[0].TypeID != 1 || load.Items[0].Quantity != 10 {
		t.Fatalf("expected 10x A first, got %+v", load.Items)
	}
	if !happrox(load.TotalVolume, 10) || !happrox(load.TotalProfit, 200) {
		t.Errorf("totals wrong: %+v", load)
	}
}

func TestFillCargo_CapitalBinds(t *testing.T) {
	opt := NewHaulingOptimizer()
	// Cargo huge, capital 250 → at buy 100 only 2 units affordable.
	load := opt.FillCargo([]HaulItem{hitem(1, "A", 100, 20, 1, 100)}, 1e9, 250)
	if load.Items[0].Quantity != 2 {
		t.Errorf("capital should bind to 2 units, got %d", load.Items[0].Quantity)
	}
}

func TestFillCargo_QtyBinds(t *testing.T) {
	opt := NewHaulingOptimizer()
	load := opt.FillCargo([]HaulItem{hitem(1, "A", 1, 5, 1, 3)}, 1e9, 1e9)
	if load.Items[0].Quantity != 3 {
		t.Errorf("available qty should bind to 3, got %d", load.Items[0].Quantity)
	}
}

func TestFillCargo_Empty(t *testing.T) {
	opt := NewHaulingOptimizer()
	load := opt.FillCargo(nil, 10, 10)
	if len(load.Items) != 0 || load.TotalProfit != 0 {
		t.Errorf("empty input → empty load, got %+v", load)
	}
}
```

- [ ] **Step 2: Run, verify fail** — `go test ./internal/services/ -run TestFillCargo` → FAIL (undefined).

- [ ] **Step 3: Implement `hauling_optimizer.go`**

```go
package services

import "sort"

// HaulItem is a candidate for a single route's cargo (net profit pre-computed).
type HaulItem struct {
	TypeID        int
	Name          string
	BuyPrice      float64 // capital cost per unit
	ProfitPerUnit float64 // net profit per unit
	UnitVolume    float64 // m³ per unit
	AvailableQty  int     // min(buy-side available, sell-side demand)
}

// HaulLoadItem is one selected position.
type HaulLoadItem struct {
	TypeID      int
	Name        string
	Quantity    int
	CapitalUsed float64
	Profit      float64
}

// HaulLoad is the optimal cargo for one route.
type HaulLoad struct {
	Items       []HaulLoadItem
	TotalProfit float64
	TotalCapital float64
	TotalVolume float64
}

// HaulingOptimizer fills one ship for one route greedily by profit per m³.
type HaulingOptimizer struct{}

func NewHaulingOptimizer() *HaulingOptimizer { return &HaulingOptimizer{} }

// FillCargo greedily adds the most profit-dense items (profit per m³) until cargo m³,
// capital or per-item availability is exhausted.
func (o *HaulingOptimizer) FillCargo(items []HaulItem, cargoM3, capital float64) HaulLoad {
	cand := make([]HaulItem, 0, len(items))
	for _, it := range items {
		if it.UnitVolume <= 0 || it.BuyPrice <= 0 || it.ProfitPerUnit <= 0 || it.AvailableQty < 1 {
			continue
		}
		cand = append(cand, it)
	}
	sort.SliceStable(cand, func(i, j int) bool {
		return cand[i].ProfitPerUnit/cand[i].UnitVolume > cand[j].ProfitPerUnit/cand[j].UnitVolume
	})

	cargoLeft, capitalLeft := cargoM3, capital
	var load HaulLoad
	for _, it := range cand {
		qty := it.AvailableQty
		if byVol := int(cargoLeft / it.UnitVolume); byVol < qty {
			qty = byVol
		}
		if byCap := int(capitalLeft / it.BuyPrice); byCap < qty {
			qty = byCap
		}
		if qty < 1 {
			continue
		}
		capUsed := float64(qty) * it.BuyPrice
		profit := float64(qty) * it.ProfitPerUnit
		load.Items = append(load.Items, HaulLoadItem{
			TypeID: it.TypeID, Name: it.Name, Quantity: qty, CapitalUsed: capUsed, Profit: profit,
		})
		cargoLeft -= float64(qty) * it.UnitVolume
		capitalLeft -= capUsed
		load.TotalProfit += profit
		load.TotalCapital += capUsed
		load.TotalVolume += float64(qty) * it.UnitVolume
	}
	return load
}
```

- [ ] **Step 4: Run** — `go test ./internal/services/ -run TestFillCargo -v` → PASS. Fix until green.

- [ ] **Step 5: Commit** — `gofmt -w internal/services/hauling_optimizer*.go && git add backend/internal/services/hauling_optimizer*.go && git commit -m "feat(hauling): cargo+capital greedy optimizer + tests (#45)"`

---

## Task 4: Cross-region matcher — TDD

**Files:** Create `backend/internal/services/hauling_matcher.go`, `..._test.go`

The matcher is pure over a flat order list (decoupled from ESI/DB). Order shape it needs:

```go
type matchOrder struct {
	TypeID     int
	StationID  int64
	IsBuyOrder bool
	Price      float64
	VolumeRemain int
}
```

For each type: cheapest active SELL order (= where to BUY, source) and highest active BUY order (= where to SELL, target). An opportunity exists when source and target are different stations and `targetBuyPrice*(1-salesTaxRate) > sourceSellPrice`. Group opportunities by `(buyStationID → sellStationID)`.

- [ ] **Step 1: Write failing tests**

```go
package services

import "testing"

func mo(typeID int, station int64, buy bool, price float64, vol int) matchOrder {
	return matchOrder{TypeID: typeID, StationID: station, IsBuyOrder: buy, Price: price, VolumeRemain: vol}
}

func TestMatchHauls_GroupsByRoute(t *testing.T) {
	orders := []matchOrder{
		// type 1: cheapest sell at station 100 (5.0), highest buy at station 200 (8.0) → profitable haul 100→200
		mo(1, 100, false, 5.0, 50), mo(1, 100, false, 6.0, 10),
		mo(1, 200, true, 8.0, 40),
		// type 2: also cheapest sell @100 (3.0), highest buy @200 (4.0) → same route 100→200
		mo(2, 100, false, 3.0, 100), mo(2, 200, true, 4.0, 100),
		// type 3: sell and buy at same station 300 → no haul
		mo(3, 300, false, 10, 5), mo(3, 300, true, 12, 5),
	}
	routes := MatchHauls(orders, 0.0) // 0% sales tax for the test
	if len(routes) != 1 {
		t.Fatalf("expected 1 route (100→200), got %d", len(routes))
	}
	r := routes[0]
	if r.BuyStationID != 100 || r.SellStationID != 200 {
		t.Errorf("route stations wrong: %d→%d", r.BuyStationID, r.SellStationID)
	}
	if len(r.Items) != 2 { // type 1 and 2
		t.Errorf("expected 2 items on route, got %d", len(r.Items))
	}
	// type 1: profit/unit = 8-5 = 3; available = min(50,40)=40
	var i1 *HaulItem
	for k := range r.Items {
		if r.Items[k].TypeID == 1 {
			i1 = &r.Items[k]
		}
	}
	if i1 == nil || i1.ProfitPerUnit != 3.0 || i1.AvailableQty != 40 || i1.BuyPrice != 5.0 {
		t.Errorf("type1 economics wrong: %+v", i1)
	}
}

func TestMatchHauls_SalesTaxRemovesThinMargin(t *testing.T) {
	orders := []matchOrder{
		mo(1, 100, false, 100, 10), mo(1, 200, true, 101, 10), // 1% gross; 5% tax → net negative
	}
	routes := MatchHauls(orders, 0.05)
	if len(routes) != 0 {
		t.Errorf("expected no profitable route after tax, got %d", len(routes))
	}
}
```

- [ ] **Step 2: Run, verify fail** — `go test ./internal/services/ -run TestMatchHauls` → FAIL.

- [ ] **Step 3: Implement `hauling_matcher.go`**

```go
package services

// matchOrder is the minimal order shape the matcher needs.
type matchOrder struct {
	TypeID       int
	StationID    int64
	IsBuyOrder   bool
	Price        float64
	VolumeRemain int
}

// RouteCandidate is one (buy station → sell station) leg with its profitable items.
type RouteCandidate struct {
	BuyStationID  int64
	SellStationID int64
	Items         []HaulItem
}

// MatchHauls finds, per item type, the cheapest sell order (buy source) and highest
// buy order (sell target) across all stations, keeps profitable cross-station hauls
// (net of salesTaxRate on the sell), and groups them by (buy station → sell station).
func MatchHauls(orders []matchOrder, salesTaxRate float64) []RouteCandidate {
	type best struct {
		sellPrice float64
		sellQty   int
		sellStn   int64
		hasSell   bool
		buyPrice  float64
		buyQty    int
		buyStn    int64
		hasBuy    bool
	}
	byType := map[int]*best{}
	for _, o := range orders {
		if o.VolumeRemain <= 0 || o.Price <= 0 {
			continue
		}
		b := byType[o.TypeID]
		if b == nil {
			b = &best{}
			byType[o.TypeID] = b
		}
		if o.IsBuyOrder { // someone buys here → a SELL target for us
			if !b.hasBuy || o.Price > b.buyPrice {
				b.buyPrice, b.buyQty, b.buyStn, b.hasBuy = o.Price, o.VolumeRemain, o.StationID, true
			}
		} else { // someone sells here → a BUY source for us
			if !b.hasSell || o.Price < b.sellPrice {
				b.sellPrice, b.sellQty, b.sellStn, b.hasSell = o.Price, o.VolumeRemain, o.StationID, true
			}
		}
	}

	grouped := map[[2]int64]*RouteCandidate{}
	for typeID, b := range byType {
		if !b.hasSell || !b.hasBuy || b.sellStn == b.buyStn {
			continue
		}
		net := b.buyPrice*(1-salesTaxRate) - b.sellPrice
		if net <= 0 {
			continue
		}
		qty := b.sellQty
		if b.buyQty < qty {
			qty = b.buyQty
		}
		key := [2]int64{b.sellStn, b.buyStn} // buy at sellStn (cheapest sell order), sell at buyStn
		rc := grouped[key]
		if rc == nil {
			rc = &RouteCandidate{BuyStationID: b.sellStn, SellStationID: b.buyStn}
			grouped[key] = rc
		}
		rc.Items = append(rc.Items, HaulItem{
			TypeID:        typeID,
			BuyPrice:      b.sellPrice,
			ProfitPerUnit: net,
			AvailableQty:  qty,
			// Name + UnitVolume filled by the service from SDE (matcher is price-only).
		})
	}

	out := make([]RouteCandidate, 0, len(grouped))
	for _, rc := range grouped {
		out = append(out, *rc)
	}
	return out
}
```

- [ ] **Step 4: Run** — `go test ./internal/services/ -run TestMatchHauls -v` → PASS. NOTE: tests don't set Name/UnitVolume; the service enriches those from the SDE before optimizing.

- [ ] **Step 5: Commit** — `gofmt -w internal/services/hauling_matcher*.go && git add backend/internal/services/hauling_matcher*.go && git commit -m "feat(hauling): cross-region order matcher + tests (#45)"`

---

## Task 5: HaulingService + handler + route + wiring

**Files:** Create `backend/internal/services/hauling_service.go`, `backend/internal/handlers/hauling.go`; Modify `cmd/api/container.go`, `cmd/api/main.go`

- [ ] **Step 1: Implement `HaulingService.FindRoutes(ctx, req, characterID, accessToken) (*models.HaulingResponse, error)`**

Orchestration (use the existing services via small injected interfaces for testability where practical; concrete wiring in container.go):
1. Resolve origin region: if `req.OriginRegionID > 0` use it; else `characterHelper.GetCharacterLocation` → system → `sdeRepo.GetRegionIDForSystem`.
2. `regions = append(GetNeighborRegions(origin), origin)`.
3. For each region: `routeFinder.fetchMarketOrders(ctx, region)` (reuse; tolerate per-region errors with a logged warn). Convert each `database.MarketOrder` → `matchOrder{TypeID, StationID: LocationID, IsBuyOrder, Price, VolumeRemain}`.
4. `salesTaxRate := SalesTaxRate(skills.Accounting)` (skills via `skillsService.GetCharacterSkills`, graceful default).
5. `cands := MatchHauls(allOrders, salesTaxRate)`.
6. Enrich each candidate's items with `Name` + `UnitVolume` from SDE (`sdeRepo.GetTypeInfo` / `cargo.GetItemVolume`); drop items with no volume. Compute effective cargo capacity for the ship (reuse the route/fitting path: `fittingService.GetShipFitting`→ `EffectiveCargo`, like `route_service.applyCharacterSkills`).
7. For each candidate route: `load := optimizer.FillCargo(items, effectiveCargo, req.Capital)`; skip empty loads. Resolve buy/sell systems from station IDs (`sdeRepo.GetSystemIDForLocation`) + station/system names. Compute travel via `navigation.ShortestPath(buySystem, sellSystem, req.AvoidLowSec)` + `navigation.CalculateTravelTime(...)`; `jumps`, `travel_time_min`, min-sec → `security_risk` (≥0.5 safe, >0.0 caution, ≤0.0 danger); if `AvoidLowSec` and route has <0.5 hop → skip. `ISKPerHour = TotalProfit / (roundTripMinutes/60)`.
8. Map to `models.HaulingRoute` (incl. per-item `SellPrice` = BuyPrice+ProfitPerUnit before tax-display, `ProfitPerM3`), rank by `ISKPerHour` desc, cap to `MaxRoutes` (default 15). Build `models.HaulingResponse` with `RegionsScanned` + `SkillsApplied`.

- [ ] **Step 2: Implement handler** `hauling.go` — mirror `internal/handlers/portfolio.go`: parse `HaulingRequest`, validate `ShipTypeID>0 && Capital>0`, extract `character_id`/`access_token` from Locals (int/string, 401 if missing), call `service.FindRoutes`, JSON; 500 on error.

- [ ] **Step 3: Wire** in `container.go`: build `HaulingService` with the existing `c.SDERepo`, the route finder / market repo, `navigation` (via `c.DB.SDE`), `fittingService`, `skillsService`, `characterHelper`, `NewHaulingOptimizer()`, logger; build `c.HaulingHandler`. Register in `main.go`:
```go
api.Post("/trading/hauling/routes", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.HaulingHandler.FindRoutes)
```

- [ ] **Step 4: Build + vet** — `go build ./... && go vet ./internal/... ./cmd/...` → clean. `go test -short ./internal/...` → PASS.

- [ ] **Step 5: Commit** — `git add backend/ && git commit -m "feat(hauling): service + handler + route + wiring (#45)"`

---

## Task 6: Backend gates + CHANGELOG

- [ ] **Step 1:** `gofmt -l internal/ cmd/` empty; `go test -short ./internal/...` PASS; `go build -tags integration ./...` ok.
- [ ] **Step 2:** Add `CHANGELOG.md` `[Unreleased] ### Added` entry for the hauling endpoint.
- [ ] **Step 3: Commit** — `git add backend CHANGELOG.md && git commit -m "chore(hauling): changelog + gates (#45)"`

---

## Task 7: Web `/hauling` + tests

Follow #44/#107 patterns exactly: `src/app/roi-calculator/page.tsx` (form + mutation + auth gate), `PortfolioResultTable.tsx` (table + waypoint button + `setWaypoint` helper in `api-client.ts`), `tests/components/PortfolioResultTable.test.tsx`, `tests/e2e/auth/roi-calculator.spec.ts`.

- [ ] **Step 1:** Types in `src/types/trading.ts`: `HaulingRequest`, `HaulingRoute`, `HaulingItem`, `HaulingResponse`.
- [ ] **Step 2:** `src/lib/api-client.ts` → `findHaulingRoutes(req)` (authed POST `/api/v1/trading/hauling/routes`).
- [ ] **Step 3:** `HaulingRouteList.tsx` (one card/row per route: buy→sell system, jumps, travel time, security badge safe/caution/danger, ISK/h, total profit, cargo fill %) + `HaulingRouteDetail.tsx` (expand → cargo table: Item, Menge, Kauf@Station, Verkauf@Station, Volumen, Gewinn, Gewinn/m³ + reuse the **`setWaypoint`** helper for a "Route an EVE übertragen" button: clear+buyStationID, then sellStationID). `formatISKWithSeparators` for ISK.
- [ ] **Step 4:** `src/app/hauling/page.tsx`: auth gate + ErrorBoundary + form (Ship select, Capital, avoid-low-sec; origin region shown from character location) → `useMutation(findHaulingRoutes)` → route list. Empty-state "Keine profitablen Routen im Umkreis". Add a nav entry for /hauling.
- [ ] **Step 5:** Vitest `tests/components/HaulingRouteList.test.tsx` (route fields, security badge, cargo table on expand, waypoint button calls buy-then-sell IDs) + Playwright `tests/e2e/auth/hauling.spec.ts`. `npm run test` + `npm run lint` green.
- [ ] **Step 6: Commit** — `git add frontend && git commit -m "feat(frontend): hauling routes page + tests (#45)"`

---

## Task 8: Flutter `/hauling` + tests

Follow `hub_comparison_*` / `roi_*` + `route_detail.dart` (waypoint) patterns exactly.

- [ ] **Step 1:** `lib/api/hauling_models.dart` — `HaulingRequest`/`HaulingRoute`/`HaulingItem`/`HaulingResponse`, null/float-robust `fromJson`.
- [ ] **Step 2:** `lib/api/trading_api.dart` add `findHaulingRoutes(HaulingRequest)`; `lib/features/trading/hauling_providers.dart` `AsyncNotifier<HaulingResponse?>`.
- [ ] **Step 3:** `lib/features/trading/hauling_screen.dart` adaptive `isTwoPane(840)`: form + route list; route detail = cargo DataTable + security badge + per-route "Route an EVE übertragen" button (reuse `route_detail.dart` waypoint pattern: buy station then sell station).
- [ ] **Step 4:** `lib/core/router.dart` add `/hauling` GoRoute + a NavigationDestination ("Hauling").
- [ ] **Step 5:** `test/hauling_models_test.dart` (parse incl. empty routes), `test/hauling_screen_layout_test.dart` (1-/2-pane), e2e group + `fakeHaulingResponse()` in `test/e2e/support/fakes.dart`. `flutter analyze` clean + `flutter test` pass.
- [ ] **Step 6: Commit** — `git add app && git commit -m "feat(app): hauling routes screen + tests (#45)"`

---

## Task 9: Ship

- [ ] **Step 1:** PR `feat/hauling-routes` → CI green → merge.
- [ ] **Step 2:** `make release VERSION=0.11.0` → release PR → merge → tag `v0.11.0` → deploy (backend + frontend) → smoke success.
- [ ] **Step 3:** Prod verify: `/version`, `/hauling` 200, `POST /trading/hauling/routes` 401 without auth. Flutter APK rebuild + on-device happy path (current region → routes → cargo + waypoint button).
- [ ] **Step 4:** Comment summary on issue #45; close.

---

## Self-Review

- **Spec coverage:** inter-hub price differentials (Task 4 matcher) ✓; navigation travel time (Task 5 step 7) ✓; cargo optimizer max profit/m³ (Task 3) ✓; risk assessment low/null-sec (Task 5 step 7 + AvoidLowSec) ✓; origin = current region + neighbors (Task 2 + Task 5 step 1-2) ✓; station-granular any→any (Task 4 groups by station, both directions via per-type best buy/sell) ✓; capital constraint (Task 3) ✓; web+flutter + waypoint reuse (Tasks 7,8) ✓.
- **Placeholders:** Task 5 describes orchestration by pointing at concrete reuse (`fetchMarketOrders`, `navigation.ShortestPath`, `GetShipFitting`, `portfolio.go` handler analogue) — reuse, not placeholder; all new pure logic (Tasks 2,3,4) has complete code.
- **Type consistency:** `HaulItem`/`HaulLoad` (optimizer, internal) ↔ `RouteCandidate` (matcher, internal, holds `[]HaulItem`) ↔ `models.HaulingRoute`/`HaulingItem` (API); service maps between them. `MatchHauls(orders, salesTaxRate)` and `FillCargo(items, cargoM3, capital)` signatures consistent across tasks. `findHaulingRoutes`/`FindRoutes` naming consistent.
