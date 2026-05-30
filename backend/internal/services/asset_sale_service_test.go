package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	applogger "github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// --- fakes for the service test ---

type fakeAssetFetcher struct{ assets []RawAsset }

func (f fakeAssetFetcher) FetchCharacterAssets(_ context.Context, _ int, _ string) ([]RawAsset, error) {
	return f.assets, nil
}

type fakeAssetSkills struct{}

func (fakeAssetSkills) GetCharacterSkills(_ context.Context, _ int, _ string) (*TradingSkills, error) {
	return &TradingSkills{Accounting: 0}, nil
}

type fakeHubFetcher struct{ ordersByRegion map[int][]esi.ESIMarketOrder }

func (f fakeHubFetcher) FetchMarketOrdersForType(_ context.Context, regionID, _ int) ([]esi.ESIMarketOrder, error) {
	return f.ordersByRegion[regionID], nil
}

type fakeFitting struct{}

func (fakeFitting) GetShipFitting(_ context.Context, _, _ int, _ string) (*FittingData, error) {
	return nil, nil
}
func (fakeFitting) InvalidateFittingCache(_ context.Context, _, _ int) {}

type fakeActiveShip struct{ typeID int }

func (f fakeActiveShip) GetActiveShipTypeID(_ context.Context, _ int, _ string) (int, error) {
	return f.typeID, nil
}

type fakeTypeNamer struct {
	name       string
	marketable bool
}

func (f fakeTypeNamer) GetTypeInfo(_ context.Context, _ int) (*database.TypeInfo, error) {
	ti := &database.TypeInfo{Name: f.name}
	if f.marketable {
		mg := 1234
		ti.MarketGroup = &mg
	}
	return ti, nil
}

// newAssetTestSDE builds an in-memory SDE matching the columns SDERepository queries.
func newAssetTestSDE(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE mapSolarSystems (_key INTEGER PRIMARY KEY, constellationID INTEGER, regionID INTEGER, securityStatus REAL, name TEXT);
		CREATE TABLE mapConstellations (_key INTEGER PRIMARY KEY, regionID INTEGER);
		CREATE TABLE npcStations (_key INTEGER PRIMARY KEY, solarSystemID INTEGER, typeID INTEGER);
		CREATE TABLE types (_key INTEGER PRIMARY KEY, name TEXT);
		INSERT INTO mapConstellations VALUES (20000020, 10000002);
		INSERT INTO mapSolarSystems VALUES (30000142, 20000020, 10000002, 0.9, '{"en":"Jita"}');
		INSERT INTO types VALUES (54, '{"en":"Jita IV - Moon 4 - CNAP"}');
		INSERT INTO npcStations VALUES (60003760, 30000142, 54);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestTakerUnitNet(t *testing.T) {
	// 100 ISK buy order, 2.5% sales tax -> 97.5 net, no broker fee.
	got := takerUnitNet(100.0, 0.025)
	if got != 97.5 {
		t.Fatalf("takerUnitNet = %v, want 97.5", got)
	}
}

func TestBestBuyInRegion(t *testing.T) {
	orders := []esi.ESIMarketOrder{
		{IsBuyOrder: true, Price: 90, VolumeRemain: 10, LocationID: 1},
		{IsBuyOrder: true, Price: 110, VolumeRemain: 5, LocationID: 2},
		{IsBuyOrder: false, Price: 200, VolumeRemain: 5, LocationID: 2}, // sell order ignored
		{IsBuyOrder: true, Price: 105, VolumeRemain: 0, LocationID: 3},  // empty ignored
	}
	price, stn, ok := bestBuyInRegion(orders)
	if !ok || price != 110 || stn != 2 {
		t.Fatalf("bestBuyInRegion = (%v,%v,%v), want (110,2,true)", price, stn, ok)
	}
}

func TestBestBuyInRegion_NoBuyOrders(t *testing.T) {
	orders := []esi.ESIMarketOrder{{IsBuyOrder: false, Price: 200, VolumeRemain: 5, LocationID: 1}}
	if _, _, ok := bestBuyInRegion(orders); ok {
		t.Fatalf("expected ok=false when no buy orders")
	}
}

func TestBestBuyByStation(t *testing.T) {
	orders := []esi.ESIMarketOrder{
		{IsBuyOrder: true, Price: 90, VolumeRemain: 10, LocationID: 1},
		{IsBuyOrder: true, Price: 95, VolumeRemain: 10, LocationID: 1}, // higher at stn 1
		{IsBuyOrder: true, Price: 80, VolumeRemain: 10, LocationID: 2},
		{IsBuyOrder: false, Price: 200, VolumeRemain: 5, LocationID: 1}, // sell ignored
	}
	got := bestBuyByStation(orders)
	if got[1] != 95 || got[2] != 80 || len(got) != 2 {
		t.Fatalf("bestBuyByStation = %v, want {1:95, 2:80}", got)
	}
}

func TestAggregateAssets(t *testing.T) {
	raw := []RawAsset{
		{TypeID: 34, LocationID: 60003760, Quantity: 100},
		{TypeID: 34, LocationID: 60003760, Quantity: 50}, // same type+loc -> summed
		{TypeID: 34, LocationID: 60008494, Quantity: 10}, // different loc -> separate
		{TypeID: 35, LocationID: 60003760, Quantity: 7},
	}
	got := aggregateAssets(raw)
	find := func(typeID int, loc int64) int {
		for _, a := range got {
			if a.typeID == typeID && a.locationID == loc {
				return a.quantity
			}
		}
		return -1
	}
	if find(34, 60003760) != 150 {
		t.Fatalf("type 34 @ 60003760 = %d, want 150", find(34, 60003760))
	}
	if find(34, 60008494) != 10 || find(35, 60003760) != 7 {
		t.Fatalf("unexpected aggregation: %+v", got)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 aggregated stacks, got %d", len(got))
	}
}

func TestAssetSaleService_ListAssets_Aggregates(t *testing.T) {
	sde := newAssetTestSDE(t)
	defer sde.Close()
	repo := database.NewSDERepository(sde)
	svc := NewAssetSaleService(
		fakeAssetSkills{},
		fakeHubFetcher{},
		fakeTypeNamer{name: "Tritanium", marketable: true},
		fakeAssetFetcher{assets: []RawAsset{
			{TypeID: 34, LocationID: 60003760, Quantity: 100, LocationFlag: "Hangar"},
			{TypeID: 34, LocationID: 60003760, Quantity: 50, LocationFlag: "Hangar"},
		}},
		fakeFitting{}, fakeActiveShip{}, repo, sde, applogger.New(),
	)
	res, err := svc.ListAssets(context.Background(), 1, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 || res.Assets[0].Quantity != 150 || !res.Assets[0].Marketable {
		t.Fatalf("unexpected: %+v", res)
	}
	if res.Assets[0].SystemID != 30000142 || res.Assets[0].RegionID != 10000002 {
		t.Fatalf("location not resolved: %+v", res.Assets[0])
	}
}

// TestAssetSaleService_SellOptions_UnresolvableOrigin_ReturnsEmptySlice
// covers the player-structure / citadel case: GetSystemIDForLocation fails →
// we return early. The Options field MUST be a non-nil slice (JSON []), never
// nil (would marshal as JSON null and crash the web client on options.length).
// Reported: 296× Shield Power Relay I in a player structure crashed sell-assets.
func TestAssetSaleService_SellOptions_UnresolvableOrigin_ReturnsEmptySlice(t *testing.T) {
	sde := newAssetTestSDE(t)
	defer sde.Close()
	repo := database.NewSDERepository(sde)
	svc := NewAssetSaleService(
		fakeAssetSkills{},
		fakeHubFetcher{ordersByRegion: map[int][]esi.ESIMarketOrder{}},
		fakeTypeNamer{name: "Shield Power Relay I", marketable: true},
		fakeAssetFetcher{},
		fakeFitting{}, fakeActiveShip{}, repo, sde, applogger.New(),
	)
	// LocationID that isn't in the SDE (mimics a player structure / citadel).
	req := &models.SellOptionsRequest{TypeID: 1183, LocationID: 1_040_000_000_000, Quantity: 296}
	res, err := svc.SellOptions(context.Background(), req, 1, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if res.Options == nil {
		t.Fatal("Options must be an empty slice, not nil (JSON null breaks the web client)")
	}
	if len(res.Options) != 0 {
		t.Fatalf("Options should be empty for unresolvable origin, got %d entries", len(res.Options))
	}
	// And the JSON encoding must contain `"options":[]`, never `"options":null`.
	blob, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(blob, []byte(`"options":[]`)) {
		t.Fatalf("expected JSON to contain `\"options\":[]`, got: %s", string(blob))
	}
	// And we surface WHY there are no options so the UI can show an actionable
	// message instead of the generic "no buyers found" empty state.
	if res.NotRoutableReason != "origin_in_player_structure" {
		t.Errorf("not_routable_reason = %q, want %q", res.NotRoutableReason, "origin_in_player_structure")
	}
}

func TestAssetSaleService_SellOptions_RanksTakerNet(t *testing.T) {
	sde := newAssetTestSDE(t)
	defer sde.Close()
	repo := database.NewSDERepository(sde)
	// Origin = Jita (region 10000002). The Forge hub has a buy order at 5.0 ISK.
	svc := NewAssetSaleService(
		fakeAssetSkills{}, // Accounting 0 -> sales tax 5%
		fakeHubFetcher{ordersByRegion: map[int][]esi.ESIMarketOrder{
			10000002: {{IsBuyOrder: true, Price: 5.0, VolumeRemain: 1_000_000, LocationID: 60003760}},
		}},
		fakeTypeNamer{name: "Tritanium", marketable: true},
		fakeAssetFetcher{},
		fakeFitting{}, fakeActiveShip{}, repo, sde, applogger.New(),
	)
	req := &models.SellOptionsRequest{TypeID: 34, LocationID: 60003760, Quantity: 1000, AvoidLowSec: true}
	res, err := svc.SellOptions(context.Background(), req, 1, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if res.OriginSystemID != 30000142 {
		t.Fatalf("origin system = %d, want 30000142", res.OriginSystemID)
	}
	if res.Best == nil {
		t.Fatalf("expected a best option")
	}
	// taker net = 5.0 * (1 - 0.05) = 4.75 per unit; total = 4750.
	if res.Best.UnitNet < 4.74 || res.Best.UnitNet > 4.76 {
		t.Fatalf("unit_net = %v, want ~4.75", res.Best.UnitNet)
	}
	if res.Best.TotalNet < 4749 || res.Best.TotalNet > 4751 {
		t.Fatalf("total_net = %v, want ~4750", res.Best.TotalNet)
	}
	// Jita hub == origin system -> 0 jumps, safe.
	if res.Best.Jumps != 0 || res.Best.SecurityRisk != "safe" {
		t.Fatalf("expected 0 jumps / safe, got %d / %s", res.Best.Jumps, res.Best.SecurityRisk)
	}
}
