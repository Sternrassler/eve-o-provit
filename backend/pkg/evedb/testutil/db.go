// Package testutil provides common test utilities for evedb packages
package testutil

import (
	"database/sql"
	"os"
	"testing"
)

// OpenTestDB opens the SDE database for testing with environment variable support
// Uses SDE_DB_PATH environment variable or falls back to relative path
// Skips the test if database is not available (e.g., in CI without SDE download)
func OpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use environment variable or default path
	dbPath := os.Getenv("SDE_DB_PATH")
	if dbPath == "" {
		dbPath = "../../../data/sde/eve-sde.db" // Default for local testing
	}

	// Skip test if SDE database not available (e.g., in CI without SDE download)
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("SDE database not found at %s - skipping test (expected in CI)", dbPath)
		return nil
	}

	db, err := sql.Open("sqlite3", "file:"+dbPath+"?mode=ro")
	if err != nil {
		t.Fatalf("Failed to open SDE database at %s: %v", dbPath, err)
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping SDE database at %s: %v", dbPath, err)
	}

	return db
}
