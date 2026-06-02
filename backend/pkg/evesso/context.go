package evesso

import "github.com/gofiber/fiber/v2"

// Request-locals keys under which NewAuthMiddleware stores the authenticated
// identity. They are the single source of truth shared by setAuthLocals (the
// writer) and AuthFromContext (the reader), so the key strings can never drift.
const (
	localKeyCharacterID = "character_id"
	localKeyAccessToken = "access_token"
)

// AuthFromContext returns the authenticated character ID and access token that
// NewAuthMiddleware placed in the request locals via setAuthLocals. ok is false
// unless both are present with the expected types and the token is non-empty.
//
// Behind NewAuthMiddleware ok is always true — the middleware rejects
// unauthenticated requests before the handler runs. The guard therefore only
// matters for handlers exercised without the middleware (e.g. in unit tests).
func AuthFromContext(c *fiber.Ctx) (characterID int, accessToken string, ok bool) {
	id, idOK := c.Locals(localKeyCharacterID).(int)
	token, tokenOK := c.Locals(localKeyAccessToken).(string)
	if !idOK || !tokenOK || token == "" {
		return 0, "", false
	}
	return id, token, true
}
