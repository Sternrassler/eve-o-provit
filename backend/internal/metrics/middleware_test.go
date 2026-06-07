package metrics

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func newTestApp() *fiber.App {
	app := fiber.New()
	app.Use(HTTPMiddleware())
	app.Get("/api/v1/types/:id", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.Get("/api/v1/boom", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadGateway, "boom")
	})
	app.Get("/api/v1/health", func(c *fiber.Ctx) error {
		return c.SendString("healthy")
	})
	return app
}

func TestHTTPMiddleware_CountsByRoutePattern(t *testing.T) {
	app := newTestApp()

	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/types/:id", "200"))
	for _, target := range []string{"/api/v1/types/34", "/api/v1/types/35"} {
		resp, err := app.Test(httptest.NewRequest("GET", target, nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	}

	got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/types/:id", "200")) - before
	if got != 2 {
		t.Errorf("expected 2 requests counted on route pattern, got %v", got)
	}
}

func TestHTTPMiddleware_ErrorStatusFromFiberError(t *testing.T) {
	app := newTestApp()

	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/boom", "502"))
	resp, err := app.Test(httptest.NewRequest("GET", "/api/v1/boom", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 502 {
		t.Fatalf("expected 502, got %d", resp.StatusCode)
	}

	got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/api/v1/boom", "502")) - before
	if got != 1 {
		t.Errorf("expected error response counted with status 502, got %v", got)
	}
}

func TestHTTPMiddleware_CollapsesUnmatchedRoutes(t *testing.T) {
	app := newTestApp()

	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "(unmatched)", "404"))
	for _, target := range []string{"/scanner-probe-1", "/scanner-probe-2/etc"} {
		if _, err := app.Test(httptest.NewRequest("GET", target, nil)); err != nil {
			t.Fatalf("request failed: %v", err)
		}
	}

	got := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "(unmatched)", "404")) - before
	if got != 2 {
		t.Errorf("expected 2 unmatched requests collapsed into one label, got %v", got)
	}
}

func TestHTTPMiddleware_ExcludesHealthAndMetrics(t *testing.T) {
	app := newTestApp()

	before := testutil.CollectAndCount(HTTPRequestsTotal)
	if _, err := app.Test(httptest.NewRequest("GET", "/api/v1/health", nil)); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	after := testutil.CollectAndCount(HTTPRequestsTotal)

	if after != before {
		t.Errorf("health endpoint must not be instrumented (series before=%d after=%d)", before, after)
	}
}
