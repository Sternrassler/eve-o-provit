// Package services - Cargo Optimization Service for optimal cargo selection
package services

import (
	"context"
	"math"
	"sort"
)

// CargoItem represents an item that can be loaded into cargo
type CargoItem struct {
	TypeID   int     // EVE Online item type ID
	Volume   float64 // Volume per single item (m³)
	Value    float64 // Profit/value per single item (ISK)
	Quantity int     // Maximum available quantity
}

// CargoSolution represents the optimal cargo selection result
type CargoSolution struct {
	Items                []CargoItem // Selected items with quantities
	TotalVolume          float64     // Total volume used (m³)
	TotalValue           float64     // Total value/profit (ISK)
	UsedCapacity         float64     // Percentage of capacity used (0-100)
	Recommendations      string      // Skill training recommendations
	EffectiveCapacity    float64     // Actual capacity after skills (m³)
	BaseCapacity         float64     // Base ship capacity before skills (m³)
	CapacityBonusPercent float64     // Total bonus from skills (%)
}

// CargoService provides cargo optimization with skill-aware capacity calculation
type CargoService struct {
	skillsService  SkillsServicer
	fittingService FittingServicer
}

// NewCargoService creates a new cargo optimization service
func NewCargoService(skillsService SkillsServicer, fittingService FittingServicer) *CargoService {
	return &CargoService{
		skillsService:  skillsService,
		fittingService: fittingService,
	}
}

// GetEffectiveCargoCapacity calculates total effective cargo capacity including skills AND fitting
// Deprecated: This is a thin wrapper around FittingService.GetShipFitting().
// TODO: Refactor callers to use FittingService directly and access fitting.Bonuses.EffectiveCargo
// Uses deterministic calculation from FittingService (Issue #77)
// FittingService.CargoBonus contains TOTAL capacity (base + skills + modules)
// Graceful degradation: If fitting unavailable, uses baseCapacity only
func (s *CargoService) GetEffectiveCargoCapacity(
	ctx context.Context,
	characterID int,
	shipTypeID int,
	baseCapacity float64,
	accessToken string,
) (float64, error) {
	// Get fitting with deterministic calculation (includes skills + modules)
	if s.fittingService == nil {
		// No fitting service available, return base capacity only
		return baseCapacity, nil
	}

	fitting, err := s.fittingService.GetShipFitting(ctx, characterID, shipTypeID, accessToken)
	if err != nil {
		// Fitting data unavailable (graceful degradation)
		// Return base capacity only
		return baseCapacity, nil
	}

	// CargoBonus already contains TOTAL effective capacity (base + skills + modules)
	// from deterministic calculation (GetShipCapacitiesDeterministic)
	totalCapacity := fitting.Bonuses.EffectiveCargo

	return totalCapacity, nil
}

// KnapsackDP solves the knapsack problem using dynamic programming
// Optimizes for maximum value while respecting capacity constraint
// Supports multiple quantities of the same item
// For large item counts (>1000), uses optimized DP with reduced granularity
func (s *CargoService) KnapsackDP(items []CargoItem, capacity float64) *CargoSolution {
	// Handle edge cases
	if len(items) == 0 || capacity <= 0 {
		return &CargoSolution{
			Items:       []CargoItem{},
			TotalVolume: 0,
			TotalValue:  0,
		}
	}

	// For very large item sets, use optimized approach
	if len(items) > 1000 {
		return s.knapsackOptimized(items, capacity)
	}

	// Standard DP for smaller item sets
	// Volume granularity: 0.01 m³ (1 cm³)
	// This allows precise volume calculations while keeping table size manageable
	capacityInt := int(capacity * 100)
	n := len(items)

	// DP table: dp[i][w] = max value using first i items with capacity w
	dp := make([][]float64, n+1)
	for i := range dp {
		dp[i] = make([]float64, capacityInt+1)
	}

	// Fill DP table
	for i := 1; i <= n; i++ {
		item := items[i-1]
		volumeInt := int(item.Volume * 100)

		// Skip items with invalid volume - copy previous row
		if volumeInt <= 0 {
			for w := 0; w <= capacityInt; w++ {
				dp[i][w] = dp[i-1][w]
			}
			continue
		}

		for w := 0; w <= capacityInt; w++ {
			// Don't take item
			dp[i][w] = dp[i-1][w]

			// Try taking item (multiple quantities)
			for qty := 1; qty <= item.Quantity; qty++ {
				totalVol := volumeInt * qty
				if totalVol <= w {
					value := dp[i-1][w-totalVol] + (item.Value * float64(qty))
					if value > dp[i][w] {
						dp[i][w] = value
					}
				} else {
					// No point trying higher quantities
					break
				}
			}
		}
	}

	// Backtrack to find solution
	solution := &CargoSolution{
		Items: []CargoItem{},
	}

	w := capacityInt
	for i := n; i > 0 && w > 0; i-- {
		// Check if item i-1 was taken
		if dp[i][w] != dp[i-1][w] {
			item := items[i-1]
			volumeInt := int(item.Volume * 100)

			// Find quantity taken
			for qty := 1; qty <= item.Quantity; qty++ {
				totalVol := volumeInt * qty
				if totalVol > w {
					break
				}

				expectedValue := dp[i-1][w-totalVol] + (item.Value * float64(qty))
				// Use small epsilon for floating point comparison
				if math.Abs(dp[i][w]-expectedValue) < 0.01 {
					// This quantity was taken
					solution.Items = append(solution.Items, CargoItem{
						TypeID:   item.TypeID,
						Volume:   item.Volume * float64(qty),
						Value:    item.Value * float64(qty),
						Quantity: qty,
					})
					solution.TotalVolume += item.Volume * float64(qty)
					solution.TotalValue += item.Value * float64(qty)
					w -= totalVol
					break
				}
			}
		}
	}

	// Calculate capacity usage
	if capacity > 0 {
		solution.UsedCapacity = (solution.TotalVolume / capacity) * 100
	}

	return solution
}

// knapsackOptimized uses a greedy approach with value/volume sorting
// This is O(n log n) and provides good approximation for large item sets
// Suitable for >1000 items where full DP would be too slow
func (s *CargoService) knapsackOptimized(items []CargoItem, capacity float64) *CargoSolution {
	// Create sortable items with efficiency metric
	type itemWithEfficiency struct {
		item       CargoItem
		efficiency float64 // value per m³
	}

	effItems := make([]itemWithEfficiency, 0, len(items))
	for _, item := range items {
		if item.Volume > 0 && item.Value > 0 {
			effItems = append(effItems, itemWithEfficiency{
				item:       item,
				efficiency: item.Value / item.Volume,
			})
		}
	}

	// Sort by efficiency (value/volume) descending using Go's built-in sort
	// This is O(n log n) using introsort
	sort.Slice(effItems, func(i, j int) bool {
		return effItems[i].efficiency > effItems[j].efficiency
	})

	solution := &CargoSolution{
		Items: []CargoItem{},
	}

	remainingCapacity := capacity

	// Greedily fill cargo with most efficient items
	for _, effItem := range effItems {
		item := effItem.item

		if item.Volume > remainingCapacity {
			continue
		}

		// Calculate how many of this item we can fit
		maxFit := int(remainingCapacity / item.Volume)
		quantity := item.Quantity
		if maxFit < quantity {
			quantity = maxFit
		}

		if quantity > 0 {
			totalVolume := item.Volume * float64(quantity)
			totalValue := item.Value * float64(quantity)

			solution.Items = append(solution.Items, CargoItem{
				TypeID:   item.TypeID,
				Volume:   totalVolume,
				Value:    totalValue,
				Quantity: quantity,
			})

			solution.TotalVolume += totalVolume
			solution.TotalValue += totalValue
			remainingCapacity -= totalVolume
		}

		// If cargo is full, stop
		if remainingCapacity < 0.01 {
			break
		}
	}

	// Calculate capacity usage
	if capacity > 0 {
		solution.UsedCapacity = (solution.TotalVolume / capacity) * 100
	}

	return solution
}
