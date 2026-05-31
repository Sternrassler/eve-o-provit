package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

func openLocSDE(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	schema := `
		CREATE TABLE npcStations (_key INTEGER PRIMARY KEY, solarSystemID INTEGER, typeID INTEGER);
		CREATE TABLE types (_key INTEGER PRIMARY KEY, name TEXT);
		CREATE TABLE mapSolarSystems (_key INTEGER PRIMARY KEY, name TEXT);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	data := `
		INSERT INTO types (_key, name) VALUES (1531, '{"en":"Caldari Navy Assembly Plant"}');
		INSERT INTO mapSolarSystems (_key, name) VALUES (30000142, '{"en":"Jita"}');
		INSERT INTO npcStations (_key, solarSystemID, typeID) VALUES (60003760, 30000142, 1531);
	`
	if _, err := db.Exec(data); err != nil {
		t.Fatalf("data: %v", err)
	}
	return db
}

func TestResolveSellLocation(t *testing.T) {
	db := openLocSDE(t)
	defer func() { _ = db.Close() }()
	repo := database.NewSDERepository(db)
	r := newLocResolver(repo)
	ctx := context.Background()

	npc := r.resolve(ctx, 60003760)
	if npc.IsStructure {
		t.Fatalf("NPC station should not be a structure: %+v", npc)
	}
	if npc.StationName != "Caldari Navy Assembly Plant" || npc.SystemName != "Jita" {
		t.Fatalf("npc resolve: %+v", npc)
	}

	cit := r.resolve(ctx, 1_035_000_000_001) // citadel id ≥ 1e12
	if !cit.IsStructure || cit.StationName != "" || cit.SystemName != "" {
		t.Fatalf("citadel should resolve to structure-only: %+v", cit)
	}
}
