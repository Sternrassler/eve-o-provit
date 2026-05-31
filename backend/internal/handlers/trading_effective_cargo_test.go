// Package handlers - tests for the effective-cargo fail-loud contract (issue #147 A3)
package handlers

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/stretchr/testify/assert"
)

// TestApplyEffectiveCargoToShip_FitErrorMarksUnavailable verifies the issue #147
// A3 contract for the single-ship path: a fitting FETCH ERROR sets
// EffectiveCargoUnavailable=true (the explicit "unknown" signal) and leaves the
// capacity at 0 — it must NOT silently fall back to the base hull as if effective.
func TestApplyEffectiveCargoToShip_FitErrorMarksUnavailable(t *testing.T) {
	ship := &models.CharacterShip{CargoCapacity: 1000}

	applyEffectiveCargoToShip(ship, nil, errors.New("esi down"))

	assert.True(t, ship.EffectiveCargoUnavailable, "error must set the unavailable flag")
	assert.Equal(t, 0.0, ship.EffectiveCargoCapacity, "no fabricated effective value on error")
}

// TestApplyEffectiveCargoToShip_NoFittingIsNotAnError verifies that the legitimate
// "no fitting / no cargo bonus" case (err==nil) is NOT flagged as unavailable.
func TestApplyEffectiveCargoToShip_NoFittingIsNotAnError(t *testing.T) {
	// fit == nil, err == nil
	ship := &models.CharacterShip{CargoCapacity: 1000}
	applyEffectiveCargoToShip(ship, nil, nil)
	assert.False(t, ship.EffectiveCargoUnavailable, "no fitting is not an error")
	assert.Equal(t, 0.0, ship.EffectiveCargoCapacity)

	// fit present but EffectiveCargo <= 0
	ship2 := &models.CharacterShip{CargoCapacity: 1000}
	applyEffectiveCargoToShip(ship2, &services.FittingData{Bonuses: services.FittingBonuses{EffectiveCargo: 0}}, nil)
	assert.False(t, ship2.EffectiveCargoUnavailable, "zero cargo bonus is not an error")
	assert.Equal(t, 0.0, ship2.EffectiveCargoCapacity)
}

// TestApplyEffectiveCargoToShip_SetsEffectiveWhenAvailable verifies the happy path.
func TestApplyEffectiveCargoToShip_SetsEffectiveWhenAvailable(t *testing.T) {
	ship := &models.CharacterShip{CargoCapacity: 1000}
	applyEffectiveCargoToShip(ship, &services.FittingData{Bonuses: services.FittingBonuses{EffectiveCargo: 1450}}, nil)
	assert.False(t, ship.EffectiveCargoUnavailable)
	assert.Equal(t, 1450.0, ship.EffectiveCargoCapacity)
}

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
