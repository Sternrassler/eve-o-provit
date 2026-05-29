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
	load := opt.FillCargo([]HaulItem{
		hitem(1, "A", 100, 20, 1, 100), // profit/m³ = 20
		hitem(2, "B", 50, 5, 1, 100),   // profit/m³ = 5
	}, 10, 1000)
	if len(load.Items) == 0 || load.Items[0].TypeID != 1 || load.Items[0].Quantity != 10 {
		t.Fatalf("expected 10x A first, got %+v", load.Items)
	}
	if !happrox(load.TotalVolume, 10) || !happrox(load.TotalProfit, 200) {
		t.Errorf("totals wrong: %+v", load)
	}
}

func TestFillCargo_CapitalBinds(t *testing.T) {
	opt := NewHaulingOptimizer()
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
