// Package database provides database connection management for dual-DB architecture
// SQLite (Read-Only SDE) + PostgreSQL (Dynamic Market Data)
package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/mattn/go-sqlite3"
)

// Config holds database configuration
type Config struct {
	// PostgreSQL
	PostgresURL string

	// SQLite SDE
	SDEPath string
}

// DB manages dual database connections
type DB struct {
	// PostgreSQL connection pool for dynamic data
	Postgres *pgxpool.Pool

	// SQLite connection for read-only SDE data
	SDE *sql.DB

	config Config
}

// New creates a new dual-database connection
func New(ctx context.Context, cfg Config) (*DB, error) {
	db := &DB{
		config: cfg,
	}

	// Connect to PostgreSQL
	pgPool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Test PostgreSQL connection
	if err := pgPool.Ping(ctx); err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	db.Postgres = pgPool

	// Connect to SQLite SDE (read-only)
	sdeDB, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", cfg.SDEPath))
	if err != nil {
		pgPool.Close()
		return nil, fmt.Errorf("failed to open SQLite SDE: %w", err)
	}

	// Test SQLite connection
	if err := sdeDB.Ping(); err != nil {
		sdeDB.Close()
		pgPool.Close()
		return nil, fmt.Errorf("failed to ping SQLite SDE: %w", err)
	}

	db.SDE = sdeDB

	return db, nil
}

// Close closes all database connections
func (db *DB) Close() {
	if db.Postgres != nil {
		db.Postgres.Close()
	}
	if db.SDE != nil {
		db.SDE.Close()
	}
}

// Health checks the health of all database connections
func (db *DB) Health(ctx context.Context) error {
	// Check PostgreSQL
	if err := db.Postgres.Ping(ctx); err != nil {
		return fmt.Errorf("PostgreSQL unhealthy: %w", err)
	}

	// Check SQLite SDE
	if err := db.SDE.Ping(); err != nil {
		return fmt.Errorf("SQLite SDE unhealthy: %w", err)
	}

	return nil
}

// AcquirePostgres acquires a PostgreSQL connection from the pool
func (db *DB) AcquirePostgres(ctx context.Context) (*pgxpool.Conn, error) {
	return db.Postgres.Acquire(ctx)
}

// BeginTx starts a PostgreSQL transaction
func (db *DB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.Postgres.Begin(ctx)
}
