# Mining ore "sell raw vs. refine" ranking — Design

**Date:** 2026-05-31
**Status:** Approved (design)

## Problem

A miner wants to know, for the ore they can mine, whether **selling the raw ore** is more
profitable than **reprocessing it and selling the refined materials** — and which ore is
worth mining at all, ranked by **ISK per hour**. The comparison must account for the
character's reprocessing, ship, mining, and trade/broker skills, and pick the best NPC
reprocessing station in the region by the player's standings.

## Goal

A new **"Mining"** view that ranks ores by ISK/hour and, per ore, shows the verdict
**sell-raw vs. refine** with concrete net-ISK figures, the best NPC reprocessing station,
and the absolute ISK/hour from the current ship's mining rate.

## Decisions (from brainstorming)

- **Output:** a ranking over ores. Per ore: mining m³/h, ISK/h raw, ISK/h refined, verdict
  (sell-raw vs. refine) + delta, best NPC station.
- **Mining rate:** computed automatically from the **current (flown) ship** + its fitted
  mining lasers/strip miners (+ ore crystals) + mining skills (deterministic dogma calc).
- **Reprocessing facility:** **all NPC stations with the reprocessing service in the
  (current) region**; per-station reprocessing tax from the player's standing with the
  station's owner corp; the calc uses the **best** (lowest-tax) station per ore.
- **Ore scope:** a **selectable security band** (high/low/null) ∩ ores the current ship's
  equipment can mine.
- **Pricing:** **Taker** — sell instantly into the highest **buy orders**; **sales tax only**
  (Accounting), no broker fee; quantities capped at the order book (same model as Sell-Assets).
- **Region:** the character's **current region** (default, like the other views).
- **Surface:** a new **Mining** view on **web + Flutter**.

## Architecture

### Reprocessing yield (new deterministic calc)

`net_yield = base_facility_rate × (1 + 0.03·Reprocessing) × (1 + 0.02·ReprocessingEfficiency)
× (1 + 0.02·OreSpecificProcessing)` — base_facility_rate = **0.50** for NPC stations.
(Implants out of scope for v1.) Built analogous to the existing cargo/skill deterministic
calc in `backend/pkg/evedb`.

Refined output of `Q` units of an ore = for each material in `typeMaterials.materials`:
`floor(Q / portionSize) · material_qty · net_yield`. `portionSize` and `volume` come from
the `types` table; `typeMaterials.materials` is JSON `[{material, quantity}, …]`.

### Reprocessing tax (new; needs standings)

Per NPC station: `tax = max(0, 0.05 − 0.0075 · standing)` where `standing` is the player's
**effective standing with the station's owner corporation** (Connections/Diplomacy effects
out of scope v1 — use the raw corp standing). Station owner from `npcStations`→`npcCorporations`;
reprocessing-service stations filtered via `stationServices`/`stationOperations`. Per-corp
standings come from a **new ESI call `GET /characters/{id}/standings`** (the app currently
fetches only the highest faction/corp standing for broker fees). The calc picks the station
with the lowest tax (highest net materials) per region.

### Mining yield (new deterministic calc)

From the current ship's fitted mining modules (strip miners / mining lasers) read from
ESI assets (same path as the cargo fitting): per module
`yield_per_cycle_m3 = miningAmount × (1 + skill & role bonuses) × crystal_multiplier`,
`m3_per_hour = Σ_modules (yield_per_cycle_m3 / cycle_duration_s) × 3600`.
Skills: **Mining** (+5%/level), **Astrogeology** (+5%/level), ship **role bonuses**
(Mining Barge/Exhumer/Mining Frigate per-level yield), and the matching **ore crystal**
multiplier when present. Mining **drones** are **out of scope for v1** (later stage).
Modeled with the existing `pkg/evedb/dogma` infrastructure; exact attribute IDs
(`miningAmount`, module `duration`) are resolved against the local SDE during implementation.

### Market valuation (taker)

Reuse the market repository. For the raw ore and for each refined material, value at the
**highest buy order**, capped at the order's `VolumeRemain` (same order-book cap as
Sell-Assets, ref. the order-book-extrapolation rule). Net = gross − **sales tax**
(Accounting skill). No broker fee (taker).

### Per-ore comparison + ranking

For each in-scope ore:
- `raw_net_per_m3` = (buy-order ISK for the ore, net of sales tax) / ore_volume_m3.
- `refine_net_per_m3` = (Σ material buy-order ISK × net_yield, net of sales tax, at the best
  station) / ore_volume_m3.
- `best_per_m3 = max(raw, refine)`, `verdict = argmax`, `delta = |raw − refine|`.
- `isk_per_hour = mining_m3_per_hour × best_per_m3`.
Rank by `isk_per_hour` desc.

### Backend endpoint

`POST /api/v1/mining/ore-ranking` (auth) body `{ region_id, sec_band }` →
`{ ship: {...}, rows: [{ ore_type_id, ore_name, mining_m3_per_hour, raw_isk_per_hour,
refine_isk_per_hour, best: "raw"|"refine", delta_isk_per_hour, best_station: {id, name, tax},
... }], ... }`. `region_id` defaults to the character's current region; `sec_band` ∈
{high, low, null}.

### Ore → security-band map

The SDE does not cleanly encode where ores spawn. A small **curated map** (ore group →
available security bands) lives in backend code/config; it's small and stable. Documented as
the one non-SDE data source.

### Frontend (web + Flutter)

New **Mining** view: a **security-band selector** (high/low/null), the current region, the
`CurrentShipCard`, and a ranked table — columns: Ore · mining m³/h · ISK/h raw · ISK/h refine ·
verdict · best station · Δ. Mirrors the existing view conventions; web first, then Flutter.

## Error handling (fail-loud — project rule)

- Current ship has **no mining modules** → a visible "kein Mining-Setup" state; never fabricate
  a yield.
- Standings / fitting / market fetch errors → surfaced visibly (no silent 0/default), consistent
  with the no-silent-fallbacks principle.
- An ore with no buy orders (raw or for a material) → shown explicitly (not silently 0).

## Testing

- **Reprocessing yield:** known ore (Veldspar 1230, portionSize 100) with known skills →
  expected mineral quantities × net_yield (deterministic, against local SDE).
- **Mining yield:** a ship with a known strip miner + mining skills → expected m³/h.
- **Reprocessing tax:** standing → tax (boundary at standing 6.67 → 0%); best-station pick
  across ≥2 stations with different owner standings.
- **Comparison/ranking:** crafted prices where refine > raw for one ore and raw > refine for
  another → correct verdicts + ISK/h ordering.
- **Taker fees:** sales-tax-only applied; broker fee NOT applied; quantities capped at order book.
- **Frontend:** view renders the ranked table; sec-band selector changes scope; no-mining-setup
  state; fail-loud error states.

## Scope / YAGNI / phasing

- **Out of scope v1:** mining drones, reprocessing implants, Upwell-structure reprocessing
  (NPC stations only), Connections/Diplomacy standing modifiers, maker (sell-order) pricing.
- **Phasing** (each phase ships independently):
  1. Reprocessing-yield + market-taker + comparison + ranking + best NPC station (no mining
     yield yet — already answers "sell raw vs. refine per ore", ordered by net ISK/m³).
  2. Mining-yield + ISK/hour (ship/miners/skills) + the new `/standings` ESI call.
  3. Web Mining view.
  4. Flutter Mining view.
