// Package handlers - Unit tests for GetMarketDataStaleness handler
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// MockPostgresQuerier implements database.PostgresQuerier for testing
type MockPostgresQuerier struct {
	QueryRowFunc func(ctx context.Context, query string, args ...interface{}) database.Row
}

func (m *MockPostgresQuerier) QueryRow(ctx context.Context, query string, args ...interface{}) database.Row {
	if m.QueryRowFunc != nil {
		return m.QueryRowFunc(ctx, query, args...)
	}
	panic("QueryRowFunc not set")
}

// MockRow implements database.Row for testing
type MockRow struct {
	ScanFunc func(dest ...interface{}) error
}

func (m *MockRow) Scan(dest ...interface{}) error {
	if m.ScanFunc != nil {
		return m.ScanFunc(dest...)
	}
	panic("ScanFunc not set")
}

// TestGetMarketDataStaleness_Success_Unit tests successful staleness query
func TestGetMarketDataStaleness_Success_Unit(t *testing.T) {
	app := fiber.New()

	// Mock PostgresQuerier
	mockPG := &MockPostgresQuerier{
		QueryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			// Verify region ID passed correctly
			assert.Equal(t, 10000002, args[0])

			// Return mock row with data
			return &MockRow{
				ScanFunc: func(dest ...interface{}) error {
					// Simulate query result: 1500 orders, fetched 5 minutes ago
					*dest[0].(*int) = 1500 // total_orders

					// latest_fetch is *time.Time (nullable), so dest[1] is **time.Time
					fetchTime := time.Now().Add(-5 * time.Minute)
					*dest[1].(**time.Time) = &fetchTime

					// age_minutes is *float64 (nullable), so dest[2] is **float64
					ageMin := 5.0
					*dest[2].(**float64) = &ageMin
					return nil
				},
			}
		},
	}

	handler := &Handler{
		postgresQuery: mockPG,
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	// Test request
	req := httptest.NewRequest("GET", "/api/v1/market/10000002/staleness", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)

	assert.Equal(t, float64(10000002), result["region_id"])
	assert.Equal(t, float64(1500), result["total_orders"])
	assert.NotEmpty(t, result["latest_fetch"])
	assert.Equal(t, 5.0, result["age_minutes"])
}

// TestGetMarketDataStaleness_MissingRegion_Unit tests missing region parameter
func TestGetMarketDataStaleness_MissingRegion_Unit(t *testing.T) {
	app := fiber.New()

	handler := &Handler{
		postgresQuery: &MockPostgresQuerier{}, // Not called
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	// Test with empty region (should not match route, but test handler logic)
	// Create request without region parameter
	req := httptest.NewRequest("GET", "/api/v1/market//staleness", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	// Fiber returns 404 for invalid route pattern
	assert.Equal(t, 404, resp.StatusCode)
}

// TestGetMarketDataStaleness_InvalidRegion_Unit tests invalid region ID format
func TestGetMarketDataStaleness_InvalidRegion_Unit(t *testing.T) {
	app := fiber.New()

	handler := &Handler{
		postgresQuery: &MockPostgresQuerier{}, // Not called
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	// Test with non-numeric region
	req := httptest.NewRequest("GET", "/api/v1/market/invalid/staleness", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Contains(t, result["error"], "invalid region ID")
	assert.NotEmpty(t, result["details"]) // strconv.Atoi error details
}

// TestGetMarketDataStaleness_QueryError_Unit tests database query error
func TestGetMarketDataStaleness_QueryError_Unit(t *testing.T) {
	app := fiber.New()

	// Mock PostgresQuerier with error
	mockPG := &MockPostgresQuerier{
		QueryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &MockRow{
				ScanFunc: func(dest ...interface{}) error {
					return errors.New("connection timeout")
				},
			}
		},
	}

	handler := &Handler{
		postgresQuery: mockPG,
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	req := httptest.NewRequest("GET", "/api/v1/market/10000002/staleness", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, "failed to query market data age", result["error"])
}

// TestGetMarketDataStaleness_NoOrders_Unit tests region with no market orders
func TestGetMarketDataStaleness_NoOrders_Unit(t *testing.T) {
	app := fiber.New()

	// Mock PostgresQuerier returning zero orders
	mockPG := &MockPostgresQuerier{
		QueryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &MockRow{
				ScanFunc: func(dest ...interface{}) error {
					// No orders: COUNT=0, MAX(fetched_at)=NULL causes error in real DB
					// Simulate sql: Scan error on NULL for time.Time
					return errors.New("sql: Scan error on column index 1, name \"latest_fetch\": unsupported Scan, storing driver.Value type <nil> into type *time.Time")
				},
			}
		},
	}

	handler := &Handler{
		postgresQuery: mockPG,
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	req := httptest.NewRequest("GET", "/api/v1/market/99999/staleness", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, "failed to query market data age", result["error"])
}

// TestGetMarketDataStaleness_NegativeRegion_Unit tests negative region ID
func TestGetMarketDataStaleness_NegativeRegion_Unit(t *testing.T) {
	app := fiber.New()

	mockPG := &MockPostgresQuerier{
		QueryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			// Verify negative region ID is passed (handler doesn't validate > 0)
			assert.Equal(t, -1, args[0])

			return &MockRow{
				ScanFunc: func(dest ...interface{}) error {
					// Simulate no results for invalid region
					*dest[0].(*int) = 0
					zeroTime := time.Time{}
					*dest[1].(**time.Time) = &zeroTime
					zeroAge := 0.0
					*dest[2].(**float64) = &zeroAge
					return nil
				},
			}
		},
	}

	handler := &Handler{
		postgresQuery: mockPG,
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	req := httptest.NewRequest("GET", "/api/v1/market/-1/staleness", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	// Handler doesn't validate regionID > 0, returns 200 with zero values
	assert.Equal(t, 200, resp.StatusCode)
}

// TestGetMarketDataStaleness_LargeRegionID_Unit tests very large region ID
func TestGetMarketDataStaleness_LargeRegionID_Unit(t *testing.T) {
	app := fiber.New()

	mockPG := &MockPostgresQuerier{
		QueryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &MockRow{
				ScanFunc: func(dest ...interface{}) error {
					*dest[0].(*int) = 0
					nowTime := time.Now()
					*dest[1].(**time.Time) = &nowTime
					zeroAge := 0.0
					*dest[2].(**float64) = &zeroAge
					return nil
				},
			}
		},
	}

	handler := &Handler{
		postgresQuery: mockPG,
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	// Test with max int value
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/market/%d/staleness", 2147483647), nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestGetMarketDataStaleness_VeryOldData_Unit tests very old market data
func TestGetMarketDataStaleness_VeryOldData_Unit(t *testing.T) {
	app := fiber.New()

	mockPG := &MockPostgresQuerier{
		QueryRowFunc: func(ctx context.Context, query string, args ...interface{}) database.Row {
			return &MockRow{
				ScanFunc: func(dest ...interface{}) error {
					// Simulate data from 7 days ago
					*dest[0].(*int) = 500
					oldTime := time.Now().Add(-7 * 24 * time.Hour)
					*dest[1].(**time.Time) = &oldTime
					oldAge := 10080.0 // 7 days in minutes
					*dest[2].(**float64) = &oldAge
					return nil
				},
			}
		},
	}

	handler := &Handler{
		postgresQuery: mockPG,
	}

	app.Get("/api/v1/market/:region/staleness", handler.GetMarketDataStaleness)

	req := httptest.NewRequest("GET", "/api/v1/market/10000002/staleness", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, 10080.0, result["age_minutes"])
}
