package evesso

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// AuthHandler handles EVE SSO authentication routes (public client / PKCE flow)
type AuthHandler struct {
	clientID    string
	redirectURI string
	validator   *TokenValidator
}

// NewAuthHandler creates a new AuthHandler with the given EVE SSO credentials
func NewAuthHandler(clientID, redirectURI string, validator *TokenValidator) *AuthHandler {
	return &AuthHandler{
		clientID:    clientID,
		redirectURI: redirectURI,
		validator:   validator,
	}
}

// callbackRequest is the body the SPA POSTs to /auth/callback.
// State is received but intentionally NOT validated server-side: per EVE's redirect-based
// PKCE flow the callback recipient (the SPA) generates and validates `state` for CSRF.
// Server-side protection for this public client is PKCE (code_verifier ↔ code_challenge,
// validated by EVE during token exchange). See pkg/evesso/README.md.
type callbackRequest struct {
	Code         string `json:"code"`
	State        string `json:"state"` // frontend-validated (CSRF); not checked here by design
	CodeVerifier string `json:"code_verifier"`
}

type characterResponse struct {
	CharacterID   int      `json:"character_id"`
	CharacterName string   `json:"character_name"`
	Scopes        []string `json:"scopes"`
	PortraitURL   string   `json:"portrait_url"`
}

// cookieSecure defaults to true (fail-closed). Set COOKIE_SECURE=false ONLY for local
// HTTP development. In production, omitting the variable keeps cookies Secure.
func cookieSecure() bool {
	return os.Getenv("COOKIE_SECURE") != "false"
}

// HandleCallback handles POST /auth/callback
// Exchanges the authorization code for tokens, verifies them, sets HttpOnly cookies
// and returns character info.
func (h *AuthHandler) HandleCallback(c *fiber.Ctx) error {
	var req callbackRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if req.Code == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing code"})
	}

	if req.CodeVerifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing code_verifier"})
	}

	tokenResp, err := ExchangeCode(c.Context(), req.Code, h.redirectURI, h.clientID, req.CodeVerifier)
	if err != nil {
		log.Printf("ERROR [auth/callback] ExchangeCode failed: %v", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "failed to exchange code"})
	}

	charInfo, err := h.validator.Validate(c.Context(), tokenResp.AccessToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "failed to verify token"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "eve_access_token",
		Value:    tokenResp.AccessToken,
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   tokenResp.ExpiresIn,
		Path:     "/",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "eve_refresh_token",
		Value:    tokenResp.RefreshToken,
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   30 * 24 * 3600,
		Path:     "/",
	})

	return c.JSON(characterResponse{
		CharacterID:   charInfo.CharacterID,
		CharacterName: charInfo.CharacterName,
		Scopes:        strings.Split(charInfo.Scopes, " "),
		PortraitURL:   GetPortraitURL(charInfo.CharacterID, 128),
	})
}

// HandleSession handles GET /auth/session
// Reads the eve_access_token cookie and validates it with EVE ESI.
// Returns {"authenticated": false} (HTTP 200) when no cookie is present.
func (h *AuthHandler) HandleSession(c *fiber.Ctx) error {
	accessToken := c.Cookies("eve_access_token")
	if accessToken == "" {
		return c.JSON(fiber.Map{"authenticated": false})
	}

	charInfo, err := h.validator.Validate(c.Context(), accessToken)
	if err != nil {
		return c.JSON(fiber.Map{"authenticated": false})
	}

	return c.JSON(fiber.Map{
		"authenticated": true,
		"character": characterResponse{
			CharacterID:   charInfo.CharacterID,
			CharacterName: charInfo.CharacterName,
			Scopes:        strings.Split(charInfo.Scopes, " "),
			PortraitURL:   GetPortraitURL(charInfo.CharacterID, 128),
		},
	})
}

// HandleRefresh handles POST /auth/refresh
// Reads the eve_refresh_token cookie, exchanges it for new tokens, and sets a new
// eve_access_token cookie.
func (h *AuthHandler) HandleRefresh(c *fiber.Ctx) error {
	refreshToken := c.Cookies("eve_refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing refresh token"})
	}

	tokenResp, err := RefreshToken(c.Context(), refreshToken, h.clientID)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "failed to refresh token"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "eve_access_token",
		Value:    tokenResp.AccessToken,
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   tokenResp.ExpiresIn,
		Path:     "/",
	})

	return c.SendStatus(fiber.StatusOK)
}

// HandleLogout handles POST /auth/logout
// Deletes both auth cookies by setting MaxAge to -1.
func (h *AuthHandler) HandleLogout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "eve_access_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   -1,
		Path:     "/",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "eve_refresh_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   cookieSecure(),
		SameSite: "Lax",
		MaxAge:   -1,
		Path:     "/",
	})

	return c.SendStatus(fiber.StatusOK)
}
