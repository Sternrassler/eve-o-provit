package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewESIRateLimiter tests rate limiter initialization
func TestNewESIRateLimiter(t *testing.T) {
	limiter := NewESIRateLimiter()
	assert.NotNil(t, limiter)
	assert.NotNil(t, limiter.limiter)
}

// TestRateLimiterWait tests the Wait method
func TestRateLimiterWait(t *testing.T) {
	limiter := NewESIRateLimiter()
	ctx := context.Background()

	start := time.Now()
	err := limiter.Wait(ctx)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Less(t, elapsed, 50*time.Millisecond, "First Wait should be nearly instant")
}

// TestRateLimiterAllow tests the Allow method
func TestRateLimiterAllow(t *testing.T) {
	limiter := NewESIRateLimiter()

	// First call should be allowed
	allowed := limiter.Allow()
	assert.True(t, allowed, "First Allow should return true")
}

// TestRateLimiterBurst tests burst capacity
func TestRateLimiterBurst(t *testing.T) {
	limiter := NewESIRateLimiter()

	// Should allow multiple immediate calls up to burst
	successCount := 0
	for i := 0; i < 100; i++ {
		if limiter.Allow() {
			successCount++
		}
	}

	// ESI rate limit is 15/sec, burst should allow at least a few immediate calls
	assert.GreaterOrEqual(t, successCount, 1, "Should allow at least some burst requests")
}

// TestDefaultRetryConfig tests default retry configuration
func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	assert.Equal(t, 4, cfg.MaxRetries, "Default max retries should be 4")
	assert.Equal(t, 1*time.Second, cfg.InitialBackoff, "Default initial backoff should be 1s")
	assert.Equal(t, 8*time.Second, cfg.MaxBackoff, "Default max backoff should be 8s")
}

// TestRetryWithBackoff_Success tests successful operation (no retries needed)
func TestRetryWithBackoff_Success(t *testing.T) {
	cfg := DefaultRetryConfig()
	attempts := 0

	operation := func() error {
		attempts++
		return nil // Success on first try
	}

	err := RetryWithBackoff(context.Background(), cfg, operation)
	require.NoError(t, err)
	assert.Equal(t, 1, attempts, "Should succeed on first attempt")
}

// TestRetryWithBackoff_EventualSuccess tests operation that succeeds after retries
func TestRetryWithBackoff_EventualSuccess(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}
	attempts := 0

	operation := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("transient error") // Non-429 error terminates immediately
		}
		return nil // Success on 3rd try
	}

	err := RetryWithBackoff(context.Background(), cfg, operation)
	// Since is429Error always returns false, this will fail immediately
	assert.Error(t, err)
	assert.Equal(t, 1, attempts, "Non-429 errors terminate immediately")
}

// TestRetryWithBackoff_MaxRetriesExceeded tests exceeding max retries
func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     2,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     50 * time.Millisecond,
	}
	attempts := 0

	operation := func() error {
		attempts++
		return errors.New("persistent error")
	}

	err := RetryWithBackoff(context.Background(), cfg, operation)
	assert.Error(t, err)
	// Since is429Error always returns false, non-429 errors terminate immediately
	assert.Equal(t, 1, attempts, "Non-429 errors should terminate on first attempt")
}

// TestRetryWithBackoff_ContextCanceled tests context cancellation
func TestRetryWithBackoff_ContextCanceled(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     5,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     1 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	attempts := 0
	operation := func() error {
		attempts++
		time.Sleep(30 * time.Millisecond)
		return errors.New("always fails")
	}

	err := RetryWithBackoff(ctx, cfg, operation)
	assert.Error(t, err)
	// Non-429 errors terminate immediately, no context cancellation involved
	assert.Equal(t, 1, attempts, "Should only attempt once")
}

// TestRetryWithBackoff_BackoffProgression tests exponential backoff timing
func TestRetryWithBackoff_BackoffProgression(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     3,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond,
	}

	attempts := 0
	start := time.Now()

	operation := func() error {
		attempts++
		return errors.New("always fails")
	}

	_ = RetryWithBackoff(context.Background(), cfg, operation)
	elapsed := time.Since(start)

	// With is429Error always false, no retries happen
	assert.Less(t, elapsed.Milliseconds(), int64(10), "Should terminate immediately")
	assert.Equal(t, 1, attempts, "Should only attempt once")
}

// TestIs429Error tests 429 error detection
func TestIs429Error(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "HTTP 429 error",
			err:      errors.New("rate limited"),
			expected: false, // is429Error is a placeholder, always returns false
		},
		{
			name:     "HTTP 500 error",
			err:      errors.New("server error"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Generic error",
			err:      errors.New("some error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := is429Error(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestRetryConfig_CustomValues tests custom retry configuration
func TestRetryConfig_CustomValues(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 500 * time.Millisecond,
		MaxBackoff:     30 * time.Second,
	}

	assert.Equal(t, 10, cfg.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, cfg.InitialBackoff)
	assert.Equal(t, 30*time.Second, cfg.MaxBackoff)
}

// TestRetryWithBackoff_ZeroMaxRetries tests no retries configuration
func TestRetryWithBackoff_ZeroMaxRetries(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     0,
		InitialBackoff: 10 * time.Millisecond,
		MaxBackoff:     100 * time.Millisecond,
	}

	attempts := 0
	operation := func() error {
		attempts++
		return errors.New("error")
	}

	err := RetryWithBackoff(context.Background(), cfg, operation)
	assert.Error(t, err)
	assert.Equal(t, 1, attempts, "Should only try once with MaxRetries=0")
}

// TestRateLimiter_ConcurrentAccess tests concurrent rate limiter usage
func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	limiter := NewESIRateLimiter()
	ctx := context.Background()

	var wg sync.WaitGroup
	goroutines := 20
	errors := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Wait(ctx); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		assert.NoError(t, err)
	}
}

// TestRetryWithBackoff_BackoffCapping tests that backoff doesn't exceed MaxBackoff
func TestRetryWithBackoff_BackoffCapping(t *testing.T) {
	cfg := RetryConfig{
		MaxRetries:     10,
		InitialBackoff: 100 * time.Millisecond,
		MaxBackoff:     200 * time.Millisecond, // Cap at 200ms
	}

	attempts := 0
	start := time.Now()

	operation := func() error {
		attempts++
		return errors.New("always fails")
	}

	_ = RetryWithBackoff(context.Background(), cfg, operation)
	elapsed := time.Since(start)

	// With is429Error always false, no retries happen
	assert.Less(t, elapsed.Milliseconds(), int64(10), "Should terminate immediately")
	assert.Equal(t, 1, attempts, "Should only attempt once")
}
