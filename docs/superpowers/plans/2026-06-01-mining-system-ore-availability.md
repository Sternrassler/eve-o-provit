# System-genaue Erz-Verfügbarkeit (High+Low) + echte Namen — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The mining ranking shows only the ores that actually spawn in the character's current system (deterministic from EVE's security + empire-quarter rules, high-sec and low-sec), with the real in-game ore names, and a High-only/High+Low willingness toggle.

**Architecture:** A new `pkg/evedb/mining/availability.go` derives a system's empire quarter + security from the SDE and returns the documented ore-group set. `reprocessing.ListOres` resolves real ore names from CCP's compression-blueprint data and drops un-belt-able "-Grade" variants. The mining service resolves the current system once, computes the ore set, and uses an `allow_low_sec` boolean (replacing `sec_band`) for both the ore set and routing. Web + Flutter switch to a High-only/High+Low toggle (clean break, no backward compat; APK rebuilt).

**Tech Stack:** Go 1.24, SQLite SDE, Next.js/TS, Flutter/Dart.

**Spec:** `docs/superpowers/specs/2026-06-01-mining-system-ore-availability-design.md`. Deferred: #161 (null), #162 (Class K), #163 (scanner/manual).

**Verified SDE facts (do not re-derive):**
- Ore group ids: Veldspar 462, Scordite 460, Plagioclase 458, Pyroxeres 459, Omber 469, Kernite 457, Jaspet 456, Hemorphite 455, Hedbergite 454.
- `mapSolarSystems`: `securityStatus`, `regionID`. `mapRegions.factionID`. Factions: 500001 Caldari, 500002 Minmatar, 500003 Amarr, 500004 Gallente, 500007 Ammatar→Amarr, 500008 Khanid→Amarr.
- Jita (30000142): The Forge (region 10000002, faction 500001 Caldari), securityStatus 0.946 → **displayed 0.9**.
- Real ore names derive from `blueprints`: a row whose `blueprintTypeID`'s type name is `Compressed <Descriptive> Blueprint` has `activities.manufacturing.materials[0].typeID` = the raw ore variant (e.g. BP "Compressed Vivid Hemorphite Blueprint" → material 17444 = Hemorphite II-Grade ⇒ 17444 is "Vivid Hemorphite"). IV-Grade/0-Grade ores have no such blueprint.
- EVE display-security rounds to 1 decimal (half away from zero); a positive value that rounds to 0.0 is shown as 0.1.

**Test DB:** `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off` from `backend/`.

---

## Task 1: Ore availability by system (`availability.go`)

**Files:**
- Create: `backend/pkg/evedb/mining/availability.go`
- Test: `backend/pkg/evedb/mining/availability_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/pkg/evedb/mining/availability_test.go
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
	// Veldspar 462, Scordite 460, Pyroxeres 459, Plagioclase 458,
	// Omber 469, Kernite 457, Jaspet 456, Hemorphite 455, Hedbergite 454.

	// Jita (Caldari, displayed 0.9): Veldspar, Scordite, Pyroxeres; no Plagioclase (needs <=0.7).
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

	// Amarr 0.3 low-sec, allowLow=true: Pyroxeres, Kernite, Jaspet (no Hemorphite until <=0.2).
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
```

- [ ] **Step 2: Run it to confirm it fails**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run 'TestSystemQuarter|TestAvailableOre' -v`
Expected: FAIL — `undefined: SystemQuarterAndSec`, `undefined: AvailableOreGroups`.

- [ ] **Step 3: Implement**

```go
// backend/pkg/evedb/mining/availability.go
package mining

import (
	"database/sql"
	"fmt"
	"math"
)

// Ore group ids (SDE "Asteroid" category).
const (
	grpVeldspar    = 462
	grpScordite    = 460
	grpPlagioclase = 458
	grpPyroxeres   = 459
	grpOmber       = 469
	grpKernite     = 457
	grpJaspet      = 456
	grpHemorphite  = 455
	grpHedbergite  = 454
)

// SystemQuarterAndSec returns a system's empire quarter ("amarr"/"caldari"/
// "gallente"/"minmatar", or "" if not an empire region) and its raw security
// status, from the SDE.
func SystemQuarterAndSec(db *sql.DB, systemID int64) (quarter string, sec float64, err error) {
	var regionID int64
	err = db.QueryRow(
		`SELECT securityStatus, regionID FROM mapSolarSystems WHERE _key = ?`, systemID,
	).Scan(&sec, &regionID)
	if err != nil {
		return "", 0, fmt.Errorf("system %d: %w", systemID, err)
	}
	var factionID sql.NullInt64
	if err = db.QueryRow(`SELECT factionID FROM mapRegions WHERE _key = ?`, regionID).Scan(&factionID); err != nil {
		return "", sec, fmt.Errorf("region %d: %w", regionID, err)
	}
	return factionToQuarter(factionID.Int64), sec, nil
}

func factionToQuarter(factionID int64) string {
	switch factionID {
	case 500003, 500007, 500008: // Amarr, Ammatar, Khanid
		return "amarr"
	case 500001: // Caldari
		return "caldari"
	case 500004: // Gallente
		return "gallente"
	case 500002: // Minmatar
		return "minmatar"
	default:
		return ""
	}
}

// displaySec rounds the raw security status to EVE's displayed 1-decimal value
// (half away from zero); a positive value that rounds to 0.0 shows as 0.1.
func displaySec(sec float64) float64 {
	r := math.Round(sec*10) / 10
	if sec > 0 && r <= 0 {
		return 0.1
	}
	return r
}

// AvailableOreGroups returns the ore groups that spawn in a system, per EVE
// University's documented high-sec/low-sec rules. Empire quarter + displayed
// security decide the set. allowLow gates low-sec systems: in a low-sec system
// with allowLow=false the result is empty (the player won't operate there).
// Null-sec (sec<=0) is out of scope (empty); see issue #161.
func AvailableOreGroups(quarter string, sec float64, allowLow bool) map[int64]bool {
	out := map[int64]bool{}
	d := displaySec(sec)
	switch {
	case d >= 0.5: // high-sec
		out[grpVeldspar] = true
		out[grpScordite] = true
		if d <= 0.9 {
			out[grpPyroxeres] = true
		}
		switch quarter {
		case "gallente", "minmatar":
			if d <= 0.9 {
				out[grpPlagioclase] = true
			}
		case "caldari":
			if d <= 0.7 {
				out[grpPlagioclase] = true
			}
			// amarr: no Plagioclase in high-sec
		}
	case d > 0 && allowLow: // low-sec, player willing
		switch quarter {
		case "amarr":
			out[grpPyroxeres], out[grpKernite], out[grpJaspet] = true, true, true
			if d <= 0.2 {
				out[grpHemorphite] = true
			}
		case "caldari":
			out[grpKernite], out[grpPyroxeres] = true, true
			if d <= 0.2 {
				out[grpHedbergite] = true
			}
		case "gallente":
			out[grpOmber], out[grpJaspet] = true, true
			if d <= 0.2 {
				out[grpHemorphite] = true
			}
		case "minmatar":
			out[grpKernite], out[grpOmber] = true, true
			if d <= 0.2 {
				out[grpHedbergite] = true
			}
		}
	}
	return out
}
```

- [ ] **Step 4: Run it to confirm it passes**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run 'TestSystemQuarter|TestAvailableOre' -v && gofmt -l pkg/evedb/mining/ && GOWORK=off go vet ./pkg/evedb/mining/...`
Expected: PASS; no gofmt/vet output.

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/evedb/mining/availability.go backend/pkg/evedb/mining/availability_test.go
git commit -m "feat(mining): deterministic per-system ore availability (high+low)"
```

---

## Task 2: Real ore names + drop un-belt-able variants (`reprocessing.go`)

**Files:**
- Modify: `backend/pkg/evedb/reprocessing/reprocessing.go`
- Modify: `backend/pkg/evedb/reprocessing/reprocessing_test.go`

- [ ] **Step 1: Write the failing test**

Add to `reprocessing_test.go`:

```go
func TestListOres_RealNamesAndNoGrade(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()
	ores, err := ListOres(db)
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]string{}
	for _, o := range ores {
		byID[o.TypeID] = o.Name
		if strings.Contains(o.Name, "-Grade") {
			t.Errorf("'-Grade' name leaked: %d %q", o.TypeID, o.Name)
		}
	}
	if byID[17470] != "Concentrated Veldspar" {
		t.Errorf("17470: got %q, want Concentrated Veldspar", byID[17470])
	}
	if byID[17444] != "Vivid Hemorphite" {
		t.Errorf("17444: got %q, want Vivid Hemorphite", byID[17444])
	}
	if byID[1230] != "Veldspar" {
		t.Errorf("1230: got %q, want Veldspar", byID[1230])
	}
	if _, ok := byID[46689]; ok { // Veldspar IV-Grade — no blueprint name → filtered
		t.Errorf("Veldspar IV-Grade (46689) must be filtered out")
	}
}
```

`strings` is already imported in this test file (used by the compressed-variant test).

- [ ] **Step 2: Run it to confirm it fails**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/reprocessing/ -run TestListOres_RealNames -v`
Expected: FAIL — names are still "Veldspar II-Grade" and 46689 present.

- [ ] **Step 3: Add the display-name resolver + apply it in `ListOres`**

Add `"strings"` to the imports of `reprocessing.go` if not present. Add this helper:

```go
// oreDisplayNames maps a raw ore variant typeID to its real in-game name, derived
// from CCP's compression blueprints: a "Compressed <Descriptive> Blueprint" lists
// the raw ore variant as its manufacturing material, so <Descriptive> is that
// ore's display name. Base ores map to their own name (identity). Variants without
// such a blueprint (e.g. IV-Grade / 0-Grade) are absent from the map.
func oreDisplayNames(db *sql.DB) (map[int64]string, error) {
	rows, err := db.Query(`
		SELECT json_extract(b.activities,'$.manufacturing.materials[0].typeID'),
		       replace(replace(json_extract(t.name,'$.en'),'Compressed ',''),' Blueprint','')
		FROM blueprints b
		JOIN types t ON t._key = b.blueprintTypeID
		WHERE json_extract(t.name,'$.en') LIKE 'Compressed %Blueprint'`)
	if err != nil {
		return nil, fmt.Errorf("ore display names: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]string{}
	for rows.Next() {
		var oreID sql.NullInt64
		var name string
		if err := rows.Scan(&oreID, &name); err != nil {
			return nil, err
		}
		if oreID.Valid && name != "" {
			out[oreID.Int64] = name
		}
	}
	return out, rows.Err()
}
```

Then, in `ListOres`, after building `out` from the rows, REPLACE the current scan loop so it renames + filters. Change the loop body from:

```go
	var out []Ore
	for rows.Next() {
		var o Ore
		if err := rows.Scan(&o.TypeID, &o.Name, &o.GroupID, &o.PortionSize, &o.VolumeM3); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
```

to:

```go
	disp, err := oreDisplayNames(db)
	if err != nil {
		return nil, err
	}
	var out []Ore
	for rows.Next() {
		var o Ore
		if err := rows.Scan(&o.TypeID, &o.Name, &o.GroupID, &o.PortionSize, &o.VolumeM3); err != nil {
			return nil, err
		}
		if name, ok := disp[o.TypeID]; ok {
			o.Name = name // real in-game name (Concentrated Veldspar, Vivid Hemorphite, …)
		} else if strings.Contains(o.Name, "-Grade") {
			continue // un-belt-able variant (IV-Grade/0-Grade): not mined, drop it
		}
		out = append(out, o)
	}
	return out, rows.Err()
```

Note: `oreDisplayNames` runs one query before the row loop; calling `db.Query` while iterating `rows` is fine because they are independent statements, but to be safe resolve `disp` BEFORE the `rows` loop as shown (it is). Confirm `rows` (the ore query) is still open — yes, the `disp` query uses a separate `*sql.Rows`.

- [ ] **Step 4: Run it to confirm it passes**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/reprocessing/ -v`
Expected: PASS (new test + existing `TestListOres_*`). If `TestListOres_ExcludesCompressedVariants` asserts `in[17470]` (Veldspar II-Grade) is present by typeID — it still is (renamed, not removed), so it passes.

- [ ] **Step 5: gofmt + vet + commit**

Run: `cd backend && gofmt -l pkg/evedb/reprocessing/ && GOWORK=off go vet ./pkg/evedb/reprocessing/...`
```bash
git add backend/pkg/evedb/reprocessing/reprocessing.go backend/pkg/evedb/reprocessing/reprocessing_test.go
git commit -m "feat(mining): real ore names from blueprints; drop un-belt-able -Grade variants"
```

---

## Task 3: Request/response model (`models/mining.go`)

**Files:**
- Modify: `backend/internal/models/mining.go`

- [ ] **Step 1: Replace `SecBand` with `AllowLowSec` in the request**

Change:
```go
type OreRankingRequest struct {
	RegionID int    `json:"region_id"`
	SecBand  string `json:"sec_band"`
}
```
to:
```go
type OreRankingRequest struct {
	RegionID    int  `json:"region_id"`
	AllowLowSec bool `json:"allow_low_sec"` // true = willing to operate in low-sec too
}
```

- [ ] **Step 2: Update the response — drop `SecBand`, add system context**

Change:
```go
type OreRankingResponse struct {
	RegionID      int          `json:"region_id"`
	SecBand       string       `json:"sec_band"`
	NoMiningSetup bool         `json:"no_mining_setup"`
	Rows          []OreRankRow `json:"rows"`
}
```
to:
```go
type OreRankingResponse struct {
	RegionID          int          `json:"region_id"`
	SystemSecurity    float64      `json:"system_security,omitempty"` // current system, displayed sec
	Quarter           string       `json:"quarter,omitempty"`         // amarr|caldari|gallente|minmatar|""
	NoMiningSetup     bool         `json:"no_mining_setup"`
	NotAvailableReason string      `json:"not_available_reason,omitempty"` // set when no ores apply here
	Rows              []OreRankRow `json:"rows"`
}
```

- [ ] **Step 3: Build**

Run: `cd backend && GOWORK=off go build ./internal/models/...`
Expected: builds.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/models/mining.go
git commit -m "feat(mining): allow_low_sec request + system-context response fields"
```

---

## Task 4: Wire system-accurate ore set into the service

**Files:**
- Modify: `backend/internal/services/mining_service.go`
- Delete: `backend/internal/services/ore_secband.go`

- [ ] **Step 1: Resolve the current system once; compute the ore set from it**

In `OreRanking`, replace the opening region/band block. The current code is:

```go
	// 1. Resolve region (request or current location).
	regionID := req.RegionID
	if regionID <= 0 {
		loc, err := s.location.GetCharacterLocation(ctx, characterID, accessToken)
		if err != nil {
			return nil, err
		}
		r, err := s.region.GetRegionIDForSystem(ctx, loc.SolarSystemID)
		if err != nil {
			return nil, err
		}
		regionID = r
	}

	resp := &models.OreRankingResponse{
		RegionID: regionID,
		SecBand:  req.SecBand,
		Rows:     []models.OreRankRow{},
	}

	// 2. Ore set for the band.
	bandGroups := secBandOreGroups(req.SecBand)
	if len(bandGroups) == 0 {
		return resp, nil
	}
```

Replace with:

```go
	// 1. Resolve the current system (origin for ore set + region + routing).
	loc, err := s.location.GetCharacterLocation(ctx, characterID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("current location unavailable: %w", err)
	}
	originSys := loc.SolarSystemID

	regionID := req.RegionID
	if regionID <= 0 {
		r, err := s.region.GetRegionIDForSystem(ctx, originSys)
		if err != nil {
			return nil, err
		}
		regionID = r
	}

	quarter, sec, err := mining.SystemQuarterAndSec(s.sdeDB, originSys)
	if err != nil {
		return nil, fmt.Errorf("system ore class: %w", err)
	}
	resp := &models.OreRankingResponse{
		RegionID:       regionID,
		SystemSecurity: math.Round(sec*10) / 10,
		Quarter:        quarter,
		Rows:           []models.OreRankRow{},
	}

	// 2. Ore set that actually spawns in this system (deterministic).
	bandGroups := mining.AvailableOreGroups(quarter, sec, req.AllowLowSec)
	if len(bandGroups) == 0 {
		resp.NotAvailableReason = "In diesem System sind mit der aktuellen Auswahl keine abbaubaren Erze verfügbar (Low-Sec? Null-Sec?)."
		return resp, nil
	}
```

This consolidates the two `GetCharacterLocation` calls. NOTE: feature #2 later in the function declared `originSys` and re-fetched location around line ~202. Find that block:

```go
	var originSys int64
	if loc, e := s.location.GetCharacterLocation(ctx, characterID, accessToken); e == nil {
		originSys = loc.SolarSystemID
	} else {
		cycleResolved = false
	}
```

and replace it with (reuse the already-resolved `originSys`; it is always set now since the function returns early on location error):

```go
	// originSys already resolved at the top; current location is mandatory now.
```

(Delete the re-fetch block entirely; `originSys` from Step 1 is in scope.)

- [ ] **Step 2: Drive routing from `AllowLowSec`**

Find `navParams := &navigation.NavigationParams{AvoidLowSec: req.SecBand == "high"}` and change to:

```go
	navParams := &navigation.NavigationParams{AvoidLowSec: !req.AllowLowSec}
```

- [ ] **Step 3: Ensure `fmt`, `math` + `mining` are imported**

`mining_service.go` already imports `math` and the `mining` evedb package (used by feature #2's `OreHoldCapacity`/`EffectiveISKPerHour`). The new code also uses `fmt.Errorf` — if `fmt` is not already in the import block, add `"fmt"`. Confirm with the build in Step 6.

- [ ] **Step 4: Delete the obsolete band map**

```bash
git rm backend/internal/services/ore_secband.go
```

The per-ore loop's `if !bandGroups[allOres[i].GroupID] { continue }` is unchanged (it now consults the system-accurate set).

- [ ] **Step 5: Remove any remaining `req.SecBand` / `resp.SecBand` references**

Run `grep -n "SecBand\|secBandOreGroups" backend/internal/services/mining_service.go` — there must be **no** matches left. If the sort/`else` branch references `m3h`/per-m³ it is unrelated and stays.

- [ ] **Step 6: Build + vet + lint**

Run: `cd backend && GOWORK=off go build ./... && GOWORK=off go vet ./internal/services/... && GOWORK=off golangci-lint run ./internal/services/...`
Expected: builds; `0 issues`. (The service test file still references the old request shape and will be fixed in Task 5; `go build ./...` does not compile tests, but `go vet` does — if vet fails on the test file, proceed to Task 5 which fixes it, then re-run vet.)

- [ ] **Step 7: Commit**

```bash
git add backend/internal/services/mining_service.go
git rm backend/internal/services/ore_secband.go
git commit -m "feat(mining): rank only the current system's actual ores; allow_low_sec toggle"
```

---

## Task 5: Service tests (`mining_service_test.go`)

**Files:**
- Modify: `backend/internal/services/mining_service_test.go`

- [ ] **Step 1: Update request construction (no more SecBand)**

Replace every `models.OreRankingRequest{RegionID: 10000002, SecBand: "high"}` with `models.OreRankingRequest{RegionID: 10000002, AllowLowSec: false}`. Remove any assertion reading `resp.SecBand`.

- [ ] **Step 2: The fake location must return a real hi-sec system**

`fakeMiningLocation.GetCharacterLocation` already returns `&CharacterLocation{SolarSystemID: 30000142}` (Jita). Jita is Caldari, displayed 0.9 → ore set {Veldspar, Scordite, Pyroxeres}. The fake market in `TestMiningService_OreRanking_Veldspar` prices Veldspar (group 462, in the set) → the Veldspar row is still produced. Keep the Veldspar assertions. If any assertion expected a non-hi-sec ore (e.g. Hemorphite) to be present, remove it — hi-sec Jita excludes it.

- [ ] **Step 3: Add an availability assertion**

Add to `TestMiningService_OreRanking_Veldspar`, after the Veldspar row is found:

```go
	// System-accurate ore set: Jita (Caldari 0.9) → no low-sec ores in the ranking.
	for _, r := range resp.Rows {
		if r.OreTypeID == 1231 || r.OreTypeID == 21 { // Hemorphite / Hedbergite (low-sec)
			t.Errorf("low-sec ore leaked into hi-sec ranking: %d %q", r.OreTypeID, r.OreName)
		}
	}
	if resp.Quarter != "caldari" {
		t.Errorf("Quarter: got %q, want caldari", resp.Quarter)
	}
```

- [ ] **Step 4: Run the mining service tests**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestMiningService -v`
Expected: PASS.

- [ ] **Step 5: Full backend gate**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... 2>&1 | grep -E "FAIL" ; gofmt -l internal pkg ; GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Expected: no FAIL; no gofmt output; `0 issues`.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/services/mining_service_test.go
git commit -m "test(mining): system-accurate ore set + allow_low_sec"
```

---

## Task 6: Web — toggle + request + system context

**Files:**
- Modify: `frontend/src/types/trading.ts`
- Modify: `frontend/src/app/mining/page.tsx`
- Modify: `frontend/src/lib/api-client.ts` (if the request is built there) — else the page

- [ ] **Step 1: Update the TS types**

In `frontend/src/types/trading.ts`, change `OreRankingRequest`:
```ts
export interface OreRankingRequest {
  region_id: number;
  sec_band: string; // "high" | "low" | "null"
}
```
to:
```ts
export interface OreRankingRequest {
  region_id: number;
  allow_low_sec: boolean;
}
```
And `OreRankingResponse`: replace `sec_band: string;` with:
```ts
  system_security?: number;
  quarter?: string;
  not_available_reason?: string;
```

- [ ] **Step 2: Replace the sec-band control with a High-only/High+Low toggle**

In `frontend/src/app/mining/page.tsx`, find the security selector (radio with high/low/null) and the state that holds it. Replace with a two-option control (e.g. a labelled checkbox/switch) backed by `const [allowLowSec, setAllowLowSec] = useState(false)`. Label: unchecked = "Nur High-Sec", checked = "High + Low-Sec". The request payload becomes `{ region_id: 0, allow_low_sec: allowLowSec }`.

Find where `sec_band` is sent (mutation/onClick building the request body) and replace with `allow_low_sec`. Grep first: `grep -rn "sec_band\|secBand\|SecBand" frontend/src`.

- [ ] **Step 3: Show the resolved system context + not-available reason**

Where the result header shows region/band, show `quarter` + `system_security` when present (e.g. "Caldari · 0.9"). When `not_available_reason` is set and `rows` is empty, render that reason text (reuse the existing empty-state styling).

- [ ] **Step 4: Tests + build + lint**

If `frontend/tests/lib/api-client-mining.test.ts` exists and asserts `sec_band`, update it to `allow_low_sec`. Run:
`cd frontend && npx vitest run && npm run build && npx eslint src/app/mining/page.tsx src/types/trading.ts`
Expected: tests pass; build ok; eslint clean.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types/trading.ts frontend/src/app/mining/page.tsx frontend/src/lib/api-client.ts frontend/tests/lib/api-client-mining.test.ts
git commit -m "feat(web): high-only/high+low toggle + system context in mining"
```

---

## Task 7: Flutter — toggle + request DTO

**Files:**
- Modify: `app/lib/api/mining_models.dart`
- Modify: `app/lib/features/mining/mining_screen.dart`
- Modify: `app/test/mining_models_test.dart`

- [ ] **Step 1: Update `OreRankingRequest` DTO**

In `mining_models.dart`, `OreRankingRequest` currently has `regionId` + `secBand` and `toJson` emitting `region_id`/`sec_band`. Replace with:
```dart
class OreRankingRequest {
  const OreRankingRequest({required this.regionId, required this.allowLowSec});
  final int regionId;
  final bool allowLowSec;
  Map<String, dynamic> toJson() => {
        'region_id': regionId,
        'allow_low_sec': allowLowSec,
      };
}
```
In `OreRankingResponse.fromJson`, drop `secBand` parsing; add `quarter`/`systemSecurity`/`notAvailableReason` fields parsed null-robustly:
```dart
      systemSecurity: (json['system_security'] as num?)?.toDouble() ?? 0.0,
      quarter: json['quarter'] as String? ?? '',
      notAvailableReason: json['not_available_reason'] as String?,
```
(Add the matching constructor params + final fields; remove the `secBand` field.)

**Important:** the screen uses `result.secBand` (e.g. `_secBandLabel(result.secBand)` in the `OreRankingTable` result header) and the helper `_secBandLabel`. Removing `secBand` will break those references. Update the header to show the system context instead, e.g. `'${result.quarter} · ${result.systemSecurity.toStringAsFixed(1)}'`, and delete the now-unused `_secBandLabel` helper. Also render `result.notAvailableReason` (when non-null and rows empty) in place of the empty state. `flutter analyze` (Step 4) will flag any leftover `secBand` reference.

- [ ] **Step 2: Update the request DTO test**

In `mining_models_test.dart`, the `OreRankingRequest.toJson` tests assert `{'region_id':…, 'sec_band':…}`. Replace with:
```dart
    test('maps region_id and allow_low_sec', () {
      const req = OreRankingRequest(regionId: 0, allowLowSec: false);
      expect(req.toJson(), {'region_id': 0, 'allow_low_sec': false});
    });
    test('allow_low_sec true', () {
      expect(const OreRankingRequest(regionId: 10000002, allowLowSec: true).toJson(),
          {'region_id': 10000002, 'allow_low_sec': true});
    });
```
Remove the old low/null `sec_band` serialization tests. Update any `OreRankingResponse.fromJson` test that referenced `secBand`.

- [ ] **Step 3: Replace the sec-band selector in the screen**

In `mining_screen.dart`, the `_InputForm` has a `RadioGroup<String>` for `_secBand` (high/low/null). Replace with a single switch/checkbox backed by `bool _allowLowSec = false`, labelled "Nur High-Sec" / "High + Low-Sec". The calculate handler builds `OreRankingRequest(regionId: 0, allowLowSec: _allowLowSec)`. Remove the `_SecBandTile` usages tied to low/null.

- [ ] **Step 4: analyze + test**

Run: `cd app && flutter analyze && flutter test`
Expected: no issues; all tests pass.

- [ ] **Step 5: Commit**

```bash
git add app/lib/api/mining_models.dart app/lib/features/mining/mining_screen.dart app/test/mining_models_test.dart
git commit -m "feat(app): high-only/high+low toggle + allow_low_sec request"
```

---

## Task 8: Verify, PR, release, deploy, APK

**Files:** none (process). **Flutter changed → APK rebuild + tablet install.**

- [ ] **Step 1: Full gates**

Run (backend): `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... && gofmt -l internal cmd pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Run (frontend): `cd ../frontend && npx vitest run && npm run build`
Run (flutter): `cd ../app && flutter analyze && flutter test`
Expected: all green; `0 issues`.

- [ ] **Step 2: PR**

```bash
git push -u origin feat/mining-system-ore-availability
gh pr create --base main --title "feat(mining): system-accurate ore availability (high+low) + real ore names" --body "<summary: deterministic per-system ore set from security+quarter (EVE-Uni rules); high-only/high+low toggle; real ore names from blueprints; -Grade variants dropped; sec_band → allow_low_sec (clean break); null/class-K/scanner = #161/#162/#163; spec+plan linked>"
```

- [ ] **Step 3: CI green → squash-merge → update main**

```bash
gh pr checks <PR#> --watch --interval 20
gh pr merge <PR#> --squash --delete-branch
git checkout main && git pull --ff-only
```

- [ ] **Step 4: Release v0.25.0**

Add a `## [Unreleased]` CHANGELOG entry (Added: system-accurate ore availability high+low, real ore names, high/low toggle; notes #161/#162/#163). Commit, then:
```bash
make release-check && make release VERSION=0.25.0
git add CHANGELOG.md && git commit -m "chore(release): v0.25.0"
git tag v0.25.0 && git push origin main && git push origin v0.25.0
```

- [ ] **Step 5: Watch deploy + prod smoke**

```bash
gh run watch <deploy-run-id> --interval 20 --exit-status
curl -s https://eveonline.sternrassler.de/api/v1/version   # → v0.25.0
curl -s -o /dev/null -w "%{http_code}\n" https://eveonline.sternrassler.de/mining  # → 200
```

- [ ] **Step 6: Rebuild + reinstall the Flutter APK (Flutter changed)**

```bash
cd app && CID="$(grep '^EVE_MOBILE_CLIENT_ID=' ../deployments/.env | cut -d= -f2- | tr -d '"'\'' \t')" \
  && flutter build apk --release --dart-define=API_BASE_URL=https://eveonline.sternrassler.de --dart-define=EVE_CLIENT_ID="$CID"
adb -s R5GL3433JKE install -r build/app/outputs/flutter-apk/app-release.apk
```
Expected: `Success`. Do not print `$CID`.

---

## Notes for the implementer

- **No silent fallbacks:** current location is mandatory (error if absent). Unknown quarter → quarter-independent ores only (Veldspar/Scordite/Pyroxeres), never a wrong full set. Low-sec system with high-only, or null-sec → empty set + `not_available_reason`.
- **Display security:** thresholds use EVE's displayed (1-decimal, half-away-from-zero) security; Jita 0.946 → 0.9 (so Pyroxeres is included for Jita).
- **Clean break:** `sec_band` is gone everywhere (backend, web, flutter). `ore_secband.go` is deleted. Null-sec / Class-K / exact-belt are out of scope (#161/#162/#163).
- **Naming is data-driven:** real names come from CCP's compression blueprints, not a hand-typed table; IV/0-Grade ores (no blueprint) are filtered as un-belt-able.
