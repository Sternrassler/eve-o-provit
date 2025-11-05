// Package handlers_test tests handler endpoints with mocks
package handlers_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/handlers"
	"github.com/Sternrassler/eve-o-provit/backend/internal/testutil"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth_Success(t *testing.T) {
	// Setup
	app := fiber.New()
	healthChecker := testutil.NewMockHealthChecker()
	sdeQuerier := testutil.NewMockSDEWithDefaults()
	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &esi.Client{} // Minimal ESI client (not used in Health)

	handler := handlers.New(healthChecker, sdeQuerier, marketQuerier, esiClient)
	app.Get("/health", handler.Health)

	// Execute
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"status":"ok"`)
	assert.Contains(t, string(body), `"service":"eve-o-provit-api"`)
}

func TestHealth_DatabaseUnhealthy(t *testing.T) {
	// Setup
	app := fiber.New()
	dbError := errors.New("database connection lost")
	healthChecker := testutil.NewMockHealthCheckerError(dbError)
	sdeQuerier := testutil.NewMockSDEWithDefaults()
	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &esi.Client{}

	handler := handlers.New(healthChecker, sdeQuerier, marketQuerier, esiClient)
	app.Get("/health", handler.Health)

	// Execute
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 503, resp.StatusCode) // Service Unavailable

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"status":"unhealthy"`)
	assert.Contains(t, string(body), "database connection lost")
}

func TestVersion_Success(t *testing.T) {
	// Setup
	app := fiber.New()
	healthChecker := testutil.NewMockHealthChecker()
	sdeQuerier := testutil.NewMockSDEWithDefaults()
	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &esi.Client{}

	handler := handlers.New(healthChecker, sdeQuerier, marketQuerier, esiClient)
	app.Get("/version", handler.Version)

	// Execute
	req := httptest.NewRequest("GET", "/version", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"version":"0.1.0"`)
	assert.Contains(t, string(body), `"service":"eve-o-provit-api"`)
}

func TestGetType_Success(t *testing.T) {
	// Setup
	app := fiber.New()
	healthChecker := testutil.NewMockHealthChecker()
	sdeQuerier := testutil.NewMockSDEWithDefaults()
	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &esi.Client{}

	handler := handlers.New(healthChecker, sdeQuerier, marketQuerier, esiClient)
	app.Get("/types/:id", handler.GetType)

	// Execute
	req := httptest.NewRequest("GET", "/types/34", nil) // Tritanium
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 200, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"type_id":34`)
	assert.Contains(t, string(body), `"name":"Test Type 34"`)
}

func TestGetType_InvalidID(t *testing.T) {
	// Setup
	app := fiber.New()
	healthChecker := testutil.NewMockHealthChecker()
	sdeQuerier := testutil.NewMockSDEWithDefaults()
	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &esi.Client{}

	handler := handlers.New(healthChecker, sdeQuerier, marketQuerier, esiClient)
	app.Get("/types/:id", handler.GetType)

	// Execute
	req := httptest.NewRequest("GET", "/types/invalid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 400, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), `"error":"invalid type ID"`)
}

func TestGetType_NotFound(t *testing.T) {
	// Setup
	app := fiber.New()
	healthChecker := testutil.NewMockHealthChecker()

	// Mock SDE to return "not found" error
	sdeQuerier := &testutil.MockSDEQuerier{
		GetTypeInfoFunc: func(ctx context.Context, typeID int) (*database.TypeInfo, error) {
			return nil, errors.New("type 99999 not found")
		},
	}

	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &esi.Client{}

	handler := handlers.New(healthChecker, sdeQuerier, marketQuerier, esiClient)
	app.Get("/types/:id", handler.GetType)

	// Execute
	req := httptest.NewRequest("GET", "/types/99999", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	// Assert
	assert.Equal(t, 404, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "type 99999 not found")
}
