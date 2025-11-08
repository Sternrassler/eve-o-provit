package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// mockFittingService for testing FittingHandler
type mockFittingService struct {
	fitting *services.FittingData
	err     error
}

func (m *mockFittingService) GetCharacterFitting(ctx context.Context, characterID int, shipTypeID int, accessToken string) (*services.FittingData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.fitting, nil
}

// TestGetCharacterFitting_Success tests successful fitting retrieval
func TestGetCharacterFitting_Success(t *testing.T) {
	mockService := &mockFittingService{
		fitting: &services.FittingData{
			ShipTypeID: 20183,
			FittedModules: []services.FittedModule{
				{
					TypeID:   1319, // Expanded Cargohold II
					TypeName: "Expanded Cargohold II",
					Slot:     "LoSlot0",
					DogmaAttribs: map[int]float64{
						38: 2500.0, // Cargo capacity bonus
					},
				},
			},
			Bonuses: services.FittingBonuses{
				CargoBonus:          2500.0,
				WarpSpeedMultiplier: 1.0,
				InertiaModifier:     1.0,
			},
			Cached: true,
		},
	}

	handler := NewFittingHandler(mockService)

	app := fiber.New()
	app.Get("/characters/:characterId/fitting/:shipTypeId", func(c *fiber.Ctx) error {
		// Simulate auth middleware
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return handler.GetCharacterFitting(c)
	})

	req := httptest.NewRequest("GET", "/characters/12345/fitting/20183", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if result["character_id"].(float64) != 12345 {
		t.Errorf("Expected character_id 12345, got %v", result["character_id"])
	}

	if result["ship_type_id"].(float64) != 20183 {
		t.Errorf("Expected ship_type_id 20183, got %v", result["ship_type_id"])
	}

	bonuses := result["bonuses"].(map[string]interface{})
	if bonuses["cargo_bonus_m3"].(float64) != 2500.0 {
		t.Errorf("Expected cargo bonus 2500, got %v", bonuses["cargo_bonus_m3"])
	}
}

// TestGetCharacterFitting_InvalidCharacterID tests invalid character ID
func TestGetCharacterFitting_InvalidCharacterID(t *testing.T) {
	handler := NewFittingHandler(&mockFittingService{})

	app := fiber.New()
	app.Get("/characters/:characterId/fitting/:shipTypeId", handler.GetCharacterFitting)

	req := httptest.NewRequest("GET", "/characters/invalid/fitting/20183", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// TestGetCharacterFitting_InvalidShipTypeID tests invalid ship type ID
func TestGetCharacterFitting_InvalidShipTypeID(t *testing.T) {
	handler := NewFittingHandler(&mockFittingService{})

	app := fiber.New()
	app.Get("/characters/:characterId/fitting/:shipTypeId", handler.GetCharacterFitting)

	req := httptest.NewRequest("GET", "/characters/12345/fitting/invalid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

// TestGetCharacterFitting_MissingToken tests missing access token
func TestGetCharacterFitting_MissingToken(t *testing.T) {
	handler := NewFittingHandler(&mockFittingService{})

	app := fiber.New()
	app.Get("/characters/:characterId/fitting/:shipTypeId", handler.GetCharacterFitting)

	req := httptest.NewRequest("GET", "/characters/12345/fitting/20183", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

// TestGetCharacterFitting_Forbidden tests character ID mismatch
func TestGetCharacterFitting_Forbidden(t *testing.T) {
	handler := NewFittingHandler(&mockFittingService{})

	app := fiber.New()
	app.Get("/characters/:characterId/fitting/:shipTypeId", func(c *fiber.Ctx) error {
		// Simulate auth middleware with different character
		c.Locals("character_id", 99999)
		c.Locals("access_token", "test-token")
		return handler.GetCharacterFitting(c)
	})

	req := httptest.NewRequest("GET", "/characters/12345/fitting/20183", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

// TestGetCharacterFitting_ServiceError tests fitting service error
func TestGetCharacterFitting_ServiceError(t *testing.T) {
	mockService := &mockFittingService{
		err: errors.New("ESI unavailable"),
	}

	handler := NewFittingHandler(mockService)

	app := fiber.New()
	app.Get("/characters/:characterId/fitting/:shipTypeId", func(c *fiber.Ctx) error {
		// Simulate auth middleware
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return handler.GetCharacterFitting(c)
	})

	req := httptest.NewRequest("GET", "/characters/12345/fitting/20183", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}
