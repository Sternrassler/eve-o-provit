package services

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
	_ "github.com/mattn/go-sqlite3"
)

// errESI is a sentinel for simulating an ESI fetch failure (e.g. expired token).
var errESI = fmt.Errorf("esi unavailable (401)")

// --- Fakes ---------------------------------------------------------------

type fakeMiningMarket struct {
	// buyPriceByType maps typeID → highest buy price returned as a single buy order.
	buyPriceByType map[int]float64
	// locByType overrides the highest-buy-order station per type (default 60000123).
	locByType map[int]int64
	// volByType overrides the highest buy order's VolumeRemain per type (default 1_000_000).
	volByType map[int]int
}

func (f *fakeMiningMarket) GetMarketOrders(_ context.Context, _ int, typeID int) ([]database.MarketOrder, error) {
	price, ok := f.buyPriceByType[typeID]
	if !ok {
		return []database.MarketOrder{}, nil
	}
	loc := int64(60000123)
	if f.locByType != nil {
		if l, ok := f.locByType[typeID]; ok {
			loc = l
		}
	}
	// Lower buy-order location: same override when set (keeps all orders unreachable),
	// otherwise the default NPC station 60000999 (to prove we pick the highest buy).
	lowerLoc := int64(60000999)
	if f.locByType != nil {
		if l, ok := f.locByType[typeID]; ok {
			lowerLoc = l
		}
	}
	topVol := 1_000_000
	if f.volByType != nil {
		if v, ok := f.volByType[typeID]; ok {
			topVol = v
		}
	}
	return []database.MarketOrder{
		{TypeID: typeID, IsBuyOrder: true, Price: price, VolumeRemain: topVol, LocationID: loc},
		// a lower buy order + a sell order, to prove we pick the highest buy.
		{TypeID: typeID, IsBuyOrder: true, Price: price * 0.5, VolumeRemain: 100, LocationID: lowerLoc},
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

func (fakeMiningNames) GetSystemIDForLocation(_ context.Context, id int64) (int64, error) {
	if id >= 1_000_000_000_000 {
		return 0, fmt.Errorf("no system for structure %d", id)
	}
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

type fakeMiningModules struct {
	ids        []int64
	modulesErr error // when set, ActiveShipFittedModuleTypeIDs fails (exercise fitting-degraded path)
}

func (f *fakeMiningModules) ActiveShipFittedModuleTypeIDs(_ context.Context, _ int, _ string) ([]int64, error) {
	if f.modulesErr != nil {
		return nil, f.modulesErr
	}
	return f.ids, nil
}

func (f *fakeMiningModules) GetShipFitting(_ context.Context, _, _ int, _ string) (*FittingData, error) {
	return &FittingData{Bonuses: FittingBonuses{WarpSpeedAUS: 3.0, AlignTime: 6.0}}, nil
}

// fakeMiningSkillsProvider implements MiningSkillsProvider with fixed skills + standings.
// skillsErr/standingsErr exercise the degradation (fail-loud) path.
type fakeMiningSkillsProvider struct {
	skills       *MiningReprocessingSkills
	standings    map[int64]float64
	skillsErr    error
	standingsErr error
}

func (f *fakeMiningSkillsProvider) GetMiningReprocessingSkills(_ context.Context, _ *sql.DB, _ int, _ string) (*MiningReprocessingSkills, error) {
	if f.skillsErr != nil {
		return nil, f.skillsErr
	}
	return f.skills, nil
}

func (f *fakeMiningSkillsProvider) GetCharacterStandings(_ context.Context, _ int, _ string) (map[int64]float64, error) {
	if f.standingsErr != nil {
		return nil, f.standingsErr
	}
	return f.standings, nil
}

// fakeMiningLocation implements MiningLocationProvider: a canned active-ship type id
// (for the hull mining-yield bonus) and an optional error to exercise the fail-loud path.
type fakeMiningLocation struct {
	shipTypeID int
	shipErr    error
}

func (f fakeMiningLocation) GetCharacterLocation(_ context.Context, _ int, _ string) (*CharacterLocation, error) {
	return &CharacterLocation{SolarSystemID: 30000142}, nil // Jita
}

func (f fakeMiningLocation) GetActiveShipTypeID(_ context.Context, _ int, _ string) (int, error) {
	return f.shipTypeID, f.shipErr
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
		// Reprocessing skills all 0; Mining Barge V + Exhumers V so the Hulk hull resolves.
		skills: &MiningReprocessingSkills{
			OreProcessing: map[int64]int{},
			SkillLevels:   map[int64]int{17940: 5, 22551: 5},
		},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}

	// Active ship = Hulk (22544): miningBarge+exhumers bonus at V/V → hullMul 1.495.
	loc := fakeMiningLocation{shipTypeID: 22544}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID:    10000002,
		AllowLowSec: false,
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

	// System-accurate ore set: Jita (Caldari 0.9) → only hi-sec ores, no low-sec leak.
	for _, r := range resp.Rows {
		if r.OreTypeID == 1231 || r.OreTypeID == 21 { // Hemorphite / Hedbergite (low-sec)
			t.Errorf("low-sec ore leaked into hi-sec ranking: %d %q", r.OreTypeID, r.OreName)
		}
	}
	if resp.Quarter != "caldari" {
		t.Errorf("Quarter: got %q, want caldari", resp.Quarter)
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

	// Base rate: Strip Miner I = 150 m³ / 45 s = 12000 m³/h at zero mining skills.
	// The Hulk hull applies a 1.495× yield bonus; Strip Miner I has no crystals (mul 1.0).
	const baseM3H = 150.0 / 45.0 * 3600.0
	const wantM3H = baseM3H * 1.495
	if row.HullYieldMultiplier < 1.4949 || row.HullYieldMultiplier > 1.4951 {
		t.Errorf("HullYieldMultiplier: got %v, want 1.495", row.HullYieldMultiplier)
	}
	if row.CrystalMultiplier != 1.0 {
		t.Errorf("CrystalMultiplier: got %v, want 1.0 (Strip Miner I has no crystals)", row.CrystalMultiplier)
	}
	if row.IsEstimate {
		t.Errorf("row should be exact (Hulk resolved, no crystal needed): %+v", *row)
	}
	if !approxEq(row.MiningM3PerHour, wantM3H) {
		t.Errorf("MiningM3PerHour with hull bonus: got %v, want %v", row.MiningM3PerHour, wantM3H)
	}
	if !approxEq(row.RawISKPerHour, row.MiningM3PerHour*row.RawNetPerM3) {
		t.Errorf("RawISKPerHour: got %v, want %v", row.RawISKPerHour, row.MiningM3PerHour*row.RawNetPerM3)
	}
	if !approxEq(row.RefineISKPerHour, row.MiningM3PerHour*row.RefineNetPerM3) {
		t.Errorf("RefineISKPerHour: got %v, want %v", row.RefineISKPerHour, row.MiningM3PerHour*row.RefineNetPerM3)
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
	// Effective ISK/h (haul-downtime cycle). Active ship = Hulk (ore hold 11500).
	// Origin/sell/reprocess all resolve to system 30000142 in this fake → travel 0,
	// so the only downtime is the per-stop overhead.
	if row.LoadVolumeM3 != 11500 {
		t.Errorf("LoadVolumeM3: got %v, want 11500 (Hulk ore hold)", row.LoadVolumeM3)
	}
	if row.EffectiveISKPerHour <= 0 {
		t.Error("EffectiveISKPerHour should be set (cycle resolvable)")
	}
	gross := row.MiningM3PerHour * row.RawNetPerM3
	if row.Best == "raw" && row.EffectiveISKPerHour >= gross {
		t.Errorf("effective (%v) must be below gross (%v) due to stop overhead", row.EffectiveISKPerHour, gross)
	}
	if row.IsEstimate {
		t.Errorf("row should be resolvable (not estimate): %+v", *row)
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

	loc := fakeMiningLocation{shipTypeID: 601} // Ibis: a real ship with no mining bonus
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID:    10000002,
		AllowLowSec: false,
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

// TestMiningService_OreRanking_EstimateWhenShipUnknown verifies fail-loud behaviour:
// when the active ship can't be resolved the hull bonus is unknown, so every row is
// marked IsEstimate with a reason and the multiplier is not fabricated (stays 1.0).
func TestMiningService_OreRanking_EstimateWhenShipUnknown(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	market := &fakeMiningMarket{buyPriceByType: map[int]float64{veldsparTypeID: 15.0, tritaniumTypeID: 5.0}}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{}},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}

	// Active-ship lookup fails → hull bonus unresolved → every row is an estimate.
	loc := fakeMiningLocation{shipErr: fmt.Errorf("esi down")}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002, AllowLowSec: false})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	if len(resp.Rows) == 0 {
		t.Fatal("expected ore rows")
	}
	for _, r := range resp.Rows {
		if !r.IsEstimate || r.EstimateReason == "" {
			t.Errorf("row must be estimate when ship unknown: %+v", r)
		}
		if r.HullYieldMultiplier != 1.0 {
			t.Errorf("unresolved hull must not fabricate a bonus: %v", r.HullYieldMultiplier)
		}
	}
}

// TestMiningService_OreRanking_VariantRealName verifies an ore VARIANT row shows the
// in-game "<Ore> II-Grade" name paired with our client-name adjective in parens.
// 17470 = SDE "Veldspar II-Grade", client "Concentrated Veldspar" → "Veldspar II-Grade (Concentrated)".
func TestMiningService_OreRanking_VariantRealName(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	const concentratedVeldsparTypeID = 17470 // group 462 (Veldspar), in Jita's hi-sec set
	market := &fakeMiningMarket{buyPriceByType: map[int]float64{
		concentratedVeldsparTypeID: 20.0,
		tritaniumTypeID:            5.0,
	}}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{}},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}
	loc := fakeMiningLocation{shipTypeID: 22544}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID:    10000002,
		AllowLowSec: false,
	})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	var row *models.OreRankRow
	for i := range resp.Rows {
		if resp.Rows[i].OreTypeID == concentratedVeldsparTypeID {
			row = &resp.Rows[i]
			break
		}
	}
	if row == nil {
		t.Fatal("Concentrated Veldspar (17470) row not found")
	}
	if row.OreName != "Veldspar II-Grade (Concentrated)" {
		t.Errorf("OreName: got %q, want %q (in-game name + our adjective)", row.OreName, "Veldspar II-Grade (Concentrated)")
	}
}

// TestMiningService_OreRanking_CitadelUnreachable verifies that an ore whose only raw
// buy order AND whose mineral buy orders are all in a citadel (≥1e12) is skipped
// entirely — not ranked, not estimated.
func TestMiningService_OreRanking_CitadelUnreachable(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	// Both Veldspar raw and its refine mineral (Tritanium) are only buyable in a citadel.
	market := &fakeMiningMarket{
		buyPriceByType: map[int]float64{
			veldsparTypeID:  15.0,
			tritaniumTypeID: 5.0,
		},
		locByType: map[int]int64{
			veldsparTypeID:  1_035_000_000_001,
			tritaniumTypeID: 1_035_000_000_001,
		},
	}
	skills := &fakeMiningSkillsProvider{
		skills: &MiningReprocessingSkills{
			OreProcessing: map[int64]int{},
			SkillLevels:   map[int64]int{17940: 5, 22551: 5},
		},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}
	loc := fakeMiningLocation{shipTypeID: 22544}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID:    10000002,
		AllowLowSec: false,
	})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}

	// Veldspar is fully unreachable (both raw and refine paths unreachable) → must be skipped.
	for _, r := range resp.Rows {
		if r.OreTypeID == veldsparTypeID {
			t.Errorf("Veldspar (both raw and refine in citadel) must be skipped, but found row: %+v", r)
		}
	}
}

// TestMiningService_OreRanking_DecoupledRawWhenRefineUnreachable verifies that when
// the refine path is unavailable (only reprocess station is a citadel) but the raw
// sell path is reachable, the Veldspar row resolves as "raw", is not an estimate,
// and has EffectiveISKPerHour > 0.
func TestMiningService_OreRanking_DecoupledRawWhenRefineUnreachable(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	// Raw Veldspar buy order at default NPC station (reachable). Tritanium also reachable.
	market := &fakeMiningMarket{
		buyPriceByType: map[int]float64{
			veldsparTypeID:  15.0,
			tritaniumTypeID: 5.0,
		},
		// No locByType override → highest buy at 60000123 (NPC, reachable).
	}
	skills := &fakeMiningSkillsProvider{
		skills: &MiningReprocessingSkills{
			OreProcessing: map[int64]int{},
			SkillLevels:   map[int64]int{17940: 5, 22551: 5},
		},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	// The ONLY reprocess station is a citadel → unreachable → refine unavailable for all ores.
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 1_035_000_000_009, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}
	loc := fakeMiningLocation{shipTypeID: 22544}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID:    10000002,
		AllowLowSec: false,
	})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}

	// Find the Veldspar row — it must exist (raw path is reachable).
	var row *models.OreRankRow
	for i := range resp.Rows {
		if resp.Rows[i].OreTypeID == veldsparTypeID {
			row = &resp.Rows[i]
			break
		}
	}
	if row == nil {
		t.Fatal("Veldspar row must be present (raw path is reachable even though refine is not)")
	}

	// Raw path wins because refine is unavailable.
	if row.Best != "raw" {
		t.Errorf("Best: got %q, want %q (refine unavailable → raw wins)", row.Best, "raw")
	}
	// IsEstimate is for hull/crystal resolution failures, not routing failures.
	if row.IsEstimate {
		t.Errorf("IsEstimate must be false (routing failure is not an estimate condition): %+v", *row)
	}
	// The raw sell station is reachable → effective ISK/h must be > 0.
	if row.EffectiveISKPerHour <= 0 {
		t.Errorf("EffectiveISKPerHour must be > 0 (raw path is reachable): %v", row.EffectiveISKPerHour)
	}
	// No reachable reprocess station → BestStationID must be 0.
	if row.BestStationID != 0 {
		t.Errorf("BestStationID: got %d, want 0 (no reachable reprocess station)", row.BestStationID)
	}
	// RawSell must be set.
	if row.RawSell == nil {
		t.Error("RawSell must be set (raw path is the only available path)")
	}
	// Only one path is a real choice → no meaningful raw-vs-refine delta.
	if row.DeltaISKPerHour != 0 {
		t.Errorf("DeltaISKPerHour must be 0 with only one reachable path: %v", row.DeltaISKPerHour)
	}
}

// TestMiningService_OreRanking_DecoupledRefineWhenRawUnreachable is the mirror of the
// raw-only case: when every raw buy order is in a citadel but a reachable reprocess
// station + mineral hub exist, the row resolves as "refine" (the !rawReachable verdict
// branch), with no RawSell and effective ISK/h > 0.
func TestMiningService_OreRanking_DecoupledRefineWhenRawUnreachable(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	// Veldspar raw is citadel-only (unreachable); its mineral Tritanium stays at the
	// default NPC station (reachable), so only the refine path can resolve.
	market := &fakeMiningMarket{
		buyPriceByType: map[int]float64{
			veldsparTypeID:  15.0,
			tritaniumTypeID: 5.0,
		},
		locByType: map[int]int64{
			veldsparTypeID: 1_035_000_000_001,
		},
	}
	skills := &fakeMiningSkillsProvider{
		skills: &MiningReprocessingSkills{
			OreProcessing: map[int64]int{},
			SkillLevels:   map[int64]int{17940: 5, 22551: 5},
		},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	// Reachable NPC reprocess station → refine path available.
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}
	loc := fakeMiningLocation{shipTypeID: 22544}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{
		RegionID:    10000002,
		AllowLowSec: false,
	})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}

	var row *models.OreRankRow
	for i := range resp.Rows {
		if resp.Rows[i].OreTypeID == veldsparTypeID {
			row = &resp.Rows[i]
			break
		}
	}
	if row == nil {
		t.Fatal("Veldspar row must be present (refine path is reachable even though raw is not)")
	}
	if row.Best != "refine" {
		t.Errorf("Best: got %q, want %q (raw unreachable → refine wins)", row.Best, "refine")
	}
	if row.IsEstimate {
		t.Errorf("IsEstimate must be false (routing failure is not an estimate condition): %+v", *row)
	}
	if row.EffectiveISKPerHour <= 0 {
		t.Errorf("EffectiveISKPerHour must be > 0 (refine path is reachable): %v", row.EffectiveISKPerHour)
	}
	if row.BestStationID == 0 {
		t.Error("BestStationID must be set (reprocess station is reachable)")
	}
	if row.RawSell != nil {
		t.Errorf("RawSell must be nil (raw path is unreachable): %+v", *row.RawSell)
	}
	if row.DeltaISKPerHour != 0 {
		t.Errorf("DeltaISKPerHour must be 0 with only one reachable path: %v", row.DeltaISKPerHour)
	}
}

// TestMiningService_OreRanking_MarketLoadsCap verifies the row reports how many full
// ore-hold loads the chosen buy order can absorb (VolumeRemain cap), and that a thin
// order yields <1 load (the fail-loud "best-price ISK/h is optimistic" signal).
func TestMiningService_OreRanking_MarketLoadsCap(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	run := func(veldVol int) *models.OreRankRow {
		market := &fakeMiningMarket{
			buyPriceByType: map[int]float64{veldsparTypeID: 15.0, tritaniumTypeID: 5.0},
			volByType:      map[int]int{veldsparTypeID: veldVol},
		}
		skills := &fakeMiningSkillsProvider{
			skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{17940: 5, 22551: 5}},
			standings: map[int64]float64{},
		}
		fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
		// Reprocess station is a citadel → refine unreachable → raw wins, so MarketLoads
		// reflects the Veldspar ore order's VolumeRemain deterministically.
		stations := &fakeStations{stations: []database.ReprocessStation{
			{StationID: 1_035_000_000_009, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
		}}
		loc := fakeMiningLocation{shipTypeID: 22544}
		svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())
		resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002, AllowLowSec: false})
		if err != nil {
			t.Fatalf("OreRanking error: %v", err)
		}
		for i := range resp.Rows {
			if resp.Rows[i].OreTypeID == veldsparTypeID {
				return &resp.Rows[i]
			}
		}
		t.Fatal("Veldspar row not found")
		return nil
	}

	// Fat order → many full loads absorbable at the best price.
	fat := run(1_000_000_000)
	if fat.Best != "raw" {
		t.Fatalf("expected raw verdict (refine unreachable), got %q", fat.Best)
	}
	if fat.MarketLoads <= 1 {
		t.Errorf("fat order: MarketLoads must be > 1, got %v", fat.MarketLoads)
	}
	// Thin order → less than one full load at the best price (fail-loud signal).
	if thin := run(5); thin.MarketLoads <= 0 || thin.MarketLoads >= 1 {
		t.Errorf("thin order: MarketLoads must be in (0,1), got %v", thin.MarketLoads)
	}
}

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// TestMiningService_OreRanking_DegradedFlags verifies the response flags + reason
// when skills/standings can't be fetched from ESI (e.g. expired token). The ranking
// is still produced, but the degradation must be surfaced (no silent fallback).
func TestMiningService_OreRanking_DegradedFlags(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	market := &fakeMiningMarket{buyPriceByType: map[int]float64{
		veldsparTypeID:  15.0,
		tritaniumTypeID: 5.0,
	}}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{
		{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05},
	}}
	loc := fakeMiningLocation{shipTypeID: 22544}

	cases := []struct {
		name           string
		skillsErr      error
		standingsErr   error
		wantSkills     bool
		wantStandings  bool
		reasonContains string
	}{
		{"both", errESI, errESI, true, true, "Mining-/Reprocessing-Skills"},
		{"skills only", errESI, nil, true, false, "Mining-/Reprocessing-Skills"},
		{"standings only", nil, errESI, false, true, "Standings"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			skills := &fakeMiningSkillsProvider{
				skills: &MiningReprocessingSkills{
					OreProcessing: map[int64]int{},
					SkillLevels:   map[int64]int{17940: 5, 22551: 5},
				},
				standings:    map[int64]float64{},
				skillsErr:    tc.skillsErr,
				standingsErr: tc.standingsErr,
			}
			svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

			resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002})
			if err != nil {
				t.Fatalf("OreRanking error: %v", err)
			}
			if resp.SkillsDegraded != tc.wantSkills {
				t.Errorf("SkillsDegraded = %v, want %v", resp.SkillsDegraded, tc.wantSkills)
			}
			if resp.StandingsDegraded != tc.wantStandings {
				t.Errorf("StandingsDegraded = %v, want %v", resp.StandingsDegraded, tc.wantStandings)
			}
			if resp.DegradedReason == "" {
				t.Fatal("DegradedReason must be set when degraded")
			}
			if !strings.Contains(resp.DegradedReason, tc.reasonContains) {
				t.Errorf("DegradedReason = %q, want substring %q", resp.DegradedReason, tc.reasonContains)
			}
			// Ranking still produced despite degradation.
			if len(resp.Rows) == 0 {
				t.Error("expected ranking rows even when degraded")
			}
		})
	}
}

// TestMiningService_OreRanking_NotDegradedByDefault guards against false positives:
// a healthy fetch must leave the flags clear and the reason empty.
func TestMiningService_OreRanking_NotDegradedByDefault(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	market := &fakeMiningMarket{buyPriceByType: map[int]float64{veldsparTypeID: 15.0, tritaniumTypeID: 5.0}}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{17940: 5, 22551: 5}},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05}}}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, fakeMiningLocation{shipTypeID: 22544}, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	if resp.SkillsDegraded || resp.StandingsDegraded || resp.DegradedReason != "" {
		t.Errorf("healthy fetch must not be degraded: skills=%v standings=%v reason=%q",
			resp.SkillsDegraded, resp.StandingsDegraded, resp.DegradedReason)
	}
}

// TestMiningService_OreRanking_FittingFetchDegraded: schlägt der Fitting-Abruf
// fehl (transienter ESI-Fehler), darf das NICHT als "kein Mining-Setup"
// erscheinen — stattdessen FittingDegraded + Reason, NoMiningSetup bleibt false.
func TestMiningService_OreRanking_FittingFetchDegraded(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	market := &fakeMiningMarket{buyPriceByType: map[int]float64{veldsparTypeID: 15.0, tritaniumTypeID: 5.0}}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{17940: 5, 22551: 5}},
		standings: map[int64]float64{},
	}
	// Fitting-Abruf scheitert → moduleIDs nil → m3h 0, aber NICHT als no-setup werten.
	fitting := &fakeMiningModules{modulesErr: errESI}
	stations := &fakeStations{stations: []database.ReprocessStation{{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05}}}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, fakeMiningLocation{shipTypeID: 22544}, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
	}
	if !resp.FittingDegraded {
		t.Error("FittingDegraded must be true when the active-ship/modules fetch fails")
	}
	if resp.NoMiningSetup {
		t.Error("NoMiningSetup must NOT be asserted on a fitting-fetch error (ESI hiccup != no mining ship)")
	}
	if !strings.Contains(resp.DegradedReason, "Schiff/Module") {
		t.Errorf("DegradedReason must mention the ship/modules degradation, got %q", resp.DegradedReason)
	}
	if len(resp.Rows) == 0 {
		t.Error("ranking rows must still be produced (per-m³ values are independent of m3h)")
	}
}
