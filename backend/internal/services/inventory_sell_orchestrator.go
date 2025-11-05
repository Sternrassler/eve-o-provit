// Package services - Orchestrator for inventory sell route calculation
package services

import (
	"context"
	"fmt"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// InventorySellOrchestratorImpl implements the InventorySellOrchestrator interface
type InventorySellOrchestratorImpl struct {
	characterService CharacterServicer
	navigationService NavigationServicer
	tradingService   TradingServicer
}

// NewInventorySellOrchestrator creates a new orchestrator instance
func NewInventorySellOrchestrator(
	characterService CharacterServicer,
	navigationService NavigationServicer,
	tradingService TradingServicer,
) *InventorySellOrchestratorImpl {
	return &InventorySellOrchestratorImpl{
		characterService:  characterService,
		navigationService: navigationService,
		tradingService:    tradingService,
	}
}

// CalculateSellRoutes orchestrates the complete sell route calculation workflow
func (o *InventorySellOrchestratorImpl) CalculateSellRoutes(
	ctx context.Context,
	req models.InventorySellRequest,
	characterID int,
	accessToken string,
) ([]models.InventorySellRoute, error) {
	// Step 1: Get character location
	location, err := o.characterService.GetCharacterLocation(ctx, characterID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get character location: %w", err)
	}

	// Step 2: Validate character is docked
	if location.StationID == nil {
		return nil, &BusinessError{
			Code:    "CHARACTER_NOT_DOCKED",
			Message: "Character must be docked at a station to calculate sell routes",
			Status:  400,
		}
	}

	// Step 3: Resolve system ID for station
	startSystemID, err := o.navigationService.GetSystemIDForLocation(ctx, *location.StationID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve system for station %d: %w", *location.StationID, err)
	}

	// Step 4: Calculate tax rate (with fallback)
	taxRate, err := o.characterService.CalculateTaxRate(ctx, characterID, accessToken)
	if err != nil {
		// Use fallback tax rate if skills API unavailable
		taxRate = 0.055
	}

	// Step 5: Calculate profitable routes
	routes, err := o.tradingService.CalculateInventorySellRoutes(ctx, req, startSystemID, taxRate)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate sell routes: %w", err)
	}

	return routes, nil
}

// BusinessError represents a business logic error with HTTP status code
type BusinessError struct {
	Code    string
	Message string
	Status  int
}

func (e *BusinessError) Error() string {
	return e.Message
}

// IsBusinessError checks if an error is a BusinessError
func IsBusinessError(err error) (*BusinessError, bool) {
	if be, ok := err.(*BusinessError); ok {
		return be, true
	}
	return nil, false
}
