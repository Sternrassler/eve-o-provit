//go:build integration || !unit

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarketRepository_Integration_UpsertAndGet tests real database operations
func TestMarketRepository_Integration_UpsertAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup PostgreSQL container
	tc := SetupPostgresContainer(t)
	tc.CreateTestSchema(t)

	// Create repository
	repo := NewMarketRepository(tc.Pool)
	ctx := context.Background()

	// Create test orders
	now := time.Now()
	minVol := 5
	orders := []MarketOrder{
		{
			OrderID:      123456789,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.50,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			MinVolume:    &minVol,
			Issued:       now.Add(-24 * time.Hour),
			Duration:     90,
			FetchedAt:    now,
		},
		{
			OrderID:      987654321,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   true,
			Price:        5.25,
			VolumeTotal:  5000,
			VolumeRemain: 5000,
			MinVolume:    nil,
			Issued:       now.Add(-48 * time.Hour),
			Duration:     90,
			FetchedAt:    now,
		},
	}

	// Test: Upsert orders
	err := repo.UpsertMarketOrders(ctx, orders)
	require.NoError(t, err)

	// Test: Get orders by region and type
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	require.NoError(t, err)
	assert.Len(t, retrieved, 2)

	// Verify first order (sorted by price DESC)
	assert.Equal(t, int64(123456789), retrieved[0].OrderID)
	assert.Equal(t, 34, retrieved[0].TypeID)
	assert.Equal(t, 5.50, retrieved[0].Price)
	assert.False(t, retrieved[0].IsBuyOrder)
	assert.NotNil(t, retrieved[0].MinVolume)
	assert.Equal(t, 5, *retrieved[0].MinVolume)

	// Verify second order
	assert.Equal(t, int64(987654321), retrieved[1].OrderID)
	assert.Equal(t, 5.25, retrieved[1].Price)
	assert.True(t, retrieved[1].IsBuyOrder)
	assert.Nil(t, retrieved[1].MinVolume)
}

// TestMarketRepository_Integration_EmptyResult tests querying non-existent data
func TestMarketRepository_Integration_EmptyResult(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupPostgresContainer(t)
	tc.CreateTestSchema(t)

	repo := NewMarketRepository(tc.Pool)
	ctx := context.Background()

	// Query for non-existent region/type
	orders, err := repo.GetMarketOrders(ctx, 99999, 99999)
	require.NoError(t, err)
	assert.Empty(t, orders)
}

// TestMarketRepository_Integration_UpdateExisting tests upsert conflict handling
func TestMarketRepository_Integration_UpdateExisting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupPostgresContainer(t)
	tc.CreateTestSchema(t)

	repo := NewMarketRepository(tc.Pool)
	ctx := context.Background()

	now := time.Now()

	// Insert initial order
	initialOrders := []MarketOrder{
		{
			OrderID:      123456789,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.50,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			Issued:       now.Add(-24 * time.Hour),
			Duration:     90,
			FetchedAt:    now,
		},
	}

	err := repo.UpsertMarketOrders(ctx, initialOrders)
	require.NoError(t, err)

	// Update with changed price and volume
	updatedOrders := []MarketOrder{
		{
			OrderID:      123456789,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.75, // Changed price
			VolumeTotal:  1000,
			VolumeRemain: 300, // Changed remaining volume
			Issued:       now.Add(-24 * time.Hour),
			Duration:     90,
			FetchedAt:    now, // Same fetch time = update
		},
	}

	err = repo.UpsertMarketOrders(ctx, updatedOrders)
	require.NoError(t, err)

	// Retrieve and verify update
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	require.NoError(t, err)
	assert.Len(t, retrieved, 1)
	assert.Equal(t, 5.75, retrieved[0].Price)
	assert.Equal(t, 300, retrieved[0].VolumeRemain)
}

// TestMarketRepository_Integration_GetAllForRegion tests bulk region query
func TestMarketRepository_Integration_GetAllForRegion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupPostgresContainer(t)
	tc.CreateTestSchema(t)
	tc.SeedTestData(t) // Use seeded data

	repo := NewMarketRepository(tc.Pool)
	ctx := context.Background()

	// Get all orders for region
	orders, err := repo.GetAllMarketOrdersForRegion(ctx, 10000002)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(orders), 3, "Should have at least 3 seeded orders")

	// Verify all orders are from correct region
	for _, order := range orders {
		assert.Equal(t, 10000002, order.RegionID)
	}
}

// TestMarketRepository_Integration_LargeDataset tests performance with many orders
func TestMarketRepository_Integration_LargeDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupPostgresContainer(t)
	tc.CreateTestSchema(t)

	repo := NewMarketRepository(tc.Pool)
	ctx := context.Background()

	// Create 1000 orders
	now := time.Now()
	orders := make([]MarketOrder, 1000)
	for i := 0; i < 1000; i++ {
		orders[i] = MarketOrder{
			OrderID:      int64(1000000 + i),
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   i%2 == 0,
			Price:        5.0 + float64(i)*0.01,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			Issued:       now.Add(-time.Duration(i) * time.Hour),
			Duration:     90,
			FetchedAt:    now,
		}
	}

	// Measure upsert performance
	start := time.Now()
	err := repo.UpsertMarketOrders(ctx, orders)
	duration := time.Since(start)

	require.NoError(t, err)
	t.Logf("Upserted 1000 orders in %v", duration)

	// Verify count
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	require.NoError(t, err)
	assert.Len(t, retrieved, 1000)

	// Performance assertion (should be under 5 seconds)
	assert.Less(t, duration.Seconds(), 5.0, "Upsert should complete in under 5 seconds")
}
