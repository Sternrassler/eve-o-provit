// Package metrics - Prometheus metrics for trading operations
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// TradingCalculationDuration tracks route calculation duration
	TradingCalculationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "trading_calculation_duration_seconds",
		Help:    "Duration of trading route calculation",
		Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to 51.2s
	})

	// TradingCacheHitRatio tracks cache hit ratio
	TradingCacheHitRatio = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "trading_cache_hit_ratio",
		Help: "Cache hit ratio for market orders",
	})

	// ESIRequestsTotal counts ESI requests by status code
	ESIRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "esi_requests_total",
		Help: "Total ESI requests by status code",
	}, []string{"status_code"})

	// ESIRateLimitErrorsTotal counts 429 errors
	ESIRateLimitErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "esi_rate_limit_errors_total",
		Help: "Total ESI rate limit errors (429)",
	})

	// WorkerPoolQueueSize tracks worker pool queue size
	WorkerPoolQueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "worker_pool_queue_size",
		Help: "Current worker pool queue size",
	}, []string{"pool_type"})

	// CacheHitsTotal counts cache hits
	CacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total cache hits",
	})

	// CacheMissesTotal counts cache misses
	CacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total cache misses",
	})
)
