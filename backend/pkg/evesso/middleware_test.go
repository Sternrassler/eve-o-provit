package evesso

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestAuthMiddleware_AcceptsBearer(t *testing.T) {
	v, signer := newTestValidator(t)
	tok := signToken(
		t, signer,
		"login.eveonline.com",
		[]string{testClientID, audienceEVE},
		"CHARACTER:EVE:777",
		time.Now().Add(20*time.Minute),
		validClaims(),
	)

	app := fiber.New()
	app.Get("/protected", NewAuthMiddleware(v), func(c *fiber.Ctx) error {
		charID := c.Locals("character_id")
		return c.SendString(fmt.Sprintf("%v", charID))
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tok)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "777" {
		t.Errorf("character_id local = %q, want %q", string(body), "777")
	}
}
