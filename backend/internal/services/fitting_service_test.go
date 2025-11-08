package services

import (
	"math"
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

// TestCalculateBonuses_CargoModules tests cargo bonus calculation with stacking penalties
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

	// With stacking penalties:
	// 1st: 2,500 × S(0) = 2,500.0
	// 2nd: 2,500 × S(1) = 2,500 × e^(-(1/2.67)^2) ≈ 2,500 × 0.8694 ≈ 2,173.5
	// Total: ≈ 4,673.5 m³ (not 5,000!)
	penalty2nd := math.Exp(-math.Pow(1.0/2.67, 2))
	expected := 2500.0 + 2500.0*penalty2nd

	assert.InDelta(t, expected, bonuses.CargoBonus, 1.0, "Cargo bonus with stacking penalties")
	assert.InDelta(t, 4673.5, bonuses.CargoBonus, 1.0, "2x Cargo ≈ 4,673 m³ (not 5,000!)")
	assert.Equal(t, 1.0, bonuses.WarpSpeedMultiplier, "No warp modules")
	assert.Equal(t, 1.0, bonuses.InertiaModifier, "No inertia modules")
}

// TestCalculateBonuses_WarpModules tests warp speed bonus calculation with stacking penalties
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

	// With stacking penalties:
	// 1st: 0.20 × S(0) = 0.20 × 1.0 = 0.20
	// 2nd: 0.20 × S(1) = 0.20 × 0.8694... ≈ 0.1738
	// 3rd: 0.20 × S(2) = 0.20 × 0.5707... ≈ 0.1141
	// Total: 0.20 + 0.1738 + 0.1141 ≈ 0.4879
	// Multiplier: 1 + 0.4879 = 1.4879
	expectedWarp := 1.4879
	assert.InDelta(t, expectedWarp, bonuses.WarpSpeedMultiplier, 0.001, "Warp multiplier with stacking penalties")
	assert.Equal(t, 0.0, bonuses.CargoBonus, "No cargo modules")
	assert.Equal(t, 1.0, bonuses.InertiaModifier, "No inertia modules")
}

// TestCalculateBonuses_InertiaModules tests inertia modifier calculation with stacking penalties
func TestCalculateBonuses_InertiaModules(t *testing.T) {
	service := &FittingService{}

	// 2x Inertial Stabilizers II (each -13% inertia)
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

	// With stacking penalties:
	// 1st: -0.13 × S(0) = -0.13 × 1.0 = -0.13
	// 2nd: -0.13 × S(1) = -0.13 × 0.8694... ≈ -0.113
	// Total: -0.13 + (-0.113) ≈ -0.243
	// Multiplier: 1 + (-0.243) = 0.757
	expectedInertia := 0.757
	assert.InDelta(t, expectedInertia, bonuses.InertiaModifier, 0.001, "Inertia modifier with stacking penalties")
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

	// Cargo: 2,500 × S(0) + 2,500 × S(1) = 2,500 + 2,173.5 ≈ 4,673.5
	assert.InDelta(t, 4673.5, bonuses.CargoBonus, 1.0, "Cargo with stacking penalties")
	// Warp: 1 + (0.20×S(0) + 0.20×S(1) + 0.20×S(2)) ≈ 1.4879
	assert.InDelta(t, 1.4879, bonuses.WarpSpeedMultiplier, 0.001, "Warp with stacking penalties")
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

// ==========================================
// Stacking Penalty Tests (EVE Online Formula)
// ==========================================

// TestApplyStackingPenalty_EmptyBonuses tests empty bonus array
func TestApplyStackingPenalty_EmptyBonuses(t *testing.T) {
	result := applyStackingPenalty([]float64{})
	assert.Equal(t, 0.0, result, "Empty bonuses should return 0")
}

// TestApplyStackingPenalty_SingleModule tests single module (100% effectiveness)
func TestApplyStackingPenalty_SingleModule(t *testing.T) {
	// Single +20% bonus should be 100% effective
	result := applyStackingPenalty([]float64{0.20})
	assert.Equal(t, 0.20, result, "Single module should have full effect")
}

// TestApplyStackingPenalty_TwoModules tests 2nd module at ~86.9% effectiveness
func TestApplyStackingPenalty_TwoModules(t *testing.T) {
	// 2x +20% bonuses
	// 1st: 0.20 × S(0) = 0.20 × 1.0 = 0.20
	// 2nd: 0.20 × S(1) = 0.20 × e^(-(1/2.67)^2) ≈ 0.20 × 0.8694 ≈ 0.1739
	// Total: ≈ 0.3739
	result := applyStackingPenalty([]float64{0.20, 0.20})
	
	// Calculate expected value
	penalty2nd := math.Exp(-math.Pow(1.0/2.67, 2))
	expected := 0.20 + 0.20*penalty2nd
	
	assert.InDelta(t, expected, result, 0.0001, "Two modules: 1st at 100%, 2nd at ~86.9%")
	assert.InDelta(t, 0.3739, result, 0.001, "Two 20% bonuses ≈ 37.39% total")
}

// TestApplyStackingPenalty_ThreeModules tests 3rd module at ~57.1% effectiveness
func TestApplyStackingPenalty_ThreeModules(t *testing.T) {
	// 3x +20% bonuses (Hyperspatial Rigs example)
	// 1st: 0.20 × S(0) = 0.20 × 1.0 = 0.20
	// 2nd: 0.20 × S(1) ≈ 0.1739
	// 3rd: 0.20 × S(2) = 0.20 × e^(-(2/2.67)^2) ≈ 0.20 × 0.5707 ≈ 0.1141
	// Total: ≈ 0.4880
	result := applyStackingPenalty([]float64{0.20, 0.20, 0.20})
	
	penalty2nd := math.Exp(-math.Pow(1.0/2.67, 2))
	penalty3rd := math.Exp(-math.Pow(2.0/2.67, 2))
	expected := 0.20 + 0.20*penalty2nd + 0.20*penalty3rd
	
	assert.InDelta(t, expected, result, 0.0001, "Three modules with correct penalties")
	assert.InDelta(t, 0.4880, result, 0.001, "Three 20% bonuses ≈ 48.8% total")
}

// TestApplyStackingPenalty_FourModules tests 4th module at ~28.3% effectiveness
func TestApplyStackingPenalty_FourModules(t *testing.T) {
	// 4x +10% bonuses
	// 1st: 0.10 × S(0) = 0.10
	// 2nd: 0.10 × S(1) ≈ 0.0869
	// 3rd: 0.10 × S(2) ≈ 0.0571
	// 4th: 0.10 × S(3) = 0.10 × e^(-(3/2.67)^2) ≈ 0.10 × 0.2830 ≈ 0.0283
	// Total: ≈ 0.2723
	result := applyStackingPenalty([]float64{0.10, 0.10, 0.10, 0.10})
	
	total := 0.0
	for i := 0; i < 4; i++ {
		penalty := math.Exp(-math.Pow(float64(i)/2.67, 2))
		total += 0.10 * penalty
	}
	
	assert.InDelta(t, total, result, 0.0001, "Four modules with correct penalties")
	assert.InDelta(t, 0.2723, result, 0.001, "Four 10% bonuses ≈ 27.23% total")
}

// TestApplyStackingPenalty_SortingOrder tests strongest-first sorting
func TestApplyStackingPenalty_SortingOrder(t *testing.T) {
	// Different bonus strengths: 10%, 20%, 15%
	// Should sort to: 20%, 15%, 10%
	// 1st: 0.20 × S(0) = 0.20
	// 2nd: 0.15 × S(1) ≈ 0.1304
	// 3rd: 0.10 × S(2) ≈ 0.0571
	result := applyStackingPenalty([]float64{0.10, 0.20, 0.15})
	
	penalty2nd := math.Exp(-math.Pow(1.0/2.67, 2))
	penalty3rd := math.Exp(-math.Pow(2.0/2.67, 2))
	expected := 0.20 + 0.15*penalty2nd + 0.10*penalty3rd
	
	assert.InDelta(t, expected, result, 0.0001, "Strongest bonuses should apply first")
	assert.InDelta(t, 0.3875, result, 0.001, "Mixed bonuses with correct order")
}

// TestApplyStackingPenalty_NegativeBonuses tests negative bonuses (e.g., inertia reduction)
func TestApplyStackingPenalty_NegativeBonuses(t *testing.T) {
	// 2x -13% inertia reduction
	// 1st: -0.13 × S(0) = -0.13
	// 2nd: -0.13 × S(1) ≈ -0.1130
	// Total: ≈ -0.243
	result := applyStackingPenalty([]float64{-0.13, -0.13})
	
	penalty2nd := math.Exp(-math.Pow(1.0/2.67, 2))
	expected := -0.13 + (-0.13)*penalty2nd
	
	assert.InDelta(t, expected, result, 0.0001, "Negative bonuses with penalties")
	assert.InDelta(t, -0.243, result, 0.001, "Two -13% bonuses ≈ -24.3% total")
}

// TestApplyStackingPenalty_RealWorldExample tests real EVE fitting
func TestApplyStackingPenalty_RealWorldExample(t *testing.T) {
	// Real scenario: 3x Hyperspatial Velocity Optimizer I
	// Each gives +20% warp speed
	// Expected: +48.8% total (NOT +72.8% without penalties!)
	bonuses := []float64{0.20, 0.20, 0.20}
	result := applyStackingPenalty(bonuses)
	
	// Verify against EVE University formula
	penalty1 := math.Exp(-math.Pow(0.0/2.67, 2)) // 1.0
	penalty2 := math.Exp(-math.Pow(1.0/2.67, 2)) // ~0.8694
	penalty3 := math.Exp(-math.Pow(2.0/2.67, 2)) // ~0.5707
	
	expected := 0.20*penalty1 + 0.20*penalty2 + 0.20*penalty3
	
	assert.InDelta(t, expected, result, 0.0001, "Real-world example matches EVE formula")
	assert.InDelta(t, 0.4880, result, 0.001, "3x Hyperspatial = +48.8% (not +72.8%!)")
	
	// WITHOUT stacking penalties (wrong calculation):
	wrongResult := 0.20 + 0.20 + 0.20 // 0.60 = +60%
	assert.NotEqual(t, wrongResult, result, "Stacking penalties must apply!")
}

// TestCalculateBonuses_CargoStacking tests cargo modules with stacking penalties
func TestCalculateBonuses_CargoStacking(t *testing.T) {
	service := &FittingService{}

	// 3x Expanded Cargohold II (each +2,500 m³)
	modules := []FittedModule{
		{TypeID: 1319, TypeName: "Expanded Cargohold II", Slot: "LoSlot0", DogmaAttribs: map[int]float64{38: 2500.0}},
		{TypeID: 1319, TypeName: "Expanded Cargohold II", Slot: "LoSlot1", DogmaAttribs: map[int]float64{38: 2500.0}},
		{TypeID: 1319, TypeName: "Expanded Cargohold II", Slot: "LoSlot2", DogmaAttribs: map[int]float64{38: 2500.0}},
	}

	bonuses := service.calculateBonuses(modules)

	// With stacking penalties:
	// 1st: 2,500 × S(0) = 2,500.0
	// 2nd: 2,500 × S(1) ≈ 2,173.5
	// 3rd: 2,500 × S(2) ≈ 1,426.75
	// Total: ≈ 6,100.25 (not 7,500!)
	penalty2 := math.Exp(-math.Pow(1.0/2.67, 2))
	penalty3 := math.Exp(-math.Pow(2.0/2.67, 2))
	expected := 2500.0 + 2500.0*penalty2 + 2500.0*penalty3
	
	assert.InDelta(t, expected, bonuses.CargoBonus, 1.0, "Cargo with stacking penalties")
	assert.InDelta(t, 6100.25, bonuses.CargoBonus, 10.0, "3x Cargo ≈ 6,100 m³ (not 7,500!)")
}

// TestGetCharacterFitting_Integration would be an integration test
// (requires Redis, SDE DB, ESI client) - see fitting_service_integration_test.go
