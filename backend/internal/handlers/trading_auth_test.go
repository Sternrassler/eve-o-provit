package handlers

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestGetCharacterLocation_NoAuthLocals: ohne gesetzte Locals → 401, kein Panic.
func TestGetCharacterLocation_NoAuthLocals(t *testing.T) {
	h := &TradingHandler{}
	app := fiber.New()
	app.Get("/loc", h.GetCharacterLocation)
	resp, err := app.Test(httptest.NewRequest("GET", "/loc", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (kein Panic)", resp.StatusCode)
	}
}
