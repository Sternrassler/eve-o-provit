package services

import (
	"math"
	"sort"
)

// Candidate is one tradeable item the optimizer may allocate to (derived from a
// route-engine result, decoupled for testability).
type Candidate struct {
	TypeID          int
	Name            string
	ProfitPerUnit   float64 // net profit per unit (skill-adjusted)
	BuyPricePerUnit float64 // capital cost per unit
	UnitVolume      float64 // m³ per unit
	DailyVolume     float64 // market daily volume (for liquidity cap)
	TripMinutes     float64 // round-trip minutes for this route
}

// OptimizeParams are the allocation constraints.
type OptimizeParams struct {
	Capital         float64
	CargoCapacity   float64
	TimeBudgetMin   float64
	LiquidityCapPct float64 // 0..100
	MaxItemPct      float64 // 0..100
}

// OutcomeItem is one allocated position (internal; mapped to models.PortfolioItem).
type OutcomeItem struct {
	TypeID      int
	Name        string
	CapitalUsed float64
	Units       int
	TripsPerDay float64
	DailyProfit float64
}

// PortfolioOutcome is the optimizer result (internal).
type PortfolioOutcome struct {
	Items                []OutcomeItem
	TotalCapitalUsed     float64
	TotalDailyProfit     float64
	TimeUsedMin          float64
	DiversificationScore int
}

// PortfolioOptimizer allocates capital across candidate items.
type PortfolioOptimizer struct{}

// NewPortfolioOptimizer creates a new optimizer.
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
