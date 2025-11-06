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
func (p *RouteWorkerPool) ProcessItems(ctx context.Context, items []models.ItemPair, cargoCapacity float64) ([]models.TradingRoute, error) {
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
			p.worker(ctx, itemQueue, results, errors, cargoCapacity)
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

// worker processes items from the queue
func (p *RouteWorkerPool) worker(ctx context.Context, itemQueue <-chan models.ItemPair, results chan<- models.TradingRoute, _ chan<- error, cargoCapacity float64) {
	for item := range itemQueue {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return
		default:
		}

		route, err := p.routeOptimizer.CalculateRoute(ctx, item, cargoCapacity)
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
