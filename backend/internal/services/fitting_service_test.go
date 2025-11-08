package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsFittedSlot tests the isFittedSlot helper function
func TestIsFittedSlot(t *testing.T) {
	tests := []struct {
		name         string
		locationFlag string
		want         bool
	}{
		// High slots
		{"HiSlot0", "HiSlot0", true},
		{"HiSlot7", "HiSlot7", true},
		
		// Med slots
		{"MedSlot0", "MedSlot0", true},
		{"MedSlot7", "MedSlot7", true},
		
		// Low slots
		{"LoSlot0", "LoSlot0", true},
		{"LoSlot7", "LoSlot7", true},
		
		// Rig slots
		{"RigSlot0", "RigSlot0", true},
		{"RigSlot2", "RigSlot2", true},
		
		// Not fitted slots
		{"Hangar", "Hangar", false},
		{"Cargo", "Cargo", false},
		{"DroneBay", "DroneBay", false},
		{"Invalid", "Invalid", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isFittedSlot(tt.locationFlag)
			assert.Equal(t, tt.want, got, "isFittedSlot(%q) = %v, want %v", tt.locationFlag, got, tt.want)
		})
	}
}

// TestCalculateBonuses_EmptyModules tests bonus calculation with no modules
func TestCalculateBonuses_EmptyModules(t *testing.T) {
	service := &FittingService{}
	
	modules := []FittedModule{}
	bonuses := service.calculateBonuses(modules)
	
	assert.Equal(t, 0.0, bonuses.CargoBonus, "Empty modules should have 0 cargo bonus")
	assert.Equal(t, 1.0, bonuses.WarpSpeedMultiplier, "Empty modules should have 1.0 warp multiplier")
	assert.Equal(t, 1.0, bonuses.InertiaModifier, "Empty modules should have 1.0 inertia modifier")
}

// TestCalculateBonuses_CargoModules tests cargo bonus calculation (ADDITIVE)
func TestCalculateBonuses_CargoModules(t *testing.T) {
	service := &FittingService{}
	
	// 2x Expanded Cargohold II (each +2,500 m³)
	modules := []FittedModule{
		{
			TypeID:   1319, // Expanded Cargohold II
			TypeName: "Expanded Cargohold II",
			Slot:     "LoSlot0",
			DogmaAttribs: map[int]float64{
				38: 2500.0, // Cargo capacity bonus
			},
		},
		{
			TypeID:   1319,
			TypeName: "Expanded Cargohold II",
			Slot:     "LoSlot1",
			DogmaAttribs: map[int]float64{
				38: 2500.0,
			},
		},
	}
	
	bonuses := service.calculateBonuses(modules)
	
	// Cargo bonus should be additive: 2,500 + 2,500 = 5,000
	assert.Equal(t, 5000.0, bonuses.CargoBonus, "Cargo bonus should be additive")
	assert.Equal(t, 1.0, bonuses.WarpSpeedMultiplier, "No warp modules")
	assert.Equal(t, 1.0, bonuses.InertiaModifier, "No inertia modules")
}

// TestCalculateBonuses_WarpModules tests warp speed bonus calculation (MULTIPLICATIVE)
func TestCalculateBonuses_WarpModules(t *testing.T) {
	service := &FittingService{}
	
	// 3x Hyperspatial Velocity Optimizer I (each +20% warp speed)
	modules := []FittedModule{
		{
			TypeID:   31370,
			TypeName: "Hyperspatial Velocity Optimizer I",
			Slot:     "RigSlot0",
			DogmaAttribs: map[int]float64{
				20: 0.20, // 20% warp speed increase
			},
		},
		{
			TypeID:   31370,
			TypeName: "Hyperspatial Velocity Optimizer I",
			Slot:     "RigSlot1",
			DogmaAttribs: map[int]float64{
				20: 0.20,
			},
		},
		{
			TypeID:   31370,
			TypeName: "Hyperspatial Velocity Optimizer I",
			Slot:     "RigSlot2",
			DogmaAttribs: map[int]float64{
				20: 0.20,
			},
		},
	}
	
	bonuses := service.calculateBonuses(modules)
	
	// Warp multiplier should be multiplicative: 1.2 × 1.2 × 1.2 = 1.728
	expectedWarp := 1.2 * 1.2 * 1.2
	assert.InDelta(t, expectedWarp, bonuses.WarpSpeedMultiplier, 0.001, "Warp multiplier should be multiplicative")
	assert.Equal(t, 0.0, bonuses.CargoBonus, "No cargo modules")
	assert.Equal(t, 1.0, bonuses.InertiaModifier, "No inertia modules")
}

// TestCalculateBonuses_InertiaModules tests inertia modifier calculation (MULTIPLICATIVE)
func TestCalculateBonuses_InertiaModules(t *testing.T) {
	service := &FittingService{}
	
	// 2x Inertial Stabilizers II (each -13% inertia = 0.87 multiplier)
	modules := []FittedModule{
		{
			TypeID:   5331,
			TypeName: "Inertial Stabilizers II",
			Slot:     "LoSlot0",
			DogmaAttribs: map[int]float64{
				70: -0.13, // -13% inertia
			},
		},
		{
			TypeID:   5331,
			TypeName: "Inertial Stabilizers II",
			Slot:     "LoSlot1",
			DogmaAttribs: map[int]float64{
				70: -0.13,
			},
		},
	}
	
	bonuses := service.calculateBonuses(modules)
	
	// Inertia modifier should be multiplicative: 0.87 × 0.87 = 0.7569
	expectedInertia := 0.87 * 0.87
	assert.InDelta(t, expectedInertia, bonuses.InertiaModifier, 0.001, "Inertia modifier should be multiplicative")
	assert.Equal(t, 0.0, bonuses.CargoBonus, "No cargo modules")
	assert.Equal(t, 1.0, bonuses.WarpSpeedMultiplier, "No warp modules")
}

// TestCalculateBonuses_MixedModules tests combined bonus calculation
func TestCalculateBonuses_MixedModules(t *testing.T) {
	service := &FittingService{}
	
	// Realistic Badger II fitting: 2x Cargo + 3x Hyperspatial Rigs
	modules := []FittedModule{
		// Cargo modules
		{
			TypeID:   1319,
			TypeName: "Expanded Cargohold II",
			Slot:     "LoSlot0",
			DogmaAttribs: map[int]float64{
				38: 2500.0,
			},
		},
		{
			TypeID:   1319,
			TypeName: "Expanded Cargohold II",
			Slot:     "LoSlot1",
			DogmaAttribs: map[int]float64{
				38: 2500.0,
			},
		},
		// Warp rigs
		{
			TypeID:   31370,
			TypeName: "Hyperspatial Velocity Optimizer I",
			Slot:     "RigSlot0",
			DogmaAttribs: map[int]float64{
				20: 0.20,
			},
		},
		{
			TypeID:   31370,
			TypeName: "Hyperspatial Velocity Optimizer I",
			Slot:     "RigSlot1",
			DogmaAttribs: map[int]float64{
				20: 0.20,
			},
		},
		{
			TypeID:   31370,
			TypeName: "Hyperspatial Velocity Optimizer I",
			Slot:     "RigSlot2",
			DogmaAttribs: map[int]float64{
				20: 0.20,
			},
		},
	}
	
	bonuses := service.calculateBonuses(modules)
	
	assert.Equal(t, 5000.0, bonuses.CargoBonus, "Cargo bonus: 2,500 + 2,500")
	assert.InDelta(t, 1.728, bonuses.WarpSpeedMultiplier, 0.001, "Warp: 1.2³")
	assert.Equal(t, 1.0, bonuses.InertiaModifier, "No inertia modules")
}

// TestGetDefaultFitting tests graceful degradation
func TestGetDefaultFitting(t *testing.T) {
	service := &FittingService{}
	
	fitting := service.getDefaultFitting(648) // Badger type ID
	
	assert.Equal(t, 648, fitting.ShipTypeID, "Ship type ID should match")
	assert.Empty(t, fitting.FittedModules, "Should have no modules")
	assert.Equal(t, 0.0, fitting.Bonuses.CargoBonus, "Default cargo bonus = 0")
	assert.Equal(t, 1.0, fitting.Bonuses.WarpSpeedMultiplier, "Default warp = 1.0")
	assert.Equal(t, 1.0, fitting.Bonuses.InertiaModifier, "Default inertia = 1.0")
}

// TestCalculateBonuses_NoRelevantAttributes tests modules without relevant dogma attributes
func TestCalculateBonuses_NoRelevantAttributes(t *testing.T) {
	service := &FittingService{}
	
	// Module with irrelevant attributes (e.g., weapons)
	modules := []FittedModule{
		{
			TypeID:   2488, // Light Missile Launcher I
			TypeName: "Light Missile Launcher I",
			Slot:     "HiSlot0",
			DogmaAttribs: map[int]float64{
				// No cargo/warp/inertia attributes
				50: 500.0, // Some other attribute
			},
		},
	}
	
	bonuses := service.calculateBonuses(modules)
	
	assert.Equal(t, 0.0, bonuses.CargoBonus, "No cargo bonus")
	assert.Equal(t, 1.0, bonuses.WarpSpeedMultiplier, "No warp bonus")
	assert.Equal(t, 1.0, bonuses.InertiaModifier, "No inertia bonus")
}

// TestGetCharacterFitting_Integration would be an integration test
// (requires Redis, SDE DB, ESI client) - see fitting_service_integration_test.go
