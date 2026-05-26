package evesso

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// bearerOrCookie returns the access token from the Authorization: Bearer header
// (mobile clients) or the eve_access_token cookie (web), in that order.
func bearerOrCookie(c *fiber.Ctx) string {
	if h := c.Get("Authorization"); len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return c.Cookies("eve_access_token")
}

// NewAuthMiddleware returns a Fiber handler that requires a valid eve_access_token cookie
// or Authorization: Bearer token, validates it locally as a JWT, and stores character info
// in locals. Returns 401 otherwise.
func NewAuthMiddleware(v *TokenValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessToken := bearerOrCookie(c)
		if accessToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or invalid credentials"})
		}
		charInfo, err := v.Validate(c.Context(), accessToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}
		setAuthLocals(c, charInfo, accessToken)
		return c.Next()
	}
}

// NewOptionalAuthMiddleware validates the Bearer token or cookie if present but allows unauthenticated requests.
func NewOptionalAuthMiddleware(v *TokenValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessToken := bearerOrCookie(c)
		if accessToken == "" {
			return c.Next()
		}
		charInfo, err := v.Validate(c.Context(), accessToken)
		if err != nil {
			return c.Next()
		}
		setAuthLocals(c, charInfo, accessToken)
		return c.Next()
	}
}

func setAuthLocals(c *fiber.Ctx, charInfo *CharacterInfo, accessToken string) {
	c.Locals("character_id", charInfo.CharacterID)
	c.Locals("character_name", charInfo.CharacterName)
	c.Locals("scopes", charInfo.Scopes)
	c.Locals("owner_hash", charInfo.CharacterOwnerHash)
	c.Locals("access_token", accessToken)
}
