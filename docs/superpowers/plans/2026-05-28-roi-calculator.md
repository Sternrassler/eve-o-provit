# ROI Calculator & Capital Allocation Optimizer (#44) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Given a region, ship, capital and time budget, suggest how to allocate the capital across multiple items for maximum expected daily profit, with liquidity + diversification constraints.

**Architecture:** A portfolio layer on the existing route engine. `RouteService.CalculateWithFilters` produces the profitable station→station routes (skill-adjusted net profit, cargo, trips, ISK/h, volume); a new greedy `PortfolioOptimizerService` allocates capital + a shared time budget across those candidates under per-item liquidity and capital caps. Surfaced via `POST /api/v1/trading/portfolio/optimize`, a web `/roi-calculator` page and a Flutter screen.

**Tech Stack:** Go/Fiber backend (pgx, existing services), Next.js 16 + React Query + Tailwind, Flutter + Riverpod + dio. Spec: `docs/superpowers/specs/2026-05-28-roi-calculator-design.md`.

---

## File Structure

**Backend (`backend/`)**
- Create `internal/models/portfolio.go` — `PortfolioRequest`, `PortfolioResult`, `PortfolioItem`.
- Create `internal/services/portfolio_service.go` — `PortfolioOptimizerService` (greedy allocator + diversification score). Pure logic over a candidate list (testable without the engine).
- Create `internal/services/portfolio_service_test.go` — unit tests.
- Create `internal/handlers/portfolio.go` — `MultiHubHandler`-style handler.
- Modify `cmd/api/container.go` — build the service + handler.
- Modify `cmd/api/main.go` — register `POST /api/v1/trading/portfolio/optimize`.

**Web (`frontend/`)**
- Modify `src/app/roi-calculator/page.tsx` — replace placeholder.
- Create `src/components/trading/PortfolioInputForm.tsx`, `PortfolioResultTable.tsx`, `DiversificationScore.tsx`.
- Modify `src/lib/api-client.ts` — `optimizePortfolio(req)`.
- Modify `src/types/trading.ts` — `PortfolioRequest/Result/Item`.
- Create `tests/components/PortfolioResultTable.test.tsx`, `tests/e2e/auth/roi-calculator.spec.ts`.

**Flutter (`app/`)**
- Create `lib/api/portfolio_models.dart`, `lib/features/trading/roi_calculator_screen.dart`, `lib/features/trading/roi_providers.dart`.
- Modify `lib/api/trading_api.dart` (add `optimizePortfolio`), `lib/core/router.dart` (route + nav).
- Create `test/portfolio_models_test.dart`, `test/roi_calculator_screen_layout_test.dart`; extend `test/e2e/app_flow_test.dart` + `test/e2e/support/fakes.dart`.

---

## Task 1: Backend models

**Files:** Create `backend/internal/models/portfolio.go`

- [ ] **Step 1: Write `portfolio.go`**

```go
package models

// PortfolioRequest is the body for POST /api/v1/trading/portfolio/optimize.
type PortfolioRequest struct {
	RegionID        int      `json:"region_id"`
	ShipTypeID      int      `json:"ship_type_id"`
	Capital         float64  `json:"capital"`            // ISK budget
	TimeBudgetMin   float64  `json:"time_budget_min"`    // available trading minutes/day
	LiquidityCapPct float64  `json:"liquidity_cap_pct"`  // 0..100, max share of daily volume per item
	MaxItemPct      float64  `json:"max_item_pct"`       // 0..100, max share of capital per item
	SecZones        []string `json:"sec_zones"`          // "high","low","null"
} // @name PortfolioRequest

// PortfolioItem is one allocated position in the suggested portfolio.
type PortfolioItem struct {
	TypeID      int     `json:"type_id"`
	Name        string  `json:"name"`
	CapitalUsed float64 `json:"capital_used"`
	Units       int     `json:"units"`
	TripsPerDay float64 `json:"trips_per_day"`
	DailyProfit float64 `json:"daily_profit"`
	ROIPercent  float64 `json:"roi_percent"` // daily_profit / capital_used * 100
} // @name PortfolioItem

// PortfolioResult is the response.
type PortfolioResult struct {
	Items                []PortfolioItem `json:"items"`
	TotalCapitalUsed     float64         `json:"total_capital_used"`
	TotalDailyProfit     float64         `json:"total_daily_profit"`
	TimeUsedMin          float64         `json:"time_used_min"`
	DiversificationScore int             `json:"diversification_score"` // 0..100
	SkillsApplied        SkillsApplied   `json:"skills_applied"`        // reuse from hub_comparison.go
} // @name PortfolioResult
```

- [ ] **Step 2: Build** — `cd backend && go build ./...` → no errors. **Commit:** `git add backend/internal/models/portfolio.go && git commit -m "feat(portfolio): models for ROI calculator (#44)"`

---

## Task 2: PortfolioOptimizerService (greedy allocator) — TDD

**Files:** Create `backend/internal/services/portfolio_service.go`, `backend/internal/services/portfolio_service_test.go`

The optimizer works over a candidate slice (decoupled from the route engine for testability):

```go
// Candidate is one tradeable item the optimizer may allocate to.
type Candidate struct {
	TypeID         int
	Name           string
	ProfitPerUnit  float64 // net profit per unit (skill-adjusted, from route calc)
	BuyPricePerUnit float64 // capital cost per unit
	UnitVolume     float64 // m³ per unit
	DailyVolume    float64 // market daily volume (for liquidity cap)
	TripMinutes    float64 // round-trip minutes for this route
}
```

- [ ] **Step 1: Write failing tests** in `portfolio_service_test.go`

```go
package services

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func cand(id int, name string, profit, buy, vol, daily, tripMin float64) Candidate {
	return Candidate{TypeID: id, Name: name, ProfitPerUnit: profit, BuyPricePerUnit: buy, UnitVolume: vol, DailyVolume: daily, TripMinutes: tripMin}
}

func TestOptimize_RespectsCapitalAndPerItemCap(t *testing.T) {
	opt := NewPortfolioOptimizer()
	// Two equally efficient items; max 50% capital per item forces a split.
	cands := []Candidate{
		cand(1, "A", 10, 100, 1, 1e9, 10),
		cand(2, "B", 10, 100, 1, 1e9, 10),
	}
	res := opt.Optimize(cands, OptimizeParams{
		Capital: 1000, CargoCapacity: 1e9, TimeBudgetMin: 1e9,
		LiquidityCapPct: 100, MaxItemPct: 50,
	})
	if len(res.Items) != 2 {
		t.Fatalf("want 2 items, got %d", len(res.Items))
	}
	for _, it := range res.Items {
		if it.CapitalUsed > 500.0001 {
			t.Errorf("per-item cap (50%% of 1000=500) violated: %v", it.CapitalUsed)
		}
	}
	if !approx(res.TotalCapitalUsed, 1000) {
		t.Errorf("should use full capital, got %v", res.TotalCapitalUsed)
	}
}

func TestOptimize_LiquidityCapLimitsUnits(t *testing.T) {
	opt := NewPortfolioOptimizer()
	// Daily volume 100, cap 10% → at most 10 units/day regardless of capital.
	cands := []Candidate{cand(1, "A", 5, 10, 1, 100, 1)}
	res := opt.Optimize(cands, OptimizeParams{
		Capital: 1e9, CargoCapacity: 1e9, TimeBudgetMin: 1e9,
		LiquidityCapPct: 10, MaxItemPct: 100,
	})
	if res.Items[0].Units > 10 {
		t.Errorf("liquidity cap (10%% of 100=10) violated: %d units", res.Items[0].Units)
	}
}

func TestOptimize_TimeBudgetLimitsTrips(t *testing.T) {
	opt := NewPortfolioOptimizer()
	// 10-min trips, 25-min budget → at most 2 trips total.
	cands := []Candidate{cand(1, "A", 5, 10, 1, 1e9, 10)}
	res := opt.Optimize(cands, OptimizeParams{
		Capital: 1e9, CargoCapacity: 100, TimeBudgetMin: 25,
		LiquidityCapPct: 100, MaxItemPct: 100,
	})
	if res.TimeUsedMin > 25.0001 {
		t.Errorf("time budget exceeded: %v", res.TimeUsedMin)
	}
	if res.Items[0].TripsPerDay > 2 {
		t.Errorf("trips should be capped at 2, got %v", res.Items[0].TripsPerDay)
	}
}

func TestOptimize_DiversificationScore(t *testing.T) {
	opt := NewPortfolioOptimizer()
	// One dominating item → low score; two equal → higher.
	one := opt.Optimize([]Candidate{cand(1, "A", 10, 100, 1, 1e9, 10)},
		OptimizeParams{Capital: 1000, CargoCapacity: 1e9, TimeBudgetMin: 1e9, LiquidityCapPct: 100, MaxItemPct: 100})
	two := opt.Optimize([]Candidate{cand(1, "A", 10, 100, 1, 1e9, 10), cand(2, "B", 10, 100, 1, 1e9, 10)},
		OptimizeParams{Capital: 1000, CargoCapacity: 1e9, TimeBudgetMin: 1e9, LiquidityCapPct: 100, MaxItemPct: 50})
	if !(two.DiversificationScore > one.DiversificationScore) {
		t.Errorf("two-item portfolio should score higher: one=%d two=%d", one.DiversificationScore, two.DiversificationScore)
	}
}

func TestOptimize_EmptyWhenCapitalTooSmall(t *testing.T) {
	opt := NewPortfolioOptimizer()
	res := opt.Optimize([]Candidate{cand(1, "A", 5, 1000, 1, 1e9, 1)},
		OptimizeParams{Capital: 100, CargoCapacity: 1e9, TimeBudgetMin: 1e9, LiquidityCapPct: 100, MaxItemPct: 100})
	if len(res.Items) != 0 {
		t.Errorf("want empty portfolio when capital < one unit, got %d items", len(res.Items))
	}
}
```

- [ ] **Step 2: Run, verify fail** — `cd backend && go test ./internal/services/ -run TestOptimize` → FAIL (undefined `NewPortfolioOptimizer`).

- [ ] **Step 3: Implement `portfolio_service.go`**

```go
package services

import (
	"math"
	"sort"
)

type Candidate struct {
	TypeID          int
	Name            string
	ProfitPerUnit   float64
	BuyPricePerUnit float64
	UnitVolume      float64
	DailyVolume     float64
	TripMinutes     float64
}

type OptimizeParams struct {
	Capital         float64
	CargoCapacity   float64
	TimeBudgetMin   float64
	LiquidityCapPct float64 // 0..100
	MaxItemPct      float64 // 0..100
}

type OutcomeItem struct {
	TypeID      int
	Name        string
	CapitalUsed float64
	Units       int
	TripsPerDay float64
	DailyProfit float64
}

type PortfolioOutcome struct {
	Items                []OutcomeItem
	TotalCapitalUsed     float64
	TotalDailyProfit     float64
	TimeUsedMin          float64
	DiversificationScore int
}

type PortfolioOptimizer struct{}

func NewPortfolioOptimizer() *PortfolioOptimizer { return &PortfolioOptimizer{} }

// Optimize greedily allocates capital + a shared time budget across candidates,
// most-efficient (profit per ISK) first, under per-item liquidity and capital caps.
func (o *PortfolioOptimizer) Optimize(cands []Candidate, p OptimizeParams) PortfolioOutcome {
	perItemCapital := p.Capital * p.MaxItemPct / 100.0

	type scored struct {
		c            Candidate
		unitsPerTrip int
		maxUnits     int // min(liquidity cap, per-item capital cap)
		efficiency   float64
	}
	var list []scored
	for _, c := range cands {
		if c.BuyPricePerUnit <= 0 || c.ProfitPerUnit <= 0 {
			continue
		}
		upt := 1
		if c.UnitVolume > 0 {
			upt = int(p.CargoCapacity / c.UnitVolume)
		}
		if upt < 1 {
			continue // doesn't fit cargo
		}
		maxUnits := int(c.DailyVolume * p.LiquidityCapPct / 100.0)
		if capCap := int(perItemCapital / c.BuyPricePerUnit); capCap < maxUnits {
			maxUnits = capCap
		}
		if maxUnits < 1 {
			continue
		}
		list = append(list, scored{c: c, unitsPerTrip: upt, maxUnits: maxUnits, efficiency: c.ProfitPerUnit / c.BuyPricePerUnit})
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].efficiency > list[j].efficiency })

	capitalLeft, timeLeft := p.Capital, p.TimeBudgetMin
	var items []OutcomeItem
	var totalCap, totalProfit, totalTime float64

	for _, s := range list {
		units := s.maxUnits
		if affordable := int(capitalLeft / s.c.BuyPricePerUnit); affordable < units {
			units = affordable
		}
		if units < 1 {
			continue
		}
		trips := ceilDiv(units, s.unitsPerTrip)
		if s.c.TripMinutes > 0 {
			if affordTrips := int(timeLeft / s.c.TripMinutes); affordTrips < trips {
				trips = affordTrips
				if t := trips * s.unitsPerTrip; t < units {
					units = t
				}
			}
		}
		if units < 1 || trips < 1 {
			continue
		}
		capUsed := float64(units) * s.c.BuyPricePerUnit
		profit := float64(units) * s.c.ProfitPerUnit
		timeUsed := float64(trips) * s.c.TripMinutes
		items = append(items, OutcomeItem{
			TypeID: s.c.TypeID, Name: s.c.Name, CapitalUsed: capUsed,
			Units: units, TripsPerDay: float64(trips), DailyProfit: profit,
		})
		capitalLeft -= capUsed
		timeLeft -= timeUsed
		totalCap += capUsed
		totalProfit += profit
		totalTime += timeUsed
	}

	return PortfolioOutcome{
		Items: items, TotalCapitalUsed: totalCap, TotalDailyProfit: totalProfit,
		TimeUsedMin: totalTime, DiversificationScore: diversificationScore(items, totalCap),
	}
}

func ceilDiv(a, b int) int {
	if b <= 0 {
		return a
	}
	return (a + b - 1) / b
}

// diversificationScore is Herfindahl-based: (1 - Σ share²) × 100, rounded.
// 0/1 items → 0 (no diversification).
func diversificationScore(items []OutcomeItem, total float64) int {
	if len(items) < 2 || total <= 0 {
		return 0
	}
	var sumSq float64
	for _, it := range items {
		share := it.CapitalUsed / total
		sumSq += share * share
	}
	return int(math.Round((1 - sumSq) * 100))
}
```

- [ ] **Step 4: Run, verify pass** — `go test ./internal/services/ -run TestOptimize -v` → all PASS. Fix logic until green.

- [ ] **Step 5: gofmt + commit** — `gofmt -w internal/services/portfolio_service*.go && git add backend/internal/services/portfolio_service*.go && git commit -m "feat(portfolio): greedy capital-allocation optimizer + tests (#44)"`

---

## Task 3: Handler + route + wiring

**Files:** Create `backend/internal/handlers/portfolio.go`; Modify `backend/cmd/api/container.go`, `backend/cmd/api/main.go`

- [ ] **Step 1: Implement `portfolio.go`** — mirror `internal/handlers/multi_hub.go`: parse `PortfolioRequest`, validate `RegionID>0 && ShipTypeID>0 && Capital>0`, extract `character_id`/`access_token` from `c.Locals` (type-assert `int`/`string`, 401 if missing), call a service method `Optimize(ctx, req, characterID, accessToken) (*models.PortfolioResult, error)`, return JSON; 500 on error (no raw error leak).

- [ ] **Step 2: Wire the service** — create a thin `PortfolioService` in `portfolio_service.go` that: calls `routeService.CalculateWithFilters` to get candidate routes, maps each route → `Candidate` (ProfitPerUnit, BuyPricePerUnit, UnitVolume, DailyVolume from volume metrics, TripMinutes from route travel time), runs `PortfolioOptimizer.Optimize`, fetches skills once for `SkillsApplied`, maps `PortfolioOutcome` → `models.PortfolioResult` (incl. ROIPercent = dailyProfit/capitalUsed*100). Inject `RouteCalculatorServicer` + `SkillsServicer` via small interfaces (testable). In `container.go` build it with the existing `routeService` + `skillsService` and `NewPortfolioOptimizer()`; build `c.PortfolioHandler`.

- [ ] **Step 3: Register route** in `main.go` after the hubs route:
```go
api.Post("/trading/portfolio/optimize", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.PortfolioHandler.Optimize)
```

- [ ] **Step 4: Build + vet** — `go build ./... && go vet ./internal/... ./cmd/...` → clean.

- [ ] **Step 5: Commit** — `git add backend/ && git commit -m "feat(portfolio): handler + route + wiring (#44)"`

---

## Task 4: Backend gates

- [ ] **Step 1:** `cd backend && gofmt -l internal/ cmd/` → empty; `go test -short ./internal/...` → PASS.
- [ ] **Step 2:** Bump `CHANGELOG.md` `[Unreleased]` with an `### Added` entry describing the portfolio endpoint.
- [ ] **Step 3: Commit** — `git add backend CHANGELOG.md && git commit -m "chore(portfolio): changelog + gates (#44)"`

---

## Task 5: Web frontend + tests

**Files:** as in File Structure. Follow the #43 patterns exactly: `src/app/multi-hub/page.tsx` (page structure, useMutation, auth gate), `src/lib/api-client.ts` `compareHubs` (authed POST, `credentials:"include"`), `src/components/trading/HubComparisonTable.tsx` (table idiom), `tests/components/HubComparisonTable.test.tsx` + `tests/e2e/auth/multi-hub.spec.ts`.

- [ ] **Step 1:** Add types to `src/types/trading.ts`: `PortfolioRequest`, `PortfolioItem`, `PortfolioResult` (snake_case → camel per existing convention; mirror the Go JSON tags).
- [ ] **Step 2:** Add `optimizePortfolio(req: PortfolioRequest)` to `src/lib/api-client.ts` (POST `/api/v1/trading/portfolio/optimize`, `credentials:"include"`).
- [ ] **Step 3:** `PortfolioResultTable.tsx` (columns: Item, Kapital, Units, Fahrten/Tag, Tagesgewinn, ROI%) + `DiversificationScore.tsx` (0–100 badge) + `PortfolioInputForm.tsx` (region, ship, capital, time, liquidity cap, max %/item, sec-zone checkboxes). Use `formatISKWithSeparators` for ISK.
- [ ] **Step 4:** Replace `src/app/roi-calculator/page.tsx` placeholder: auth gate + ErrorBoundary + form → `useMutation(optimizePortfolio)` → results (table + totals + diversification). Loading/error/empty states ("kein sinnvolles Portfolio für dieses Budget").
- [ ] **Step 5:** Vitest `tests/components/PortfolioResultTable.test.tsx` (renders allocations, totals, ROI%, empty-state) + Playwright `tests/e2e/auth/roi-calculator.spec.ts` (authed; fill form, submit, assert allocation table). Run `npm run test` + `npm run lint` → green.
- [ ] **Step 6: Commit** — `git add frontend && git commit -m "feat(frontend): ROI calculator page + tests (#44)"`

---

## Task 6: Flutter + tests

**Files:** as in File Structure. Follow the #43 `hub_comparison_*` patterns exactly.

- [ ] **Step 1:** `lib/api/portfolio_models.dart` — DTOs with null/float-robust `fromJson` (mirror `hub_comparison_models.dart`).
- [ ] **Step 2:** `lib/api/trading_api.dart` add `optimizePortfolio(PortfolioRequest)`; `lib/features/trading/roi_providers.dart` `AsyncNotifier<PortfolioResult?>`.
- [ ] **Step 3:** `lib/features/trading/roi_calculator_screen.dart` adaptive via `isTwoPane(840)`: input form + result table (Item/Kapital/Units/Fahrten/Tagesgewinn/ROI%) + diversification + empty-state.
- [ ] **Step 4:** `lib/core/router.dart` add `/roi-calculator` GoRoute + a NavigationDestination ("ROI").
- [ ] **Step 5:** Tests — `test/portfolio_models_test.dart` (parse incl. empty items), `test/roi_calculator_screen_layout_test.dart` (1-/2-pane), group in `test/e2e/app_flow_test.dart` + `fakePortfolioResponse()` in `test/e2e/support/fakes.dart`. Run `flutter analyze` (clean) + `flutter test` (all pass).
- [ ] **Step 6: Commit** — `git add app && git commit -m "feat(app): ROI calculator screen + tests (#44)"`

---

## Task 7: Ship

- [ ] **Step 1:** PR `feat/roi-calculator` → CI green → merge.
- [ ] **Step 2:** `make release VERSION=x.y.0` → release PR → merge → tag `vx.y.0` → deploy (backend + frontend images) → smoke success.
- [ ] **Step 3:** Prod verify: `/version`, `/roi-calculator` 200, `POST /portfolio/optimize` 401 without auth (wired). Flutter APK rebuild + on-device happy path.
- [ ] **Step 4:** Comment on #43-style summary on issue #44; close.

---

## Self-Review

- **Spec coverage:** ROI per item (Task 2 efficiency + ROIPercent map in Task 3) ✓; ROI ranking (sorted greedy + result order) ✓; portfolio builder/allocation (Task 2) ✓; daily profit projection (TotalDailyProfit) ✓; diversification score (Task 2) ✓; skills applied (Task 3 fetch) ✓; inputs incl. liquidity cap + sec zones + max%/item (Task 1 request + Task 2 caps) ✓; web + flutter (Tasks 5,6) ✓.
- **Placeholders:** none — Task 2 Step 3 now contains the complete `Optimize` implementation (greedy, caps, `ceilDiv`, `diversificationScore`). Task 3 describes the handler/wiring by pointing at the concrete `multi_hub.go` analogue (same shape, fully specified), which is reuse-not-placeholder.
- **Type consistency:** `Candidate`, `OptimizeParams`, `PortfolioOutcome`/`OutcomeItem` (internal) vs `models.Portfolio*` (API) are distinct by design; handler/service maps between them (Task 3 Step 2). `optimizePortfolio` name consistent web+flutter.
