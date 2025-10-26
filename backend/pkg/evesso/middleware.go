package evesso

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates Bearer tokens and extracts character info
func AuthMiddleware(c *fiber.Ctx) error {
	// Extract Bearer token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	// Check Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Authorization header format",
		})
	}

	accessToken := parts[1]

	// Verify token with EVE ESI
	charInfo, err := VerifyToken(c.Context(), accessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	// Store character info in locals for use in handlers
	c.Locals("character_id", charInfo.CharacterID)
	c.Locals("character_name", charInfo.CharacterName)
	c.Locals("scopes", charInfo.Scopes)
	c.Locals("owner_hash", charInfo.CharacterOwnerHash)

	return c.Next()
}
