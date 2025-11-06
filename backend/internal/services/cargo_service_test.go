package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSkillsService is a mock implementation of SkillsServicer for testing
type mockSkillsService struct {
	skills *TradingSkills
	err    error
}

func (m *mockSkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.skills != nil {
		return m.skills, nil
	}
	// Default: no skills
	return &TradingSkills{}, nil
}

// TestCargoService_CalculateCapacity tests cargo capacity calculation with various skill levels
func TestCargoService_CalculateCapacity(t *testing.T) {
	service := NewCargoService(&mockSkillsService{})
	baseCapacity := 1000.0

	tests := []struct {
		name             string
		skills           *TradingSkills
		expectedCapacity float64
		expectedBonusPct float64
	}{
		{
			name: "No skills",
			skills: &TradingSkills{
				SpaceshipCommand:   0,
				GallenteIndustrial: 0,
			},
			expectedCapacity: 1000.0,
			expectedBonusPct: 0.0,
		},
		{
			name: "Spaceship Command V only",
			skills: &TradingSkills{
				SpaceshipCommand:   5,
				GallenteIndustrial: 0,
			},
			expectedCapacity: 1250.0, // 1000 * 1.25
			expectedBonusPct: 25.0,
		},
		{
			name: "Gallente Industrial V only",
			skills: &TradingSkills{
				SpaceshipCommand:   0,
				GallenteIndustrial: 5,
			},
			expectedCapacity: 1250.0, // 1000 * 1.25
			expectedBonusPct: 25.0,
		},
		{
			name: "Spaceship Command V + Gallente Industrial V",
			skills: &TradingSkills{
				SpaceshipCommand:   5,
				GallenteIndustrial: 5,
			},
			expectedCapacity: 1562.5, // 1000 * 1.25 * 1.25
			expectedBonusPct: 50.0,
		},
		{
			name: "Spaceship Command III + Caldari Industrial IV",
			skills: &TradingSkills{
				SpaceshipCommand: 3,
				CaldarIndustrial: 4,
			},
			expectedCapacity: 1380.0, // 1000 * 1.15 * 1.20
			expectedBonusPct: 35.0,
		},
		{
			name: "Multiple racial skills - uses highest",
			skills: &TradingSkills{
				SpaceshipCommand:   5,
				GallenteIndustrial: 3,
				CaldarIndustrial:   5, // Highest
				AmarrIndustrial:    2,
				MinmatarIndustrial: 4,
			},
			expectedCapacity: 1562.5, // Uses Caldari V (highest)
			expectedBonusPct: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capacity, bonusPct := service.CalculateCargoCapacity(baseCapacity, tt.skills)
			assert.InDelta(t, tt.expectedCapacity, capacity, 0.1,
				"Expected capacity %.2f, got %.2f", tt.expectedCapacity, capacity)
			assert.InDelta(t, tt.expectedBonusPct, bonusPct, 0.1,
				"Expected bonus %.2f%%, got %.2f%%", tt.expectedBonusPct, bonusPct)
		})
	}
}

// TestCargoService_KnapsackDP tests the knapsack optimization algorithm
func TestCargoService_KnapsackDP(t *testing.T) {
	service := NewCargoService(&mockSkillsService{})

	t.Run("Simple optimal selection", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 10, Value: 100, Quantity: 5}, // value/vol = 10
			{TypeID: 2, Volume: 20, Value: 150, Quantity: 3}, // value/vol = 7.5
			{TypeID: 3, Volume: 5, Value: 60, Quantity: 10},  // value/vol = 12 (best)
		}

		solution := service.KnapsackDP(items, 50)

		// Optimal: 10x Item3 (50m³, 600 ISK)
		assert.Equal(t, 600.0, solution.TotalValue, "Should select highest value/volume items")
		assert.LessOrEqual(t, solution.TotalVolume, 50.0, "Should not exceed capacity")
		assert.Greater(t, solution.UsedCapacity, 95.0, "Should use most of capacity")
	})

	t.Run("Mixed quantities", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 15, Value: 200, Quantity: 2}, // value/vol = 13.33
			{TypeID: 2, Volume: 10, Value: 120, Quantity: 3}, // value/vol = 12
			{TypeID: 3, Volume: 5, Value: 50, Quantity: 5},   // value/vol = 10
		}

		solution := service.KnapsackDP(items, 50)

		// Should maximize value within 50m³ capacity
		assert.LessOrEqual(t, solution.TotalVolume, 50.0)
		assert.Greater(t, solution.TotalValue, 0.0)
		assert.Greater(t, len(solution.Items), 0, "Should select at least one item")
	})

	t.Run("Empty items", func(t *testing.T) {
		items := []CargoItem{}
		solution := service.KnapsackDP(items, 100)

		assert.Equal(t, 0.0, solution.TotalValue)
		assert.Equal(t, 0.0, solution.TotalVolume)
		assert.Equal(t, 0, len(solution.Items))
	})

	t.Run("Zero capacity", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 10, Value: 100, Quantity: 5},
		}
		solution := service.KnapsackDP(items, 0)

		assert.Equal(t, 0.0, solution.TotalValue)
		assert.Equal(t, 0.0, solution.TotalVolume)
		assert.Equal(t, 0, len(solution.Items))
	})

	t.Run("Item too large for capacity", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 100, Value: 1000, Quantity: 5},
		}
		solution := service.KnapsackDP(items, 50)

		assert.Equal(t, 0.0, solution.TotalValue, "Should not fit any items")
		assert.Equal(t, 0.0, solution.TotalVolume)
	})

	t.Run("Exact fit", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 25, Value: 100, Quantity: 2},
		}
		solution := service.KnapsackDP(items, 50)

		assert.Equal(t, 200.0, solution.TotalValue, "Should fit exactly 2 items")
		assert.Equal(t, 50.0, solution.TotalVolume)
		assert.Equal(t, 100.0, solution.UsedCapacity)
	})

	t.Run("Multiple quantities of same item", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 5, Value: 100, Quantity: 20}, // Can fit 10 in 50m³
		}
		solution := service.KnapsackDP(items, 50)

		assert.Equal(t, 1000.0, solution.TotalValue, "Should take 10 items")
		assert.Equal(t, 50.0, solution.TotalVolume)
		assert.Equal(t, 1, len(solution.Items), "Should be one entry with quantity 10")
	})

	t.Run("Fractional volumes", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 0.5, Value: 10, Quantity: 100}, // Small items
			{TypeID: 2, Volume: 1.25, Value: 30, Quantity: 50}, // Fractional volume
		}
		solution := service.KnapsackDP(items, 50)

		assert.LessOrEqual(t, solution.TotalVolume, 50.0)
		assert.Greater(t, solution.TotalValue, 0.0)
	})
}

// TestCargoService_OptimizeCargo tests the complete optimization workflow
func TestCargoService_OptimizeCargo(t *testing.T) {
	t.Run("With skills - full workflow", func(t *testing.T) {
		mockSkills := &mockSkillsService{
			skills: &TradingSkills{
				SpaceshipCommand:   5,
				GallenteIndustrial: 5,
			},
		}
		service := NewCargoService(mockSkills)

		items := []CargoItem{
			{TypeID: 1, Volume: 10, Value: 100, Quantity: 20},
		}

		solution, err := service.OptimizeCargo(
			context.Background(),
			12345,
			"test-token",
			1000.0, // Base capacity
			items,
		)

		require.NoError(t, err)
		assert.Equal(t, 1000.0, solution.BaseCapacity)
		assert.Equal(t, 1562.5, solution.EffectiveCapacity, "Should apply skill bonuses")
		assert.Equal(t, 50.0, solution.CapacityBonusPercent)
		assert.Greater(t, solution.TotalValue, 0.0)
	})

	t.Run("Without skills - fallback to defaults", func(t *testing.T) {
		mockSkills := &mockSkillsService{
			skills: &TradingSkills{}, // No skills
		}
		service := NewCargoService(mockSkills)

		items := []CargoItem{
			{TypeID: 1, Volume: 10, Value: 100, Quantity: 20},
		}

		solution, err := service.OptimizeCargo(
			context.Background(),
			12345,
			"test-token",
			1000.0,
			items,
		)

		require.NoError(t, err)
		assert.Equal(t, 1000.0, solution.BaseCapacity)
		assert.Equal(t, 1000.0, solution.EffectiveCapacity, "No skill bonus")
		assert.Equal(t, 0.0, solution.CapacityBonusPercent)
	})

	t.Run("Nearly full cargo - generates recommendations", func(t *testing.T) {
		mockSkills := &mockSkillsService{
			skills: &TradingSkills{
				SpaceshipCommand:   0,
				GallenteIndustrial: 0,
			},
		}
		service := NewCargoService(mockSkills)

		// Items that will fill >95% of capacity
		items := []CargoItem{
			{TypeID: 1, Volume: 10, Value: 100, Quantity: 100},
		}

		solution, err := service.OptimizeCargo(
			context.Background(),
			12345,
			"test-token",
			100.0, // Small capacity to ensure >95% usage
			items,
		)

		require.NoError(t, err)
		assert.Greater(t, solution.UsedCapacity, 95.0)
		assert.NotEmpty(t, solution.Recommendations, "Should provide skill training recommendations")
		assert.Contains(t, solution.Recommendations, "Spaceship Command")
		assert.Contains(t, solution.Recommendations, "Industrial")
	})

	t.Run("Skills service error - graceful degradation", func(t *testing.T) {
		mockSkills := &mockSkillsService{
			err: assert.AnError, // Simulate error
		}
		service := NewCargoService(mockSkills)

		items := []CargoItem{
			{TypeID: 1, Volume: 10, Value: 100, Quantity: 10},
		}

		solution, err := service.OptimizeCargo(
			context.Background(),
			12345,
			"test-token",
			1000.0,
			items,
		)

		require.NoError(t, err, "Should not fail if skills unavailable")
		// Should use base capacity without bonuses
		assert.Equal(t, 1000.0, solution.EffectiveCapacity)
	})
}

// TestCargoService_EdgeCases tests various edge cases
func TestCargoService_EdgeCases(t *testing.T) {
	service := NewCargoService(&mockSkillsService{})

	t.Run("Invalid item volume", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 0, Value: 100, Quantity: 10},  // Zero volume
			{TypeID: 2, Volume: -5, Value: 100, Quantity: 10}, // Negative volume
			{TypeID: 3, Volume: 10, Value: 100, Quantity: 10}, // Valid
		}

		solution := service.KnapsackDP(items, 100)

		// Should skip invalid items and only process valid ones
		assert.Greater(t, solution.TotalValue, 0.0, "Should process valid items")
	})

	t.Run("Zero quantity items", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 10, Value: 100, Quantity: 0}, // Zero quantity
		}

		solution := service.KnapsackDP(items, 100)

		assert.Equal(t, 0.0, solution.TotalValue, "Should not select zero quantity items")
	})

	t.Run("Very large capacity", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 0.01, Value: 1, Quantity: 10000},
		}

		solution := service.KnapsackDP(items, 100)

		// Should handle large capacity efficiently
		assert.LessOrEqual(t, solution.TotalVolume, 100.0)
		assert.Greater(t, solution.TotalValue, 0.0)
	})

	t.Run("Very small volumes", func(t *testing.T) {
		items := []CargoItem{
			{TypeID: 1, Volume: 0.001, Value: 0.1, Quantity: 1000},
		}

		solution := service.KnapsackDP(items, 1.0)

		// Should handle sub-centimeter volumes
		assert.LessOrEqual(t, solution.TotalVolume, 1.0)
	})
}

// TestCargoService_CapacityCalculation_SkillCombinations tests various skill combinations
func TestCargoService_CapacityCalculation_SkillCombinations(t *testing.T) {
	service := NewCargoService(&mockSkillsService{})
	baseCapacity := 5000.0 // Typical hauler capacity

	tests := []struct {
		name             string
		skills           *TradingSkills
		expectedCapacity float64
	}{
		{
			name: "Beginner - Level 1 skills",
			skills: &TradingSkills{
				SpaceshipCommand:   1,
				GallenteIndustrial: 1,
			},
			expectedCapacity: 5512.5, // 5000 * 1.05 * 1.05
		},
		{
			name: "Intermediate - Level 3 skills",
			skills: &TradingSkills{
				SpaceshipCommand:   3,
				GallenteIndustrial: 3,
			},
			expectedCapacity: 6612.5, // 5000 * 1.15 * 1.15
		},
		{
			name: "Advanced - Level 4 skills",
			skills: &TradingSkills{
				SpaceshipCommand: 4,
				CaldarIndustrial: 4,
			},
			expectedCapacity: 7200.0, // 5000 * 1.20 * 1.20
		},
		{
			name: "Expert - Max skills",
			skills: &TradingSkills{
				SpaceshipCommand: 5,
				AmarrIndustrial:  5,
			},
			expectedCapacity: 7812.5, // 5000 * 1.25 * 1.25
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capacity, _ := service.CalculateCargoCapacity(baseCapacity, tt.skills)
			assert.InDelta(t, tt.expectedCapacity, capacity, 0.1)
		})
	}
}
