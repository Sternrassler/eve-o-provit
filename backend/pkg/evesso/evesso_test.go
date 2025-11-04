package evesso

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyToken_EmptyToken tests empty token handling (SECURITY)
func TestVerifyToken_EmptyToken(t *testing.T) {
	ctx := context.Background()

	charInfo, err := VerifyToken(ctx, "")

	assert.Error(t, err)
	assert.Nil(t, charInfo)
	assert.Contains(t, err.Error(), "access token is empty")
}

// TestAuthMiddleware_MissingHeader tests missing Authorization header (SECURITY)
func TestAuthMiddleware_MissingHeader(t *testing.T) {
	app := fiber.New()

	app.Use("/protected", AuthMiddleware)
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Success")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, "Missing Authorization header", response["error"])
}

// TestAuthMiddleware_InvalidHeaderFormat tests invalid header format (SECURITY)
func TestAuthMiddleware_InvalidHeaderFormat(t *testing.T) {
	app := fiber.New()

	app.Use("/protected", AuthMiddleware)
	app.Get("/protected", func(c *fiber.Ctx) error {
		return c.SendString("Success")
	})

	tests := []struct {
		name        string
		authHeader  string
		expectError string
	}{
		{
			name:        "missing Bearer prefix",
			authHeader:  "InvalidToken123",
			expectError: "Invalid Authorization header format",
		},
		{
			name:        "wrong prefix",
			authHeader:  "Basic dGVzdDp0ZXN0",
			expectError: "Invalid Authorization header format",
		},
		{
			name:        "empty token",
			authHeader:  "Bearer ",
			expectError: "Invalid or expired token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tt.authHeader)

			resp, err := app.Test(req)
			require.NoError(t, err)
			assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
		})
	}
}

// TestGetPortraitURL tests character portrait URL generation
func TestGetPortraitURL(t *testing.T) {
	tests := []struct {
		name        string
		characterID int
		size        int
		expectedURL string
	}{
		{
			name:        "128px portrait",
			characterID: 12345,
			size:        128,
			expectedURL: "https://images.evetech.net/characters/12345/portrait?size=128",
		},
		{
			name:        "256px portrait",
			characterID: 67890,
			size:        256,
			expectedURL: "https://images.evetech.net/characters/67890/portrait?size=256",
		},
		{
			name:        "512px portrait",
			characterID: 11111,
			size:        512,
			expectedURL: "https://images.evetech.net/characters/11111/portrait?size=512",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := GetPortraitURL(tt.characterID, tt.size)
			assert.Equal(t, tt.expectedURL, url)
		})
	}
}

// TestCharacterInfo_Unmarshal tests CharacterInfo JSON unmarshaling
func TestCharacterInfo_Unmarshal(t *testing.T) {
	t.Run("valid character info", func(t *testing.T) {
		jsonData := `{
			"CharacterID": 12345,
			"CharacterName": "Test Character",
			"ExpiresOn": "2025-12-31T23:59:59",
			"Scopes": "publicData esi-markets.read",
			"TokenType": "Character",
			"CharacterOwnerHash": "abc123",
			"IntellectualProperty": "EVE"
		}`

		var charInfo CharacterInfo
		err := json.Unmarshal([]byte(jsonData), &charInfo)

		require.NoError(t, err)
		assert.Equal(t, 12345, charInfo.CharacterID)
		assert.Equal(t, "Test Character", charInfo.CharacterName)
		assert.Equal(t, "2025-12-31T23:59:59", charInfo.ExpiresOn)
		assert.Equal(t, "publicData esi-markets.read", charInfo.Scopes)
		assert.Equal(t, "Character", charInfo.TokenType)
		assert.Equal(t, "abc123", charInfo.CharacterOwnerHash)
		assert.Equal(t, "EVE", charInfo.IntellectualProperty)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonData := `{invalid json}`

		var charInfo CharacterInfo
		err := json.Unmarshal([]byte(jsonData), &charInfo)

		assert.Error(t, err)
	})
}

// TestTokenSecurity_NoLeakage tests that tokens aren't leaked in error messages (SECURITY)
func TestTokenSecurity_NoLeakage(t *testing.T) {
	ctx := context.Background()
	token := "super-secret-token-12345"

	_, err := VerifyToken(ctx, token)

	// Error should not contain the token
	if err != nil {
		assert.NotContains(t, err.Error(), token, "Token leaked in error message")
	}
}

// TestContextCancellation tests context cancellation during verify
func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	charInfo, err := VerifyToken(ctx, "test-token")

	assert.Error(t, err)
	assert.Nil(t, charInfo)
	// Error should be related to context cancellation
}
