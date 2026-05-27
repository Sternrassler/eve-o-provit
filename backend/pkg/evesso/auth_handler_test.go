package evesso

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestCookieSecure_FailClosed(t *testing.T) {
	cases := []struct {
		env  string
		set  bool
		want bool
	}{
		{set: false, want: true},
		{env: "true", set: true, want: true},
		{env: "false", set: true, want: false},
		{env: "anything", set: true, want: true},
	}
	for _, c := range cases {
		os.Unsetenv("COOKIE_SECURE")
		if c.set {
			os.Setenv("COOKIE_SECURE", c.env)
		}
		if got := cookieSecure(); got != c.want {
			t.Errorf("COOKIE_SECURE=%q (set=%v): cookieSecure()=%v, want %v", c.env, c.set, got, c.want)
		}
	}
	os.Unsetenv("COOKIE_SECURE")
}

// newTestAuthHandler creates an AuthHandler with:
//   - a real in-memory TokenValidator (signed JWT via newTestValidator)
//   - stubbed exchange function (returns a signed access token)
//
// The returned signer key signs tokens that the validator will accept.
func newTestAuthHandler(t *testing.T) (*AuthHandler, func(code string) string) {
	t.Helper()
	v, signer := newTestValidator(t)

	// Helper: produce a signed access token accepted by v
	makeToken := func(clientID string) string {
		return signToken(
			t, signer,
			"login.eveonline.com",
			[]string{clientID, audienceEVE},
			"CHARACTER:EVE:90000001",
			time.Now().Add(20*time.Minute),
			validClaims(),
		)
	}

	h := NewAuthHandler(testClientID, "http://localhost/callback", v)
	// Stub the exchange seam so no real HTTP call is made.
	h.exchange = func(_ context.Context, code, _, clientID, _ string) (*TokenResponse, error) {
		return &TokenResponse{
			AccessToken:  makeToken(clientID),
			RefreshToken: "stub-refresh-" + code,
			ExpiresIn:    1200,
		}, nil
	}
	h.refreshFn = func(_ context.Context, refreshToken, clientID string) (*TokenResponse, error) {
		return &TokenResponse{
			AccessToken:  makeToken(clientID),
			RefreshToken: "new-refresh-" + refreshToken,
			ExpiresIn:    1200,
		}, nil
	}

	return h, makeToken
}

// TestHandleMobileCallback_ReturnsTokensInBody verifies the mobile endpoint:
//   - returns 200
//   - body JSON contains non-empty access_token and refresh_token
//   - body JSON contains a character object
func TestHandleMobileCallback_ReturnsTokensInBody(t *testing.T) {
	h, _ := newTestAuthHandler(t)
	mobileClientID := "mobile-client-id"
	h.WithMobile(mobileClientID, "eveauth-eveoprovit://callback")
	// Also allow the validator to accept the mobile client ID.
	h.validator.AddAcceptedClientID(mobileClientID)

	app := newTestApp(h)

	body, _ := json.Marshal(map[string]string{
		"code":          "authcode123",
		"code_verifier": "verifier456",
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/mobile/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	accessToken, _ := result["access_token"].(string)
	if accessToken == "" {
		t.Errorf("access_token is empty; got body: %v", result)
	}
	refreshToken, _ := result["refresh_token"].(string)
	if refreshToken == "" {
		t.Errorf("refresh_token is empty; got body: %v", result)
	}
	character, ok := result["character"].(map[string]interface{})
	if !ok || character == nil {
		t.Errorf("character missing or wrong type; got body: %v", result)
	} else {
		charID, _ := character["character_id"].(float64)
		if int(charID) != 90000001 {
			t.Errorf("character_id = %v, want 90000001", charID)
		}
		charName, _ := character["character_name"].(string)
		if charName == "" {
			t.Errorf("character_name is empty")
		}
	}
}

// TestHandleMobileCallback_MissingCode verifies 400 when code is absent.
func TestHandleMobileCallback_MissingCode(t *testing.T) {
	h, _ := newTestAuthHandler(t)
	h.WithMobile("mobile-client-id", "eveauth-eveoprovit://callback")
	app := newTestApp(h)

	body, _ := json.Marshal(map[string]string{"code_verifier": "v"})
	req := httptest.NewRequest(http.MethodPost, "/auth/mobile/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestHandleMobileCallback_MissingCodeVerifier verifies 400 when code_verifier is absent.
func TestHandleMobileCallback_MissingCodeVerifier(t *testing.T) {
	h, _ := newTestAuthHandler(t)
	h.WithMobile("mobile-client-id", "eveauth-eveoprovit://callback")
	app := newTestApp(h)

	body, _ := json.Marshal(map[string]string{"code": "c"})
	req := httptest.NewRequest(http.MethodPost, "/auth/mobile/callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// TestHandleMobileRefresh_ReturnsTokensInBody verifies mobile refresh returns new tokens.
func TestHandleMobileRefresh_ReturnsTokensInBody(t *testing.T) {
	h, _ := newTestAuthHandler(t)
	mobileClientID := "mobile-client-id"
	h.WithMobile(mobileClientID, "eveauth-eveoprovit://callback")
	h.validator.AddAcceptedClientID(mobileClientID)
	app := newTestApp(h)

	body, _ := json.Marshal(map[string]string{"refresh_token": "old-refresh"})
	req := httptest.NewRequest(http.MethodPost, "/auth/mobile/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	accessToken, _ := result["access_token"].(string)
	if accessToken == "" {
		t.Errorf("access_token is empty")
	}
	refreshToken, _ := result["refresh_token"].(string)
	if refreshToken == "" {
		t.Errorf("refresh_token is empty")
	}
	if !strings.Contains(refreshToken, "old-refresh") {
		t.Errorf("refresh_token %q doesn't contain the original token", refreshToken)
	}
	if _, hasChar := result["character"]; hasChar {
		t.Errorf("refresh response must not include character, got %v", result["character"])
	}
}

// TestHandleMobile_NotConfigured verifies both mobile endpoints return 503
// when no mobile client ID is configured.
func TestHandleMobile_NotConfigured(t *testing.T) {
	h, _ := newTestAuthHandler(t) // no WithMobile call -> mobileClientID == ""
	app := newTestApp(h)

	for _, path := range []string{"/auth/mobile/callback", "/auth/mobile/refresh"} {
		body, _ := json.Marshal(map[string]string{
			"code":          "c",
			"code_verifier": "v",
			"refresh_token": "r",
		})
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test(%s): %v", path, err)
		}
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("%s status = %d, want 503", path, resp.StatusCode)
		}
	}
}

// newTestApp creates a minimal Fiber app with the mobile auth routes wired.
func newTestApp(h *AuthHandler) *fiber.App {
	app := fiber.New()
	auth := app.Group("/auth")
	auth.Post("/mobile/callback", h.HandleMobileCallback)
	auth.Post("/mobile/refresh", h.HandleMobileRefresh)
	return app
}

// TestHandleMobileRefresh_MissingRefreshToken verifies 400 when refresh_token absent.
func TestHandleMobileRefresh_MissingRefreshToken(t *testing.T) {
	h, _ := newTestAuthHandler(t)
	h.WithMobile("mobile-client-id", "eveauth-eveoprovit://callback")
	app := newTestApp(h)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/auth/mobile/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
