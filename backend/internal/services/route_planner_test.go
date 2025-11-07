// Package services_test - Tests for RoutePlanner
package services_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRoutePlanner_GetSystemIDFromLocation(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSDE := testutil.NewMockSDEWithDefaults()
	planner := services.NewRoutePlanner(&sql.DB{}, mockSDE, nil)

	// Execute - Jita 4-4 station
	systemID := planner.GetSystemIDFromLocation(ctx, 60003760)

	// Assert - Should resolve to Jita system
	assert.Equal(t, int64(30000142), systemID)
}

func TestRoutePlanner_GetSystemIDFromLocation_AlreadySystemID(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Mock that returns error (location not found)
	mockSDE := &testutil.MockSDEQuerier{
		GetSystemIDForLocationFunc: func(ctx context.Context, locationID int64) (int64, error) {
			return 0, assert.AnError
		},
	}

	planner := services.NewRoutePlanner(&sql.DB{}, mockSDE, nil)

	// Execute - Pass a system ID
	systemID := planner.GetSystemIDFromLocation(ctx, 30000142)

	// Assert - Should fallback to returning input
	assert.Equal(t, int64(30000142), systemID)
}

func TestRoutePlanner_CalculateTravelTime(t *testing.T) {
	planner := services.NewRoutePlanner(&sql.DB{}, nil, nil)

	tests := []struct {
		name          string
		jumps         int
		numberOfTours int
		expected      time.Duration
	}{
		{
			name:          "Single tour, 5 jumps",
			jumps:         5,
			numberOfTours: 1,
			expected:      time.Duration(5*30*2) * time.Second, // Round trip
		},
		{
			name:          "Multiple tours, 10 jumps",
			jumps:         10,
			numberOfTours: 3,
			expected:      time.Duration((3-1)*10*30*2+10*30) * time.Second,
		},
		{
			name:          "Zero jumps",
			jumps:         0,
			numberOfTours: 1,
			expected:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration := planner.CalculateTravelTime(tt.jumps, tt.numberOfTours)
			assert.Equal(t, tt.expected, duration)
		})
	}
}

func TestRoutePlanner_CalculateRoute_InvalidInput(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSDE := testutil.NewMockSDEWithDefaults()
	planner := services.NewRoutePlanner(&sql.DB{}, mockSDE, nil) // SQL.DB placeholder

	item := models.ItemPair{
		TypeID:            34,
		ItemName:          "Tritanium",
		BuySystemID:       30000142,
		SellSystemID:      30000144,
		BuyStationID:      60003760,
		SellStationID:     60003760,
		BuyPrice:          100.0,
		SellPrice:         120.0,
		AvailableQuantity: 1000,
	}

	// Execute with invalid quantity
	_, err := planner.CalculateRoute(ctx, item, 1000.0, 1, 0)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid quantity per tour")
}

func TestRoutePlanner_CalculateRoute_NavigationError(t *testing.T) {
	t.Skip("Requires working SDE database - tested in integration tests")
	// navigation.ShortestPath requires real DB
}

func TestRoutePlanner_NewRoutePlanner_WithRedis(t *testing.T) {
	// Verify constructor accepts Redis client
	mockSDE := testutil.NewMockSDEWithDefaults()

	// Create without Redis
	planner := services.NewRoutePlanner(&sql.DB{}, mockSDE, nil)
	assert.NotNil(t, planner)

	// TODO: Test with actual Redis when available
}

func TestRoutePlanner_CalculateJumpTime(t *testing.T) {
	planner := services.NewRoutePlanner(&sql.DB{}, nil, nil)

	tests := []struct {
		name            string
		jumps           int
		baseWarpSpeed   float64
		baseAlignTime   float64
		navigationLevel int
		evasiveLevel    int
		expectedMin     float64 // Minimum expected time
		expectedMax     float64 // Maximum expected time
	}{
		{
			name:            "No skills, 5 jumps",
			jumps:           5,
			baseWarpSpeed:   3.0,
			baseAlignTime:   8.0,
			navigationLevel: 0,
			evasiveLevel:    0,
			expectedMin:     100.0, // 5 jumps × (8s align + 3s warp + 10s docking) = 105s
			expectedMax:     110.0,
		},
		{
			name:            "Navigation V, 5 jumps",
			jumps:           5,
			baseWarpSpeed:   3.0,
			baseAlignTime:   8.0,
			navigationLevel: 5,
			evasiveLevel:    0,
			expectedMin:     95.0, // Faster warp speed
			expectedMax:     105.0,
		},
		{
			name:            "Evasive V, 5 jumps",
			jumps:           5,
			baseWarpSpeed:   3.0,
			baseAlignTime:   8.0,
			navigationLevel: 0,
			evasiveLevel:    5,
			expectedMin:     85.0, // Faster align time
			expectedMax:     95.0,
		},
		{
			name:            "Both skills maxed, 5 jumps",
			jumps:           5,
			baseWarpSpeed:   3.0,
			baseAlignTime:   8.0,
			navigationLevel: 5,
			evasiveLevel:    5,
			expectedMin:     85.0, // Both improvements
			expectedMax:     95.0,
		},
		{
			name:            "Zero jumps",
			jumps:           0,
			baseWarpSpeed:   3.0,
			baseAlignTime:   8.0,
			navigationLevel: 5,
			evasiveLevel:    5,
			expectedMin:     0.0,
			expectedMax:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := planner.CalculateJumpTime(tt.jumps, tt.baseWarpSpeed, tt.baseAlignTime, tt.navigationLevel, tt.evasiveLevel)

			assert.GreaterOrEqual(t, result, tt.expectedMin, "Travel time should be >= minimum")
			assert.LessOrEqual(t, result, tt.expectedMax, "Travel time should be <= maximum")

			// Verify skills reduce travel time (except for zero jumps)
			if tt.jumps > 0 && (tt.navigationLevel > 0 || tt.evasiveLevel > 0) {
				baseTime := planner.CalculateJumpTime(tt.jumps, tt.baseWarpSpeed, tt.baseAlignTime, 0, 0)
				assert.Less(t, result, baseTime, "Skilled travel time should be less than base time")
			}
		})
	}
}

func TestRoutePlanner_NavigationSkillsImprovement(t *testing.T) {
	planner := services.NewRoutePlanner(&sql.DB{}, nil, nil)

	// Test different skill combinations
	tests := []struct {
		name                    string
		navigationLevel         int
		evasiveLevel            int
		minImprovementPercent   float64
		maxImprovementPercent   float64
		jumps                   int
		baseWarpSpeed           float64
		baseAlignTime           float64
	}{
		{
			name:                  "Navigation V only",
			navigationLevel:       5,
			evasiveLevel:          0,
			minImprovementPercent: 2.0, // Warp speed improvement is ~3% of total time
			maxImprovementPercent: 5.0,
			jumps:                 10,
			baseWarpSpeed:         3.0,
			baseAlignTime:         8.0,
		},
		{
			name:                  "Evasive V only",
			navigationLevel:       0,
			evasiveLevel:          5,
			minImprovementPercent: 9.0, // Align time improvement is ~10% of total time
			maxImprovementPercent: 12.0,
			jumps:                 10,
			baseWarpSpeed:         3.0,
			baseAlignTime:         8.0,
		},
		{
			name:                  "Both skills maxed",
			navigationLevel:       5,
			evasiveLevel:          5,
			minImprovementPercent: 11.0, // Combined improvement ~12%
			maxImprovementPercent: 15.0,
			jumps:                 10,
			baseWarpSpeed:         3.0,
			baseAlignTime:         8.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseTime := planner.CalculateJumpTime(tt.jumps, tt.baseWarpSpeed, tt.baseAlignTime, 0, 0)
			skilledTime := planner.CalculateJumpTime(tt.jumps, tt.baseWarpSpeed, tt.baseAlignTime, tt.navigationLevel, tt.evasiveLevel)

			improvement := ((baseTime - skilledTime) / baseTime) * 100

			assert.GreaterOrEqual(t, improvement, tt.minImprovementPercent,
				"Skills should provide at least %.1f%% improvement", tt.minImprovementPercent)
			assert.LessOrEqual(t, improvement, tt.maxImprovementPercent,
				"Improvement should be at most %.1f%%", tt.maxImprovementPercent)

			t.Logf("Base time: %.1fs, Skilled time: %.1fs, Improvement: %.1f%%",
				baseTime, skilledTime, improvement)
		})
	}
}
