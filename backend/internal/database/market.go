// Package database - Market data repository
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBPool is an interface for database connections (supports both pgxpool.Pool and pgxmock)
type DBPool interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Close()
}

// MarketOrder represents a market order from ESI
type MarketOrder struct {
	OrderID      int64     `json:"order_id"`
	TypeID       int       `json:"type_id"`
	RegionID     int       `json:"region_id"`
	LocationID   int64     `json:"location_id"`
	IsBuyOrder   bool      `json:"is_buy_order"`
	Price        float64   `json:"price"`
	VolumeTotal  int       `json:"volume_total"`
	VolumeRemain int       `json:"volume_remain"`
	MinVolume    *int      `json:"min_volume,omitempty"`
	Issued       time.Time `json:"issued"` // Maps to issued_at in DB
	Duration     int       `json:"duration"`
	FetchedAt    time.Time `json:"fetched_at"` // Maps to cached_at in DB
}

// PriceHistory represents aggregated price history data
type PriceHistory struct {
	ID         int       `json:"id"`
	TypeID     int       `json:"type_id"`
	RegionID   int       `json:"region_id"`
	Date       time.Time `json:"date"`
	Highest    *float64  `json:"highest,omitempty"`
	Lowest     *float64  `json:"lowest,omitempty"`
	Average    *float64  `json:"average,omitempty"`
	Volume     *int64    `json:"volume,omitempty"`
	OrderCount *int      `json:"order_count,omitempty"`
}

// MarketRepository handles market data operations
type MarketRepository struct {
	db DBPool
}

// NewMarketRepository creates a new market repository
func NewMarketRepository(db DBPool) *MarketRepository {
	return &MarketRepository{db: db}
}

// UpsertMarketOrders inserts or updates market orders using batch processing for performance
func (r *MarketRepository) UpsertMarketOrders(ctx context.Context, orders []MarketOrder) error {
	if len(orders) == 0 {
		return nil
	}

	// Use pgx.Batch for high-performance batch inserts
	// This is significantly faster than individual Exec calls in a loop
	// Especially critical for large datasets (e.g., 177k orders for Domain region)
	const batchSize = 1000 // Process in chunks to avoid memory issues

	for i := 0; i < len(orders); i += batchSize {
		end := i + batchSize
		if end > len(orders) {
			end = len(orders)
		}

		batch := orders[i:end]
		if err := r.upsertBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to upsert batch %d-%d: %w", i, end, err)
		}
	}

	return nil
}

// upsertBatch performs a single batch upsert operation
func (r *MarketRepository) upsertBatch(ctx context.Context, orders []MarketOrder) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Prepare batch
	batch := &pgx.Batch{}
	query := `
		INSERT INTO market_orders (
			order_id, type_id, region_id, location_id, is_buy_order,
			price, volume_total, volume_remain, min_volume,
			issued_at, duration, cached_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (order_id) DO UPDATE SET
			price = EXCLUDED.price,
			volume_remain = EXCLUDED.volume_remain,
			cached_at = EXCLUDED.cached_at
	`

	for _, order := range orders {
		batch.Queue(query,
			order.OrderID,
			order.TypeID,
			order.RegionID,
			order.LocationID,
			order.IsBuyOrder,
			order.Price,
			order.VolumeTotal,
			order.VolumeRemain,
			order.MinVolume,
			order.Issued,
			order.Duration,
			order.FetchedAt,
		)
	}

	// Send batch and close results immediately
	results := tx.SendBatch(ctx, batch)

	// Check all results
	for i := 0; i < batch.Len(); i++ {
		if _, err := results.Exec(); err != nil {
			results.Close()
			return fmt.Errorf("batch exec failed at index %d: %w", i, err)
		}
	}

	// Close results before commit
	if err := results.Close(); err != nil {
		return fmt.Errorf("failed to close batch results: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetMarketOrders retrieves market orders for a region and type
func (r *MarketRepository) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]MarketOrder, error) {
	query := `
		SELECT 
			order_id, type_id, region_id, location_id, is_buy_order,
			price, volume_total, volume_remain, min_volume,
			issued_at, duration, cached_at
		FROM market_orders
		WHERE region_id = $1 AND type_id = $2
		ORDER BY price DESC, cached_at DESC
	`

	rows, err := r.db.Query(ctx, query, regionID, typeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query market orders: %w", err)
	}
	defer rows.Close()

	var orders []MarketOrder
	for rows.Next() {
		var order MarketOrder
		err := rows.Scan(
			&order.OrderID,
			&order.TypeID,
			&order.RegionID,
			&order.LocationID,
			&order.IsBuyOrder,
			&order.Price,
			&order.VolumeTotal,
			&order.VolumeRemain,
			&order.MinVolume,
			&order.Issued,
			&order.Duration,
			&order.FetchedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return orders, nil
}

// GetAllMarketOrdersForRegion retrieves all market orders for a region (for route calculation)
func (r *MarketRepository) GetAllMarketOrdersForRegion(ctx context.Context, regionID int) ([]MarketOrder, error) {
	query := `
		SELECT 
			order_id, type_id, region_id, location_id, is_buy_order,
			price, volume_total, volume_remain, min_volume,
			issued_at, duration, cached_at
		FROM market_orders
		WHERE region_id = $1
		ORDER BY type_id, is_buy_order, price
	`

	rows, err := r.db.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query market orders: %w", err)
	}
	defer rows.Close()

	var orders []MarketOrder
	for rows.Next() {
		var order MarketOrder
		err := rows.Scan(
			&order.OrderID,
			&order.TypeID,
			&order.RegionID,
			&order.LocationID,
			&order.IsBuyOrder,
			&order.Price,
			&order.VolumeTotal,
			&order.VolumeRemain,
			&order.MinVolume,
			&order.Issued,
			&order.Duration,
			&order.FetchedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan market order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return orders, nil
}

// CleanOldMarketOrders removes market orders older than the specified duration
func (r *MarketRepository) CleanOldMarketOrders(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM market_orders
		WHERE cached_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to clean old orders: %w", err)
	}

	return result.RowsAffected(), nil
}

// UpsertPriceHistory inserts or updates price history records
func (r *MarketRepository) UpsertPriceHistory(ctx context.Context, history []PriceHistory) error {
	if len(history) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO price_history (
			type_id, region_id, date, highest, lowest, average, volume, order_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (type_id, region_id, date) DO UPDATE SET
			highest = EXCLUDED.highest,
			lowest = EXCLUDED.lowest,
			average = EXCLUDED.average,
			volume = EXCLUDED.volume,
			order_count = EXCLUDED.order_count
	`

	for _, h := range history {
		_, err := tx.Exec(ctx, query,
			h.TypeID,
			h.RegionID,
			h.Date,
			h.Highest,
			h.Lowest,
			h.Average,
			h.Volume,
			h.OrderCount,
		)
		if err != nil {
			return fmt.Errorf("failed to upsert price history: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetVolumeHistory retrieves volume history for a type in a region
// Returns data for the last 'days' days, ordered by date descending
func (r *MarketRepository) GetVolumeHistory(ctx context.Context, typeID, regionID, days int) ([]PriceHistory, error) {
	query := `
		SELECT 
			id, type_id, region_id, date, highest, lowest, average, volume, order_count
		FROM price_history
		WHERE type_id = $1 AND region_id = $2
			AND date >= CURRENT_DATE - $3::INTEGER
		ORDER BY date DESC
	`

	rows, err := r.db.Query(ctx, query, typeID, regionID, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query volume history: %w", err)
	}
	defer rows.Close()

	var history []PriceHistory
	for rows.Next() {
		var h PriceHistory
		err := rows.Scan(
			&h.ID,
			&h.TypeID,
			&h.RegionID,
			&h.Date,
			&h.Highest,
			&h.Lowest,
			&h.Average,
			&h.Volume,
			&h.OrderCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan price history: %w", err)
		}
		history = append(history, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return history, nil
}
