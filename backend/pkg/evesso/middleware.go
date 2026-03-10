package evesso

import (
	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates tokens from the eve_access_token cookie and extracts character info
func AuthMiddleware(c *fiber.Ctx) error {
	accessToken := c.Cookies("eve_access_token")
	if accessToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing authentication cookie",
		})
	}

	// Verify token with EVE ESI
	charInfo, err := VerifyToken(c.Context(), accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Store character info and access token in locals for use in handlers
	c.Locals("character_id", charInfo.CharacterID)
	c.Locals("character_name", charInfo.CharacterName)
	c.Locals("scopes", charInfo.Scopes)
	c.Locals("owner_hash", charInfo.CharacterOwnerHash)
	c.Locals("access_token", accessToken)

	return c.Next()
}

// OptionalAuthMiddleware validates the eve_access_token cookie if present, but allows
// unauthenticated requests to proceed. Sets character_id, character_name, scopes,
// owner_hash, and access_token in locals if authenticated.
func OptionalAuthMiddleware(c *fiber.Ctx) error {
	accessToken := c.Cookies("eve_access_token")
	if accessToken == "" {
		return c.Next()
	}

	// Verify token with EVE ESI
	charInfo, err := VerifyToken(c.Context(), accessToken)
	if err != nil {
		// Invalid token - proceed unauthenticated
		return c.Next()
	}

	// Store character info and access token in locals for use in handlers
	c.Locals("character_id", charInfo.CharacterID)
	c.Locals("character_name", charInfo.CharacterName)
	c.Locals("scopes", charInfo.Scopes)
	c.Locals("owner_hash", charInfo.CharacterOwnerHash)
	c.Locals("access_token", accessToken)

	return c.Next()
}
