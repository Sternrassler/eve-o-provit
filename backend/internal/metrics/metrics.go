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

	// TradingCacheHitsTotal counts cache hits
	TradingCacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trading_cache_hits_total",
		Help: "Total trading cache hits",
	})

	// TradingCacheMissesTotal counts cache misses
	TradingCacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trading_cache_misses_total",
		Help: "Total trading cache misses",
	})

	// TradingWorkerPoolQueueSize tracks worker pool queue size
	TradingWorkerPoolQueueSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "trading_worker_pool_queue_size",
		Help: "Current trading worker pool queue size",
	}, []string{"pool_type"})
)
