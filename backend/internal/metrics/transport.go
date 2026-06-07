package metrics

import (
	"net/http"
	"strconv"
	"time"
)

// ESITransport instruments an http.RoundTripper with the ESI request metrics.
// It counts every transport attempt (including library-internal retries) by
// HTTP status, which is the truthful measure for the ESI error limit.
type ESITransport struct {
	base http.RoundTripper
}

// NewESITransport wraps base with ESI metrics instrumentation.
// A nil base falls back to http.DefaultTransport.
func NewESITransport(base http.RoundTripper) *ESITransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &ESITransport{base: base}
}

// RoundTrip implements http.RoundTripper.
func (t *ESITransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()
	resp, err := t.base.RoundTrip(req)

	status := "transport_error"
	if err == nil {
		status = strconv.Itoa(resp.StatusCode)
	}
	ESIRequestsTotal.WithLabelValues(status).Inc()
	ESIRequestDuration.Observe(time.Since(start).Seconds())
	return resp, err
}
