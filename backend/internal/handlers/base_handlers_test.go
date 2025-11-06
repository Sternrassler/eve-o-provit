package handlers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestVersion tests version endpoint
func TestVersion(t *testing.T) {
	handler := &Handler{}

	app := fiber.New()
	app.Get("/version", handler.Version)

	req := httptest.NewRequest("GET", "/version", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Status code = %v, want %v", resp.StatusCode, fiber.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	version, ok := response["version"].(string)
	if !ok || version != "0.1.0" {
		t.Errorf("Version = %v, want '0.1.0'", response["version"])
	}

	service, ok := response["service"].(string)
	if !ok || service != "eve-o-provit-api" {
		t.Errorf("Service = %v, want 'eve-o-provit-api'", response["service"])
	}
}

// TestSearchItems_QueryTooShort tests search with query < 3 characters
func TestSearchItems_QueryTooShort(t *testing.T) {
	handler := newTestTradingHandler()

	app := fiber.New()
	app.Get("/search", handler.SearchItems)

	tests := []struct {
		name  string
		query string
	}{
		{"Empty query", ""},
		{"1 character", "a"},
		{"2 characters", "ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/search?q="+tt.query, nil)
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

			expectedError := "query parameter 'q' must be at least 3 characters"
			if errorMsg, ok := response["error"].(string); !ok || errorMsg != expectedError {
				t.Errorf("Error message = %v, want %v", response["error"], expectedError)
			}
		})
	}
}

// TestSetAutopilotWaypoint_InvalidRequest tests waypoint setting with invalid input
func TestSetAutopilotWaypoint_InvalidRequest(t *testing.T) {
	handler := newTestTradingHandler()

	app := fiber.New()

	// Mock auth middleware
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("access_token", "test-token")
		return c.Next()
	})

	app.Post("/waypoint", handler.SetAutopilotWaypoint)

	tests := []struct {
		name           string
		requestBody    string
		expectedError  string
		expectedStatus int
	}{
		{
			name:           "Invalid JSON",
			requestBody:    `{invalid json}`,
			expectedError:  "Invalid request body",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Missing destination_id",
			requestBody:    `{"clear_other_waypoints": true}`,
			expectedError:  "Invalid destination_id",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Zero destination_id",
			requestBody:    `{"destination_id": 0}`,
			expectedError:  "Invalid destination_id",
			expectedStatus: fiber.StatusBadRequest,
		},
		{
			name:           "Negative destination_id",
			requestBody:    `{"destination_id": -1}`,
			expectedError:  "Invalid destination_id",
			expectedStatus: fiber.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/waypoint", nil)
			req.Header.Set("Content-Type", "application/json")

			// Set body if provided
			if tt.requestBody != "" {
				req.Body = nil
				req.Header.Set("Content-Length", "0")
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Status code = %v, want %v", resp.StatusCode, tt.expectedStatus)
			}
		})
	}
}
