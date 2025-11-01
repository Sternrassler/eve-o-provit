package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

// TestCalculateRoutes_ValidationErrors tests request validation
func TestCalculateRoutes_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json}`,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Invalid request body",
		},
		{
			name:           "Missing region_id",
			requestBody:    `{"ship_type_id": 648}`,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Invalid region_id",
		},
		{
			name:           "Invalid region_id",
			requestBody:    `{"region_id": 0, "ship_type_id": 648}`,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Invalid region_id",
		},
		{
			name:           "Missing ship_type_id",
			requestBody:    `{"region_id": 10000002}`,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Invalid ship_type_id",
		},
		{
			name:           "Invalid ship_type_id",
			requestBody:    `{"region_id": 10000002, "ship_type_id": 0}`,
			expectedStatus: fiber.StatusBadRequest,
			expectedError:  "Invalid ship_type_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()

			// Create a mock handler (we don't need actual calculator for validation tests)
			handler := &TradingHandler{}
			app.Post("/test", handler.CalculateRoutes)

			req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Status code = %v, want %v", resp.StatusCode, tt.expectedStatus)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if errorMsg, ok := response["error"].(string); !ok || errorMsg != tt.expectedError {
				t.Errorf("Error message = %v, want %v", response["error"], tt.expectedError)
			}
		})
	}
}

// TestCalculateRoutes_ValidRequest tests valid request parsing
func TestCalculateRoutes_ValidRequest(t *testing.T) {
	requestBody := `{
		"region_id": 10000002,
		"ship_type_id": 648,
		"cargo_capacity": 15000
	}`

	app := fiber.New()

	var capturedRequest models.RouteCalculationRequest
	app.Post("/test", func(c *fiber.Ctx) error {
		if err := c.BodyParser(&capturedRequest); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	if capturedRequest.RegionID != 10000002 {
		t.Errorf("RegionID = %v, want 10000002", capturedRequest.RegionID)
	}

	if capturedRequest.ShipTypeID != 648 {
		t.Errorf("ShipTypeID = %v, want 648", capturedRequest.ShipTypeID)
	}

	if capturedRequest.CargoCapacity != 15000 {
		t.Errorf("CargoCapacity = %v, want 15000", capturedRequest.CargoCapacity)
	}
}

// TestCalculateRoutes_OptionalCargoCapacity tests that cargo_capacity is optional
func TestCalculateRoutes_OptionalCargoCapacity(t *testing.T) {
	requestBody := `{
		"region_id": 10000002,
		"ship_type_id": 648
	}`

	app := fiber.New()

	var capturedRequest models.RouteCalculationRequest
	app.Post("/test", func(c *fiber.Ctx) error {
		if err := c.BodyParser(&capturedRequest); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(requestBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	// CargoCapacity should default to 0 when not provided
	if capturedRequest.CargoCapacity != 0 {
		t.Errorf("CargoCapacity = %v, want 0 (default)", capturedRequest.CargoCapacity)
	}
}

// TestCharacterEndpoints_Authentication tests that character endpoints require auth
// Note: This is a placeholder test. Real authentication testing should be done
// in integration tests with the actual auth middleware
func TestCharacterEndpoints_Authentication(t *testing.T) {
	t.Skip("Authentication tests require integration test setup with middleware")
}

// TestResponseStructures tests that response structures are correct
func TestResponseStructures(t *testing.T) {
	t.Run("RouteCalculationResponse", func(t *testing.T) {
		response := models.RouteCalculationResponse{
			RegionID:          10000002,
			RegionName:        "The Forge",
			ShipTypeID:        648,
			ShipName:          "Badger",
			CargoCapacity:     15000,
			CalculationTimeMS: 1234,
			Routes: []models.TradingRoute{
				{
					ItemTypeID:    34,
					ItemName:      "Tritanium",
					BuyPrice:      5.0,
					SellPrice:     6.0,
					Quantity:      1000,
					ProfitPerUnit: 1.0,
					TotalProfit:   1000.0,
					SpreadPercent: 20.0,
					ISKPerHour:    100000.0,
					Jumps:         5,
				},
			},
		}

		// Marshal to JSON to verify structure
		jsonData, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}

		// Unmarshal back to verify all fields
		var decoded models.RouteCalculationResponse
		if err := json.Unmarshal(jsonData, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if decoded.RegionID != response.RegionID {
			t.Errorf("RegionID mismatch after JSON round-trip")
		}

		if len(decoded.Routes) != len(response.Routes) {
			t.Errorf("Routes count mismatch after JSON round-trip")
		}
	})
}
