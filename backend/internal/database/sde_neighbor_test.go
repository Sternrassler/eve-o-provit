package database

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestGetNeighborRegions builds a minimal in-memory graph: region 10 has systems 1,2;
// region 20 has system 3; region 30 has system 4. Gates: 1↔3 (cross to 20),
// 1↔2 (same region 10), 2↔4 (cross to 30). Neighbors of 10 must be {20, 30}.
func TestGetNeighborRegions(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE mapSolarSystems (_key INTEGER PRIMARY KEY, regionID INTEGER NOT NULL)`,
		`CREATE TABLE v_stargate_graph (from_system_id INTEGER, to_system_id INTEGER)`,
		`INSERT INTO mapSolarSystems (_key, regionID) VALUES (1,10),(2,10),(3,20),(4,30)`,
		// bidirectional edges (the real view emits both directions too)
		`INSERT INTO v_stargate_graph (from_system_id, to_system_id) VALUES
			(1,3),(3,1), (1,2),(2,1), (2,4),(4,2)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("setup %q: %v", s, err)
		}
	}

	repo := NewSDERepository(db)
	got, err := repo.GetNeighborRegions(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetNeighborRegions: %v", err)
	}
	if len(got) != 2 || got[0] != 20 || got[1] != 30 {
		t.Errorf("neighbors of region 10 = %v, want [20 30]", got)
	}
}
