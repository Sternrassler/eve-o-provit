package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
	_ "github.com/mattn/go-sqlite3"
)

// --- Fakes ---------------------------------------------------------------

type fakeMiningMarket struct {
	// buyPriceByType maps typeID → highest buy price returned as a single buy order.
	buyPriceByType map[int]float64
}

func (f *fakeMiningMarket) GetMarketOrders(_ context.Context, _ int, typeID int) ([]database.MarketOrder, error) {
	price, ok := f.buyPriceByType[typeID]
	if !ok {
		return []database.MarketOrder{}, nil
	}
	return []database.MarketOrder{
		{TypeID: typeID, IsBuyOrder: true, Price: price, VolumeRemain: 1_000_000, LocationID: 60000123},
		// a lower buy order + a sell order, to prove we pick the highest buy.
		{TypeID: typeID, IsBuyOrder: true, Price: price * 0.5, VolumeRemain: 100, LocationID: 60000999},
		{TypeID: typeID, IsBuyOrder: false, Price: price * 2, VolumeRemain: 100},
	}, nil
}

// fakeMiningNames resolves canned station/system/type names. Station 60000001 is the
// reprocess station; 60000123 is the sell-order station (both NPC); others → structure.
type fakeMiningNames struct{}

func (fakeMiningNames) GetStationName(_ context.Context, id int64) (string, error) {
	switch id {
	case 60000001:
		return "Test Reprocess Station", nil
	case 60000123:
		return "Jita IV-4", nil
	}
	return fmt.Sprintf("Station-%d", id), nil // unknown → resolver treats as structure
}

func (fakeMiningNames) GetSystemName(_ context.Context, _ int64) (string, error) { return "Jita", nil }

func (fakeMiningNames) GetSystemIDForLocation(_ context.Context, _ int64) (int64, error) {
	return 30000142, nil
}

func (fakeMiningNames) GetTypeInfo(_ context.Context, typeID int) (*database.TypeInfo, error) {
	if typeID == tritaniumTypeID {
		return &database.TypeInfo{Name: "Tritanium"}, nil
	}
	return &database.TypeInfo{Name: ""}, nil
}

type fakeStations struct{ stations []database.ReprocessStation }

func (f *fakeStations) GetRegionReprocessStations(_ context.Context, _ int) ([]database.ReprocessStation, error) {
	return f.stations, nil
}

type fakeMiningModules struct{ ids []int64 }

func (f *fakeMiningModules) ActiveShipFittedModuleTypeIDs(_ context.Context, _ int, _ string) ([]int64, error) {
	return f.ids, nil
}

// fakeMiningSkillsProvider implements MiningSkillsProvider with fixed skills + standings.
type fakeMiningSkillsProvider struct {
	skills    *MiningReprocessingSkills
	standings map[int64]float64
}

func (f *fakeMiningSkillsProvider) GetMiningReprocessingSkills(_ context.Context, _ *sql.DB, _ int, _ string) (*MiningReprocessingSkills, error) {
	return f.skills, nil
}

func (f *fakeMiningSkillsProvider) GetCharacterStandings(_ context.Context, _ int, _ string) (map[int64]float64, error) {
	return f.standings, nil
}

// --- Test ---------------------------------------------------------------

const (
	veldsparTypeID    = 1230
	stripMinerITypeID = 17482
	tritaniumTypeID   = 34
)

// TestMiningService_OreRanking_Veldspar verifies a Veldspar row is produced with the
// correct Best verdict and per-m³ values consistent with CompareOre, and that isk/h
// equals m3h × per-m³ when a mining module is fitted.
func TestMiningService_OreRanking_Veldspar(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	// Known buy prices: high Veldspar buy (raw favored) — pick numbers where raw wins.
	const veldBuy = 15.0
	const tritBuy = 5.0
	market := &fakeMiningMarket{buyPriceByType: map[int]float64{
		veldsparTypeID:  veldBuy,
		tritaniumTypeID: tritBuy,
	}}

	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}}, // all 0
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}

	svc := NewMiningService(sdeDB, stations, market, skills, fitting, nil, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID: 10000002,
		SecBand:  "high",
	})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	if resp.NoMiningSetup {
		t.Fatal("expected NoMiningSetup=false (Strip Miner fitted)")
	}

	// Find the Veldspar row.
	var row *models.OreRankRow
	for i := range resp.Rows {
		if resp.Rows[i].OreTypeID == veldsparTypeID {
			row = &resp.Rows[i]
			break
		}
	}
	if row == nil {
		t.Fatal("Veldspar row not found in response")
	}
	if row.Best == "" {
		t.Error("Best verdict not set")
	}

	// Recompute the expected CompareOre result independently and assert consistency.
	// Veldspar (typeID 1230): portionSize 100, volume 0.1 m³, material Tritanium ×400.
	net := 0.50 // NetYield at all-zero reprocessing skills = baseRate
	want := CompareOre(OreCompareInput{
		PortionSize:  100,
		OreVolumeM3:  0.1,
		OreBuyPrice:  veldBuy,
		Materials:    []MaterialValue{{Qty: 400, BuyPrice: tritBuy}},
		NetYield:     net,
		StationTake:  ReprocessTax(0.05, 0), // base take, neutral standing
		SalesTaxRate: SalesTaxRate(0),
	})
	if !approxEq(row.RawNetPerM3, want.RawNetPerM3) {
		t.Errorf("RawNetPerM3: got %v, want %v", row.RawNetPerM3, want.RawNetPerM3)
	}
	if !approxEq(row.RefineNetPerM3, want.RefineNetPerM3) {
		t.Errorf("RefineNetPerM3: got %v, want %v", row.RefineNetPerM3, want.RefineNetPerM3)
	}
	if row.Best != want.Best {
		t.Errorf("Best: got %q, want %q", row.Best, want.Best)
	}

	// isk/h = m3h × per-m³. Strip Miner I: 150 m³ / 45 s = 12000 m³/h at zero skills.
	const wantM3H = 150.0 / 45.0 * 3600.0
	if !approxEq(row.MiningM3PerHour, wantM3H) {
		t.Errorf("MiningM3PerHour: got %v, want %v", row.MiningM3PerHour, wantM3H)
	}
	if !approxEq(row.RawISKPerHour, wantM3H*row.RawNetPerM3) {
		t.Errorf("RawISKPerHour: got %v, want %v", row.RawISKPerHour, wantM3H*row.RawNetPerM3)
	}
	if !approxEq(row.RefineISKPerHour, wantM3H*row.RefineNetPerM3) {
		t.Errorf("RefineISKPerHour: got %v, want %v", row.RefineISKPerHour, wantM3H*row.RefineNetPerM3)
	}
	if row.BestStationID != 60000001 {
		t.Errorf("BestStationID: got %d, want 60000001", row.BestStationID)
	}

	// Locations: reprocess station name+system, raw sell location, per-mineral breakdown.
	if row.BestStationName != "Test Reprocess Station" || row.BestStationSystem != "Jita" {
		t.Errorf("reprocess station: name=%q system=%q", row.BestStationName, row.BestStationSystem)
	}
	if row.RawSell == nil || row.RawSell.IsStructure || row.RawSell.StationName != "Jita IV-4" || row.RawSell.SystemName != "Jita" {
		t.Errorf("RawSell: %+v", row.RawSell)
	}
	if len(row.Materials) != 1 {
		t.Fatalf("Materials: want 1 (Tritanium), got %d", len(row.Materials))
	}
	mat := row.Materials[0]
	if mat.MaterialTypeID != tritaniumTypeID || mat.MaterialName != "Tritanium" {
		t.Errorf("material: id=%d name=%q", mat.MaterialTypeID, mat.MaterialName)
	}
	if mat.EffectiveQty != 200 { // floor(400 × 0.50)
		t.Errorf("EffectiveQty: got %d, want 200", mat.EffectiveQty)
	}
	if mat.BuyPrice != tritBuy || mat.Sell.StationName != "Jita IV-4" {
		t.Errorf("material sell: price=%v sell=%+v", mat.BuyPrice, mat.Sell)
	}
}

// TestMiningService_OreRanking_NoMiningSetup verifies that with no fitted mining modules
// the response flags NoMiningSetup and isk/h is 0, while per-m³ values are still computed.
func TestMiningService_OreRanking_NoMiningSetup(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	market := &fakeMiningMarket{buyPriceByType: map[int]float64{
		veldsparTypeID:  15.0,
		tritaniumTypeID: 5.0,
	}}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: nil} // no modules
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}

	svc := NewMiningService(sdeDB, stations, market, skills, fitting, nil, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID: 10000002,
		SecBand:  "high",
	})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	if !resp.NoMiningSetup {
		t.Fatal("expected NoMiningSetup=true (no modules)")
	}
	if len(resp.Rows) == 0 {
		t.Fatal("expected per-m³ rows even with no mining setup")
	}
	for _, r := range resp.Rows {
		if r.RawISKPerHour != 0 || r.RefineISKPerHour != 0 {
			t.Errorf("isk/h must be 0 with no mining setup: %+v", r)
		}
		if r.MiningM3PerHour != 0 {
			t.Errorf("MiningM3PerHour must be 0 with no mining setup: %v", r.MiningM3PerHour)
		}
	}
}

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-6 }
