// Package services - ESI Rate Limiter
package services

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// ESIRateLimiter implements token bucket rate limiting for ESI API
type ESIRateLimiter struct {
	limiter *rate.Limiter
}

// NewESIRateLimiter creates a new ESI rate limiter
// ESI Spec: 300 requests/minute with burst of 400
func NewESIRateLimiter() *ESIRateLimiter {
	// 300 requests/minute = 5 requests/second
	// Burst capacity: 400
	return &ESIRateLimiter{
		limiter: rate.NewLimiter(rate.Limit(5.0), 400),
	}
}

// Wait blocks until a token is available
func (l *ESIRateLimiter) Wait(ctx context.Context) error {
	return l.limiter.Wait(ctx)
}

// Allow checks if a request can proceed without blocking
func (l *ESIRateLimiter) Allow() bool {
	return l.limiter.Allow()
}

// RetryConfig defines retry behavior for ESI errors
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:     4, // 1s, 2s, 4s, 8s
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     8 * time.Second,
	}
}

// RetryWithBackoff executes a function with exponential backoff on 429 errors
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, fn func() error) error {
	var lastErr error
	backoff := cfg.InitialBackoff

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Try the operation
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if it's a rate limit error (simplified check)
		// In production, this would check for specific 429 error type
		if !is429Error(err) {
			return err
		}

		// Last attempt, don't sleep
		if attempt == cfg.MaxRetries {
			break
		}

		// Wait with exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			// Double the backoff for next attempt
			backoff *= 2
			if backoff > cfg.MaxBackoff {
				backoff = cfg.MaxBackoff
			}
		}
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// is429Error checks if error is a 429 rate limit error
func is429Error(_ error) bool {
	// Simplified check - in production, this would check error type/status code
	// This is a placeholder for proper error type checking
	return false
}
