package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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

// mockFittingService for testing CargoService
type mockFittingService struct {
	fitting *FittingData
	err     error
}

func (m *mockFittingService) GetShipFitting(ctx context.Context, characterID int, shipTypeID int, accessToken string) (*FittingData, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.fitting, nil
}

func (m *mockFittingService) InvalidateFittingCache(ctx context.Context, characterID int, shipTypeID int) {
	// No-op for mock
}

// TestCargoService_KnapsackDP tests the knapsack optimization algorithm
func TestCargoService_KnapsackDP(t *testing.T) {
	service := NewCargoService(&mockSkillsService{}, nil)

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

// TestCargoService_EdgeCases tests various edge cases
func TestCargoService_EdgeCases(t *testing.T) {
	service := NewCargoService(&mockSkillsService{}, nil)

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

// TestGetEffectiveCargoCapacity_NoFitting tests capacity when fitting unavailable (graceful degradation)
func TestGetEffectiveCargoCapacity_NoFitting(t *testing.T) {
	mockSkills := &mockSkillsService{
		skills: &TradingSkills{
			SpaceshipCommand:   5, // +25%
			GallenteIndustrial: 5, // +25%
		},
	}

	mockFitting := &mockFittingService{
		err: errors.New("no fitting data"),
	}

	service := NewCargoService(mockSkills, mockFitting)

	ctx := context.Background()
	capacity, err := service.GetEffectiveCargoCapacity(ctx, 12345, 20183, 500.0, "token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Graceful degradation: Returns base capacity when fitting unavailable
	expected := 500.0
	if capacity != expected {
		t.Errorf("Expected %.2f m³, got %.2f m³", expected, capacity)
	}
}

// TestGetEffectiveCargoCapacity_WithFitting tests capacity with deterministic fitting
func TestGetEffectiveCargoCapacity_WithFitting(t *testing.T) {
	mockSkills := &mockSkillsService{
		skills: &TradingSkills{
			SpaceshipCommand:   5, // +25%
			GallenteIndustrial: 5, // +25%
		},
	}

	mockFitting := &mockFittingService{
		fitting: &FittingData{
			Bonuses: FittingBonuses{
				EffectiveCargo: 5781.25, // Deterministic: base + skills + modules
			},
		},
	}

	service := NewCargoService(mockSkills, mockFitting)

	ctx := context.Background()
	capacity, err := service.GetEffectiveCargoCapacity(ctx, 12345, 20183, 500.0, "token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// FittingService returns deterministic EffectiveCargo
	expected := 5781.25
	if capacity != expected {
		t.Errorf("Expected %.2f m³, got %.2f m³", expected, capacity)
	}
}

// TestGetEffectiveCargoCapacity_NoSkillsWithFitting tests fitting with deterministic calculation
func TestGetEffectiveCargoCapacity_NoSkillsWithFitting(t *testing.T) {
	mockSkills := &mockSkillsService{
		err: errors.New("skills unavailable"),
	}

	mockFitting := &mockFittingService{
		fitting: &FittingData{
			Bonuses: FittingBonuses{
				EffectiveCargo: 3000.0, // Deterministic: base 500 + fitting 2500
			},
		},
	}

	service := NewCargoService(mockSkills, mockFitting)

	ctx := context.Background()
	capacity, err := service.GetEffectiveCargoCapacity(ctx, 12345, 20183, 500.0, "token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// FittingService calculates deterministic capacity (handles skills internally)
	expected := 3000.0
	if capacity != expected {
		t.Errorf("Expected %.2f m³, got %.2f m³", expected, capacity)
	}
}

// TestGetEffectiveCargoCapacity_NilFittingService tests graceful degradation without FittingService
func TestGetEffectiveCargoCapacity_NilFittingService(t *testing.T) {
	mockSkills := &mockSkillsService{
		skills: &TradingSkills{
			SpaceshipCommand:  3, // +15%
			CaldariIndustrial: 4, // +20%
		},
	}

	// No fitting service (nil)
	service := NewCargoService(mockSkills, nil)

	ctx := context.Background()
	capacity, err := service.GetEffectiveCargoCapacity(ctx, 12345, 20183, 1000.0, "token")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Graceful degradation: Returns base capacity when FittingService is nil
	expected := 1000.0
	if capacity != expected {
		t.Errorf("Expected %.2f m³, got %.2f m³", expected, capacity)
	}
}

// TestGetEffectiveCargoCapacity_NoSkillsNoFitting tests worst case: both skills AND fitting unavailable
func TestGetEffectiveCargoCapacity_NoSkillsNoFitting(t *testing.T) {
	mockSkills := &mockSkillsService{
		err: errors.New("skills unavailable"),
	}

	mockFitting := &mockFittingService{
		err: errors.New("fitting unavailable"),
	}

	service := NewCargoService(mockSkills, mockFitting)

	ctx := context.Background()
	capacity, err := service.GetEffectiveCargoCapacity(ctx, 12345, 20183, 500.0, "token")

	if err != nil {
		t.Fatalf("Expected no error (graceful degradation), got %v", err)
	}

	// Worst case: base capacity only (no skills, no fitting)
	expected := 500.0
	if capacity != expected {
		t.Errorf("Expected %.2f m³, got %.2f m³", expected, capacity)
	}
}
