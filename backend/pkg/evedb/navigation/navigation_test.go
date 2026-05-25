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
			wantAlignTime:   13.3, // Approximate
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
