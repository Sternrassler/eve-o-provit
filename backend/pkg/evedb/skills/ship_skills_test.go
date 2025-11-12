package skills

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	_ "github.com/mattn/go-sqlite3"
)

// TestGetShipCargoSkills_Nereus validates deterministic skill derivation for Nereus
func TestGetShipCargoSkills_Nereus(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	result, err := GetShipCargoSkills(db, 650) // Nereus
	if err != nil {
		t.Fatalf("GetShipCargoSkills failed: %v", err)
	}

	// Validate ship info
	if result.ShipTypeID != 650 {
		t.Errorf("Expected ship type ID 650, got %d", result.ShipTypeID)
	}

	if result.ShipName != "Nereus" {
		t.Errorf("Expected ship name 'Nereus', got %s", result.ShipName)
	}

	// Validate base capacity (Attribute 38)
	expectedBase := 2700.0
	if result.BaseCapacity != expectedBase {
		t.Errorf("Expected base capacity %.1f m³, got %.1f m³", expectedBase, result.BaseCapacity)
	}

	// Validate required skills
	if len(result.Skills) == 0 {
		t.Fatal("Expected at least one required skill, got none")
	}

	// Find Gallente Hauler skill (Type 3340)
	var gallenteHauler *ShipSkillRequirement
	for i := range result.Skills {
		if result.Skills[i].SkillTypeID == 3340 {
			gallenteHauler = &result.Skills[i]
			break
		}
	}

	if gallenteHauler == nil {
		t.Fatal("Expected Gallente Hauler skill (3340) in requirements, not found")
	}

	// Validate minimum level
	if gallenteHauler.MinimumLevel != 1 {
		t.Errorf("Expected minimum level 1, got %d", gallenteHauler.MinimumLevel)
	}

	// Validate bonus per level
	expectedBonus := 5.0 // 5% per level
	if gallenteHauler.BonusPerLevel != expectedBonus {
		t.Errorf("Expected bonus per level %.1f%%, got %.1f%%", expectedBonus, gallenteHauler.BonusPerLevel)
	}

	// Validate attribute ID
	if gallenteHauler.AttributeID != 496 {
		t.Errorf("Expected attribute ID 496, got %d", gallenteHauler.AttributeID)
	}

	t.Logf("✅ Nereus skills deterministically derived from SDE:")
	t.Logf("   Base Capacity: %.1f m³", result.BaseCapacity)
	t.Logf("   Required Skill: %d (Level %d)", gallenteHauler.SkillTypeID, gallenteHauler.MinimumLevel)
	t.Logf("   Bonus: %.1f%% per level", gallenteHauler.BonusPerLevel)
}

// TestGetBaseCapacity validates simple capacity query
func TestGetBaseCapacity(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	capacity, err := GetBaseCapacity(db, 650) // Nereus
	if err != nil {
		t.Fatalf("GetBaseCapacity failed: %v", err)
	}

	expected := 2700.0
	if capacity != expected {
		t.Errorf("Expected capacity %.1f m³, got %.1f m³", expected, capacity)
	}

	t.Logf("✅ Base capacity: %.1f m³", capacity)
}

// TestGetBaseCapacity_NotFound validates error handling
func TestGetBaseCapacity_NotFound(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	_, err := GetBaseCapacity(db, 99999999)
	if err == nil {
		t.Fatal("Expected error for non-existent ship, got nil")
	}

	t.Logf("✅ Error handling works: %v", err)
}
