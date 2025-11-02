// Package services provides business logic for trading operations
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/metrics"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
	"github.com/redis/go-redis/v9"
)

const (
	// MinSpreadPercent is the minimum spread percentage to consider profitable
	MinSpreadPercent = 5.0
	// MaxRoutes is the maximum number of routes to return
	MaxRoutes = 50
	// CalculationTimeout is the total timeout for route calculation
	CalculationTimeout = 30 * time.Second
	// MarketFetchTimeout is the timeout for market order fetching
	MarketFetchTimeout = 15 * time.Second
	// RouteCalculationTimeout is the timeout for route calculation phase
	RouteCalculationTimeout = 25 * time.Second
)

// RouteCalculator handles trading route calculations
type RouteCalculator struct {
	esiClient   *esi.Client
	marketRepo  *database.MarketRepository
	sdeDB       *sql.DB
	sdeRepo     *database.SDERepository
	cache       map[string]*models.CachedData
	cacheMu     sync.RWMutex
	marketCache *MarketOrderCache
	navCache    *NavigationCache
	workerPool  *RouteWorkerPool
	rateLimiter *ESIRateLimiter
	redisClient *redis.Client
}

// NewRouteCalculator creates a new route calculator instance
func NewRouteCalculator(esiClient *esi.Client, sdeDB *sql.DB, sdeRepo *database.SDERepository, marketRepo *database.MarketRepository, redisClient *redis.Client) *RouteCalculator {
	rc := &RouteCalculator{
		esiClient:   esiClient,
		marketRepo:  marketRepo,
		sdeDB:       sdeDB,
		sdeRepo:     sdeRepo,
		cache:       make(map[string]*models.CachedData),
		redisClient: redisClient,
		rateLimiter: NewESIRateLimiter(),
	}

	// Initialize caches if Redis is available
	if redisClient != nil {
		fetcher := NewMarketOrderFetcher(esiClient)
		rc.marketCache = NewMarketOrderCache(redisClient, fetcher)
		rc.navCache = NewNavigationCache(redisClient)
	}

	// Initialize worker pool
	rc.workerPool = NewRouteWorkerPool(rc)

	return rc
}

// Calculate computes profitable trading routes for a region with timeout support
func (rc *RouteCalculator) Calculate(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.TradingCalculationDuration.Observe(duration)
		log.Printf("Route calculation completed in %.2fs", duration)
	}()

	// Create context with timeout
	calcCtx, cancel := context.WithTimeout(ctx, CalculationTimeout)
	defer cancel()

	// Get ship info if cargo capacity not provided
	if cargoCapacity == 0 {
		shipCap, err := cargo.GetShipCapacities(rc.sdeDB, int64(shipTypeID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get ship capacities: %w", err)
		}
		cargoCapacity = shipCap.BaseCargoHold
	}

	// Get ship name
	shipInfo, err := rc.sdeRepo.GetTypeInfo(calcCtx, shipTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ship info: %w", err)
	}

	// Get region name
	regionName, err := rc.getRegionName(calcCtx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get region name: %w", err)
	}

	// Fetch market orders with timeout
	marketCtx, marketCancel := context.WithTimeout(calcCtx, MarketFetchTimeout)
	defer marketCancel()

	orders, err := rc.fetchMarketOrders(marketCtx, regionID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Market order fetch timeout after %v", MarketFetchTimeout)
			return nil, err
		}
		return nil, fmt.Errorf("failed to fetch market orders: %w", err)
	}

	// Find profitable items with volume filtering
	profitableItems, err := rc.findProfitableItems(calcCtx, orders, cargoCapacity)
	if err != nil {
		return nil, fmt.Errorf("failed to find profitable items: %w", err)
	}
	log.Printf("Found %d orders, %d profitable items", len(orders), len(profitableItems))

	// Calculate routes using worker pool with timeout
	routeCtx, routeCancel := context.WithTimeout(calcCtx, RouteCalculationTimeout)
	defer routeCancel()

	routes, err := rc.workerPool.ProcessItems(routeCtx, profitableItems, cargoCapacity)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("failed to calculate routes: %w", err)
	}

	// Check if we timed out
	timedOut := errors.Is(routeCtx.Err(), context.DeadlineExceeded) || errors.Is(calcCtx.Err(), context.DeadlineExceeded)

	// Sort by ISK per hour (descending)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].ISKPerHour > routes[j].ISKPerHour
	})

	// Limit to top 50
	if len(routes) > MaxRoutes {
		routes = routes[:MaxRoutes]
	}

	calculationTime := time.Since(startTime).Milliseconds()

	response := &models.RouteCalculationResponse{
		RegionID:          regionID,
		RegionName:        regionName,
		ShipTypeID:        shipTypeID,
		ShipName:          shipInfo.Name,
		CargoCapacity:     cargoCapacity,
		CalculationTimeMS: calculationTime,
		Routes:            routes,
	}

	// Add timeout warning if applicable
	if timedOut {
		response.Warning = fmt.Sprintf("Calculation timeout after %v, showing partial results", CalculationTimeout)
		log.Printf("WARNING: %s", response.Warning)
	}

	return response, nil
}

// fetchMarketOrders fetches market orders with Redis caching
func (rc *RouteCalculator) fetchMarketOrders(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	// Try Redis cache first if available
	if rc.marketCache != nil {
		orders, err := rc.marketCache.Get(ctx, regionID)
		if err == nil {
			metrics.TradingCacheHitsTotal.Inc()
			log.Printf("Cache hit for region %d market orders", regionID)
			return orders, nil
		}
		metrics.TradingCacheMissesTotal.Inc()
		log.Printf("Cache miss for region %d market orders", regionID)
	}

	// Fallback to in-memory cache
	cacheKey := fmt.Sprintf("market_orders_%d", regionID)
	rc.cacheMu.RLock()
	if cached, exists := rc.cache[cacheKey]; exists && time.Now().Before(cached.ExpiresAt) {
		rc.cacheMu.RUnlock()
		metrics.TradingCacheHitsTotal.Inc()
		return cached.Data.([]database.MarketOrder), nil
	}
	rc.cacheMu.RUnlock()

	metrics.TradingCacheMissesTotal.Inc()

	// Fetch fresh data from ESI (this stores in DB)
	if err := rc.esiClient.FetchMarketOrders(ctx, regionID); err != nil {
		return nil, err
	}

	// Get all orders from database for this region
	allOrders, err := rc.marketRepo.GetAllMarketOrdersForRegion(ctx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders from database: %w", err)
	}

	// Update in-memory cache
	rc.cacheMu.Lock()
	rc.cache[cacheKey] = &models.CachedData{
		Data:      allOrders,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	rc.cacheMu.Unlock()

	// Update Redis cache asynchronously if available
	if rc.marketCache != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = rc.marketCache.Set(cacheCtx, regionID, allOrders)
		}()
	}

	return allOrders, nil
}

// findProfitableItems identifies items with profitable spread and volume filter
func (rc *RouteCalculator) findProfitableItems(ctx context.Context, orders []database.MarketOrder, cargoCapacity float64) ([]models.ItemPair, error) {
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

		// Get item info
		itemInfo, err := rc.sdeRepo.GetTypeInfo(ctx, typeID)
		if err != nil {
			log.Printf("Skipped typeID %d - GetTypeInfo failed: %v", typeID, err)
			continue
		}

		// Get item volume
		itemVol, err := cargo.GetItemVolume(rc.sdeDB, int64(typeID))
		if err != nil {
			log.Printf("Skipped typeID %d (%s) - GetItemVolume failed: %v", typeID, itemInfo.Name, err)
			continue
		}

		// In-memory volume filter: Skip items that are too large
		// Minimum threshold: item must fill at least 10% of cargo
		minQuantity := 1
		if itemVol.Volume > 0 {
			minQuantity = int(cargoCapacity * 0.10 / itemVol.Volume)
			if minQuantity < 1 {
				minQuantity = 1
			}
		}

		// Skip if item won't fit enough in cargo (reduces candidates by ~80%)
		if itemVol.Volume*float64(minQuantity) > cargoCapacity {
			continue
		}

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

	// Get security status for both systems
	buySecurityStatus := rc.getSystemSecurityStatus(ctx, item.BuySystemID)
	sellSecurityStatus := rc.getSystemSecurityStatus(ctx, item.SellSystemID)

	route = models.TradingRoute{
		ItemTypeID:         item.TypeID,
		ItemName:           item.ItemName,
		BuySystemID:        item.BuySystemID,
		BuySystemName:      buySystemName,
		BuyStationID:       item.BuyStationID,
		BuyStationName:     buyStationName,
		BuyPrice:           item.BuyPrice,
		SellSystemID:       item.SellSystemID,
		SellSystemName:     sellSystemName,
		SellStationID:      item.SellStationID,
		SellStationName:    sellStationName,
		SellPrice:          item.SellPrice,
		BuySecurityStatus:  buySecurityStatus,
		SellSecurityStatus: sellSecurityStatus,
		Quantity:           quantity,
		ProfitPerUnit:      profitPerUnit,
		TotalProfit:        totalProfit,
		SpreadPercent:      item.SpreadPercent,
		TravelTimeSeconds:  travelTimeSeconds,
		RoundTripSeconds:   roundTripSeconds,
		ISKPerHour:         iskPerHour,
		Jumps:              travelResult.Jumps,
		ItemVolume:         item.ItemVolume,
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
	// Get system name from SDE
	systemName, err := rc.sdeRepo.GetSystemName(ctx, systemID)
	if err != nil {
		log.Printf("Warning: failed to get system name for %d: %v", systemID, err)
		systemName = fmt.Sprintf("System-%d", systemID)
	}

	// Get station name from SDE
	stationName, err := rc.sdeRepo.GetStationName(ctx, stationID)
	if err != nil {
		log.Printf("Warning: failed to get station name for %d: %v", stationID, err)
		stationName = fmt.Sprintf("Station-%d", stationID)
	}

	return systemName, stationName
}

// getSystemSecurityStatus retrieves the security status of a solar system from SDE
func (rc *RouteCalculator) getSystemSecurityStatus(ctx context.Context, systemID int64) float64 {
	query := `SELECT securityStatus FROM mapSolarSystems WHERE _key = ?`
	var secStatus float64
	err := rc.sdeDB.QueryRowContext(ctx, query, systemID).Scan(&secStatus)
	if err != nil {
		log.Printf("Warning: failed to get security status for system %d: %v", systemID, err)
		return 1.0 // Default to high-sec if lookup fails
	}
	return secStatus
}
