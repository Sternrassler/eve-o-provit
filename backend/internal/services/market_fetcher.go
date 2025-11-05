// Package services - Market data fetching with caching
package services

import (
	"context"
	"fmt"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/redis/go-redis/v9"
)

// MarketFetcher handles fetching market orders with caching
type MarketFetcher struct {
	esiClient   *esi.Client
	marketRepo  database.MarketQuerier
	cache       *MarketOrderCache
	rateLimiter *ESIRateLimiter
}

// NewMarketFetcher creates a new market fetcher
func NewMarketFetcher(esiClient *esi.Client, marketRepo database.MarketQuerier, redisClient *redis.Client) *MarketFetcher {
	mf := &MarketFetcher{
		esiClient:   esiClient,
		marketRepo:  marketRepo,
		rateLimiter: NewESIRateLimiter(),
	}

	// Initialize cache if Redis available
	if redisClient != nil {
		mf.cache = NewMarketOrderCache(redisClient)
	}

	return mf
}

// FetchMarketOrders retrieves market orders for a region (with caching)
func (mf *MarketFetcher) FetchMarketOrders(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	// Create timeout context
	fetchCtx, cancel := context.WithTimeout(ctx, MarketFetchTimeout)
	defer cancel()

	// Try cache first if available
	if mf.cache != nil {
		cached, err := mf.cache.Get(fetchCtx, regionID)
		if err == nil && len(cached) > 0 {
			return cached, nil
		}
	}

	// Wait for rate limit
	if err := mf.rateLimiter.Wait(fetchCtx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	// Fetch from database (may be stale)
	orders, err := mf.marketRepo.GetAllMarketOrdersForRegion(fetchCtx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market orders: %w", err)
	}

	// Store in cache if available
	if mf.cache != nil {
		_ = mf.cache.Set(fetchCtx, regionID, orders)
	}

	return orders, nil
}

// RefreshMarketOrders forces a fresh fetch from ESI and updates DB + cache
func (mf *MarketFetcher) RefreshMarketOrders(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	// This would use ESI client to fetch fresh data
	// For now, delegating to existing GetAllMarketOrdersForRegion
	// TODO: Implement ESI pagination fetch here
	return mf.FetchMarketOrders(ctx, regionID)
}
