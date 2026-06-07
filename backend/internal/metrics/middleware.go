package metrics

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// HTTPMiddleware records request count and latency for every handled route.
// Excluded to avoid self-instrumentation and health-ping noise (same approach
// as depot): /metrics, /swagger/* and the health endpoint.
func HTTPMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()
		if path == "/metrics" || path == "/api/v1/health" || strings.HasPrefix(path, "/swagger") {
			return c.Next()
		}

		start := time.Now()
		err := c.Next()

		status := c.Response().StatusCode()
		if err != nil {
			// The global error handler runs after this middleware — mirror its
			// status mapping so the label matches the wire response.
			status = fiber.StatusInternalServerError
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				status = fiberErr.Code
			}
		}

		// Fiber reports "/" as route for unmatched paths — collapse those to a
		// single label value so path scanners cannot inflate cardinality.
		route := c.Route().Path
		if route == "/" && path != "/" {
			route = "(unmatched)"
		}

		// fasthttp reuses request buffers across requests (zero-copy strings) —
		// label values MUST be cloned before Prometheus stores them, otherwise
		// the next request mutates the stored label (prod: method="POS") and
		// Gather fails with duplicate-label errors (/metrics → 500).
		method := strings.Clone(c.Method())
		route = strings.Clone(route)

		HTTPRequestsTotal.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
		HTTPRequestDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
		return err
	}
}
