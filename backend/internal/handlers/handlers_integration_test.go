//go:build integration || !unit

package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupRedisContainer creates Redis testcontainer and returns client
func setupRedisContainer(t *testing.T) *redis.Client {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections"),
	}

	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := redisC.Terminate(ctx); err != nil {
			t.Logf("failed to terminate Redis container: %v", err)
		}
	})

	host, err := redisC.Host(ctx)
	require.NoError(t, err)

	port, err := redisC.MappedPort(ctx, "6379")
	require.NoError(t, err)

	client := redis.NewClient(&redis.Options{
		Addr: host + ":" + port.Port(),
	})

	// Test connection
	err = client.Ping(ctx).Err()
	require.NoError(t, err, "Failed to connect to Redis testcontainer")

	return client
}

// TestGetMarketOrders_Integration tests handler with real database
func TestGetMarketOrders_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup PostgreSQL container
	tc := database.SetupPostgresContainer(t)
	tc.CreateTestSchema(t)
	tc.SeedTestData(t)

	// Setup Redis container
	redisClient := setupRedisContainer(t)

	// Create market repository
	marketRepo := database.NewMarketRepository(tc.Pool)

	// Create ESI client for handler (reads from marketRepo)
	esiClient, err := esi.NewClient(redisClient, esi.Config{
		UserAgent:      "eve-o-provit-integration-test",
		RateLimit:      20,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}, marketRepo)
	require.NoError(t, err)

	// Create handler with ESI client (db=nil, sdeRepo=nil, marketRepo, esiClient)
	handler := New(nil, nil, marketRepo, esiClient)

	// Create Fiber app
	app := fiber.New()
	app.Get("/market/:region/:type", handler.GetMarketOrders)

	// Test: Get seeded market orders
	req := httptest.NewRequest("GET", "/market/10000002/34", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Verify response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Parse response body (handler now returns array directly)
	var orders []database.MarketOrder
	err = json.NewDecoder(resp.Body).Decode(&orders)
	require.NoError(t, err)

	// Verify we got the seeded data
	assert.GreaterOrEqual(t, len(orders), 2, "Should have at least 2 seeded orders")

	// Verify order structure
	for _, order := range orders {
		assert.Equal(t, 10000002, order.RegionID)
		assert.Equal(t, 34, order.TypeID)
		assert.Greater(t, order.Price, 0.0)
	}
}

// TestGetMarketOrders_Integration_InvalidParams tests error handling
func TestGetMarketOrders_Integration_InvalidParams(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := database.SetupPostgresContainer(t)
	tc.CreateTestSchema(t)

	redisClient := setupRedisContainer(t)
	marketRepo := database.NewMarketRepository(tc.Pool)

	// Create ESI client
	esiClient, err := esi.NewClient(redisClient, esi.Config{
		UserAgent:      "eve-o-provit-integration-test",
		RateLimit:      20,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}, marketRepo)
	require.NoError(t, err)

	handler := New(nil, nil, marketRepo, esiClient)

	app := fiber.New()
	app.Get("/market/:region/:type", handler.GetMarketOrders)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
	}{
		{
			name:           "invalid region ID",
			url:            "/market/abc/34",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "invalid type ID",
			url:            "/market/10000002/xyz",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "negative region ID",
			url:            "/market/-1/34",
			expectedStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestGetMarketOrders_Integration_EmptyResult tests handling of non-existent data
func TestGetMarketOrders_Integration_EmptyResult(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := database.SetupPostgresContainer(t)
	tc.CreateTestSchema(t)
	// No seed data = empty result

	redisClient := setupRedisContainer(t)
	marketRepo := database.NewMarketRepository(tc.Pool)

	// Create ESI client
	esiClient, err := esi.NewClient(redisClient, esi.Config{
		UserAgent:      "eve-o-provit-integration-test",
		RateLimit:      20,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}, marketRepo)
	require.NoError(t, err)

	handler := New(nil, nil, marketRepo, esiClient)

	app := fiber.New()
	app.Get("/market/:region/:type", handler.GetMarketOrders)

	req := httptest.NewRequest("GET", "/market/10000002/34", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	// Parse response body (handler now returns array directly)
	var orders []database.MarketOrder
	err = json.NewDecoder(resp.Body).Decode(&orders)
	require.NoError(t, err)
	assert.Empty(t, orders)
}

// TestHealth_Integration tests health endpoint (requires full DB setup)
func TestHealth_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Skip("Health endpoint requires full database.DB setup with SDE - use E2E tests instead")

	// To properly test this, we would need:
	// 1. PostgreSQL testcontainer
	// 2. SQLite SDE database
	// 3. Full database.DB initialization
	// This is better suited for E2E tests
}

// TestVersion_Integration tests version endpoint
func TestVersion_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Version doesn't need any dependencies
	handler := &Handler{} // Empty handler is fine

	app := fiber.New()
	app.Get("/version", handler.Version)

	req := httptest.NewRequest("GET", "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]string
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.NotEmpty(t, result["version"])
}

// TestGetMarketDataStaleness_Integration tests staleness check endpoint
func TestGetMarketDataStaleness_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := database.SetupPostgresContainer(t)
	tc.CreateTestSchema(t)
	tc.SeedTestData(t)

	marketRepo := database.NewMarketRepository(tc.Pool)
	// Need DB with Postgres for GetMarketDataStaleness (uses postgresQuery)
	db := &database.DB{Postgres: tc.Pool}
	handler := New(db, nil, marketRepo, nil)

	app := fiber.New()
	app.Get("/market/staleness/:region", handler.GetMarketDataStaleness)

	// Test with valid region
	req := httptest.NewRequest("GET", "/market/staleness/10000002", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.NotNil(t, result["region_id"])
	assert.NotNil(t, result["total_orders"])
	assert.NotNil(t, result["latest_fetch"])
	assert.NotNil(t, result["age_minutes"])
}
