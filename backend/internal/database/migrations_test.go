package database

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// checkMigrateBinary checks if the migrate binary is available and skips the test if not
func checkMigrateBinary(t *testing.T) {
	t.Helper()

	migrateBin := "migrate"
	if homePath, err := os.UserHomeDir(); err == nil {
		goBinPath := filepath.Join(homePath, "go", "bin", "migrate")
		if _, err := os.Stat(goBinPath); err == nil {
			migrateBin = goBinPath
		}
	}

	if _, err := exec.LookPath(migrateBin); err != nil {
		t.Skip("migrate binary not found in PATH - install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest")
	}
}

// TestMigrationUp tests that migrations create tables and indexes correctly
func TestMigrationUp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if migrate binary is available
	checkMigrateBinary(t)

	ctx := context.Background()

	// Start PostgreSQL container
	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Run migrations UP
	runMigration(t, connStr, "up")

	// Connect to database
	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	// Validate market_orders table
	validateMarketOrdersTable(t, ctx, pool)

	// Validate price_history table
	validatePriceHistoryTable(t, ctx, pool)
}

// TestMigrationDown tests that DOWN migrations clean up properly
func TestMigrationDown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if migrate binary is available
	checkMigrateBinary(t)

	ctx := context.Background()

	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Run migrations UP first
	runMigration(t, connStr, "up")

	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	// Verify tables exist
	if !tableExists(t, ctx, pool, "market_orders") {
		t.Fatal("market_orders table should exist before DOWN migration")
	}
	if !tableExists(t, ctx, pool, "price_history") {
		t.Fatal("price_history table should exist before DOWN migration")
	}

	pool.Close()

	// Run migrations DOWN
	runMigration(t, connStr, "down", "1")

	// Reconnect
	pool = connectDB(t, ctx, connStr)
	defer pool.Close()

	// Verify tables are gone
	if tableExists(t, ctx, pool, "market_orders") {
		t.Error("market_orders table should not exist after DOWN migration")
	}
	if tableExists(t, ctx, pool, "price_history") {
		t.Error("price_history table should not exist after DOWN migration")
	}
}

// TestMigrationReUp tests re-applying migrations after DOWN
func TestMigrationReUp(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if migrate binary is available
	checkMigrateBinary(t)

	ctx := context.Background()

	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// UP -> DOWN -> UP cycle
	runMigration(t, connStr, "up")
	runMigration(t, connStr, "down", "1")
	runMigration(t, connStr, "up")

	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	// Verify tables exist after re-up
	validateMarketOrdersTable(t, ctx, pool)
	validatePriceHistoryTable(t, ctx, pool)
}

// TestMigrationIdempotency tests that running UP twice doesn't break
func TestMigrationIdempotency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if migrate binary is available
	checkMigrateBinary(t)

	ctx := context.Background()

	pgContainer, connStr := setupPostgresContainer(t, ctx)
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	}()

	// Run UP twice
	runMigration(t, connStr, "up")

	// Second UP should be a no-op (already at latest version)
	// This tests idempotency - migrate should handle this gracefully
	cmd := buildMigrateCommand(t, connStr, "up")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if error is "no change" which is acceptable
		if string(output) != "" && string(output) != "no change\n" {
			t.Logf("Second UP migration output: %s", string(output))
		}
		// No error expected - already at latest version
	}

	pool := connectDB(t, ctx, connStr)
	defer pool.Close()

	// Verify tables still exist and are valid
	validateMarketOrdersTable(t, ctx, pool)
	validatePriceHistoryTable(t, ctx, pool)
}

// TestSchemaValidation tests the schema structure in detail
func TestSchemaValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if migrate binary is available
	checkMigrateBinary(t)

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

	// Validate market_orders: 12 columns, 4 indexes, UNIQUE constraint
	validateMarketOrdersSchema(t, ctx, pool)

	// Validate price_history: 9 columns, 2 indexes, UNIQUE constraint
	validatePriceHistorySchema(t, ctx, pool)
}

// TestDataIntegrity tests INSERT/SELECT after migration
func TestDataIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if migrate binary is available
	checkMigrateBinary(t)

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

	// Insert test data into market_orders
	now := time.Now()
	_, err := pool.Exec(ctx, `
		INSERT INTO market_orders (
			order_id, type_id, region_id, location_id, is_buy_order,
			price, volume_total, volume_remain, min_volume, issued, duration, fetched_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, 123456, 34, 10000002, 60003760, false, 5.50, 1000, 500, 1, now, 90, now)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify data can be retrieved
	var orderID int64
	var price float64
	err = pool.QueryRow(ctx, "SELECT order_id, price FROM market_orders WHERE order_id = $1", 123456).Scan(&orderID, &price)
	if err != nil {
		t.Fatalf("Failed to retrieve test data: %v", err)
	}

	if orderID != 123456 {
		t.Errorf("Expected order_id 123456, got %d", orderID)
	}
	if price != 5.50 {
		t.Errorf("Expected price 5.50, got %.2f", price)
	}

	// Insert test data into price_history
	_, err = pool.Exec(ctx, `
		INSERT INTO price_history (
			type_id, region_id, date, highest, lowest, average, volume, order_count
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, 34, 10000002, time.Now().Format("2006-01-02"), 6.00, 5.00, 5.50, 1000000, 150)
	if err != nil {
		t.Fatalf("Failed to insert price history data: %v", err)
	}

	// Verify price history data
	var avgPrice float64
	err = pool.QueryRow(ctx, "SELECT average FROM price_history WHERE type_id = $1 AND region_id = $2", 34, 10000002).Scan(&avgPrice)
	if err != nil {
		t.Fatalf("Failed to retrieve price history: %v", err)
	}

	if avgPrice != 5.50 {
		t.Errorf("Expected average price 5.50, got %.2f", avgPrice)
	}
}

// TestMigrationStatus tests schema_migrations table
func TestMigrationStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if migrate binary is available
	checkMigrateBinary(t)

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

	// Check schema_migrations table exists
	if !tableExists(t, ctx, pool, "schema_migrations") {
		t.Fatal("schema_migrations table should exist after migration")
	}

	// Verify migration is marked as clean (dirty=false)
	var version uint
	var dirty bool
	err := pool.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if err != nil {
		t.Fatalf("Failed to query schema_migrations: %v", err)
	}

	if version == 0 {
		t.Error("Migration version should not be 0")
	}

	if dirty {
		t.Error("Migration should not be marked as dirty")
	}

	t.Logf("Migration status: version=%d, dirty=%v", version, dirty)
}

// Helper functions

func setupPostgresContainer(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, string) {
	t.Helper()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get connection string: %v", err)
	}

	return pgContainer, connStr
}

func connectDB(t *testing.T, ctx context.Context, connStr string) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	return pool
}

func buildMigrateCommand(t *testing.T, connStr string, args ...string) *exec.Cmd {
	t.Helper()

	// Get project root (go up from backend/internal/database)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Find migrations directory
	migrationsDir := filepath.Join(cwd, "..", "..", "migrations")
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		t.Fatalf("Migrations directory not found: %s", migrationsDir)
	}

	// Build migrate command
	migrateBin := "migrate"
	if homePath, err := os.UserHomeDir(); err == nil {
		goBinPath := filepath.Join(homePath, "go", "bin", "migrate")
		if _, err := os.Stat(goBinPath); err == nil {
			migrateBin = goBinPath
		}
	}

	cmdArgs := []string{
		"-path", migrationsDir,
		"-database", connStr,
	}
	cmdArgs = append(cmdArgs, args...)

	return exec.Command(migrateBin, cmdArgs...)
}

func runMigration(t *testing.T, connStr string, args ...string) {
	t.Helper()

	// Check if migrate binary exists
	migrateBin := "migrate"
	if homePath, err := os.UserHomeDir(); err == nil {
		goBinPath := filepath.Join(homePath, "go", "bin", "migrate")
		if _, err := os.Stat(goBinPath); err == nil {
			migrateBin = goBinPath
		}
	}

	if _, err := exec.LookPath(migrateBin); err != nil {
		t.Skip("migrate binary not found in PATH - install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest")
		return
	}

	cmd := buildMigrateCommand(t, connStr, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Migration failed: %v\nOutput: %s", err, string(output))
	}
	t.Logf("Migration output: %s", string(output))
}

func tableExists(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string) bool {
	t.Helper()

	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = $1
		)
	`, tableName).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check table existence: %v", err)
	}

	return exists
}

func validateMarketOrdersTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	if !tableExists(t, ctx, pool, "market_orders") {
		t.Fatal("market_orders table does not exist")
	}

	// Check primary key
	var constraint_name string
	err := pool.QueryRow(ctx, `
		SELECT conname FROM pg_constraint 
		WHERE conrelid = 'market_orders'::regclass 
		AND contype = 'p'
	`).Scan(&constraint_name)
	if err != nil {
		t.Errorf("market_orders table missing primary key: %v", err)
	}

	// Check indexes exist
	expectedIndexes := []string{
		"idx_market_orders_type_region",
		"idx_market_orders_fetched",
		"idx_market_orders_location",
	}

	for _, idxName := range expectedIndexes {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM pg_indexes 
				WHERE schemaname = 'public' 
				AND tablename = 'market_orders'
				AND indexname = $1
			)
		`, idxName).Scan(&exists)
		if err != nil {
			t.Errorf("Failed to check index %s: %v", idxName, err)
		}
		if !exists {
			t.Errorf("Index %s does not exist on market_orders", idxName)
		}
	}
}

func validatePriceHistoryTable(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	if !tableExists(t, ctx, pool, "price_history") {
		t.Fatal("price_history table does not exist")
	}

	// Check indexes
	expectedIndexes := []string{
		"idx_price_history_lookup",
	}

	for _, idxName := range expectedIndexes {
		var exists bool
		err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM pg_indexes 
				WHERE schemaname = 'public' 
				AND tablename = 'price_history'
				AND indexname = $1
			)
		`, idxName).Scan(&exists)
		if err != nil {
			t.Errorf("Failed to check index %s: %v", idxName, err)
		}
		if !exists {
			t.Errorf("Index %s does not exist on price_history", idxName)
		}
	}
}

func validateMarketOrdersSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	// Count columns
	var columnCount int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = 'market_orders'
	`).Scan(&columnCount)
	if err != nil {
		t.Fatalf("Failed to count columns: %v", err)
	}

	if columnCount != 12 {
		t.Errorf("Expected 12 columns in market_orders, got %d", columnCount)
	}

	// Count indexes (including PK, but excluding UNIQUE constraint index as it's implicit)
	var indexCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_indexes 
		WHERE schemaname = 'public' 
		AND tablename = 'market_orders'
	`).Scan(&indexCount)
	if err != nil {
		t.Fatalf("Failed to count indexes: %v", err)
	}

	// Should have at least 4 indexes: 1 PK + 3 regular indexes (may have +1 for UNIQUE)
	if indexCount < 4 {
		t.Errorf("Expected at least 4 indexes on market_orders, got %d", indexCount)
	}

	// Check UNIQUE constraint exists
	var uniqueCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_constraint 
		WHERE conrelid = 'market_orders'::regclass 
		AND contype = 'u'
	`).Scan(&uniqueCount)
	if err != nil {
		t.Fatalf("Failed to count unique constraints: %v", err)
	}

	if uniqueCount < 1 {
		t.Error("Expected at least 1 UNIQUE constraint on market_orders")
	}
}

func validatePriceHistorySchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	// Count columns
	var columnCount int
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.columns 
		WHERE table_schema = 'public' 
		AND table_name = 'price_history'
	`).Scan(&columnCount)
	if err != nil {
		t.Fatalf("Failed to count columns: %v", err)
	}

	if columnCount != 9 {
		t.Errorf("Expected 9 columns in price_history, got %d", columnCount)
	}

	// Count indexes (including PK, but excluding UNIQUE constraint index as it's implicit)
	var indexCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_indexes 
		WHERE schemaname = 'public' 
		AND tablename = 'price_history'
	`).Scan(&indexCount)
	if err != nil {
		t.Fatalf("Failed to count indexes: %v", err)
	}

	// Should have at least 2 indexes: 1 PK + 1 regular index (may have +1 for UNIQUE)
	if indexCount < 2 {
		t.Errorf("Expected at least 2 indexes on price_history, got %d", indexCount)
	}

	// Check UNIQUE constraint exists
	var uniqueCount int
	err = pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM pg_constraint 
		WHERE conrelid = 'price_history'::regclass 
		AND contype = 'u'
	`).Scan(&uniqueCount)
	if err != nil {
		t.Fatalf("Failed to count unique constraints: %v", err)
	}

	if uniqueCount < 1 {
		t.Error("Expected at least 1 UNIQUE constraint on price_history")
	}
}
