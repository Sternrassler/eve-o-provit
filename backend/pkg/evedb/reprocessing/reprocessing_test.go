package reprocessing

import (
	"strings"
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

func TestNetYield(t *testing.T) {
	// base 0.50, Reprocessing 5 (+15%), Reprocessing Efficiency 5 (+10%), ore skill 4 (+8%)
	got := NetYield(0.50, ReprocessingSkills{Reprocessing: 5, ReprocessingEfficiency: 5, OreProcessing: 4})
	want := 0.50 * 1.15 * 1.10 * 1.08
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("want %.6f got %.6f", want, got)
	}
}

func TestListOres_RealNamesAndNoGrade(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()
	ores, err := ListOres(db)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]string{}
	for _, o := range ores {
		byID[o.TypeID] = o.Name
		if strings.Contains(o.Name, "-Grade") {
			t.Errorf("'-Grade' name leaked: %d %q", o.TypeID, o.Name)
		}
	}
	if byID[17470] != "Concentrated Veldspar" {
		t.Errorf("17470: got %q, want Concentrated Veldspar", byID[17470])
	}
	if byID[17444] != "Vivid Hemorphite" {
		t.Errorf("17444: got %q, want Vivid Hemorphite", byID[17444])
	}
	if byID[1230] != "Veldspar" {
		t.Errorf("1230: got %q, want Veldspar", byID[1230])
	}
	if _, ok := byID[46689]; ok { // Veldspar IV-Grade — no blueprint name → filtered
		t.Errorf("Veldspar IV-Grade (46689) must be filtered out")
	}
}

func TestListOres_ExcludesCompressedVariants(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ores, err := ListOres(db)
	if err != nil {
		t.Fatal(err)
	}
	in := map[int64]bool{}
	for _, o := range ores {
		in[o.TypeID] = true
		if strings.Contains(o.Name, "Compressed") {
			t.Fatalf("compressed variant leaked into ore list: %d %q", o.TypeID, o.Name)
		}
	}
	if !in[1230] || !in[17470] { // base Veldspar + Veldspar II-Grade (raw, mineable) stay
		t.Fatal("raw Veldspar variants must remain in the list")
	}
	if in[62516] || in[28430] { // Compressed Veldspar + Batch Compressed Veldspar II-Grade gone
		t.Fatal("compressed/batch-compressed variants must be excluded")
	}
}
