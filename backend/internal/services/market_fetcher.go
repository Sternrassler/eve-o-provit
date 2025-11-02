// Package services - Market Order Fetcher with Worker Pool
package services

import (
	"context"
	"fmt"
	"log"
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

	start := time.Now()

	// Get first page to determine total page count
	firstPage, totalPages, err := f.getFirstPageAndCount(fetchCtx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get first page: %w", err)
	}

	log.Printf("ESI Pagination: Region %d has %d total pages (first page: %d orders)", regionID, totalPages, len(firstPage))

	if totalPages == 1 {
		// Only one page, return early
		log.Printf("ESI Pagination: Region %d complete (1 page, %d orders) in %.2fs", regionID, len(firstPage), time.Since(start).Seconds())
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

	log.Printf("ESI Pagination: Starting %d workers for %d remaining pages", f.workerCount, totalPages-1)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < f.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			f.worker(fetchCtx, regionID, pageQueue, results, errors, workerID)
		}(i)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	fetchedPages := 1 // First page already fetched
	for pageOrders := range results {
		allOrders = append(allOrders, pageOrders...)
		fetchedPages++

		// Log progress every 50 pages
		if fetchedPages%50 == 0 {
			log.Printf("ESI Pagination: Progress %d/%d pages (%.1f%%)", fetchedPages, totalPages, float64(fetchedPages)/float64(totalPages)*100)
		}
	}

	// Check for errors
	select {
	case err := <-errors:
		if err != nil {
			log.Printf("ESI Pagination: Warning - worker error (partial data): %v", err)
			log.Printf("ESI Pagination: Returning partial results (%d/%d pages, %d orders) in %.2fs", fetchedPages, totalPages, len(allOrders), time.Since(start).Seconds())
			return allOrders, fmt.Errorf("worker error (partial data): %w", err)
		}
	default:
	}

	log.Printf("ESI Pagination: Region %d complete (%d/%d pages, %d orders) in %.2fs", regionID, fetchedPages, totalPages, len(allOrders), time.Since(start).Seconds())
	return allOrders, nil
}

// worker processes pages from the queue
func (f *MarketOrderFetcher) worker(ctx context.Context, regionID int, pageQueue <-chan int, results chan<- []database.MarketOrder, errors chan<- error, workerID int) {
	pagesProcessed := 0
	for page := range pageQueue {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			log.Printf("ESI Pagination: Worker %d stopping (context cancelled, processed %d pages)", workerID, pagesProcessed)
			return
		default:
		}

		orders, err := f.fetchPage(ctx, regionID, page)
		if err != nil {
			log.Printf("ESI Pagination: Worker %d failed page %d: %v", workerID, page, err)
			// Non-blocking error send
			select {
			case errors <- err:
			default:
			}
			return
		}

		pagesProcessed++

		// Send results
		select {
		case results <- orders:
		case <-ctx.Done():
			log.Printf("ESI Pagination: Worker %d stopping (context cancelled after %d pages)", workerID, pagesProcessed)
			return
		}
	}

	if pagesProcessed > 0 {
		log.Printf("ESI Pagination: Worker %d completed (%d pages)", workerID, pagesProcessed)
	}
}

// getFirstPageAndCount fetches the first page and extracts total page count from headers
func (f *MarketOrderFetcher) getFirstPageAndCount(ctx context.Context, regionID int) ([]database.MarketOrder, int, error) {
	// Fetch first page from ESI (page 1)
	esiOrders, totalPages, err := f.esiClient.FetchMarketOrdersPage(ctx, regionID, 1)
	if err != nil {
		return nil, 0, fmt.Errorf("fetch first page: %w", err)
	}

	// Convert ESI orders to database models
	fetchedAt := time.Now()
	dbOrders := make([]database.MarketOrder, 0, len(esiOrders))
	for _, esiOrder := range esiOrders {
		var minVolume *int
		if esiOrder.MinVolume > 0 {
			minVolume = &esiOrder.MinVolume
		}

		dbOrders = append(dbOrders, database.MarketOrder{
			OrderID:      esiOrder.OrderID,
			TypeID:       esiOrder.TypeID,
			RegionID:     regionID,
			LocationID:   esiOrder.LocationID,
			IsBuyOrder:   esiOrder.IsBuyOrder,
			Price:        esiOrder.Price,
			VolumeTotal:  esiOrder.VolumeTotal,
			VolumeRemain: esiOrder.VolumeRemain,
			MinVolume:    minVolume,
			Issued:       esiOrder.Issued,
			Duration:     esiOrder.Duration,
			FetchedAt:    fetchedAt,
		})
	}

	return dbOrders, totalPages, nil
}

// fetchPage fetches a single page of market orders
func (f *MarketOrderFetcher) fetchPage(ctx context.Context, regionID, page int) ([]database.MarketOrder, error) {
	// Fetch page from ESI
	esiOrders, _, err := f.esiClient.FetchMarketOrdersPage(ctx, regionID, page)
	if err != nil {
		return nil, fmt.Errorf("fetch page %d: %w", page, err)
	}

	// Convert ESI orders to database models
	fetchedAt := time.Now()
	dbOrders := make([]database.MarketOrder, 0, len(esiOrders))
	for _, esiOrder := range esiOrders {
		var minVolume *int
		if esiOrder.MinVolume > 0 {
			minVolume = &esiOrder.MinVolume
		}

		dbOrders = append(dbOrders, database.MarketOrder{
			OrderID:      esiOrder.OrderID,
			TypeID:       esiOrder.TypeID,
			RegionID:     regionID,
			LocationID:   esiOrder.LocationID,
			IsBuyOrder:   esiOrder.IsBuyOrder,
			Price:        esiOrder.Price,
			VolumeTotal:  esiOrder.VolumeTotal,
			VolumeRemain: esiOrder.VolumeRemain,
			MinVolume:    minVolume,
			Issued:       esiOrder.Issued,
			Duration:     esiOrder.Duration,
			FetchedAt:    fetchedAt,
		})
	}

	return dbOrders, nil
}
