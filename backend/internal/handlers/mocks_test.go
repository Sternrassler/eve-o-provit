// Package handlers - Mock interfaces for handler unit tests
package handlers

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// MockMarketService is a mock implementation of MarketService for testing
type MockMarketService struct {
	FetchAndStoreMarketOrdersFunc func(ctx context.Context, regionID int) (int, error)
	GetMarketOrdersFunc           func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error)
}

// FetchAndStoreMarketOrders mock implementation
func (m *MockMarketService) FetchAndStoreMarketOrders(ctx context.Context, regionID int) (int, error) {
	if m.FetchAndStoreMarketOrdersFunc != nil {
		return m.FetchAndStoreMarketOrdersFunc(ctx, regionID)
	}
	return 0, nil
}

// GetMarketOrders mock implementation
func (m *MockMarketService) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
	if m.GetMarketOrdersFunc != nil {
		return m.GetMarketOrdersFunc(ctx, regionID, typeID)
	}
	return nil, nil
}

// MockTradingService is a mock implementation of TradingService for testing
type MockTradingService struct {
	CalculateInventorySellRoutesFunc func(ctx context.Context, req models.InventorySellRequest, startSystemID int64, taxRate float64) ([]models.InventorySellRoute, error)
}

// CalculateInventorySellRoutes mock implementation
func (m *MockTradingService) CalculateInventorySellRoutes(
	ctx context.Context,
	req models.InventorySellRequest,
	startSystemID int64,
	taxRate float64,
) ([]models.InventorySellRoute, error) {
	if m.CalculateInventorySellRoutesFunc != nil {
		return m.CalculateInventorySellRoutesFunc(ctx, req, startSystemID, taxRate)
	}
	return nil, nil
}
