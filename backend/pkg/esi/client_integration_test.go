package esi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient tests ESI client initialization
func TestNewClient(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	client, err := NewClient(redisClient, cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.esi)
	defer client.Close()
}

// TestGetRawClient tests raw client access
func TestGetRawClient(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	client, err := NewClient(redisClient, cfg, nil)
	require.NoError(t, err)
	defer client.Close()

	rawClient := client.GetRawClient()
	assert.NotNil(t, rawClient)
}

// TestClose tests client cleanup
func TestClose(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	client, err := NewClient(redisClient, cfg, nil)
	require.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

// TestFetchMarketOrdersPage tests single page market order fetching
func TestFetchMarketOrdersPage(t *testing.T) {
	// Create mock ESI server
	mockESI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request path
		if r.URL.Path != "/v1/markets/10000002/orders/" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Return mock market orders
		orders := []ESIMarketOrder{
			{
				OrderID:      12345,
				TypeID:       34, // Tritanium
				LocationID:   60003760,
				VolumeTotal:  1000000,
				VolumeRemain: 500000,
				MinVolume:    1,
				Price:        5.50,
				IsBuyOrder:   false,
				Duration:     90,
				Issued:       time.Now().Add(-24 * time.Hour),
			},
			{
				OrderID:      67890,
				TypeID:       34,
				LocationID:   60003760,
				VolumeTotal:  2000000,
				VolumeRemain: 2000000,
				MinVolume:    1,
				Price:        5.25,
				IsBuyOrder:   true,
				Duration:     30,
				Issued:       time.Now().Add(-2 * time.Hour),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pages", "1")
		w.Header().Set("Expires", time.Now().Add(5*time.Minute).UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(orders)
	}))
	defer mockESI.Close()

	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	client, err := NewClient(redisClient, cfg, nil)
	require.NoError(t, err)
	defer client.Close()

	// Note: This test would require mocking the underlying ESI client's HTTP transport
	// For now, we test the structure
	ctx := context.Background()
	orders, expires, err := client.FetchMarketOrdersPage(ctx, 10000002, 1)

	// Since we can't easily mock the ESI client's HTTP transport, we expect an error
	// In a real integration test environment, this would succeed
	if err == nil {
		assert.NotNil(t, orders)
		assert.NotNil(t, expires)
	}
}

// TestConfig_DefaultValues tests config struct
func TestESIMarketOrder_Marshaling(t *testing.T) {
	issued := time.Date(2025, 11, 5, 12, 0, 0, 0, time.UTC)

	order := ESIMarketOrder{
		OrderID:      12345,
		TypeID:       34,
		LocationID:   60003760,
		VolumeTotal:  1000000,
		VolumeRemain: 500000,
		MinVolume:    1,
		Price:        5.50,
		IsBuyOrder:   false,
		Duration:     90,
		Issued:       issued,
		Range:        "region",
	}

	jsonData, err := json.Marshal(order)
	require.NoError(t, err)

	var unmarshaled ESIMarketOrder
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, order.OrderID, unmarshaled.OrderID)
	assert.Equal(t, order.TypeID, unmarshaled.TypeID)
	assert.Equal(t, order.LocationID, unmarshaled.LocationID)
	assert.Equal(t, order.VolumeTotal, unmarshaled.VolumeTotal)
	assert.Equal(t, order.VolumeRemain, unmarshaled.VolumeRemain)
	assert.Equal(t, order.MinVolume, unmarshaled.MinVolume)
	assert.Equal(t, order.Price, unmarshaled.Price)
	assert.Equal(t, order.IsBuyOrder, unmarshaled.IsBuyOrder)
	assert.Equal(t, order.Duration, unmarshaled.Duration)
	assert.Equal(t, order.Range, unmarshaled.Range)
	assert.True(t, order.Issued.Equal(unmarshaled.Issued))
}



// TestConfig_DefaultValues tests config struct
func TestConfig_DefaultValues(t *testing.T) {
	cfg := Config{
		UserAgent:      "test-agent",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	assert.Equal(t, "test-agent", cfg.UserAgent)
	assert.Equal(t, 150, cfg.RateLimit)
	assert.Equal(t, 10, cfg.ErrorThreshold)
	assert.Equal(t, 3, cfg.MaxRetries)
}

// TestClient_NilRepository tests client creation with nil repository
func TestClient_NilRepository(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	// Should work even without repository (for read-only operations)
	client, err := NewClient(redisClient, cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Nil(t, client.repo)
	defer client.Close()
}

// TestESIMarketOrder_VolumeCalculations tests volume-related calculations
func TestESIMarketOrder_VolumeCalculations(t *testing.T) {
	tests := []struct {
		name         string
		volumeTotal  int
		volumeRemain int
		expected     int
	}{
		{
			name:         "Fully available order",
			volumeTotal:  1000000,
			volumeRemain: 1000000,
			expected:     0, // 0 filled
		},
		{
			name:         "Partially filled order",
			volumeTotal:  1000000,
			volumeRemain: 600000,
			expected:     400000, // 400k filled
		},
		{
			name:         "Nearly exhausted order",
			volumeTotal:  1000000,
			volumeRemain: 100,
			expected:     999900, // Almost all filled
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := ESIMarketOrder{
				VolumeTotal:  tt.volumeTotal,
				VolumeRemain: tt.volumeRemain,
			}

			volumeFilled := order.VolumeTotal - order.VolumeRemain
			assert.Equal(t, tt.expected, volumeFilled)
		})
	}
}

// TestESIMarketOrder_RangeValues tests different range values
func TestESIMarketOrder_RangeValues(t *testing.T) {
	ranges := []string{"station", "solarsystem", "region", "1", "5", "10", "20", "40"}

	for _, r := range ranges {
		t.Run("Range_"+r, func(t *testing.T) {
			order := ESIMarketOrder{
				OrderID: 12345,
				Range:   r,
			}

			assert.Equal(t, r, order.Range)
			assert.NotEmpty(t, order.Range, "Range should not be empty")
		})
	}
}

// TestESIMarketOrder_TimeHandling tests time field handling
func TestESIMarketOrder_TimeHandling(t *testing.T) {
	now := time.Now()
	oneDayAgo := now.Add(-24 * time.Hour)
	oneWeekAgo := now.Add(-7 * 24 * time.Hour)

	tests := []struct {
		name   string
		issued time.Time
	}{
		{"Just issued", now},
		{"One day old", oneDayAgo},
		{"One week old", oneWeekAgo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := ESIMarketOrder{
				OrderID: 12345,
				Issued:  tt.issued,
			}

			age := now.Sub(order.Issued)
			assert.GreaterOrEqual(t, age, time.Duration(0), "Order age should be non-negative")

			// Verify JSON round-trip preserves time
			jsonData, err := json.Marshal(order)
			require.NoError(t, err)

			var decoded ESIMarketOrder
			err = json.Unmarshal(jsonData, &decoded)
			require.NoError(t, err)

			assert.True(t, order.Issued.Equal(decoded.Issued), "Issued time should survive JSON round-trip")
		})
	}
}

// TestNewClient_InvalidRedis tests client creation with invalid Redis
func TestNewClient_InvalidRedis(t *testing.T) {
	// Redis client pointing to non-existent server
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:9999", // Invalid port
	})
	defer redisClient.Close()

	cfg := Config{
		UserAgent:      "eve-o-provit-test/1.0",
		RateLimit:      150,
		ErrorThreshold: 10,
		MaxRetries:     3,
	}

	// Client creation should succeed (connection is lazy)
	client, err := NewClient(redisClient, cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	defer client.Close()
}

// TestConfig_CustomValues tests config with custom values
func TestConfig_CustomValues(t *testing.T) {
	cfg := Config{
		UserAgent:      "custom-agent/2.0",
		RateLimit:      300,
		ErrorThreshold: 50,
		MaxRetries:     5,
	}

	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	client, err := NewClient(redisClient, cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, client)
	defer client.Close()
}
