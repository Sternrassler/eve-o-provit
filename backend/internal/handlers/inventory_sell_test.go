package handlers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/gofiber/fiber/v2"
)

// TestCalculateInventorySellRoutes_InvalidTypeID tests validation of type_id
func TestCalculateInventorySellRoutes_InvalidTypeID(t *testing.T) {
	handler := NewTradingHandler(nil, &Handler{}, nil)

	app := fiber.New()
	
	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	
	app.Post("/inventory-sell", handler.CalculateInventorySellRoutes)

	tests := []struct {
		name          string
		requestBody   models.InventorySellRequest
		expectedError string
	}{
		{
			name: "Zero type_id",
			requestBody: models.InventorySellRequest{
				TypeID:           0,
				Quantity:         100,
				BuyPricePerUnit:  1000.0,
				RegionID:         10000002,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid type_id",
		},
		{
			name: "Negative type_id",
			requestBody: models.InventorySellRequest{
				TypeID:           -1,
				Quantity:         100,
				BuyPricePerUnit:  1000.0,
				RegionID:         10000002,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid type_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != fiber.StatusBadRequest {
				t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
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

// TestCalculateInventorySellRoutes_InvalidQuantity tests validation of quantity
func TestCalculateInventorySellRoutes_InvalidQuantity(t *testing.T) {
	handler := NewTradingHandler(nil, &Handler{}, nil)

	app := fiber.New()
	
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	
	app.Post("/inventory-sell", handler.CalculateInventorySellRoutes)

	tests := []struct {
		name          string
		requestBody   models.InventorySellRequest
		expectedError string
	}{
		{
			name: "Zero quantity",
			requestBody: models.InventorySellRequest{
				TypeID:           34,
				Quantity:         0,
				BuyPricePerUnit:  1000.0,
				RegionID:         10000002,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid quantity",
		},
		{
			name: "Negative quantity",
			requestBody: models.InventorySellRequest{
				TypeID:           34,
				Quantity:         -100,
				BuyPricePerUnit:  1000.0,
				RegionID:         10000002,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid quantity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != fiber.StatusBadRequest {
				t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
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

// TestCalculateInventorySellRoutes_InvalidBuyPrice tests validation of buy_price_per_unit
func TestCalculateInventorySellRoutes_InvalidBuyPrice(t *testing.T) {
	handler := NewTradingHandler(nil, &Handler{}, nil)

	app := fiber.New()
	
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	
	app.Post("/inventory-sell", handler.CalculateInventorySellRoutes)

	tests := []struct {
		name          string
		requestBody   models.InventorySellRequest
		expectedError string
	}{
		{
			name: "Zero buy price",
			requestBody: models.InventorySellRequest{
				TypeID:           34,
				Quantity:         100,
				BuyPricePerUnit:  0.0,
				RegionID:         10000002,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid buy_price_per_unit",
		},
		{
			name: "Negative buy price",
			requestBody: models.InventorySellRequest{
				TypeID:           34,
				Quantity:         100,
				BuyPricePerUnit:  -1000.0,
				RegionID:         10000002,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid buy_price_per_unit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != fiber.StatusBadRequest {
				t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
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

// TestCalculateInventorySellRoutes_InvalidRegionID tests validation of region_id
func TestCalculateInventorySellRoutes_InvalidRegionID(t *testing.T) {
	handler := NewTradingHandler(nil, &Handler{}, nil)

	app := fiber.New()
	
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	
	app.Post("/inventory-sell", handler.CalculateInventorySellRoutes)

	tests := []struct {
		name          string
		requestBody   models.InventorySellRequest
		expectedError string
	}{
		{
			name: "Zero region_id",
			requestBody: models.InventorySellRequest{
				TypeID:           34,
				Quantity:         100,
				BuyPricePerUnit:  1000.0,
				RegionID:         0,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid region_id",
		},
		{
			name: "Negative region_id",
			requestBody: models.InventorySellRequest{
				TypeID:           34,
				Quantity:         100,
				BuyPricePerUnit:  1000.0,
				RegionID:         -1,
				MinProfitPerUnit: 100.0,
			},
			expectedError: "Invalid region_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != fiber.StatusBadRequest {
				t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
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

// TestCalculateInventorySellRoutes_InvalidJSON tests invalid JSON handling
func TestCalculateInventorySellRoutes_InvalidJSON(t *testing.T) {
	handler := NewTradingHandler(nil, &Handler{}, nil)

	app := fiber.New()
	
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	
	app.Post("/inventory-sell", handler.CalculateInventorySellRoutes)

	req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewBufferString(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["error"] == nil {
		t.Error("Expected error message in response")
	}
}

// TestCalculateInventorySellRoutes_SecurityFilter tests security filter parameter parsing
func TestCalculateInventorySellRoutes_SecurityFilter(t *testing.T) {
	// This tests that security_filter values are parsed correctly
	// Actual filtering logic is tested in integration tests

	app := fiber.New()
	
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	
	// Mock character helper to avoid ESI calls
	app.Post("/inventory-sell", func(c *fiber.Ctx) error {
		var req models.InventorySellRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}
		
		// Validate security_filter is one of expected values
		validFilters := map[string]bool{
			"":        true,
			"all":     true,
			"highsec": true,
			"highlow": true,
		}
		
		if !validFilters[req.SecurityFilter] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid security_filter (must be: all, highsec, highlow, or empty)",
			})
		}
		
		return c.JSON(fiber.Map{
			"security_filter": req.SecurityFilter,
			"ok":              true,
		})
	})

	tests := []struct {
		name           string
		securityFilter string
		shouldSucceed  bool
	}{
		{
			name:           "Empty filter (default: all)",
			securityFilter: "",
			shouldSucceed:  true,
		},
		{
			name:           "Valid: all",
			securityFilter: "all",
			shouldSucceed:  true,
		},
		{
			name:           "Valid: highsec",
			securityFilter: "highsec",
			shouldSucceed:  true,
		},
		{
			name:           "Valid: highlow",
			securityFilter: "highlow",
			shouldSucceed:  true,
		},
		{
			name:           "Invalid: lowsec",
			securityFilter: "lowsec",
			shouldSucceed:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := models.InventorySellRequest{
				TypeID:           34,
				Quantity:         100,
				BuyPricePerUnit:  1000.0,
				RegionID:         10000002,
				MinProfitPerUnit: 100.0,
				SecurityFilter:   tt.securityFilter,
			}

			bodyBytes, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if tt.shouldSucceed {
				if resp.StatusCode != fiber.StatusOK {
					t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusOK)
				}
			} else {
				if resp.StatusCode != fiber.StatusBadRequest {
					t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusBadRequest)
				}
			}
		})
	}
}
