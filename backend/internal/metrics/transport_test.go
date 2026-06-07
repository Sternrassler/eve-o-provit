package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestESITransport_CountsByStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(420) // ESI error-limited
	}))
	defer srv.Close()

	client := &http.Client{Transport: NewESITransport(nil)}

	before := testutil.ToFloat64(ESIRequestsTotal.WithLabelValues("420"))
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	_ = resp.Body.Close()

	got := testutil.ToFloat64(ESIRequestsTotal.WithLabelValues("420")) - before
	if got != 1 {
		t.Errorf("expected 1 request counted with status 420, got %v", got)
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

func TestESITransport_CountsTransportErrors(t *testing.T) {
	client := &http.Client{Transport: NewESITransport(failingTransport{})}

	before := testutil.ToFloat64(ESIRequestsTotal.WithLabelValues("transport_error"))
	if _, err := client.Get("http://esi.invalid/"); err == nil {
		t.Fatal("expected transport error")
	}

	got := testutil.ToFloat64(ESIRequestsTotal.WithLabelValues("transport_error")) - before
	if got != 1 {
		t.Errorf("expected 1 transport error counted, got %v", got)
	}
}
