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

func TestKnapsackDP_RoundsVolumeNotTruncate_F7(t *testing.T) {
	// v=0.999: int(0.999*100)=99 (truncate) erlaubt 5 Stück (5×0.999=4.995 > 4.99 → Overflow!),
	// int(math.Round(0.999*100))=100 begrenzt korrekt auf 4 (4×0.999=3.996 ≤ 4.99).
	svc := &CargoService{}
	items := []CargoItem{{TypeID: 1, Volume: 0.999, Value: 100, Quantity: 10}}
	sol := svc.KnapsackDP(items, 4.99)
	if len(sol.Items) == 0 {
		t.Fatal("no items selected")
	}
	qty := sol.Items[0].Quantity
	if qty != 4 {
		t.Errorf("Quantity = %d, want 4; Truncation würde 5 ergeben (Volumen-Overflow)", qty)
	}
	if sol.TotalVolume > 4.99+1e-9 {
		t.Errorf("TotalVolume %.4f überschreitet Kapazität 4.99 (Rundungsbug)", sol.TotalVolume)
	}
}
