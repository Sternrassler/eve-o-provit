// Package services - Trading service orchestrating route calculation workflow
package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
)

// TradingService orchestrates trading route calculation workflow
// Combines MarketFetcher, ProfitAnalyzer, and RoutePlanner
type TradingService struct {
	marketFetcher  *MarketFetcher
	profitAnalyzer *ProfitAnalyzer
	routePlanner   *RoutePlanner
	sdeQuerier     database.SDEQuerier
	esiClient      ESIMarketClient
	sdeDB          *sql.DB
}

// ESIMarketClient interface for ESI market data access
type ESIMarketClient interface {
	GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error)
}

// NewTradingService creates a new trading service instance
func NewTradingService(
	marketFetcher *MarketFetcher,
	profitAnalyzer *ProfitAnalyzer,
	routePlanner *RoutePlanner,
	sdeQuerier database.SDEQuerier,
	esiClient ESIMarketClient,
	sdeDB *sql.DB,
) *TradingService {
	return &TradingService{
		marketFetcher:  marketFetcher,
		profitAnalyzer: profitAnalyzer,
		routePlanner:   routePlanner,
		sdeQuerier:     sdeQuerier,
		esiClient:      esiClient,
		sdeDB:          sdeDB,
	}
}

// CalculateInventorySellRoutes calculates optimal sell routes for inventory items
// Extracts business logic from CalculateInventorySellRoutes handler
func (s *TradingService) CalculateInventorySellRoutes(
	ctx context.Context,
	req models.InventorySellRequest,
	startSystemID int64,
	taxRate float64,
) ([]models.InventorySellRoute, error) {
	// Fetch all buy orders for the item in the region
	orders, err := s.esiClient.GetMarketOrders(ctx, req.RegionID, req.TypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market orders: %w", err)
	}

	log.Printf("[DEBUG] Raw ESI orders count: %d for type_id=%d region_id=%d", len(orders), req.TypeID, req.RegionID)
	if len(orders) > 0 {
		log.Printf("[DEBUG] First order sample: Price=%.2f, IsBuy=%v, Volume=%d", orders[0].Price, orders[0].IsBuyOrder, orders[0].VolumeRemain)
	}

	// Filter for buy orders only
	var buyOrders []struct {
		Price        float64
		VolumeRemain int
		LocationID   int64
	}
	for _, order := range orders {
		if order.IsBuyOrder {
			buyOrders = append(buyOrders, struct {
				Price        float64
				VolumeRemain int
				LocationID   int64
			}{
				Price:        order.Price,
				VolumeRemain: order.VolumeRemain,
				LocationID:   order.LocationID,
			})
		}
	}

	log.Printf("[DEBUG] InventorySell: Found %d buy orders for type_id=%d in region_id=%d", len(buyOrders), req.TypeID, req.RegionID)

	// Calculate routes for each buy order
	var routes []models.InventorySellRoute
	skipped := map[string]int{}

	for _, order := range buyOrders {
		// Calculate net price after tax
		netPrice := order.Price * (1 - taxRate)
		profitPerUnit := netPrice - req.BuyPricePerUnit

		log.Printf("[DEBUG] Order: price=%.2f ISK, taxRate=%.4f, netPrice=%.2f ISK, buyPrice=%.2f ISK, profit=%.2f ISK, minProfit=%.2f ISK",
			order.Price, taxRate, netPrice, req.BuyPricePerUnit, profitPerUnit, req.MinProfitPerUnit)

		// Filter by minimum profit
		if profitPerUnit < req.MinProfitPerUnit {
			skipped["profit_too_low"]++
			log.Printf("[DEBUG] Skipped: profit %.2f < min %.2f", profitPerUnit, req.MinProfitPerUnit)
			continue
		}

		// Calculate available quantity
		availableQuantity := req.Quantity
		if order.VolumeRemain < availableQuantity {
			availableQuantity = order.VolumeRemain
		}

		// Get station/system information
		systemID, err := s.sdeQuerier.GetSystemIDForLocation(ctx, order.LocationID)
		if err != nil {
			skipped["invalid_location"]++
			continue // Skip invalid locations
		}

		// Calculate route navigation
		travelResult, err := navigation.ShortestPath(s.sdeDB, startSystemID, systemID, false)
		if err != nil {
			skipped["navigation_failed"]++
			continue // Skip if route calculation fails
		}

		// Get min security status of route
		minRouteSecurity := s.getMinRouteSecurityStatus(ctx, travelResult.Route)

		// Apply security filter
		if req.SecurityFilter == "highsec" && minRouteSecurity < 0.5 {
			skipped["security_highsec"]++
			continue
		}
		if req.SecurityFilter == "highlow" && minRouteSecurity <= 0.0 {
			skipped["security_highlow"]++
			continue
		}

		systemName, _ := s.sdeQuerier.GetSystemName(ctx, systemID)
		stationName, _ := s.sdeQuerier.GetStationName(ctx, order.LocationID)
		sellSec := s.getSystemSecurityStatus(ctx, systemID)

		route := models.InventorySellRoute{
			SellStationID:          order.LocationID,
			SellStationName:        stationName,
			SellSystemID:           systemID,
			SellSystemName:         systemName,
			SellSecurityStatus:     sellSec,
			BuyOrderPrice:          order.Price,
			TaxRate:                taxRate,
			NetPricePerUnit:        netPrice,
			ProfitPerUnit:          profitPerUnit,
			AvailableQuantity:      availableQuantity,
			TotalProfit:            profitPerUnit * float64(availableQuantity),
			RouteJumps:             len(travelResult.Route) - 1,
			RouteSystemIDs:         travelResult.Route,
			MinRouteSecurityStatus: minRouteSecurity,
		}

		routes = append(routes, route)
	}

	// Sort by profit per unit (descending)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].ProfitPerUnit > routes[j].ProfitPerUnit
	})

	log.Printf("[DEBUG] InventorySell: Generated %d routes. Skipped: %+v", len(routes), skipped)

	return routes, nil
}

// getMinRouteSecurityStatus finds the minimum security status across all systems in a route
func (s *TradingService) getMinRouteSecurityStatus(ctx context.Context, route []int64) float64 {
	if len(route) == 0 {
		return 1.0
	}

	minSec := 1.0
	for _, systemID := range route {
		sec := s.getSystemSecurityStatus(ctx, systemID)
		if sec < minSec {
			minSec = sec
		}
	}
	return minSec
}

// getSystemSecurityStatus retrieves security status for a system
func (s *TradingService) getSystemSecurityStatus(ctx context.Context, systemID int64) float64 {
	security, err := s.sdeQuerier.GetSystemSecurityStatus(ctx, systemID)
	if err != nil {
		return 1.0 // Default to high-sec on error
	}
	return security
}
