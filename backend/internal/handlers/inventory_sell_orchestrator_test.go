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
)

// MockInventorySellOrchestrator implements InventorySellOrchestrator for testing
type MockInventorySellOrchestrator struct {
	CalculateSellRoutesFunc func(ctx context.Context, req models.InventorySellRequest, characterID int, accessToken string) ([]models.InventorySellRoute, error)
}

func (m *MockInventorySellOrchestrator) CalculateSellRoutes(ctx context.Context, req models.InventorySellRequest, characterID int, accessToken string) ([]models.InventorySellRoute, error) {
	if m.CalculateSellRoutesFunc != nil {
		return m.CalculateSellRoutesFunc(ctx, req, characterID, accessToken)
	}
	return []models.InventorySellRoute{}, nil
}

// TestCalculateInventorySellRoutes_Success_WithOrchestrator tests successful route calculation via orchestrator
func TestCalculateInventorySellRoutes_Success_WithOrchestrator(t *testing.T) {
	mockOrch := &MockInventorySellOrchestrator{
		CalculateSellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, characterID int, accessToken string) ([]models.InventorySellRoute, error) {
			// Verify request passed to orchestrator
			if req.TypeID != 34 {
				t.Errorf("Expected TypeID 34, got %d", req.TypeID)
			}
			if characterID != 12345 {
				t.Errorf("Expected characterID 12345, got %d", characterID)
			}
			if accessToken != "test-token" {
				t.Errorf("Expected accessToken 'test-token', got '%s'", accessToken)
			}

			return []models.InventorySellRoute{
				{
					SellStationID:   60003760,
					SellStationName: "Jita IV - Moon 4",
					ProfitPerUnit:   150.0,
					TotalProfit:     15000.0,
				},
			}, nil
		},
	}

	tradingHandler := &TradingHandler{
		inventorySellOrchestrator: mockOrch,
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	app.Post("/inventory-sell", tradingHandler.CalculateInventorySellRoutes)

	reqBody := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	routes, ok := result["routes"].([]interface{})
	if !ok {
		t.Fatal("Expected routes array in response")
	}
	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}
}

// TestCalculateInventorySellRoutes_OrchestratorError tests orchestrator error handling
func TestCalculateInventorySellRoutes_OrchestratorError(t *testing.T) {
	mockOrch := &MockInventorySellOrchestrator{
		CalculateSellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, characterID int, accessToken string) ([]models.InventorySellRoute, error) {
			return nil, errors.New("ESI market orders unavailable")
		},
	}

	tradingHandler := &TradingHandler{
		inventorySellOrchestrator: mockOrch,
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	app.Post("/inventory-sell", tradingHandler.CalculateInventorySellRoutes)

	reqBody := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 500 {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["error"] != "Failed to calculate sell routes" {
		t.Errorf("Expected error 'Failed to calculate sell routes', got '%v'", result["error"])
	}
}

// TestCalculateInventorySellRoutes_BusinessError_NotDocked tests business error handling
func TestCalculateInventorySellRoutes_BusinessError_NotDocked(t *testing.T) {
	mockOrch := &MockInventorySellOrchestrator{
		CalculateSellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, characterID int, accessToken string) ([]models.InventorySellRoute, error) {
			return nil, &services.BusinessError{
				Code:    "CHARACTER_NOT_DOCKED",
				Message: "Character must be docked at a station to calculate sell routes",
				Status:  400,
			}
		},
	}

	tradingHandler := &TradingHandler{
		inventorySellOrchestrator: mockOrch,
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	app.Post("/inventory-sell", tradingHandler.CalculateInventorySellRoutes)

	reqBody := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["error"] != "Character must be docked at a station to calculate sell routes" {
		t.Errorf("Unexpected error: %v", result["error"])
	}
}

// TestCalculateInventorySellRoutes_ValidationError tests request validation
func TestCalculateInventorySellRoutes_ValidationError(t *testing.T) {
	mockOrch := &MockInventorySellOrchestrator{
		CalculateSellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, characterID int, accessToken string) ([]models.InventorySellRoute, error) {
			t.Fatal("Orchestrator should not be called for invalid requests")
			return nil, nil
		},
	}

	tradingHandler := &TradingHandler{
		inventorySellOrchestrator: mockOrch,
	}

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	app.Post("/inventory-sell", tradingHandler.CalculateInventorySellRoutes)

	// Invalid request: TypeID = 0
	reqBody := models.InventorySellRequest{
		TypeID:          0,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}

	if resp.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if result["error"] != "Invalid type_id" {
		t.Errorf("Expected error 'Invalid type_id', got '%v'", result["error"])
	}
}
