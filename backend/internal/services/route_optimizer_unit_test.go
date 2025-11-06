package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewRouteOptimizer tests RouteOptimizer initialization
func TestNewRouteOptimizer(t *testing.T) {
	t.Run("with nil dependencies", func(t *testing.T) {
		optimizer := NewRouteOptimizer(nil, nil)

		assert.NotNil(t, optimizer, "RouteOptimizer should be initialized even with nil dependencies")
	})
}

// TestCalculateRoute_Integration tests CalculateRoute with mocked data
func TestCalculateRoute_Integration(t *testing.T) {
	t.Skip("Requires full integration test setup - tested in integration tests")
	// This would require:
	// - Mock SDE repository
	// - Mock SDE database
	// - Mock ItemPair data
}
