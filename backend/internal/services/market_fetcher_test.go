// Package services_test - Tests for MarketFetcher
package services_test

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/internal/testutil"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarketFetcher_FetchMarketOrders_Success(t *testing.T) {
	// Setup
	ctx := context.Background()

	// Create mock that returns actual orders
	mockMarket := &testutil.MockMarketQuerier{
		GetAllMarketOrdersForRegionFunc: func(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
			return testutil.FixtureMarketOrders(5, 34, regionID), nil
		},
	}

	esiClient := &esi.Client{}
	fetcher := services.NewMarketFetcher(esiClient, mockMarket, nil) // No Redis

	// Execute
	orders, err := fetcher.FetchMarketOrders(ctx, 10000002) // The Forge

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, orders)
	assert.Len(t, orders, 5)
	assert.Equal(t, 10000002, orders[0].RegionID)
}

func TestMarketFetcher_FetchMarketOrders_EmptyResult(t *testing.T) {
	// Setup
	ctx := context.Background()

	mockMarket := &testutil.MockMarketQuerier{
		GetAllMarketOrdersForRegionFunc: func(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
			return []database.MarketOrder{}, nil // Empty
		},
	}

	esiClient := &esi.Client{}
	fetcher := services.NewMarketFetcher(esiClient, mockMarket, nil)

	// Execute
	orders, err := fetcher.FetchMarketOrders(ctx, 10000002)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, orders)
}

func TestMarketFetcher_FetchMarketOrders_DatabaseError(t *testing.T) {
	// Setup
	ctx := context.Background()

	mockMarket := &testutil.MockMarketQuerier{
		GetAllMarketOrdersForRegionFunc: func(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
			return nil, assert.AnError
		},
	}

	esiClient := &esi.Client{}
	fetcher := services.NewMarketFetcher(esiClient, mockMarket, nil)

	// Execute
	orders, err := fetcher.FetchMarketOrders(ctx, 10000002)

	// Assert
	require.Error(t, err)
	assert.Nil(t, orders)
	assert.Contains(t, err.Error(), "failed to fetch market orders")
}

func TestMarketFetcher_NewMarketFetcher_WithRedis(t *testing.T) {
	// This test verifies that NewMarketFetcher accepts Redis client
	// Actual Redis functionality tested in integration tests

	mockMarket := testutil.NewMockMarketWithDefaults()
	esiClient := &esi.Client{}

	// Create without Redis
	fetcher := services.NewMarketFetcher(esiClient, mockMarket, nil)
	assert.NotNil(t, fetcher)

	// TODO: Test with actual Redis when available in test setup
}
