# Instance-accurate ship dropdown — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** List each owned ship instance individually (custom name + its own fitting's effective cargo) in the ROI dropdown, and feed the selected instance's exact cargo into the optimizer — fixing the type-level arbitrary-instance cargo bug (Iteron shown 4.9k vs in-game 6.09k).

**Architecture:** Backend enriches each `CharacterAssetShip` per item_id (its real fitted modules) and attaches the custom ship name from ESI `/assets/names`. Frontend stops deduping by type_id (dedupe by item_id), labels each option `name (cargo)`, and sends the selected instance's effective cargo as the existing `cargo_capacity` override so the optimizer never recomputes per-type.

**Tech Stack:** Go 1.24 / Fiber backend, SQLite SDE via `pkg/evedb`, Next.js 16 / React / TypeScript frontend, vitest.

Reference spec: `docs/superpowers/specs/2026-05-31-instance-accurate-ship-dropdown-design.md`.

**Verification env:** backend Go tests that touch SDE use `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db` and run with `GOWORK=off`. Commits: hooks need gitleaks on PATH — prefix git commits with `PATH="$HOME/.local/bin:$PATH"`.

---

## Task 1: Per-item fitting resolution (backend core)

Today `fetchFittingFromESI` picks the *first* singleton of a type. Extract the module-resolution + effective-cargo computation so it can run for a *specific* ship item_id, given the already-fetched assets.

**Files:**
- Modify: `backend/internal/services/fitting_service.go` (refactor `fetchFittingFromESI`; add `EffectiveCargoForShipItem`)
- Test: `backend/internal/services/fitting_effective_cargo_test.go` (new)

- [ ] **Step 1: Write the failing test** — two same-type ships, different modules → different effective cargo.

`backend/internal/services/fitting_effective_cargo_test.go`:
```go
package services

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

// Iteron Mark V (657), Gallente Industrial (3340) L1 → 6090 with no modules.
// A Reinforced Bulkhead (1300, dogma attr 149 < 1) reduces it below base.
func TestEffectiveCargoForShipItem_PerInstance(t *testing.T) {
	db := testutil.OpenTestDB(t) // SDE_DB_PATH
	s := &FittingService{sdeDB: db}
	ctx := context.Background()

	skills := &cargo.CharacterSkills{Skills: []struct {
		SkillID           int64 `json:"skill_id"`
		ActiveSkillLevel  int   `json:"active_skill_level"`
		TrainedSkillLevel int   `json:"trained_skill_level"`
	}{{SkillID: 3340, ActiveSkillLevel: 1, TrainedSkillLevel: 1}}}

	// Empty Iteron (item 111): no fitted modules.
	emptyAssets := []esiAsset{{ItemID: 111, TypeID: 657, IsSingleton: true}}
	empty, unavailEmpty := s.EffectiveCargoForShipItem(ctx, 657, 111, emptyAssets, skills)
	if unavailEmpty {
		t.Fatalf("empty ship should not be unavailable")
	}
	if empty < 6089 || empty > 6091 {
		t.Errorf("empty Iteron effective: want ~6090, got %.1f", empty)
	}

	// Bulkhead Iteron (item 222) with a Reinforced Bulkhead II (type 1306) in a low slot.
	bulkAssets := []esiAsset{
		{ItemID: 222, TypeID: 657, IsSingleton: true},
		{ItemID: 999, TypeID: 1306, LocationID: 222, LocationFlag: "LoSlot0"},
	}
	bulk, _ := s.EffectiveCargoForShipItem(ctx, 657, 222, bulkAssets, skills)
	if bulk >= empty {
		t.Errorf("bulkhead ship cargo should be below the empty ship: empty=%.1f bulk=%.1f", empty, bulk)
	}
}
```
> Verify the Reinforced Bulkhead type id (1306) and that it has dogma attr 149 in the SDE before finalizing; if the chosen module has no cargo modifier in the test DB, pick one that does (query: `SELECT _key,json_extract(name,'$.en') FROM types WHERE json_extract(name,'$.en') LIKE 'Reinforced Bulkhead%';` then confirm attr 149 via typeDogma). Adjust the type id in the test to a confirmed one.

- [ ] **Step 2: Run the test, verify it fails to compile** (`EffectiveCargoForShipItem` undefined).

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestEffectiveCargoForShipItem_PerInstance -v`
Expected: build error `s.EffectiveCargoForShipItem undefined`.

- [ ] **Step 3: Extract a helper and add the method.** In `fitting_service.go`, factor the module-collection loop in `fetchFittingFromESI` (the `for _, asset := range assets { if asset.LocationID == shipItemID && isFittedSlot(...) }` block) into:
```go
// fittedItemsForShip collects the cargo.FittedItem list for a specific ship item_id
// from an already-fetched asset list.
func (s *FittingService) fittedItemsForShip(ctx context.Context, assets []esiAsset, shipItemID int64) []cargo.FittedItem {
	items := make([]cargo.FittedItem, 0)
	for _, asset := range assets {
		if asset.LocationID == shipItemID && isFittedSlot(asset.LocationFlag) {
			items = append(items, cargo.FittedItem{TypeID: int64(asset.TypeID), Slot: asset.LocationFlag})
		}
	}
	return items
}

// EffectiveCargoForShipItem computes the effective cargo for ONE specific ship instance
// (by item_id) using that ship's own fitted modules. Returns (effective, unavailable);
// unavailable=true means the calc errored and the caller should fail loud.
func (s *FittingService) EffectiveCargoForShipItem(
	ctx context.Context, shipTypeID int, shipItemID int64, assets []esiAsset, charSkills *cargo.CharacterSkills,
) (float64, bool) {
	items := s.fittedItemsForShip(ctx, assets, shipItemID)
	caps, err := cargo.GetShipCapacitiesDeterministic(ctx, s.sdeDB, int64(shipTypeID), charSkills, items)
	if err != nil {
		s.logger.Warn("effective cargo per-item calc failed", "shipTypeID", shipTypeID, "itemID", shipItemID, "error", err)
		return 0, true
	}
	return caps.EffectiveCargoHold, false
}
```
Then rewrite the relevant part of `fetchFittingFromESI` to call `s.fittedItemsForShip(ctx, assets, shipItemID)` instead of the inline loop (keep existing behaviour for the single-fitting path). Note `s.logger` may be nil in the test's bare struct — guard with `if s.logger != nil` inside `EffectiveCargoForShipItem`, or set a noop logger in the test. Prefer the nil guard.

- [ ] **Step 4: Run the test, verify it passes.**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestEffectiveCargoForShipItem_PerInstance -v`
Expected: PASS.

- [ ] **Step 5: Commit.**
```bash
PATH="$HOME/.local/bin:$PATH" git add backend/internal/services/fitting_service.go backend/internal/services/fitting_effective_cargo_test.go
PATH="$HOME/.local/bin:$PATH" git commit -m "feat(backend): per-ship-item effective cargo computation"
```

---

## Task 2: Custom ship names from ESI `/assets/names`

**Files:**
- Modify: `backend/internal/services/fitting_service.go` (add `FetchAssetNames`)
- Modify: `backend/internal/models/trading.go` (add `Name` to `CharacterAssetShip`)
- Test: `backend/internal/services/asset_names_test.go` (new)

- [ ] **Step 1: Add the model field.** In `backend/internal/models/trading.go`, `CharacterAssetShip` struct, add after `TypeName`:
```go
	Name string `json:"name,omitempty"` // Custom ship name from ESI /assets/names; falls back to TypeName
```

- [ ] **Step 2: Write the failing test** for the names parser/fallback.

`backend/internal/services/asset_names_test.go`:
```go
package services

import "testing"

func TestParseAssetNames_MapsAndFallsBack(t *testing.T) {
	// ESI /assets/names returns [{item_id, name}]; "None" or empty means unnamed.
	raw := []byte(`[{"item_id":111,"name":"Ix-ITE-1"},{"item_id":222,"name":"None"}]`)
	names, err := parseAssetNames(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if names[111] != "Ix-ITE-1" {
		t.Errorf("want Ix-ITE-1, got %q", names[111])
	}
	if _, ok := names[222]; ok {
		t.Errorf(`"None" should be dropped so the caller falls back to the type name`)
	}
}
```

- [ ] **Step 3: Run it, verify it fails** (`parseAssetNames` undefined).

Run: `GOWORK=off go test ./internal/services/ -run TestParseAssetNames -v`
Expected: build error.

- [ ] **Step 4: Implement `parseAssetNames` + `FetchAssetNames`** in `fitting_service.go` (model the HTTP call on `fetchESIAssets`, but POST the item_ids):
```go
func parseAssetNames(body []byte) (map[int64]string, error) {
	var rows []struct {
		ItemID int64  `json:"item_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode asset names: %w", err)
	}
	out := make(map[int64]string, len(rows))
	for _, r := range rows {
		if r.Name == "" || r.Name == "None" {
			continue // unnamed -> caller falls back to type name
		}
		out[r.ItemID] = r.Name
	}
	return out, nil
}

// FetchAssetNames resolves custom names for the given item_ids (best-effort: on error
// returns an empty map so labels fall back to the type name — never blocks the list).
func (s *FittingService) FetchAssetNames(ctx context.Context, characterID int, itemIDs []int64, accessToken string) map[int64]string {
	if len(itemIDs) == 0 {
		return map[int64]string{}
	}
	bodyBytes, _ := json.Marshal(itemIDs) // ESI accepts up to 1000 ids
	endpoint := fmt.Sprintf("/latest/characters/%d/assets/names/", characterID)
	req, err := http.NewRequestWithContext(ctx, "POST", esiconfig.BaseURL+endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		s.logger.Warn("asset names: build request failed", "error", err)
		return map[int64]string{}
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.esiClient.Do(req)
	if err != nil {
		s.logger.Warn("asset names: esi request failed", "error", err)
		return map[int64]string{}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		s.logger.Warn("asset names: non-200", "status", resp.StatusCode)
		return map[int64]string{}
	}
	raw, _ := io.ReadAll(resp.Body)
	names, err := parseAssetNames(raw)
	if err != nil {
		s.logger.Warn("asset names: parse failed", "error", err)
		return map[int64]string{}
	}
	return names
}
```
Add `"bytes"` to the imports if missing.

- [ ] **Step 5: Run the test, verify it passes.**

Run: `GOWORK=off go test ./internal/services/ -run TestParseAssetNames -v`
Expected: PASS.

- [ ] **Step 6: Commit.**
```bash
PATH="$HOME/.local/bin:$PATH" git add backend/internal/services/fitting_service.go backend/internal/services/asset_names_test.go backend/internal/models/trading.go
PATH="$HOME/.local/bin:$PATH" git commit -m "feat(backend): fetch custom ship names from ESI assets/names"
```

---

## Task 3: Wire per-instance enrichment + names into `/character/ships`

Replace the per-type enrichment with: fetch assets + skills once, enrich each ship by its item_id, attach names.

**Files:**
- Modify: `backend/internal/handlers/trading.go` (`enrichShipsWithEffectiveCargo` → per-item; attach names)
- Test: `backend/internal/handlers/trading_ships_instance_test.go` (new, if a handler-level seam exists; otherwise assert via the fitting-service seam already covered in Task 1 and keep this task as the wiring change validated by `go build` + existing handler tests)

- [ ] **Step 1:** Read `enrichShipsWithEffectiveCargo` (~`trading.go:678`). It currently loops ships and calls `GetShipFitting(ctx, characterID, int(ships[idx].TypeID), accessToken)`. Rewrite it to:
  1. Fetch assets once via the fitting service (expose `FetchESIAssetsPublic` or reuse an existing accessor; if `fetchESIAssets` is unexported, add an exported `FetchAssetsForCharacter(ctx, characterID, token) ([]esiAsset, error)` wrapper in `fitting_service.go`).
  2. Fetch character skills once (the service already has a skills path used inside `GetShipFitting`; expose it similarly or reuse `skillsService.GetCharacterSkills`).
  3. For each ship: `eff, unavail := h.fittingService.EffectiveCargoForShipItem(ctx, int(ships[idx].TypeID), ships[idx].ItemID, assets, charSkills)`; set `ships[idx].EffectiveCargoCapacity = eff` (only when `eff > 0`) and `ships[idx].EffectiveCargoUnavailable = unavail`.
  4. Collect ship item_ids, call `names := h.fittingService.FetchAssetNames(ctx, characterID, itemIDs, accessToken)`, and set `ships[idx].Name = names[ships[idx].ItemID]` falling back to `ships[idx].TypeName` when absent.

  Concrete loop body:
```go
itemIDs := make([]int64, 0, len(ships))
for i := range ships {
	itemIDs = append(itemIDs, ships[i].ItemID)
}
names := h.fittingService.FetchAssetNames(ctx, characterID, itemIDs, accessToken)
for i := range ships {
	if !ships[i].IsSingleton {
		continue
	}
	eff, unavail := h.fittingService.EffectiveCargoForShipItem(ctx, int(ships[i].TypeID), ships[i].ItemID, assets, charSkills)
	if eff > 0 {
		ships[i].EffectiveCargoCapacity = eff
	}
	ships[i].EffectiveCargoUnavailable = unavail
	if n := names[ships[i].ItemID]; n != "" {
		ships[i].Name = n
	} else {
		ships[i].Name = ships[i].TypeName
	}
}
```

- [ ] **Step 2: Build + run existing handler/service tests.**

Run: `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go build ./... && GOWORK=off go test ./internal/...`
Expected: builds; existing tests pass (update any test that asserted the old per-type enrichment signature).

- [ ] **Step 3: Commit.**
```bash
PATH="$HOME/.local/bin:$PATH" git add backend/internal/handlers/trading.go backend/internal/services/fitting_service.go
PATH="$HOME/.local/bin:$PATH" git commit -m "feat(backend): enrich character ships per-instance with names + own fitting"
```

---

## Task 4: Optimizer uses the selected instance's cargo override

**Files:**
- Modify: `backend/internal/models/portfolio.go` (add `CargoCapacity`)
- Modify: `backend/internal/services/portfolio_service.go:35` (pass it through)
- Test: `backend/internal/services/portfolio_cargo_override_test.go` (new) — assert the request built for the route calc carries the override. If `Optimize` has no seam to inspect the route request, assert behaviourally: with a tiny `CargoCapacity`, `upt = CargoCapacity/UnitVolume` shrinks units (existing `cargoCapacity` flows into `OptimizeParams.CargoCapacity`).

- [ ] **Step 1:** Add to `PortfolioRequest` (`portfolio.go`):
```go
	CargoCapacity   float64  `json:"cargo_capacity,omitempty"` // Optional: selected ship instance's effective cargo (m³). When >0, overrides the per-type fitting recompute.
```

- [ ] **Step 2:** In `portfolio_service.go` where it builds the `RouteCalculationRequest` (line ~35), add `CargoCapacity: req.CargoCapacity,`.

- [ ] **Step 3: Write a failing test** proving the override reaches the route calc. Simplest reliable seam: a unit test on `Optimize` with a mocked routes service capturing the request; if such a mock exists in `portfolio_service_test.go`, assert `captured.CargoCapacity == req.CargoCapacity`. Otherwise add the assertion to an existing `Optimize` test that already mocks routes.

- [ ] **Step 4: Run, implement, run** — Expected: PASS.

Run: `GOWORK=off go test ./internal/services/ -run Portfolio -v`

- [ ] **Step 5: Commit.**
```bash
PATH="$HOME/.local/bin:$PATH" git add backend/internal/models/portfolio.go backend/internal/services/portfolio_service.go backend/internal/services/portfolio_cargo_override_test.go
PATH="$HOME/.local/bin:$PATH" git commit -m "feat(backend): portfolio optimize honours cargo_capacity override"
```

---

## Task 5: Frontend types + api-client carry item_id and name

**Files:**
- Modify: `frontend/src/types/trading.ts` (`Ship`)
- Modify: `frontend/src/lib/api-client.ts` (`BackendShipsResponse`, mapping)
- Test: `frontend/tests/lib/api-client-instances.test.ts` (new)

- [ ] **Step 1: Write the failing test.**

`frontend/tests/lib/api-client-instances.test.ts`:
```ts
// @vitest-environment jsdom
import { describe, it, expect, afterEach, vi } from "vitest";
import { fetchCharacterShips } from "@/lib/api-client";

afterEach(() => vi.restoreAllMocks());

it("carries item_id and name per instance", async () => {
  vi.spyOn(globalThis, "fetch").mockResolvedValue(
    new Response(JSON.stringify({ ships: [
      { item_id: 111, type_id: 657, type_name: "Iteron Mark V", name: "Ix-ITE-1", cargo_capacity: 5800, effective_cargo_capacity: 6090 },
      { item_id: 222, type_id: 657, type_name: "Iteron Mark V", name: "Ix-ITE-2", cargo_capacity: 5800, effective_cargo_capacity: 4900 },
    ], count: 2 }), { status: 200, headers: { "Content-Type": "application/json" } }),
  );
  const ships = await fetchCharacterShips();
  expect(ships.map(s => s.item_id)).toEqual([111, 222]);
  expect(ships[0].name).toBe("Ix-ITE-1");
  expect(ships[1].effective_cargo_capacity).toBe(4900);
});
```

- [ ] **Step 2: Run, verify it fails** (`item_id`/`name` not mapped).

Run: `cd frontend && npx vitest run tests/lib/api-client-instances.test.ts`

- [ ] **Step 3: Implement.** `types/trading.ts` `Ship`: add `item_id: number;` and `name: string;` (keep `name` as the display name — note `Ship.name` already exists as the ship/type name; reuse it for the instance name and add `item_id`). In `api-client.ts` `BackendShipsResponse.ships[]` add `item_id: number; name?: string;`, and in the map set `item_id: ship.item_id` and `name: ship.name ?? ship.type_name`.

- [ ] **Step 4: Run, verify it passes.**

- [ ] **Step 5: Commit.**
```bash
PATH="$HOME/.local/bin:$PATH" git add frontend/src/types/trading.ts frontend/src/lib/api-client.ts frontend/tests/lib/api-client-instances.test.ts
PATH="$HOME/.local/bin:$PATH" git commit -m "feat(frontend): carry ship item_id + instance name through api-client"
```

---

## Task 6: ShipSelect lists instances (dedupe by item_id)

**Files:**
- Modify: `frontend/src/components/trading/ShipSelect.tsx`
- Modify: `frontend/src/lib/mock-data/ships.ts` (fallback ships need `item_id` = `type_id` and `name`)
- Test: `frontend/tests/components/ShipSelect.test.tsx` (new)

- [ ] **Step 1: Write the failing test** — two same-type instances both appear, labelled by name+cargo, option value = item_id.
```tsx
// @vitest-environment jsdom
import { render, screen } from "@testing-library/react";
import { ShipSelect } from "@/components/trading/ShipSelect";
import { vi } from "vitest";

vi.mock("@/lib/api-client", () => ({
  fetchCharacterShips: vi.fn().mockResolvedValue([
    { item_id: 111, type_id: 657, name: "Ix-ITE-1", cargo_capacity: 5800, effective_cargo_capacity: 6090 },
    { item_id: 222, type_id: 657, name: "Ix-ITE-2", cargo_capacity: 5800, effective_cargo_capacity: 4900 },
  ]),
  fetchCharacterShip: vi.fn().mockResolvedValue({}),
}));

it("renders one option per instance, not per type", async () => {
  render(<ShipSelect value="" onChange={() => {}} authenticated />);
  expect(await screen.findByText(/Ix-ITE-1/)).toBeInTheDocument();
  expect(await screen.findByText(/Ix-ITE-2/)).toBeInTheDocument();
});
```
> Match the existing component-test render pattern (Radix Select renders options in a portal; if `findByText` can't see closed-select content, assert via the open state or by querying the items the existing tests use — mirror `tests/components/trading/ShipFittingCard.test.tsx` style).

- [ ] **Step 2: Run, verify it fails** (dedupe-by-type collapses the two).

- [ ] **Step 3: Implement.** In `ShipSelect.tsx`:
  - Change the dedupe `Set<number>` from `type_id` to `item_id` (`seen.add(s.item_id)`, `seen.has(s.item_id)`; active ship deduped by `item_id`).
  - Option `key`/`value` = `ship.item_id.toString()`.
  - Label = `${ship.name} (${cargoDisplay})` (cargoDisplay from the existing effective/unavailable/Basis logic).
  - The active ship object built from `fetchCharacterShip` must include `item_id` (use `active.ship_item_id`) and `name` (`active.ship_type_name || active.ship_name`).
  - `onChange` still emits the option value (now item_id). The parent maps item_id→{type_id, effective} (Task 7).

- [ ] **Step 4: Run, verify it passes** + full FE suite (`npx vitest run`).

- [ ] **Step 5: Commit.**
```bash
PATH="$HOME/.local/bin:$PATH" git add frontend/src/components/trading/ShipSelect.tsx frontend/src/lib/mock-data/ships.ts frontend/tests/components/ShipSelect.test.tsx
PATH="$HOME/.local/bin:$PATH" git commit -m "feat(frontend): list ship instances per item_id with name+cargo"
```

---

## Task 7: ROI page sends the selected instance's cargo override

**Files:**
- Modify: `frontend/src/types/trading.ts` (`PortfolioRequest` add `cargo_capacity?`)
- Modify: `frontend/src/app/roi-calculator/page.tsx` (`buildRequest` + selection mapping)
- Modify: `frontend/src/components/trading/PortfolioInputForm.tsx` (ShipSelect value/onChange now item_id; expose selected ship object up if needed)
- Test: covered by an updated ROI integration test if present; otherwise a unit test on `buildRequest`.

- [ ] **Step 1:** Add `cargo_capacity?: number;` to the FE `PortfolioRequest` interface.

- [ ] **Step 2:** The ROI page must know the selected ship's effective cargo + type_id from the chosen item_id. Lift the ships list (or a `Map<item_id, Ship>`) to the page, or have `ShipSelect` call `onChange(itemId, ship)` with the full ship. Simplest: `ShipSelect` accepts an `onSelect?(ship: Ship)` and the page stores the selected `Ship`. `buildRequest` then uses:
```ts
ship_type_id: selectedShip ? selectedShip.type_id : parseInt(form.ship, 10),
cargo_capacity:
  selectedShip?.effective_cargo_capacity != null && selectedShip.effective_cargo_capacity > 0
    ? selectedShip.effective_cargo_capacity
    : undefined,
```
> When `effective_cargo_unavailable` is set, leave `cargo_capacity` undefined so the backend recomputes (and the UI already shows the "fitted unbekannt" marker).

- [ ] **Step 3: Write/extend the failing test** for `buildRequest` (export it or test via the page) asserting `cargo_capacity` equals the selected instance's effective cargo, and is omitted when unavailable.

- [ ] **Step 4: Run, implement, run** — FE suite green.

- [ ] **Step 5: Commit.**
```bash
PATH="$HOME/.local/bin:$PATH" git add frontend/src/types/trading.ts frontend/src/app/roi-calculator/page.tsx frontend/src/components/trading/PortfolioInputForm.tsx frontend/tests/...
PATH="$HOME/.local/bin:$PATH" git commit -m "feat(frontend): ROI sends selected instance cargo as override"
```

---

## Task 8: Full verification + PR

- [ ] **Step 1: Backend gates.**

Run: `cd backend && gofmt -l internal pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./... && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./...`
Expected: gofmt empty, vet clean, golangci 0 issues, tests pass.

- [ ] **Step 2: Frontend gates.**

Run: `cd frontend && npx vitest run && npx eslint <changed files> && rm -rf .next && npm run build`
Expected: tests pass, eslint clean, build OK.

- [ ] **Step 3: Push + PR.**
```bash
git push -u origin feat/instance-accurate-ship-dropdown
gh pr create -R Sternrassler/eve-o-provit --base main --title "feat: instance-accurate ship dropdown (#147 follow-up)" --body-file <pr body>
```

- [ ] **Step 4: After CI green + merge — release** (separate, gated): CHANGELOG entry, `make release VERSION=0.18.0`, tag → prod deploy + smoke-test, then on-prod E2E (the Iteron instances now show their own correct cargo).

---

## Self-review notes

- **Spec coverage:** per-instance enrichment (Task 1, 3), custom names (Task 2, 3), item_id dedupe + labels (Task 5, 6), optimizer override (Task 4, 7), fail-loud unavailable (Task 1 returns unavailable; Task 3 sets it; Task 7 omits override when unavailable). All covered.
- **Open verification during execution:** confirm the Reinforced Bulkhead type id used in Task 1's test actually carries dogma attr 149 in the local SDE; confirm `FittingService` field names (`sdeDB`, `esiClient`, `logger`, `skillsService`) and the skills-fetch accessor for Task 3; confirm the routes-service mock seam for Task 4. These are lookups, not design changes.
