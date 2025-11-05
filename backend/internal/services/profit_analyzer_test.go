// Package services_test - Tests for ProfitAnalyzer
package services_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfitAnalyzer_CalculateProfitPerTour(t *testing.T) {
	analyzer := services.NewProfitAnalyzer(nil, nil)

	tests := []struct {
		name      string
		buyPrice  float64
		sellPrice float64
		quantity  int
		expected  float64
	}{
		{"Simple profit", 100.0, 150.0, 10, 500.0},
		{"Zero profit", 100.0, 100.0, 10, 0.0},
		{"Loss", 150.0, 100.0, 10, -500.0},
		{"Large quantity", 1000.0, 1500.0, 1000, 500000.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profit := analyzer.CalculateProfitPerTour(tt.buyPrice, tt.sellPrice, tt.quantity)
			assert.Equal(t, tt.expected, profit)
		})
	}
}

func TestProfitAnalyzer_CalculateQuantityPerTour(t *testing.T) {
	analyzer := services.NewProfitAnalyzer(nil, nil)

	tests := []struct {
		name          string
		itemVolume    float64
		cargoCapacity float64
		expected      int
		expectError   bool
	}{
		{"Simple fit", 1.0, 100.0, 100, false},
		{"Exact fit", 10.0, 100.0, 10, false},
		{"Doesn't fit", 200.0, 100.0, 0, true},
		{"Zero volume", 0.0, 100.0, 0, true},
		{"Negative volume", -1.0, 100.0, 0, true},
		{"Fractional", 0.5, 100.0, 200, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qty, err := analyzer.CalculateQuantityPerTour(tt.itemVolume, tt.cargoCapacity)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, qty)
			}
		})
	}
}

func TestProfitAnalyzer_CalculateNumberOfTours(t *testing.T) {
	analyzer := services.NewProfitAnalyzer(nil, nil)

	tests := []struct {
		name              string
		availableQuantity int
		quantityPerTour   int
		expected          int
	}{
		{"Single tour", 50, 100, 1},
		{"Multiple tours", 500, 100, 5},
		{"Exactly 10 tours", 1000, 100, 10},
		{"Capped at 10", 2000, 100, 10},
		{"Zero quantity per tour", 100, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tours := analyzer.CalculateNumberOfTours(tt.availableQuantity, tt.quantityPerTour)
			assert.Equal(t, tt.expected, tours)
		})
	}
}

func TestProfitAnalyzer_FindProfitableItems_WithMocks(t *testing.T) {
	t.Skip("Requires working SDE database - tested in integration tests")

	// This test would require a real or mocked cargo.GetItemVolume
	// For unit testing, use the calculation functions instead
}

func TestProfitAnalyzer_FindProfitableItems_NoSpread(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSDE := testutil.NewMockSDEWithDefaults()
	analyzer := services.NewProfitAnalyzer(&sql.DB{}, mockSDE)

	// Create orders with no spread (same price)
	orders := []database.MarketOrder{
		testutil.FixtureMarketOrder(1001, 34, 10000002, false), // Sell: 100.50
		testutil.FixtureMarketOrder(1002, 34, 10000002, true),  // Buy: 100.50
	}

	systemIDResolver := func(locationID int64) int64 {
		return 30000142
	}

	// Execute
	items, err := analyzer.FindProfitableItems(ctx, orders, 1000.0, systemIDResolver)

	// Assert - No profitable items (spread < 5%)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestProfitAnalyzer_FindProfitableItems_MissingBuyOrders(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSDE := testutil.NewMockSDEWithDefaults()
	analyzer := services.NewProfitAnalyzer(&sql.DB{}, mockSDE)

	// Create only sell orders (no buy orders)
	orders := []database.MarketOrder{
		testutil.FixtureMarketOrder(1001, 34, 10000002, false),
		testutil.FixtureMarketOrder(1002, 35, 10000002, false),
	}

	systemIDResolver := func(locationID int64) int64 {
		return 30000142
	}

	// Execute
	items, err := analyzer.FindProfitableItems(ctx, orders, 1000.0, systemIDResolver)

	// Assert - No profitable pairs without both buy and sell
	require.NoError(t, err)
	assert.Empty(t, items)
}
