package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/stretchr/testify/assert"
)

// MockESIMarketClient implements ESIMarketClient interface
type MockESIMarketClient struct {
	orders []database.MarketOrder
	err    error
}

func (m *MockESIMarketClient) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.orders, nil
}

func TestTradingService_NewTradingService(t *testing.T) {
	marketFetcher := &MarketFetcher{}
	profitAnalyzer := &ProfitAnalyzer{}
	routePlanner := &RoutePlanner{}
	sdeQuerier := &database.SDERepository{}
	esiClient := &MockESIMarketClient{}
	sdeDB := (*sql.DB)(nil)

	service := NewTradingService(marketFetcher, profitAnalyzer, routePlanner, sdeQuerier, esiClient, sdeDB)

	assert.NotNil(t, service)
	assert.Equal(t, marketFetcher, service.marketFetcher)
	assert.Equal(t, profitAnalyzer, service.profitAnalyzer)
	assert.Equal(t, routePlanner, service.routePlanner)
}

func TestTradingService_CalculateInventorySellRoutes_IntegrationRequired(t *testing.T) {
	t.Skip("Requires working SDE database for navigation.ShortestPath - tested in integration tests")

	// This test would require:
	// - Real SDE database with mapSolarSystems
	// - navigation.ShortestPath working (needs DB)
	// - Complex market order fixtures
	//
	// Better suited for integration tests
}

func TestTradingService_CalculateInventorySellRoutes_NoOrders(t *testing.T) {
	marketFetcher := &MarketFetcher{}
	profitAnalyzer := &ProfitAnalyzer{}
	routePlanner := &RoutePlanner{}
	sdeQuerier := &database.SDERepository{}
	esiClient := &MockESIMarketClient{orders: []database.MarketOrder{}} // Empty orders
	sdeDB := (*sql.DB)(nil)

	service := NewTradingService(marketFetcher, profitAnalyzer, routePlanner, sdeQuerier, esiClient, sdeDB)

	ctx := context.Background()
	req := models.InventorySellRequest{
		TypeID:           34,
		Quantity:         1000,
		BuyPricePerUnit:  5.00,
		MinProfitPerUnit: 0.50,
		RegionID:         10000002,
		SecurityFilter:   "highsec",
	}

	routes, err := service.CalculateInventorySellRoutes(ctx, req, 30000142, 0.055)

	assert.NoError(t, err)
	assert.Empty(t, routes)
}

func TestTradingService_GetSystemSecurityStatus_HighSec(t *testing.T) {
	t.Skip("Requires working SDE database - tested in integration tests")

	// This test needs real SDE DB query: SELECT security FROM mapSolarSystems
}

func TestTradingService_GetMinRouteSecurityStatus_EmptyRoute(t *testing.T) {
	service := &TradingService{}

	ctx := context.Background()
	route := []int64{}

	minSec := service.getMinRouteSecurityStatus(ctx, route)

	assert.Equal(t, 1.0, minSec, "Empty route should default to 1.0 (high-sec)")
}

func TestTradingService_GetMinRouteSecurityStatus_SingleSystem(t *testing.T) {
	t.Skip("Requires working SDE database - tested in integration tests")

	// Needs real SDE DB query
}
