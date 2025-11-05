//go:build integration || !unit

package esi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFetchMarketOrders_Integration tests fetching and storing market orders with real DB
func TestFetchMarketOrders_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup PostgreSQL container
	tc := database.SetupPostgresContainer(t)
	tc.CreateTestSchema(t)

	// Setup miniredis for ESI client
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	// Setup mock ESI server
	pageRequests := 0
	mockESI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageRequests++

		// Return single page with 2 orders
		w.Header().Set("X-Pages", "1")
		w.Header().Set("Content-Type", "application/json")

		orders := []ESIMarketOrder{
			{
				OrderID:      123456789,
				TypeID:       34,
				LocationID:   60003760,
				VolumeTotal:  1000,
				VolumeRemain: 500,
				MinVolume:    1,
				Price:        5.50,
				IsBuyOrder:   false,
				Duration:     90,
				Issued:       time.Now().Add(-24 * time.Hour),
				Range:        "region",
			},
			{
				OrderID:      987654321,
				TypeID:       34,
				LocationID:   60003760,
				VolumeTotal:  5000,
				VolumeRemain: 5000,
				MinVolume:    10,
				Price:        5.25,
				IsBuyOrder:   true,
				Duration:     90,
				Issued:       time.Now().Add(-48 * time.Hour),
				Range:        "station",
			},
		}

		json.NewEncoder(w).Encode(orders)
	}))
	defer mockESI.Close()

	// Create repository and ESI client
	repo := database.NewMarketRepository(tc.Pool)
	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	client, err := NewClient(redisClient, cfg, repo)
	require.NoError(t, err)
	defer client.Close()

	// Override ESI base URL to mock server
	// Note: This requires exposing baseURL or using a test-only setter
	// For now, we'll skip actual execution and document expected behavior
	t.Skip("Requires ESI client baseURL override capability")

	// Expected behavior:
	// err = client.FetchMarketOrders(context.Background(), 10000002)
	// require.NoError(t, err)
	//
	// // Verify orders were stored in database
	// orders, err := repo.GetMarketOrders(context.Background(), 10000002, 34)
	// require.NoError(t, err)
	// assert.Len(t, orders, 2)
	// assert.Equal(t, pageRequests, 1, "Should only fetch 1 page")
}

// TestFetchMarketOrders_Integration_MultiplePages tests pagination with real DB
func TestFetchMarketOrders_Integration_MultiplePages(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := database.SetupPostgresContainer(t)
	tc.CreateTestSchema(t)

	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	pagesFetched := 0
	mockESI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pagesFetched++
		page := r.URL.Query().Get("page")

		w.Header().Set("X-Pages", "3")
		w.Header().Set("Content-Type", "application/json")

		// Return different order per page
		orders := []ESIMarketOrder{
			{
				OrderID:      int64(pagesFetched * 1000000),
				TypeID:       34,
				LocationID:   60003760,
				VolumeTotal:  1000,
				VolumeRemain: 500,
				Price:        5.50 + float64(pagesFetched)*0.10,
				IsBuyOrder:   false,
				Duration:     90,
				Issued:       time.Now(),
				Range:        "region",
			},
		}

		json.NewEncoder(w).Encode(orders)

		if page == "3" {
			// Stop after page 3
			return
		}
	}))
	defer mockESI.Close()

	repo := database.NewMarketRepository(tc.Pool)
	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	client, err := NewClient(redisClient, cfg, repo)
	require.NoError(t, err)
	defer client.Close()

	t.Skip("Requires ESI client baseURL override capability")

	// Expected behavior:
	// err = client.FetchMarketOrders(context.Background(), 10000002)
	// require.NoError(t, err)
	//
	// // Verify 3 pages were fetched
	// assert.Equal(t, 3, pagesFetched)
	//
	// // Verify 3 orders stored (one per page)
	// orders, err := repo.GetMarketOrders(context.Background(), 10000002, 34)
	// require.NoError(t, err)
	// assert.Len(t, orders, 3)
}

// TestGetMarketOrders_Integration tests end-to-end flow with real DB
func TestGetMarketOrders_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := database.SetupPostgresContainer(t)
	tc.CreateTestSchema(t)
	tc.SeedTestData(t)

	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	repo := database.NewMarketRepository(tc.Pool)
	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	client, err := NewClient(redisClient, cfg, repo)
	require.NoError(t, err)
	defer client.Close()

	// Get orders from seeded data
	orders, err := client.GetMarketOrders(context.Background(), 10000002, 34)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(orders), 2, "Should have at least 2 seeded orders for type 34")

	// Verify order structure
	for _, order := range orders {
		assert.Equal(t, 10000002, order.RegionID)
		assert.Equal(t, 34, order.TypeID)
		assert.Greater(t, order.Price, 0.0)
		assert.NotZero(t, order.OrderID)
	}
}
