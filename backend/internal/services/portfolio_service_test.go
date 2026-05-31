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

// TestOptimize_SortsResultByISKPerHour: the displayed list must be ordered by
// ISK/h descending, even when the internal allocation order (by efficiency)
// would put items differently. Two items, same daily profit, but item B has
// half the trip time → item B must come first.
func TestOptimize_SortsResultByISKPerHour(t *testing.T) {
	opt := NewPortfolioOptimizer()
	slow := cand(1, "Slow", 100, 1, 0, 1e9, 60) // 60 min/trip
	fast := cand(2, "Fast", 100, 1, 0, 1e9, 10) // 10 min/trip
	res := opt.Optimize([]Candidate{slow, fast}, OptimizeParams{
		Capital: 1e9, CargoCapacity: 1e9, TimeBudgetMin: 1e9,
		LiquidityCapPct: 100, MaxItemPct: 50,
	})
	if len(res.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(res.Items))
	}
	if res.Items[0].Name != "Fast" || res.Items[1].Name != "Slow" {
		t.Errorf("items must be sorted by ISK/h desc; got %s, %s", res.Items[0].Name, res.Items[1].Name)
	}
	if res.Items[0].ISKPerHour <= res.Items[1].ISKPerHour {
		t.Errorf("ISK/h not descending: %v then %v", res.Items[0].ISKPerHour, res.Items[1].ISKPerHour)
	}
}

// TestOptimize_RespectsSellSideAvailability is a focused variant of the
// general order-book test: even if the cheapest sell-order has unlimited
// volume, the buyer-side `highest_buy.VolumeRemain` (already baked into
// route_finder's AvailableQuantity) must cap the recommendation.
// MaxAvailableUnits encodes that joint min(buy, sell) limit.
func TestOptimize_RespectsSellSideAvailability(t *testing.T) {
	opt := NewPortfolioOptimizer()
	c := cand(1, "Carbon", 300, 85, 0.1, 1_000_000, 10)
	c.MaxAvailableUnits = 5 // imagine: cheapest sell has 1000, but highest buy only wants 5
	res := opt.Optimize([]Candidate{c}, OptimizeParams{
		Capital: 1e9, CargoCapacity: 1e9, TimeBudgetMin: 1e9,
		LiquidityCapPct: 100, MaxItemPct: 100,
	})
	if len(res.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.Items))
	}
	if res.Items[0].Units > 5 {
		t.Errorf("sell-side cap violated: got %d units, want ≤ 5", res.Items[0].Units)
	}
}

// TestOptimize_CapsAtOrderBookAvailability covers the case we hit on prod:
// the cheapest sell-order tier has only N units, but the optimizer was
// extrapolating that price across hundreds of units (capital/price). With
// MaxAvailableUnits set from the route's order-book limit, the allocation
// must not exceed N — otherwise the projected ROI is fiction.
func TestOptimize_CapsAtOrderBookAvailability(t *testing.T) {
	opt := NewPortfolioOptimizer()
	c := cand(1, "Carbon", 300, 85, 0.1, 1_000_000, 10) // huge daily volume, big capital
	c.MaxAvailableUnits = 2                             // but only 2 units at this price
	res := opt.Optimize([]Candidate{c}, OptimizeParams{
		Capital: 1e9, CargoCapacity: 1e9, TimeBudgetMin: 1e9,
		LiquidityCapPct: 100, MaxItemPct: 100,
	})
	if len(res.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.Items))
	}
	if res.Items[0].Units > 2 {
		t.Errorf("order-book cap violated: got %d units, want ≤ 2", res.Items[0].Units)
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
	if two.DiversificationScore <= one.DiversificationScore {
		t.Errorf("two-item portfolio should score higher: one=%d two=%d", one.DiversificationScore, two.DiversificationScore)
	}
}

func TestOptimize_CarriesBuySellLocation(t *testing.T) {
	opt := NewPortfolioOptimizer()
	c := cand(1, "A", 10, 100, 1, 1e9, 10)
	c.BuyStationName, c.BuyStationID, c.BuySystemName = "Jita IV-4", 60003760, "Jita"
	c.SellStationName, c.SellStationID, c.SellSystemName = "Rens VI-8", 60004588, "Rens"
	res := opt.Optimize([]Candidate{c},
		OptimizeParams{Capital: 1000, CargoCapacity: 1e9, TimeBudgetMin: 1e9, LiquidityCapPct: 100, MaxItemPct: 100})
	if len(res.Items) != 1 {
		t.Fatalf("want 1 item, got %d", len(res.Items))
	}
	it := res.Items[0]
	if it.BuyStationID != 60003760 || it.SellStationID != 60004588 || it.BuySystemName != "Jita" || it.SellSystemName != "Rens" {
		t.Errorf("buy/sell location not carried through: %+v", it)
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
