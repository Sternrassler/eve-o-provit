// Package database provides interface definitions for testability
package database

import (
	"context"
	"time"
)

// HealthChecker defines the interface for database health checking
type HealthChecker interface {
	Health(ctx context.Context) error
}

// SDEQuerier defines the interface for SDE (Static Data Export) queries
type SDEQuerier interface {
	GetTypeInfo(ctx context.Context, typeID int) (*TypeInfo, error)
	SearchTypes(ctx context.Context, searchTerm string, limit int) ([]TypeInfo, error)
	GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error)
	GetSystemName(ctx context.Context, systemID int64) (string, error)
	GetStationName(ctx context.Context, stationID int64) (string, error)
	GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error)
	SearchItems(ctx context.Context, searchTerm string, limit int) ([]struct {
		TypeID    int
		Name      string
		GroupName string
	}, error)
}

// RegionQuerier defines the interface for region queries
type RegionQuerier interface {
	GetAllRegions(ctx context.Context) ([]RegionData, error)
}

// RegionData represents a region from SDE
type RegionData struct {
	ID   int64
	Name string
}

// MarketQuerier defines the interface for market data queries
type MarketQuerier interface {
	UpsertMarketOrders(ctx context.Context, orders []MarketOrder) error
	GetMarketOrders(ctx context.Context, regionID, typeID int) ([]MarketOrder, error)
	GetAllMarketOrdersForRegion(ctx context.Context, regionID int) ([]MarketOrder, error)
	CleanOldMarketOrders(ctx context.Context, olderThan time.Duration) (int64, error)
}

// PostgresQuerier defines the interface for raw Postgres queries
// Used for operations that haven't been migrated to repository pattern yet
type PostgresQuerier interface {
	// QueryRow executes a query that returns at most one row
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
}

// Row defines the interface for scanning query results
type Row interface {
	// Scan copies the columns from the matched row into the values pointed at by dest
	Scan(dest ...interface{}) error
}

// Compile-time interface compliance checks
var (
	_ HealthChecker = (*DB)(nil)
	_ SDEQuerier    = (*SDERepository)(nil)
	_ MarketQuerier = (*MarketRepository)(nil)
)
