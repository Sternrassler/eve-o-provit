# Mining ranking — sell & reprocess locations — Design

**Date:** 2026-05-31
**Status:** Approved (design)

## Problem

The Mining ore-ranking tells you *whether* to sell raw or reprocess, but not *where*: which
NPC station to reprocess at, and where to sell the raw ore or the refined minerals. The
backend already picks the best reprocessing station internally but only surfaces its tax
(no name), and it values sales at the best buy order but discards that order's location.

## Goal

Per ore row, surface the actionable locations:
- **Reprocess** (when refine is suggested): the chosen best NPC station — **name + system**.
- **Sell, raw** (when raw is suggested): the location of the best buy order for the ore.
- **Sell, refine**: a **per-mineral breakdown** — each output mineral with its effective
  quantity, buy price, and the location of its best buy order.

## Decisions (from brainstorming)

- Refine sell location: **per-mineral breakdown** (each mineral its own best-buy location),
  expandable per row.
- Buy orders: **best regardless of location** (incl. citadels) — **no valuation change**. NPC
  stations resolve to a name + system; citadels (LocationID not in SDE) show as
  "Player-Structure" (the region is already known — the ranking is region-scoped).
- Reprocessing stays **NPC stations only** (as today).

## Architecture

### Sell-location resolution (new)

`highestBuyPrice(region, type)` → `highestBuyOrder(region, type) (price float64, locationID int64, ok bool)`.

New helper `resolveSellLocation(ctx, locationID) SellLocation`:
- `SellLocation{ StationName string; SystemName string; IsStructure bool }`.
- NPC station (resolvable via `SDERepository`): `StationName` = `GetStationName` (station-type
  name, e.g. "Caldari Navy Assembly Plant"); `SystemName` = the station's `solarSystemID` →
  `GetSystemName`; `IsStructure = false`.
- Citadel / unresolvable (LocationID ≥ 1e12, or station lookup fails): `IsStructure = true`,
  `StationName = ""`, `SystemName = ""` (UI shows "Player-Structure"; region known from context).
- Failures to resolve are surfaced as the structure/unknown state — never a fabricated NPC name
  (fail-loud).

### Reprocess station resolution

The service already has `best.StationID`. Resolve `best_station_name` (= `GetStationName`)
and `best_station_system` (= station `solarSystemID` → `GetSystemName`).

### Response model (`OreRankRow` additions)

```go
type SellLocation struct {
	StationName string `json:"station_name,omitempty"`
	SystemName  string `json:"system_name,omitempty"`
	IsStructure bool   `json:"is_structure"`
}
type RefineMaterial struct {
	MaterialTypeID int64        `json:"material_type_id"`
	MaterialName   string       `json:"material_name"`
	EffectiveQty   int64        `json:"effective_qty"` // qty × netYield per portionSize batch, floored
	BuyPrice       float64      `json:"buy_price"`
	Sell           SellLocation `json:"sell"`
}
// OreRankRow gains:
//   BestStationName   string           `json:"best_station_name,omitempty"`
//   BestStationSystem string           `json:"best_station_system,omitempty"`
//   RawSell           *SellLocation    `json:"raw_sell,omitempty"`     // ore's best-buy location
//   Materials         []RefineMaterial `json:"materials,omitempty"`    // per-mineral breakdown
```

`MaterialName` resolved via `SDERepository.GetTypeInfo`. `EffectiveQty = floor(material.Quantity × netYield)`.

### Service flow (MiningService.OreRanking)

For each in-scope ore, in addition to today's comparison:
- ore best-buy → `RawSell = resolveSellLocation(oreLocationID)`.
- per material: best-buy (price + locationID) → `Materials += {…, EffectiveQty, BuyPrice, Sell: resolveSellLocation(matLocationID)}`.
- reprocess: `BestStationName/System` from `best.StationID`.
Caching: `resolveSellLocation` results memoized per request (many ores share the same hub
station) to avoid repeated SDE lookups.

### Frontend (web + Flutter)

Each ore row becomes **expandable**. Collapsed: today's columns (+ the reprocess station
name shown compactly when best=refine). Expanded:
- best=refine → the reprocess station (name + system) and the **per-mineral table**
  (Mineral · effektive Menge · Buy-Preis · Verkaufsort).
- best=raw → the raw ore's sell location.
- Citadel locations render as "Player-Structure".
Mirror the existing expandable/detail patterns in the app (web + Flutter).

## Error handling (fail-loud)

- Unresolvable location → "Player-Structure" / "unbekannt"; never a fabricated station name.
- Station/system/type-name resolution errors → logged + surfaced as the unknown state, not
  silently dropped.

## Testing

- `resolveSellLocation`: NPC station → name + system, `IsStructure=false`; citadel id (≥ 1e12)
  → `IsStructure=true`, empty name (against an in-memory SDE with npcStations + mapSolarSystems).
- `highestBuyOrder`: returns the price AND location of the max buy order; `ok=false` when none.
- Service: a Veldspar row carries `RawSell` (ore's buy location) and a `Materials` entry for
  Tritanium with `EffectiveQty = floor(400 × netYield)`, its buy price, and sell location; a
  refine-verdict ore carries the reprocess station name+system.
- Frontend: expanding a refine row shows the per-mineral table + reprocess station; a raw row
  shows the raw sell location; a citadel location shows "Player-Structure".

## Scope / YAGNI

- No system-accurate citadel resolution (SDE has no citadel data) — deliberately "Player-Structure".
- Station name is the station-*type* name + system (the SDE has no unique per-station name column);
  good enough to navigate to.
- Reprocessing remains NPC-only; valuation unchanged (best buy regardless of location).
