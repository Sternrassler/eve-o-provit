package evedb_test

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb"
)

func TestOpenDB(t *testing.T) {
	// Test with non-existent file (should fail gracefully)
	db, err := evedb.Open("/nonexistent/sde.sqlite")
	if err == nil {
		t.Error("Expected error for non-existent database")
		db.Close()
	}
}

func TestDBPing(t *testing.T) {
	t.Skip("Requires actual SDE database - integration test")

	// This test requires data/sde/sde.sqlite to exist
	db, err := evedb.Open("../../data/sde/sde.sqlite")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}
}
