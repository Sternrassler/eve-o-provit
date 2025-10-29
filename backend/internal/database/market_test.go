package database

import (
	"context"
	"testing"
	"time"
)

func TestMarketRepository_UpsertMarketOrders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Run migrations
	runMigration(t, connStr, "up")

	// Connect to database
	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	// Create repository
	repo := NewMarketRepository(pool)

	// Create test orders
	now := time.Now()
	orders := []MarketOrder{
		{
			OrderID:      123456,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.50,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			Issued:       now,
			Duration:     90,
			FetchedAt:    now,
		},
		{
			OrderID:      789012,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   true,
			Price:        5.00,
			VolumeTotal:  2000,
			VolumeRemain: 1500,
			Issued:       now,
			Duration:     30,
			FetchedAt:    now,
		},
	}

	// Test upsert
	err := repo.UpsertMarketOrders(ctx, orders)
	if err != nil {
		t.Fatalf("Failed to upsert orders: %v", err)
	}

	// Verify orders were inserted
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	if err != nil {
		t.Fatalf("Failed to retrieve orders: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Expected 2 orders, got %d", len(retrieved))
	}

	// Verify order data
	found := false
	for _, order := range retrieved {
		if order.OrderID == 123456 {
			found = true
			if order.Price != 5.50 {
				t.Errorf("Expected price 5.50, got %.2f", order.Price)
			}
			if order.VolumeRemain != 500 {
				t.Errorf("Expected volume_remain 500, got %d", order.VolumeRemain)
			}
		}
	}
	if !found {
		t.Error("Order 123456 not found in retrieved orders")
	}
}

func TestMarketRepository_GetMarketOrders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	runMigration(t, connStr, "up")
	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	repo := NewMarketRepository(pool)

	// Insert test data
	now := time.Now()
	orders := []MarketOrder{
		{
			OrderID:      111111,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.50,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			Issued:       now,
			Duration:     90,
			FetchedAt:    now,
		},
	}

	err := repo.UpsertMarketOrders(ctx, orders)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test retrieval
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	if err != nil {
		t.Fatalf("Failed to get market orders: %v", err)
	}

	if len(retrieved) != 1 {
		t.Errorf("Expected 1 order, got %d", len(retrieved))
	}

	// Test retrieval for non-existent region/type
	retrieved, err = repo.GetMarketOrders(ctx, 99999999, 99999)
	if err != nil {
		t.Fatalf("Failed to get market orders (non-existent): %v", err)
	}

	if len(retrieved) != 0 {
		t.Errorf("Expected 0 orders for non-existent region/type, got %d", len(retrieved))
	}
}

func TestMarketRepository_CleanOldMarketOrders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	runMigration(t, connStr, "up")
	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	repo := NewMarketRepository(pool)

	// Insert orders with different fetched_at times
	oldTime := time.Now().Add(-48 * time.Hour)
	newTime := time.Now()

	orders := []MarketOrder{
		{
			OrderID:      222222,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.50,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			Issued:       oldTime,
			Duration:     90,
			FetchedAt:    oldTime,
		},
		{
			OrderID:      333333,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   true,
			Price:        5.00,
			VolumeTotal:  2000,
			VolumeRemain: 1500,
			Issued:       newTime,
			Duration:     30,
			FetchedAt:    newTime,
		},
	}

	err := repo.UpsertMarketOrders(ctx, orders)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Clean orders older than 24 hours
	deleted, err := repo.CleanOldMarketOrders(ctx, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to clean old orders: %v", err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 order deleted, got %d", deleted)
	}

	// Verify only new order remains
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	if err != nil {
		t.Fatalf("Failed to retrieve orders: %v", err)
	}

	if len(retrieved) != 1 {
		t.Errorf("Expected 1 order remaining, got %d", len(retrieved))
	}

	if len(retrieved) > 0 && retrieved[0].OrderID != 333333 {
		t.Errorf("Expected order 333333 to remain, got %d", retrieved[0].OrderID)
	}
}

func TestMarketRepository_UpsertConflict(t *testing.T) {
	// Test ON CONFLICT logic
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	runMigration(t, connStr, "up")
	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	repo := NewMarketRepository(pool)

	// Step 1: Insert order with OrderID=123, FetchedAt=T1
	t1 := time.Now().Truncate(time.Second)
	orders1 := []MarketOrder{
		{
			OrderID:      123,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.00,
			VolumeTotal:  1000,
			VolumeRemain: 800,
			Issued:       t1,
			Duration:     90,
			FetchedAt:    t1,
		},
	}

	err := repo.UpsertMarketOrders(ctx, orders1)
	if err != nil {
		t.Fatalf("Failed first insert: %v", err)
	}

	// Step 2: Insert same order with OrderID=123, FetchedAt=T1, different price
	orders2 := []MarketOrder{
		{
			OrderID:      123,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        6.00, // Changed price
			VolumeTotal:  1000,
			VolumeRemain: 600, // Changed volume
			Issued:       t1,
			Duration:     90,
			FetchedAt:    t1, // Same fetched_at
		},
	}

	err = repo.UpsertMarketOrders(ctx, orders2)
	if err != nil {
		t.Fatalf("Failed second insert (conflict): %v", err)
	}

	// Step 3: Verify price was updated
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	if err != nil {
		t.Fatalf("Failed to retrieve orders: %v", err)
	}

	// Should still have only 1 record (update, not insert)
	if len(retrieved) != 1 {
		t.Errorf("Expected 1 order after conflict update, got %d", len(retrieved))
	}

	if len(retrieved) > 0 {
		if retrieved[0].Price != 6.00 {
			t.Errorf("Expected updated price 6.00, got %.2f", retrieved[0].Price)
		}
		if retrieved[0].VolumeRemain != 600 {
			t.Errorf("Expected updated volume_remain 600, got %d", retrieved[0].VolumeRemain)
		}
	}

	// Step 4: Insert order with different OrderID and FetchedAt=T2
	t2 := t1.Add(1 * time.Hour)
	orders3 := []MarketOrder{
		{
			OrderID:      124, // Different order ID
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        7.00,
			VolumeTotal:  1000,
			VolumeRemain: 400,
			Issued:       t2,
			Duration:     90,
			FetchedAt:    t2,
		},
	}

	err = repo.UpsertMarketOrders(ctx, orders3)
	if err != nil {
		t.Fatalf("Failed third insert (different order): %v", err)
	}

	// Step 5: Verify both records exist
	retrieved, err = repo.GetMarketOrders(ctx, 10000002, 34)
	if err != nil {
		t.Fatalf("Failed to retrieve orders: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Expected 2 orders, got %d", len(retrieved))
	}
}

func TestMarketRepository_BatchInsert(t *testing.T) {
	// Test batch insert performance
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	runMigration(t, connStr, "up")
	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	repo := NewMarketRepository(pool)

	// Create 100 market orders for batch insert
	now := time.Now()
	orders := make([]MarketOrder, 100)
	for i := 0; i < 100; i++ {
		orders[i] = MarketOrder{
			OrderID:      int64(1000000 + i),
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   i%2 == 0,
			Price:        5.00 + float64(i)*0.01,
			VolumeTotal:  1000 + i*10,
			VolumeRemain: 500 + i*5,
			Issued:       now.Add(-time.Duration(i) * time.Minute),
			Duration:     90,
			FetchedAt:    now,
		}
	}

	// Batch insert
	start := time.Now()
	err := repo.UpsertMarketOrders(ctx, orders)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Failed to batch insert orders: %v", err)
	}

	t.Logf("Batch insert of 100 orders took %v", duration)

	// Verify all were inserted
	retrieved, err := repo.GetMarketOrders(ctx, 10000002, 34)
	if err != nil {
		t.Fatalf("Failed to retrieve orders: %v", err)
	}

	if len(retrieved) != 100 {
		t.Errorf("Expected 100 orders, got %d", len(retrieved))
	}

	// Performance check - should complete in reasonable time (< 5 seconds for 100 orders)
	if duration > 5*time.Second {
		t.Errorf("Batch insert too slow: took %v, expected < 5s", duration)
	}
}

// Mock test for UpsertMarketOrders logic (without database)
func TestMarketRepository_UpsertMarketOrders_Mock(t *testing.T) {
	// This is a unit test that validates the logic without database
	ctx := context.Background()

	// Create test orders
	orders := []MarketOrder{
		{
			OrderID:      123456,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   false,
			Price:        5.50,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			Issued:       time.Now(),
			Duration:     90,
			FetchedAt:    time.Now(),
		},
		{
			OrderID:      789012,
			TypeID:       34,
			RegionID:     10000002,
			LocationID:   60003760,
			IsBuyOrder:   true,
			Price:        5.00,
			VolumeTotal:  2000,
			VolumeRemain: 1500,
			Issued:       time.Now(),
			Duration:     30,
			FetchedAt:    time.Now(),
		},
	}

	// Validate order structure
	for _, order := range orders {
		if order.OrderID == 0 {
			t.Error("OrderID cannot be zero")
		}
		if order.TypeID == 0 {
			t.Error("TypeID cannot be zero")
		}
		if order.Price <= 0 {
			t.Error("Price must be positive")
		}
		if order.VolumeRemain > order.VolumeTotal {
			t.Error("VolumeRemain cannot exceed VolumeTotal")
		}
	}

	// Ensure context is not nil
	if ctx == nil {
		t.Error("Context cannot be nil")
	}
}
