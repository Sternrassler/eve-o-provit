package mining

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

func has(m map[int64]bool, ids ...int64) bool {
	for _, id := range ids {
		if !m[id] {
			return false
		}
	}
	return true
}

func TestSystemQuarterAndSec(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	q, sec, err := SystemQuarterAndSec(db, 30000142) // Jita
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if q != "caldari" {
		t.Errorf("Jita quarter: got %q, want caldari", q)
	}
	if sec < 0.94 || sec > 0.95 {
		t.Errorf("Jita sec: got %v, want ~0.946", sec)
	}
}

func TestAvailableOreGroups(t *testing.T) {
	// Jita (Caldari, displayed 0.9): Veldspar 462, Scordite 460, Pyroxeres 459; no Plagioclase 458.
	g := AvailableOreGroups("caldari", 0.946, false)
	if !has(g, 462, 460, 459) || g[458] || len(g) != 3 {
		t.Errorf("caldari 0.9: got %v", g)
	}
	// Gallente 0.7 hi-sec: + Plagioclase.
	g = AvailableOreGroups("gallente", 0.7, false)
	if !has(g, 462, 460, 459, 458) || len(g) != 4 {
		t.Errorf("gallente 0.7: got %v", g)
	}
	// Amarr 0.6 hi-sec: no Plagioclase.
	g = AvailableOreGroups("amarr", 0.6, false)
	if !has(g, 462, 460, 459) || g[458] {
		t.Errorf("amarr 0.6: got %v", g)
	}
	// Amarr 0.3 low-sec, allowLow=true: Pyroxeres 459, Kernite 457, Jaspet 456; no Hemorphite 455, no Veldspar.
	g = AvailableOreGroups("amarr", 0.3, true)
	if !has(g, 459, 457, 456) || g[455] || g[462] {
		t.Errorf("amarr 0.3 low: got %v", g)
	}
	// Amarr 0.2 low: + Hemorphite.
	if g := AvailableOreGroups("amarr", 0.2, true); !g[455] {
		t.Errorf("amarr 0.2 must include Hemorphite")
	}
	// Low-sec but high-only → empty.
	if g := AvailableOreGroups("amarr", 0.3, false); len(g) != 0 {
		t.Errorf("low+high-only must be empty: %v", g)
	}
	// Null → empty even with allowLow.
	if g := AvailableOreGroups("amarr", 0.0, true); len(g) != 0 {
		t.Errorf("null must be empty: %v", g)
	}
	// Unknown quarter, hi-sec: quarter-independent ores only.
	if g := AvailableOreGroups("", 0.8, false); !has(g, 462, 460, 459) || g[458] {
		t.Errorf("unknown quarter: got %v", g)
	}
}
