// Package handlers - Character handler tests
package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSkillsService implements services.SkillsServicer for testing
type mockSkillsService struct {
	skills *services.TradingSkills
	err    error
}

func (m *mockSkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*services.TradingSkills, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.skills, nil
}

func TestCharacterHandler_GetCharacterSkills_Success(t *testing.T) {
	// Setup mock service
	mockService := &mockSkillsService{
		skills: &services.TradingSkills{
			Accounting:      5,
			BrokerRelations: 4,
			Navigation:      5,
		},
	}

	// Create handler
	handler := NewCharacterHandler(mockService)

	// Create Fiber app
	app := fiber.New()
	app.Get("/api/v1/characters/:characterId/skills", handler.GetCharacterSkills)

	// Create request with auth locals
	req := httptest.NewRequest("GET", "/api/v1/characters/12345/skills", nil)
	req.Header.Set("Content-Type", "application/json")

	// Execute request with locals
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Manually set locals (simulating AuthMiddleware)
	// Note: Fiber's Test() doesn't support locals, so we need to test differently
	// Let's use a middleware to set locals
	app2 := fiber.New()
	app2.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	app2.Get("/api/v1/characters/:characterId/skills", handler.GetCharacterSkills)

	req2 := httptest.NewRequest("GET", "/api/v1/characters/12345/skills", nil)
	resp, err = app2.Test(req2, -1)
	require.NoError(t, err)

	// Verify response
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, float64(12345), result["character_id"])
	skills := result["skills"].(map[string]interface{})
	assert.Equal(t, float64(5), skills["Accounting"])
	assert.Equal(t, float64(4), skills["BrokerRelations"])
	assert.Equal(t, float64(5), skills["Navigation"])
}

func TestCharacterHandler_GetCharacterSkills_InvalidCharacterID(t *testing.T) {
	// Setup mock service
	mockService := &mockSkillsService{}

	// Create handler
	handler := NewCharacterHandler(mockService)

	// Create Fiber app
	app := fiber.New()
	app.Get("/api/v1/characters/:characterId/skills", handler.GetCharacterSkills)

	// Create request with invalid character ID
	req := httptest.NewRequest("GET", "/api/v1/characters/invalid/skills", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Verify error response
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "Invalid character_id", result["error"])
}

func TestCharacterHandler_GetCharacterSkills_MissingToken(t *testing.T) {
	// Setup mock service
	mockService := &mockSkillsService{}

	// Create handler
	handler := NewCharacterHandler(mockService)

	// Create Fiber app with middleware that sets character_id but NOT access_token
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		// access_token intentionally NOT set
		return c.Next()
	})
	app.Get("/api/v1/characters/:characterId/skills", handler.GetCharacterSkills)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/characters/12345/skills", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Verify unauthorized response
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "Missing access token", result["error"])
}

func TestCharacterHandler_GetCharacterSkills_WrongCharacter(t *testing.T) {
	// Setup mock service
	mockService := &mockSkillsService{}

	// Create handler
	handler := NewCharacterHandler(mockService)

	// Create Fiber app with middleware that sets authenticated character as 11111
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 11111) // Authenticated as 11111
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	app.Get("/api/v1/characters/:characterId/skills", handler.GetCharacterSkills)

	// Try to access skills for character 12345 (not the authenticated character)
	req := httptest.NewRequest("GET", "/api/v1/characters/12345/skills", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Verify forbidden response
	assert.Equal(t, fiber.StatusForbidden, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "Cannot access skills for other characters", result["error"])
}

func TestCharacterHandler_GetCharacterSkills_ServiceError(t *testing.T) {
	// Setup mock service that returns error
	mockService := &mockSkillsService{
		err: assert.AnError,
	}

	// Create handler
	handler := NewCharacterHandler(mockService)

	// Create Fiber app
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("character_id", 12345)
		c.Locals("access_token", "test-token")
		return c.Next()
	})
	app.Get("/api/v1/characters/:characterId/skills", handler.GetCharacterSkills)

	// Create request
	req := httptest.NewRequest("GET", "/api/v1/characters/12345/skills", nil)
	resp, err := app.Test(req, -1)
	require.NoError(t, err)

	// Verify error response
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "Failed to fetch character skills", result["error"])
	assert.NotNil(t, result["details"])
}
