// Package database - Market data repository
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

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
	Issued       time.Time `json:"issued"`
	Duration     int       `json:"duration"`
	FetchedAt    time.Time `json:"fetched_at"`
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
	db *pgxpool.Pool
}

// NewMarketRepository creates a new market repository
func NewMarketRepository(db *pgxpool.Pool) *MarketRepository {
	return &MarketRepository{db: db}
}

// UpsertMarketOrders inserts or updates market orders
func (r *MarketRepository) UpsertMarketOrders(ctx context.Context, orders []MarketOrder) error {
	if len(orders) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO market_orders (
			order_id, type_id, region_id, location_id, is_buy_order,
			price, volume_total, volume_remain, min_volume,
			issued, duration, fetched_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (order_id, fetched_at) DO UPDATE SET
			price = EXCLUDED.price,
			volume_remain = EXCLUDED.volume_remain
	`

	for _, order := range orders {
		_, err := tx.Exec(ctx, query,
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
		if err != nil {
			return fmt.Errorf("failed to upsert order %d: %w", order.OrderID, err)
		}
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
			issued, duration, fetched_at
		FROM market_orders
		WHERE region_id = $1 AND type_id = $2
		ORDER BY price DESC, fetched_at DESC
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
			issued, duration, fetched_at
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
		WHERE fetched_at < $1
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.Exec(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to clean old orders: %w", err)
	}

	return result.RowsAffected(), nil
}
