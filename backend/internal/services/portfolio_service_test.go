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
