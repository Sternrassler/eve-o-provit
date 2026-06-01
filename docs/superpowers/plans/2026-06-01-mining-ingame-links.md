# In-Game Market + Route Links in Mining Ranking (Web) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** From the web Mining ranking, click an ore/mineral name to open its EVE market window, and click a reprocess station / sell location to set an in-game autopilot waypoint.

**Architecture:** Reuse the existing web client functions `openMarketDetails(typeId)` and `setWaypoint(destId, opts)` (already used by the ROI table; web SSO already holds both scopes). The only backend change is adding a `location_id` to `SellLocation` so sell locations have a waypoint target; the reprocess station already exposes `best_station_id`. No new endpoints, no new scopes, no Flutter, no deploy gate.

**Tech Stack:** Go 1.24 (backend), Next.js/React/TS + vitest (web).

**Spec:** `docs/superpowers/specs/2026-06-01-mining-ingame-links-design.md`.

**Verified facts:**
- `openMarketDetails(typeId: number)` and `setWaypoint(destinationId: number, opts: { clearOtherWaypoints?: boolean })` exist in `frontend/src/lib/api-client.ts` and throw human messages on 401/403/404.
- Web SSO requests `esi-ui.open_window.v1` + `esi-ui.write_waypoint.v1` (`frontend/src/lib/auth-context.tsx:33,36`).
- `frontend/src/components/trading/OreRankingTable.tsx` is a client component; each row is an `OreRankRow` sub-component with `useState` expand. Ore name is rendered at the chevron span; refine detail shows the reprocess station via `formatStation(...)` and a per-mineral table with `formatSell(m.sell)`; raw detail shows `formatSell(row.raw_sell)`.
- `models.SellLocation` (backend) currently has `StationName/SystemName/IsStructure` only. `locResolver.resolve(ctx, locationID)` (`backend/internal/services/mining_location.go`) already receives the `locationID` and caches a `SellLocation`.
- The mock pattern for `@/lib/api-client` + `@/hooks/use-toast` is established in `frontend/tests/components/PortfolioResultTable.test.tsx`.

**Test commands:**
- Backend (from `backend/`): `SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestResolveSellLocation -v`
- Web (from `frontend/`): `npx vitest run tests/components/OreRankingTable.test.tsx`

---

## File Structure

- `backend/internal/models/mining.go` (Modify) — `SellLocation.LocationID`.
- `backend/internal/services/mining_location.go` (Modify) — populate `LocationID` in `resolve`.
- `backend/internal/services/mining_location_test.go` (Modify) — assert `LocationID`.
- `frontend/src/types/trading.ts` (Modify) — `SellLocation.location_id?`.
- `frontend/src/components/trading/OreRankingTable.tsx` (Modify) — toast handlers + market/route links.
- `frontend/tests/components/OreRankingTable.test.tsx` (Modify) — mocks + click tests.

---

## Task 1: Backend — `location_id` on `SellLocation`

**Files:**
- Modify: `backend/internal/models/mining.go`
- Modify: `backend/internal/services/mining_location.go`
- Modify: `backend/internal/services/mining_location_test.go`

- [ ] **Step 1: Extend the resolve test (failing)**

In `backend/internal/services/mining_location_test.go`, `TestResolveSellLocation` currently checks the NPC and citadel cases. Add `LocationID` assertions. After the existing NPC block (the `npc := r.resolve(ctx, 60003760)` checks), add:
```go
	if npc.LocationID != 60003760 {
		t.Errorf("NPC LocationID: got %d, want 60003760", npc.LocationID)
	}
```
After the existing citadel block (`cit := r.resolve(ctx, 1_035_000_000_001)`), add:
```go
	if cit.LocationID != 1_035_000_000_001 {
		t.Errorf("citadel LocationID: got %d, want 1035000000001", cit.LocationID)
	}
```

- [ ] **Step 2: Run it to confirm it fails**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestResolveSellLocation -v`
Expected: FAIL — `npc.LocationID` undefined (compile error: SellLocation has no field LocationID).

- [ ] **Step 3: Add the field to `SellLocation`**

In `backend/internal/models/mining.go`, the struct is:
```go
type SellLocation struct {
	StationName string `json:"station_name,omitempty"`
	SystemName  string `json:"system_name,omitempty"`
	IsStructure bool   `json:"is_structure"`
}
```
Add the field:
```go
type SellLocation struct {
	StationName string `json:"station_name,omitempty"`
	SystemName  string `json:"system_name,omitempty"`
	IsStructure bool   `json:"is_structure"`
	LocationID  int64  `json:"location_id,omitempty"` // station/structure id — in-game waypoint target
}
```

- [ ] **Step 4: Populate it in `resolve`**

In `backend/internal/services/mining_location.go`, the `resolve` method builds `loc` then caches it:
```go
	r.cache[locationID] = loc
	return loc
```
Insert the assignment immediately before the cache write so both the NPC and structure branches get it:
```go
	loc.LocationID = locationID
	r.cache[locationID] = loc
	return loc
```

- [ ] **Step 5: Run the test to confirm it passes**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/services/ -run TestResolveSellLocation -v`
Expected: PASS.

- [ ] **Step 6: Wider backend gate**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... && gofmt -l internal && GOWORK=off golangci-lint run ./internal/...`
Expected: ok; no gofmt output; `0 issues`.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/models/mining.go backend/internal/services/mining_location.go backend/internal/services/mining_location_test.go
git commit -m "feat(mining): expose sell-location id for in-game waypoints"
```

---

## Task 2: Web — market + route links

**Files:**
- Modify: `frontend/src/types/trading.ts`
- Modify: `frontend/src/components/trading/OreRankingTable.tsx`
- Modify: `frontend/tests/components/OreRankingTable.test.tsx`

- [ ] **Step 1: Add `location_id` to the `SellLocation` TS type**

In `frontend/src/types/trading.ts`, the interface is:
```ts
export interface SellLocation {
  station_name?: string;
  system_name?: string;
  is_structure: boolean;
}
```
Add the field:
```ts
export interface SellLocation {
  station_name?: string;
  system_name?: string;
  is_structure: boolean;
  location_id?: number;
}
```

- [ ] **Step 2: Add the mocks + failing tests**

In `frontend/tests/components/OreRankingTable.test.tsx`, change the first import line and add mocks + mocked handles. Replace:
```ts
import { describe, it, expect } from "vitest";
import { render, screen, within, fireEvent } from "@testing-library/react";
import { OreRankingTable } from "@/components/trading/OreRankingTable";
import { OreRankRow } from "@/types/trading";
```
with:
```ts
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within, fireEvent } from "@testing-library/react";
import { OreRankingTable } from "@/components/trading/OreRankingTable";
import { OreRankRow } from "@/types/trading";
import { openMarketDetails, setWaypoint } from "@/lib/api-client";

vi.mock("@/hooks/use-toast", () => ({
  useToast: () => ({ toast: vi.fn() }),
}));
vi.mock("@/lib/api-client", () => ({
  openMarketDetails: vi.fn().mockResolvedValue(undefined),
  setWaypoint: vi.fn().mockResolvedValue(undefined),
}));

const openMarketMock = vi.mocked(openMarketDetails);
const setWaypointMock = vi.mocked(setWaypoint);

beforeEach(() => {
  openMarketMock.mockClear();
  setWaypointMock.mockClear();
});
```
Then add these tests inside the existing `describe("OreRankingTable", () => { ... })` block (e.g. after the last `it(...)`):
```ts
  it("opens the ore market on ore-name click without expanding the row", () => {
    render(<OreRankingTable rows={[makeRow({ ore_type_id: 1230, ore_name: "Veldspar" })]} />);
    fireEvent.click(screen.getByRole("button", { name: "Veldspar" }));
    expect(openMarketMock).toHaveBeenCalledWith(1230);
    // stopPropagation: the row stayed collapsed.
    expect(screen.queryByTestId("ore-ranking-detail")).not.toBeInTheDocument();
  });

  it("routes to the reprocess station and opens/routes minerals in the refine detail", () => {
    const r = makeRow({
      ore_type_id: 1230, ore_name: "Veldspar", best: "refine",
      best_station_id: 60003760, best_station_name: "Jita IV-4", best_station_system: "Jita",
      materials: [{
        material_type_id: 34, material_name: "Tritanium", effective_qty: 100, buy_price: 5,
        sell: { station_name: "Amarr VIII", system_name: "Amarr", is_structure: false, location_id: 60008494 },
      }],
    });
    render(<OreRankingTable rows={[r]} />);
    fireEvent.click(screen.getByTestId("ore-ranking-row")); // expand

    fireEvent.click(screen.getByRole("button", { name: /Jita IV-4 — Jita/ }));
    expect(setWaypointMock).toHaveBeenCalledWith(60003760, { clearOtherWaypoints: true });

    fireEvent.click(screen.getByRole("button", { name: "Tritanium" }));
    expect(openMarketMock).toHaveBeenCalledWith(34);

    fireEvent.click(screen.getByRole("button", { name: /Amarr VIII — Amarr/ }));
    expect(setWaypointMock).toHaveBeenCalledWith(60008494, { clearOtherWaypoints: true });
  });

  it("routes to the raw ore sell location in the raw detail", () => {
    const r = makeRow({
      ore_type_id: 1228, ore_name: "Scordite", best: "raw",
      raw_sell: { station_name: "Dodixie IX", system_name: "Dodixie", is_structure: false, location_id: 60011866 },
    });
    render(<OreRankingTable rows={[r]} />);
    fireEvent.click(screen.getByTestId("ore-ranking-row")); // expand
    fireEvent.click(screen.getByRole("button", { name: /Dodixie IX — Dodixie/ }));
    expect(setWaypointMock).toHaveBeenCalledWith(60011866, { clearOtherWaypoints: true });
  });
```

- [ ] **Step 3: Run the tests to confirm the new ones fail**

Run: `cd frontend && npx vitest run tests/components/OreRankingTable.test.tsx`
Expected: the 3 new tests FAIL (no `button` with those names yet); pre-existing tests still pass.

- [ ] **Step 4: Add imports + handlers in `OreRankingTable.tsx`**

Add to the import block:
```ts
import { useToast } from "@/hooks/use-toast";
import { openMarketDetails, setWaypoint } from "@/lib/api-client";
```
Inside the `OreRankRow` function, right after `const [open, setOpen] = useState(false);`, add:
```tsx
  const { toast } = useToast();
  const linkCls = "text-left hover:underline underline-offset-2 hover:text-primary transition-colors";
  const openMarket = async (typeId: number, name: string) => {
    try {
      await openMarketDetails(typeId);
      toast({
        title: "Markt-Detail an EVE gesendet",
        description: `${name} — falls nichts passiert: Markt-Fenster im Spiel (Alt+R) öffnen und nochmal klicken.`,
      });
    } catch (err) {
      toast({
        title: "Fehler",
        description: err instanceof Error ? err.message : "Unbekannter Fehler",
        variant: "destructive",
      });
    }
  };
  const setRoute = async (destId: number, label: string) => {
    try {
      await setWaypoint(destId, { clearOtherWaypoints: true });
      toast({ title: "Route gesetzt", description: `Waypoint in EVE gesetzt: ${label}` });
    } catch (err) {
      toast({
        title: "Fehler",
        description: err instanceof Error ? err.message : "Unbekannter Fehler",
        variant: "destructive",
      });
    }
  };
```

- [ ] **Step 5: Ore name → market button**

In the ore summary `<td>`, replace the bare `{row.ore_name}` (inside the chevron span) with:
```tsx
            <button
              type="button"
              className={linkCls}
              title="Marktdetails im EVE-Client öffnen"
              onClick={(e) => {
                e.stopPropagation();
                openMarket(row.ore_type_id, row.ore_name);
              }}
            >
              {row.ore_name}
            </button>
```

- [ ] **Step 6: Reprocess station → route link**

Replace the reprocess station div:
```tsx
                <div className="text-sm">
                  <span className="text-muted-foreground">Aufbereiten bei: </span>
                  {formatStation(row.best_station_name, row.best_station_system)}
                </div>
```
with:
```tsx
                <div className="text-sm">
                  <span className="text-muted-foreground">Aufbereiten bei: </span>
                  {row.best_station_id ? (
                    <button
                      type="button"
                      className={linkCls}
                      title="Route in EVE setzen"
                      onClick={() =>
                        setRoute(
                          row.best_station_id!,
                          formatStation(row.best_station_name, row.best_station_system)
                        )
                      }
                    >
                      {formatStation(row.best_station_name, row.best_station_system)}
                    </button>
                  ) : (
                    formatStation(row.best_station_name, row.best_station_system)
                  )}
                </div>
```

- [ ] **Step 7: Mineral name → market, sell location → route**

Replace the per-mineral `<tr>`:
```tsx
                    {(row.materials ?? []).map((m) => (
                      <tr key={m.material_type_id}>
                        <td className="py-1 pr-4">{m.material_name || `Typ ${m.material_type_id}`}</td>
                        <td className="py-1 pr-4 text-right">{m.effective_qty.toLocaleString("de-DE")}</td>
                        <td className="py-1 pr-4 text-right">{formatISK(m.buy_price)}</td>
                        <td className="py-1">{formatSell(m.sell)}</td>
                      </tr>
                    ))}
```
with:
```tsx
                    {(row.materials ?? []).map((m) => (
                      <tr key={m.material_type_id}>
                        <td className="py-1 pr-4">
                          <button
                            type="button"
                            className={linkCls}
                            title="Marktdetails im EVE-Client öffnen"
                            onClick={() =>
                              openMarket(m.material_type_id, m.material_name || `Typ ${m.material_type_id}`)
                            }
                          >
                            {m.material_name || `Typ ${m.material_type_id}`}
                          </button>
                        </td>
                        <td className="py-1 pr-4 text-right">{m.effective_qty.toLocaleString("de-DE")}</td>
                        <td className="py-1 pr-4 text-right">{formatISK(m.buy_price)}</td>
                        <td className="py-1">
                          {m.sell.location_id ? (
                            <button
                              type="button"
                              className={linkCls}
                              title="Route in EVE setzen"
                              onClick={() => setRoute(m.sell.location_id!, formatSell(m.sell))}
                            >
                              {formatSell(m.sell)}
                            </button>
                          ) : (
                            formatSell(m.sell)
                          )}
                        </td>
                      </tr>
                    ))}
```

- [ ] **Step 8: Raw sell location → route link**

Replace:
```tsx
                <span className="text-muted-foreground">Roh verkaufen bei: </span>
                {formatSell(row.raw_sell)}
```
with:
```tsx
                <span className="text-muted-foreground">Roh verkaufen bei: </span>
                {row.raw_sell?.location_id ? (
                  <button
                    type="button"
                    className={linkCls}
                    title="Route in EVE setzen"
                    onClick={() => setRoute(row.raw_sell!.location_id!, formatSell(row.raw_sell))}
                  >
                    {formatSell(row.raw_sell)}
                  </button>
                ) : (
                  formatSell(row.raw_sell)
                )}
```

- [ ] **Step 9: Run tests + build + lint**

Run: `cd frontend && npx vitest run tests/components/OreRankingTable.test.tsx && npm run build && npx eslint src/components/trading/OreRankingTable.tsx src/types/trading.ts`
Expected: all tests pass (3 new + pre-existing); build ok; eslint clean. (If a pre-existing test that uses `screen.getByText("Veldspar")` now matches text inside the new button — that still works, `getByText` matches by text content.)

- [ ] **Step 10: Commit**

```bash
git add frontend/src/types/trading.ts frontend/src/components/trading/OreRankingTable.tsx frontend/tests/components/OreRankingTable.test.tsx
git commit -m "feat(web): in-game market + route links in mining ranking"
```

---

## Task 3: Verify, PR, release, deploy

**Files:** none (process). **No Flutter change → no APK rebuild.**

- [ ] **Step 1: Full backend + frontend gates**

Run: `cd backend && SDE_DB_PATH=/home/ix/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db GOWORK=off go test ./internal/... ./pkg/... && gofmt -l internal cmd pkg && GOWORK=off go vet ./... && GOWORK=off golangci-lint run ./internal/... ./pkg/... ./cmd/...`
Then: `cd ../frontend && npx vitest run && npm run build`
Expected: backend ok + `0 issues`; frontend all tests pass + build ok.

- [ ] **Step 2: Push branch + open PR**

```bash
git push -u origin feat/mining-ingame-links
gh pr create --base main --title "feat(web): in-game market + route links in mining ranking" --body "<summary: ore/mineral name opens EVE market; reprocess station + sell locations set autopilot waypoint; backend adds SellLocation.location_id; web-only, no new scope/Flutter; spec+plan linked>"
```

- [ ] **Step 3: Wait for CI green, then squash-merge + delete branch**

```bash
gh pr checks <PR#> --watch --interval 20
gh pr merge <PR#> --squash --delete-branch
git checkout main && git pull --ff-only
```

- [ ] **Step 4: Release v0.24.0**

Add a `## [Unreleased]` CHANGELOG entry (Added: in-game market + route links in the mining ranking, web-only), commit, then:
```bash
make release-check
make release VERSION=0.24.0
git add CHANGELOG.md && git commit -m "chore(release): v0.24.0"
git tag v0.24.0 && git push origin main && git push origin v0.24.0
```

- [ ] **Step 5: Watch deploy + prod smoke**

```bash
gh run watch <deploy-run-id> --interval 20 --exit-status
curl -s https://eveonline.sternrassler.de/api/v1/version   # → v0.24.0
curl -s -o /dev/null -w "%{http_code}\n" https://eveonline.sternrassler.de/mining  # → 200
```

No tablet APK step — this feature does not touch Flutter.

---

## Notes for the implementer

- **No silent fallbacks:** a missing destination id (no reprocess station, citadel without a resolvable target) renders plain text instead of a link; a failing ESI call (EVE not running, no docking access) surfaces a destructive toast — never a silent no-op.
- **stopPropagation:** the ore-name market button must call `e.stopPropagation()` so opening the market does not also toggle the row's expand. Mineral/location buttons live inside the already-open detail panel, so they don't need it.
- **Backward compatibility:** sell locations / minerals without `location_id` (older API responses, or mineral sells in unresolvable citadels) render as plain text exactly as before.
