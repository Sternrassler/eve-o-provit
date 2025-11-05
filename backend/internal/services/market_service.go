// Package services - Market data management service
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-esi-client/pkg/pagination"
	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
)

// MarketService orchestrates market data fetching and storage
// Extracts business logic from GetMarketOrders handler
type MarketService struct {
	marketQuerier database.MarketQuerier
	esiClient     ESIRawClient
}

// ESIRawClient interface for raw ESI client access (for BatchFetcher)
type ESIRawClient interface {
	GetRawClient() *client.Client
}

// NewMarketService creates a new market service instance
func NewMarketService(marketQuerier database.MarketQuerier, esiClient ESIRawClient) *MarketService {
	return &MarketService{
		marketQuerier: marketQuerier,
		esiClient:     esiClient,
	}
}

// FetchAndStoreMarketOrders fetches fresh market data from ESI and stores it in database
// Returns the number of orders fetched and stored
func (s *MarketService) FetchAndStoreMarketOrders(ctx context.Context, regionID int) (int, error) {
	// Use esi-client BatchFetcher for parallel pagination (10 workers)
	config := pagination.DefaultConfig()
	fetcher := pagination.NewBatchFetcher(s.esiClient.GetRawClient(), config)
	endpoint := fmt.Sprintf("/v1/markets/%d/orders/", regionID)

	// Fetch all pages in parallel
	results, err := fetcher.FetchAllPages(ctx, endpoint)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch market data from ESI: %w", err)
	}

	// Convert paginated results to MarketOrder structs
	allOrders := make([]database.MarketOrder, 0)
	for pageNum := 1; pageNum <= len(results); pageNum++ {
		pageData, ok := results[pageNum]
		if !ok {
			continue
		}

		// ESI response structure (matches database.MarketOrder fields)
		var orders []database.MarketOrder
		if err := json.Unmarshal(pageData, &orders); err != nil {
			return 0, fmt.Errorf("failed to parse market data from page %d: %w", pageNum, err)
		}

		// Add region ID and timestamp
		for i := range orders {
			orders[i].RegionID = regionID
			orders[i].FetchedAt = time.Now()
		}

		allOrders = append(allOrders, orders...)
	}

	// Store in database
	if err := s.marketQuerier.UpsertMarketOrders(ctx, allOrders); err != nil {
		return 0, fmt.Errorf("failed to store market data: %w", err)
	}

	return len(allOrders), nil
}

// GetMarketOrders retrieves market orders from database for a specific region and type
func (s *MarketService) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
	// Delegate to marketQuerier to fetch from database
	orders, err := s.marketQuerier.GetMarketOrders(ctx, regionID, typeID)
	if err != nil {
		return nil, fmt.Errorf("failed to query market orders: %w", err)
	}
	return orders, nil
}
