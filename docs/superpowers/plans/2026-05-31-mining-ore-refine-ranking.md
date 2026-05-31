# Mining ore sell-vs-refine ranking — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A "Mining" view that ranks ores by ISK/hour and, per ore, says whether to sell the raw ore or reprocess + sell the materials — using the current ship's mining rate, reprocessing/trade skills, region buy-order prices, and the best NPC reprocessing station by standing.

**Architecture:** New backend deterministic calcs (reprocessing yield, mining yield) built on the existing `pkg/evedb` + `pkg/evedb/dogma` patterns, a new `MiningService` + `POST /api/v1/mining/ore-ranking` endpoint reusing the market repo + skills service, plus a new ESI `/standings` fetch. New web + Flutter "Mining" views mirroring the existing trading views.

**Tech Stack:** Go 1.24 / Fiber, SQLite SDE (`pkg/evedb`), Next.js 16 / React / TS (web), Flutter/Dart (`app/`).

Spec: `docs/superpowers/specs/2026-05-31-mining-ore-refine-ranking-design.md`.

**Verification env:** backend SDE tests use `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db` + `GOWORK=off`. Commits: prefix with `PATH="$HOME/.local/bin:$PATH"` (gitleaks). golangci-lint must be 0 issues. Branch: `feat/mining-ore-ranking` (already created).

**Confirmed SDE facts (use directly):**
- Ores = `types` in `groups` whose `categories.name.en = 'Asteroid'`, `published=1`. Each has `portionSize` (reprocess batch, e.g. Veldspar 100) and `volume` m³/unit (Veldspar 0.1).
- Reprocess output: `typeMaterials.materials` is JSON `[{"materialTypeID":34,"quantity":400}]` keyed by ore `_key` (Veldspar 1230 → 400 Tritanium per 100 units).
- NPC stations: `npcStations(_key, ownerID, reprocessingEfficiency=0.5, reprocessingStationsTake=0.05, solarSystemID, typeID)`. Region via `mapSolarSystems`→`regionID` (see `SDERepository.GetRegionIDForSystem`).
- Mining modules: dogma attr **77** = `miningAmount` (m³/cycle, Strip Miner I = 150), attr **73** = `duration` ms (Strip Miner I = 45000).
- Skills (confirm ids via SDE in-task): Reprocessing **3385**, Reprocessing Efficiency **3389**, ore-specific "<Ore> Processing" (e.g. Veldspar Processing **12195**), Mining **3386**, Astrogeology **3410**.

---

## PHASE 1 — Backend core: reprocessing yield + market taker + comparison + ranking + best station

### Task 1: Ore catalog + reprocessing output (SDE)

**Files:** Create `backend/pkg/evedb/reprocessing/reprocessing.go` + `..._test.go`.

- [ ] **Step 1 — failing test** (`reprocessing_test.go`), against the local SDE:
```go
package reprocessing

import (
	"testing"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

func TestOreOutput_Veldspar(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ore, err := GetOre(db, 1230) // Veldspar
	if err != nil { t.Fatal(err) }
	if ore.PortionSize != 100 || ore.VolumeM3 != 0.1 {
		t.Fatalf("portion/volume: %d / %v", ore.PortionSize, ore.VolumeM3)
	}
	// Veldspar reprocesses to 400 Tritanium (type 34) per 100-unit portion.
	if len(ore.Materials) != 1 || ore.Materials[0].MaterialTypeID != 34 || ore.Materials[0].Quantity != 400 {
		t.Fatalf("materials: %+v", ore.Materials)
	}
}

func TestListOres_IncludesVeldsparAndExcludesNonOre(t *testing.T) {
	db := testutil.OpenTestDB(t)
	ores, err := ListOres(db)
	if err != nil { t.Fatal(err) }
	var found bool
	for _, o := range ores { if o.TypeID == 1230 { found = true } }
	if !found { t.Fatal("Veldspar (1230) missing from ore list") }
	if len(ores) < 10 { t.Fatalf("expected many ores, got %d", len(ores)) }
}
```

- [ ] **Step 2 — run, expect compile failure.** `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/reprocessing/ -v`

- [ ] **Step 3 — implement** `reprocessing.go`:
```go
package reprocessing

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type Material struct {
	MaterialTypeID int64 `json:"materialTypeID"`
	Quantity       int64 `json:"quantity"`
}

type Ore struct {
	TypeID      int64
	Name        string
	GroupID     int64
	PortionSize int64
	VolumeM3    float64
	Materials   []Material
}

// GetOre loads one ore's reprocessing portion + output materials.
func GetOre(db *sql.DB, oreTypeID int64) (*Ore, error) {
	var o Ore
	var matsJSON sql.NullString
	err := db.QueryRow(`
		SELECT t._key, COALESCE(json_extract(t.name,'$.en'),'?'), t.groupID,
		       COALESCE(t.portionSize,1), COALESCE(t.volume,0), tm.materials
		FROM types t LEFT JOIN typeMaterials tm ON tm._key = t._key
		WHERE t._key = ?`, oreTypeID).Scan(&o.TypeID, &o.Name, &o.GroupID, &o.PortionSize, &o.VolumeM3, &matsJSON)
	if err != nil {
		return nil, fmt.Errorf("get ore %d: %w", oreTypeID, err)
	}
	if matsJSON.Valid && matsJSON.String != "" {
		if err := json.Unmarshal([]byte(matsJSON.String), &o.Materials); err != nil {
			return nil, fmt.Errorf("parse materials for %d: %w", oreTypeID, err)
		}
	}
	return &o, nil
}

// ListOres returns all published asteroid ores (category 'Asteroid').
func ListOres(db *sql.DB) ([]Ore, error) {
	rows, err := db.Query(`
		SELECT t._key, COALESCE(json_extract(t.name,'$.en'),'?'), t.groupID,
		       COALESCE(t.portionSize,1), COALESCE(t.volume,0)
		FROM types t
		JOIN groups g ON t.groupID = g._key
		JOIN categories c ON g.categoryID = c._key
		WHERE json_extract(c.name,'$.en') = 'Asteroid' AND t.published = 1
		ORDER BY t._key`)
	if err != nil { return nil, fmt.Errorf("list ores: %w", err) }
	defer func() { _ = rows.Close() }()
	var out []Ore
	for rows.Next() {
		var o Ore
		if err := rows.Scan(&o.TypeID, &o.Name, &o.GroupID, &o.PortionSize, &o.VolumeM3); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
```
> `ListOres` intentionally omits materials (loaded per-ore via `GetOre` when needed) to keep the list query cheap.

- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit:** `PATH="$HOME/.local/bin:$PATH" git add backend/pkg/evedb/reprocessing/ && PATH="$HOME/.local/bin:$PATH" git commit -m "feat(backend): SDE ore catalog + reprocessing output"`

### Task 2: Reprocessing yield from skills

**Files:** add to `backend/pkg/evedb/reprocessing/reprocessing.go`; test in `reprocessing_test.go`.

- [ ] **Step 1 — failing test:**
```go
func TestNetYield(t *testing.T) {
	// base 0.50, Reprocessing 5 (+15%), Reprocessing Efficiency 5 (+10%), ore skill 4 (+8%)
	got := NetYield(0.50, ReprocessingSkills{Reprocessing: 5, ReprocessingEfficiency: 5, OreProcessing: 4})
	want := 0.50 * 1.15 * 1.10 * 1.08
	if diff := got - want; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("want %.6f got %.6f", want, got)
	}
}
```

- [ ] **Step 2 — run, expect fail.**
- [ ] **Step 3 — implement:**
```go
type ReprocessingSkills struct {
	Reprocessing           int // +3%/level
	ReprocessingEfficiency int // +2%/level
	OreProcessing          int // ore-specific, +2%/level
}

// NetYield = base × (1+0.03·R) × (1+0.02·RE) × (1+0.02·OP). base = station reprocessingEfficiency (0.50 NPC).
func NetYield(baseRate float64, s ReprocessingSkills) float64 {
	return baseRate *
		(1 + 0.03*float64(s.Reprocessing)) *
		(1 + 0.02*float64(s.ReprocessingEfficiency)) *
		(1 + 0.02*float64(s.OreProcessing))
}
```
- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit:** `... -m "feat(backend): reprocessing net-yield from skills"`

### Task 3: Best NPC reprocessing station in a region (by standing)

**Files:** add `GetRegionReprocessStations` to `backend/internal/database/sde.go` (mirror existing `SDERepository` methods); test `backend/internal/database/sde_reprocess_test.go`.

- [ ] **Step 1 — failing test** (The Forge = 10000002 has many NPC stations):
```go
func TestGetRegionReprocessStations_TheForge(t *testing.T) {
	r := newTestSDERepo(t) // mirror the existing sde_test.go setup
	st, err := r.GetRegionReprocessStations(context.Background(), 10000002)
	if err != nil { t.Fatal(err) }
	if len(st) == 0 { t.Fatal("no reprocess stations in The Forge") }
	for _, s := range st {
		if s.BaseRate <= 0 || s.BaseTake < 0 || s.OwnerCorpID == 0 {
			t.Fatalf("bad station row: %+v", s)
		}
	}
}
```
> Read `backend/internal/database/sde_test.go` for the exact repo-construction helper and mirror it (don't invent `newTestSDERepo` if a differently-named helper exists).

- [ ] **Step 2 — run, expect fail.**
- [ ] **Step 3 — implement** in `sde.go`:
```go
type ReprocessStation struct {
	StationID   int64
	OwnerCorpID int64
	BaseRate    float64 // reprocessingEfficiency (0.50)
	BaseTake    float64 // reprocessingStationsTake (0.05)
}

// GetRegionReprocessStations lists NPC stations with reprocessing in a region.
func (r *SDERepository) GetRegionReprocessStations(ctx context.Context, regionID int) ([]ReprocessStation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT n._key, n.ownerID, COALESCE(n.reprocessingEfficiency,0), COALESCE(n.reprocessingStationsTake,0)
		FROM npcStations n
		JOIN mapSolarSystems s ON n.solarSystemID = s._key
		JOIN constellations cn ON s.constellationID = cn._key
		WHERE cn.regionID = ? AND COALESCE(n.reprocessingEfficiency,0) > 0`, regionID)
	if err != nil { return nil, fmt.Errorf("region reprocess stations: %w", err) }
	defer func() { _ = rows.Close() }()
	var out []ReprocessStation
	for rows.Next() {
		var s ReprocessStation
		if err := rows.Scan(&s.StationID, &s.OwnerCorpID, &s.BaseRate, &s.BaseTake); err != nil { return nil, err }
		out = append(out, s)
	}
	return out, rows.Err()
}
```
> Verify the `mapSolarSystems`→region join columns against the SDE before finalizing (the repo already joins to region in `GetRegionIDForSystem` — reuse that exact join shape: it may go `mapSolarSystems.constellationID → constellations.regionID`, or a direct `regionID`. Match what `GetRegionIDForSystem` does).

- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit:** `... -m "feat(backend): list region NPC reprocessing stations"`

### Task 4: Reprocessing tax from standing + best-station selection (pure logic)

**Files:** `backend/internal/services/mining_tax.go` + `mining_tax_test.go`.

- [ ] **Step 1 — failing test:**
```go
func TestReprocessTax(t *testing.T) {
	// tax = max(0, baseTake − 0.0075·standing)
	if v := ReprocessTax(0.05, 0); v != 0.05 { t.Fatalf("standing 0 → %v", v) }
	if v := ReprocessTax(0.05, 4); v < 0.0199 || v > 0.0201 { t.Fatalf("standing 4 → %v", v) } // 0.05-0.03
	if v := ReprocessTax(0.05, 10); v != 0 { t.Fatalf("standing 10 → %v", v) }
}

func TestBestStation_PicksLowestTax(t *testing.T) {
	stations := []StationStanding{{StationID: 1, BaseTake: 0.05, Standing: 0}, {StationID: 2, BaseTake: 0.05, Standing: 6.67}}
	best := BestStation(stations)
	if best == nil || best.StationID != 2 { t.Fatalf("expected station 2 (zero tax), got %+v", best) }
}
```

- [ ] **Step 2 — run, expect fail.**
- [ ] **Step 3 — implement:**
```go
package services

import "math"

func ReprocessTax(baseTake, standing float64) float64 {
	return math.Max(0, baseTake-0.0075*standing)
}

type StationStanding struct {
	StationID int64
	BaseRate  float64
	BaseTake  float64
	Standing  float64 // player's standing with the station's owner corp
}

// BestStation returns the station with the lowest reprocessing tax (nil if none).
func BestStation(s []StationStanding) *StationStanding {
	var best *StationStanding
	bestTax := math.Inf(1)
	for i := range s {
		if tax := ReprocessTax(s[i].BaseTake, s[i].Standing); tax < bestTax {
			bestTax, best = tax, &s[i]
		}
	}
	return best
}
```
- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit:** `... -m "feat(backend): reprocessing tax + best-station logic"`

### Task 5: Per-ore comparison (raw vs refine, taker, net ISK/m³) — pure logic

**Files:** `backend/internal/services/mining_compare.go` + `mining_compare_test.go`.

Inputs are already-resolved buy prices (ISK/unit) so this is pure and fully testable.

- [ ] **Step 1 — failing test:**
```go
func TestCompareOre_RefineWins(t *testing.T) {
	in := OreCompareInput{
		PortionSize: 100, OreVolumeM3: 0.1, OreBuyPrice: 8.0, // ore ISK/unit
		Materials: []MaterialValue{{Qty: 400, BuyPrice: 5.0}}, // 400 mineral @5 per 100 ore
		NetYield: 0.90, StationTake: 0.02, SalesTaxRate: 0.04,
	}
	r := CompareOre(in)
	// raw: per ore unit 8 net of 4% tax = 7.68 → per m³ /0.1 = 76.8
	// refine per 100 ore: 400*0.90*5 = 1800 gross, minus 2% station take = 1764, minus 4% sales tax = 1693.44; per ore unit /100 = 16.9344; per m³ /0.1 = 169.344
	if r.RawNetPerM3 < 76.7 || r.RawNetPerM3 > 76.9 { t.Fatalf("raw/m3 %.4f", r.RawNetPerM3) }
	if r.RefineNetPerM3 < 169.3 || r.RefineNetPerM3 > 169.4 { t.Fatalf("refine/m3 %.4f", r.RefineNetPerM3) }
	if r.Best != "refine" { t.Fatalf("best %s", r.Best) }
}

func TestCompareOre_RawWins(t *testing.T) {
	in := OreCompareInput{PortionSize: 100, OreVolumeM3: 0.1, OreBuyPrice: 100, Materials: []MaterialValue{{Qty: 400, BuyPrice: 1}}, NetYield: 0.5, StationTake: 0.05, SalesTaxRate: 0.04}
	if CompareOre(in).Best != "raw" { t.Fatal("raw should win") }
}
```

- [ ] **Step 2 — run, expect fail.**
- [ ] **Step 3 — implement:**
```go
package services

type MaterialValue struct {
	Qty      int64   // material output per ONE portionSize batch (from typeMaterials)
	BuyPrice float64 // ISK/unit highest buy order
}

type OreCompareInput struct {
	PortionSize  int64
	OreVolumeM3  float64
	OreBuyPrice  float64
	Materials    []MaterialValue
	NetYield     float64 // reprocessing.NetYield(...) for the best station
	StationTake  float64 // ReprocessTax(...) for the best station
	SalesTaxRate float64 // from Accounting skill
}

type OreCompareResult struct {
	RawNetPerM3    float64
	RefineNetPerM3 float64
	Best           string // "raw" | "refine"
	DeltaPerM3     float64
}

func CompareOre(in OreCompareInput) OreCompareResult {
	salesMul := 1 - in.SalesTaxRate
	// raw: ISK per ore unit, net of sales tax, per m³
	rawPerM3 := in.OreBuyPrice * salesMul / in.OreVolumeM3
	// refine: value of one portionSize batch → per ore unit → per m³
	var batchGross float64
	for _, m := range in.Materials {
		batchGross += float64(m.Qty) * in.NetYield * m.BuyPrice
	}
	batchNet := batchGross * (1 - in.StationTake) * salesMul
	refinePerUnit := batchNet / float64(in.PortionSize)
	refinePerM3 := refinePerUnit / in.OreVolumeM3
	r := OreCompareResult{RawNetPerM3: rawPerM3, RefineNetPerM3: refinePerM3}
	if refinePerM3 >= rawPerM3 { r.Best = "refine" } else { r.Best = "raw" }
	r.DeltaPerM3 = abs(refinePerM3 - rawPerM3)
	return r
}

func abs(f float64) float64 { if f < 0 { return -f }; return f }
```
> If `abs` collides with an existing helper in `package services`, reuse the existing one and drop this.

- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit:** `... -m "feat(backend): per-ore raw-vs-refine taker comparison"`

### Task 6: MiningService.OreRanking orchestration (no mining yield yet) + endpoint

**Files:** Create `backend/internal/services/mining_service.go`, `backend/internal/handlers/mining.go`, `backend/internal/models/mining.go`; register route in `cmd/api/main.go` + wire in the container (`internal/container/*.go`). Test `backend/internal/services/mining_service_test.go`.

The service: for the region's in-scope ores → buy prices (market repo) + reprocessing skills + best station per ore (standings) → `CompareOre` → ranked rows. Until Phase 2, `MiningM3PerHour = 0` and ranking falls back to `best net ISK/m³` (the response includes both, the row's `isk_per_hour` = 0 for now; sort by `best_net_per_m3`).

- [ ] **Step 1 — read** these for exact interfaces, then write the service against them: `internal/database/market.go` (buy-order accessor + `Order` struct fields), `internal/services/skills_service.go` (`GetCharacterSkills` → `TradingSkills`; the sales-tax/Accounting field), `internal/handlers/portfolio.go` + `cmd/api/main.go:156` (route + handler + auth-middleware pattern), `internal/container` (how services/handlers are constructed and injected).

- [ ] **Step 2 — model** (`models/mining.go`):
```go
package models

type OreRankingRequest struct {
	RegionID int    `json:"region_id"` // 0 = character's current region
	SecBand  string `json:"sec_band"`  // "high" | "low" | "null"
}

type OreRankRow struct {
	OreTypeID        int64   `json:"ore_type_id"`
	OreName          string  `json:"ore_name"`
	MiningM3PerHour  float64 `json:"mining_m3_per_hour"`
	RawISKPerHour    float64 `json:"raw_isk_per_hour"`
	RefineISKPerHour float64 `json:"refine_isk_per_hour"`
	RawNetPerM3      float64 `json:"raw_net_per_m3"`
	RefineNetPerM3   float64 `json:"refine_net_per_m3"`
	Best             string  `json:"best"`
	DeltaISKPerHour  float64 `json:"delta_isk_per_hour"`
	BestStationID    int64   `json:"best_station_id,omitempty"`
	BestStationTax   float64 `json:"best_station_tax"`
}

type OreRankingResponse struct {
	RegionID int          `json:"region_id"`
	SecBand  string       `json:"sec_band"`
	Rows     []OreRankRow `json:"rows"`
}
```

- [ ] **Step 3 — failing test** for the service with faked deps (mirror how `portfolio_service_test.go` fakes the routes service). Construct `MiningService` with a fake market repo returning a known Veldspar buy price + a known Tritanium buy price, a fake skills service, a fake SDE repo returning one station + the ore list filtered to Veldspar; assert the response has a Veldspar row with `Best` set and the per-m³ values matching `CompareOre`. Write the fakes minimally (record/return canned data).

- [ ] **Step 4 — implement** `mining_service.go` (`OreRanking(ctx, characterID, accessToken, req) (*models.OreRankingResponse, error)`):
  - resolve `regionID` (req or current via location — reuse the existing current-region resolution used by hauling/trading).
  - `ores := reprocessing.ListOres(sdeDB)` filtered by `secBandOres(req.SecBand)` (the curated map, Task 7) and to ores the ship can mine (Phase 2 narrows this; Phase 1: just the sec-band map).
  - skills := skillsService.GetCharacterSkills(...) → reprocessing skills + Accounting sales-tax rate. (Add reprocessing-skill fetching to skills_service in this task — see note.)
  - stations := sdeRepo.GetRegionReprocessStations(regionID); standings (Phase 2 adds the ESI standings; Phase 1: assume standing 0 → BestStation = lowest BaseTake).
  - per ore: `ore := reprocessing.GetOre(...)`; material buy prices via market repo; `net := reprocessing.NetYield(bestStation.BaseRate, reproSkills+oreProcessingForThisOre)`; `CompareOre(...)`; build row (isk/h = 0 until Phase 2).
  - sort rows by `RefineNetPerM3`/`RawNetPerM3` max desc (Phase 2 re-sorts by isk/h).
  > **Reprocessing skills in skills_service:** extend `TradingSkills` (or add a `GetReprocessingSkills`) to fetch Reprocessing (3385), Reprocessing Efficiency (3389), and the ore-specific Processing skill per ore group from ESI skills. For the ore-specific skill, map ore group → "<Group> Processing" skill type via SDE name lookup (`<GroupName> Processing`), or 0 if untrained. Keep it a focused addition with its own unit coverage.

- [ ] **Step 5 — handler + route:** `mining.go` `OreRanking(c *fiber.Ctx)` mirroring `portfolio.go` (parse body, get characterID+token from locals/auth middleware, call service, JSON). Register: `api.Post("/mining/ore-ranking", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.MiningHandler.OreRanking)` in `cmd/api/main.go`; construct `MiningHandler`+`MiningService` in the container and add to the `Container` struct.

- [ ] **Step 6 — verify + commit:** `cd backend && gofmt -l internal pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./... && SDE_DB_PATH=… GOWORK=off go test ./...` then `... -m "feat(backend): mining ore-ranking service + endpoint (phase 1)"`.

### Task 7: Curated ore → security-band map

**Files:** `backend/internal/services/ore_secband.go` + test.

- [ ] **Step 1 — failing test:** `secBandOres("high")` contains Veldspar group (462) and Scordite; `secBandOres("null")` contains Arkonor (450); high does NOT contain Arkonor.
- [ ] **Step 2-4 — implement** a `map[string]map[int64]bool` (band → ore group ids), seeded from the well-known EVE ore availability (highsec: Veldspar/Scordite/Plagioclase/Pyroxeres/Omber/Kernite/Jaspet/Hemorphite/Hedbergite; lowsec adds Gneiss/Ochre/Spodumain-low; null/anomalies add Arkonor/Bistot/Crokite/etc.). Expose `secBandOreGroups(band string) map[int64]bool` and have Task 6's filter use ore `GroupID`. Document each group id with a comment.
- [ ] **Step 5 — commit:** `... -m "feat(backend): curated ore→security-band map"`

---

## PHASE 2 — Mining yield (m³/h) + standings ESI + ISK/hour

### Task 8: Mining yield from the current ship's fitted miners + skills

**Files:** `backend/pkg/evedb/mining/mining.go` + test. Reuse the asset/fitting access already in `fitting_service.go` (`fittedItemsForShip`, `fetchESIAssets`).

- [ ] **Step 1 — failing test** (Strip Miner I: attr77=150 m³, attr73=45000 ms; Mining V +25%, Astrogeology V +25%):
```go
func TestMiningRate_OneStripMinerI(t *testing.T) {
	db := testutil.OpenTestDB(t)
	// one Strip Miner I (17482); Mining 5, Astrogeology 5
	m3h, err := MiningRateM3PerHour(db, []int64{17482}, MiningSkills{Mining: 5, Astrogeology: 5}, 1.0)
	if err != nil { t.Fatal(err) }
	// per cycle 150 × (1+0.25)(1+0.25)=234.375 m³ / 45s × 3600 = 18750 m³/h
	if m3h < 18740 || m3h > 18760 { t.Fatalf("m3/h %.1f", m3h) }
}
```

- [ ] **Step 2 — run, expect fail.**
- [ ] **Step 3 — implement:** for each module type id, read dogma attr 77 (`miningAmount`) + 73 (`duration` ms) from `typeDogma`; `perCycle = miningAmount × (1+0.05·Mining) × (1+0.05·Astrogeology) × crystalMul`; `m3h += perCycle / (duration/1000) × 3600`. `crystalMul` param (default 1.0; ore-specific crystal handled by the caller). Skip non-mining modules (no attr 77). Confirm Mining=3386 / Astrogeology=3410 skill ids via SDE name lookup in a comment.
- [ ] **Step 4 — run, expect PASS.**
- [ ] **Step 5 — commit:** `... -m "feat(backend): mining yield m³/h from fitted miners + skills"`

### Task 9: Character standings via ESI + mining/reprocessing skill fetch

**Files:** extend `backend/internal/services/skills_service.go` (or a new `standings_service.go`): `GetCharacterStandings(ctx, characterID, token) (map[int64]float64, error)` — corp/faction id → standing, from `GET /characters/{id}/standings/` (model on the existing ESI calls in `fitting_service.go`/`skills_service.go`). Also fetch Mining/Astrogeology + Reprocessing skills (if not already). Test the JSON parse with a fixture (`[{"from_type":"npc_corp","from_id":1000035,"standing":7.5}]` → map[1000035]=7.5).

- [ ] TDD as in prior tasks (parse test first). Commit: `... -m "feat(backend): ESI character standings + mining/reprocessing skills"`.

### Task 10: Wire mining yield + standings into OreRanking → ISK/hour

**Files:** `mining_service.go`, `mining_service_test.go`.

- [ ] Update `OreRanking`: compute `m3h := mining.MiningRateM3PerHour(...)` from the current ship's fitted miners (via fitting_service assets) + skills; per ore apply the matching ore crystal multiplier if the ship has that crystal (else 1.0). Use real per-station standings (Task 9) for `BestStation`. Set `row.RawISKPerHour = m3h × RawNetPerM3`, `RefineISKPerHour = m3h × RefineNetPerM3`, `DeltaISKPerHour = m3h × DeltaPerM3`. **Sort rows by max(raw,refine) ISK/hour desc.** If the ship has no mining modules → return rows with `m3h=0` and a top-level flag/empty so the UI shows "kein Mining-Setup" (fail-loud, don't fabricate).
- [ ] Extend the service test: with a known m³/h, assert isk/h = m3h × per-m³ and the ordering.
- [ ] Verify gates + commit: `... -m "feat(backend): ore-ranking ISK/hour from mining yield + per-station standings"`.

---

## PHASE 3 — Web "Mining" view

### Task 11: API client + types (web)

**Files:** `frontend/src/types/trading.ts` (add `OreRankingRequest`, `OreRankRow`, `OreRankingResponse` mirroring the Go JSON tags), `frontend/src/lib/api-client.ts` (`fetchOreRanking(req): Promise<OreRankingResponse>` mirroring `optimizePortfolio` — POST, credentials, throw on !ok, validate `rows` is an array). Test `frontend/tests/lib/api-client-mining.test.ts` (mock fetch → returns rows; malformed → throws). TDD; commit `feat(frontend): mining ore-ranking api client`.

### Task 12: Mining page + ranked table (web)

**Files:** `frontend/src/app/mining/page.tsx`, `frontend/src/components/trading/OreRankingTable.tsx`, nav entry; test `frontend/tests/components/OreRankingTable.test.tsx`.

- [ ] Mirror `roi-calculator/page.tsx` structure: `useCurrentShip()` + `CurrentShipCard`, a **sec-band selector** (high/low/null — a simple Select), region (current), a "berechnen" trigger calling `fetchOreRanking({ region_id: 0, sec_band })`. Render `OreRankingTable` (columns: Ore · m³/h · ISK/h raw · ISK/h refine · Verdict (best, colored) · Station-Tax · Δ). Show a "kein Mining-Setup" notice when the response indicates no mining modules. Add a nav link "Mining" alongside the others.
- [ ] Component test: given rows, the table renders ore names, the verdict per row, and highlights the better column. TDD.
- [ ] Verify: `npx vitest run && rm -rf .next && npm run build && npx eslint <files>`; commit `feat(frontend): mining view with ore ranking table`.

---

## PHASE 4 — Flutter "Mining" view

### Task 13: Flutter models + API + screen

**Files:** `app/lib/api/mining_models.dart` (request/response mirroring the JSON), `app/lib/features/mining/mining_api.dart` + `mining_providers.dart` + `mining_screen.dart` + an `ore_ranking_table` widget; tests in `app/test/`.

- [ ] Mirror an existing screen (e.g. `roi_calculator_screen.dart`): `currentShipProvider` + `CurrentShipCard`, sec-band selector, call the endpoint, render the ranked table. Reuse the dio client + Riverpod patterns exactly.
- [ ] Tests: model fromJson, a widget test rendering the ranked rows + verdict, no-mining-setup state.
- [ ] Verify: `cd app && flutter analyze && flutter test`; commit `feat(flutter): mining ore-ranking screen`.

---

## Task 14: Full verification + PR

- [ ] Backend: `gofmt -l internal pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./... && SDE_DB_PATH=… GOWORK=off go test ./...` → all green, golangci 0.
- [ ] Web: `cd frontend && npx vitest run && rm -rf .next && npm run build`. Flutter: `cd app && flutter analyze && flutter test`.
- [ ] Push `feat/mining-ore-ranking`; open PR (base main). After CI green + merge → release `v0.20.0` (gated; CHANGELOG + `make release` + tag → deploy + smoke-test; Flutter APK rebuilt/installed separately).

---

## Self-review notes

- **Spec coverage:** reprocessing yield (T2), ore catalog/materials (T1), best NPC station by standing (T3/T4/T9), per-ore raw-vs-refine taker (T5), mining yield (T8), ISK/hour + ranking (T10), sec-band selectable + curated map (T7, T12/T13), endpoint (T6), web view (T11/T12), Flutter view (T13), fail-loud no-mining-setup (T10/T12/T13). All covered.
- **Phasing:** Phase 1 ships a working "sell-raw vs refine per ore, ranked by net ISK/m³" without mining yield; Phase 2 adds ISK/hour; 3/4 are the views. Each phase is independently testable.
- **Verify during execution:** exact `mapSolarSystems`→region join (match `GetRegionIDForSystem`); the market repo's buy-order accessor signature; the Accounting/sales-tax field on `TradingSkills`; the ore-specific "<Group> Processing" skill name→id lookup; the ESI `/standings` response shape; the mining/astrogeology skill ids (3386/3410). These are lookups, not design changes.
