// Package services provides business logic for trading operations
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Sternrassler/eve-esi-client/pkg/pagination"
	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/metrics"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/redis/go-redis/v9"
)

// RouteFinder handles finding profitable trade items from market data
type RouteFinder struct {
	esiClient   *esi.Client
	marketRepo  *database.MarketRepository
	sdeRepo     *database.SDERepository
	sdeDB       *sql.DB
	marketCache *MarketOrderCache
	redisClient *redis.Client
}

// NewRouteFinder creates a new route finder instance
func NewRouteFinder(
	esiClient *esi.Client,
	marketRepo *database.MarketRepository,
	sdeRepo *database.SDERepository,
	sdeDB *sql.DB,
	redisClient *redis.Client,
) *RouteFinder {
	rf := &RouteFinder{
		esiClient:   esiClient,
		marketRepo:  marketRepo,
		sdeRepo:     sdeRepo,
		sdeDB:       sdeDB,
		redisClient: redisClient,
	}

	// Initialize market cache if Redis is available
	if redisClient != nil {
		rf.marketCache = NewMarketOrderCache(redisClient)
	}

	return rf
}

// FindProfitableItems identifies items with profitable spread and volume filter
func (rf *RouteFinder) FindProfitableItems(ctx context.Context, regionID int, cargoCapacity float64) ([]models.ItemPair, error) {
	// Fetch market orders
	orders, err := rf.fetchMarketOrders(ctx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market orders: %w", err)
	}

	log.Printf("Found %d market orders for region %d", len(orders), regionID)

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
		itemInfo, err := rf.sdeRepo.GetTypeInfo(ctx, typeID)
		if err != nil {
			log.Printf("Skipped typeID %d - GetTypeInfo failed: %v", typeID, err)
			continue
		}

		// Get item volume
		itemVol, err := cargo.GetItemVolume(rf.sdeDB, int64(typeID))
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

		// Calculate available volume - limited by BOTH buy and sell side
		// We can only trade the minimum of what we can buy AND what we can sell
		buyAvailable := lowestSell.VolumeRemain  // How much we can buy
		sellAvailable := highestBuy.VolumeRemain // How much we can sell (demand)

		// Take the minimum - we're bottlenecked by the smaller side
		availableQuantity := buyAvailable
		if sellAvailable < buyAvailable {
			availableQuantity = sellAvailable
		}

		availableVolumeM3 := float64(availableQuantity) * itemVol.Volume

		profitableItems = append(profitableItems, models.ItemPair{
			TypeID:            typeID,
			ItemName:          itemInfo.Name,
			ItemVolume:        itemVol.Volume,
			BuyStationID:      lowestSell.LocationID, // Buy from sell orders
			BuySystemID:       rf.getSystemIDFromLocation(ctx, lowestSell.LocationID),
			BuyPrice:          lowestSell.Price,
			SellStationID:     highestBuy.LocationID, // Sell to buy orders
			SellSystemID:      rf.getSystemIDFromLocation(ctx, highestBuy.LocationID),
			SellPrice:         highestBuy.Price,
			SpreadPercent:     spread,
			AvailableVolumeM3: availableVolumeM3,
			AvailableQuantity: availableQuantity,
		})
	}

	return profitableItems, nil
}

// fetchMarketOrders fetches market orders with Redis caching
func (rf *RouteFinder) fetchMarketOrders(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	// Try Redis cache first if available
	if rf.marketCache != nil {
		orders, err := rf.marketCache.Get(ctx, regionID)
		if err == nil {
			metrics.TradingCacheHitsTotal.Inc()
			log.Printf("Cache hit for region %d market orders", regionID)
			return orders, nil
		}
		metrics.TradingCacheMissesTotal.Inc()
		log.Printf("Cache miss for region %d market orders", regionID)
	}

	metrics.TradingCacheMissesTotal.Inc()

	// Fetch fresh data from ESI using BatchFetcher for parallel pagination (much faster)
	config := pagination.DefaultConfig()
	fetcher := pagination.NewBatchFetcher(rf.esiClient.GetRawClient(), config)
	endpoint := fmt.Sprintf("/v1/markets/%d/orders/", regionID)

	// Fetch all pages in parallel
	results, err := fetcher.FetchAllPages(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market data from ESI: %w", err)
	}

	// Convert paginated results to MarketOrder structs
	allOrders := make([]database.MarketOrder, 0)
	fetchedAt := time.Now()

	for pageNum := 1; pageNum <= len(results); pageNum++ {
		pageData, ok := results[pageNum]
		if !ok {
			continue
		}

		// Parse page data
		var orders []database.MarketOrder
		if err := json.Unmarshal(pageData, &orders); err != nil {
			return nil, fmt.Errorf("failed to parse market data from page %d: %w", pageNum, err)
		}

		// Add region ID and timestamp
		for i := range orders {
			orders[i].RegionID = regionID
			orders[i].FetchedAt = fetchedAt
		}

		allOrders = append(allOrders, orders...)
	}

	// Store in database using batch upsert
	if err := rf.marketRepo.UpsertMarketOrders(ctx, allOrders); err != nil {
		return nil, fmt.Errorf("failed to store market data: %w", err)
	}

	// Update Redis cache asynchronously if available
	if rf.marketCache != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = rf.marketCache.Set(cacheCtx, regionID, allOrders)
		}()
	}

	return allOrders, nil
}

// getSystemIDFromLocation retrieves the system ID for a location
func (rf *RouteFinder) getSystemIDFromLocation(ctx context.Context, locationID int64) int64 {
	systemID, err := rf.sdeRepo.GetSystemIDForLocation(ctx, locationID)
	if err != nil {
		log.Printf("Warning: failed to get system ID for location %d: %v", locationID, err)
		return 0
	}
	return systemID
}
