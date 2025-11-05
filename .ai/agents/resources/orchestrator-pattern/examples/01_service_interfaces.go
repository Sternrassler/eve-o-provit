// Package services - Service Interfaces für Dependency Injection
// File: internal/services/interfaces.go
package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// ============================================================================
// Character Service Interface
// ============================================================================

// CharacterService provides character-related operations (location, skills, tax)
type CharacterService interface {
	// GetLocation fetches character location from ESI with caching
	GetLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error)

	// CalculateTaxRate calculates effective tax rate based on character skills
	// Returns broker fee + sales tax (0.0 - 1.0 range)
	CalculateTaxRate(ctx context.Context, characterID int, accessToken string) (float64, error)

	// GetSkills fetches character skills from ESI with caching
	GetSkills(ctx context.Context, characterID int, accessToken string) (*CharacterSkills, error)
}

// Ensure CharacterHelper implements CharacterService
var _ CharacterService = (*CharacterHelper)(nil)

// ============================================================================
// Navigation Service Interface
// ============================================================================

// NavigationService provides SDE-based navigation and location queries
type NavigationService interface {
	// GetSystemIDForLocation resolves station/structure ID to solar system ID
	GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error)

	// GetSystemName returns localized name for solar system
	GetSystemName(ctx context.Context, systemID int64) (string, error)

	// GetStationName returns name for station (NPC or player-owned)
	GetStationName(ctx context.Context, stationID int64) (string, error)

	// GetRegionIDForSystem returns region ID containing the system
	GetRegionIDForSystem(ctx context.Context, systemID int64) (int64, error)

	// CalculateRoute computes shortest path between two systems
	CalculateRoute(ctx context.Context, fromSystemID, toSystemID int64, avoidLowsec bool) (*RouteResult, error)
}

// RouteResult represents navigation route calculation result
type RouteResult struct {
	Route       []int64 // System IDs in order
	JumpCount   int
	MinSecurity float64 // Minimum security status along route
}

// ============================================================================
// Trading Service Interface
// ============================================================================

// TradingServiceInterface provides trading route calculation
type TradingServiceInterface interface {
	// CalculateInventorySellRoutes finds profitable sell opportunities
	CalculateInventorySellRoutes(
		ctx context.Context,
		req models.InventorySellRequest,
		startSystemID int64,
		taxRate float64,
	) ([]models.InventorySellRoute, error)

	// CalculateBuyRoutes finds profitable buy opportunities (future)
	// CalculateBuyRoutes(ctx context.Context, req models.BuyRoutesRequest) ([]models.BuyRoute, error)
}

// Ensure TradingService implements interface
var _ TradingServiceInterface = (*TradingService)(nil)

// ============================================================================
// Inventory Sell Orchestrator Interface (Facade)
// ============================================================================

// InventorySellOrchestrator orchestrates the complete inventory-sell workflow
// Combines: Character location → Tax calculation → Route calculation
type InventorySellOrchestrator interface {
	// CalculateSellRoutes orchestrates full workflow for selling inventory items
	// Handles:
	// 1. Character location retrieval
	// 2. Docked state validation
	// 3. System ID resolution
	// 4. Tax rate calculation
	// 5. Route calculation with filters
	CalculateSellRoutes(
		ctx context.Context,
		req models.InventorySellRequest,
		characterID int,
		accessToken string,
	) ([]models.InventorySellRoute, error)
}

// ============================================================================
// Route Calculation Service Interface
// ============================================================================

// RouteCalculationService provides route calculation for trading
type RouteCalculationService interface {
	// Calculate computes optimal trade routes for a region
	Calculate(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResult, error)
}

// Ensure RouteCalculator implements interface
var _ RouteCalculationService = (*RouteCalculator)(nil)
