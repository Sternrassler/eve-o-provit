package mining

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

func TestCrystalCapable(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	// Modulated Strip Miner II (17912) loads Mining Crystals → capable.
	ok, err := CrystalCapable(db, []int64{17912})
	if err != nil {
		t.Fatalf("CrystalCapable err: %v", err)
	}
	if !ok {
		t.Error("Modulated Strip Miner II must be crystal-capable")
	}

	// Strip Miner I (17482) has no crystal charge group → not capable.
	ok, err = CrystalCapable(db, []int64{17482})
	if err != nil {
		t.Fatalf("CrystalCapable err: %v", err)
	}
	if ok {
		t.Error("Strip Miner I must NOT be crystal-capable")
	}
}

func TestOreCrystalMultiplierT2(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	// Veldspar group (462) → Veldspar Mining Crystal II, attr 782 = 1.75.
	mul, found, err := OreCrystalMultiplierT2(db, 462)
	if err != nil {
		t.Fatalf("OreCrystalMultiplierT2 err: %v", err)
	}
	if !found {
		t.Fatal("Veldspar group must have a T2 crystal")
	}
	if mul < 1.749 || mul > 1.751 {
		t.Errorf("Veldspar T2 crystal mult: got %v, want 1.75", mul)
	}

	// Mercoxit group (468) has no name-matching crystal → not found (no silent 1.0).
	_, found, err = OreCrystalMultiplierT2(db, 468)
	if err != nil {
		t.Fatalf("OreCrystalMultiplierT2 err: %v", err)
	}
	if found {
		t.Error("Mercoxit group must report no matching T2 crystal (found=false)")
	}
}
