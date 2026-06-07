// Package metrics — Prometheus metrics for the eve-o-provit backend.
//
// All metrics share the "eveoprovit_" namespace so they stay unambiguous in
// the portfolio-wide Prometheus (one scrape job per app, same convention as
// depot's "depot_" prefix).
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal counts handled HTTP requests by method, route pattern
	// and response status. Route patterns (not raw paths) keep label
	// cardinality bounded by the number of registered routes.
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eveoprovit_http_requests_total",
		Help: "Total HTTP requests by method, route pattern and status code",
	}, []string{"method", "route", "status"})

	// HTTPRequestDuration tracks request latency by method and route pattern.
	// Buckets reach 60s because trading route calculations may legitimately
	// run for >30s (ROUTE_CALCULATION_TIMEOUT).
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "eveoprovit_http_request_duration_seconds",
		Help:    "HTTP request duration by method and route pattern",
		Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
	}, []string{"method", "route"})

	// ESIRequestsTotal counts outbound ESI requests by HTTP status — every
	// transport attempt including internal retries, which is the truthful
	// measure for the ESI error limit (watch 420s). "transport_error" marks
	// requests that never produced a response.
	ESIRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "eveoprovit_esi_requests_total",
		Help: "Outbound ESI requests by HTTP status (transport_error if no response)",
	}, []string{"status"})

	// ESIRequestDuration tracks outbound ESI request latency.
	ESIRequestDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "eveoprovit_esi_request_duration_seconds",
		Help:    "Outbound ESI request duration",
		Buckets: prometheus.DefBuckets,
	})

	// TradingCalculationDuration tracks route calculation duration
	TradingCalculationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "eveoprovit_trading_calculation_duration_seconds",
		Help:    "Duration of trading route calculation",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to 51.2s
	})

	// TradingCacheHitsTotal counts market-order cache hits
	TradingCacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "eveoprovit_trading_cache_hits_total",
		Help: "Total trading market-order cache hits",
	})

	// TradingCacheMissesTotal counts market-order cache misses
	TradingCacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "eveoprovit_trading_cache_misses_total",
		Help: "Total trading market-order cache misses",
	})
)
