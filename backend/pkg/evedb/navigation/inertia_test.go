package navigation

import (
	"context"
	"math"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	_ "github.com/mattn/go-sqlite3"
)

// TestGetShipInertiaDeterministic_Scenario1_BaseInertia validates base inertia + align time without skills/modules
func TestGetShipInertiaDeterministic_Scenario1_BaseInertia(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Scenario 1: Iteron Mark V - Base inertia (no skills, no modules)
	// Expected: Base Inertia ~0.55, Mass ~18,900,000 kg, Align Time ~14.4s
	result, err := GetShipInertiaDeterministic(ctx, db, 650, nil, nil)
	if err != nil {
		t.Fatalf("GetShipInertiaDeterministic failed: %v", err)
	}

	// Validations
	if result.ShipTypeID != 650 {
		t.Errorf("Expected ShipTypeID 650, got %d", result.ShipTypeID)
	}

	if result.ShipName == "" {
		t.Error("Expected non-empty ShipName")
	}

	if result.BaseInertia <= 0 {
		t.Errorf("Expected positive BaseInertia, got %.3f", result.BaseInertia)
	}

	if result.ShipMass <= 0 {
		t.Errorf("Expected positive ShipMass, got %.0f kg", result.ShipMass)
	}

	// Without skills/modules, effective should equal base
	if result.EffectiveInertia != result.BaseInertia {
		t.Errorf("Expected EffectiveInertia %.3f to equal BaseInertia %.3f",
			result.EffectiveInertia, result.BaseInertia)
	}

	// No bonuses applied
	if len(result.AppliedBonuses) != 0 {
		t.Errorf("Expected 0 bonuses, got %d", len(result.AppliedBonuses))
	}

	// Validate align time calculation: ln(2) × inertia × mass / 500000
	expectedAlignTime := math.Log(2) * result.BaseInertia * result.ShipMass / 500000.0
	tolerance := 0.01 // 10ms tolerance
	if math.Abs(result.AlignTime-expectedAlignTime) > tolerance {
		t.Errorf("Expected AlignTime %.3fs, got %.3fs", expectedAlignTime, result.AlignTime)
	}

	t.Logf("✓ Scenario 1: Base Inertia = %.3f, Mass = %.0f kg, Align Time = %.2fs",
		result.BaseInertia, result.ShipMass, result.AlignTime)
}

// TestGetShipInertiaDeterministic_Scenario2_EvasiveManeuveringSkills validates Evasive Maneuvering skill bonuses
func TestGetShipInertiaDeterministic_Scenario2_EvasiveManeuveringSkills(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Scenario 2: Iteron Mark V with Evasive Maneuvering skill levels I-V
	// Evasive Maneuvering Skill: -5% inertia per level (max -25% at level V)
	// Lower inertia = faster align = better!
	testCases := []struct {
		level            int
		expectedBonus    float64 // Negative bonus (reduction)
		expectedMultiple float64 // Multiplier (< 1.0 for reduction)
	}{
		{1, -5.0, 0.95},  // -5%
		{2, -10.0, 0.90}, // -10%
		{3, -15.0, 0.85}, // -15%
		{4, -20.0, 0.80}, // -20%
		{5, -25.0, 0.75}, // -25%
	}

	for _, tc := range testCases {
		// Mock character skills (Evasive Maneuvering skill at given level)
		charSkills := &cargo.CharacterSkills{
			Skills: []struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{
				{SkillID: 3452, ActiveSkillLevel: tc.level, TrainedSkillLevel: tc.level}, // Evasive Maneuvering
			},
		}

		result, err := GetShipInertiaDeterministic(ctx, db, 650, charSkills, nil)
		if err != nil {
			t.Fatalf("Level %d: GetShipInertiaDeterministic failed: %v", tc.level, err)
		}

		// Calculate expected effective inertia
		expectedEffective := result.BaseInertia * tc.expectedMultiple

		// Validate effective inertia (with tolerance for floating-point)
		tolerance := 0.001
		if math.Abs(result.EffectiveInertia-expectedEffective) > tolerance {
			t.Errorf("Level %d: Expected EffectiveInertia %.3f, got %.3f",
				tc.level, expectedEffective, result.EffectiveInertia)
		}

		// Validate applied bonuses
		if len(result.AppliedBonuses) != 1 {
			t.Errorf("Level %d: Expected 1 bonus, got %d", tc.level, len(result.AppliedBonuses))
		} else {
			bonus := result.AppliedBonuses[0]
			if bonus.Source != "Skill" {
				t.Errorf("Level %d: Expected Source 'Skill', got '%s'", tc.level, bonus.Source)
			}
			if bonus.Name != "Evasive Maneuvering" {
				t.Errorf("Level %d: Expected Name 'Evasive Maneuvering', got '%s'", tc.level, bonus.Name)
			}
			if math.Abs(bonus.Value-tc.expectedBonus) > 0.01 {
				t.Errorf("Level %d: Expected Value %.1f%%, got %.1f%%", tc.level, tc.expectedBonus, bonus.Value)
			}
			if bonus.Count != tc.level {
				t.Errorf("Level %d: Expected Count %d, got %d", tc.level, tc.level, bonus.Count)
			}
		}

		// Validate align time improved (lower is better)
		expectedAlignTime := math.Log(2) * result.EffectiveInertia * result.ShipMass / 500000.0
		if math.Abs(result.AlignTime-expectedAlignTime) > 0.01 {
			t.Errorf("Level %d: Expected AlignTime %.3fs, got %.3fs",
				tc.level, expectedAlignTime, result.AlignTime)
		}

		t.Logf("✓ Level %d: Inertia %.3f → %.3f (%.0f%%), Align Time %.2fs",
			tc.level, result.BaseInertia, result.EffectiveInertia, tc.expectedBonus, result.AlignTime)
	}
}

// TestGetShipInertiaDeterministic_Scenario3_FullFitting validates skills + modules
func TestGetShipInertiaDeterministic_Scenario3_FullFitting(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Scenario 3: Iteron Mark V with Evasive Maneuvering V + 2× Inertial Stabilizers II
	// Expected: Base → Skill (-25%) → Modules (-13% each with stacking)
	// Result: ~46% faster align time

	// Mock character skills (Evasive Maneuvering V)
	charSkills := &cargo.CharacterSkills{
		Skills: []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{
			{SkillID: 3452, ActiveSkillLevel: 5, TrainedSkillLevel: 5}, // Evasive Maneuvering V
		},
	}

	// Mock fitted modules (2× Inertial Stabilizer II)
	fittedItems := []cargo.FittedItem{
		{TypeID: 2605, Slot: "LowSlot0"}, // Inertial Stabilizer II
		{TypeID: 2605, Slot: "LowSlot1"}, // Inertial Stabilizer II (stacking penalty applies)
	}

	result, err := GetShipInertiaDeterministic(ctx, db, 650, charSkills, fittedItems)
	if err != nil {
		t.Fatalf("GetShipInertiaDeterministic failed: %v", err)
	}

	// Validations
	if result.EffectiveInertia >= result.BaseInertia {
		t.Errorf("Expected EffectiveInertia < BaseInertia (modules should reduce inertia)")
	}

	// Should have skill bonus + module bonuses
	if len(result.AppliedBonuses) < 2 {
		t.Errorf("Expected at least 2 bonuses (skill + modules), got %d", len(result.AppliedBonuses))
	}

	// Validate align time improved significantly
	baseAlignTime := math.Log(2) * result.BaseInertia * result.ShipMass / 500000.0
	improvement := (baseAlignTime - result.AlignTime) / baseAlignTime * 100.0

	if improvement <= 0 {
		t.Errorf("Expected align time improvement, got %.1f%%", improvement)
	}

	t.Logf("✓ Scenario 3: Full Fitting")
	t.Logf("  Base Inertia: %.3f → Effective: %.3f", result.BaseInertia, result.EffectiveInertia)
	t.Logf("  Base Align: %.2fs → Effective: %.2fs (%.0f%% faster)", baseAlignTime, result.AlignTime, improvement)
	t.Logf("  Applied Bonuses: %d", len(result.AppliedBonuses))
	for _, bonus := range result.AppliedBonuses {
		t.Logf("    - %s: %s (%.2f%%, Count: %d)", bonus.Source, bonus.Name, bonus.Value, bonus.Count)
	}
}

// TestGetShipInertiaDeterministic_Scenario4_ErrorHandling validates error cases
func TestGetShipInertiaDeterministic_Scenario4_ErrorHandling(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test 1: Invalid ship type ID
	_, err := GetShipInertiaDeterministic(ctx, db, 99999999, nil, nil)
	if err == nil {
		t.Error("Expected error for invalid ship type ID, got nil")
	} else {
		t.Logf("✓ Invalid ship type ID error: %v", err)
	}

	// Test 2: Valid ship without fitted items (should not fail)
	result, err := GetShipInertiaDeterministic(ctx, db, 650, nil, []cargo.FittedItem{})
	if err != nil {
		t.Errorf("Expected no error with empty fittedItems, got: %v", err)
	}
	if result.EffectiveInertia != result.BaseInertia {
		t.Error("Expected EffectiveInertia to equal BaseInertia with no modules")
	}
	t.Logf("✓ Empty fitted items handled correctly")
}

// TestCalculateAlignTimeFormula validates the align time formula
func TestCalculateAlignTimeFormula(t *testing.T) {
	// Test align time formula: ln(2) × inertia × mass / 500000
	// Example: Iteron Mark V (Inertia: 0.55, Mass: 18,900,000 kg)
	testCases := []struct {
		name          string
		inertia       float64
		mass          float64
		expectedAlign float64
	}{
		{
			name:          "Iteron Mark V Base",
			inertia:       0.55,
			mass:          18900000.0,
			expectedAlign: 14.4, // ~14.4 seconds
		},
		{
			name:          "Iteron Mark V Skilled",
			inertia:       0.4125, // -25% from Evasive Maneuvering V
			mass:          18900000.0,
			expectedAlign: 10.8, // ~10.8 seconds
		},
		{
			name:          "Iteron Mark V Full Fit",
			inertia:       0.298, // Skilled + 2× Inertial Stabilizers
			mass:          18900000.0,
			expectedAlign: 7.8, // ~7.8 seconds
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateAlignTimeFromInertia(tc.inertia, tc.mass)
			tolerance := 0.5 // 500ms tolerance for approximation
			if math.Abs(result-tc.expectedAlign) > tolerance {
				t.Errorf("%s: Expected %.1fs, got %.1fs", tc.name, tc.expectedAlign, result)
			} else {
				t.Logf("✓ %s: %.1fs (expected %.1fs)", tc.name, result, tc.expectedAlign)
			}
		})
	}
}
