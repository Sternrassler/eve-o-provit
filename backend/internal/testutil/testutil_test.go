// Package testutil_test verifies mock functionality
package testutil_test

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMockSDEQuerier_DefaultBehavior(t *testing.T) {
	mock := testutil.NewMockSDEWithDefaults()

	ctx := context.Background()

	// Test GetTypeInfo
	typeInfo, err := mock.GetTypeInfo(ctx, 34)
	assert.NoError(t, err)
	assert.Equal(t, 34, typeInfo.TypeID)
	assert.Equal(t, "Test Type 34", typeInfo.Name)

	// Test GetSystemIDForLocation (Jita 4-4)
	systemID, err := mock.GetSystemIDForLocation(ctx, 60003760)
	assert.NoError(t, err)
	assert.Equal(t, int64(30000142), systemID)

	// Test GetSystemName (Jita)
	systemName, err := mock.GetSystemName(ctx, 30000142)
	assert.NoError(t, err)
	assert.Equal(t, "Jita", systemName)

	// Test GetRegionIDForSystem
	regionID, err := mock.GetRegionIDForSystem(ctx, 30000142)
	assert.NoError(t, err)
	assert.Equal(t, 10000002, regionID)
}

func TestMockMarketQuerier_DefaultBehavior(t *testing.T) {
	mock := testutil.NewMockMarketWithDefaults()

	ctx := context.Background()

	// Test GetMarketOrders returns fixture data
	orders, err := mock.GetMarketOrders(ctx, 10000002, 34)
	assert.NoError(t, err)
	assert.Len(t, orders, 10)
	assert.Equal(t, 34, orders[0].TypeID)
	assert.Equal(t, 10000002, orders[0].RegionID)
}

func TestMockHealthChecker_DefaultBehavior(t *testing.T) {
	mock := testutil.NewMockHealthChecker()

	ctx := context.Background()

	// Test Health returns nil (healthy)
	err := mock.Health(ctx)
	assert.NoError(t, err)
}

func TestMockHealthChecker_ErrorBehavior(t *testing.T) {
	expectedErr := assert.AnError
	mock := testutil.NewMockHealthCheckerError(expectedErr)

	ctx := context.Background()

	// Test Health returns configured error
	err := mock.Health(ctx)
	assert.ErrorIs(t, err, expectedErr)
}

func TestFixtureTypeInfo(t *testing.T) {
	typeInfo := testutil.FixtureTypeInfo(123)

	assert.Equal(t, 123, typeInfo.TypeID)
	assert.Equal(t, "Test Type 123", typeInfo.Name)
	assert.Equal(t, 1.0, typeInfo.Volume)
	assert.NotNil(t, typeInfo.MarketGroup)
	assert.NotNil(t, typeInfo.CategoryID)
}

func TestFixtureMarketOrder(t *testing.T) {
	order := testutil.FixtureMarketOrder(1001, 34, 10000002, false)

	assert.Equal(t, int64(1001), order.OrderID)
	assert.Equal(t, 34, order.TypeID)
	assert.Equal(t, 10000002, order.RegionID)
	assert.False(t, order.IsBuyOrder)
	assert.Equal(t, 100.50, order.Price)
}

func TestFixtureMarketOrders(t *testing.T) {
	orders := testutil.FixtureMarketOrders(5, 34, 10000002)

	assert.Len(t, orders, 5)

	// Check alternating buy/sell orders
	assert.True(t, orders[0].IsBuyOrder)  // i=0, even
	assert.False(t, orders[1].IsBuyOrder) // i=1, odd
	assert.True(t, orders[2].IsBuyOrder)  // i=2, even
}
