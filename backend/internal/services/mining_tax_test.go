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

func TestBestStation_PicksLowestTax(t *testing.T) {
	stations := []StationStanding{
		{StationID: 1, BaseTake: 0.05, Standing: 0},
		{StationID: 2, BaseTake: 0.05, Standing: 6.67},
	}
	best := BestStation(stations)
	if best == nil || best.StationID != 2 {
		t.Fatalf("expected station 2 (zero tax), got %+v", best)
	}
}
