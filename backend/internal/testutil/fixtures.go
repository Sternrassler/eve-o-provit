// Package testutil provides test utilities and fixtures
package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
)

// FixtureTypeInfo creates a sample TypeInfo for testing
func FixtureTypeInfo(typeID int) *database.TypeInfo {
	marketGroup := 123
	categoryID := 456
	categoryName := "Test Category"

	return &database.TypeInfo{
		TypeID:       typeID,
		Name:         fmt.Sprintf("Test Type %d", typeID),
		Volume:       1.0,
		Capacity:     10.0,
		BasePrice:    1000.0,
		MarketGroup:  &marketGroup,
		CategoryID:   &categoryID,
		CategoryName: &categoryName,
	}
}

// FixtureMarketOrder creates a sample MarketOrder for testing
func FixtureMarketOrder(orderID int64, typeID int, regionID int, isBuyOrder bool) database.MarketOrder {
	minVol := 1
	return database.MarketOrder{
		OrderID:      orderID,
		TypeID:       typeID,
		RegionID:     regionID,
		LocationID:   60003760, // Jita 4-4
		IsBuyOrder:   isBuyOrder,
		Price:        100.50,
		VolumeTotal:  1000,
		VolumeRemain: 500,
		MinVolume:    &minVol,
		Issued:       time.Now().Add(-24 * time.Hour),
		Duration:     90,
		FetchedAt:    time.Now(),
	}
}

// FixtureMarketOrders creates a slice of sample market orders for testing
func FixtureMarketOrders(count int, typeID int, regionID int) []database.MarketOrder {
	orders := make([]database.MarketOrder, count)
	for i := 0; i < count; i++ {
		isBuyOrder := i%2 == 0
		orders[i] = FixtureMarketOrder(int64(1000+i), typeID, regionID, isBuyOrder)
	}
	return orders
}

// NewMockSDEWithDefaults creates a MockSDEQuerier with sensible defaults
func NewMockSDEWithDefaults() *MockSDEQuerier {
	return &MockSDEQuerier{
		GetTypeInfoFunc: func(ctx context.Context, typeID int) (*database.TypeInfo, error) {
			return FixtureTypeInfo(typeID), nil
		},
		GetSystemIDForLocationFunc: func(ctx context.Context, locationID int64) (int64, error) {
			// Jita 4-4 (station) -> Jita (system)
			if locationID == 60003760 {
				return 30000142, nil
			}
			return 30000142, nil // Default to Jita
		},
		GetSystemNameFunc: func(ctx context.Context, systemID int64) (string, error) {
			if systemID == 30000142 {
				return "Jita", nil
			}
			return fmt.Sprintf("System-%d", systemID), nil
		},
		GetStationNameFunc: func(ctx context.Context, stationID int64) (string, error) {
			if stationID == 60003760 {
				return "Jita IV - Moon 4 - Caldari Navy Assembly Plant", nil
			}
			return fmt.Sprintf("Station-%d", stationID), nil
		},
		GetRegionIDForSystemFunc: func(ctx context.Context, systemID int64) (int, error) {
			// Jita -> The Forge
			if systemID == 30000142 {
				return 10000002, nil
			}
			return 10000002, nil // Default to The Forge
		},
	}
}

// NewMockMarketWithDefaults creates a MockMarketQuerier with sensible defaults
func NewMockMarketWithDefaults() *MockMarketQuerier {
	// In-memory orders storage for stateful mock
	ordersStore := make(map[string][]database.MarketOrder)

	return &MockMarketQuerier{
		UpsertMarketOrdersFunc: func(ctx context.Context, orders []database.MarketOrder) error {
			for _, order := range orders {
				key := fmt.Sprintf("%d-%d", order.RegionID, order.TypeID)
				ordersStore[key] = append(ordersStore[key], order)
			}
			return nil
		},
		GetMarketOrdersFunc: func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
			key := fmt.Sprintf("%d-%d", regionID, typeID)
			if orders, ok := ordersStore[key]; ok {
				return orders, nil
			}
			// Return fixture orders if not in store
			return FixtureMarketOrders(10, typeID, regionID), nil
		},
	}
}

// NewMockHealthChecker creates a MockHealthChecker that always returns healthy
func NewMockHealthChecker() *MockHealthChecker {
	return &MockHealthChecker{
		HealthFunc: func(ctx context.Context) error {
			return nil
		},
	}
}

// NewMockHealthCheckerError creates a MockHealthChecker that always returns an error
func NewMockHealthCheckerError(err error) *MockHealthChecker {
	return &MockHealthChecker{
		HealthFunc: func(ctx context.Context) error {
			return err
		},
	}
}
