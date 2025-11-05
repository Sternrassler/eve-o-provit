package esi

import (
	"context"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/alicebob/miniredis/v2"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetMarketOrders_WithData tests retrieving market orders from database
func TestGetMarketOrders_WithData(t *testing.T) {
	// Setup pgxmock
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Setup Redis
	s := miniredis.RunT(t)
	defer s.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	// Create repository with mock pool
	repo := database.NewMarketRepository(mock)

	// Create ESI client
	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}
	client, err := NewClient(redisClient, cfg, repo)
	require.NoError(t, err)
	defer client.Close()

	// Setup expected query
	minVol1 := 1
	minVol2 := 10
	rows := pgxmock.NewRows([]string{
		"order_id", "type_id", "region_id", "location_id", "is_buy_order",
		"price", "volume_total", "volume_remain", "min_volume",
		"issued", "duration", "fetched_at",
	}).
		AddRow(
			int64(123456789), 34, 10000002, int64(60003760), false,
			5.50, 1000, 500, &minVol1,
			time.Now().Add(-24*time.Hour), 90, time.Now(),
		).
		AddRow(
			int64(987654321), 34, 10000002, int64(60003760), true,
			5.25, 5000, 5000, &minVol2,
			time.Now().Add(-48*time.Hour), 90, time.Now(),
		)

	mock.ExpectQuery(`SELECT (.+) FROM market_orders WHERE region_id = \$1 AND type_id = \$2`).
		WithArgs(10000002, 34).
		WillReturnRows(rows)

	// Execute
	orders, err := client.GetMarketOrders(context.Background(), 10000002, 34)

	// Verify
	require.NoError(t, err)
	assert.Len(t, orders, 2)
	assert.Equal(t, int64(123456789), orders[0].OrderID)
	assert.Equal(t, 34, orders[0].TypeID)
	assert.Equal(t, 5.50, orders[0].Price)
	assert.False(t, orders[0].IsBuyOrder)

	assert.Equal(t, int64(987654321), orders[1].OrderID)
	assert.Equal(t, 5.25, orders[1].Price)
	assert.True(t, orders[1].IsBuyOrder)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetMarketOrders_Empty tests retrieving orders when database is empty
func TestGetMarketOrders_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	s := miniredis.RunT(t)
	defer s.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	repo := database.NewMarketRepository(mock)
	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}
	client, err := NewClient(redisClient, cfg, repo)
	require.NoError(t, err)
	defer client.Close()

	// Empty result set
	rows := pgxmock.NewRows([]string{
		"order_id", "type_id", "region_id", "location_id", "is_buy_order",
		"price", "volume_total", "volume_remain", "min_volume",
		"issued", "duration", "fetched_at",
	})

	mock.ExpectQuery(`SELECT (.+) FROM market_orders WHERE region_id = \$1 AND type_id = \$2`).
		WithArgs(10000002, 999999).
		WillReturnRows(rows)

	orders, err := client.GetMarketOrders(context.Background(), 10000002, 999999)

	require.NoError(t, err)
	assert.Empty(t, orders)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestGetMarketOrders_DatabaseError tests error handling
func TestGetMarketOrders_DatabaseError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	s := miniredis.RunT(t)
	defer s.Close()
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	repo := database.NewMarketRepository(mock)
	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}
	client, err := NewClient(redisClient, cfg, repo)
	require.NoError(t, err)
	defer client.Close()

	// Simulate database error
	mock.ExpectQuery(`SELECT (.+) FROM market_orders WHERE region_id = \$1 AND type_id = \$2`).
		WithArgs(10000002, 34).
		WillReturnError(assert.AnError)

	orders, err := client.GetMarketOrders(context.Background(), 10000002, 34)

	assert.Error(t, err)
	assert.Nil(t, orders)
	assert.Contains(t, err.Error(), "failed to query market orders")
	assert.NoError(t, mock.ExpectationsWereMet())
}
