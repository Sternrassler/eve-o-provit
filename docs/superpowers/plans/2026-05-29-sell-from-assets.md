# Sell-from-Assets Implementation Plan (Issue #107)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let a character pick an owned item and get a ranked list of where to sell it for the most net ISK (taker, instant-sell to buy orders) plus the route there, across the 5 major hubs and every station in the item's current region.

**Architecture:** New `AssetSaleService` behind two endpoints — `GET /api/v1/trading/assets` (list owned items) and `POST /api/v1/trading/assets/sell-options` (rank sell locations for one item). Heavy hub-price + route computation runs only for the selected item. Reuses `MajorHubs`, the `hubOrderFetcher` interface (`FetchMarketOrdersForType`), `SkillsServicer`, fee helpers, SDE navigation, and the `securityRisk` helper (all in package `services`).

**Tech Stack:** Go 1.24 / Fiber backend, pgx + SQLite SDE, ESI client; Next.js 16 web; Flutter web. TDD with `go test`.

**Spec:** `docs/superpowers/specs/2026-05-29-sell-from-assets-design.md`

---

## File Structure

**Backend (built here, TDD):**
- `internal/models/assets.go` — `AssetItem`, `SellOption`, `SellOptionsResponse`, `SellOptionsRequest` (reuse existing `models.SkillsApplied`).
- `internal/services/asset_sale_service.go` — `AssetSaleService` + pure helpers (`takerUnitNet`, `bestBuyInRegion`, `bestBuyByStation`, `aggregateAssets`, `minRouteSecurityStatus`).
- `internal/services/asset_sale_service_test.go` — unit tests for the pure helpers + service via fakes.
- `internal/services/asset_fetcher.go` — concrete paginated ESI assets fetcher (`ESIAssetFetcher`) + `RawAsset` type + `assetFetcher` interface.
- `internal/services/asset_fetcher_test.go` — pagination/parse test via `httptest`.
- `internal/handlers/assets.go` — `AssetsHandler` (`ListAssets`, `SellOptions`).
- `internal/handlers/assets_test.go` — handler validation tests (400/401).
- `cmd/api/container.go`, `cmd/api/main.go` — wire handler + register routes.
- `CHANGELOG.md` — `[Unreleased] ### Added` entry.

**Frontends (parallel subagents, against the locked contract):**
- Web `/sell-assets` page + asset picker + result + tests.
- Flutter `/sell-assets` screen + models + providers + tests.

---

## Task 1: Models

**Files:**
- Create: `backend/internal/models/assets.go`

- [ ] **Step 1: Create the models file**

```go
package models

// AssetItem is one aggregated owned stack: a type at a location with a summed quantity.
type AssetItem struct {
	TypeID       int    `json:"type_id"`
	Name         string `json:"name"`
	Quantity     int    `json:"quantity"`
	LocationID   int64  `json:"location_id"`
	LocationName string `json:"location_name"`
	SystemID     int    `json:"system_id"`
	RegionID     int    `json:"region_id"`
	Marketable   bool   `json:"marketable"`
}

// AssetsResponse is the GET /trading/assets payload.
type AssetsResponse struct {
	Assets []AssetItem `json:"assets"`
	Count  int         `json:"count"`
}

// SellOptionsRequest is the POST /trading/assets/sell-options body.
type SellOptionsRequest struct {
	TypeID      int   `json:"type_id"`
	LocationID  int64 `json:"location_id"`
	Quantity    int   `json:"quantity"`
	AvoidLowSec bool  `json:"avoid_low_sec"`
}

// SellOption is one ranked place to sell the item (taker / sell into a buy order).
type SellOption struct {
	Scope         string  `json:"scope"` // "hub" | "current_region"
	RegionID      int     `json:"region_id"`
	RegionName    string  `json:"region_name"`
	StationID     int64   `json:"station_id"`
	StationName   string  `json:"station_name"`
	SystemName    string  `json:"system_name"`
	BuyPrice      float64 `json:"buy_price"`
	UnitNet       float64 `json:"unit_net"`
	TotalNet      float64 `json:"total_net"`
	Jumps         int     `json:"jumps"`
	TravelTimeMin float64 `json:"travel_time_min"`
	SecurityRisk  string  `json:"security_risk"` // "safe" | "caution" | "danger"
	HasData       bool    `json:"has_data"`
}

// SellOptionsResponse is the ranked result for one selected item.
type SellOptionsResponse struct {
	TypeID         int           `json:"type_id"`
	Name           string        `json:"name"`
	Quantity       int           `json:"quantity"`
	OriginSystemID int           `json:"origin_system_id"`
	Best           *SellOption   `json:"best"`
	Options        []SellOption  `json:"options"`
	SkillsApplied  SkillsApplied `json:"skills_applied"`
}
```

- [ ] **Step 2: Build to verify it compiles**

Run: `cd backend && go build ./internal/models/`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
cd backend && git add internal/models/assets.go
git commit -m "feat(assets): models for sell-from-assets (#107)"
```

---

## Task 2: Pure pricing + ranking helpers

**Files:**
- Create: `backend/internal/services/asset_sale_service.go` (helpers only this task)
- Test: `backend/internal/services/asset_sale_service_test.go`

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/services/ -run 'TestTakerUnitNet|TestBestBuy' -v`
Expected: FAIL — `undefined: takerUnitNet` etc.

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/services/asset_sale_service.go`:

```go
package services

import (
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
)

// takerUnitNet is the per-unit proceeds when selling instantly into a buy order:
// buy price minus sales tax. No broker fee (that applies only to placed orders).
func takerUnitNet(buyPrice, salesTaxRate float64) float64 {
	return buyPrice * (1 - salesTaxRate)
}

// bestBuyInRegion returns the highest active buy-order price across all stations in a
// region, plus the station it sits at.
func bestBuyInRegion(orders []esi.ESIMarketOrder) (price float64, stationID int64, ok bool) {
	for _, o := range orders {
		if !o.IsBuyOrder || o.VolumeRemain <= 0 {
			continue
		}
		if !ok || o.Price > price {
			price, stationID, ok = o.Price, o.LocationID, true
		}
	}
	return price, stationID, ok
}

// bestBuyByStation returns the highest active buy-order price per station.
func bestBuyByStation(orders []esi.ESIMarketOrder) map[int64]float64 {
	best := map[int64]float64{}
	for _, o := range orders {
		if !o.IsBuyOrder || o.VolumeRemain <= 0 {
			continue
		}
		if cur, seen := best[o.LocationID]; !seen || o.Price > cur {
			best[o.LocationID] = o.Price
		}
	}
	return best
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/services/ -run 'TestTakerUnitNet|TestBestBuy' -v`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/services/asset_sale_service.go internal/services/asset_sale_service_test.go
git commit -m "feat(assets): taker net + best-buy helpers (#107)"
```

---

## Task 3: Asset aggregation helper

**Files:**
- Create: `backend/internal/services/asset_fetcher.go` (RawAsset type only this task)
- Modify: `backend/internal/services/asset_sale_service.go` (add `aggregateAssets`)
- Test: `backend/internal/services/asset_sale_service_test.go` (add)

- [ ] **Step 1: Write the failing test (append to asset_sale_service_test.go)**

```go
func TestAggregateAssets(t *testing.T) {
	raw := []RawAsset{
		{TypeID: 34, LocationID: 60003760, Quantity: 100},
		{TypeID: 34, LocationID: 60003760, Quantity: 50},  // same type+loc -> summed
		{TypeID: 34, LocationID: 60008494, Quantity: 10},  // different loc -> separate
		{TypeID: 35, LocationID: 60003760, Quantity: 7},
	}
	got := aggregateAssets(raw)
	// key (type,loc)
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/services/ -run TestAggregateAssets -v`
Expected: FAIL — `undefined: RawAsset` / `aggregateAssets`.

- [ ] **Step 3: Write minimal implementation**

Create `backend/internal/services/asset_fetcher.go`:

```go
package services

// RawAsset is the minimal ESI asset shape the aggregator needs.
type RawAsset struct {
	TypeID       int
	LocationID   int64
	Quantity     int
	LocationFlag string
}
```

Append to `backend/internal/services/asset_sale_service.go`:

```go
// aggStack is an aggregated (type, location) quantity, pre-enrichment.
type aggStack struct {
	typeID     int
	locationID int64
	quantity   int
}

// aggregateAssets sums quantities for identical (type, location) pairs.
func aggregateAssets(raw []RawAsset) []aggStack {
	idx := map[[2]int64]int{} // (typeID, locationID) -> index in out
	var out []aggStack
	for _, r := range raw {
		key := [2]int64{int64(r.TypeID), r.LocationID}
		if i, ok := idx[key]; ok {
			out[i].quantity += r.Quantity
			continue
		}
		idx[key] = len(out)
		out = append(out, aggStack{typeID: r.TypeID, locationID: r.LocationID, quantity: r.Quantity})
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/services/ -run TestAggregateAssets -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/services/asset_fetcher.go internal/services/asset_sale_service.go internal/services/asset_sale_service_test.go
git commit -m "feat(assets): asset aggregation by type+location (#107)"
```

---

## Task 4: Paginated ESI assets fetcher

**Files:**
- Modify: `backend/internal/services/asset_fetcher.go`
- Test: `backend/internal/services/asset_fetcher_test.go`

- [ ] **Step 1: Write the failing test**

```go
package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestESIAssetFetcher_Paginates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Pages", "2")
		switch r.URL.Query().Get("page") {
		case "", "1":
			fmt.Fprint(w, `[{"type_id":34,"location_id":60003760,"quantity":100,"location_flag":"Hangar"}]`)
		case "2":
			fmt.Fprint(w, `[{"type_id":35,"location_id":60003760,"quantity":7,"location_flag":"Hangar"}]`)
		}
	}))
	defer srv.Close()

	f := &ESIAssetFetcher{baseURL: srv.URL, client: srv.Client()}
	got, err := f.FetchCharacterAssets(context.Background(), 123, "token")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 2 || got[0].TypeID != 34 || got[0].Quantity != 100 || got[1].TypeID != 35 {
		t.Fatalf("unexpected assets: %+v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/services/ -run TestESIAssetFetcher -v`
Expected: FAIL — `undefined: ESIAssetFetcher`.

- [ ] **Step 3: Write minimal implementation (append to asset_fetcher.go)**

```go
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// assetFetcher fetches a character's full asset list (keeps AssetSaleService testable).
type assetFetcher interface {
	FetchCharacterAssets(ctx context.Context, characterID int, accessToken string) ([]RawAsset, error)
}

// ESIAssetFetcher calls /characters/{id}/assets/ with pagination (X-Pages).
type ESIAssetFetcher struct {
	baseURL string
	client  *http.Client
}

// NewESIAssetFetcher creates a fetcher against the live ESI base URL.
func NewESIAssetFetcher() *ESIAssetFetcher {
	return &ESIAssetFetcher{
		baseURL: "https://esi.evetech.net",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type esiAsset struct {
	TypeID       int    `json:"type_id"`
	LocationID   int64  `json:"location_id"`
	Quantity     int    `json:"quantity"`
	LocationFlag string `json:"location_flag"`
}

func (f *ESIAssetFetcher) FetchCharacterAssets(ctx context.Context, characterID int, accessToken string) ([]RawAsset, error) {
	base := fmt.Sprintf("%s/latest/characters/%d/assets/", f.baseURL, characterID)
	first, pages, err := f.page(ctx, base, accessToken, 1)
	if err != nil {
		return nil, err
	}
	all := first
	for p := 2; p <= pages; p++ {
		more, _, err := f.page(ctx, base, accessToken, p)
		if err != nil {
			return nil, err
		}
		all = append(all, more...)
	}
	return all, nil
}

func (f *ESIAssetFetcher) page(ctx context.Context, base, token string, page int) ([]RawAsset, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"?page="+strconv.Itoa(page), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, 0, fmt.Errorf("unauthorized")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("ESI assets status %d", resp.StatusCode)
	}
	pages := 1
	if x := resp.Header.Get("X-Pages"); x != "" {
		if v, err := strconv.Atoi(x); err == nil {
			pages = v
		}
	}
	var raw []esiAsset
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, 0, err
	}
	out := make([]RawAsset, 0, len(raw))
	for _, a := range raw {
		out = append(out, RawAsset{TypeID: a.TypeID, LocationID: a.LocationID, Quantity: a.Quantity, LocationFlag: a.LocationFlag})
	}
	return out, pages, nil
}
```

Note: keep the existing `RawAsset` struct from Task 3 at the top of the file; add these imports and code below it.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/services/ -run TestESIAssetFetcher -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/services/asset_fetcher.go internal/services/asset_fetcher_test.go
git commit -m "feat(assets): paginated ESI assets fetcher (#107)"
```

---

## Task 5: AssetSaleService (ListAssets + SellOptions)

**Files:**
- Modify: `backend/internal/services/asset_sale_service.go`
- Test: `backend/internal/services/asset_sale_service_test.go` (add service test with fakes)

- [ ] **Step 1: Write the failing test (append)**

```go
import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	applogger "github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// fakeAssetFetcher / fakeSkills / fakeHubFetcher / fakeTypeNamer for the service test.
type fakeAssetFetcher struct{ assets []RawAsset }

func (f fakeAssetFetcher) FetchCharacterAssets(_ context.Context, _ int, _ string) ([]RawAsset, error) {
	return f.assets, nil
}

type fakeSkills2 struct{}

func (fakeSkills2) GetCharacterSkills(_ context.Context, _ int, _ string) (*TradingSkills, error) {
	return &TradingSkills{Accounting: 0}, nil // sales tax = 8% (Accounting 0)
}

// newTestSDE builds an in-memory SDE with the columns the service queries.
func newAssetTestSDE(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE mapSolarSystems (_key INTEGER PRIMARY KEY, regionID INTEGER, securityStatus REAL, name TEXT);
		CREATE TABLE staStations (stationID INTEGER PRIMARY KEY, solarSystemID INTEGER, regionID INTEGER, stationName TEXT);
		INSERT INTO mapSolarSystems VALUES (30000142, 10000002, 0.9, 'Jita');
		INSERT INTO staStations VALUES (60003760, 30000142, 10000002, 'Jita IV - Moon 4 - CNAP');
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}
```

NOTE for the implementer: the exact SDE schema (`mapSolarSystems._key`, `staStations`) must match what `SDERepository.GetSystemIDForLocation` / `GetStationName` / `GetSystemName` / `GetRegionIDForSystem` query. Before writing this test, open `internal/database/sde.go` lines 134-230 and copy the real column/table names into the `CREATE TABLE` statements. Then assert:

```go
func TestAssetSaleService_ListAssets_Aggregates(t *testing.T) {
	sde := newAssetTestSDE(t)
	defer sde.Close()
	repo := database.NewSDERepository(sde)
	svc := NewAssetSaleService(
		fakeSkills2{},
		fakeHubFetcher{},          // reuse the multi_hub fakeMarket pattern; returns no orders here
		fakeTypeNamer{name: "Tritanium", marketable: true},
		fakeAssetFetcher{assets: []RawAsset{
			{TypeID: 34, LocationID: 60003760, Quantity: 100, LocationFlag: "Hangar"},
			{TypeID: 34, LocationID: 60003760, Quantity: 50, LocationFlag: "Hangar"},
		}},
		repo, sde, applogger.New(),
	)
	res, err := svc.ListAssets(context.Background(), 1, "tok")
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 1 || res.Assets[0].Quantity != 150 || !res.Assets[0].Marketable {
		t.Fatalf("unexpected: %+v", res)
	}
}
```

Define `fakeHubFetcher` and `fakeTypeNamer` in the test file:

```go
type fakeHubFetcher struct{ ordersByRegion map[int][]esi.ESIMarketOrder }

func (f fakeHubFetcher) FetchMarketOrdersForType(_ context.Context, regionID, _ int) ([]esi.ESIMarketOrder, error) {
	return f.ordersByRegion[regionID], nil
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/services/ -run TestAssetSaleService -v`
Expected: FAIL — `undefined: NewAssetSaleService`.

- [ ] **Step 3: Write the implementation (append to asset_sale_service.go)**

```go
import (
	"context"
	"database/sql"
	"sort"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// AssetSaleService lists owned items and ranks taker sell locations (Issue #107).
type AssetSaleService struct {
	skills  SkillsServicer
	market  hubOrderFetcher
	types   typeNamer
	assets  assetFetcher
	sdeRepo *database.SDERepository
	sdeDB   *sql.DB
	logger  *logger.Logger
}

// NewAssetSaleService creates a new service.
func NewAssetSaleService(
	skills SkillsServicer,
	market hubOrderFetcher,
	types typeNamer,
	assets assetFetcher,
	sdeRepo *database.SDERepository,
	sdeDB *sql.DB,
	logger *logger.Logger,
) *AssetSaleService {
	return &AssetSaleService{skills: skills, market: market, types: types, assets: assets, sdeRepo: sdeRepo, sdeDB: sdeDB, logger: logger}
}

// ListAssets returns the character's owned items aggregated by (type, location).
func (s *AssetSaleService) ListAssets(ctx context.Context, characterID int, accessToken string) (*models.AssetsResponse, error) {
	raw, err := s.assets.FetchCharacterAssets(ctx, characterID, accessToken)
	if err != nil {
		return nil, err
	}
	stacks := aggregateAssets(raw)
	items := make([]models.AssetItem, 0, len(stacks))
	for _, st := range stacks {
		item := models.AssetItem{TypeID: st.typeID, Quantity: st.quantity, LocationID: st.locationID}
		if info, err := s.types.GetTypeInfo(ctx, st.typeID); err == nil && info != nil {
			item.Name = info.Name
			item.Marketable = info.MarketGroup != nil
		}
		if sysID, err := s.sdeRepo.GetSystemIDForLocation(ctx, st.locationID); err == nil {
			item.SystemID = int(sysID)
			if rid, err := s.sdeRepo.GetRegionIDForSystem(ctx, sysID); err == nil {
				item.RegionID = rid
			}
		}
		if name, err := s.sdeRepo.GetStationName(ctx, st.locationID); err == nil {
			item.LocationName = name
		}
		items = append(items, item)
	}
	return &models.AssetsResponse{Assets: items, Count: len(items)}, nil
}

// SellOptions ranks taker sell locations for one owned item.
func (s *AssetSaleService) SellOptions(ctx context.Context, req *models.SellOptionsRequest, characterID int, accessToken string) (*models.SellOptionsResponse, error) {
	skills, serr := s.skills.GetCharacterSkills(ctx, characterID, accessToken)
	if serr != nil || skills == nil {
		skills = &TradingSkills{}
	}
	salesTaxRate := SalesTaxRate(skills.Accounting)

	name := ""
	if info, err := s.types.GetTypeInfo(ctx, req.TypeID); err == nil && info != nil {
		name = info.Name
	}

	originSys, err := s.sdeRepo.GetSystemIDForLocation(ctx, req.LocationID)
	resp := &models.SellOptionsResponse{
		TypeID: req.TypeID, Name: name, Quantity: req.Quantity,
		SkillsApplied: models.SkillsApplied{
			Applied: serr == nil, Accounting: skills.Accounting, BrokerRelations: skills.BrokerRelations,
			SalesTaxRate: salesTaxRate,
			BrokerFeeRate: BrokerFeeRate(skills.BrokerRelations, skills.AdvancedBrokerRelations, skills.FactionStanding, skills.CorpStanding),
		},
	}
	if err != nil {
		return resp, nil // origin unresolvable -> empty options
	}
	resp.OriginSystemID = int(originSys)
	currentRegion, _ := s.sdeRepo.GetRegionIDForSystem(ctx, originSys)

	var options []models.SellOption

	// 5 major hubs: region-best buy order, route to hub system.
	for _, hub := range MajorHubs {
		orders, err := s.market.FetchMarketOrdersForType(ctx, hub.RegionID, req.TypeID)
		if err != nil {
			s.logger.Warn("sell-options: hub fetch failed", "error", err, "region", hub.RegionID)
			options = append(options, models.SellOption{Scope: "hub", RegionID: hub.RegionID, RegionName: hub.RegionName, SystemName: hub.Name, HasData: false})
			continue
		}
		price, stationID, ok := bestBuyInRegion(orders)
		opt := s.buildOption(ctx, "hub", hub.RegionID, hub.RegionName, stationID, int64(hub.SystemID), originSys, price, ok, req, salesTaxRate)
		if opt.SystemName == "" {
			opt.SystemName = hub.Name
		}
		options = append(options, opt)
	}

	// Current region: per-station best buy, route to each station's system.
	if currentRegion != 0 {
		orders, err := s.market.FetchMarketOrdersForType(ctx, currentRegion, req.TypeID)
		if err == nil {
			for stationID, price := range bestBuyByStation(orders) {
				sysID, err := s.sdeRepo.GetSystemIDForLocation(ctx, stationID)
				if err != nil {
					continue // unresolvable (player structure) -> skip
				}
				opt := s.buildOption(ctx, "current_region", currentRegion, "", stationID, sysID, originSys, price, true, req, salesTaxRate)
				options = append(options, opt)
			}
		}
	}

	sort.SliceStable(options, func(i, j int) bool { return options[i].TotalNet > options[j].TotalNet })
	resp.Options = options
	for i := range options {
		if options[i].HasData && options[i].TotalNet > 0 {
			best := options[i]
			resp.Best = &best
			break
		}
	}
	return resp, nil
}

// buildOption computes net + route for one sell location. destSys is the system to route
// to; stationID is the buy-order station (for the waypoint + name).
func (s *AssetSaleService) buildOption(ctx context.Context, scope string, regionID int, regionName string, stationID, destSys, originSys int64, buyPrice float64, hasData bool, req *models.SellOptionsRequest, salesTaxRate float64) models.SellOption {
	opt := models.SellOption{Scope: scope, RegionID: regionID, RegionName: regionName, StationID: stationID, HasData: hasData}
	if name, err := s.sdeRepo.GetStationName(ctx, stationID); err == nil {
		opt.StationName = name
	}
	if name, err := s.sdeRepo.GetSystemName(ctx, destSys); err == nil {
		opt.SystemName = name
	}
	if !hasData {
		return opt
	}
	opt.BuyPrice = buyPrice
	opt.UnitNet = takerUnitNet(buyPrice, salesTaxRate)
	opt.TotalNet = opt.UnitNet * float64(req.Quantity)

	if destSys == originSys {
		opt.Jumps = 0
		opt.TravelTimeMin = 0
		opt.SecurityRisk = securityRisk(minRouteSecurityStatus(s.sdeDB, []int64{originSys}))
		return opt
	}
	travel, err := navigation.CalculateTravelTime(s.sdeDB, originSys, destSys, &navigation.NavigationParams{AvoidLowSec: req.AvoidLowSec}, false)
	if err != nil {
		opt.HasData = false // no route (e.g. high-sec-only filter) -> not actionable
		return opt
	}
	opt.Jumps = travel.Jumps
	opt.TravelTimeMin = travel.TotalMinutes
	opt.SecurityRisk = securityRisk(minRouteSecurityStatus(s.sdeDB, travel.Route))
	return opt
}

// minRouteSecurityStatus returns the lowest securityStatus along a route (1.0 if empty/unknown).
func minRouteSecurityStatus(db *sql.DB, route []int64) float64 {
	if len(route) == 0 {
		return 1.0
	}
	minSec := 1.0
	for _, sysID := range route {
		var sec float64
		if err := db.QueryRowContext(context.Background(), "SELECT securityStatus FROM mapSolarSystems WHERE _key = ?", sysID).Scan(&sec); err != nil {
			continue
		}
		if sec < minSec {
			minSec = sec
		}
	}
	return minSec
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd backend && go test ./internal/services/ -run 'TestAssetSaleService|TestTakerUnitNet|TestBestBuy|TestAggregateAssets' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd backend && git add internal/services/asset_sale_service.go internal/services/asset_sale_service_test.go
git commit -m "feat(assets): AssetSaleService list + taker sell-options ranking (#107)"
```

---

## Task 6: Handler + routes + wiring

**Files:**
- Create: `backend/internal/handlers/assets.go`
- Test: `backend/internal/handlers/assets_test.go`
- Modify: `backend/cmd/api/container.go`, `backend/cmd/api/main.go`

- [ ] **Step 1: Write the failing handler validation test**

```go
package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAssetsHandler_SellOptions_BadRequest(t *testing.T) {
	app := fiber.New()
	h := &AssetsHandler{} // service nil: validation happens before service call
	app.Post("/sell", func(c *fiber.Ctx) error {
		c.Locals("character_id", 1)
		c.Locals("access_token", "tok")
		return h.SellOptions(c)
	})
	req := httptest.NewRequest("POST", "/sell", strings.NewReader(`{"type_id":0,"location_id":1,"quantity":0}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestAssetsHandler_SellOptions_Unauthorized(t *testing.T) {
	app := fiber.New()
	h := &AssetsHandler{}
	app.Post("/sell", h.SellOptions)
	req := httptest.NewRequest("POST", "/sell", strings.NewReader(`{"type_id":34,"location_id":60003760,"quantity":10}`))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != 401 {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/handlers/ -run TestAssetsHandler -v`
Expected: FAIL — `undefined: AssetsHandler`.

- [ ] **Step 3: Write the handler (create assets.go)**

```go
package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
)

// AssetsHandler serves the sell-from-assets endpoints (Issue #107).
type AssetsHandler struct {
	service *services.AssetSaleService
}

// NewAssetsHandler creates a new assets handler.
func NewAssetsHandler(service *services.AssetSaleService) *AssetsHandler {
	return &AssetsHandler{service: service}
}

func authCtx(c *fiber.Ctx) (int, string, bool) {
	cid, ok1 := c.Locals("character_id").(int)
	tok, ok2 := c.Locals("access_token").(string)
	return cid, tok, ok1 && ok2
}

// ListAssets handles GET /api/v1/trading/assets
//
// @Summary List owned items aggregated by type and location
// @Tags Trading
// @Produce json
// @Success 200 {object} models.AssetsResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/assets [get]
func (h *AssetsHandler) ListAssets(c *fiber.Ctx) error {
	cid, tok, ok := authCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}
	res, err := h.service.ListAssets(c.UserContext(), cid, tok)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list assets"})
	}
	return c.JSON(res)
}

// SellOptions handles POST /api/v1/trading/assets/sell-options
//
// @Summary Rank taker sell locations for one owned item
// @Tags Trading
// @Accept json
// @Produce json
// @Param request body models.SellOptionsRequest true "Selected item"
// @Success 200 {object} models.SellOptionsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/assets/sell-options [post]
func (h *AssetsHandler) SellOptions(c *fiber.Ctx) error {
	var req models.SellOptionsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.TypeID <= 0 || req.Quantity <= 0 || req.LocationID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type_id, location_id and quantity are required"})
	}
	cid, tok, ok := authCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}
	res, err := h.service.SellOptions(c.UserContext(), &req, cid, tok)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to compute sell options"})
	}
	return c.JSON(res)
}
```

Note: the `TestAssetsHandler_SellOptions_BadRequest` test validates *before* auth (validation runs first in `SellOptions`), so the `400` path is reached even though `character_id` is set. The `Unauthorized` test omits Locals so `authCtx` returns false after validation passes — both match the handler order above (validation → auth).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/handlers/ -run TestAssetsHandler -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Wire into the container**

In `backend/cmd/api/container.go`, add field to `AppContainer`:
```go
	AssetsHandler      *handlers.AssetsHandler
```
And after the hauling wiring block (around line 164), add:
```go
	// Sell-from-assets (#107) — reuses hub price fetch + skills + navigation.
	assetSaleService := services.NewAssetSaleService(skillsService, c.ESIClient, c.SDERepo, services.NewESIAssetFetcher(), c.SDERepo, c.DB.SDE, c.AppLogger)
	c.AssetsHandler = handlers.NewAssetsHandler(assetSaleService)
```
NOTE: confirm `c.ESIClient` satisfies `hubOrderFetcher` (it has `FetchMarketOrdersForType`; `competition_collector.go:67` calls it on the ESI client). If the method lives on a sub-client, mirror exactly how `multiHubService` is constructed at container.go:154 (it passes the same fetcher the multi-hub service uses). Use that identical argument.

- [ ] **Step 6: Register routes in main.go**

In `backend/cmd/api/main.go`, after the hauling route (line 157), add:
```go
	api.Get("/trading/assets", evesso.NewAuthMiddleware(c.TokenValidator), c.AssetsHandler.ListAssets)
	api.Post("/trading/assets/sell-options", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.AssetsHandler.SellOptions)
```

- [ ] **Step 7: Build + vet + full short test**

Run: `cd backend && gofmt -l . && go build ./... && go vet ./... && go test -short ./internal/... ./cmd/...`
Expected: gofmt prints nothing; build/vet clean; tests PASS.

- [ ] **Step 8: Commit**

```bash
cd backend && git add internal/handlers/assets.go internal/handlers/assets_test.go cmd/api/container.go cmd/api/main.go
git commit -m "feat(assets): handler + routes + wiring (#107)"
```

---

## Task 7: CHANGELOG + backend gate

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Add an Unreleased entry**

Under `## [Unreleased]` add:
```markdown
### Added

- **Sell-from-Assets (#107):** `GET /api/v1/trading/assets` (owned items aggregated by type + location) and `POST /api/v1/trading/assets/sell-options` — for a selected item, ranks taker sell locations (instant sell into a buy order, net of sales tax) across the 5 major hubs + every station in the item's current region, each with route (jumps, travel time, security risk). Ranked by total net proceeds.
```

- [ ] **Step 2: Run the release-check + full backend gate**

Run: `cd backend && make test` (or from repo root `make pr-check` if Docker is available)
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add CHANGELOG.md && git commit -m "docs(changelog): sell-from-assets endpoints (#107)"
```

---

## Task 8: Web frontend (subagent)

Dispatch a subagent with the locked contract (spec §"API Contract"). Brief:

- **Page** `frontend/src/app/sell-assets/page.tsx` (NEW route, auth-gated, ErrorBoundary) — follow `src/app/roi-calculator/page.tsx` / `multi-hub/page.tsx`.
- **API** `src/lib/api-client.ts`: add `listAssets()` (GET) and `findSellOptions(req)` (POST), both authed `credentials:"include"`. Reuse existing `setWaypoint`.
- **Types** in `src/types/trading.ts`: `AssetItem`, `AssetsResponse`, `SellOptionsRequest`, `SellOption`, `SellOptionsResponse` (reuse `SkillsApplied`).
- **Components:**
  - `AssetPicker.tsx` — searchable/filterable list (by name / quantity / location); non-marketable rows visually disabled; selecting one + entering quantity (default = stack quantity) + avoid-low-sec checkbox fires `findSellOptions`.
  - `SellOptionsResult.tsx` — `best` highlighted; option list (system, station, scope badge hub/region, buy price, unit net, **total net**, jumps, travel min, security badge safe/caution/danger); "Route an EVE übertragen" button → `setWaypoint(station_id, {clearOtherWaypoints:true})`. Empty/`has_data:false` rows shown muted ("kein Marktpreis").
- **Nav:** add `/sell-assets` entry in `src/components/navigation.tsx` (desktop + mobile), consistent with the others.
- **Tests:** Vitest `tests/components/SellOptionsResult.test.tsx` (option fields, security badge color, best highlight, waypoint call); authed Playwright `tests/e2e/auth/sell-assets.spec.ts` mirroring `roi-calculator.spec.ts`.
- **Verify:** `npm run test` + `npm run lint` + `tsc --noEmit` clean for touched files.

After it returns: verify (`npx vitest run tests/components/SellOptionsResult.test.tsx`, eslint touched files), then commit `feat(web): sell-from-assets page + asset picker + sell options (#107)`.

---

## Task 9: Flutter frontend (subagent)

Dispatch a subagent with the same locked contract. Brief:

- **Models** `lib/api/asset_models.dart`: `AssetItem`, `AssetsResponse`, `SellOptionsRequest`, `SellOption`, `SellOptionsResponse` (reuse `SkillsApplied`), null/float-robust `fromJson`.
- **Service** `lib/api/trading_api.dart`: add `listAssets()` and `findSellOptions(req)` (token via existing dio interceptor).
- **Providers** `lib/features/trading/sell_assets_providers.dart`: assets `FutureProvider`, sell-options `AsyncNotifier` (match ROI/Hub shape).
- **Screen** `lib/features/trading/sell_assets_screen.dart`: adaptive `isTwoPane(840)`; left = asset picker (search field + list, non-marketable disabled), tapping one opens quantity + avoid-low-sec + "Verkaufsorte suchen"; right/result = best highlighted + option list with security chip (safe=green/caution=amber/danger=red), scope badge, total net, jumps, travel; tap option → detail with "Route an EVE übertragen" (waypoint to `station_id`). Empty state "Keine Verkaufsorte gefunden".
- **Router** `lib/core/router.dart`: register `/sell-assets` + nav destination (e.g. sell/tag icon), consistent indexing.
- **Tests:** widget test (option list, security chip colors, tap→detail, waypoint) + `fromJson` model test.
- **Verify:** `flutter analyze` clean; `flutter test` passes. Do NOT build/run on device.

After it returns: verify (`flutter analyze` touched files, `flutter test` new tests), then commit `feat(app): sell-from-assets screen + asset picker + sell options (#107)`.

---

## Task 10: Ship

- [ ] PR `feat/sell-from-assets` → CI (lint/test/vuln) green → squash-merge (`Closes #107`).
- [ ] `make release VERSION=0.12.0` → commit CHANGELOG transform → tag `v0.12.0` → push → watch deploy.
- [ ] Prod verify: `GET /sell-assets` → 200; `POST /api/v1/trading/assets/sell-options` without auth → 401; `GET /api/v1/version` → `v0.12.0`.
- [ ] Rebuild APK with `--dart-define=API_BASE_URL=https://eveonline.sternrassler.de --dart-define=EVE_CLIENT_ID=<mobile client id from deployments/.env>`, install on device, on-device smoke (login → Sell-Assets → pick item → options).
- [ ] Comment + close #107.

---

## Self-Review Notes

- **Spec coverage:** assets endpoint (Task 1/4/5/6) ✓; per-item taker hub ranking (Task 2/5) ✓; current-region per-station (Task 5) ✓; route + security (Task 5 buildOption) ✓; UI picker + result + waypoint (Task 8/9) ✓; structures/low-null graceful (Task 5: unresolvable skip, no-route → has_data:false; origin unresolvable → empty options) ✓; "everything" assets + marketable flag (Task 5 ListAssets) ✓.
- **Type consistency:** `RawAsset` (Task 3) reused in Task 4/5; `bestBuyInRegion`/`bestBuyByStation`/`takerUnitNet` (Task 2) used in Task 5; `hubOrderFetcher`/`typeNamer`/`SkillsServicer` reused from `multi_hub_service.go`; `securityRisk` reused from `hauling_service.go`; `minRouteSecurityStatus` defined in Task 5.
- **Known implementer check-points (flagged inline, not placeholders):** (a) Task 5 test SDE schema must mirror the real `sde.go` queries — copy column names; (b) Task 6 container wiring must pass the *same* `FetchMarketOrdersForType` provider the multi-hub service uses (confirm at container.go:154).
