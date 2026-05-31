package database

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestGetRegionReprocessStations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	schema := `
		CREATE TABLE npcStations (_key INTEGER PRIMARY KEY, ownerID INTEGER,
			reprocessingEfficiency REAL, reprocessingStationsTake REAL, solarSystemID INTEGER);
		CREATE TABLE mapSolarSystems (_key INTEGER PRIMARY KEY, constellationID INTEGER);
		CREATE TABLE mapConstellations (_key INTEGER PRIMARY KEY, regionID INTEGER);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	data := `
		INSERT INTO mapConstellations (_key, regionID) VALUES (20000020, 10000002), (20000999, 10000099);
		INSERT INTO mapSolarSystems (_key, constellationID) VALUES (30000142, 20000020), (30009999, 20000999);
		INSERT INTO npcStations (_key, ownerID, reprocessingEfficiency, reprocessingStationsTake, solarSystemID) VALUES
			(60003760, 1000035, 0.5, 0.05, 30000142),   -- in region 10000002, reprocess
			(60000001, 1000040, 0.0, 0.05, 30000142),   -- in region, NO reprocess (excluded)
			(60009999, 1000050, 0.5, 0.05, 30009999);   -- other region (excluded)
	`
	if _, err := db.Exec(data); err != nil {
		t.Fatalf("data: %v", err)
	}

	repo := NewSDERepository(db)
	st, err := repo.GetRegionReprocessStations(context.Background(), 10000002)
	if err != nil {
		t.Fatal(err)
	}
	if len(st) != 1 {
		t.Fatalf("want 1 station, got %d: %+v", len(st), st)
	}
	s := st[0]
	if s.StationID != 60003760 || s.OwnerCorpID != 1000035 || s.BaseRate != 0.5 || s.BaseTake != 0.05 {
		t.Fatalf("bad station: %+v", s)
	}
}
