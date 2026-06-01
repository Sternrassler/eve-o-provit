# Mining ISK/h: Hull-Boni + Erz-Crystals — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the mining ranking's m³/h (and thus ISK/h) reflect the actual ship — applying the hull's ore-mining-yield bonus (data-driven from SDE dogma) and a best-case T2 ore crystal per ore — and mark any row we cannot fully resolve as an estimate (never a silent 1.0).

**Architecture:** Two new pure functions in `pkg/evedb/mining` (hull multiplier, crystal multiplier) read the SDE. The mining service multiplies the existing base m³/h by `hullMul` (hull-wide) and a per-ore `crystalMul`, and sets `is_estimate`/`estimate_reason` when either cannot be resolved. Web + Flutter replace the blanket "lower bound" note with a per-row estimate marker.

**Tech Stack:** Go 1.24 (backend), SQLite SDE (`typeDogma`/`dogmaEffects`/`dogmaAttributes`), Next.js/TS (web), Flutter/Dart (tablet).

**Verified SDE facts (used below, do not re-derive):**
- Ore-mining yield attribute = `77` (`miningAmount`).
- Recognised hull ship-bonus effects on attr 77 (by `dogmaEffects.displayName`) and their **scaling skill** (the modifier's `skillTypeID` is a constant `3386` and must be ignored):
  - `miningBargeBonusOreMiningYield` → Mining Barge skill `17940`
  - `exhumersBonusOreMiningYield` → Exhumers skill `22551`
  - `miningFrigateBonusOreMiningYield` → Mining Frigate skill `32918`
- Ship bonus is **linear per level**, applied **multiplicatively** across bonuses: `mult = Π (1 + hullAttrValue/100 × skillLevel)`. Hulk (`22544`) has `miningBargeBonusOreMiningYield` value 3 and `exhumersBonusOreMiningYield` value 6 → at Mining Barge V + Exhumers V: `(1+0.03×5)(1+0.06×5) = 1.15×1.30 = 1.495`.
- Crystal-capable miner ⟺ a fitted module's charge-group attribute (`604`/`609`/`610`/`611`) equals `482` (Mining Crystal group). Modulated Strip Miner II = `17912` (capable); Strip Miner I = `17482` (not).
- A T2 ore crystal is `"<oreGroupName> Mining Crystal II"`; its yield multiplier is attribute `782`. Veldspar (group `462`) → `Veldspar Mining Crystal II` (`18618`), attr 782 = `1.75`. Mercoxit (group `468`) has **no** matching name → crystal not found → estimate.

**Test DB:** run backend SDE-touching tests with
`SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off`.

---

## File Structure

- `backend/pkg/evedb/mining/crystals.go` (Create) — crystal-capability + per-ore-group T2 crystal multiplier.
- `backend/pkg/evedb/mining/crystals_test.go` (Create).
- `backend/pkg/evedb/mining/hull_bonus.go` (Create) — `HullMiningYieldMultiplier`.
- `backend/pkg/evedb/mining/hull_bonus_test.go` (Create).
- `backend/internal/services/skills_service.go` (Modify) — add `SkillLevels` to `MiningReprocessingSkills` + populate.
- `backend/internal/models/mining.go` (Modify) — 4 new `OreRankRow` fields.
- `backend/internal/services/mining_service.go` (Modify) — extend `MiningLocationProvider`; apply hull+crystal per ore; estimate markers.
- `backend/internal/services/mining_service_test.go` (Modify) — fakes + assertions.
- `frontend/src/types/trading.ts` + `frontend/src/components/trading/OreRankingTable.tsx` + `frontend/tests/components/OreRankingTable.test.tsx` (Modify).
- `app/lib/api/mining_models.dart` + `app/lib/features/mining/mining_screen.dart` + `app/test/mining_models_test.dart` + `app/test/mining_screen_layout_test.dart` (Modify).

---

## Task 1: Crystal multiplier (`pkg/evedb/mining/crystals.go`)

**Files:**
- Create: `backend/pkg/evedb/mining/crystals.go`
- Test: `backend/pkg/evedb/mining/crystals_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/pkg/evedb/mining/crystals_test.go
package mining

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

func TestCrystalCapable(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	// Modulated Strip Miner II (17912) loads Mining Crystals → capable.
	ok, err := CrystalCapable(db, []int64{17912})
	if err != nil {
		t.Fatalf("CrystalCapable err: %v", err)
	}
	if !ok {
		t.Error("Modulated Strip Miner II must be crystal-capable")
	}

	// Strip Miner I (17482) has no crystal charge group → not capable.
	ok, err = CrystalCapable(db, []int64{17482})
	if err != nil {
		t.Fatalf("CrystalCapable err: %v", err)
	}
	if ok {
		t.Error("Strip Miner I must NOT be crystal-capable")
	}
}

func TestOreCrystalMultiplierT2(t *testing.T) {
	db := testutil.OpenTestDB(t)
	defer func() { _ = db.Close() }()

	// Veldspar group (462) → Veldspar Mining Crystal II, attr 782 = 1.75.
	mul, found, err := OreCrystalMultiplierT2(db, 462)
	if err != nil {
		t.Fatalf("OreCrystalMultiplierT2 err: %v", err)
	}
	if !found {
		t.Fatal("Veldspar group must have a T2 crystal")
	}
	if mul < 1.749 || mul > 1.751 {
		t.Errorf("Veldspar T2 crystal mult: got %v, want 1.75", mul)
	}

	// Mercoxit group (468) has no name-matching crystal → not found (no silent 1.0).
	_, found, err = OreCrystalMultiplierT2(db, 468)
	if err != nil {
		t.Fatalf("OreCrystalMultiplierT2 err: %v", err)
	}
	if found {
		t.Error("Mercoxit group must report no matching T2 crystal (found=false)")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run 'TestCrystal|TestOreCrystal' -v`
Expected: FAIL — `undefined: CrystalCapable`, `undefined: OreCrystalMultiplierT2`.

- [ ] **Step 3: Write the implementation**

```go
// backend/pkg/evedb/mining/crystals.go
package mining

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// miningCrystalGroupID is the SDE market/charge group for ore Mining Crystals.
const miningCrystalGroupID = 482

// chargeGroupAttrIDs are the dogma attributes that declare which charge groups a
// module accepts (chargeGroup1..4). A mining module is crystal-capable iff one of
// these equals miningCrystalGroupID.
var chargeGroupAttrIDs = []int64{604, 609, 610, 611}

// CrystalCapable reports whether any of the given module type ids can load ore
// Mining Crystals (i.e. a Modulated Strip Miner / Deep Core laser). Strip Miner I
// and basic Miners cannot — for those crystals genuinely do not apply.
func CrystalCapable(db *sql.DB, moduleTypeIDs []int64) (bool, error) {
	for _, tid := range moduleTypeIDs {
		attrs, err := typeAttrs(db, tid)
		if err != nil {
			return false, err
		}
		for _, aid := range chargeGroupAttrIDs {
			if int64(attrs[aid]) == miningCrystalGroupID {
				return true, nil
			}
		}
	}
	return false, nil
}

// OreCrystalMultiplierT2 returns the best-case (T2) ore crystal yield multiplier
// for the ore group, read live from the SDE. The crystal is found by name
// ("<oreGroupName> Mining Crystal II"); its yield is attribute 782. found=false
// when the group has no matching crystal (e.g. Mercoxit under this naming) — the
// caller must then mark the row as an estimate, never substitute 1.0.
func OreCrystalMultiplierT2(db *sql.DB, oreGroupID int64) (mult float64, found bool, err error) {
	var groupName string
	err = db.QueryRow(
		`SELECT json_extract(name,'$.en') FROM groups WHERE _key = ?`, oreGroupID,
	).Scan(&groupName)
	if err == sql.ErrNoRows || groupName == "" {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("ore group %d name: %w", oreGroupID, err)
	}

	crystalName := groupName + " Mining Crystal II"
	var crystalTypeID int64
	err = db.QueryRow(
		`SELECT _key FROM types WHERE json_extract(name,'$.en') = ?`, crystalName,
	).Scan(&crystalTypeID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("crystal %q: %w", crystalName, err)
	}

	attrs, err := typeAttrs(db, crystalTypeID)
	if err != nil {
		return 0, false, err
	}
	const attrAsteroidYieldMultiplier = 782
	v, ok := attrs[attrAsteroidYieldMultiplier]
	if !ok || v <= 0 {
		return 0, false, nil
	}
	return v, true, nil
}

// typeAttrs returns the typeDogma dogmaAttributes of a type as attributeID→value.
// Empty map (not an error) when the type has no dogma row.
func typeAttrs(db *sql.DB, typeID int64) (map[int64]float64, error) {
	var attrsJSON sql.NullString
	err := db.QueryRow(`SELECT dogmaAttributes FROM typeDogma WHERE _key = ?`, typeID).Scan(&attrsJSON)
	if err == sql.ErrNoRows {
		return map[int64]float64{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("type attrs %d: %w", typeID, err)
	}
	if !attrsJSON.Valid || attrsJSON.String == "" {
		return map[int64]float64{}, nil
	}
	var list []struct {
		AttributeID int64   `json:"attributeID"`
		Value       float64 `json:"value"`
	}
	if err := json.Unmarshal([]byte(attrsJSON.String), &list); err != nil {
		return nil, fmt.Errorf("parse attrs %d: %w", typeID, err)
	}
	m := make(map[int64]float64, len(list))
	for _, a := range list {
		m[a.AttributeID] = a.Value
	}
	return m, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run 'TestCrystal|TestOreCrystal' -v`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/evedb/mining/crystals.go backend/pkg/evedb/mining/crystals_test.go
git commit -m "feat(mining): crystal-capability + best-case T2 ore crystal multiplier"
```

---

## Task 2: Hull mining-yield multiplier (`pkg/evedb/mining/hull_bonus.go`)

**Files:**
- Create: `backend/pkg/evedb/mining/hull_bonus.go`
- Test: `backend/pkg/evedb/mining/hull_bonus_test.go`

- [ ] **Step 1: Write the failing test**

```go
// backend/pkg/evedb/mining/hull_bonus_test.go
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
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run TestHull -v`
Expected: FAIL — `undefined: HullMiningYieldMultiplier`.

- [ ] **Step 3: Write the implementation**

```go
// backend/pkg/evedb/mining/hull_bonus.go
package mining

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// recognisedHullYieldEffects maps a ship-bonus effect (by dogmaEffects.displayName)
// that modifies ore-mining yield (attr 77) to the skill whose level scales it.
// The modifier's own skillTypeID is a constant (3386) in the SDE and must be ignored.
// A hull whose attr-77 ship bonus is NOT in this set is reported unresolved, so the
// caller marks the row as an estimate instead of computing a partial (wrong) value.
var recognisedHullYieldEffects = map[string]int64{
	"miningBargeBonusOreMiningYield":  17940, // Mining Barge
	"exhumersBonusOreMiningYield":     22551, // Exhumers
	"miningFrigateBonusOreMiningYield": 32918, // Mining Frigate
}

const attrMiningAmountModified = 77

// HullMiningYieldMultiplier returns the multiplicative ore-mining-yield bonus the
// hull grants (role + per-skill-level), read from SDE dogma. resolved=false when the
// hull has an attr-77 ship modifier whose effect is not recognised — the caller must
// then mark the row as an estimate, never silently use 1.0. A hull with no attr-77
// ship bonus returns (1.0, true): genuinely no bonus, not a fallback.
func HullMiningYieldMultiplier(db *sql.DB, hullTypeID int64, skillLevels map[int64]int) (mult float64, resolved bool, err error) {
	hullAttrs, err := typeAttrs(db, hullTypeID)
	if err != nil {
		return 0, false, err
	}
	effectIDs, err := typeEffectIDs(db, hullTypeID)
	if err != nil {
		return 0, false, err
	}

	mult = 1.0
	resolved = true
	for _, eid := range effectIDs {
		displayName, mods, err := effectYieldModifiers(db, eid)
		if err != nil {
			return 0, false, err
		}
		if len(mods) == 0 {
			continue // this effect doesn't touch attr-77 on the ship
		}
		skillID, ok := recognisedHullYieldEffects[displayName]
		if !ok {
			resolved = false // unrecognised mining bonus on this hull
			continue
		}
		for _, m := range mods {
			value := hullAttrs[m.ModifyingAttributeID] // per-level % on this hull
			level := skillLevels[skillID]
			mult *= 1.0 + (value/100.0)*float64(level)
		}
	}
	return mult, resolved, nil
}

// typeEffectIDs returns the dogmaEffects ids referenced by a type's typeDogma.
func typeEffectIDs(db *sql.DB, typeID int64) ([]int64, error) {
	var effectsJSON sql.NullString
	err := db.QueryRow(`SELECT dogmaEffects FROM typeDogma WHERE _key = ?`, typeID).Scan(&effectsJSON)
	if err == sql.ErrNoRows || !effectsJSON.Valid || effectsJSON.String == "" {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("type effects %d: %w", typeID, err)
	}
	var refs []struct {
		EffectID int64 `json:"effectID"`
	}
	if err := json.Unmarshal([]byte(effectsJSON.String), &refs); err != nil {
		return nil, fmt.Errorf("parse effects %d: %w", typeID, err)
	}
	out := make([]int64, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.EffectID)
	}
	return out, nil
}

type yieldModifier struct {
	ModifyingAttributeID int64 `json:"modifyingAttributeID"`
}

// effectYieldModifiers returns an effect's displayName and the subset of its
// modifierInfo that modifies ore-mining yield (attr 77) on the ship.
func effectYieldModifiers(db *sql.DB, effectID int64) (displayName string, mods []yieldModifier, err error) {
	var name sql.NullString
	var modifierJSON sql.NullString
	err = db.QueryRow(
		`SELECT displayName, modifierInfo FROM dogmaEffects WHERE _key = ?`, effectID,
	).Scan(&name, &modifierJSON)
	if err == sql.ErrNoRows {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("effect %d: %w", effectID, err)
	}
	if !modifierJSON.Valid || modifierJSON.String == "" {
		return name.String, nil, nil
	}
	var all []struct {
		Domain               string `json:"domain"`
		ModifiedAttributeID  int64  `json:"modifiedAttributeID"`
		ModifyingAttributeID int64  `json:"modifyingAttributeID"`
	}
	if err := json.Unmarshal([]byte(modifierJSON.String), &all); err != nil {
		return "", nil, fmt.Errorf("parse modifierInfo %d: %w", effectID, err)
	}
	for _, m := range all {
		if m.ModifiedAttributeID == attrMiningAmountModified && m.Domain == "shipID" {
			mods = append(mods, yieldModifier{ModifyingAttributeID: m.ModifyingAttributeID})
		}
	}
	return name.String, mods, nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/ -run TestHull -v`
Expected: PASS. If the Venture sub-case fails (resolved unexpectedly true), the unrecognised role-bonus assumption is wrong — inspect with
`sqlite3 <sde> "SELECT displayName,modifierInfo FROM dogmaEffects WHERE _key IN (SELECT json_extract(value,'$.effectID') FROM typeDogma, json_each(dogmaEffects) WHERE typeDogma._key=32880);"` and adjust the recognised set; do **not** weaken the resolved flag to pass.

- [ ] **Step 5: Run the whole mining package + gofmt/vet**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./pkg/evedb/mining/... && gofmt -l backend/pkg/evedb/mining/ && GOWORK=off go vet ./pkg/evedb/mining/...`
Expected: ok, no gofmt output, no vet output.

- [ ] **Step 6: Commit**

```bash
git add backend/pkg/evedb/mining/hull_bonus.go backend/pkg/evedb/mining/hull_bonus_test.go
git commit -m "feat(mining): data-driven hull ore-mining-yield multiplier"
```

---

## Task 3: Expose all skill levels (`skills_service.go`)

**Files:**
- Modify: `backend/internal/services/skills_service.go`

- [ ] **Step 1: Add the field to `MiningReprocessingSkills`**

Find the struct (it has `Reprocessing/ReprocessingEfficiency/Mining/Astrogeology/Accounting/OreProcessing`) and add:

```go
	OreProcessing          map[int64]int // ore groupID → processing skill level
	SkillLevels            map[int64]int // skillTypeID → active level (for hull bonuses)
```

- [ ] **Step 2: Populate it in `GetMiningReprocessingSkills`**

In the function that builds the `MiningReprocessingSkills` (the loop over the character's skills that already sets `Mining`, `Astrogeology`, etc.), initialise and fill the map. Locate the skill-iteration loop and, inside it, capture every skill level by id. Concretely, where the loop variable is the per-skill record (the same one whose `.ActiveSkillLevel` feeds `skills.Mining`), add:

```go
	skills.SkillLevels = make(map[int64]int)
	// ... inside the existing `for _, skill := range <skillList>` loop body:
	skills.SkillLevels[int64(skill.SkillID)] = skill.ActiveSkillLevel
```

Use the field names already present on the skill record in this file (e.g. `skill.SkillID`, `skill.ActiveSkillLevel` — confirm against the existing assignments like `skills.Accounting = skill.ActiveSkillLevel`). Initialise the map once before the loop.

- [ ] **Step 3: Build + the existing skills tests still pass**

Run: `GOWORK=off go build ./internal/services/... && GOWORK=off go test ./internal/services/ -run Skill -v`
Expected: builds; existing skills tests pass.

- [ ] **Step 4: Commit**

```bash
git add backend/internal/services/skills_service.go
git commit -m "feat(skills): expose per-skill levels for hull bonus lookup"
```

---

## Task 4: Response model fields (`models/mining.go`)

**Files:**
- Modify: `backend/internal/models/mining.go`

- [ ] **Step 1: Add fields to `OreRankRow`**

Append inside the `OreRankRow` struct (after `Materials`):

```go
	// Yield accuracy (hull role/skill bonus + best-case ore crystal):
	HullYieldMultiplier float64 `json:"hull_yield_multiplier,omitempty"` // hull-wide (same on every row); 1.0 = none
	CrystalMultiplier   float64 `json:"crystal_multiplier,omitempty"`   // per ore; 1.0 = no crystals used
	IsEstimate          bool    `json:"is_estimate,omitempty"`          // hull bonus / crystal not fully resolved
	EstimateReason      string  `json:"estimate_reason,omitempty"`      // short reason, only when IsEstimate
```

- [ ] **Step 2: Build**

Run: `GOWORK=off go build ./internal/models/...`
Expected: builds.

- [ ] **Step 3: Commit**

```bash
git add backend/internal/models/mining.go
git commit -m "feat(mining): OreRankRow yield-multiplier + estimate fields"
```

---

## Task 5: Wire hull + crystal into the service (`mining_service.go`)

**Files:**
- Modify: `backend/internal/services/mining_service.go`

- [ ] **Step 1: Extend `MiningLocationProvider` to expose the active ship type**

The constructor already injects `characterHelper` as the `location` provider, and `*CharacterHelper` already implements `GetActiveShipTypeID(ctx, characterID, accessToken) (int, error)`. Add that method to the interface so the service can call it without a new constructor param:

```go
type MiningLocationProvider interface {
	GetCharacterLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error)
	GetActiveShipTypeID(ctx context.Context, characterID int, accessToken string) (int, error)
}
```

- [ ] **Step 2: After the base `m3h` is computed, resolve hull multiplier + crystal capability once**

Locate the block that ends with:

```go
	if m3h == 0 {
		resp.NoMiningSetup = true
	}
```

Immediately after it, add:

```go
	// Hull ore-mining-yield bonus (role + per-skill-level), applied to every ore.
	hullMul := 1.0
	hullResolved := true
	hullTypeID, shipErr := s.location.GetActiveShipTypeID(ctx, characterID, accessToken)
	if shipErr != nil {
		// Fail-loud: current ship unknown → mark rows as estimate, do not assume "no bonus".
		hullResolved = false
		if s.logger != nil {
			s.logger.Warn("ore ranking: active ship unavailable for hull bonus", "error", shipErr)
		}
	} else {
		hm, resolved, err := mining.HullMiningYieldMultiplier(s.sdeDB, int64(hullTypeID), skills.SkillLevels)
		if err != nil {
			return nil, err
		}
		hullMul = hm
		hullResolved = resolved
	}

	// Best-case T2 ore crystals apply only if a crystal-capable miner is fitted.
	crystalCapable, err := mining.CrystalCapable(s.sdeDB, moduleIDs)
	if err != nil {
		return nil, err
	}
```

Note: `skills.SkillLevels` may be nil when the skills call degraded to zero skills (see the earlier `skills = &MiningReprocessingSkills{...}` fallback). `HullMiningYieldMultiplier` reads it with map indexing, which is nil-safe (returns 0 levels) — no extra guard needed.

- [ ] **Step 3: Compute per-ore crystal multiplier + effective m³/h inside the ore loop**

In the per-ore loop (`for i := range allOres { ... }`), after `net := reprocessing.NetYield(...)` is computed and before building `row`, add:

```go
		// Per-ore best-case crystal multiplier (1.0 when no crystal-capable miner).
		crystalMul := 1.0
		oreIsEstimate := !hullResolved
		oreEstimateReason := ""
		if !hullResolved {
			oreEstimateReason = "Schiffs-Bonus nicht auflösbar"
		}
		if crystalCapable {
			cm, found, cErr := mining.OreCrystalMultiplierT2(s.sdeDB, o.GroupID)
			if cErr != nil {
				return nil, cErr
			}
			if found {
				crystalMul = cm
			} else {
				// Crystal-capable setup but no crystal for this ore → estimate, never silent 1.0.
				oreIsEstimate = true
				if oreEstimateReason == "" {
					oreEstimateReason = "Kein Crystal für dieses Erz"
				}
			}
		}
		oreM3h := m3h * hullMul * crystalMul
```

- [ ] **Step 4: Use `oreM3h` + set the new fields when building `row`**

Replace the `row := models.OreRankRow{...}` literal's `MiningM3PerHour`, `RawISKPerHour`, `RefineISKPerHour`, `DeltaISKPerHour` to use `oreM3h`, and add the four new fields:

```go
		row := models.OreRankRow{
			OreTypeID:           o.TypeID,
			OreName:             o.Name,
			MiningM3PerHour:     oreM3h,
			RawNetPerM3:         cmp.RawNetPerM3,
			RefineNetPerM3:      cmp.RefineNetPerM3,
			Best:                cmp.Best,
			RawISKPerHour:       oreM3h * cmp.RawNetPerM3,
			RefineISKPerHour:    oreM3h * cmp.RefineNetPerM3,
			DeltaISKPerHour:     oreM3h * cmp.DeltaPerM3,
			BestStationID:       bestStationID,
			BestStationTax:      stationTax,
			BestStationName:     bestStationName,
			BestStationSystem:   bestStationSystem,
			Materials:           breakdown,
			HullYieldMultiplier: hullMul,
			CrystalMultiplier:   crystalMul,
			IsEstimate:          oreIsEstimate,
			EstimateReason:      oreEstimateReason,
		}
```

- [ ] **Step 5: Fix the sort to rank by per-ore ISK/h**

The sort closure currently gates on `m3h > 0` and compares `RawISKPerHour`/`RefineISKPerHour`, which now already embed `oreM3h` — so it remains correct. Leave the `m3h > 0` gate as-is (base mining-rate presence). No change required; re-read to confirm it references the row fields, not a per-row `oreM3h` local.

- [ ] **Step 6: Build + vet**

Run: `GOWORK=off go build ./internal/services/... ./cmd/... && GOWORK=off go vet ./internal/services/...`
Expected: builds. If `cmd/api/container.go` fails to compile because a `*CharacterHelper` no longer satisfies `MiningLocationProvider`, confirm `GetActiveShipTypeID` exists on `*CharacterHelper` (it does, `character_helper.go`); no container change is expected.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/services/mining_service.go
git commit -m "feat(mining): apply hull bonus + ore crystals to ISK/h, mark estimates"
```

---

## Task 6: Service tests (`mining_service_test.go`)

**Files:**
- Modify: `backend/internal/services/mining_service_test.go`

- [ ] **Step 1: Teach the location fake the new method + add a hull/skill fake**

The existing tests pass `nil` for the `location` provider (6th arg). Add a fake implementing the extended interface, returning a Hulk and the skills that resolve it. Add near the other fakes:

```go
type fakeMiningLocation struct {
	shipTypeID int
	shipErr    error
}

func (f fakeMiningLocation) GetCharacterLocation(_ context.Context, _ int, _ string) (*CharacterLocation, error) {
	return &CharacterLocation{}, nil
}
func (f fakeMiningLocation) GetActiveShipTypeID(_ context.Context, _ int, _ string) (int, error) {
	return f.shipTypeID, f.shipErr
}
```

The skills fake (`fakeMiningSkillsProvider`) returns a `*MiningReprocessingSkills`; in the Veldspar test set `SkillLevels` so the Hulk resolves, e.g. add to the constructed skills:

```go
	skills: &MiningReprocessingSkills{
		OreProcessing: map[int64]int{},
		SkillLevels:   map[int64]int{17940: 5, 22551: 5}, // Mining Barge V, Exhumers V
	},
```

- [ ] **Step 2: Update both `NewMiningService(...)` calls to inject the location fake**

Replace the `nil` 6th argument with `fakeMiningLocation{shipTypeID: 22544}` (Hulk) in `TestMiningService_OreRanking_Veldspar` and `fakeMiningLocation{shipTypeID: 17482}` (Strip Miner I hull-less case is fine; any non-mining type) in `TestMiningService_OreRanking_NoMiningSetup`. Keep the existing 7th (`region`) `nil` argument as-is.

- [ ] **Step 3: Add assertions to `TestMiningService_OreRanking_Veldspar`**

The fitted module is `stripMinerITypeID = 17482` (Strip Miner I — not crystal-capable), so `crystalMul` stays 1.0; the Hulk hull applies `hullMul = 1.495`. Assert:

```go
	if row.HullYieldMultiplier < 1.4949 || row.HullYieldMultiplier > 1.4951 {
		t.Errorf("HullYieldMultiplier: got %v, want 1.495", row.HullYieldMultiplier)
	}
	if row.CrystalMultiplier != 1.0 {
		t.Errorf("CrystalMultiplier: got %v, want 1.0 (Strip Miner I has no crystals)", row.CrystalMultiplier)
	}
	if row.IsEstimate {
		t.Errorf("row should be exact (Hulk resolved, no crystal needed): %+v", row)
	}
	// isk/h now includes the hull bonus: m3h × hullMul × netPerM3.
	wantRawISK := row.MiningM3PerHour * row.RawNetPerM3
	if !approxEq(row.RawISKPerHour, wantRawISK) {
		t.Errorf("RawISKPerHour: got %v, want %v", row.RawISKPerHour, wantRawISK)
	}
	// MiningM3PerHour reflects base × 1.495.
	if !approxEq(row.MiningM3PerHour, wantM3H*1.495) {
		t.Errorf("MiningM3PerHour with hull bonus: got %v, want %v", row.MiningM3PerHour, wantM3H*1.495)
	}
```

Where `wantM3H` is the existing base-rate constant already asserted earlier in the test. Update the earlier `MiningM3PerHour == wantM3H` assertion to `wantM3H*1.495` (or remove the old one in favour of the new), and update the `RawISKPerHour`/`RefineISKPerHour` assertions that used the old `row.MiningM3PerHour` to keep using `row.MiningM3PerHour` (they already multiply by it, so they stay correct).

- [ ] **Step 4: Add an estimate test (unknown hull / crystal-capable + no crystal)**

```go
func TestMiningService_OreRanking_EstimateWhenShipUnknown(t *testing.T) {
	sdeDB := testutil.OpenTestDB(t)
	defer func() { _ = sdeDB.Close() }()

	market := &fakeMiningMarket{buyPriceByType: map[int]float64{veldsparTypeID: 15.0, tritaniumTypeID: 5.0}}
	skills := &fakeMiningSkillsProvider{
		skills:    &MiningReprocessingSkills{OreProcessing: map[int64]int{}, SkillLevels: map[int64]int{}},
		standings: map[int64]float64{},
	}
	fitting := &fakeMiningModules{ids: []int64{stripMinerITypeID}}
	stations := &fakeStations{stations: []database.ReprocessStation{{StationID: 60000001, OwnerCorpID: 1000035, BaseRate: 0.50, BaseTake: 0.05}}}

	// Active-ship lookup fails → hull bonus unresolved → every row is an estimate.
	loc := fakeMiningLocation{shipErr: fmt.Errorf("esi down")}
	svc := NewMiningService(sdeDB, stations, market, skills, fitting, loc, nil, fakeMiningNames{}, logger.NewNoop())

	resp, err := svc.OreRanking(context.Background(), 42, "token", models.OreRankingRequest{RegionID: 10000002, SecBand: "high"})
	if err != nil {
		t.Fatalf("OreRanking error: %v", err)
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
```

- [ ] **Step 5: Run the mining service tests**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestMiningService -v`
Expected: PASS (Veldspar with hull bonus, NoMiningSetup, Estimate).

- [ ] **Step 6: Full backend gate for touched packages**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/evedb/mining/... && gofmt -l backend/internal backend/pkg/evedb/mining && GOWORK=off golangci-lint run ./internal/... ./pkg/evedb/mining/... ./cmd/...`
Expected: ok; no gofmt output; `0 issues`.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/services/mining_service_test.go
git commit -m "test(mining): hull bonus + crystal + estimate-marker coverage"
```

---

## Task 7: Web — types + per-row estimate marker

**Files:**
- Modify: `frontend/src/types/trading.ts`
- Modify: `frontend/src/components/trading/OreRankingTable.tsx`
- Modify: `frontend/tests/components/OreRankingTable.test.tsx`

- [ ] **Step 1: Add the fields to `OreRankRow` (after `materials?`)**

```ts
  hull_yield_multiplier?: number;
  crystal_multiplier?: number;
  is_estimate?: boolean;
  estimate_reason?: string;
```

- [ ] **Step 2: Write the failing test (estimate marker rendered; exact row has none)**

Add to `OreRankingTable.test.tsx`:

```ts
  it("marks an estimate row and omits the marker on an exact row", () => {
    const mixed: OreRankRow[] = [
      makeRow({ ore_type_id: 1230, ore_name: "Veldspar", is_estimate: false, hull_yield_multiplier: 1.495 }),
      makeRow({ ore_type_id: 1228, ore_name: "Scordite", is_estimate: true, estimate_reason: "Kein Crystal für dieses Erz" }),
    ];
    render(<OreRankingTable rows={mixed} />);

    const est = screen.getAllByTestId("ore-ranking-row").find((r) => r.getAttribute("data-ore-type-id") === "1228")!;
    expect(within(est).getByTestId("ore-estimate-badge")).toBeInTheDocument();

    const exact = screen.getAllByTestId("ore-ranking-row").find((r) => r.getAttribute("data-ore-type-id") === "1230")!;
    expect(within(exact).queryByTestId("ore-estimate-badge")).not.toBeInTheDocument();
  });
```

- [ ] **Step 3: Run it to verify it fails**

Run: `npx vitest run tests/components/OreRankingTable.test.tsx`
Expected: FAIL — no `ore-estimate-badge`.

- [ ] **Step 4: Render the marker in `OreRankingTable.tsx`**

In the summary `<tr>` (the `OreRankRow` component), add a small badge next to the ore name, shown only when `row.is_estimate`:

```tsx
            {row.ore_name}
            {row.is_estimate && (
              <span
                data-testid="ore-estimate-badge"
                title={row.estimate_reason || "Schätzwert"}
                className="ml-2 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium text-amber-800 dark:bg-amber-900/30 dark:text-amber-200"
              >
                ≈ Schätzung
              </span>
            )}
```

Remove the blanket "skills-only Untergrenze" note if it lives in this component or the `/mining` page (`frontend/src/app/mining/page.tsx`); replace with a one-line legend: "Zeilen mit ≈ sind Schätzwerte (Schiffs-Bonus/Crystal nicht auflösbar)." Search: `grep -rn "Untergrenze" frontend/src`.

- [ ] **Step 5: Run tests + build + lint**

Run: `npx vitest run tests/components/OreRankingTable.test.tsx && npm run build && npx eslint src/components/trading/OreRankingTable.tsx src/types/trading.ts`
Expected: tests pass; build ok; eslint clean.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/types/trading.ts frontend/src/components/trading/OreRankingTable.tsx frontend/tests/components/OreRankingTable.test.tsx frontend/src/app/mining/page.tsx
git commit -m "feat(web): per-row estimate marker for mining yield accuracy"
```

---

## Task 8: Flutter — model fields + per-row estimate marker

**Files:**
- Modify: `app/lib/api/mining_models.dart`
- Modify: `app/lib/features/mining/mining_screen.dart`
- Modify: `app/test/mining_models_test.dart`
- Modify: `app/test/mining_screen_layout_test.dart`

- [ ] **Step 1: Add fields to `OreRankRow` (Dart) + parse them**

Add constructor params + fields:

```dart
    this.hullYieldMultiplier = 1.0,
    this.crystalMultiplier = 1.0,
    this.isEstimate = false,
    this.estimateReason,
```
```dart
  final double hullYieldMultiplier;
  final double crystalMultiplier;
  final bool isEstimate;
  final String? estimateReason;
```

In `fromJson`:

```dart
      hullYieldMultiplier: (json['hull_yield_multiplier'] as num?)?.toDouble() ?? 1.0,
      crystalMultiplier: (json['crystal_multiplier'] as num?)?.toDouble() ?? 1.0,
      isEstimate: json['is_estimate'] as bool? ?? false,
      estimateReason: json['estimate_reason'] as String?,
```

- [ ] **Step 2: Write the failing model test**

Add to `mining_models_test.dart` (inside `OreRankRow.fromJson` group):

```dart
    test('parses estimate + multiplier fields; defaults to exact', () {
      final est = OreRankRow.fromJson(const {
        'ore_type_id': 1228, 'ore_name': 'Scordite', 'best': 'raw',
        'is_estimate': true, 'estimate_reason': 'Kein Crystal für dieses Erz',
        'hull_yield_multiplier': 1.495, 'crystal_multiplier': 1.0,
      });
      expect(est.isEstimate, isTrue);
      expect(est.estimateReason, 'Kein Crystal für dieses Erz');
      expect(est.hullYieldMultiplier, closeTo(1.495, 0.001));

      final exact = OreRankRow.fromJson(const {'ore_type_id': 1230, 'ore_name': 'Veldspar', 'best': 'refine'});
      expect(exact.isEstimate, isFalse);
      expect(exact.crystalMultiplier, 1.0);
    });
```

- [ ] **Step 3: Run model tests to verify fail→pass**

Run: `flutter test test/mining_models_test.dart`
Expected: FAIL before Step 1 applied, PASS after.

- [ ] **Step 4: Render the marker in `mining_screen.dart`**

In `_OreRankTile.build`, in the title `Row`, after the ore-name `Expanded`, insert a badge when `row.isEstimate`:

```dart
            if (row.isEstimate)
              Padding(
                key: ValueKey('mining-estimate-${row.oreTypeId}'),
                padding: const EdgeInsets.only(left: 6),
                child: Tooltip(
                  message: row.estimateReason ?? 'Schätzwert',
                  child: const Text('≈', style: TextStyle(color: Color(0xFFFFB300), fontWeight: FontWeight.w700)),
                ),
              ),
```

Update the footer note in `_ResultPane` (the `Text` starting "Hinweis: ISK/h ist eine skills-basierte Untergrenze …") to: "Hinweis: Zeilen mit ≈ sind Schätzwerte (Schiffs-Bonus oder Crystal nicht auflösbar)."

- [ ] **Step 5: Write the failing widget test**

Add to `mining_screen_layout_test.dart` a row with `isEstimate: true` in `_detailResponse()` (e.g. add `isEstimate: true, estimateReason: 'x'` to the Pyroxeres row) and a test:

```dart
  testWidgets('Estimate rows show the ≈ marker', (tester) async {
    await _pumpScreen(tester, 1280, notifier: _DetailNotifier.new);
    expect(find.byKey(const ValueKey('mining-estimate-1224')), findsOneWidget);
    // An exact row carries no marker.
    expect(find.byKey(const ValueKey('mining-estimate-1230')), findsNothing);
  });
```

- [ ] **Step 6: analyze + test**

Run: `flutter analyze lib/api/mining_models.dart lib/features/mining/mining_screen.dart test/mining_models_test.dart test/mining_screen_layout_test.dart && flutter test test/mining_models_test.dart test/mining_screen_layout_test.dart`
Expected: no issues; all tests pass.

- [ ] **Step 7: Commit**

```bash
git add app/lib/api/mining_models.dart app/lib/features/mining/mining_screen.dart app/test/mining_models_test.dart app/test/mining_screen_layout_test.dart
git commit -m "feat(app): per-row estimate marker for mining yield accuracy"
```

---

## Task 9: Full verification, PR, release, deploy, APK

**Files:** none (process).

- [ ] **Step 1: Full backend gate**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... && cd backend && gofmt -l internal cmd pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Expected: ok; no gofmt output; `0 issues`.

- [ ] **Step 2: Full frontend gate**

Run: `cd frontend && npx vitest run && npm run build`
Expected: all tests pass; build ok.

- [ ] **Step 3: Full Flutter gate**

Run: `cd app && flutter analyze && flutter test`
Expected: no issues; all tests pass.

- [ ] **Step 4: Push branch + open PR to `main`**

```bash
git push -u origin feat/mining-hull-bonus-crystals
gh pr create --base main --title "feat(mining): hull bonus + ore crystals in ISK/h" --body "<summary: data-driven hull yield bonus + best-case T2 crystals; per-row estimate marker (no silent 1.0); verdict/ranking unaffected by hull bonus; spec + plan linked>"
```

- [ ] **Step 5: Wait for CI green, then squash-merge + delete branch**

```bash
gh pr checks <PR#> --watch --interval 20
gh pr merge <PR#> --squash --delete-branch
git checkout main && git pull --ff-only
```

- [ ] **Step 6: Release v0.22.0**

Add a `## [Unreleased]` entry to `CHANGELOG.md` (Added: hull bonus + crystals in mining ISK/h; per-row estimate marker), commit, then:

```bash
make release-check
make release VERSION=0.22.0
git add CHANGELOG.md && git commit -m "chore(release): v0.22.0"
git tag v0.22.0 && git push origin main && git push origin v0.22.0
```

- [ ] **Step 7: Watch deploy + prod smoke**

```bash
gh run watch <deploy-run-id> --interval 20 --exit-status
curl -s https://eveonline.sternrassler.de/api/v1/version   # → v0.22.0
curl -s -o /dev/null -w "%{http_code}\n" https://eveonline.sternrassler.de/mining  # → 200
```

- [ ] **Step 8: Rebuild + reinstall the Flutter APK on the tablet (Flutter changed)**

Build the release APK against prod with the mobile client id sourced from `deployments/.env` **without echoing it**, then install on device `R5GL3433JKE`:

```bash
cd app && CID="$(grep '^EVE_MOBILE_CLIENT_ID=' ../deployments/.env | cut -d= -f2- | tr -d '"'\'' \t')" \
  && flutter build apk --release --dart-define=API_BASE_URL=https://eveonline.sternrassler.de --dart-define=EVE_CLIENT_ID="$CID"
adb -s R5GL3433JKE install -r build/app/outputs/flutter-apk/app-release.apk
```

Expected: `Success`. Do not print `$CID`.

---

## Notes for the implementer

- **No silent fallbacks (hard rule):** `hullMul`/`crystalMul` = 1.0 is allowed **only** when it is genuinely correct (no hull bonus / no crystal-capable miner). Whenever a value cannot be resolved (ship unknown, unrecognised hull bonus, crystal-capable but no crystal for an ore), set `IsEstimate=true` with a reason — never quietly use 1.0.
- **Verdict invariance:** the hull multiplier scales every ore equally and must not change `Best`. The crystal multiplier is per-ore and may reorder rows — that is expected and correct.
- **Recognised hull-bonus set is intentionally bounded:** barges + exhumers + mining frigates by `Mining Barge/Exhumers/Mining Frigate` skill. Hulls with other mining-yield effects (e.g. Venture's role bonus, Orca/Rorqual) resolve to `IsEstimate` rather than a fabricated number. Extending the set later is a one-line addition to `recognisedHullYieldEffects` plus a characterization test.
