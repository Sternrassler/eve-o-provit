package services
// Package services - Orchestrator Unit Test
// File: internal/services/inventory_sell_orchestrator_test.go
package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Mock Services (for Orchestrator Testing)
// ============================================================================

type MockCharacterService struct {
	GetLocationFunc      func(ctx context.Context, charID int, token string) (*CharacterLocation, error)
	CalculateTaxRateFunc func(ctx context.Context, charID int, token string) (float64, error)
}

func (m *MockCharacterService) GetLocation(ctx context.Context, charID int, token string) (*CharacterLocation, error) {
	if m.GetLocationFunc != nil {
		return m.GetLocationFunc(ctx, charID, token)
	}
	stationID := int64(60003760)
	return &CharacterLocation{StationID: &stationID}, nil
}

func (m *MockCharacterService) CalculateTaxRate(ctx context.Context, charID int, token string) (float64, error) {
	if m.CalculateTaxRateFunc != nil {
		return m.CalculateTaxRateFunc(ctx, charID, token)
	}
	return 0.03, nil // Default 3% tax
}

func (m *MockCharacterService) GetSkills(ctx context.Context, charID int, token string) (*CharacterSkills, error) {
	return nil, nil
}

type MockNavigationService struct {
	GetSystemIDForLocationFunc func(ctx context.Context, locationID int64) (int64, error)
}

func (m *MockNavigationService) GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error) {
	if m.GetSystemIDForLocationFunc != nil {
		return m.GetSystemIDForLocationFunc(ctx, locationID)
	}
	return 30000142, nil // Default: Jita system ID
}

func (m *MockNavigationService) GetSystemName(ctx context.Context, systemID int64) (string, error) {
	return "Jita", nil
}

func (m *MockNavigationService) GetStationName(ctx context.Context, stationID int64) (string, error) {
	return "Jita IV - Moon 4", nil
}

func (m *MockNavigationService) GetRegionIDForSystem(ctx context.Context, systemID int64) (int64, error) {
	return 10000002, nil
}

func (m *MockNavigationService) CalculateRoute(ctx context.Context, from, to int64, avoidLowsec bool) (*RouteResult, error) {
	return &RouteResult{Route: []int64{from, to}, JumpCount: 1}, nil
}

type MockTradingService struct {
	CalculateInventorySellRoutesFunc func(ctx context.Context, req models.InventorySellRequest, systemID int64, taxRate float64) ([]models.InventorySellRoute, error)
}

func (m *MockTradingService) CalculateInventorySellRoutes(
	ctx context.Context,
	req models.InventorySellRequest,
	startSystemID int64,
	taxRate float64,
) ([]models.InventorySellRoute, error) {
	if m.CalculateInventorySellRoutesFunc != nil {
		return m.CalculateInventorySellRoutesFunc(ctx, req, startSystemID, taxRate)
	}
	return []models.InventorySellRoute{{ProfitPerUnit: 100.0}}, nil
}

// ============================================================================
// Orchestrator Tests
// ============================================================================

func TestInventorySellOrchestrator_Success(t *testing.T) {
	// Setup mocks
	mockChar := &MockCharacterService{}
	mockNav := &MockNavigationService{}
	mockTrading := &MockTradingService{}

	orchestrator := NewInventorySellOrchestrator(mockChar, mockNav, mockTrading)

	// Test request
	req := models.InventorySellRequest{
		TypeID:           34,
		Quantity:         100,
		BuyPricePerUnit:  5.0,
		RegionID:         10000002,
		MinProfitPerUnit: 1.0,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")

	// Assertions
	require.NoError(t, err)
	assert.Len(t, routes, 1)
	assert.Equal(t, 100.0, routes[0].ProfitPerUnit)
}

func TestInventorySellOrchestrator_NotDocked(t *testing.T) {
	// Mock: Character in space (no station)
	mockChar := &MockCharacterService{
		GetLocationFunc: func(ctx context.Context, charID int, token string) (*CharacterLocation, error) {
			return &CharacterLocation{StationID: nil}, nil // In space
		},
	}
	mockNav := &MockNavigationService{}
	mockTrading := &MockTradingService{}

	orchestrator := NewInventorySellOrchestrator(mockChar, mockNav, mockTrading)

	req := models.InventorySellRequest{
		TypeID:           34,
		Quantity:         100,
		BuyPricePerUnit:  5.0,
		RegionID:         10000002,
		MinProfitPerUnit: 1.0,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")

	// Assertions
	assert.ErrorIs(t, err, ErrNotDocked)
	assert.Nil(t, routes)
}

func TestInventorySellOrchestrator_TaxRateFallback(t *testing.T) {
	// Mock: Tax calculation fails (ESI timeout)
	var capturedTaxRate float64
	mockChar := &MockCharacterService{
		CalculateTaxRateFunc: func(ctx context.Context, charID int, token string) (float64, error) {
			return 0, errors.New("ESI timeout")
		},
	}
	mockNav := &MockNavigationService{}
	mockTrading := &MockTradingService{
		CalculateInventorySellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, systemID int64, taxRate float64) ([]models.InventorySellRoute, error) {
			capturedTaxRate = taxRate
			return []models.InventorySellRoute{}, nil
		},
	}

	orchestrator := NewInventorySellOrchestrator(mockChar, mockNav, mockTrading)

	req := models.InventorySellRequest{
		TypeID:           34,
		Quantity:         100,
		BuyPricePerUnit:  5.0,
		RegionID:         10000002,
		MinProfitPerUnit: 1.0,
	}

	_, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, 0.055, capturedTaxRate, "Should use fallback tax rate on error")
}

func TestInventorySellOrchestrator_LocationNotFound(t *testing.T) {
	// Mock: Station not found in SDE
	mockChar := &MockCharacterService{}
	mockNav := &MockNavigationService{
		GetSystemIDForLocationFunc: func(ctx context.Context, locationID int64) (int64, error) {
			return 0, errors.New("station not found in SDE")
		},
	}
	mockTrading := &MockTradingService{}

	orchestrator := NewInventorySellOrchestrator(mockChar, mockNav, mockTrading)

	req := models.InventorySellRequest{
		TypeID:           34,
		Quantity:         100,
		BuyPricePerUnit:  5.0,
		RegionID:         10000002,
		MinProfitPerUnit: 1.0,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")

	// Assertions
	assert.ErrorIs(t, err, ErrLocationNotFound)
	assert.Nil(t, routes)
}

// ============================================================================
// Test Benefits
// ============================================================================

/*
Orchestrator Unit Tests:
- ✅ Test business workflow in isolation (no HTTP layer)
- ✅ Test error paths (not docked, location errors, tax fallback)
- ✅ Verify service orchestration logic
- ✅ No Fiber/HTTP dependencies (faster tests)
- ✅ Mock only what's needed for each test

Handler Tests (after refactoring):
- Focus on HTTP layer (status codes, JSON structure)
- Mock only the orchestrator (1 mock instead of 4)
- Simple setup (10-15 lines instead of 50+)

Result: Complete coverage with simpler, faster, more maintainable tests
*/
