package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// Mock CharacterServicer for orchestrator tests
type MockCharacterService struct {
	GetCharacterLocationFunc func(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error)
	CalculateTaxRateFunc     func(ctx context.Context, characterID int, accessToken string) (float64, error)
}

func (m *MockCharacterService) GetCharacterLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error) {
	if m.GetCharacterLocationFunc != nil {
		return m.GetCharacterLocationFunc(ctx, characterID, accessToken)
	}
	stationID := int64(60003760)
	return &CharacterLocation{StationID: &stationID}, nil
}

func (m *MockCharacterService) CalculateTaxRate(ctx context.Context, characterID int, accessToken string) (float64, error) {
	if m.CalculateTaxRateFunc != nil {
		return m.CalculateTaxRateFunc(ctx, characterID, accessToken)
	}
	return 0.03, nil
}

// Mock NavigationServicer for orchestrator tests
type MockNavigationService struct {
	GetSystemIDForLocationFunc func(ctx context.Context, locationID int64) (int64, error)
	GetRegionIDForSystemFunc   func(ctx context.Context, systemID int64) (int, error)
}

func (m *MockNavigationService) GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error) {
	if m.GetSystemIDForLocationFunc != nil {
		return m.GetSystemIDForLocationFunc(ctx, locationID)
	}
	return 30000142, nil // Jita
}

func (m *MockNavigationService) GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error) {
	if m.GetRegionIDForSystemFunc != nil {
		return m.GetRegionIDForSystemFunc(ctx, systemID)
	}
	return 10000002, nil
}

// Mock TradingServicer for orchestrator tests
type MockTradingService struct {
	CalculateInventorySellRoutesFunc func(ctx context.Context, req models.InventorySellRequest, startSystemID int64, taxRate float64) ([]models.InventorySellRoute, error)
}

func (m *MockTradingService) CalculateInventorySellRoutes(ctx context.Context, req models.InventorySellRequest, startSystemID int64, taxRate float64) ([]models.InventorySellRoute, error) {
	if m.CalculateInventorySellRoutesFunc != nil {
		return m.CalculateInventorySellRoutesFunc(ctx, req, startSystemID, taxRate)
	}
	return []models.InventorySellRoute{}, nil
}

// TestInventorySellOrchestrator_Success tests complete successful workflow
func TestInventorySellOrchestrator_Success(t *testing.T) {
	var capturedSystemID int64
	var capturedTaxRate float64

	mockCharService := &MockCharacterService{
		GetCharacterLocationFunc: func(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error) {
			stationID := int64(60003760)
			return &CharacterLocation{
				StationID:     &stationID,
				SolarSystemID: 30000142,
			}, nil
		},
		CalculateTaxRateFunc: func(ctx context.Context, characterID int, accessToken string) (float64, error) {
			return 0.03, nil
		},
	}

	mockNavService := &MockNavigationService{
		GetSystemIDForLocationFunc: func(ctx context.Context, locationID int64) (int64, error) {
			return 30000142, nil
		},
	}

	mockTradingService := &MockTradingService{
		CalculateInventorySellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, startSystemID int64, taxRate float64) ([]models.InventorySellRoute, error) {
			capturedSystemID = startSystemID
			capturedTaxRate = taxRate
			return []models.InventorySellRoute{
				{
					SellStationID:   60008494,
					SellStationName: "Amarr VIII",
					ProfitPerUnit:   200.0,
					TotalProfit:     20000.0,
				},
			}, nil
		},
	}

	orchestrator := NewInventorySellOrchestrator(mockCharService, mockNavService, mockTradingService)

	req := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}

	if capturedSystemID != 30000142 {
		t.Errorf("Expected system ID 30000142, got %d", capturedSystemID)
	}

	if capturedTaxRate != 0.03 {
		t.Errorf("Expected tax rate 0.03, got %f", capturedTaxRate)
	}
}

// TestInventorySellOrchestrator_CharacterNotDocked tests character in space validation
func TestInventorySellOrchestrator_CharacterNotDocked(t *testing.T) {
	mockCharService := &MockCharacterService{
		GetCharacterLocationFunc: func(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error) {
			return &CharacterLocation{
				StationID:     nil, // In space
				SolarSystemID: 30000142,
			}, nil
		},
	}

	orchestrator := NewInventorySellOrchestrator(mockCharService, &MockNavigationService{}, &MockTradingService{})

	req := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")
	if err == nil {
		t.Fatal("Expected error for character not docked")
	}

	if routes != nil {
		t.Errorf("Expected nil routes, got %v", routes)
	}

	if be, ok := IsBusinessError(err); !ok {
		t.Errorf("Expected BusinessError, got %T", err)
	} else {
		if be.Status != 400 {
			t.Errorf("Expected status 400, got %d", be.Status)
		}
		if be.Code != "CHARACTER_NOT_DOCKED" {
			t.Errorf("Expected code CHARACTER_NOT_DOCKED, got %s", be.Code)
		}
	}
}

// TestInventorySellOrchestrator_LocationError tests character location fetch error
func TestInventorySellOrchestrator_LocationError(t *testing.T) {
	mockCharService := &MockCharacterService{
		GetCharacterLocationFunc: func(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error) {
			return nil, errors.New("ESI API unavailable")
		},
	}

	orchestrator := NewInventorySellOrchestrator(mockCharService, &MockNavigationService{}, &MockTradingService{})

	req := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")
	if err == nil {
		t.Fatal("Expected error for location fetch failure")
	}

	if routes != nil {
		t.Errorf("Expected nil routes, got %v", routes)
	}

	// Should be internal error, not business error
	if _, ok := IsBusinessError(err); ok {
		t.Error("Expected internal error, got BusinessError")
	}
}

// TestInventorySellOrchestrator_SystemResolutionError tests system ID resolution error
func TestInventorySellOrchestrator_SystemResolutionError(t *testing.T) {
	mockNavService := &MockNavigationService{
		GetSystemIDForLocationFunc: func(ctx context.Context, locationID int64) (int64, error) {
			return 0, errors.New("station not found in SDE")
		},
	}

	orchestrator := NewInventorySellOrchestrator(&MockCharacterService{}, mockNavService, &MockTradingService{})

	req := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")
	if err == nil {
		t.Fatal("Expected error for system resolution failure")
	}

	if routes != nil {
		t.Errorf("Expected nil routes, got %v", routes)
	}
}

// TestInventorySellOrchestrator_TaxRateFallback tests tax rate fallback on error
func TestInventorySellOrchestrator_TaxRateFallback(t *testing.T) {
	var capturedTaxRate float64

	mockCharService := &MockCharacterService{
		CalculateTaxRateFunc: func(ctx context.Context, characterID int, accessToken string) (float64, error) {
			return 0, errors.New("ESI skills API unavailable")
		},
	}

	mockTradingService := &MockTradingService{
		CalculateInventorySellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, startSystemID int64, taxRate float64) ([]models.InventorySellRoute, error) {
			capturedTaxRate = taxRate
			return []models.InventorySellRoute{}, nil
		},
	}

	orchestrator := NewInventorySellOrchestrator(mockCharService, &MockNavigationService{}, mockTradingService)

	req := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}

	_, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")
	if err != nil {
		t.Fatalf("Expected no error with fallback tax rate, got: %v", err)
	}

	if capturedTaxRate != 0.055 {
		t.Errorf("Expected fallback tax rate 0.055, got %f", capturedTaxRate)
	}
}

// TestInventorySellOrchestrator_TradingServiceError tests trading service error
func TestInventorySellOrchestrator_TradingServiceError(t *testing.T) {
	mockTradingService := &MockTradingService{
		CalculateInventorySellRoutesFunc: func(ctx context.Context, req models.InventorySellRequest, startSystemID int64, taxRate float64) ([]models.InventorySellRoute, error) {
			return nil, errors.New("no market orders available")
		},
	}

	orchestrator := NewInventorySellOrchestrator(&MockCharacterService{}, &MockNavigationService{}, mockTradingService)

	req := models.InventorySellRequest{
		TypeID:          34,
		Quantity:        100,
		BuyPricePerUnit: 1000.0,
		RegionID:        10000002,
	}

	routes, err := orchestrator.CalculateSellRoutes(context.Background(), req, 12345, "test-token")
	if err == nil {
		t.Fatal("Expected error from trading service")
	}

	if routes != nil {
		t.Errorf("Expected nil routes, got %v", routes)
	}
}
