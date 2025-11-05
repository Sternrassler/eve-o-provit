// Package testutil provides test utilities and mocks
package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
)

// MockHealthChecker is a mock implementation of database.HealthChecker
type MockHealthChecker struct {
	HealthFunc func(ctx context.Context) error
}

// Health calls the mock function or returns nil
func (m *MockHealthChecker) Health(ctx context.Context) error {
	if m.HealthFunc != nil {
		return m.HealthFunc(ctx)
	}
	return nil
}

// MockSDEQuerier is a mock implementation of database.SDEQuerier
type MockSDEQuerier struct {
	GetTypeInfoFunc            func(ctx context.Context, typeID int) (*database.TypeInfo, error)
	SearchTypesFunc            func(ctx context.Context, searchTerm string, limit int) ([]database.TypeInfo, error)
	GetSystemIDForLocationFunc func(ctx context.Context, locationID int64) (int64, error)
	GetSystemNameFunc          func(ctx context.Context, systemID int64) (string, error)
	GetStationNameFunc         func(ctx context.Context, stationID int64) (string, error)
	GetRegionIDForSystemFunc   func(ctx context.Context, systemID int64) (int, error)
	SearchItemsFunc            func(ctx context.Context, searchTerm string, limit int) ([]struct {
		TypeID    int
		Name      string
		GroupName string
	}, error)
}

// GetTypeInfo calls the mock function or returns a default TypeInfo
func (m *MockSDEQuerier) GetTypeInfo(ctx context.Context, typeID int) (*database.TypeInfo, error) {
	if m.GetTypeInfoFunc != nil {
		return m.GetTypeInfoFunc(ctx, typeID)
	}
	return &database.TypeInfo{
		TypeID:   typeID,
		Name:     fmt.Sprintf("Type-%d", typeID),
		Volume:   1.0,
		Capacity: 0.0,
	}, nil
}

// SearchTypes calls the mock function or returns empty slice
func (m *MockSDEQuerier) SearchTypes(ctx context.Context, searchTerm string, limit int) ([]database.TypeInfo, error) {
	if m.SearchTypesFunc != nil {
		return m.SearchTypesFunc(ctx, searchTerm, limit)
	}
	return []database.TypeInfo{}, nil
}

// GetSystemIDForLocation calls the mock function or returns a default value
func (m *MockSDEQuerier) GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error) {
	if m.GetSystemIDForLocationFunc != nil {
		return m.GetSystemIDForLocationFunc(ctx, locationID)
	}
	return 30000142, nil // Jita by default
}

// GetSystemName calls the mock function or returns a default name
func (m *MockSDEQuerier) GetSystemName(ctx context.Context, systemID int64) (string, error) {
	if m.GetSystemNameFunc != nil {
		return m.GetSystemNameFunc(ctx, systemID)
	}
	return fmt.Sprintf("System-%d", systemID), nil
}

// GetStationName calls the mock function or returns a default name
func (m *MockSDEQuerier) GetStationName(ctx context.Context, stationID int64) (string, error) {
	if m.GetStationNameFunc != nil {
		return m.GetStationNameFunc(ctx, stationID)
	}
	return fmt.Sprintf("Station-%d", stationID), nil
}

// GetRegionIDForSystem calls the mock function or returns a default region ID
func (m *MockSDEQuerier) GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error) {
	if m.GetRegionIDForSystemFunc != nil {
		return m.GetRegionIDForSystemFunc(ctx, systemID)
	}
	return 10000002, nil // The Forge by default
}

// SearchItems calls the mock function or returns empty slice
func (m *MockSDEQuerier) SearchItems(ctx context.Context, searchTerm string, limit int) ([]struct {
	TypeID    int
	Name      string
	GroupName string
}, error) {
	if m.SearchItemsFunc != nil {
		return m.SearchItemsFunc(ctx, searchTerm, limit)
	}
	return []struct {
		TypeID    int
		Name      string
		GroupName string
	}{}, nil
}

// MockMarketQuerier is a mock implementation of database.MarketQuerier
type MockMarketQuerier struct {
	UpsertMarketOrdersFunc          func(ctx context.Context, orders []database.MarketOrder) error
	GetMarketOrdersFunc             func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error)
	GetAllMarketOrdersForRegionFunc func(ctx context.Context, regionID int) ([]database.MarketOrder, error)
	CleanOldMarketOrdersFunc        func(ctx context.Context, olderThan time.Duration) (int64, error)
}

// UpsertMarketOrders calls the mock function or returns nil
func (m *MockMarketQuerier) UpsertMarketOrders(ctx context.Context, orders []database.MarketOrder) error {
	if m.UpsertMarketOrdersFunc != nil {
		return m.UpsertMarketOrdersFunc(ctx, orders)
	}
	return nil
}

// GetMarketOrders calls the mock function or returns empty slice
func (m *MockMarketQuerier) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
	if m.GetMarketOrdersFunc != nil {
		return m.GetMarketOrdersFunc(ctx, regionID, typeID)
	}
	return []database.MarketOrder{}, nil
}

// GetAllMarketOrdersForRegion calls the mock function or returns empty slice
func (m *MockMarketQuerier) GetAllMarketOrdersForRegion(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	if m.GetAllMarketOrdersForRegionFunc != nil {
		return m.GetAllMarketOrdersForRegionFunc(ctx, regionID)
	}
	return []database.MarketOrder{}, nil
}

// CleanOldMarketOrders calls the mock function or returns 0
func (m *MockMarketQuerier) CleanOldMarketOrders(ctx context.Context, olderThan time.Duration) (int64, error) {
	if m.CleanOldMarketOrdersFunc != nil {
		return m.CleanOldMarketOrdersFunc(ctx, olderThan)
	}
	return 0, nil
}

// Compile-time interface compliance checks
var (
	_ database.HealthChecker = (*MockHealthChecker)(nil)
	_ database.SDEQuerier    = (*MockSDEQuerier)(nil)
	_ database.MarketQuerier = (*MockMarketQuerier)(nil)
)
