package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAssetsHandler_SellOptions_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &AssetsHandler{} // validation happens before any service call
	app.Post("/sell", func(c *fiber.Ctx) error {
		c.Locals("character_id", 1)
		c.Locals("access_token", "tok")
		return h.SellOptions(c)
	})
	req := httptest.NewRequest("POST", "/sell", strings.NewReader(`{"type_id":0,"location_id":1,"quantity":0}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestAssetsHandler_SellOptions_Unauthorized(t *testing.T) {
	app := fiber.New()
	h := &AssetsHandler{}
	app.Post("/sell", h.SellOptions)
	req := httptest.NewRequest("POST", "/sell", strings.NewReader(`{"type_id":34,"location_id":60003760,"quantity":10}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestAssetsHandler_ListAssets_Unauthorized(t *testing.T) {
	app := fiber.New()
	h := &AssetsHandler{}
	app.Get("/assets", h.ListAssets)
	req := httptest.NewRequest("GET", "/assets", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
