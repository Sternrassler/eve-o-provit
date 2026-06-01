package mining

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

func TestHullMiningYieldMultiplier(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	// Hulk (22544): miningBargeBonus(+3%/lvl, skill 17940) × exhumersBonus(+6%/lvl, skill 22551).
	// At Mining Barge V + Exhumers V: (1+0.03*5)*(1+0.06*5) = 1.15*1.30 = 1.495.
	mul, resolved, err := HullMiningYieldMultiplier(db, 22544, map[int64]int{17940: 5, 22551: 5})
	if err != nil {
		t.Fatalf("HullMiningYieldMultiplier err: %v", err)
	}
	if !resolved {
		t.Error("Hulk bonuses must be fully resolved")
	}
	if mul < 1.4949 || mul > 1.4951 {
		t.Errorf("Hulk V/V mult: got %v, want 1.495", mul)
	}

	// Zero skills → no bonus applied (multiplier 1.0) but still resolved.
	mul, resolved, err = HullMiningYieldMultiplier(db, 22544, map[int64]int{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !resolved || mul < 0.9999 || mul > 1.0001 {
		t.Errorf("Hulk zero-skill: got mul=%v resolved=%v, want 1.0/true", mul, resolved)
	}

	// A non-mining hull (Ibis frigate, 601) has no attr-77 ship bonus → 1.0, resolved.
	mul, resolved, err = HullMiningYieldMultiplier(db, 601, map[int64]int{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !resolved || mul < 0.9999 || mul > 1.0001 {
		t.Errorf("Ibis: got mul=%v resolved=%v, want 1.0/true", mul, resolved)
	}

	// Venture (32880) carries an unrecognised role bonus on attr 77 → resolved=false
	// (we never partially-compute a hull's yield and pass it off as exact).
	_, resolved, err = HullMiningYieldMultiplier(db, 32880, map[int64]int{32918: 5})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if resolved {
		t.Error("Venture has an unrecognised attr-77 bonus → resolved must be false")
	}
}
