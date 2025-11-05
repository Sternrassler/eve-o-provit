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
