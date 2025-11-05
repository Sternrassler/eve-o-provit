// Package handlers - Unit tests for CalculateRoutes handler
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// MockRouteCalculator implements services.RouteCalculatorServicer for testing
type MockRouteCalculator struct {
	CalculateFunc func(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error)
}

func (m *MockRouteCalculator) Calculate(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
	if m.CalculateFunc != nil {
		return m.CalculateFunc(ctx, regionID, shipTypeID, cargoCapacity)
	}
	panic("CalculateFunc not set")
}

// TestCalculateRoutes_Success_Unit tests successful route calculation
func TestCalculateRoutes_Success_Unit(t *testing.T) {
	app := fiber.New()

	// Mock RouteCalculator
	mockCalc := &MockRouteCalculator{
		CalculateFunc: func(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
			// Verify parameters
			assert.Equal(t, 10000002, regionID)
			assert.Equal(t, 648, shipTypeID)
			assert.Equal(t, 0.0, cargoCapacity) // Not provided in request

			return &models.RouteCalculationResponse{
				RegionID:          10000002,
				RegionName:        "The Forge",
				ShipTypeID:        648,
				ShipName:          "Badger",
				CargoCapacity:     15000.0,
				CalculationTimeMS: 1234,
				Routes: []models.TradingRoute{
					{
						ItemTypeID:    34,
						ItemName:      "Tritanium",
						ISKPerHour:    5000000.0,
						TotalProfit:   250000.0,
						SpreadPercent: 8.5,
					},
				},
			}, nil
		},
	}

	handler := &TradingHandler{
		calculator: mockCalc,
	}

	app.Post("/api/v1/trading/routes/calculate", handler.CalculateRoutes)

	// Create request
	reqBody := models.RouteCalculationRequest{
		RegionID:   10000002,
		ShipTypeID: 648,
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/v1/trading/routes/calculate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.RouteCalculationResponse
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)

	assert.Equal(t, 10000002, result.RegionID)
	assert.Equal(t, "The Forge", result.RegionName)
	assert.Equal(t, 648, result.ShipTypeID)
	assert.Equal(t, "Badger", result.ShipName)
	assert.Equal(t, 15000.0, result.CargoCapacity)
	assert.Equal(t, 1, len(result.Routes))
	assert.Equal(t, "Tritanium", result.Routes[0].ItemName)
}

// TestCalculateRoutes_WithCargoCapacity_Unit tests with custom cargo capacity
func TestCalculateRoutes_WithCargoCapacity_Unit(t *testing.T) {
	app := fiber.New()

	mockCalc := &MockRouteCalculator{
		CalculateFunc: func(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
			// Verify custom cargo capacity is passed
			assert.Equal(t, 20000.0, cargoCapacity)

			return &models.RouteCalculationResponse{
				RegionID:      regionID,
				ShipTypeID:    shipTypeID,
				CargoCapacity: cargoCapacity,
				Routes:        []models.TradingRoute{},
			}, nil
		},
	}

	handler := &TradingHandler{calculator: mockCalc}
	app.Post("/calculate", handler.CalculateRoutes)

	reqBody := models.RouteCalculationRequest{
		RegionID:      10000002,
		ShipTypeID:    648,
		CargoCapacity: 20000.0,
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/calculate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

// TestCalculateRoutes_InvalidRequestBody_Unit tests invalid JSON body
func TestCalculateRoutes_InvalidRequestBody_Unit(t *testing.T) {
	app := fiber.New()

	handler := &TradingHandler{
		calculator: &MockRouteCalculator{}, // Not called
	}

	app.Post("/calculate", handler.CalculateRoutes)

	// Send invalid JSON
	req := httptest.NewRequest("POST", "/calculate", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid request body", result["error"])
}

// TestCalculateRoutes_InvalidRegionID_Unit tests validation of region_id
func TestCalculateRoutes_InvalidRegionID_Unit(t *testing.T) {
	testCases := []struct {
		name     string
		regionID int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()

			handler := &TradingHandler{
				calculator: &MockRouteCalculator{}, // Not called
			}

			app.Post("/calculate", handler.CalculateRoutes)

			reqBody := models.RouteCalculationRequest{
				RegionID:   tc.regionID,
				ShipTypeID: 648,
			}
			bodyJSON, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/calculate", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode)

			var result map[string]interface{}
			err = parseJSON(resp.Body, &result)
			assert.NoError(t, err)
			assert.Equal(t, "Invalid region_id", result["error"])
		})
	}
}

// TestCalculateRoutes_InvalidShipTypeID_Unit tests validation of ship_type_id
func TestCalculateRoutes_InvalidShipTypeID_Unit(t *testing.T) {
	testCases := []struct {
		name       string
		shipTypeID int
	}{
		{"zero", 0},
		{"negative", -1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := fiber.New()

			handler := &TradingHandler{
				calculator: &MockRouteCalculator{}, // Not called
			}

			app.Post("/calculate", handler.CalculateRoutes)

			reqBody := models.RouteCalculationRequest{
				RegionID:   10000002,
				ShipTypeID: tc.shipTypeID,
			}
			bodyJSON, _ := json.Marshal(reqBody)

			req := httptest.NewRequest("POST", "/calculate", bytes.NewReader(bodyJSON))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)

			assert.NoError(t, err)
			assert.Equal(t, 400, resp.StatusCode)

			var result map[string]interface{}
			err = parseJSON(resp.Body, &result)
			assert.NoError(t, err)
			assert.Equal(t, "Invalid ship_type_id", result["error"])
		})
	}
}

// TestCalculateRoutes_CalculatorError_Unit tests calculator service error
func TestCalculateRoutes_CalculatorError_Unit(t *testing.T) {
	app := fiber.New()

	mockCalc := &MockRouteCalculator{
		CalculateFunc: func(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
			return nil, errors.New("failed to fetch market orders")
		},
	}

	handler := &TradingHandler{calculator: mockCalc}
	app.Post("/calculate", handler.CalculateRoutes)

	reqBody := models.RouteCalculationRequest{
		RegionID:   10000002,
		ShipTypeID: 648,
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/calculate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to calculate routes", result["error"])
	assert.Contains(t, result["details"], "failed to fetch market orders")
}

// TestCalculateRoutes_PartialResults_Unit tests timeout warning with partial results
func TestCalculateRoutes_PartialResults_Unit(t *testing.T) {
	app := fiber.New()

	mockCalc := &MockRouteCalculator{
		CalculateFunc: func(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
			return &models.RouteCalculationResponse{
				RegionID:          10000002,
				RegionName:        "The Forge",
				ShipTypeID:        648,
				ShipName:          "Badger",
				CargoCapacity:     15000.0,
				CalculationTimeMS: 30000,
				Routes: []models.TradingRoute{
					{ItemName: "Partial Route 1"},
					{ItemName: "Partial Route 2"},
				},
				Warning: "Calculation timeout after 30s, showing partial results",
			}, nil
		},
	}

	handler := &TradingHandler{calculator: mockCalc}
	app.Post("/calculate", handler.CalculateRoutes)

	reqBody := models.RouteCalculationRequest{
		RegionID:   10000002,
		ShipTypeID: 648,
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/calculate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 206, resp.StatusCode) // PartialContent

	// Verify Warning header is set
	assert.Contains(t, resp.Header.Get("Warning"), "Calculation timeout after 30s")

	var result models.RouteCalculationResponse
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(result.Routes))
	assert.NotEmpty(t, result.Warning)
}

// TestCalculateRoutes_EmptyRoutes_Unit tests successful calculation with no profitable routes
func TestCalculateRoutes_EmptyRoutes_Unit(t *testing.T) {
	app := fiber.New()

	mockCalc := &MockRouteCalculator{
		CalculateFunc: func(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
			return &models.RouteCalculationResponse{
				RegionID:          10000002,
				RegionName:        "The Forge",
				ShipTypeID:        648,
				ShipName:          "Badger",
				CargoCapacity:     15000.0,
				CalculationTimeMS: 500,
				Routes:            []models.TradingRoute{}, // No profitable routes
			}, nil
		},
	}

	handler := &TradingHandler{calculator: mockCalc}
	app.Post("/calculate", handler.CalculateRoutes)

	reqBody := models.RouteCalculationRequest{
		RegionID:   10000002,
		ShipTypeID: 648,
	}
	bodyJSON, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/calculate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result models.RouteCalculationResponse
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, 0, len(result.Routes))
}

// Compile-time check: Ensure MockRouteCalculator implements the interface
var _ services.RouteCalculatorServicer = (*MockRouteCalculator)(nil)
