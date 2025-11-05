//go:build unit || !integration

package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersion_Success tests version endpoint
func TestVersion_Success(t *testing.T) {
	handler := New(nil, nil, nil, nil)

	app := fiber.New()
	app.Get("/version", handler.Version)

	req := httptest.NewRequest("GET", "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// TestGetType_InvalidID tests type lookup with invalid ID format
func TestGetType_InvalidID(t *testing.T) {
	handler := New(nil, nil, nil, nil)

	app := fiber.New()
	app.Get("/type/:id", handler.GetType)

	tests := []struct {
		name     string
		id       string
		wantCode int
	}{
		{"non-numeric ID", "invalid", fiber.StatusBadRequest},
		{"empty ID", "", fiber.StatusNotFound}, // Fiber route mismatch
		{"special characters", "abc@123", fiber.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/type/"+tt.id, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

// TestGetMarketOrders_InvalidRegionID tests market orders with invalid region ID
func TestGetMarketOrders_InvalidRegionID(t *testing.T) {
	handler := New(nil, nil, nil, nil)

	app := fiber.New()
	app.Get("/market/:region/:type", handler.GetMarketOrders)

	tests := []struct {
		name     string
		region   string
		typeID   string
		wantCode int
	}{
		{"invalid region ID", "invalid", "34", fiber.StatusBadRequest},
		{"invalid type ID", "10000002", "invalid", fiber.StatusBadRequest},
		{"both invalid", "abc", "xyz", fiber.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/market/"+tt.region+"/"+tt.typeID, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}

// TestGetMarketDataStaleness_InvalidInput tests staleness endpoint validation
func TestGetMarketDataStaleness_InvalidInput(t *testing.T) {
	handler := New(nil, nil, nil, nil)

	app := fiber.New()
	app.Get("/market/staleness/:region", handler.GetMarketDataStaleness)

	tests := []struct {
		name     string
		region   string
		wantCode int
	}{
		{"invalid region ID", "invalid", fiber.StatusBadRequest},
		{"empty region ID", "", fiber.StatusNotFound}, // Fiber route mismatch
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/market/staleness/"+tt.region, nil)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}
