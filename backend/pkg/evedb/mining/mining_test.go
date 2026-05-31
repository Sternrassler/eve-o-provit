package mining

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	_ "github.com/mattn/go-sqlite3"
)

func TestMiningRate_OneStripMinerI(t *testing.T) {
	db := testutil.OpenTestDB(t)
	// Strip Miner I (17482): attr77=150 m³, attr73=45000 ms. Mining 5 (+25%), Astrogeology 5 (+25%).
	m3h, err := MiningRateM3PerHour(db, []int64{17482}, MiningSkills{Mining: 5, Astrogeology: 5}, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	// per cycle 150 × 1.25 × 1.25 = 234.375 m³ / 45 s × 3600 = 18750 m³/h
	if m3h < 18740 || m3h > 18760 {
		t.Fatalf("m3/h %.1f (want ~18750)", m3h)
	}
}

func TestMiningRate_NonMiningModuleSkipped(t *testing.T) {
	db := testutil.OpenTestDB(t)
	// A non-mining type (e.g. Tritanium 34) has no miningAmount → contributes 0.
	m3h, err := MiningRateM3PerHour(db, []int64{34}, MiningSkills{Mining: 5, Astrogeology: 5}, 1.0)
	if err != nil {
		t.Fatal(err)
	}
	if m3h != 0 {
		t.Fatalf("non-mining module should contribute 0, got %.1f", m3h)
	}
}
