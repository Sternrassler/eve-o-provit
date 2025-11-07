// Package services - Route Calculation Worker Pool
package services

import (
	"context"
	"log"
	"sync"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// RouteWorkerPool handles parallel route calculation
type RouteWorkerPool struct {
	workerCount    int
	routeOptimizer *RouteOptimizer
}

// NewRouteWorkerPool creates a new route worker pool
func NewRouteWorkerPool(routeOptimizer *RouteOptimizer) *RouteWorkerPool {
	return &RouteWorkerPool{
		workerCount:    50, // Process 50 item pairs in parallel
		routeOptimizer: routeOptimizer,
	}
}

// ProcessItems calculates routes for all items in parallel
// Accepts effective capacity (with skills), base capacity, and skill bonus percentage
func (p *RouteWorkerPool) ProcessItems(ctx context.Context, items []models.ItemPair, effectiveCapacity float64) ([]models.TradingRoute, error) {
	return p.ProcessItemsWithCapacityInfo(ctx, items, effectiveCapacity, effectiveCapacity, 0)
}

// ProcessItemsWithCapacityInfo calculates routes with detailed capacity information
func (p *RouteWorkerPool) ProcessItemsWithCapacityInfo(ctx context.Context, items []models.ItemPair, effectiveCapacity, baseCapacity, skillBonusPercent float64) ([]models.TradingRoute, error) {
	if len(items) == 0 {
		return []models.TradingRoute{}, nil
	}

	// Create channels
	itemQueue := make(chan models.ItemPair, len(items))
	results := make(chan models.TradingRoute, len(items))
	errors := make(chan error, p.workerCount)

	// Fill work queue
	for _, item := range items {
		itemQueue <- item
	}
	close(itemQueue)

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < p.workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			p.workerWithCapacityInfo(ctx, itemQueue, results, errors, effectiveCapacity, baseCapacity, skillBonusPercent)
		}(i)
	}

	// Wait for workers to complete
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results
	routes := make([]models.TradingRoute, 0, len(items))
	for route := range results {
		routes = append(routes, route)
	}

	// Check for first error (if any)
	select {
	case err := <-errors:
		if err != nil {
			log.Printf("Warning: worker error: %v", err)
		}
	default:
	}

	return routes, nil
}

// worker processes items from the queue (backward compatibility)
func (p *RouteWorkerPool) worker(ctx context.Context, itemQueue <-chan models.ItemPair, results chan<- models.TradingRoute, errors chan<- error, cargoCapacity float64) {
	p.workerWithCapacityInfo(ctx, itemQueue, results, errors, cargoCapacity, cargoCapacity, 0)
}

// workerWithCapacityInfo processes items with detailed capacity tracking
func (p *RouteWorkerPool) workerWithCapacityInfo(ctx context.Context, itemQueue <-chan models.ItemPair, results chan<- models.TradingRoute, _ chan<- error, effectiveCapacity, baseCapacity, skillBonusPercent float64) {
	for item := range itemQueue {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		route, err := p.routeOptimizer.CalculateRouteWithCapacityInfo(ctx, item, effectiveCapacity, baseCapacity, skillBonusPercent)
		if err != nil {
			// Log but don't fail the entire operation
			log.Printf("Warning: skipped route for item %d (%s): %v", item.TypeID, item.ItemName, err)
			continue
		}

		// Send result
		select {
		case results <- route:
		case <-ctx.Done():
			return
		}
	}
}
