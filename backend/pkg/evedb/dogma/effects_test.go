package dogma

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	_ "github.com/mattn/go-sqlite3"
)

// TestGetModuleEffects_ExpandedCargohold validates deterministic module effect derivation
func TestGetModuleEffects_ExpandedCargohold(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	// Expanded Cargohold I (Type 1317)
	effect, err := GetModuleEffects(db, 1317)
	if err != nil {
		t.Fatalf("GetModuleEffects failed: %v", err)
	}

	// Validate type info
	if effect.TypeID != 1317 {
		t.Errorf("Expected type ID 1317, got %d", effect.TypeID)
	}

	// Validate attributes (Attribute 149 = cargoCapacityMultiplier)
	expectedValue := 1.175 // +17.5% bonus
	if val, ok := effect.Attributes[149]; !ok {
		t.Error("Expected attribute 149 (cargoCapacityMultiplier) not found")
	} else if val != expectedValue {
		t.Errorf("Expected attribute 149 value %.3f, got %.3f", expectedValue, val)
	}

	// Validate stackable
	if !effect.IsStackable {
		t.Error("Expected Expanded Cargohold I to be stackable")
	}

	// Validate effects contain cargo modifier
	cargoMods := FindCargoModifiers(effect)
	if len(cargoMods) == 0 {
		t.Fatal("Expected cargo modifiers, found none")
	}

	// Check operation code
	mod := cargoMods[0]
	if mod.Operation != 4 {
		t.Errorf("Expected operation code 4 (PostPercent), got %d", mod.Operation)
	}

	if mod.ModifiedAttributeID != 38 {
		t.Errorf("Expected modified attribute 38 (capacity), got %d", mod.ModifiedAttributeID)
	}

	if mod.ModifyingAttributeID != 149 {
		t.Errorf("Expected modifying attribute 149, got %d", mod.ModifyingAttributeID)
	}

	t.Logf("✅ Expanded Cargohold I effect deterministically derived:")
	t.Logf("   Attribute 149: %.3f (+17.5%% bonus)", effect.Attributes[149])
	t.Logf("   Operation Code: %d (PostPercent)", mod.Operation)
	t.Logf("   Stackable: %v", effect.IsStackable)
}

// TestGetModuleEffects_MediumRig validates deterministic rig effect derivation
func TestGetModuleEffects_MediumRig(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	// Medium Cargohold Optimization I (Type 31119)
	effect, err := GetModuleEffects(db, 31119)
	if err != nil {
		t.Fatalf("GetModuleEffects failed: %v", err)
	}

	// Validate type info
	if effect.TypeID != 31119 {
		t.Errorf("Expected type ID 31119, got %d", effect.TypeID)
	}

	// Validate attributes (Attribute 614 = cargoCapacityBonus)
	expectedValue := 15.0 // +15% bonus
	if val, ok := effect.Attributes[614]; !ok {
		t.Error("Expected attribute 614 (cargoCapacityBonus) not found")
	} else if val != expectedValue {
		t.Errorf("Expected attribute 614 value %.1f, got %.1f", expectedValue, val)
	}

	// Validate stackable
	if !effect.IsStackable {
		t.Error("Expected Medium Cargohold Optimization I to be stackable")
	}

	// Validate effects contain cargo modifier
	cargoMods := FindCargoModifiers(effect)
	if len(cargoMods) == 0 {
		t.Fatal("Expected cargo modifiers, found none")
	}

	// Check operation code (should be 6 for rigs)
	mod := cargoMods[0]
	if mod.Operation != 6 {
		t.Errorf("Expected operation code 6 (PostPercent), got %d", mod.Operation)
	}

	t.Logf("✅ Medium Cargohold Optimization I effect deterministically derived:")
	t.Logf("   Attribute 614: %.1f (+15%% bonus)", effect.Attributes[614])
	t.Logf("   Operation Code: %d (PostPercent)", mod.Operation)
	t.Logf("   Stackable: %v", effect.IsStackable)
}

// TestApplyModifier_Operation4 validates PostPercent operation (modules)
func TestApplyModifier_Operation4(t *testing.T) {
	baseCapacity := 2700.0
	modifier := ModifierInfo{
		Operation:            4,
		ModifiedAttributeID:  38,
		ModifyingAttributeID: 149,
	}
	modifierValue := 1.175 // +17.5%

	// 1 module
	result1 := ApplyModifier(baseCapacity, modifier, modifierValue, 1)
	expected1 := 2700.0 * 1.175
	if result1 != expected1 {
		t.Errorf("1 module: expected %.2f, got %.2f", expected1, result1)
	}

	// 5 modules (stacked)
	result5 := ApplyModifier(baseCapacity, modifier, modifierValue, 5)
	expected5 := 6047.18 // 2700.0 × 1.175^5 (actual calculation)
	if !almostEqual(result5, expected5, 1.0) {
		t.Errorf("5 modules: expected %.2f, got %.2f", expected5, result5)
	}

	t.Logf("✅ Operation 4 (PostPercent) calculation:")
	t.Logf("   Base: %.1f m³", baseCapacity)
	t.Logf("   1 Module: %.2f m³", result1)
	t.Logf("   5 Modules: %.2f m³", result5)
}

// TestApplyModifier_Operation6 validates PostPercent operation (rigs)
func TestApplyModifier_Operation6(t *testing.T) {
	baseCapacity := 2700.0
	modifier := ModifierInfo{
		Operation:            6,
		ModifiedAttributeID:  38,
		ModifyingAttributeID: 614,
	}
	modifierValue := 15.0 // +15%

	// 3 rigs
	result3 := ApplyModifier(baseCapacity, modifier, modifierValue, 3)
	expected3 := 2700.0 * 1.520875 // (1.15)^3
	if !almostEqual(result3, expected3, 0.01) {
		t.Errorf("3 rigs: expected %.2f, got %.2f", expected3, result3)
	}

	t.Logf("✅ Operation 6 (PostPercent) calculation:")
	t.Logf("   Base: %.1f m³", baseCapacity)
	t.Logf("   3 Rigs: %.2f m³", result3)
}

// TestCalculateCargoBonus_Combined validates full workflow
func TestCalculateCargoBonus_Combined(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer db.Close()

	baseCapacity := 2700.0

	// Get module effects
	module, err := GetModuleEffects(db, 1317) // Expanded Cargohold I
	if err != nil {
		t.Fatalf("GetModuleEffects failed: %v", err)
	}

	rig, err := GetModuleEffects(db, 31119) // Medium Cargohold Optimization I
	if err != nil {
		t.Fatalf("GetModuleEffects failed: %v", err)
	}

	// Simulate 5 modules + 3 rigs
	modules := map[int64][]ModuleEffect{
		1317:  {*module, *module, *module, *module, *module}, // 5×
		31119: {*rig, *rig, *rig},                            // 3×
	}

	finalCapacity := CalculateCargoBonus(baseCapacity, modules)

	// Expected: 2700 × (1.175)^5 × (1.15)^3
	// Actual calculation: 6047.18 × 1.520875 = 9197.01
	expected := 9197.01

	if !almostEqual(finalCapacity, expected, 1.0) {
		t.Errorf("Expected final capacity ~%.2f m³, got %.2f m³", expected, finalCapacity)
	}

	t.Logf("✅ Combined cargo bonus calculation:")
	t.Logf("   Base: %.1f m³", baseCapacity)
	t.Logf("   + 5× Expanded Cargohold I: %.2f m³", 6047.18)
	t.Logf("   + 3× Medium Cargohold Optimization I: %.2f m³", finalCapacity)
	t.Logf("   Final: %.2f m³", finalCapacity)
}

// Helper: almostEqual checks float equality with tolerance
func almostEqual(a, b, tolerance float64) bool {
	if a == b {
		return true
	}
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
