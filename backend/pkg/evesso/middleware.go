package evesso

import (
	"github.com/gofiber/fiber/v2"
)

// NewAuthMiddleware returns a Fiber handler that requires a valid eve_access_token cookie,
// validates it locally as a JWT, and stores character info in locals. Returns 401 otherwise.
func NewAuthMiddleware(v *TokenValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessToken := c.Cookies("eve_access_token")
		if accessToken == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing authentication cookie"})
		}
		charInfo, err := v.Validate(c.Context(), accessToken)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}
		setAuthLocals(c, charInfo, accessToken)
		return c.Next()
	}
}

// NewOptionalAuthMiddleware validates the cookie if present but allows unauthenticated requests.
func NewOptionalAuthMiddleware(v *TokenValidator) fiber.Handler {
	return func(c *fiber.Ctx) error {
		accessToken := c.Cookies("eve_access_token")
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
