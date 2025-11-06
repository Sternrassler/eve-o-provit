package services

import (
	"context"
	"math/rand"
	"testing"
)

// generateRandomItems creates random cargo items for benchmarking
func generateRandomItems(count int) []CargoItem {
	items := make([]CargoItem, count)
	for i := 0; i < count; i++ {
		items[i] = CargoItem{
			TypeID:   1000 + i,
			Volume:   0.01 + rand.Float64()*100,       // 0.01-100 m³
			Value:    100 + rand.Float64()*10000,      // 100-10100 ISK
			Quantity: 1 + rand.Intn(100),              // 1-100 items
		}
	}
	return items
}

// BenchmarkCargoService_KnapsackDP_100Items benchmarks knapsack with 100 items
func BenchmarkCargoService_KnapsackDP_100Items(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	items := generateRandomItems(100)
	capacity := 5000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}

// BenchmarkCargoService_KnapsackDP_500Items benchmarks knapsack with 500 items
func BenchmarkCargoService_KnapsackDP_500Items(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	items := generateRandomItems(500)
	capacity := 5000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}

// BenchmarkCargoService_KnapsackDP_1000Items benchmarks knapsack with 1000 items
func BenchmarkCargoService_KnapsackDP_1000Items(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	items := generateRandomItems(1000)
	capacity := 5000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}

// BenchmarkCargoService_KnapsackDP_10000Items benchmarks knapsack with 10k items
// Acceptance criteria: Must complete in < 100ms
func BenchmarkCargoService_KnapsackDP_10000Items(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	items := generateRandomItems(10000)
	capacity := 5000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}

// BenchmarkCargoService_OptimizeCargo benchmarks the complete optimization workflow
func BenchmarkCargoService_OptimizeCargo(b *testing.B) {
	mockSkills := &mockSkillsService{
		skills: &TradingSkills{
			SpaceshipCommand:   5,
			GallenteIndustrial: 5,
		},
	}
	service := NewCargoService(mockSkills)
	items := generateRandomItems(1000)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.OptimizeCargo(ctx, 12345, "test-token", 5000.0, items)
	}
}

// BenchmarkCargoService_CalculateCapacity benchmarks capacity calculation
func BenchmarkCargoService_CalculateCapacity(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	skills := &TradingSkills{
		SpaceshipCommand:   5,
		GallenteIndustrial: 5,
	}
	baseCapacity := 5000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CalculateCargoCapacity(baseCapacity, skills)
	}
}

// BenchmarkCargoService_SmallCapacity tests performance with small cargo holds
func BenchmarkCargoService_SmallCapacity(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	items := generateRandomItems(1000)
	capacity := 100.0 // Small capacity (e.g., frigate)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}

// BenchmarkCargoService_LargeCapacity tests performance with large cargo holds
func BenchmarkCargoService_LargeCapacity(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	items := generateRandomItems(1000)
	capacity := 50000.0 // Large capacity (e.g., freighter)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}

// BenchmarkCargoService_HighQuantityItems tests performance with many quantities per item
func BenchmarkCargoService_HighQuantityItems(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	
	// Few item types but high quantities
	items := make([]CargoItem, 50)
	for i := 0; i < 50; i++ {
		items[i] = CargoItem{
			TypeID:   1000 + i,
			Volume:   0.5 + rand.Float64()*10,
			Value:    100 + rand.Float64()*1000,
			Quantity: 500 + rand.Intn(500), // High quantities (500-1000)
		}
	}
	capacity := 5000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}

// BenchmarkCargoService_MixedVolumes tests performance with varied item volumes
func BenchmarkCargoService_MixedVolumes(b *testing.B) {
	service := NewCargoService(&mockSkillsService{})
	
	items := make([]CargoItem, 1000)
	for i := 0; i < 1000; i++ {
		var volume float64
		switch i % 5 {
		case 0:
			volume = 0.01 // Very small
		case 1:
			volume = 0.1
		case 2:
			volume = 1.0
		case 3:
			volume = 10.0
		case 4:
			volume = 100.0 // Very large
		}
		items[i] = CargoItem{
			TypeID:   1000 + i,
			Volume:   volume,
			Value:    volume * 10, // Proportional value
			Quantity: 10 + rand.Intn(90),
		}
	}
	capacity := 5000.0

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.KnapsackDP(items, capacity)
	}
}
