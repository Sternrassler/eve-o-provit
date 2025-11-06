// Package services - Service layer interfaces for dependency injection and testing
package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// CharacterServicer defines the interface for character-related operations
type CharacterServicer interface {
	// GetCharacterLocation retrieves the current location of a character
	GetCharacterLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error)

	// CalculateTaxRate calculates the broker tax rate for a character based on their skills
	// Returns fallback rate (0.055) if skills cannot be fetched
	CalculateTaxRate(ctx context.Context, characterID int, accessToken string) (float64, error)
}

// TradingServicer defines the interface for trading calculations
type TradingServicer interface {
	// CalculateInventorySellRoutes calculates profitable sell routes for inventory items
	CalculateInventorySellRoutes(
		ctx context.Context,
		req models.InventorySellRequest,
		startSystemID int64,
		taxRate float64,
	) ([]models.InventorySellRoute, error)
}

// NavigationServicer defines the interface for navigation-related operations
type NavigationServicer interface {
	// GetSystemIDForLocation resolves a station/structure location ID to its solar system ID
	GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error)

	// GetRegionIDForSystem retrieves the region ID for a given solar system
	GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error)
}

// InventorySellOrchestrator defines the interface for inventory sell route orchestration
// This orchestrator handles the complete workflow for calculating sell routes
type InventorySellOrchestrator interface {
	// CalculateSellRoutes orchestrates the complete sell route calculation workflow:
	// 1. Get character location (must be docked)
	// 2. Resolve system ID for station
	// 3. Calculate tax rate (with fallback)
	// 4. Calculate profitable routes
	CalculateSellRoutes(
		ctx context.Context,
		req models.InventorySellRequest,
		characterID int,
		accessToken string,
	) ([]models.InventorySellRoute, error)
}

// RouteCalculatorServicer defines the interface for route calculation
type RouteCalculatorServicer interface {
	// Calculate computes profitable trading routes for a region
	Calculate(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error)
}

// SkillsServicer defines the interface for character skills operations
type SkillsServicer interface {
	// GetCharacterSkills fetches and caches character skills from ESI
	// Returns default skills (all = 0) if ESI fetch fails (graceful degradation)
	GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error)
}

// FeeServicer defines the interface for trading fee calculations
type FeeServicer interface {
	// CalculateFees calculates all trading fees for a transaction
	// Integrates with SkillsService for accurate skill-based fee calculation
	// Falls back to worst-case fees if skills unavailable
	CalculateFees(
		ctx context.Context,
		characterID int,
		accessToken string,
		buyValue float64,
		sellValue float64,
	) (*Fees, error)

	// CalculateSalesTax calculates sales tax based on Accounting skill level
	// Base: 5%, Max reduction: 50% (Accounting V), Min fee: 100 ISK
	CalculateSalesTax(accountingLevel int, orderValue float64) float64

	// CalculateBrokerFee calculates broker fee based on skills and standing
	// Base: 3%, Reduced by Broker Relations + Advanced + Standing, Min: 1%, Min fee: 100 ISK
	CalculateBrokerFee(
		brokerRelationsLevel int,
		advancedBrokerRelationsLevel int,
		factionStanding float64,
		orderValue float64,
	) float64
}

// CargoServicer defines the interface for cargo optimization operations
type CargoServicer interface {
	// CalculateCargoCapacity calculates effective cargo capacity with skill bonuses
	// Returns (effectiveCapacity, totalBonusPercent)
	CalculateCargoCapacity(baseCapacity float64, skills *TradingSkills) (float64, float64)

	// KnapsackDP solves the knapsack problem using dynamic programming
	// Optimizes for maximum value while respecting capacity constraint
	KnapsackDP(items []CargoItem, capacity float64) *CargoSolution

	// OptimizeCargo optimizes cargo selection with skill-aware capacity calculation
	// Includes skill training recommendations when cargo is nearly full
	OptimizeCargo(
		ctx context.Context,
		characterID int,
		accessToken string,
		baseCapacity float64,
		items []CargoItem,
	) (*CargoSolution, error)
}
