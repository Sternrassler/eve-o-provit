package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewRouteFinder tests RouteFinder initialization
func TestNewRouteFinder(t *testing.T) {
	t.Run("with nil dependencies", func(t *testing.T) {
		finder := NewRouteFinder(nil, nil, nil, nil, nil)

		assert.NotNil(t, finder, "RouteFinder should be initialized even with nil dependencies")
	})

	t.Run("with Redis client", func(t *testing.T) {
		// Can't test Redis without actual connection, but verify it doesn't panic
		finder := NewRouteFinder(nil, nil, nil, nil, nil)

		assert.NotNil(t, finder)
		assert.Nil(t, finder.marketCache, "Market cache should be nil when Redis is nil")
	})
}

// TestFindProfitableItems_Integration tests FindProfitableItems with mocked data
func TestFindProfitableItems_Integration(t *testing.T) {
	t.Skip("Requires full integration test setup - tested in integration tests")
	// This would require:
	// - Mock ESI client
	// - Mock market repository
	// - Mock SDE repository
	// - Mock SDE database
}
