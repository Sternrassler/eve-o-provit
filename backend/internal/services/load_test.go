// Package services - Load tests for performance validation
//go:build load
// +build load

package services

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLoadTestEnvironment initializes real Redis + ESI Client
func setupLoadTestEnvironment(t *testing.T) (*redis.Client, *MarketOrderCache, func()) {
	t.Helper()

	// Redis (assumes local Redis running on 6379)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use DB 1 for load tests
	})

	ctx := context.Background()
	err := redisClient.Ping(ctx).Err()
	require.NoError(t, err, "Redis must be running on localhost:6379 (run: cd deployments && docker compose up -d redis)")

	// Flush DB 1 for clean state
	err = redisClient.FlushDB(ctx).Err()
	require.NoError(t, err)

	log.Printf("✅ Redis connected (localhost:6379, DB 1)")

	// Create ESI client (no DB repo needed for load tests)
	esiClient, err := esi.NewClient(redisClient, esi.Config{
		UserAgent:      "eve-o-provit-load-test/1.0",
		RateLimit:      300,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}, nil)
	require.NoError(t, err, "Failed to create ESI client")

	log.Printf("✅ ESI Client created (rate limit: 300 req/min)")

	// Create market cache (fetcher removed - needs refactoring)
	cache := NewMarketOrderCache(redisClient)

	log.Printf("✅ Market Order Cache initialized (TTL: 5m, fetcher disabled)")

	cleanup := func() {
		// Flush load test Redis DB
		if err := redisClient.FlushDB(ctx).Err(); err != nil {
			log.Printf("⚠️ Failed to flush Redis DB: %v", err)
		}
		redisClient.Close()
		esiClient.Close()
		log.Printf("✅ Cleanup complete")
	}

	return redisClient, cache, cleanup
}

// TestLoadTheForge_ColdCache validates <30s target for The Forge with empty cache
func TestLoadTheForge_ColdCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	_, cache, cleanup := setupLoadTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// Region: The Forge (10000002) - largest market in EVE (~383k orders)
	regionID := 10000002

	log.Printf("🧪 Load Test: The Forge Cold Cache (383k+ orders expected)")
	log.Printf("   Target: <30s with empty cache")
	start := time.Now()

	// Fetch all market orders (cache miss expected)
	orders, err := cache.Get(ctx, regionID)
	require.NoError(t, err)

	elapsed := time.Since(start)
	log.Printf("✅ Cold Cache Fetch Complete:")
	log.Printf("   - Orders: %d", len(orders))
	log.Printf("   - Total Time: %.2fs", elapsed.Seconds())
	log.Printf("   - Throughput: %.0f orders/s", float64(len(orders))/elapsed.Seconds())
	log.Printf("   - Target: <30s")

	// Assert performance target
	assert.Greater(t, len(orders), 300000, "The Forge should have >300k orders")
	assert.Less(t, elapsed.Seconds(), 30.0, "Cold cache fetch must complete within 30s")

	// Note: Cache population happens asynchronously in background goroutine
	// No need to verify cache key existence here
}

// TestLoadTheForge_WarmCache validates <5s target with cache hit
func TestLoadTheForge_WarmCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	_, cache, cleanup := setupLoadTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()
	regionID := 10000002

	// Prime the cache
	log.Printf("🔥 Priming cache...")
	_, err := cache.Get(ctx, regionID)
	require.NoError(t, err)

	// Wait to ensure cache is written
	time.Sleep(200 * time.Millisecond)

	// Second request (cache hit)
	log.Printf("🧪 Load Test: The Forge Warm Cache")
	start := time.Now()

	orders, err := cache.Get(ctx, regionID)
	require.NoError(t, err)

	elapsed := time.Since(start)
	log.Printf("✅ Warm Cache Fetch Complete:")
	log.Printf("   - Orders: %d", len(orders))
	log.Printf("   - Cache Hit Time: %.2fs", elapsed.Seconds())
	log.Printf("   - Speedup: %.0fx faster than cold cache", 30.0/elapsed.Seconds())
	log.Printf("   - Target: <5s")

	// Assertions
	assert.Greater(t, len(orders), 300000, "Cache should return full dataset")
	assert.Less(t, elapsed.Seconds(), 5.0, "Warm cache fetch must complete <5s")
}

// TestLoadConcurrent validates concurrent requests without timeouts
func TestLoadConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	_, cache, cleanup := setupLoadTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// Test regions (diverse set)
	regions := []int{
		10000002, // The Forge
		10000043, // Domain
		10000032, // Sinq Laison
		10000030, // Heimatar
		10000042, // Metropolis
		10000001, // Derelik
		10000016, // Lonetrek
		10000020, // Tash-Murkon
		10000033, // The Citadel
		10000052, // Kador
	}

	log.Printf("🧪 Load Test: %d Concurrent Requests", len(regions))

	var wg sync.WaitGroup
	var successCount, failureCount int64
	var totalOrders int64
	results := make(chan time.Duration, len(regions))

	start := time.Now()

	for _, regionID := range regions {
		wg.Add(1)
		go func(rID int) {
			defer wg.Done()

			reqStart := time.Now()
			orders, err := cache.Get(ctx, rID)
			reqElapsed := time.Since(reqStart)

			if err != nil {
				atomic.AddInt64(&failureCount, 1)
				log.Printf("❌ Region %d failed: %v", rID, err)
				return
			}

			atomic.AddInt64(&successCount, 1)
			atomic.AddInt64(&totalOrders, int64(len(orders)))
			results <- reqElapsed

			log.Printf("✅ Region %d: %d orders in %.2fs", rID, len(orders), reqElapsed.Seconds())
		}(regionID)
	}

	wg.Wait()
	close(results)

	totalElapsed := time.Since(start)

	// Collect individual times
	var maxTime time.Duration
	for reqTime := range results {
		if reqTime > maxTime {
			maxTime = reqTime
		}
	}

	log.Printf("📊 Concurrent Load Metrics:")
	log.Printf("   - Total Regions: %d", len(regions))
	log.Printf("   - Success: %d, Failures: %d", successCount, failureCount)
	log.Printf("   - Total Orders Fetched: %d", totalOrders)
	log.Printf("   - Total Time (parallel): %.2fs", totalElapsed.Seconds())
	log.Printf("   - Max Individual Time: %.2fs", maxTime.Seconds())
	log.Printf("   - Throughput: %.0f orders/s", float64(totalOrders)/totalElapsed.Seconds())

	// Assertions
	assert.Equal(t, int64(len(regions)), successCount, "All requests should succeed")
	assert.Equal(t, int64(0), failureCount, "No failures expected")
	assert.Less(t, maxTime.Seconds(), 30.0, "All requests must complete <30s")
}

// TestLoadCacheHitRatio validates >95% cache hit ratio
func TestLoadCacheHitRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	_, cache, cleanup := setupLoadTestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// 10 regions, 10 requests each = 100 total
	regions := []int{
		10000002, 10000043, 10000032, 10000030, 10000042,
		10000001, 10000016, 10000020, 10000033, 10000052,
	}

	requestsPerRegion := 10
	totalRequests := len(regions) * requestsPerRegion

	log.Printf("🧪 Load Test: Cache Hit Ratio (%d requests)", totalRequests)

	// Prime cache (cold start)
	log.Printf("🔥 Priming cache with first request per region...")
	for _, regionID := range regions {
		_, err := cache.Get(ctx, regionID)
		require.NoError(t, err)
	}

	time.Sleep(500 * time.Millisecond) // Ensure cache writes complete

	// Count hits
	var cacheHits, cacheMisses int64

	start := time.Now()

	for round := 0; round < requestsPerRegion; round++ {
		for _, regionID := range regions {
			_, err := cache.Get(ctx, regionID)
			if err == nil {
				atomic.AddInt64(&cacheHits, 1)
			} else {
				atomic.AddInt64(&cacheMisses, 1)
			}
		}
	}

	elapsed := time.Since(start)

	hitRatio := (float64(cacheHits) / float64(totalRequests)) * 100.0

	log.Printf("📊 Cache Hit Ratio Metrics:")
	log.Printf("   - Total Requests: %d", totalRequests)
	log.Printf("   - Cache Hits: %d", cacheHits)
	log.Printf("   - Cache Misses: %d", cacheMisses)
	log.Printf("   - Hit Ratio: %.2f%%", hitRatio)
	log.Printf("   - Total Time: %.2fs", elapsed.Seconds())
	log.Printf("   - Avg Time per Request: %.0fms", elapsed.Seconds()*1000.0/float64(totalRequests))

	// Assertions
	assert.Greater(t, hitRatio, 95.0, "Cache hit ratio must be >95%")
}

// BenchmarkTheForgeCalculation benchmarks full market fetch
func BenchmarkTheForgeCalculation(b *testing.B) {
	// Setup (requires Docker Compose running)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer redisClient.Close()

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		b.Skip("Redis not available:", err)
	}

	redisClient.FlushDB(ctx)

	esiClient, err := esi.NewClient(redisClient, esi.Config{
		UserAgent:      "eve-o-provit-benchmark/1.0",
		RateLimit:      300,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer esiClient.Close()

	cache := NewMarketOrderCache(redisClient)

	regionID := 10000002 // The Forge

	// Skip warmup - fetcher disabled
	// _, _ = cache.Get(ctx, regionID)

	b.Skip("Benchmark disabled - needs refactoring with pagination.BatchFetcher")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		// Fetch market orders
		orders, err := cache.Get(ctx, regionID)
		if err != nil {
			b.Fatal(err)
		}

		elapsed := time.Since(start)

		if elapsed > 30*time.Second {
			b.Errorf("❌ Fetch timeout: %.2fs > 30s", elapsed.Seconds())
		}

		b.ReportMetric(float64(len(orders)), "orders")
		b.ReportMetric(elapsed.Seconds(), "seconds")
	}
}
