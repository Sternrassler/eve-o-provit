// Package evedb provides SQLite access to EVE SDE data
// Migrated from eve-sde project
package evedb

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
	path string
}

// Open opens a connection to the SDE SQLite database
func Open(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=ro", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		conn: conn,
		path: dbPath,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying *sql.DB connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Path returns the database file path
func (db *DB) Path() string {
	return db.path
}

// Ping verifies database connectivity
func (db *DB) Ping() error {
	return db.conn.Ping()
}
