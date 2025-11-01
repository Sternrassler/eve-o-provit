// Package services - Benchmark tests for performance optimization
package services

import (
	"context"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// BenchmarkWorkerPoolProcessing benchmarks the worker pool route calculation
func BenchmarkWorkerPoolProcessing(b *testing.B) {
	// Create test data
	items := make([]models.ItemPair, 100)
	for i := 0; i < 100; i++ {
		items[i] = models.ItemPair{
			TypeID:        34 + i,
			ItemName:      "Test Item",
			ItemVolume:    1.0,
			BuyStationID:  60003760,
			BuySystemID:   30000142,
			BuyPrice:      1000.0,
			SellStationID: 60003761,
			SellSystemID:  30000144,
			SellPrice:     2000.0,
			SpreadPercent: 100.0,
		}
	}

	// This is a simplified benchmark without actual RouteCalculator
	// In real scenario, you would use a real calculator with mocked dependencies
	
	b.Run("Sequential", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			routes := make([]models.TradingRoute, 0, len(items))
			for _, item := range items {
				// Simulate route calculation
				route := models.TradingRoute{
					ItemTypeID:    item.TypeID,
					ItemName:      item.ItemName,
					BuyPrice:      item.BuyPrice,
					SellPrice:     item.SellPrice,
					Quantity:      100,
					ProfitPerUnit: item.SellPrice - item.BuyPrice,
					TotalProfit:   (item.SellPrice - item.BuyPrice) * 100,
					ISKPerHour:    100000.0,
				}
				routes = append(routes, route)
			}
		}
	})

	b.Run("Parallel_10_Workers", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			results := make(chan models.TradingRoute, len(items))
			
			// Simplified parallel processing
			sem := make(chan struct{}, 10) // 10 workers
			for _, item := range items {
				sem <- struct{}{}
				go func(itm models.ItemPair) {
					defer func() { <-sem }()
					route := models.TradingRoute{
						ItemTypeID:    itm.TypeID,
						ItemName:      itm.ItemName,
						BuyPrice:      itm.BuyPrice,
						SellPrice:     itm.SellPrice,
						Quantity:      100,
						ProfitPerUnit: itm.SellPrice - itm.BuyPrice,
						TotalProfit:   (itm.SellPrice - itm.BuyPrice) * 100,
						ISKPerHour:    100000.0,
					}
					select {
					case results <- route:
					case <-ctx.Done():
					}
				}(item)
			}
			
			// Wait for all workers
			for i := 0; i < 10; i++ {
				sem <- struct{}{}
			}
			close(results)
			
			// Collect results
			routes := make([]models.TradingRoute, 0, len(items))
			for route := range results {
				routes = append(routes, route)
			}
		}
	})

	b.Run("Parallel_50_Workers", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			results := make(chan models.TradingRoute, len(items))
			
			// Simplified parallel processing
			sem := make(chan struct{}, 50) // 50 workers
			for _, item := range items {
				sem <- struct{}{}
				go func(itm models.ItemPair) {
					defer func() { <-sem }()
					route := models.TradingRoute{
						ItemTypeID:    itm.TypeID,
						ItemName:      itm.ItemName,
						BuyPrice:      itm.BuyPrice,
						SellPrice:     itm.SellPrice,
						Quantity:      100,
						ProfitPerUnit: itm.SellPrice - itm.BuyPrice,
						TotalProfit:   (itm.SellPrice - itm.BuyPrice) * 100,
						ISKPerHour:    100000.0,
					}
					select {
					case results <- route:
					case <-ctx.Done():
					}
				}(item)
			}
			
			// Wait for all workers
			for i := 0; i < 50; i++ {
				sem <- struct{}{}
			}
			close(results)
			
			// Collect results
			routes := make([]models.TradingRoute, 0, len(items))
			for route := range results {
				routes = append(routes, route)
			}
		}
	})
}

// BenchmarkCacheCompression benchmarks gzip compression performance
func BenchmarkCacheCompression(b *testing.B) {
	// Create mock market orders (simulating The Forge size)
	orders := make([]byte, 50*1024*1024) // 50 MB of data
	for i := range orders {
		orders[i] = byte(i % 256)
	}

	b.Run("Uncompressed", func(b *testing.B) {
		b.SetBytes(int64(len(orders)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate storage/retrieval
			_ = orders
		}
	})

	// Note: Actual gzip compression would be tested with real MarketOrderCache
	// This is a placeholder to show performance difference
}

// BenchmarkContextTimeout benchmarks context timeout handling
func BenchmarkContextTimeout(b *testing.B) {
	b.Run("No_Timeout", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			select {
			case <-ctx.Done():
				b.Fatal("unexpected timeout")
			default:
				// Simulate work
				time.Sleep(1 * time.Microsecond)
			}
		}
	})

	b.Run("With_Timeout_NoExpire", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			select {
			case <-ctx.Done():
				b.Fatal("unexpected timeout")
			default:
				// Simulate fast work
				time.Sleep(1 * time.Microsecond)
			}
			cancel()
		}
	})
}
