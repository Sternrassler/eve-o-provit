package services

import "testing"

func TestReprocessTax(t *testing.T) {
	if v := ReprocessTax(0.05, 0); v != 0.05 {
		t.Fatalf("standing 0 → %v", v)
	}
	if v := ReprocessTax(0.05, 4); v < 0.0199 || v > 0.0201 { // 0.05 - 0.03
		t.Fatalf("standing 4 → %v", v)
	}
	if v := ReprocessTax(0.05, 10); v != 0 {
		t.Fatalf("standing 10 → %v", v)
	}
}

func TestBestStation_MaximizesNetYield(t *testing.T) {
	// Same rate, station 2 has zero tax (standing ≥ 6.67) → more net material.
	stations := []StationStanding{
		{StationID: 1, BaseRate: 0.50, BaseTake: 0.05, Standing: 0},
		{StationID: 2, BaseRate: 0.50, BaseTake: 0.05, Standing: 6.67},
	}
	if best := BestStation(stations); best == nil || best.StationID != 2 {
		t.Fatalf("expected station 2 (zero tax), got %+v", best)
	}
}

func TestBestStation_HigherRateBeatsLowerTax(t *testing.T) {
	// Station A: rate 0.50, tax 0.05 → net 0.475. Station B: rate 0.35, tax 0 → net 0.35.
	// A wins despite higher tax, because it refines more.
	stations := []StationStanding{
		{StationID: 1, BaseRate: 0.50, BaseTake: 0.05, Standing: 0},
		{StationID: 2, BaseRate: 0.35, BaseTake: 0.0, Standing: 0},
	}
	if best := BestStation(stations); best == nil || best.StationID != 1 {
		t.Fatalf("expected station 1 (higher net yield), got %+v", best)
	}
}
