package services

import (
	"math"
	"testing"
)

func TestKnapsackDP_RoundsVolumeNotTruncate(t *testing.T) {
	svc := &CargoService{}
	items := []CargoItem{{TypeID: 1, Volume: 0.998, Value: 100, Quantity: 100}}
	sol := svc.KnapsackDP(items, 10.0)
	if len(sol.Items) == 0 || sol.Items[0].Quantity != 10 {
		got := 0
		if len(sol.Items) > 0 {
			got = sol.Items[0].Quantity
		}
		t.Errorf("Quantity = %d, want 10 (0.998 korrekt gerundet: 10*0.998=9.98 ≤ 10)", got)
	}
}

func TestKnapsackDP_UsesGreedyForHugeCapacity(t *testing.T) {
	svc := &CargoService{}
	items := []CargoItem{
		{TypeID: 1, Volume: 100, Value: 1000, Quantity: 5000},
		{TypeID: 2, Volume: 50, Value: 400, Quantity: 5000},
	}
	sol := svc.KnapsackDP(items, 1_000_000.0)
	if sol.TotalVolume <= 0 || sol.TotalValue <= 0 {
		t.Errorf("Greedy-Pfad lieferte leeres Ergebnis: vol=%.2f val=%.2f", sol.TotalVolume, sol.TotalValue)
	}
	if sol.TotalVolume > 1_000_000.0+0.01 {
		t.Errorf("TotalVolume %.2f überschreitet Kapazität", sol.TotalVolume)
	}
}

func TestKnapsackDP_BacktrackBillionsISK(t *testing.T) {
	svc := &CargoService{}
	items := []CargoItem{
		{TypeID: 1, Volume: 1.0, Value: 2_000_000_000, Quantity: 3},
		{TypeID: 2, Volume: 1.0, Value: 1_000_000_000, Quantity: 3},
	}
	sol := svc.KnapsackDP(items, 5.0)
	if math.Abs(sol.TotalValue-8_000_000_000) > 1.0 {
		t.Errorf("TotalValue = %.0f, want 8e9 (3×2Mrd + 2×1Mrd)", sol.TotalValue)
	}
}
