// Package services - Redis Cache Implementation
package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/redis/go-redis/v9"
)

// MarketOrderCache provides Redis caching for market orders
// TODO: Refactor to use pagination.BatchFetcher instead of removed MarketOrderFetcher
type MarketOrderCache struct {
	redis *redis.Client
	ttl   time.Duration
	// fetcher *MarketOrderFetcher // Removed - needs refactoring
}

// NewMarketOrderCache creates a new market order cache
// TODO: Add BatchFetcher parameter after refactoring
func NewMarketOrderCache(redisClient *redis.Client) *MarketOrderCache {
	return &MarketOrderCache{
		redis: redisClient,
		ttl:   5 * time.Minute,
		// fetcher: fetcher, // Removed - needs refactoring
	}
}

// Get retrieves market orders from cache or fetches if not present
// TODO: Re-implement using pagination.BatchFetcher
func (c *MarketOrderCache) Get(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	cacheKey := fmt.Sprintf("market_orders:%d", regionID)

	// Try to get from cache
	data, err := c.redis.Get(ctx, cacheKey).Bytes()
	if err == nil {
		// Cache hit - decompress and unmarshal
		orders, err := c.decompress(data)
		if err == nil {
			return orders, nil
		}
		// If decompression fails, fall through to fetch
	}

	// Cache miss - TODO: implement fetching via BatchFetcher
	return nil, fmt.Errorf("cache miss and fetcher not implemented yet")
}

// Set stores market orders in cache with compression
func (c *MarketOrderCache) Set(ctx context.Context, regionID int, orders []database.MarketOrder) error {
	cacheKey := fmt.Sprintf("market_orders:%d", regionID)

	// Compress data
	compressed, err := c.compress(orders)
	if err != nil {
		return fmt.Errorf("failed to compress orders: %w", err)
	}

	// Store in Redis
	if err := c.redis.Set(ctx, cacheKey, compressed, c.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// RefreshBackground refreshes cache in background
// TODO: Re-implement using pagination.BatchFetcher
func (c *MarketOrderCache) RefreshBackground(regionID int) {
	// Disabled until refactoring complete
}

// compress compresses market orders using gzip
func (c *MarketOrderCache) compress(orders []database.MarketOrder) ([]byte, error) {
	// Marshal to JSON
	jsonData, err := json.Marshal(orders)
	if err != nil {
		return nil, err
	}

	// Compress with gzip
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write(jsonData); err != nil {
		return nil, err
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// decompress decompresses market orders from gzip
func (c *MarketOrderCache) decompress(data []byte) ([]database.MarketOrder, error) {
	// Decompress gzip
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	jsonData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON
	var orders []database.MarketOrder
	if err := json.Unmarshal(jsonData, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

// NavigationCache provides Redis caching for navigation data
type NavigationCache struct {
	redis *redis.Client
	ttl   time.Duration
}

// NewNavigationCache creates a new navigation cache
func NewNavigationCache(redisClient *redis.Client) *NavigationCache {
	return &NavigationCache{
		redis: redisClient,
		ttl:   1 * time.Hour,
	}
}

// NavigationResult represents cached navigation data
type NavigationResult struct {
	TravelTimeSeconds float64 `json:"travel_time_seconds"`
	Jumps             int     `json:"jumps"`
}

// Get retrieves navigation result from cache
func (c *NavigationCache) Get(ctx context.Context, systemA, systemB int64) (*NavigationResult, error) {
	cacheKey := fmt.Sprintf("nav:%d:%d", systemA, systemB)

	data, err := c.redis.Get(ctx, cacheKey).Bytes()
	if err != nil {
		return nil, err
	}

	var result NavigationResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Set stores navigation result in cache
func (c *NavigationCache) Set(ctx context.Context, systemA, systemB int64, result NavigationResult) error {
	cacheKey := fmt.Sprintf("nav:%d:%d", systemA, systemB)

	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, cacheKey, data, c.ttl).Err()
}
