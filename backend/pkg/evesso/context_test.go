package evesso

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAuthFromContext(t *testing.T) {
	tests := []struct {
		name    string
		locals  func(c *fiber.Ctx)
		wantID  int
		wantTok string
		wantOK  bool
	}{
		{"both present", func(c *fiber.Ctx) { c.Locals(localKeyCharacterID, 42); c.Locals(localKeyAccessToken, "tok") }, 42, "tok", true},
		{"missing token", func(c *fiber.Ctx) { c.Locals(localKeyCharacterID, 42) }, 0, "", false},
		{"missing character id", func(c *fiber.Ctx) { c.Locals(localKeyAccessToken, "tok") }, 0, "", false},
		{"empty token", func(c *fiber.Ctx) { c.Locals(localKeyCharacterID, 42); c.Locals(localKeyAccessToken, "") }, 0, "", false},
		{"character id wrong type", func(c *fiber.Ctx) { c.Locals(localKeyCharacterID, "42"); c.Locals(localKeyAccessToken, "tok") }, 0, "", false},
		{"nothing set", func(c *fiber.Ctx) {}, 0, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", func(c *fiber.Ctx) error {
				tt.locals(c)
				id, tok, ok := AuthFromContext(c)
				return c.JSON(fiber.Map{"id": id, "tok": tok, "ok": ok})
			})

			resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			var got struct {
				ID  int    `json:"id"`
				Tok string `json:"tok"`
				OK  bool   `json:"ok"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
				t.Fatalf("decode failed: %v", err)
			}
			if got.ID != tt.wantID || got.Tok != tt.wantTok || got.OK != tt.wantOK {
				t.Errorf("AuthFromContext = (%d, %q, %v), want (%d, %q, %v)",
					got.ID, got.Tok, got.OK, tt.wantID, tt.wantTok, tt.wantOK)
			}
		})
	}
}
