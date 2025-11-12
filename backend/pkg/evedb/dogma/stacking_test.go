package dogma

import (
	"math"
	"testing"
)

// TestCalculateStackingPenalty validates EVE's stacking penalty formula
func TestCalculateStackingPenalty(t *testing.T) {
	tests := []struct {
		n        int
		expected float64
		name     string
	}{
		{0, 1.000, "1st module (100%)"},
		{1, 0.869, "2nd module (87%)"},
		{2, 0.571, "3rd module (57%)"},
		{3, 0.283, "4th module (28%)"},
		{4, 0.106, "5th module (11%)"},
		{5, 0.030, "6th module (3%)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateStackingPenalty(tt.n)

			// Allow 0.5% tolerance for rounding
			tolerance := 0.005
			if math.Abs(result-tt.expected) > tolerance {
				t.Errorf("calculateStackingPenalty(%d) = %.3f, expected %.3f (±%.3f)",
					tt.n, result, tt.expected, tolerance)
			}

			t.Logf("✓ Module %d: %.1f%% effectiveness", tt.n+1, result*100)
		})
	}
}

// TestApplyModifierWithStacking_CompareToWithoutStacking validates stacking penalty impact
func TestApplyModifierWithStacking_CompareToWithoutStacking(t *testing.T) {
	baseValue := 100.0
	modifierValue := 20.0 // +20% bonus
	count := 3

	mod := ModifierInfo{
		ModifyingAttributeID: 624, // WarpSBonus
		Operation:            6,   // PostPercent
	}

	// Without stacking (old behavior)
	withoutStacking := ApplyModifier(baseValue, mod, modifierValue, count)
	// 100 × (1.20)³ = 100 × 1.728 = 172.8

	// With stacking (new behavior)
	withStacking := ApplyModifierWithStacking(nil, baseValue, mod, modifierValue, count, true)
	// 100 × 1.20 × 1.174 × 1.114 = ~156.7

	expectedWithout := 172.8
	expectedWith := 156.7

	if math.Abs(withoutStacking-expectedWithout) > 1.0 {
		t.Errorf("Without stacking: expected %.1f, got %.1f", expectedWithout, withoutStacking)
	}

	if math.Abs(withStacking-expectedWith) > 1.0 {
		t.Errorf("With stacking: expected %.1f, got %.1f", expectedWith, withStacking)
	}

	difference := ((withoutStacking - withStacking) / withStacking) * 100
	t.Logf("✓ Without stacking: %.1f (+72.8%%)", withoutStacking)
	t.Logf("✓ With stacking:    %.1f (+56.7%%)", withStacking)
	t.Logf("✓ Stacking penalty reduces bonus by %.1f%%", difference)
}

// TestApplyModifierWithStacking_NonStackableAttribute validates bypass for non-stackable attributes
func TestApplyModifierWithStacking_NonStackableAttribute(t *testing.T) {
	baseValue := 100.0
	modifierValue := 20.0
	count := 3

	mod := ModifierInfo{
		ModifyingAttributeID: 600, // warpSpeedMultiplier (not stackable)
		Operation:            6,   // PostPercent
	}

	// With isStackable=false, should behave like ApplyModifier
	resultStacking := ApplyModifierWithStacking(nil, baseValue, mod, modifierValue, count, false)
	resultRegular := ApplyModifier(baseValue, mod, modifierValue, count)

	if resultStacking != resultRegular {
		t.Errorf("Non-stackable attribute should behave like ApplyModifier: stacking=%.2f, regular=%.2f",
			resultStacking, resultRegular)
	}

	t.Logf("✓ Non-stackable attribute bypasses stacking penalties: %.2f", resultStacking)
}

// TestApplyModifierWithStacking_SingleModule validates single module behavior
func TestApplyModifierWithStacking_SingleModule(t *testing.T) {
	baseValue := 100.0
	modifierValue := 20.0
	count := 1

	mod := ModifierInfo{
		ModifyingAttributeID: 624,
		Operation:            6, // PostPercent
	}

	// Single module should have 100% effectiveness regardless of stackable flag
	result := ApplyModifierWithStacking(nil, baseValue, mod, modifierValue, count, true)
	expected := 120.0 // 100 × 1.20

	if math.Abs(result-expected) > 0.1 {
		t.Errorf("Single module: expected %.1f, got %.1f", expected, result)
	}

	t.Logf("✓ Single module: %.1f (100%% effectiveness)", result)
}
