# Instance-accurate ship dropdown — Design

**Date:** 2026-05-31
**Status:** Approved (design)

## Problem

The ROI ship dropdown is **type-level**: it deduplicates owned ships by `type_id`, and
the effective cargo for a type is computed from the **first** singleton instance of
that type found in the character's assets:

```go
// internal/services/fitting_service.go — fetchFittingFromESI
for _, asset := range assets {
    if asset.TypeID == shipTypeID && asset.IsSingleton {
        shipItemID = asset.ItemID
        break // ← first instance of the type wins
    }
}
```

A character owning two ships of the same type, fitted differently, sees an **arbitrary**
instance's effective cargo. A cargo-reducing module on that instance (e.g. Reinforced
Bulkheads, dogma attr 149 < 1) yields an effective value **below the base hull** — observed
live: an Iteron Mark V shown as **4.9k m³** while the inspected (empty) hull is **6.09k m³**.

The deterministic cargo calculation itself is **correct** (verified against the local SDE:
Iteron 657 with Gallente Industrial L1 → exactly 6090 m³). The fault is purely the
arbitrary instance selection feeding the type-level display + optimizer.

## Goal

List each owned ship **instance** individually (no type dedupe), each with its own custom
ship name and the correct effective cargo of **its** fitting. Selecting an instance feeds
that instance's exact cargo into the ROI optimizer.

## Architecture

### Backend (`backend/`)

1. **Per-instance enrichment.** `enrichShipsWithEffectiveCargo` currently calls
   `GetShipFitting(characterID, shipTypeID)` per ship, which resolves the first instance of
   the type. Replace with an **item-scoped** computation: an instance-aware variant of
   `fetchFittingFromESI` that takes the concrete `shipItemID` and aggregates the modules of
   **that** ship (`asset.LocationID == shipItemID && isFittedSlot(...)`). Each
   `CharacterAssetShip` gets the effective cargo of its own fitting. The existing
   `effective_cargo_unavailable` flag stays per-instance (set on a fitting fetch error).

2. **Custom ship names.** One batch ESI call `POST /characters/{id}/assets/names` with the
   ship item_ids; attach `name` to each `CharacterAssetShip`
   (`Name string json:"name,omitempty"`). Fallback to the type name when a ship is unnamed
   or the names call fails (fail-soft for labels — never blocks the list; logged).

3. `/character/ships` already builds one `CharacterAssetShip` per ship asset, so it is
   already per-instance — the change is making the effective cargo + name per-instance too.

### Frontend (`frontend/`)

4. `Ship` type gains `item_id: number` and `name: string`. `fetchCharacterShips` maps them
   through.

5. `ShipSelect` deduplicates by **`item_id`** instead of `type_id`. Each option:
   - label: `name (effective cargo)` — fitted value, or `Basis X` / `Basis X — fitted unbekannt`
     using the existing per-instance logic;
   - value: `item_id` (unique).
   The active/flown ship is merged + deduplicated by `item_id`.

6. ROI page: on submit, pass the **selected instance's** effective cargo as `cargo_capacity`
   plus its `ship_type_id`. (Selection maps option `item_id` → `{type_id, effective_cargo}`.)

### Optimizer wiring

7. `RouteCalculationRequest.CargoCapacity` already overrides the per-type recompute
   (`route_service.go:147`: when non-zero, used directly). Add a `cargo_capacity` field to
   the ROI/portfolio request and have `PortfolioService.Optimize` pass it into
   `RouteCalculationRequest.CargoCapacity`. Result: the optimizer uses the selected
   instance's exact cargo and never recomputes per-type → the root cause is gone end-to-end.

## Data flow

```
ESI assets ──► /character/ships ──► per-instance {item_id, type_id, name,
                                     effective_cargo_capacity, effective_cargo_unavailable}
   (assets/names enriches `name`; per-item fitting enriches effective cargo)
        │
        ▼
ShipSelect (one option per item_id, label name+cargo, value item_id)
        │  user picks an instance
        ▼
ROI submit ──► optimize { ship_type_id, cargo_capacity: <instance effective> }
        │
        ▼
PortfolioService.Optimize ──► RouteCalculationRequest.CargoCapacity (override)
        │
        ▼
route_service.Calculate uses cargoCapacity directly (no per-type fitting recompute)
```

## Error handling (fail-loud, per project principle)

- Per-instance fitting fetch error → that instance's `effective_cargo_unavailable=true` →
  UI shows `Basis X — fitted unbekannt` (visible, not silent).
- `/assets/names` failure → fall back to the type name for labels; log; do **not** block
  the ship list.
- Malformed `/character/ships` response on the client still throws (existing fail-loud
  array validation).

## Testing

- **Backend:** two same-type ships with different fittings produce **different** effective
  cargo (the regression for the Iteron case: empty instance → 6090; bulkhead instance → its
  reduced value, attributed to its own ship). Names attachment + fallback when names call
  fails. Optimizer uses the `cargo_capacity` override (no per-type recompute).
- **Frontend:** `ShipSelect` lists instances without deduping by type; labels with
  `name (cargo)`; selecting passes the correct `cargo_capacity`. `api-client` carries
  `item_id`/`name`. Existing fail-loud tests stay green.

## Scope / YAGNI

- Only assembled (singleton) ships, as today; packaged ships excluded.
- No change to ship-type-level features elsewhere (trading page, hauling) beyond what the
  shared `Ship` type / `fetchCharacterShips` require to keep compiling.
