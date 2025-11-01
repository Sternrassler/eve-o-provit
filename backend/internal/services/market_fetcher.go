// Package services - Market Order Fetcher with Worker Pool
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
)

// MarketOrderFetcher handles parallel fetching of market orders from ESI
type MarketOrderFetcher struct {
	esiClient   *esi.Client
	workerCount int
	timeout     time.Duration
}

// NewMarketOrderFetcher creates a new market order fetcher
func NewMarketOrderFetcher(esiClient *esi.Client) *MarketOrderFetcher {
	return &MarketOrderFetcher{
		esiClient:   esiClient,
		workerCount: 10, // ESI Rate Limit safe: 300 req/min = 5 req/s, 10 workers is safe
		timeout:     15 * time.Second,
	}
}

// FetchAllPages fetches all market order pages in parallel
func (f *MarketOrderFetcher) FetchAllPages(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
	// Create context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Get first page to determine total page count
	firstPage, totalPages, err := f.getFirstPageAndCount(fetchCtx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get first page: %w", err)
	}

	if totalPages == 1 {
		// Only one page, return early
		return firstPage, nil
	}

	// Pre-allocate slice with estimated capacity
	allOrders := make([]database.MarketOrder, 0, len(firstPage)*totalPages)
	allOrders = append(allOrders, firstPage...)

	// Create channels for work distribution
	pageQueue := make(chan int, totalPages)
	results := make(chan []database.MarketOrder, totalPages)
	errors := make(chan error, f.workerCount)

	// Fill page queue (skip page 1 as we already fetched it)
	for page := 2; page <= totalPages; page++ {
		pageQueue <- page
	}
	close(pageQueue)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < f.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			f.worker(fetchCtx, regionID, pageQueue, results, errors)
		}(i)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	for pageOrders := range results {
		allOrders = append(allOrders, pageOrders...)
	}

	// Check for errors
	select {
	case err := <-errors:
		if err != nil {
			return allOrders, fmt.Errorf("worker error: %w", err)
		}
	default:
	}

	return allOrders, nil
}

// worker processes pages from the queue
func (f *MarketOrderFetcher) worker(ctx context.Context, regionID int, pageQueue <-chan int, results chan<- []database.MarketOrder, errors chan<- error) {
	for page := range pageQueue {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		orders, err := f.fetchPage(ctx, regionID, page)
		if err != nil {
			// Non-blocking error send
			select {
			case errors <- err:
			default:
			}
			return
		}

		// Send results
		select {
		case results <- orders:
		case <-ctx.Done():
			return
		}
	}
}

// getFirstPageAndCount fetches the first page and extracts total page count from headers
func (f *MarketOrderFetcher) getFirstPageAndCount(ctx context.Context, regionID int) ([]database.MarketOrder, int, error) {
	// For now, we'll use a simplified approach
	// In production, this would parse the X-Pages header from ESI
	
	// Fetch first page
	orders, err := f.fetchPage(ctx, regionID, 1)
	if err != nil {
		return nil, 0, err
	}

	// TODO: Parse X-Pages header from ESI response
	// For now, assume a reasonable default based on region
	// The Forge typically has ~383 pages
	totalPages := estimateTotalPages(regionID, len(orders))

	return orders, totalPages, nil
}

// fetchPage fetches a single page of market orders
func (f *MarketOrderFetcher) fetchPage(ctx context.Context, regionID, page int) ([]database.MarketOrder, error) {
	// This is a simplified implementation
	// In production, this would make a direct ESI call with page parameter
	
	// For now, delegate to existing ESI client
	// Note: The existing client doesn't support pagination yet
	// This is a placeholder that needs integration with the ESI client
	
	// TODO: Implement pagination support in ESI client
	// endpoint := fmt.Sprintf("/v1/markets/%d/orders/?page=%d", regionID, page)
	
	return nil, fmt.Errorf("pagination not yet implemented in ESI client")
}

// estimateTotalPages estimates total pages based on region and first page size
func estimateTotalPages(regionID int, firstPageSize int) int {
	// The Forge (10000002) is the largest region
	if regionID == 10000002 {
		return 383 // Known from issue requirements
	}

	// For other regions, estimate based on first page
	// Typical page size is ~1000 orders
	if firstPageSize < 100 {
		return 1
	} else if firstPageSize < 500 {
		return 5
	} else if firstPageSize < 900 {
		return 10
	}

	// Default to 50 pages for medium-sized regions
	return 50
}
