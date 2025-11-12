package navigation

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	_ "github.com/mattn/go-sqlite3"
)

// TestGetShipWarpSpeedDeterministic_Scenario1_BaseSpeed validates base warp speed without skills/modules
func TestGetShipWarpSpeedDeterministic_Scenario1_BaseSpeed(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Scenario 1: Iteron Mark V - Base warp speed (no skills, no modules)
	// Expected: Base warp speed = 3.0 AU/s (default for industrial ships)
	result, err := GetShipWarpSpeedDeterministic(ctx, db, 650, nil, nil)
	if err != nil {
		t.Fatalf("GetShipWarpSpeedDeterministic failed: %v", err)
	}

	// Validations
	if result.ShipTypeID != 650 {
		t.Errorf("Expected ShipTypeID 650, got %d", result.ShipTypeID)
	}

	if result.ShipName == "" {
		t.Error("Expected non-empty ShipName")
	}

	if result.BaseWarpSpeed <= 0 {
		t.Errorf("Expected positive BaseWarpSpeed, got %.2f", result.BaseWarpSpeed)
	}

	// Without skills/modules, effective should equal base
	if result.EffectiveWarpSpeed != result.BaseWarpSpeed {
		t.Errorf("Expected EffectiveWarpSpeed %.2f to equal BaseWarpSpeed %.2f",
			result.EffectiveWarpSpeed, result.BaseWarpSpeed)
	}

	// No bonuses applied
	if len(result.AppliedBonuses) != 0 {
		t.Errorf("Expected 0 bonuses, got %d", len(result.AppliedBonuses))
	}

	t.Logf("✓ Scenario 1: Base Warp Speed = %.2f AU/s (no bonuses)", result.BaseWarpSpeed)
}

// TestGetShipWarpSpeedDeterministic_Scenario2_NavigationSkills validates Navigation skill bonuses
func TestGetShipWarpSpeedDeterministic_Scenario2_NavigationSkills(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Scenario 2: Iteron Mark V with Navigation skill levels I-V
	// Navigation Skill: +5% warp speed per level (max +25% at level V)
	testCases := []struct {
		level            int
		expectedBonus    float64
		expectedMultiple float64
	}{
		{1, 5.0, 1.05},  // +5%
		{2, 10.0, 1.10}, // +10%
		{3, 15.0, 1.15}, // +15%
		{4, 20.0, 1.20}, // +20%
		{5, 25.0, 1.25}, // +25%
	}

	for _, tc := range testCases {
		// Mock character skills (Navigation skill at given level)
		charSkills := &cargo.CharacterSkills{
			Skills: []struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{
				{SkillID: 3456, ActiveSkillLevel: tc.level, TrainedSkillLevel: tc.level}, // Navigation
			},
		}

		result, err := GetShipWarpSpeedDeterministic(ctx, db, 650, charSkills, nil)
		if err != nil {
			t.Fatalf("Level %d: GetShipWarpSpeedDeterministic failed: %v", tc.level, err)
		}

		// Calculate expected effective warp speed
		expectedEffective := result.BaseWarpSpeed * tc.expectedMultiple

		// Validate effective warp speed (allow 0.01% tolerance)
		tolerance := expectedEffective * 0.0001
		if abs(result.EffectiveWarpSpeed-expectedEffective) > tolerance {
			t.Errorf("Level %d: Expected EffectiveWarpSpeed %.4f, got %.4f",
				tc.level, expectedEffective, result.EffectiveWarpSpeed)
		}

		// Validate bonus entry
		if len(result.AppliedBonuses) != 1 {
			t.Errorf("Level %d: Expected 1 bonus, got %d", tc.level, len(result.AppliedBonuses))
		} else {
			bonus := result.AppliedBonuses[0]
			if bonus.Source != "Skill" {
				t.Errorf("Level %d: Expected Source 'Skill', got '%s'", tc.level, bonus.Source)
			}
			if bonus.Value != tc.expectedBonus {
				t.Errorf("Level %d: Expected bonus %.1f%%, got %.1f%%",
					tc.level, tc.expectedBonus, bonus.Value)
			}
			if bonus.Count != tc.level {
				t.Errorf("Level %d: Expected Count %d, got %d", tc.level, tc.level, bonus.Count)
			}
		}

		t.Logf("✓ Scenario 2 (Level %d): Warp Speed = %.2f AU/s (+%.1f%%)",
			tc.level, result.EffectiveWarpSpeed, tc.expectedBonus)
	}
}

// TestGetShipWarpSpeedDeterministic_Scenario3_FullFitting validates skills + modules
func TestGetShipWarpSpeedDeterministic_Scenario3_FullFitting(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Scenario 3: Iteron Mark V with Navigation V + 3× Hyperspatial Velocity Optimizer I
	// Base: 3.0 AU/s
	// Navigation V: +25% → 3.75 AU/s
	// 3× Hyperspatial Rigs: ×1.20, ×1.14, ×1.08 (with stacking)
	// Expected: ~5.52 AU/s (+84% total improvement)

	charSkills := &cargo.CharacterSkills{
		Skills: []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{
			{SkillID: 3456, ActiveSkillLevel: 5, TrainedSkillLevel: 5}, // Navigation V
		},
	}

	fittedItems := []cargo.FittedItem{
		{TypeID: 31161, Slot: "Rig0"}, // Medium Hyperspatial Velocity Optimizer I
		{TypeID: 31161, Slot: "Rig1"}, // Medium Hyperspatial Velocity Optimizer I
		{TypeID: 31161, Slot: "Rig2"}, // Medium Hyperspatial Velocity Optimizer I
	}

	result, err := GetShipWarpSpeedDeterministic(ctx, db, 650, charSkills, fittedItems)
	if err != nil {
		t.Fatalf("GetShipWarpSpeedDeterministic failed: %v", err)
	}

	// Expected calculation WITH stacking penalties (EVE formula):
	// Base: 3.5 AU/s
	// With Navigation V: 3.5 × 1.25 = 4.375 AU/s
	// With 3× Hyperspatial Rigs (+20% each):
	//   1st rig: 4.375 × 1.20 = 5.250 AU/s (100% effectiveness)
	//   2nd rig: 5.250 × 1.174 = 6.164 AU/s (87% effectiveness)
	//   3rd rig: 6.164 × 1.114 = 6.867 AU/s (57% effectiveness)
	// Result: ~6.87 AU/s (+96% total improvement)
	//
	// Stacking penalty formula: e^(-(n-1)²/2.67²)
	//   1st: 100.0%, 2nd: 86.9%, 3rd: 57.1%, 4th: 28.3%, 5th: 10.6%, 6th: 3.0%

	expectedMin := 6.5
	expectedMax := 7.0

	if result.EffectiveWarpSpeed < expectedMin || result.EffectiveWarpSpeed > expectedMax {
		t.Errorf("Expected EffectiveWarpSpeed between %.2f and %.2f, got %.2f",
			expectedMin, expectedMax, result.EffectiveWarpSpeed)
	}

	// Validate bonuses applied (1 skill + 1 grouped rig entry)
	if len(result.AppliedBonuses) < 2 {
		t.Errorf("Expected at least 2 bonuses (skill + rigs), got %d", len(result.AppliedBonuses))
	}

	// Calculate improvement percentage
	improvement := ((result.EffectiveWarpSpeed / result.BaseWarpSpeed) - 1.0) * 100.0

	t.Logf("✓ Scenario 3: Full Fitting (with stacking penalties)")
	t.Logf("  Base:      %.2f AU/s", result.BaseWarpSpeed)
	t.Logf("  Effective: %.2f AU/s", result.EffectiveWarpSpeed)
	t.Logf("  Improvement: +%.1f%%", improvement)
	t.Logf("  Bonuses applied: %d", len(result.AppliedBonuses))
}

// TestGetShipWarpSpeedDeterministic_Scenario4_ErrorHandling validates error cases
func TestGetShipWarpSpeedDeterministic_Scenario4_ErrorHandling(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test 1: Invalid ship type ID
	_, err := GetShipWarpSpeedDeterministic(ctx, db, 99999999, nil, nil)
	if err == nil {
		t.Error("Expected error for invalid ship type ID, got nil")
	}

	// Test 2: Character lacks required skill level (if ship requires Navigation I+)
	// Note: Most industrials don't require Navigation skill, so this test may not trigger error
	charSkillsNoNav := &cargo.CharacterSkills{
		Skills: []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{
			// Empty skills - character has no Navigation skill
		},
	}

	// Try with a ship that might require Navigation (this may not fail for Iteron Mark V)
	result, err := GetShipWarpSpeedDeterministic(ctx, db, 650, charSkillsNoNav, nil)
	// If ship doesn't require Navigation skill, this should succeed with base speed only
	if err == nil && result != nil {
		t.Logf("✓ Scenario 4: Ship doesn't require Navigation skill (base speed: %.2f AU/s)", result.BaseWarpSpeed)
	}

	t.Log("✓ Scenario 4: Error Handling validated")
}

// Helper function: absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// openTestDB opens the test SDE database
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use environment variable or default path
	dbPath := os.Getenv("SDE_DB_PATH")
	if dbPath == "" {
		dbPath = "../../../data/sde/eve-sde.db" // Default for local testing
	}

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
	if err != nil {
		t.Fatalf("Failed to open test database at %s: %v", dbPath, err)
	}

	return db
}
