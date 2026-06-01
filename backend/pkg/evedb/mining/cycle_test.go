package mining

import (
	"math"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

func approxEqCycle(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestOreHoldCapacity(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	// Hulk (22544): ore hold 11500 (attr 1556), not the 350 cargo.
	cap, found, err := OreHoldCapacity(db, 22544)
	if err != nil || !found {
		t.Fatalf("Hulk ore hold: found=%v err=%v", found, err)
	}
	if cap != 11500 {
		t.Errorf("Hulk ore hold: got %v, want 11500", cap)
	}

	// Ibis (601): no ore hold → falls back to types.capacity 125.
	cap, found, err = OreHoldCapacity(db, 601)
	if err != nil || !found {
		t.Fatalf("Ibis capacity: found=%v err=%v", found, err)
	}
	if cap != 125 {
		t.Errorf("Ibis capacity: got %v, want 125", cap)
	}
}

func TestEffectiveISKPerHour(t *testing.T) {
	eff, cycleMin, fillMin := EffectiveISKPerHour(12000, 12000, 100, 600, 75)
	if !approxEqCycle(eff, 1_200_000.0/1.1875) {
		t.Errorf("eff: got %v, want %v", eff, 1_200_000.0/1.1875)
	}
	if !approxEqCycle(cycleMin, 1.1875*60) {
		t.Errorf("cycleMin: got %v, want %v", cycleMin, 1.1875*60)
	}
	if !approxEqCycle(fillMin, 60) {
		t.Errorf("fillMin: got %v, want 60", fillMin)
	}

	eff0, cycle0, _ := EffectiveISKPerHour(12000, 12000, 100, 0, 75)
	wantCycle0 := (1.0 + 75.0/3600.0)
	if !approxEqCycle(cycle0, wantCycle0*60) {
		t.Errorf("same-system cycleMin: got %v, want %v", cycle0, wantCycle0*60)
	}
	if !approxEqCycle(eff0, 1_200_000.0/wantCycle0) {
		t.Errorf("same-system eff: got %v, want %v", eff0, 1_200_000.0/wantCycle0)
	}

	if eff, _, _ := EffectiveISKPerHour(12000, 0, 100, 600, 75); eff != 0 {
		t.Errorf("zero rate eff: got %v, want 0", eff)
	}
}
