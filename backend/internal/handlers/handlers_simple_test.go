// Package handlers - Unit tests for simple endpoint handlers
package handlers

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// TestVersion_Unit tests version endpoint
func TestVersion_Unit(t *testing.T) {
	app := fiber.New()
	handler := &Handler{}

	app.Get("/version", handler.Version)

	req := httptest.NewRequest("GET", "/version", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, "0.1.0", result["version"])
	assert.Equal(t, "eve-o-provit-api", result["service"])
}

// TestGetType_Success_Unit tests successful type lookup
func TestGetType_Success_Unit(t *testing.T) {
	app := fiber.New()

	// Mock SDEQuerier with extension
	mockSDE := &MockSDEQuerierExtended{}
	mockSDE.GetTypeInfoFunc = func(ctx context.Context, typeID int) (*database.TypeInfo, error) {
		assert.Equal(t, 34, typeID)
		return &database.TypeInfo{
			TypeID: 34,
			Name:   "Tritanium",
			Volume: 0.01,
		}, nil
	}

	handler := &Handler{sdeQuerier: mockSDE}
	app.Get("/type/:id", handler.GetType)

	req := httptest.NewRequest("GET", "/type/34", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var result database.TypeInfo
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, 34, result.TypeID)
	assert.Equal(t, "Tritanium", result.Name)
	assert.Equal(t, 0.01, result.Volume)
}

// TestGetType_InvalidID_Unit tests invalid type ID
func TestGetType_InvalidID_Unit(t *testing.T) {
	app := fiber.New()
	handler := &Handler{}

	app.Get("/type/:id", handler.GetType)

	req := httptest.NewRequest("GET", "/type/invalid", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, "invalid type ID", result["error"])
}

// TestGetType_NotFound_Unit tests type not found
func TestGetType_NotFound_Unit(t *testing.T) {
	app := fiber.New()

	mockSDE := &MockSDEQuerierExtended{}
	mockSDE.GetTypeInfoFunc = func(ctx context.Context, typeID int) (*database.TypeInfo, error) {
		return nil, errors.New("type not found")
	}

	handler := &Handler{sdeQuerier: mockSDE}
	app.Get("/type/:id", handler.GetType)

	req := httptest.NewRequest("GET", "/type/99999", nil)
	resp, err := app.Test(req)

	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var result map[string]interface{}
	err = parseJSON(resp.Body, &result)
	assert.NoError(t, err)
	assert.Equal(t, "type not found", result["error"])
}

// Extend MockSDEQuerier with GetTypeInfo function
type MockSDEQuerierExtended struct {
	MockSDEQuerier
	GetTypeInfoFunc func(ctx context.Context, typeID int) (*database.TypeInfo, error)
}

func (m *MockSDEQuerierExtended) GetTypeInfo(ctx context.Context, typeID int) (*database.TypeInfo, error) {
	if m.GetTypeInfoFunc != nil {
		return m.GetTypeInfoFunc(ctx, typeID)
	}
	return nil, nil
}
