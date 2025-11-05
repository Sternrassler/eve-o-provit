//go:build unit || !integration

package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarketOrderCache_compress_Success tests successful compression
func TestMarketOrderCache_compress_Success(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)

	orders := []database.MarketOrder{
		{OrderID: 1, TypeID: 100, RegionID: 10000002, Price: 1000.50, VolumeRemain: 100},
		{OrderID: 2, TypeID: 200, RegionID: 10000002, Price: 2000.75, VolumeRemain: 50},
	}

	compressed, err := cache.compress(orders)
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)

	// Verify it's valid gzip
	gzipReader, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer gzipReader.Close()

	var decompressed []database.MarketOrder
	err = json.NewDecoder(gzipReader).Decode(&decompressed)
	require.NoError(t, err)
	assert.Equal(t, orders, decompressed)
}

// TestMarketOrderCache_compress_EmptySlice tests compression of empty slice
func TestMarketOrderCache_compress_EmptySlice(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)

	orders := []database.MarketOrder{}

	compressed, err := cache.compress(orders)
	require.NoError(t, err)
	assert.NotEmpty(t, compressed)

	// Should decompress to empty slice
	decompressed, err := cache.decompress(compressed)
	require.NoError(t, err)
	assert.Empty(t, decompressed)
}

// TestMarketOrderCache_decompress_InvalidGzip tests decompression with invalid gzip data
func TestMarketOrderCache_decompress_InvalidGzip(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)

	// Random bytes that are not valid gzip
	invalidData := []byte("this is not gzip compressed data")

	orders, err := cache.decompress(invalidData)
	assert.Error(t, err)
	assert.Nil(t, orders)
}

// TestMarketOrderCache_decompress_InvalidJSON tests decompression with gzip but invalid JSON
func TestMarketOrderCache_decompress_InvalidJSON(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)

	// Valid gzip but invalid JSON content
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write([]byte("invalid json content {{{"))
	require.NoError(t, err)
	require.NoError(t, gzipWriter.Close())

	orders, err := cache.decompress(buf.Bytes())
	assert.Error(t, err)
	assert.Nil(t, orders)
}

// TestMarketOrderCache_Set_Success tests successful Set operation
func TestMarketOrderCache_Set_Success(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	orders := []database.MarketOrder{
		{OrderID: 1, TypeID: 100, RegionID: 10000002, Price: 1000.50, VolumeRemain: 100},
	}

	err := cache.Set(ctx, 10000002, orders)
	require.NoError(t, err)

	// Verify data was stored in Redis
	cacheKey := "market_orders:10000002"
	assert.True(t, s.Exists(cacheKey))

	// Verify TTL was set
	ttl := s.TTL(cacheKey)
	assert.True(t, ttl > 0 && ttl <= 30*time.Minute)
}

// TestMarketOrderCache_Set_RedisError tests Set with Redis connection error
func TestMarketOrderCache_Set_RedisError(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	orders := []database.MarketOrder{
		{OrderID: 1, TypeID: 100, RegionID: 10000002, Price: 1000.50, VolumeRemain: 100},
	}

	// Close miniredis to simulate connection error
	s.Close()

	err := cache.Set(ctx, 10000002, orders)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to set cache")
}

// TestMarketOrderCache_Get_CacheMiss tests Get with cache miss
func TestMarketOrderCache_Get_CacheMiss(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	orders, err := cache.Get(ctx, 10000002)
	assert.Error(t, err)
	assert.Nil(t, orders)
	assert.Contains(t, err.Error(), "cache miss")
}

// TestMarketOrderCache_Get_Success tests successful Get operation
func TestMarketOrderCache_Get_Success(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	orders := []database.MarketOrder{
		{OrderID: 1, TypeID: 100, RegionID: 10000002, Price: 1000.50, VolumeRemain: 100},
		{OrderID: 2, TypeID: 200, RegionID: 10000002, Price: 2000.75, VolumeRemain: 50},
	}

	// Set orders in cache
	err := cache.Set(ctx, 10000002, orders)
	require.NoError(t, err)

	// Get orders from cache
	retrieved, err := cache.Get(ctx, 10000002)
	require.NoError(t, err)
	assert.Equal(t, orders, retrieved)
}

// TestMarketOrderCache_Get_DecompressError tests Get with decompression error
// Note: Get() currently falls back to "cache miss" error when decompression fails
func TestMarketOrderCache_Get_DecompressError(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)
	ctx := context.Background()

	// Manually set invalid data in Redis (not gzip compressed)
	cacheKey := "market_orders:10000002"
	err := redisClient.Set(ctx, cacheKey, []byte("invalid gzip data"), 30*time.Minute).Err()
	require.NoError(t, err)

	// Get should fail - currently returns "cache miss" instead of decompression error
	// This is the actual behavior (falls through to cache miss on decompress failure)
	orders, err := cache.Get(ctx, 10000002)
	assert.Error(t, err)
	assert.Nil(t, orders)
	assert.Contains(t, err.Error(), "cache miss")
}

// TestNavigationCache_Get_Success tests successful Get operation
func TestNavigationCache_Get_Success(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Set test data using NavigationCache.Set
	systemA := int64(30000142)
	systemB := int64(30000144)
	expectedResult := NavigationResult{
		TravelTimeSeconds: 123.45,
		Jumps:             5,
	}

	err := cache.Set(ctx, systemA, systemB, expectedResult)
	require.NoError(t, err)

	// Get from cache
	result, err := cache.Get(ctx, systemA, systemB)
	require.NoError(t, err)
	assert.Equal(t, &expectedResult, result)
}

// TestNavigationCache_Set_Success tests successful Set operation
func TestNavigationCache_Set_Success(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	systemA := int64(30000142)
	systemB := int64(30000144)
	navResult := NavigationResult{
		TravelTimeSeconds: 100.0,
		Jumps:             3,
	}

	err := cache.Set(ctx, systemA, systemB, navResult)
	require.NoError(t, err)

	// Verify data was stored in Redis with correct key
	cacheKey := "nav:30000142:30000144"
	stored, err := redisClient.Get(ctx, cacheKey).Result()
	require.NoError(t, err)

	var storedResult NavigationResult
	err = json.Unmarshal([]byte(stored), &storedResult)
	require.NoError(t, err)
	assert.Equal(t, navResult, storedResult)
}

// TestNavigationCache_Set_RedisError tests Set with Redis connection error
func TestNavigationCache_Set_RedisError(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	navResult := NavigationResult{
		TravelTimeSeconds: 100.0,
		Jumps:             3,
	}

	// Close miniredis to simulate connection error
	s.Close()

	err := cache.Set(ctx, 30000142, 30000144, navResult)
	assert.Error(t, err)
}

// TestNavigationCache_Get_RedisError tests Get with Redis connection error
func TestNavigationCache_Get_RedisError(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Close miniredis to simulate connection error
	s.Close()

	result, err := cache.Get(ctx, 30000142, 30000144)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestNavigationCache_Get_CorruptJSON tests Get with corrupt JSON data
func TestNavigationCache_Get_CorruptJSON(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewNavigationCache(redisClient)
	ctx := context.Background()

	// Store invalid JSON in Redis
	cacheKey := "nav:30000142:30000144"
	err := redisClient.Set(ctx, cacheKey, []byte("invalid json {{{"), 1*time.Hour).Err()
	require.NoError(t, err)

	result, err := cache.Get(ctx, 30000142, 30000144)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestMarketOrderCache_RefreshBackground tests that RefreshBackground is callable
// Note: This is currently a no-op stub waiting for BatchFetcher refactoring
func TestMarketOrderCache_RefreshBackground(t *testing.T) {
	s := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	cache := NewMarketOrderCache(redisClient)

	// Should not panic or error - currently a no-op
	cache.RefreshBackground(10000002)
}
