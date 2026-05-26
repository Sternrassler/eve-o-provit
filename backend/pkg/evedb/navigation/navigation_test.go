package navigation

import (
	"database/sql"
	"math"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestCalculateWarpTime(t *testing.T) {
	tests := []struct {
		name        string
		distanceAU  float64
		warpSpeedAU float64
		wantRange   [2]float64 // min and max expected values
	}{
		{
			name:        "Cruiser 50 AU",
			distanceAU:  50.0,
			warpSpeedAU: 3.0,
			wantRange:   [2]float64{40.0, 55.0}, // Approximate range
		},
		{
			name:        "Interceptor 50 AU",
			distanceAU:  50.0,
			warpSpeedAU: 6.0,
			wantRange:   [2]float64{20.0, 30.0},
		},
		{
			name:        "Battleship 50 AU",
			distanceAU:  50.0,
			warpSpeedAU: 1.5,
			wantRange:   [2]float64{85.0, 100.0},
		},
		{
			name:        "Short warp 5 AU",
			distanceAU:  5.0,
			warpSpeedAU: 3.0,
			wantRange:   [2]float64{20.0, 35.0}, // Short warps still need accel/decel phases
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateWarpTime(tt.distanceAU, tt.warpSpeedAU)
			if got < tt.wantRange[0] || got > tt.wantRange[1] {
				t.Errorf("CalculateWarpTime() = %v, want in range [%v, %v]", got, tt.wantRange[0], tt.wantRange[1])
			}
		})
	}
}

func TestCalculateSimplifiedWarpTime(t *testing.T) {
	tests := []struct {
		name        string
		distanceAU  float64
		warpSpeedAU float64
		want        float64
	}{
		{
			name:        "Default cruiser 15 AU",
			distanceAU:  15.0,
			warpSpeedAU: 3.0,
			want:        7.0, // (15 / 3) * 1.4
		},
		{
			name:        "Interceptor 15 AU",
			distanceAU:  15.0,
			warpSpeedAU: 6.0,
			want:        3.5, // (15 / 6) * 1.4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSimplifiedWarpTime(tt.distanceAU, tt.warpSpeedAU)
			if math.Abs(got-tt.want) > 0.1 {
				t.Errorf("CalculateSimplifiedWarpTime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEffectiveParams(t *testing.T) {
	tests := []struct {
		name            string
		params          *NavigationParams
		wantWarpSpeed   float64
		wantAlignTime   float64
		wantAvgWarpDist float64
		wantSource      string
	}{
		{
			name:            "Nil params uses defaults",
			params:          nil,
			wantWarpSpeed:   DefaultWarpSpeed,
			wantAlignTime:   DefaultAlignTime,
			wantAvgWarpDist: DefaultAvgWarpDistance,
			wantSource:      "default",
		},
		{
			name: "Custom warp speed",
			params: &NavigationParams{
				WarpSpeed: ptrFloat64(6.0),
			},
			wantWarpSpeed:   6.0,
			wantAlignTime:   DefaultAlignTime,
			wantAvgWarpDist: DefaultAvgWarpDistance,
			wantSource:      "provided",
		},
		{
			name: "Calculated align time from ship params",
			params: &NavigationParams{
				ShipMass:        ptrFloat64(12000000),
				InertiaModifier: ptrFloat64(0.4),
			},
			wantWarpSpeed:   DefaultWarpSpeed,
			wantAlignTime:   6.65, // ln(2) × 0.4 × 12_000_000 / 500_000 ≈ 6.65s
			wantAvgWarpDist: DefaultAvgWarpDistance,
			wantSource:      "calculated",
		},
		{
			name: "All custom params",
			params: &NavigationParams{
				WarpSpeed:       ptrFloat64(8.0),
				AlignTime:       ptrFloat64(2.5),
				AvgWarpDistance: ptrFloat64(20.0),
			},
			wantWarpSpeed:   8.0,
			wantAlignTime:   2.5,
			wantAvgWarpDist: 20.0,
			wantSource:      "provided",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warpSpeed, alignTime, avgWarpDist, source := getEffectiveParams(tt.params)

			if math.Abs(warpSpeed-tt.wantWarpSpeed) > 0.1 {
				t.Errorf("warpSpeed = %v, want %v", warpSpeed, tt.wantWarpSpeed)
			}

			// Allow 10% tolerance for calculated align time
			tolerance := tt.wantAlignTime * 0.1
			if math.Abs(alignTime-tt.wantAlignTime) > tolerance {
				t.Errorf("alignTime = %v, want approximately %v", alignTime, tt.wantAlignTime)
			}

			if math.Abs(avgWarpDist-tt.wantAvgWarpDist) > 0.1 {
				t.Errorf("avgWarpDist = %v, want %v", avgWarpDist, tt.wantAvgWarpDist)
			}

			if source != tt.wantSource {
				t.Errorf("source = %v, want %v", source, tt.wantSource)
			}
		})
	}
}

// Helper function to create float64 pointers
func ptrFloat64(v float64) *float64 {
	return &v
}

// setupTestDataWithLowSecBypass creates a fixture with a topology that has a shorter route
// through a low-sec system and a longer all-high-sec route:
//
//	A(10,hs) --1--> L(11,ls) --1--> B(12,hs)   [2 jumps via low-sec]
//	A(10,hs) --1--> C(13,hs) --1--> D(14,hs) --1--> B(12,hs)  [3 jumps via high-sec]
//
// avoidLowSec=false → shortest path A→B = 2 jumps (A→L→B)
// avoidLowSec=true  → shortest path A→B = 3 jumps (A→C→D→B), L is excluded
func setupTestDataWithLowSecBypass(t *testing.T, db *sql.DB) {
	t.Helper()
	schema := `
		CREATE TABLE mapRegions (
			_key INTEGER PRIMARY KEY,
			name TEXT
		);
		CREATE TABLE mapConstellations (
			_key INTEGER PRIMARY KEY,
			regionID INTEGER,
			name TEXT
		);
		CREATE TABLE mapSolarSystems (
			_key INTEGER PRIMARY KEY,
			solarSystemID INTEGER,
			name TEXT,
			securityStatus REAL,
			constellationID INTEGER,
			regionID INTEGER,
			wormholeClassID INTEGER,
			border INTEGER,
			corridor INTEGER,
			hub INTEGER
		);
		CREATE TABLE mapStargates (
			_key INTEGER PRIMARY KEY,
			solarSystemID INTEGER,
			typeID INTEGER,
			destination TEXT
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("setupTestDataWithLowSecBypass: schema: %v", err)
	}
	testData := `
		INSERT INTO mapRegions (_key, name) VALUES (100, '{"en":"Region"}');
		INSERT INTO mapConstellations (_key, regionID, name) VALUES (200, 100, '{"en":"Const"}');

		-- Systems: A=10 (hs), L=11 (ls), B=12 (hs), C=13 (hs), D=14 (hs)
		INSERT INTO mapSolarSystems (_key, solarSystemID, name, securityStatus, constellationID, regionID, wormholeClassID, border, corridor, hub) VALUES
			(10, 10, '{"en":"A"}', 0.9, 200, 100, NULL, 0, 0, 0),
			(11, 11, '{"en":"L"}', 0.3, 200, 100, NULL, 0, 0, 0),
			(12, 12, '{"en":"B"}', 0.8, 200, 100, NULL, 0, 0, 0),
			(13, 13, '{"en":"C"}', 0.6, 200, 100, NULL, 0, 0, 0),
			(14, 14, '{"en":"D"}', 0.5, 200, 100, NULL, 0, 0, 0);

		-- Short route A<->L<->B (via low-sec)
		INSERT INTO mapStargates (_key, solarSystemID, typeID, destination) VALUES
			(2001, 10, 16, '{"solarSystemID": 11}'),
			(2002, 11, 16, '{"solarSystemID": 10}'),
			(2003, 11, 16, '{"solarSystemID": 12}'),
			(2004, 12, 16, '{"solarSystemID": 11}'),
			-- Long all-high-sec route A<->C<->D<->B
			(2005, 10, 16, '{"solarSystemID": 13}'),
			(2006, 13, 16, '{"solarSystemID": 10}'),
			(2007, 13, 16, '{"solarSystemID": 14}'),
			(2008, 14, 16, '{"solarSystemID": 13}'),
			(2009, 14, 16, '{"solarSystemID": 12}'),
			(2010, 12, 16, '{"solarSystemID": 14}');
	`
	if _, err := db.Exec(testData); err != nil {
		t.Fatalf("setupTestDataWithLowSecBypass: data: %v", err)
	}
}

// TestShortestPath_AvoidLowSec_BothEndpoints verifies that avoidLowSec=true excludes
// edges whose from_system_id is low-sec, not only to_system_id (bug fix F15).
// Topology: A(hs)→L(ls)→B(hs) is 2 jumps but routes through low-sec;
// A(hs)→C(hs)→D(hs)→B(hs) is 3 jumps, all high-sec.
func TestShortestPath_AvoidLowSec_BothEndpoints(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	resetGraphCache()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	setupTestDataWithLowSecBypass(t, db)
	if err := initializeNavigationViewsIntegration(db); err != nil {
		t.Fatalf("Failed to initialize views: %v", err)
	}

	const sysA, sysB, sysL = int64(10), int64(12), int64(11)

	// Without avoidLowSec the short path via L should be preferred (2 jumps).
	resetGraphCache()
	pathAll, err := ShortestPath(db, sysA, sysB, false)
	if err != nil {
		t.Fatalf("ShortestPath(avoidLowSec=false): %v", err)
	}
	if pathAll.Jumps != 2 {
		t.Errorf("avoidLowSec=false: expected 2 jumps via low-sec, got %d (route: %v)", pathAll.Jumps, pathAll.Route)
	}
	// Confirm it actually goes through L
	foundL := false
	for _, sys := range pathAll.Route {
		if sys == sysL {
			foundL = true
		}
	}
	if !foundL {
		t.Errorf("avoidLowSec=false: expected route through low-sec system %d, got %v", sysL, pathAll.Route)
	}

	// With avoidLowSec the short path via L must be excluded; longer all-high-sec route (3 jumps).
	resetGraphCache()
	pathHS, err := ShortestPath(db, sysA, sysB, true)
	if err != nil {
		t.Fatalf("ShortestPath(avoidLowSec=true): %v", err)
	}
	if pathHS.Jumps != 3 {
		t.Errorf("avoidLowSec=true: expected 3 jumps via high-sec only, got %d (route: %v)", pathHS.Jumps, pathHS.Route)
	}

	// All systems in the returned route must be high-sec (>= 0.45).
	for _, sysID := range pathHS.Route {
		var sec float64
		if err := db.QueryRow(
			"SELECT securityStatus FROM mapSolarSystems WHERE _key = ?", sysID,
		).Scan(&sec); err != nil {
			t.Errorf("could not query securityStatus for system %d: %v", sysID, err)
			continue
		}
		if sec < 0.45 {
			t.Errorf("avoidLowSec=true: route contains low-sec system %d (sec=%.2f)", sysID, sec)
		}
	}
}

// TestGraphCache_LoadsOnce prüft, dass der statische Graph nur einmal pro avoidLowSec-Variante geladen wird.
func TestGraphCache_LoadsOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	resetGraphCache()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	setupTestData(t, db)
	if err := initializeNavigationViewsIntegration(db); err != nil {
		t.Fatalf("Failed to initialize views: %v", err)
	}

	loads0 := graphLoadCount()
	_, err = ShortestPath(db, 1, 2, false)
	if err != nil {
		t.Fatalf("ShortestPath: %v", err)
	}
	_, _ = ShortestPath(db, 1, 2, false)
	loads1 := graphLoadCount()
	if loads1-loads0 != 1 {
		t.Errorf("Graph wurde %d mal geladen, erwartet 1 (Cache greift nicht)", loads1-loads0)
	}
}
