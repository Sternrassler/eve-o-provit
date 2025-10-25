package evesso

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
)

const (
	// Cookie names
	SessionCookieName = "eve_session"
	StateCookieName   = "oauth_state"

	// Cookie settings
	SessionCookieMaxAge = 24 * 60 * 60 // 24 hours in seconds
	StateCookieMaxAge   = 5 * 60       // 5 minutes in seconds
)

// Handler manages HTTP endpoints for EVE SSO authentication
type Handler struct {
	client         *Client
	sessionManager *SessionManager
}

// NewHandler creates a new authentication handler
func NewHandler(client *Client, sessionManager *SessionManager) *Handler {
	return &Handler{
		client:         client,
		sessionManager: sessionManager,
	}
}

// HandleLogin initiates the OAuth2 flow by redirecting to EVE SSO
func (h *Handler) HandleLogin(c *fiber.Ctx) error {
	state, err := GenerateState()
	if err != nil {
		log.Printf("Failed to generate state: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate state parameter",
		})
	}

	// Store state in secure cookie for validation
	c.Cookie(&fiber.Cookie{
		Name:     StateCookieName,
		Value:    state,
		MaxAge:   StateCookieMaxAge,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Lax",
	})

	authURL := h.client.GetAuthURL(state)
	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

// HandleCallback processes the OAuth2 callback from EVE SSO
func (h *Handler) HandleCallback(c *fiber.Ctx) error {
	// Validate state parameter (CSRF protection)
	state := c.Query("state")
	storedState := c.Cookies(StateCookieName)

	if state == "" || storedState == "" || state != storedState {
		log.Printf("State mismatch: query=%s, cookie=%s", state, storedState)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid state parameter",
		})
	}

	// Clear state cookie
	c.ClearCookie(StateCookieName)

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		log.Printf("Missing authorization code")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing authorization code",
		})
	}

	// Exchange code for access token
	token, err := h.client.ExchangeCode(c.Context(), code)
	if err != nil {
		log.Printf("Failed to exchange code: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to exchange authorization code",
		})
	}

	// Verify character and get character info
	charInfo, err := h.client.VerifyCharacter(c.Context(), token.AccessToken)
	if err != nil {
		log.Printf("Failed to verify character: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to verify character",
		})
	}

	// Create session token
	sessionToken, err := h.sessionManager.CreateToken(charInfo)
	if err != nil {
		log.Printf("Failed to create session token: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create session",
		})
	}

	// Set session cookie
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    sessionToken,
		MaxAge:   SessionCookieMaxAge,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
		Path:     "/",
	})

	// Redirect to frontend (assuming frontend is on different port in dev)
	// In production, this would redirect to the same domain
	frontendURL := c.Query("redirect_uri", "http://localhost:3000")
	return c.Redirect(frontendURL, fiber.StatusTemporaryRedirect)
}

// HandleLogout invalidates the session by clearing the cookie
func (h *Handler) HandleLogout(c *fiber.Ctx) error {
	// Clear session cookie
	c.ClearCookie(SessionCookieName)

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// HandleVerify verifies the current session and returns session info
func (h *Handler) HandleVerify(c *fiber.Ctx) error {
	sessionToken := c.Cookies(SessionCookieName)
	if sessionToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No session found",
		})
	}

	claims, err := h.sessionManager.ValidateToken(sessionToken)
	if err != nil {
		log.Printf("Failed to validate session: %v", err)
		c.ClearCookie(SessionCookieName)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid session",
		})
	}

	return c.JSON(fiber.Map{
		"authenticated":  true,
		"character_id":   claims.CharacterID,
		"character_name": claims.CharacterName,
		"scopes":         claims.Scopes,
		"expires_at":     claims.ExpiresAt.Time.Format(time.RFC3339),
	})
}

// HandleRefresh refreshes the session token
func (h *Handler) HandleRefresh(c *fiber.Ctx) error {
	sessionToken := c.Cookies(SessionCookieName)
	if sessionToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No session found",
		})
	}

	newToken, err := h.sessionManager.RefreshToken(sessionToken)
	if err != nil {
		log.Printf("Failed to refresh session: %v", err)
		c.ClearCookie(SessionCookieName)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Failed to refresh session",
		})
	}

	// Set new session cookie
	c.Cookie(&fiber.Cookie{
		Name:     SessionCookieName,
		Value:    newToken,
		MaxAge:   SessionCookieMaxAge,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
		Path:     "/",
	})

	return c.JSON(fiber.Map{
		"message": "Session refreshed successfully",
	})
}

// HandleCharacter returns current character information
func (h *Handler) HandleCharacter(c *fiber.Ctx) error {
	sessionToken := c.Cookies(SessionCookieName)
	if sessionToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No session found",
		})
	}

	claims, err := h.sessionManager.ValidateToken(sessionToken)
	if err != nil {
		log.Printf("Failed to validate session: %v", err)
		c.ClearCookie(SessionCookieName)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid session",
		})
	}

	return c.JSON(fiber.Map{
		"character_id":   claims.CharacterID,
		"character_name": claims.CharacterName,
		"scopes":         claims.Scopes,
		"portrait_url":   getPortraitURL(claims.CharacterID, 128),
	})
}

// getPortraitURL returns the character portrait URL
func getPortraitURL(characterID int, size int) string {
	return fmt.Sprintf("https://images.evetech.net/characters/%d/portrait?size=%d", characterID, size)
}

// AuthMiddleware is a Fiber middleware that requires authentication
func (h *Handler) AuthMiddleware(c *fiber.Ctx) error {
	sessionToken := c.Cookies(SessionCookieName)
	if sessionToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	claims, err := h.sessionManager.ValidateToken(sessionToken)
	if err != nil {
		c.ClearCookie(SessionCookieName)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid or expired session",
		})
	}

	// Store claims in locals for use in handlers
	c.Locals("character_id", claims.CharacterID)
	c.Locals("character_name", claims.CharacterName)
	c.Locals("scopes", claims.Scopes)

	return c.Next()
}
