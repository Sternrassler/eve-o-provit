package reprocessing

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	_ "github.com/mattn/go-sqlite3"
)

func TestOreOutput_Veldspar(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ore, err := GetOre(db, 1230)
	if err != nil {
		t.Fatal(err)
	}
	if ore.PortionSize != 100 || ore.VolumeM3 != 0.1 {
		t.Fatalf("portion/volume: %d / %v", ore.PortionSize, ore.VolumeM3)
	}
	if len(ore.Materials) != 1 || ore.Materials[0].MaterialTypeID != 34 || ore.Materials[0].Quantity != 400 {
		t.Fatalf("materials: %+v", ore.Materials)
	}
}

func TestListOres_IncludesVeldsparAndExcludesNonOre(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ores, err := ListOres(db)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, o := range ores {
		if o.TypeID == 1230 {
			found = true
		}
	}
	if !found {
		t.Fatal("Veldspar (1230) missing from ore list")
	}
	if len(ores) < 10 {
		t.Fatalf("expected many ores, got %d", len(ores))
	}
}
