package services

import (
	"context"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMarketOrderCache tests cache initialization
func TestNewMarketOrderCache(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.redis)
	assert.Equal(t, 5*time.Minute, cache.ttl)
}

// TestMarketOrderCache_SetAndGet tests cache set and get operations
func TestMarketOrderCache_SetAndGet(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	// Test data
	regionID := 10000002
	orders := []database.MarketOrder{
		{
			OrderID:      12345,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			VolumeTotal:  1000000,
			VolumeRemain: 500000,
			Price:        5.50,
			IsBuyOrder:   false,
		},
		{
			OrderID:      67890,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			VolumeTotal:  2000000,
			VolumeRemain: 2000000,
			Price:        5.25,
			IsBuyOrder:   true,
		},
	}

	// Set cache
	err := cache.Set(ctx, regionID, orders)
	require.NoError(t, err)

	// Get from cache
	cachedOrders, err := cache.Get(ctx, regionID)
	require.NoError(t, err)
	assert.NotNil(t, cachedOrders)
	assert.Len(t, cachedOrders, 2)
	assert.Equal(t, int64(12345), cachedOrders[0].OrderID)
	assert.Equal(t, int64(67890), cachedOrders[1].OrderID)
	assert.Equal(t, 5.50, cachedOrders[0].Price)
	assert.Equal(t, 5.25, cachedOrders[1].Price)
}

// TestMarketOrderCache_GetMiss tests cache miss scenario
func TestMarketOrderCache_GetMiss(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	// Try to get from empty cache
	orders, err := cache.Get(ctx, 10000002)
	assert.Error(t, err)
	assert.Nil(t, orders)
}

// TestMarketOrderCache_Expiration tests TTL expiration
func TestMarketOrderCache_Expiration(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	regionID := 10000002
	orders := []database.MarketOrder{
		{OrderID: 12345, TypeID: 34, Price: 5.50},
	}

	// Set with default TTL (5 minutes from struct)
	err := cache.Set(ctx, regionID, orders)
	require.NoError(t, err)

	// Fast-forward time in miniredis beyond default TTL
	s.FastForward(6 * time.Minute)

	// Should be expired
	cachedOrders, err := cache.Get(ctx, regionID)
	assert.Error(t, err)
	assert.Nil(t, cachedOrders)
}

// TestMarketOrderCache_EmptyOrders tests caching empty order list
func TestMarketOrderCache_EmptyOrders(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	regionID := 10000002
	emptyOrders := []database.MarketOrder{}

	// Set empty orders
	err := cache.Set(ctx, regionID, emptyOrders)
	require.NoError(t, err)

	// Get should return empty slice
	cachedOrders, err := cache.Get(ctx, regionID)
	require.NoError(t, err)
	assert.NotNil(t, cachedOrders)
	assert.Len(t, cachedOrders, 0)
}

// TestMarketOrderCache_LargeDataset tests compression with large dataset
func TestMarketOrderCache_LargeDataset(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	// Large order set to trigger compression
	orders := make([]database.MarketOrder, 100)
	for i := 0; i < 100; i++ {
		orders[i] = database.MarketOrder{
			OrderID:      int64(i + 1000),
			TypeID:       34,
			RegionID:     10000002,
			VolumeTotal:  1000000,
			VolumeRemain: 500000,
			Price:        5.50 + float64(i)*0.01,
		}
	}

	err := cache.Set(ctx, 10000002, orders)
	require.NoError(t, err)

	cachedOrders, err := cache.Get(ctx, 10000002)
	require.NoError(t, err)
	assert.Len(t, cachedOrders, 100)
	assert.Equal(t, int64(1000), cachedOrders[0].OrderID)
	assert.Equal(t, int64(1099), cachedOrders[99].OrderID)
	assert.InDelta(t, 5.50, cachedOrders[0].Price, 0.001)
	assert.InDelta(t, 6.49, cachedOrders[99].Price, 0.001)
}

// TestNewNavigationCache tests navigation cache initialization
func TestNewNavigationCache(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.redis)
	assert.Equal(t, 1*time.Hour, cache.ttl)
}

// TestNavigationCache_SetAndGet tests navigation cache operations
func TestNavigationCache_SetAndGet(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Test navigation data
	systemA := int64(30000142) // Jita
	systemB := int64(30002187) // Amarr
	result := NavigationResult{
		TravelTimeSeconds: 450.0,
		Jumps:             10,
	}

	// Set navigation result
	err := cache.Set(ctx, systemA, systemB, result)
	require.NoError(t, err)

	// Get navigation result
	cachedResult, err := cache.Get(ctx, systemA, systemB)
	require.NoError(t, err)
	assert.NotNil(t, cachedResult)
	assert.Equal(t, 10, cachedResult.Jumps)
	assert.InDelta(t, 450.0, cachedResult.TravelTimeSeconds, 0.001)
}

// TestNavigationCache_GetMiss tests navigation cache miss
func TestNavigationCache_GetMiss(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Try to get non-existent route
	result, err := cache.Get(ctx, 30000142, 30002187)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestNavigationCache_Expiration tests navigation cache TTL
func TestNavigationCache_Expiration(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	result := NavigationResult{
		TravelTimeSeconds: 300.0,
		Jumps:             5,
	}

	// Set with default TTL (1 hour)
	err := cache.Set(ctx, 30000142, 30002187, result)
	require.NoError(t, err)

	// Fast-forward time beyond TTL
	s.FastForward(2 * time.Hour)

	// Should be expired
	cachedResult, err := cache.Get(ctx, 30000142, 30002187)
	assert.Error(t, err)
	assert.Nil(t, cachedResult)
}

// TestNavigationCache_ZeroJumps tests caching zero-jump route (same system)
func TestNavigationCache_ZeroJumps(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Same system route
	sameSystem := int64(30000142)
	result := NavigationResult{
		TravelTimeSeconds: 0.0,
		Jumps:             0,
	}

	err := cache.Set(ctx, sameSystem, sameSystem, result)
	require.NoError(t, err)

	cachedResult, err := cache.Get(ctx, sameSystem, sameSystem)
	require.NoError(t, err)
	assert.NotNil(t, cachedResult)
	assert.Equal(t, 0, cachedResult.Jumps)
	assert.Equal(t, 0.0, cachedResult.TravelTimeSeconds)
}

// TestCacheKeyFormat tests that cache keys are correctly formatted
func TestCacheKeyFormat(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	orders := []database.MarketOrder{{OrderID: 12345}}
	err := cache.Set(ctx, 10000002, orders)
	require.NoError(t, err)

	// Check that key exists in Redis (miniredis exposes keys)
	keys := s.Keys()
	assert.NotEmpty(t, keys, "Cache key should exist in Redis")
	assert.Contains(t, keys[0], "market_orders:10000002", "Key should contain region ID")
}

// TestNavigationCacheKeyFormat tests navigation cache key format
func TestNavigationCacheKeyFormat(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	result := NavigationResult{TravelTimeSeconds: 100.0, Jumps: 3}
	err := cache.Set(ctx, 30000142, 30002187, result)
	require.NoError(t, err)

	// Check key format
	keys := s.Keys()
	assert.NotEmpty(t, keys)
	assert.Contains(t, keys[0], "nav:30000142:30002187", "Key should contain both system IDs")
}

// TestMarketOrderCache_MultipleRegions tests caching orders for different regions
func TestMarketOrderCache_MultipleRegions(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	// Cache orders for different regions
	regions := []struct {
		regionID int
		orders   []database.MarketOrder
	}{
		{
			regionID: 10000002, // The Forge (Jita)
			orders: []database.MarketOrder{
				{OrderID: 1001, TypeID: 34, Price: 5.50},
				{OrderID: 1002, TypeID: 34, Price: 5.25},
			},
		},
		{
			regionID: 10000043, // Domain (Amarr)
			orders: []database.MarketOrder{
				{OrderID: 2001, TypeID: 34, Price: 5.60},
				{OrderID: 2002, TypeID: 34, Price: 5.30},
			},
		},
		{
			regionID: 10000032, // Sinq Laison (Dodixie)
			orders: []database.MarketOrder{
				{OrderID: 3001, TypeID: 34, Price: 5.55},
			},
		},
	}

	// Set all regions
	for _, r := range regions {
		err := cache.Set(ctx, r.regionID, r.orders)
		require.NoError(t, err)
	}

	// Verify all regions independently
	for _, r := range regions {
		cachedOrders, err := cache.Get(ctx, r.regionID)
		require.NoError(t, err)
		assert.Len(t, cachedOrders, len(r.orders))
		assert.Equal(t, r.orders[0].OrderID, cachedOrders[0].OrderID)
		assert.Equal(t, r.orders[0].Price, cachedOrders[0].Price)
	}

	// Verify keys exist in Redis
	keys := s.Keys()
	assert.Len(t, keys, 3, "Should have 3 separate cache keys")
}

// TestNavigationCache_BidirectionalRoutes tests caching routes in both directions
func TestNavigationCache_BidirectionalRoutes(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	jita := int64(30000142)
	amarr := int64(30002187)

	// Route from Jita to Amarr
	jitaToAmarr := NavigationResult{
		TravelTimeSeconds: 450.0,
		Jumps:             10,
	}
	err := cache.Set(ctx, jita, amarr, jitaToAmarr)
	require.NoError(t, err)

	// Route from Amarr to Jita (different route characteristics)
	amarrToJita := NavigationResult{
		TravelTimeSeconds: 460.0, // Slightly different timing
		Jumps:             10,
	}
	err = cache.Set(ctx, amarr, jita, amarrToJita)
	require.NoError(t, err)

	// Verify both directions independently
	cachedJitaToAmarr, err := cache.Get(ctx, jita, amarr)
	require.NoError(t, err)
	assert.Equal(t, 10, cachedJitaToAmarr.Jumps)
	assert.InDelta(t, 450.0, cachedJitaToAmarr.TravelTimeSeconds, 0.1)

	cachedAmarrToJita, err := cache.Get(ctx, amarr, jita)
	require.NoError(t, err)
	assert.Equal(t, 10, cachedAmarrToJita.Jumps)
	assert.InDelta(t, 460.0, cachedAmarrToJita.TravelTimeSeconds, 0.1)

	// Verify separate keys
	keys := s.Keys()
	assert.Len(t, keys, 2, "Should have 2 separate cache keys for bidirectional routes")
}

// TestMarketOrderCache_CompressDecompress tests compression round-trip
func TestMarketOrderCache_CompressDecompress(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)

	// Create test orders
	orders := []database.MarketOrder{
		{
			OrderID:      123456,
			TypeID:       34,
			Price:        5.50,
			VolumeRemain: 1000,
		},
		{
			OrderID:      789012,
			TypeID:       35,
			Price:        10.25,
			VolumeRemain: 5000,
		},
	}

	// Compress
	compressed, err := cache.compress(orders)
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)
	// Compressed data should be smaller than JSON for large datasets
	assert.Less(t, len(compressed), 500, "Compressed data should be reasonably small")

	// Decompress
	decompressed, err := cache.decompress(compressed)
	require.NoError(t, err)
	assert.Len(t, decompressed, 2)
	assert.Equal(t, int64(123456), decompressed[0].OrderID)
	assert.Equal(t, 34, decompressed[0].TypeID)
	assert.Equal(t, 5.50, decompressed[0].Price)
	assert.Equal(t, int64(789012), decompressed[1].OrderID)
}

// TestMarketOrderCache_DecompressInvalidData tests error handling for corrupt data
func TestMarketOrderCache_DecompressInvalidData(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "invalid gzip data",
			data: []byte("not gzip compressed"),
		},
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "truncated gzip",
			data: []byte{0x1f, 0x8b}, // Gzip magic bytes only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, err := cache.decompress(tt.data)
			assert.Error(t, err)
			assert.Nil(t, orders)
		})
	}
}

// TestNavigationCache_GetMissing tests cache miss behavior
func TestNavigationCache_GetMissing(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Try to get non-existent route
	result, err := cache.Get(ctx, 30000142, 30002187)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "redis: nil") // Redis returns "redis: nil" for missing keys
}

// TestMarketOrderCache_SetEmpty tests setting empty orders
func TestMarketOrderCache_SetEmpty(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	// Set empty slice
	err := cache.Set(ctx, 10000002, []database.MarketOrder{})
	require.NoError(t, err)

	// Verify it was stored
	orders, err := cache.Get(ctx, 10000002)
	require.NoError(t, err)
	assert.Empty(t, orders)
}

// TestNavigationCache_SetGet tests navigation cache round-trip
func TestNavigationCache_SetGet(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Store route
	result := NavigationResult{
		TravelTimeSeconds: 350.5,
		Jumps:             7,
	}
	err := cache.Set(ctx, 30000142, 30002187, result)
	require.NoError(t, err)

	// Retrieve route
	cached, err := cache.Get(ctx, 30000142, 30002187)
	require.NoError(t, err)
	assert.Equal(t, 7, cached.Jumps)
	assert.InDelta(t, 350.5, cached.TravelTimeSeconds, 0.1)
}

// TestNavigationCache_GetCorruptData tests handling of corrupt cached data
func TestNavigationCache_GetCorruptData(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Store corrupt JSON directly in Redis
	cacheKey := "nav:30000142:30002187"
	err := redisClient.Set(ctx, cacheKey, "invalid json{", cache.ttl).Err()
	require.NoError(t, err)

	// Try to retrieve - should fail JSON unmarshal
	result, err := cache.Get(ctx, 30000142, 30002187)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "invalid character")
}

// TestMarketOrderCache_GetCorruptCompression tests handling of corrupt compressed data
func TestMarketOrderCache_GetCorruptCompression(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	// Store corrupt data (not valid gzip) in Redis
	cacheKey := "market_orders:10000002"
	err := redisClient.Set(ctx, cacheKey, []byte("not a valid gzip"), cache.ttl).Err()
	require.NoError(t, err)

	// Try to retrieve - should fail decompression
	orders, err := cache.Get(ctx, 10000002)
	assert.Error(t, err)
	assert.Nil(t, orders)
}
