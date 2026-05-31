// Package handlers - tests for the effective-cargo fail-loud contract (issue #147 A3)
package handlers

import (
	"encoding/json"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestEffectiveCargoUnavailable_WireForm guards the exact JSON field name the
// frontend depends on: effective_cargo_unavailable (omitempty → only present when true).
func TestEffectiveCargoUnavailable_WireForm(t *testing.T) {
	// Asset ship: flag true must serialize the field.
	b, err := json.Marshal(models.CharacterAssetShip{EffectiveCargoUnavailable: true})
	assert.NoError(t, err)
	assert.Contains(t, string(b), `"effective_cargo_unavailable":true`)

	// Flag false must omit the field (omitempty).
	b2, err := json.Marshal(models.CharacterAssetShip{EffectiveCargoUnavailable: false})
	assert.NoError(t, err)
	assert.NotContains(t, string(b2), "effective_cargo_unavailable")

	// Same for the single-ship model.
	b3, err := json.Marshal(models.CharacterShip{EffectiveCargoUnavailable: true})
	assert.NoError(t, err)
	assert.Contains(t, string(b3), `"effective_cargo_unavailable":true`)
}
