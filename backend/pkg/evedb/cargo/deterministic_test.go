package cargo

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	_ "github.com/mattn/go-sqlite3"
)

// TestGetShipCapacitiesDeterministic_Nereus_Scenario1 validates minimum skill (Issue #77 Scenario 1)
func TestGetShipCapacitiesDeterministic_Nereus_Scenario1(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	// Nereus with Gallente Hauler I (minimum)
	charSkills := &CharacterSkills{
		Skills: []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{
			{SkillID: 3340, TrainedSkillLevel: 1}, // Gallente Hauler I
		},
	}

	result, err := GetShipCapacitiesDeterministic(context.Background(), db, 650, charSkills, nil)
	if err != nil {
		t.Fatalf("GetShipCapacitiesDeterministic failed: %v", err)
	}

	// Expected: 2700 × 1.05 = 2835 m³
	expectedEffective := 2835.0
	if !almostEqual(result.EffectiveCargoHold, expectedEffective, 0.1) {
		t.Errorf("Expected %.1f m³, got %.1f m³", expectedEffective, result.EffectiveCargoHold)
	}

	t.Logf("✅ Scenario 1: Nereus + Gallente Hauler I → %.1f m³", result.EffectiveCargoHold)
}

// TestGetShipCapacitiesDeterministic_Nereus_Scenario2 validates trained skill (Issue #77 Scenario 2)
func TestGetShipCapacitiesDeterministic_Nereus_Scenario2(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	// Nereus with Gallente Hauler V (trained!)
	charSkills := &CharacterSkills{
		Skills: []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{
			{SkillID: 3340, TrainedSkillLevel: 5}, // Gallente Hauler V
		},
	}

	result, err := GetShipCapacitiesDeterministic(context.Background(), db, 650, charSkills, nil)
	if err != nil {
		t.Fatalf("GetShipCapacitiesDeterministic failed: %v", err)
	}

	// Expected: 2700 × 1.25 = 3375 m³
	expectedEffective := 3375.0
	if !almostEqual(result.EffectiveCargoHold, expectedEffective, 0.1) {
		t.Errorf("Expected %.1f m³, got %.1f m³", expectedEffective, result.EffectiveCargoHold)
	}

	t.Logf("✅ Scenario 2: Nereus + Gallente Hauler V → %.1f m³", result.EffectiveCargoHold)
}

// TestGetShipCapacitiesDeterministic_Nereus_Scenario3 validates full fitting (Issue #77 Scenario 3)
func TestGetShipCapacitiesDeterministic_Nereus_Scenario3(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	// Nereus with Gallente Hauler I + 5× Module + 3× Rigs
	charSkills := &CharacterSkills{
		Skills: []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{
			{SkillID: 3340, TrainedSkillLevel: 1}, // Gallente Hauler I
		},
	}

	fittedItems := []FittedItem{
		{TypeID: 1317, Slot: "LoSlot0"}, // Expanded Cargohold I
		{TypeID: 1317, Slot: "LoSlot1"},
		{TypeID: 1317, Slot: "LoSlot2"},
		{TypeID: 1317, Slot: "LoSlot3"},
		{TypeID: 1317, Slot: "LoSlot4"},
		{TypeID: 31119, Slot: "RigSlot0"}, // Medium Cargohold Optimization I
		{TypeID: 31119, Slot: "RigSlot1"},
		{TypeID: 31119, Slot: "RigSlot2"},
	}

	result, err := GetShipCapacitiesDeterministic(context.Background(), db, 650, charSkills, fittedItems)
	if err != nil {
		t.Fatalf("GetShipCapacitiesDeterministic failed: %v", err)
	}

	// Expected: 2700 × 1.05 × 1.175^5 × 1.15^3 = 9656.9 m³
	expectedEffective := 9656.9
	if !almostEqual(result.EffectiveCargoHold, expectedEffective, 10.0) {
		t.Errorf("Expected ~%.1f m³, got %.1f m³", expectedEffective, result.EffectiveCargoHold)
	}

	// Validate all bonuses
	if len(result.AppliedBonuses) != 3 {
		t.Errorf("Expected 3 bonuses, got %d", len(result.AppliedBonuses))
	}

	t.Logf("✅ Scenario 3: Nereus + Full Fitting → %.1f m³ (Expected: %.1f m³)",
		result.EffectiveCargoHold, expectedEffective)
	for _, bonus := range result.AppliedBonuses {
		t.Logf("   %s: %s (×%d, op=%d, val=%.2f)",
			bonus.Source, bonus.Name, bonus.Count, bonus.Operation, bonus.Value)
	}
}

// TestGetShipCapacitiesDeterministic_Nereus_Scenario4 validates error handling (Issue #77 Scenario 4)
func TestGetShipCapacitiesDeterministic_Nereus_Scenario4(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	// Character without required skill
	charSkills := &CharacterSkills{
		Skills: []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{
			// No skill 3340
		},
	}

	_, err := GetShipCapacitiesDeterministic(context.Background(), db, 650, charSkills, nil)
	if err == nil {
		t.Fatal("Expected error for missing skill, got nil")
	}

	t.Logf("✅ Scenario 4: Error handling → %v", err)
}

// Helper: almostEqual checks float equality with tolerance
func almostEqual(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
