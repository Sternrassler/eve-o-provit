package evesso

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAuthMiddleware_MissingCookie tests missing eve_access_token cookie (SECURITY)
func TestAuthMiddleware_MissingCookie(t *testing.T) {
	v := NewTokenValidatorWithKeySet("test-client", jwk.NewSet())
	app := fiber.New()
	app.Use("/protected", NewAuthMiddleware(v))
	app.Get("/protected", func(c *fiber.Ctx) error { return c.SendString("ok") })
	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401", resp.StatusCode)
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
			"Scopes": "publicData esi-markets.read",
			"CharacterOwnerHash": "abc123"
		}`

		var charInfo CharacterInfo
		err := json.Unmarshal([]byte(jsonData), &charInfo)

		require.NoError(t, err)
		assert.Equal(t, 12345, charInfo.CharacterID)
		assert.Equal(t, "Test Character", charInfo.CharacterName)
		assert.Equal(t, "publicData esi-markets.read", charInfo.Scopes)
		assert.Equal(t, "abc123", charInfo.CharacterOwnerHash)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonData := `{invalid json}`

		var charInfo CharacterInfo
		err := json.Unmarshal([]byte(jsonData), &charInfo)

		assert.Error(t, err)
	})
}
