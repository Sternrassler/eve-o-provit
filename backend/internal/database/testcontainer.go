// Package database - Testcontainer utilities for integration tests
//go:build integration || !unit

package database

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestPostgresContainer holds a PostgreSQL testcontainer instance
type TestPostgresContainer struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
	ConnStr   string
}

// SetupPostgresContainer creates and starts a PostgreSQL testcontainer
// This is used for integration tests that require a real database
func SetupPostgresContainer(t *testing.T) *TestPostgresContainer {
	t.Helper()

	ctx := context.Background()

	// Create PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("eve_o_provit_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to get connection string: %v", err)
	}

	// Create pgxpool connection
	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to parse pool config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		container.Terminate(ctx)
		t.Fatalf("Failed to create pool: %v", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		container.Terminate(ctx)
		t.Fatalf("Failed to ping database: %v", err)
	}

	tc := &TestPostgresContainer{
		Container: container,
		Pool:      pool,
		ConnStr:   connStr,
	}

	// Cleanup on test completion
	t.Cleanup(func() {
		tc.Close()
	})

	return tc
}

// ApplyMigrations applies SQL migrations from a directory
func (tc *TestPostgresContainer) ApplyMigrations(t *testing.T, migrationsDir string) {
	t.Helper()

	ctx := context.Background()

	// Open standard sql.DB for migration execution
	db, err := sql.Open("pgx", tc.ConnStr)
	if err != nil {
		t.Fatalf("Failed to open DB for migrations: %v", err)
	}
	defer db.Close()

	// Read and execute migration files
	migrations, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		t.Fatalf("Failed to find migration files: %v", err)
	}

	if len(migrations) == 0 {
		t.Logf("No migration files found in %s", migrationsDir)
		return
	}

	for _, migration := range migrations {
		t.Logf("Applying migration: %s", filepath.Base(migration))

		// Read migration file
		content, err := filepath.Glob(migration)
		if err != nil {
			t.Fatalf("Failed to read migration %s: %v", migration, err)
		}

		// Execute migration (simplified - in production use proper migration tool)
		_, err = db.ExecContext(ctx, string(content[0]))
		if err != nil {
			t.Fatalf("Failed to execute migration %s: %v", migration, err)
		}
	}

	t.Logf("Applied %d migrations successfully", len(migrations))
}

// CreateTestSchema creates minimal test schema without full migrations
func (tc *TestPostgresContainer) CreateTestSchema(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	// Create minimal schema for testing
	schema := `
		CREATE TABLE IF NOT EXISTS market_orders (
			order_id BIGINT NOT NULL,
			type_id INTEGER NOT NULL,
			region_id INTEGER NOT NULL,
			location_id BIGINT NOT NULL,
			is_buy_order BOOLEAN NOT NULL,
			price DOUBLE PRECISION NOT NULL,
			volume_total INTEGER NOT NULL,
			volume_remain INTEGER NOT NULL,
			min_volume INTEGER,
			issued_at TIMESTAMP NOT NULL,
			duration INTEGER NOT NULL,
			cached_at TIMESTAMP NOT NULL,
			PRIMARY KEY (order_id, cached_at)
		);

		CREATE INDEX IF NOT EXISTS idx_market_orders_region_type 
			ON market_orders(region_id, type_id);

		CREATE TABLE IF NOT EXISTS price_history (
			id SERIAL PRIMARY KEY,
			type_id INTEGER NOT NULL,
			region_id INTEGER NOT NULL,
			date DATE NOT NULL,
			highest DOUBLE PRECISION,
			lowest DOUBLE PRECISION,
			average DOUBLE PRECISION,
			volume BIGINT,
			order_count INTEGER,
			UNIQUE(type_id, region_id, date)
		);
	`

	_, err := tc.Pool.Exec(ctx, schema)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}

	t.Log("Test schema created successfully")
}

// SeedTestData inserts test data into the database
func (tc *TestPostgresContainer) SeedTestData(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	// Insert sample market orders
	seedSQL := `
		INSERT INTO market_orders (
			order_id, type_id, region_id, location_id, is_buy_order,
			price, volume_total, volume_remain, min_volume,
			issued, duration, fetched_at
		) VALUES
			(123456789, 34, 10000002, 60003760, false, 5.50, 1000, 500, 1, NOW() - INTERVAL '1 day', 90, NOW()),
			(987654321, 34, 10000002, 60003760, true, 5.25, 5000, 5000, 10, NOW() - INTERVAL '2 days', 90, NOW()),
			(111222333, 35, 10000002, 60003760, false, 10.00, 2000, 1500, 5, NOW() - INTERVAL '3 hours', 30, NOW())
		ON CONFLICT (order_id, fetched_at) DO NOTHING;
	`

	_, err := tc.Pool.Exec(ctx, seedSQL)
	if err != nil {
		t.Fatalf("Failed to seed test data: %v", err)
	}

	t.Log("Test data seeded successfully")
}

// Close terminates the container and closes the pool
func (tc *TestPostgresContainer) Close() {
	if tc.Pool != nil {
		tc.Pool.Close()
	}
	if tc.Container != nil {
		ctx := context.Background()
		tc.Container.Terminate(ctx)
	}
}

// Truncate removes all data from test tables
func (tc *TestPostgresContainer) Truncate(t *testing.T, tables ...string) {
	t.Helper()

	ctx := context.Background()

	for _, table := range tables {
		_, err := tc.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Fatalf("Failed to truncate table %s: %v", table, err)
		}
	}
}
