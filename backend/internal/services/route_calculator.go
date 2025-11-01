// Package services provides business logic for trading operations
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
)

const (
	// MaxMarketPages is the maximum number of ESI market pages to fetch (simplified)
	MaxMarketPages = 10
	// MinSpreadPercent is the minimum spread percentage to consider profitable
	MinSpreadPercent = 5.0
	// MaxRoutes is the maximum number of routes to return
	MaxRoutes = 50
	// CacheTTL is the cache time-to-live
	CacheTTL = 5 * time.Minute
)

// RouteCalculator handles trading route calculations
type RouteCalculator struct {
	esiClient  *esi.Client
	marketRepo *database.MarketRepository
	sdeDB      *sql.DB
	sdeRepo    *database.SDERepository
	cache      map[string]*models.CachedData
	cacheMu    sync.RWMutex
}

// NewRouteCalculator creates a new route calculator instance
func NewRouteCalculator(esiClient *esi.Client, sdeDB *sql.DB, sdeRepo *database.SDERepository, marketRepo *database.MarketRepository) *RouteCalculator {
	return &RouteCalculator{
		esiClient:  esiClient,
		marketRepo: marketRepo,
		sdeDB:      sdeDB,
		sdeRepo:    sdeRepo,
		cache:      make(map[string]*models.CachedData),
	}
}

// Calculate computes profitable trading routes for a region
func (rc *RouteCalculator) Calculate(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
	startTime := time.Now()

	// Get ship info if cargo capacity not provided
	if cargoCapacity == 0 {
		shipCap, err := cargo.GetShipCapacities(rc.sdeDB, int64(shipTypeID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get ship capacities: %w", err)
		}
		cargoCapacity = shipCap.BaseCargoHold
	}

	// Get ship name
	shipInfo, err := rc.sdeRepo.GetTypeInfo(ctx, shipTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ship info: %w", err)
	}

	// Get region name
	regionName, err := rc.getRegionName(ctx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get region name: %w", err)
	}

	// Fetch market orders
	orders, err := rc.fetchMarketOrders(ctx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market orders: %w", err)
	}

	// Find profitable items
	profitableItems, err := rc.findProfitableItems(ctx, orders)
	if err != nil {
		return nil, fmt.Errorf("failed to find profitable items: %w", err)
	}
	log.Printf("DEBUG: Found %d orders, %d profitable items", len(orders), len(profitableItems))

	// Calculate routes for each profitable item
	routes := make([]models.TradingRoute, 0, len(profitableItems))
	for _, item := range profitableItems {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		route, err := rc.calculateRoute(ctx, item, cargoCapacity)
		if err != nil {
			// Log error but continue with other items
			log.Printf("Warning: skipped route for item %d (%s): %v", item.TypeID, item.ItemName, err)
			continue
		}
		routes = append(routes, route)
	}

	// Sort by ISK per hour (descending)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].ISKPerHour > routes[j].ISKPerHour
	})

	// Limit to top 50
	if len(routes) > MaxRoutes {
		routes = routes[:MaxRoutes]
	}

	calculationTime := time.Since(startTime).Milliseconds()

	return &models.RouteCalculationResponse{
		RegionID:          regionID,
		RegionName:        regionName,
		ShipTypeID:        shipTypeID,
		ShipName:          shipInfo.Name,
		CargoCapacity:     cargoCapacity,
		CalculationTimeMS: calculationTime,
		Routes:            routes,
	}, nil
}

// fetchMarketOrders fetches market orders from ESI (max 10 pages)
func (rc *RouteCalculator) fetchMarketOrders(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("market_orders_%d", regionID)
	rc.cacheMu.RLock()
	if cached, exists := rc.cache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		rc.cacheMu.RUnlock()
		return cached.Data.([]database.MarketOrder), nil
	}
	rc.cacheMu.RUnlock()

	// Fetch fresh data from ESI (this stores in DB)
	if err := rc.esiClient.FetchMarketOrders(ctx, regionID); err != nil {
		return nil, err
	}

	// Get all orders from database for this region
	allOrders, err := rc.marketRepo.GetAllMarketOrdersForRegion(ctx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders from database: %w", err)
	}

	// Cache the result
	rc.cacheMu.Lock()
	rc.cache[cacheKey] = &models.CachedData{
		Data:      allOrders,
		ExpiresAt: time.Now().Add(CacheTTL),
	}
	rc.cacheMu.Unlock()

	return allOrders, nil
}

// findProfitableItems identifies items with profitable spread
func (rc *RouteCalculator) findProfitableItems(ctx context.Context, orders []database.MarketOrder) ([]models.ItemPair, error) {
	// Group orders by type_id
	ordersByType := make(map[int][]database.MarketOrder)
	for _, order := range orders {
		ordersByType[order.TypeID] = append(ordersByType[order.TypeID], order)
	}

	var profitableItems []models.ItemPair

	// Analyze each type
	for typeID, typeOrders := range ordersByType {
		// Find lowest sell price and highest buy price
		var lowestSell, highestBuy *database.MarketOrder

		for i := range typeOrders {
			order := &typeOrders[i]
			if order.IsBuyOrder {
				if highestBuy == nil || order.Price > highestBuy.Price {
					highestBuy = order
				}
			} else {
				if lowestSell == nil || order.Price < lowestSell.Price {
					lowestSell = order
				}
			}
		}

		// Skip if we don't have both buy and sell orders
		if lowestSell == nil || highestBuy == nil {
			continue
		}

		// Calculate spread (sell to buy orders at highestBuy.Price, buy from sell orders at lowestSell.Price)
		spread := ((highestBuy.Price - lowestSell.Price) / lowestSell.Price) * 100

		// Skip if spread is too low or negative
		if spread < MinSpreadPercent {
			continue
		}

		log.Printf("DEBUG: Checking profitable item - TypeID=%d, Spread=%.2f%%, LowestSell=%.2f, HighestBuy=%.2f",
			typeID, spread, lowestSell.Price, highestBuy.Price)

		// Get item info
		itemInfo, err := rc.sdeRepo.GetTypeInfo(ctx, typeID)
		if err != nil {
			log.Printf("DEBUG: Skipped typeID %d - GetTypeInfo failed: %v", typeID, err)
			continue
		}

		// Get item volume
		itemVol, err := cargo.GetItemVolume(rc.sdeDB, int64(typeID))
		if err != nil {
			log.Printf("DEBUG: Skipped typeID %d (%s) - GetItemVolume failed: %v", typeID, itemInfo.Name, err)
			continue
		}

		log.Printf("DEBUG: Added profitable item - TypeID=%d (%s), Volume=%.2f, Spread=%.2f%%",
			typeID, itemInfo.Name, itemVol.Volume, spread)

		profitableItems = append(profitableItems, models.ItemPair{
			TypeID:        typeID,
			ItemName:      itemInfo.Name,
			ItemVolume:    itemVol.Volume,
			BuyStationID:  lowestSell.LocationID, // Buy from sell orders
			BuySystemID:   rc.getSystemIDFromLocation(ctx, lowestSell.LocationID),
			BuyPrice:      lowestSell.Price,
			SellStationID: highestBuy.LocationID, // Sell to buy orders
			SellSystemID:  rc.getSystemIDFromLocation(ctx, highestBuy.LocationID),
			SellPrice:     highestBuy.Price,
			SpreadPercent: spread,
		})
	}

	return profitableItems, nil
}

// calculateRoute calculates a complete trading route with travel time and profit
func (rc *RouteCalculator) calculateRoute(ctx context.Context, item models.ItemPair, cargoCapacity float64) (models.TradingRoute, error) {
	var route models.TradingRoute

	// Calculate quantity that fits in cargo
	if item.ItemVolume <= 0 {
		return route, fmt.Errorf("invalid item volume: %f", item.ItemVolume)
	}
	quantity := int(cargoCapacity / item.ItemVolume)
	if quantity <= 0 {
		return route, fmt.Errorf("item too large for cargo")
	}

	// Calculate profit
	profitPerUnit := item.SellPrice - item.BuyPrice
	totalProfit := profitPerUnit * float64(quantity)

	// Calculate travel time
	travelResult, err := navigation.ShortestPath(rc.sdeDB, item.BuySystemID, item.SellSystemID, false)
	if err != nil {
		return route, fmt.Errorf("failed to calculate route: %w", err)
	}

	// Calculate travel time in seconds (simplified)
	travelTimeSeconds := float64(travelResult.Jumps) * 30.0 // ~30 seconds per jump average
	roundTripSeconds := travelTimeSeconds * 2

	// Calculate ISK per hour
	var iskPerHour float64
	if roundTripSeconds > 0 {
		iskPerHour = (totalProfit / roundTripSeconds) * 3600
	}

	// Get system and station names
	buySystemName, buyStationName := rc.getLocationNames(ctx, item.BuySystemID, item.BuyStationID)
	sellSystemName, sellStationName := rc.getLocationNames(ctx, item.SellSystemID, item.SellStationID)

	route = models.TradingRoute{
		ItemTypeID:        item.TypeID,
		ItemName:          item.ItemName,
		BuySystemID:       item.BuySystemID,
		BuySystemName:     buySystemName,
		BuyStationID:      item.BuyStationID,
		BuyStationName:    buyStationName,
		BuyPrice:          item.BuyPrice,
		SellSystemID:      item.SellSystemID,
		SellSystemName:    sellSystemName,
		SellStationID:     item.SellStationID,
		SellStationName:   sellStationName,
		SellPrice:         item.SellPrice,
		Quantity:          quantity,
		ProfitPerUnit:     profitPerUnit,
		TotalProfit:       totalProfit,
		SpreadPercent:     item.SpreadPercent,
		TravelTimeSeconds: travelTimeSeconds,
		RoundTripSeconds:  roundTripSeconds,
		ISKPerHour:        iskPerHour,
		Jumps:             travelResult.Jumps,
		ItemVolume:        item.ItemVolume,
	}

	return route, nil
}

// Helper functions

func (rc *RouteCalculator) getRegionName(ctx context.Context, regionID int) (string, error) {
	// Query SDE for region name (name is JSON with language codes)
	query := `SELECT name FROM mapRegions WHERE _key = ?`
	var nameJSON string
	err := rc.sdeDB.QueryRowContext(ctx, query, regionID).Scan(&nameJSON)
	if err != nil {
		return "", fmt.Errorf("region %d not found", regionID)
	}

	// Parse JSON to get English name
	// Format: {"en":"The Forge","de":"..."}
	// Simple extraction for "en" key
	var nameMap map[string]string
	if err := json.Unmarshal([]byte(nameJSON), &nameMap); err != nil {
		return fmt.Sprintf("Region-%d", regionID), nil
	}

	if name, ok := nameMap["en"]; ok {
		return name, nil
	}

	return fmt.Sprintf("Region-%d", regionID), nil
}

func (rc *RouteCalculator) getSystemIDFromLocation(ctx context.Context, locationID int64) int64 {
	// Use repository method to query SDE database
	systemID, err := rc.sdeRepo.GetSystemIDForLocation(ctx, locationID)
	if err != nil {
		// Log warning but don't fail the entire calculation
		log.Printf("Warning: failed to get system ID for location %d: %v", locationID, err)
		return 0
	}
	return systemID
}

func (rc *RouteCalculator) getLocationNames(ctx context.Context, systemID, stationID int64) (string, string) {
	// TODO(Phase 2): Implement actual SDE lookups for system/station names
	// Current: Returns placeholder IDs until SDE queries are added
	// Impact: Frontend displays "System-30000142" instead of "Jita"
	systemName := fmt.Sprintf("System-%d", systemID)
	stationName := fmt.Sprintf("Station-%d", stationID)
	return systemName, stationName
}
