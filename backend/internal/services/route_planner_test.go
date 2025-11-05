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
