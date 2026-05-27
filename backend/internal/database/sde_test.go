package database

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// TestGetSystemIDForLocation tests the GetSystemIDForLocation method
func TestGetSystemIDForLocation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create test schema
	schema := `
		CREATE TABLE npcStations (
			_key INTEGER PRIMARY KEY,
			stationName TEXT,
			solarSystemID INTEGER
		);

		CREATE TABLE mapDenormalize (
			itemID INTEGER PRIMARY KEY,
			itemName TEXT,
			solarSystemID INTEGER
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Insert test data
	testData := `
		-- NPC Stations (ID range 60000000 - 64000000)
		INSERT INTO npcStations (_key, stationName, solarSystemID) VALUES
			(60003760, 'Jita IV - Moon 4 - Caldari Navy Assembly Plant', 30000142),
			(60008494, 'Amarr VIII (Oris) - Emperor Family Academy', 30002187);

		-- Player Structures/Citadels (higher IDs)
		INSERT INTO mapDenormalize (itemID, itemName, solarSystemID) VALUES
			(1000000000001, 'Test Citadel Alpha', 30000142),
			(1000000000002, 'Test Citadel Beta', 30002187);
	`

	if _, err := db.Exec(testData); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Create repository
	repo := NewSDERepository(db)
	ctx := context.Background()

	tests := []struct {
		name         string
		locationID   int64
		wantSystemID int64
		wantErr      bool
	}{
		{
			name:         "NPC Station - Jita Trade Hub",
			locationID:   60003760,
			wantSystemID: 30000142,
			wantErr:      false,
		},
		{
			name:         "NPC Station - Amarr Trade Hub",
			locationID:   60008494,
			wantSystemID: 30002187,
			wantErr:      false,
		},
		{
			name:         "Player Citadel - Alpha",
			locationID:   1000000000001,
			wantSystemID: 30000142,
			wantErr:      false,
		},
		{
			name:         "Player Citadel - Beta",
			locationID:   1000000000002,
			wantSystemID: 30002187,
			wantErr:      false,
		},
		{
			name:         "Direct System ID - Jita",
			locationID:   30000142,
			wantSystemID: 30000142,
			wantErr:      false,
		},
		{
			name:         "Direct System ID - Amarr",
			locationID:   30002187,
			wantSystemID: 30002187,
			wantErr:      false,
		},
		{
			name:         "Invalid Location ID",
			locationID:   99999999,
			wantSystemID: 0,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			systemID, err := repo.GetSystemIDForLocation(ctx, tt.locationID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for location %d, got nil", tt.locationID)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for location %d: %v", tt.locationID, err)
				return
			}

			if systemID != tt.wantSystemID {
				t.Errorf("GetSystemIDForLocation(%d) = %d, want %d", tt.locationID, systemID, tt.wantSystemID)
			}
		})
	}
}

// TestGetSystemIDForLocation_EdgeCases tests edge cases
func TestGetSystemIDForLocation_EdgeCases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create minimal schema
	schema := `
		CREATE TABLE npcStations (
			_key INTEGER PRIMARY KEY,
			solarSystemID INTEGER
		);

		CREATE TABLE mapDenormalize (
			itemID INTEGER PRIMARY KEY,
			solarSystemID INTEGER
		);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	repo := NewSDERepository(db)
	ctx := context.Background()

	t.Run("Empty database - system ID in valid range", func(t *testing.T) {
		systemID, err := repo.GetSystemIDForLocation(ctx, 30000142)
		if err != nil {
			t.Errorf("Expected no error for valid system ID, got: %v", err)
		}
		if systemID != 30000142 {
			t.Errorf("Expected system ID 30000142, got %d", systemID)
		}
	})

	t.Run("Empty database - invalid location", func(t *testing.T) {
		_, err := repo.GetSystemIDForLocation(ctx, 12345)
		if err == nil {
			t.Error("Expected error for invalid location, got nil")
		}
	})

	t.Run("System ID boundary - lower bound", func(t *testing.T) {
		systemID, err := repo.GetSystemIDForLocation(ctx, 30000000)
		if err != nil {
			t.Errorf("Expected no error for lower boundary system ID, got: %v", err)
		}
		if systemID != 30000000 {
			t.Errorf("Expected system ID 30000000, got %d", systemID)
		}
	})

	t.Run("System ID boundary - upper bound minus 1", func(t *testing.T) {
		systemID, err := repo.GetSystemIDForLocation(ctx, 39999999)
		if err != nil {
			t.Errorf("Expected no error for upper boundary system ID, got: %v", err)
		}
		if systemID != 39999999 {
			t.Errorf("Expected system ID 39999999, got %d", systemID)
		}
	})

	t.Run("Just outside system ID range - too high", func(t *testing.T) {
		_, err := repo.GetSystemIDForLocation(ctx, 40000000)
		if err == nil {
			t.Error("Expected error for location just outside system ID range, got nil")
		}
	})
}

// TestGetSystemSecurityStatus guards against a regression where the query
// referenced a non-existent `security` column, making SQLite fail every
// statement so all systems defaulted to 1.0 (high-sec) and the security-zone
// filter could never surface low/null-sec routes.
//
// The schema below mirrors the real eve-sde `mapSolarSystems` table — note it
// has `securityStatus` and NO `security` column.
func TestGetSystemSecurityStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	schema := `
		CREATE TABLE mapSolarSystems (
			_key INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			regionID INTEGER NOT NULL,
			securityStatus REAL NOT NULL
		);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	testData := `
		INSERT INTO mapSolarSystems (_key, name, regionID, securityStatus) VALUES
			(30000142, 'Jita',     10000002, 0.9459),
			(30002659, 'Dodixie',  10000032, 0.8684),
			(30003504, 'Vestouve', 10000048, 0.04),
			(30000789, 'Null Hole',10000048, -0.12);
	`
	if _, err := db.Exec(testData); err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	repo := NewSDERepository(db)
	ctx := context.Background()

	tests := []struct {
		name     string
		systemID int64
		want     float64
		wantErr  bool
	}{
		{name: "high-sec (Jita)", systemID: 30000142, want: 0.9459},
		{name: "low-sec (Vestouve)", systemID: 30003504, want: 0.04},
		{name: "null-sec (negative)", systemID: 30000789, want: -0.12},
		{name: "unknown system → error", systemID: 39999999, want: 1.0, wantErr: true},
	}

	const eps = 1e-6
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetSystemSecurityStatus(ctx, tt.systemID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for system %d, got nil", tt.systemID)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for system %d: %v", tt.systemID, err)
			}
			if diff := got - tt.want; diff > eps || diff < -eps {
				t.Errorf("GetSystemSecurityStatus(%d) = %v, want %v", tt.systemID, got, tt.want)
			}
		})
	}
}
