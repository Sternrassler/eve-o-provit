package services

import (
	"context"
	"math"
	"sort"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// PortfolioService orchestrates ROI/capital-allocation: it pulls candidate routes
// from the route engine, applies the sec-zone filter, runs the greedy optimizer,
// and maps the result to the API model with skills metadata (Issue #44).
type PortfolioService struct {
	routes    RouteCalculatorServicer
	skills    SkillsServicer
	optimizer *PortfolioOptimizer
	logger    *logger.Logger
}

// NewPortfolioService creates a new portfolio service.
func NewPortfolioService(routes RouteCalculatorServicer, skills SkillsServicer, logger *logger.Logger) *PortfolioService {
	return &PortfolioService{routes: routes, skills: skills, optimizer: NewPortfolioOptimizer(), logger: logger}
}

// Optimize computes the suggested capital allocation for the request.
func (s *PortfolioService) Optimize(ctx context.Context, req *models.PortfolioRequest, characterID int, accessToken string) (*models.PortfolioResult, error) {
	// Route calc needs the character in context for skill-aware cargo/fees.
	ctx = context.WithValue(ctx, contextKeyCharacterID, characterID)
	ctx = context.WithValue(ctx, contextKeyAccessToken, accessToken)

	resp, err := s.routes.CalculateWithFilters(ctx, &models.RouteCalculationRequest{
		RegionID:             req.RegionID,
		ShipTypeID:           req.ShipTypeID,
		IncludeVolumeMetrics: true,
	})
	if err != nil {
		return nil, err
	}

	cands := make([]Candidate, 0, len(resp.Routes))
	for i := range resp.Routes {
		r := resp.Routes[i]
		if !secZoneAllowed(r.MinRouteSecurityStatus, req.SecZones) {
			continue
		}
		dailyVol := 0.0
		if r.VolumeMetrics != nil {
			dailyVol = r.VolumeMetrics.DailyVolumeAvg
		}
		cands = append(cands, Candidate{
			TypeID:          r.ItemTypeID,
			Name:            r.ItemName,
			ProfitPerUnit:   r.ProfitPerUnit,
			BuyPricePerUnit: r.BuyPrice,
			UnitVolume:      r.ItemVolume,
			DailyVolume:     dailyVol,
			TripMinutes:     r.RoundTripSeconds / 60.0,
			BuySystemName:   r.BuySystemName,
			BuyStationName:  r.BuyStationName,
			BuyStationID:    r.BuyStationID,
			SellSystemName:  r.SellSystemName,
			SellStationName: r.SellStationName,
			SellStationID:   r.SellStationID,
		})
	}

	outcome := s.optimizer.Optimize(cands, OptimizeParams{
		Capital:         req.Capital,
		CargoCapacity:   resp.CargoCapacity,
		TimeBudgetMin:   req.TimeBudgetMin,
		LiquidityCapPct: req.LiquidityCapPct,
		MaxItemPct:      req.MaxItemPct,
	})

	skills, serr := s.skills.GetCharacterSkills(ctx, characterID, accessToken)
	if serr != nil || skills == nil {
		skills = &TradingSkills{}
	}

	items := make([]models.PortfolioItem, 0, len(outcome.Items))
	for _, it := range outcome.Items {
		roi := 0.0
		if it.CapitalUsed > 0 {
			roi = it.DailyProfit / it.CapitalUsed * 100
		}
		items = append(items, models.PortfolioItem{
			TypeID:          it.TypeID,
			Name:            it.Name,
			CapitalUsed:     it.CapitalUsed,
			Units:           it.Units,
			TripsPerDay:     it.TripsPerDay,
			DailyProfit:     it.DailyProfit,
			ROIPercent:      roi,
			BuySystemName:   it.BuySystemName,
			BuyStationName:  it.BuyStationName,
			BuyStationID:    it.BuyStationID,
			SellSystemName:  it.SellSystemName,
			SellStationName: it.SellStationName,
			SellStationID:   it.SellStationID,
		})
	}

	return &models.PortfolioResult{
		Items:                items,
		TotalCapitalUsed:     outcome.TotalCapitalUsed,
		TotalDailyProfit:     outcome.TotalDailyProfit,
		TimeUsedMin:          outcome.TimeUsedMin,
		DiversificationScore: outcome.DiversificationScore,
		SkillsApplied: models.SkillsApplied{
			Applied:         serr == nil,
			Accounting:      skills.Accounting,
			BrokerRelations: skills.BrokerRelations,
			SalesTaxRate:    SalesTaxRate(skills.Accounting),
			BrokerFeeRate:   BrokerFeeRate(skills.BrokerRelations, skills.AdvancedBrokerRelations, skills.FactionStanding, skills.CorpStanding),
		},
	}, nil
}

// secZoneAllowed reports whether a route's minimum security status falls within the
// requested zones (empty = all allowed). high ≥0.5, low 0–0.5, null ≤0.0.
func secZoneAllowed(minSec float64, zones []string) bool {
	if len(zones) == 0 {
		return true
	}
	var zone string
	switch {
	case minSec >= 0.5:
		zone = "high"
	case minSec > 0.0:
		zone = "low"
	default:
		zone = "null"
	}
	for _, z := range zones {
		if z == zone {
			return true
		}
	}
	return false
}

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
	// Execution location (buy at the buy station, sell at the sell station).
	BuySystemName   string
	BuyStationName  string
	BuyStationID    int64
	SellSystemName  string
	SellStationName string
	SellStationID   int64
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
	// Execution location, carried through from the candidate.
	BuySystemName   string
	BuyStationName  string
	BuyStationID    int64
	SellSystemName  string
	SellStationName string
	SellStationID   int64
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
			BuySystemName: s.c.BuySystemName, BuyStationName: s.c.BuyStationName, BuyStationID: s.c.BuyStationID,
			SellSystemName: s.c.SellSystemName, SellStationName: s.c.SellStationName, SellStationID: s.c.SellStationID,
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
