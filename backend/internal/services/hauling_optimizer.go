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
	Items        []HaulLoadItem
	TotalProfit  float64
	TotalCapital float64
	TotalVolume  float64
}

// HaulingOptimizer fills one ship for one route greedily by profit per m³.
type HaulingOptimizer struct{}

// NewHaulingOptimizer creates a new optimizer.
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
