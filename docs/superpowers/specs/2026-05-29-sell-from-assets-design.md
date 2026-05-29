# Sell-from-Assets — Design Spec (Issue #107)

**Date:** 2026-05-29
**Status:** Approved (design), pending implementation plan

## Goal

Let a character pick an item they already own and get a ranked answer to *"where do I sell this for the most ISK, and how do I get there?"* — net proceeds after fees/skills, plus the route (jumps, travel time, security risk) from the item's current location to each sell location.

This is a **liquidation** flow (selling owned inventory), distinct from #45 hauling (buy-to-resell arbitrage) and #43 multi-hub (station-trading spread).

## Scope Decisions (locked)

1. **Sell model = Taker (instant sell to a buy order).** Net proceeds per unit = `best_buy_order_price × (1 − sales_tax_rate)`. **No broker fee** (broker fee applies only when placing an order; selling directly into an existing buy order incurs sales tax only). Placing one's own sell order (maker) is out of scope.
2. **Asset scope = everything.** No market-group filter. All assets are listed (aggregated by type + location, with quantity). Non-marketable items (no buy order anywhere) surface as `has_data:false` ("kein Marktpreis"), not hidden.
3. **Comparison set = 5 major hubs + all stations in the item's current region.**
   - Hubs: best buy order in the hub's region; route to the hub system.
   - Current region: best buy order per station; route to that station's system (often 0 jumps).
4. **Ranking = total net proceeds** (`unit_net × quantity`), descending. Jumps / travel time / security risk are shown alongside so the user weighs value vs. effort (not used as the sort key).
5. **Frontends = Web + Flutter** (parallel subagents against the locked API contract, as in #43/#44/#45).

## Architecture

New `AssetSaleService` with two responsibilities behind two endpoints:

```
[Web/Flutter picker]
   └─ GET /api/v1/trading/assets ──────────────► list owned items (cheap)
[user selects an item]
   └─ POST /api/v1/trading/assets/sell-options ─► rank sell locations (expensive)
```

Keeping listing and ranking separate avoids the N-items × hubs × routes blow-up: the heavy hub-price + route computation runs only for the one selected item.

### Reuse (no re-implementation)

- **Hub set + best prices + fees + skills:** `services.MajorHubs`, `bestPrices`, `MarketRepository.FetchMarketOrdersForType`, `SalesTaxRate`, `GetCharacterSkills`, `models.SkillsApplied` — the `MultiHubComparisonService` is the template.
- **Region order fetch (current region):** `RouteFinder.fetchMarketOrders(ctx, region)` (region-wide, paginated) — same as #45.
- **Navigation:** `SDERepository.GetSystemIDForLocation`, `navigation.CalculateTravelTime(db, from, to, &NavigationParams{AvoidLowSec}, false)`, security via `mapSolarSystems.securityStatus` (the `securityRisk(minSec)` helper pattern from #45: safe ≥0.5 / caution >0 / danger ≤0).
- **Names:** `SDERepository.GetStationName`, `GetSystemName`, `GetTypeInfo`.
- **Assets fetch:** generalize the existing `/characters/{id}/assets/` call in `handlers/trading.go` (currently ship-only, no quantity, single page) into a paginated, general fetch.

### New

- **General assets fetch** — paginated (ESI `X-Pages`), captures `quantity`, aggregates by `(type_id, location_id)` summing quantity, resolves name/system/region, flags `marketable` (has a market group via SDE).
- **`AssetSaleService`** — `ListAssets` and `SellOptions`.
- **`AssetsHandler`** — two endpoints + wiring.

## Data Flow — sell-options

1. Resolve `origin_system_id` from `location_id` (`GetSystemIDForLocation`); resolve current `region_id`.
2. Skills once → `sales_tax_rate`.
3. Build comparison entries:
   - For each of the 5 `MajorHubs`: fetch region orders for the type, `bestPrices` → best buy; `unit_net = best_buy × (1 − tax)`; route origin→hub system; `scope:"hub"`.
   - For the current region: fetch region orders for the type, group buy orders by `location_id`, best buy per station; route origin→station system; `scope:"current_region"`.
4. For each entry compute `total_net = unit_net × quantity`, `jumps`, `travel_time_min`, `security_risk`. Entries whose station/system can't be resolved (player structures, unknown IDs) are skipped; entries with no buy order get `has_data:false`.
5. Sort by `total_net` desc. `best` = first entry with `has_data && total_net > 0` (or `null` if none).

## API Contract (locked)

```
GET /api/v1/trading/assets        (auth)
→ 200 { "assets": [ {
        "type_id": int, "name": string, "quantity": int,
        "location_id": int64, "location_name": string,
        "system_id": int, "region_id": int, "marketable": bool
      } ], "count": int }
   401 if unauthenticated

POST /api/v1/trading/assets/sell-options   (auth)
  body { "type_id": int, "location_id": int64, "quantity": int, "avoid_low_sec": bool }
→ 200 {
    "type_id": int, "name": string, "quantity": int,
    "origin_system_id": int,
    "best": <option|null>,
    "options": [ {
       "scope": "hub" | "current_region",
       "region_id": int, "region_name": string,
       "station_id": int64, "station_name": string, "system_name": string,
       "buy_price": float, "unit_net": float, "total_net": float,
       "jumps": int, "travel_time_min": float,
       "security_risk": "safe"|"caution"|"danger", "has_data": bool
    } ],
    "skills_applied": { "applied": bool, "accounting": int, "broker_relations": int,
                        "sales_tax_rate": float, "broker_fee_rate": float }
  }
   400 if type_id<=0 || quantity<=0 || location_id<=0
   401 if unauthenticated
```

`options` is pre-sorted by `total_net` desc. `broker_fee_rate` is reported for transparency but **not** applied to taker proceeds.

## Error Handling

- ESI assets 401 → propagate 401.
- Per-hub / per-region order-fetch failure → log + skip that entry (don't fail the whole request).
- Unresolvable `location_id` (player structures, citadels) → skip the entry; for the *origin* item location, if unresolvable, return 200 with `options:[]` and `best:null` (can't compute routes) rather than erroring.
- Items with no buy orders anywhere → `options` entries all `has_data:false`, `best:null`.

## Testing

- **Backend unit:** taker net math (`unit_net = buy × (1−tax)`, no broker fee); ranking by total_net; current-region per-station grouping (best buy per station); asset aggregation (sum quantity by type+location); `marketable` flag. Use in-memory/fake market like `multi_hub_service_test.go`.
- **Backend handler:** 400 (missing/invalid fields), 401 (no auth), 200 happy path.
- **Web:** Vitest for the result component (option fields, security badge color, best-option highlight, waypoint button → `setWaypoint(station_id)`), plus an authed Playwright spec mirroring the existing ones.
- **Flutter:** widget test (option list, security chip colors, tap→detail, waypoint) + model fromJson test.

## Components / Files

**Backend**
- `internal/services/asset_sale_service.go` — `AssetSaleService`, `ListAssets`, `SellOptions` (+ taker net + ranking helpers).
- `internal/services/asset_sale_service_test.go`.
- `internal/handlers/assets.go` — `AssetsHandler` (`ListAssets`, `SellOptions`).
- `internal/models/` — `AssetItem`, `SellOption`, `SellOptionsResponse` (reuse `SkillsApplied`).
- `cmd/api/container.go` + `main.go` — wire handler, register `GET /trading/assets` and `POST /trading/assets/sell-options`.
- General paginated assets fetch (shared helper or in the service).

**Web** (`/sell-assets`): API helpers `listAssets()` + `findSellOptions(req)`, `AssetPicker` (search/filter by name/qty/location), `SellOptionsResult` (best highlight, option list, security badge, route, "Route an EVE übertragen" via existing `setWaypoint`), nav entry.

**Flutter** (`/sell-assets`): models, dio methods, providers (AsyncNotifier), adaptive screen (picker → result), security chips, waypoint button, router + nav entry.

## Out of Scope

- Maker / sell-order placement (broker fee, order competition) — possible future issue.
- Buy-order *range* semantics (station vs region range) — hubs use region-best routed to hub system; current region uses per-station best. Good-enough approximation; documented here.
- Multi-item batch optimization / "liquidate my whole hangar" planning.

## Known Limitations

- **Items in player structures / citadels (accepted, not fixed).** The route is computed from the item's current location. The SDE only knows NPC stations, so an item whose `location_id` is a player-owned structure (Upwell citadel/engineering complex, `location_id` outside the NPC-station/system ranges) cannot be resolved to a system. For such items the service returns 200 with `best:null` / empty `options`, and the UI shows "Keine Verkaufsorte". This is an accepted limitation (decided 2026-05-29) — we explicitly do **not** resolve structure locations. A possible future enhancement would be to still show hub net proceeds *without* a route when the origin is an unresolvable structure.

## Follow-up (shipped 2026-05-29, v0.12.1)

- Asset picker is **sortable by name or quantity**, ascending/descending (web + Flutter). Client-side only; no API change.
