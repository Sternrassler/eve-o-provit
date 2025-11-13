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

// NavigationServicer defines the interface for navigation-related operations
type NavigationServicer interface {
	// GetSystemIDForLocation resolves a station/structure location ID to its solar system ID
	GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error)

	// GetRegionIDForSystem retrieves the region ID for a given solar system
	GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error)
}

// RouteCalculatorServicer defines the interface for route calculation
type RouteCalculatorServicer interface {
	// Calculate computes profitable trading routes for a region
	// warpSpeed and alignTime are optional deterministic values from frontend (nil = use defaults)
	Calculate(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64, warpSpeed, alignTime *float64) (*models.RouteCalculationResponse, error)

	// CalculateWithFilters computes profitable trading routes with volume filtering
	CalculateWithFilters(ctx context.Context, req *models.RouteCalculationRequest) (*models.RouteCalculationResponse, error)
}

// SkillsServicer defines the interface for character skills operations
type SkillsServicer interface {
	// GetCharacterSkills fetches and caches character skills from ESI
	// Returns default skills (all = 0) if ESI fetch fails (graceful degradation)
	GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error)
}

// FittingServicer defines the interface for ship fitting operations
type FittingServicer interface {
	// GetShipFitting fetches and caches ship fitting from ESI
	// Returns empty fitting (no bonuses) if ESI fetch fails (graceful degradation)
	GetShipFitting(
		ctx context.Context,
		characterID int,
		shipTypeID int,
		accessToken string,
	) (*FittingData, error)

	// InvalidateFittingCache removes fitting data from Redis cache
	InvalidateFittingCache(ctx context.Context, characterID int, shipTypeID int)
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
	// Base: 3%, Reduced by Broker Relations + Advanced + Faction + Corp Standing, Min: 1%, Min fee: 100 ISK
	CalculateBrokerFee(
		brokerRelationsLevel int,
		advancedBrokerRelationsLevel int,
		factionStanding float64,
		corpStanding float64,
		orderValue float64,
	) float64
}

// CargoServicer defines the interface for cargo optimization operations
type CargoServicer interface {
	// GetEffectiveCargoCapacity calculates total effective cargo capacity including skills AND fitting
	// Uses deterministic calculation from FittingService (Issue #77)
	// Returns final effective capacity from deterministic SDE + ESI data
	GetEffectiveCargoCapacity(
		ctx context.Context,
		characterID int,
		shipTypeID int,
		baseCapacity float64,
		accessToken string,
	) (float64, error)

	// KnapsackDP solves the knapsack problem using dynamic programming
	// Optimizes for maximum value while respecting capacity constraint
	KnapsackDP(items []CargoItem, capacity float64) *CargoSolution
}

// ShipServicer defines the interface for ship-related operations
type ShipServicer interface {
	// GetShipCapacities retrieves cargo capacity for a ship type
	GetShipCapacities(ctx context.Context, shipTypeID int64) (*ShipCapacities, error)
}

// SystemServicer defines the interface for system-related operations
type SystemServicer interface {
	// GetSystemInfo retrieves combined system and region information
	GetSystemInfo(ctx context.Context, systemID int64) (*SystemInfo, error)

	// GetStationName retrieves station name by ID
	GetStationName(ctx context.Context, stationID int64) (string, error)
}

// SystemInfo contains system, region and location information
type SystemInfo struct {
	SystemName string
	RegionID   int64
	RegionName string
}

// ShipCapacities represents ship cargo capacity information
type ShipCapacities struct {
	ShipTypeID             int64
	ShipName               string
	BaseCargoHold          float64
	EffectiveCargoHold     float64
	BaseTotalCapacity      float64
	EffectiveTotalCapacity float64
	SkillBonus             float64
	SkillsApplied          bool
}
