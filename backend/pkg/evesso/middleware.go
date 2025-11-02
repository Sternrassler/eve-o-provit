package evesso

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthMiddleware validates Bearer tokens and extracts character info
func AuthMiddleware(c *fiber.Ctx) error {
	// Extract Bearer token from Authorization header
	authHeader := c.Get("Authorization")
	log.Printf("[AuthMiddleware] Path: %s, AuthHeader present: %v", c.Path(), authHeader != "")
	
	if authHeader == "" {
		log.Printf("[AuthMiddleware] Missing Authorization header for path: %s", c.Path())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing Authorization header",
		})
	}

	// Check Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		log.Printf("[AuthMiddleware] Invalid Authorization format for path: %s", c.Path())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid Authorization header format",
		})
	}

	accessToken := parts[1]

	// Verify token with EVE ESI
	charInfo, err := VerifyToken(c.Context(), accessToken)
	if err != nil {
		log.Printf("[AuthMiddleware] Token verification failed for path: %s, error: %v", c.Path(), err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	log.Printf("[AuthMiddleware] Token verified successfully for path: %s, character: %s", c.Path(), charInfo.CharacterName)

	// Store character info and access token in locals for use in handlers
	c.Locals("character_id", charInfo.CharacterID)
	c.Locals("character_name", charInfo.CharacterName)
	c.Locals("scopes", charInfo.Scopes)
	c.Locals("owner_hash", charInfo.CharacterOwnerHash)
	c.Locals("access_token", accessToken)

	return c.Next()
}
