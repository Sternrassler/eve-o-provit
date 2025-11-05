// Package services - Redis Cache Integration Tests
//go:build integration
// +build integration

package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupRedisContainer starts a Redis container for testing
func setupRedisContainer(t *testing.T) (*redis.Client, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	host, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	port, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	redisClient := redis.NewClient(&redis.Options{
		Addr: host + ":" + port.Port(),
	})

	// Verify connection
	_, err = redisClient.Ping(ctx).Result()
	require.NoError(t, err)

	cleanup := func() {
		redisClient.Close()
		redisContainer.Terminate(ctx)
	}

	return redisClient, cleanup
}

// TestMarketOrderCache_SetAndGet_Integration tests basic cache set and get with real Redis
func TestMarketOrderCache_SetAndGet_Integration(t *testing.T) {
	redisClient, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	cache := &MarketOrderCache{
		redis: redisClient,
		ttl:   5 * time.Minute,
	}

	// Test data
	regionID := 10000002
	orders := []database.MarketOrder{
		{
			OrderID:      12345,
			TypeID:       34,
			RegionID:     regionID,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        100.0,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			Duration:     90,
			FetchedAt:    time.Now(),
		},
		{
			OrderID:      12346,
			TypeID:       35,
			RegionID:     regionID,
			LocationID:   60003761,
			IsBuyOrder:   true,
			Price:        95.0,
			VolumeTotal:  2000,
			VolumeRemain: 1500,
			Duration:     90,
			FetchedAt:    time.Now(),
		},
	}

	// Set cache
	err := cache.Set(ctx, regionID, orders)
	require.NoError(t, err)

	// Get from cache (direct cache read, not using Get which would fetch)
	cacheKey := "market_orders:10000002"
	data, err := redisClient.Get(ctx, cacheKey).Bytes()
	require.NoError(t, err)

	// Decompress and verify
	cachedOrders, err := cache.decompress(data)
	require.NoError(t, err)
	assert.Equal(t, len(orders), len(cachedOrders))
	assert.Equal(t, orders[0].OrderID, cachedOrders[0].OrderID)
	assert.Equal(t, orders[1].Price, cachedOrders[1].Price)
}

// TestMarketOrderCache_CacheMiss_Integration tests cache miss behavior with real Redis
func TestMarketOrderCache_CacheMiss_Integration(t *testing.T) {
	redisClient, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()

	// Try to get non-existent key
	cacheKey := "market_orders:99999"
	_, err := redisClient.Get(ctx, cacheKey).Bytes()
	assert.Error(t, err)
	assert.Equal(t, redis.Nil, err)
}

// TestMarketOrderCache_TTLExpiration_Integration tests TTL expiration with real Redis
func TestMarketOrderCache_TTLExpiration_Integration(t *testing.T) {
	redisClient, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	cache := &MarketOrderCache{
		redis: redisClient,
		ttl:   1 * time.Second, // Short TTL for testing
	}

	regionID := 10000002
	orders := []database.MarketOrder{
		{OrderID: 12345, TypeID: 34, Price: 100.0},
	}

	// Set cache
	err := cache.Set(ctx, regionID, orders)
	require.NoError(t, err)

	// Verify data exists
	cacheKey := "market_orders:10000002"
	_, err = redisClient.Get(ctx, cacheKey).Bytes()
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Verify data expired
	_, err = redisClient.Get(ctx, cacheKey).Bytes()
	assert.Error(t, err)
	assert.Equal(t, redis.Nil, err)
}

// TestMarketOrderCache_GzipCompression_Integration tests gzip compression with real Redis
func TestMarketOrderCache_GzipCompression_Integration(t *testing.T) {
	redisClient, cleanup := setupRedisContainer(t)
	defer cleanup()

	cache := &MarketOrderCache{
		redis: redisClient,
		ttl:   5 * time.Minute,
	}

	// Generate large dataset (simulate 100k orders)
	regionID := 10000002
	orders := make([]database.MarketOrder, 100000)
	for i := 0; i < 100000; i++ {
		orders[i] = database.MarketOrder{
			OrderID:      int64(i + 1),
			TypeID:       34,
			RegionID:     regionID,
			LocationID:   60003760,
			Price:        100.0 + float64(i),
			VolumeTotal:  1000,
			VolumeRemain: 500,
		}
	}

	// Compress
	compressed, err := cache.compress(orders)
	require.NoError(t, err)

	// Decompress
	decompressed, err := cache.decompress(compressed)
	require.NoError(t, err)

	// Verify data integrity
	assert.Equal(t, len(orders), len(decompressed))
	assert.Equal(t, orders[0].OrderID, decompressed[0].OrderID)
	assert.Equal(t, orders[99999].Price, decompressed[99999].Price)

	// Verify compression ratio
	// Uncompressed JSON size estimate: ~100k orders * ~150 bytes = ~15MB
	// Compressed should be < 20% (< 3MB)
	t.Logf("Compressed size: %d bytes (%.2f%% of estimated uncompressed)",
		len(compressed), float64(len(compressed))/float64(100000*150)*100)
	assert.Less(t, len(compressed), 100000*150*20/100, "Compression should reduce size to <20%")
}

// TestMarketOrderCache_ConcurrentAccess_Integration tests concurrent cache access with real Redis
func TestMarketOrderCache_ConcurrentAccess_Integration(t *testing.T) {
	redisClient, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	cache := &MarketOrderCache{
		redis: redisClient,
		ttl:   5 * time.Minute,
	}

	regionID := 10000002
	numGoroutines := 10
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			orders := []database.MarketOrder{
				{OrderID: int64(id), TypeID: 34, Price: float64(id)},
			}
			_ = cache.Set(ctx, regionID, orders)
		}(i)
	}

	wg.Wait()

	// Verify last write wins (any valid data is acceptable)
	cacheKey := "market_orders:10000002"
	data, err := redisClient.Get(ctx, cacheKey).Bytes()
	require.NoError(t, err)

	cachedOrders, err := cache.decompress(data)
	require.NoError(t, err)
	assert.Equal(t, 1, len(cachedOrders), "Should have exactly 1 order (last write wins)")
}

// TestNavigationCache_SetAndGet_Integration tests navigation cache with real Redis
func TestNavigationCache_SetAndGet_Integration(t *testing.T) {
	redisClient, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	cache := NewNavigationCache(redisClient)

	systemA := int64(30000142) // Jita
	systemB := int64(30000144) // Perimeter
	result := NavigationResult{
		TravelTimeSeconds: 30.0,
		Jumps:             1,
	}

	// Set cache
	err := cache.Set(ctx, systemA, systemB, result)
	require.NoError(t, err)

	// Get from cache
	cached, err := cache.Get(ctx, systemA, systemB)
	require.NoError(t, err)
	assert.Equal(t, result.TravelTimeSeconds, cached.TravelTimeSeconds)
	assert.Equal(t, result.Jumps, cached.Jumps)
}

// TestNavigationCache_TTL_Integration tests navigation cache TTL with real Redis
func TestNavigationCache_TTL_Integration(t *testing.T) {
	redisClient, cleanup := setupRedisContainer(t)
	defer cleanup()

	ctx := context.Background()
	cache := &NavigationCache{
		redis: redisClient,
		ttl:   1 * time.Second, // Short TTL for testing
	}

	systemA := int64(30000142)
	systemB := int64(30000144)
	result := NavigationResult{TravelTimeSeconds: 30.0, Jumps: 1}

	// Set cache
	err := cache.Set(ctx, systemA, systemB, result)
	require.NoError(t, err)

	// Verify exists
	_, err = cache.Get(ctx, systemA, systemB)
	require.NoError(t, err)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Verify expired
	_, err = cache.Get(ctx, systemA, systemB)
	assert.Error(t, err)
}
