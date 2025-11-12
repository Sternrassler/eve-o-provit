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

	// Store character info and access token in locals for use in handlers
	c.Locals("character_id", charInfo.CharacterID)
	c.Locals("character_name", charInfo.CharacterName)
	c.Locals("scopes", charInfo.Scopes)
	c.Locals("owner_hash", charInfo.CharacterOwnerHash)
	c.Locals("access_token", accessToken)

	return c.Next()
}

// OptionalAuthMiddleware validates Bearer tokens if present, but allows unauthenticated requests
// Sets character_id, character_name, scopes, owner_hash, access_token in locals if authenticated
func OptionalAuthMiddleware(c *fiber.Ctx) error {
	// Extract Bearer token from Authorization header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		log.Printf("DEBUG [OptionalAuth]: No Authorization header")
		// No auth provided - allow request to proceed without character context
		return c.Next()
	}

	// Check Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		log.Printf("DEBUG [OptionalAuth]: Invalid Authorization format (parts=%d)", len(parts))
		// Invalid format - ignore and proceed unauthenticated
		return c.Next()
	}

	accessToken := parts[1]
	log.Printf("DEBUG [OptionalAuth]: Found Bearer token (len=%d)", len(accessToken))

	// Verify token with EVE ESI
	charInfo, err := VerifyToken(c.Context(), accessToken)
	if err != nil {
		log.Printf("DEBUG [OptionalAuth]: Token verification failed: %v", err)
		// Invalid token - ignore and proceed unauthenticated
		return c.Next()
	}

	log.Printf("DEBUG [OptionalAuth]: Token verified, setting locals for character_id=%d", charInfo.CharacterID)
	// Store character info and access token in locals for use in handlers
	c.Locals("character_id", charInfo.CharacterID)
	c.Locals("character_name", charInfo.CharacterName)
	c.Locals("scopes", charInfo.Scopes)
	c.Locals("owner_hash", charInfo.CharacterOwnerHash)
	c.Locals("access_token", accessToken)

	return c.Next()
}
