# Mining: Markt-Mengen-Cap als „volle Ladungen" pro Zeile — Design

**Datum:** 2026-06-02
**Status:** Approved (Design)

## Problem

Das Mining-Ranking nimmt den höchsten erreichbaren Buy-Preis und tut so, als ließe sich **unbegrenzt** dazu verkaufen — `VolumeRemain` (Order-Kapazität) wird im Mining-Pfad nirgends genutzt. Bei dünnen Orders ist die ISK/h ab der ersten Ladung Fiktion. Es fehlt zudem jede Mengen-/Ladungs-Anzeige.

## Ziel

Pro Zeile anzeigen, **wie viele komplette Erzraum-Ladungen** (bezogen auf das aktuelle Schiff) die gewählte Order aufnimmt — plus fail-loud-Warnung, wenn der Bestpreis nicht mal eine volle Ladung trägt.

## Entscheidungen (vom Owner bestätigt)

1. **Cap-Quelle:** die EINE beste erreichbare Buy-Order (deren `VolumeRemain`), passend zur bestehenden Bestpreis-Logik. Keine Order-Book-Tiefe.
2. **ISK/h-Wirkung:** nur anzeigen — ISK/h und Ranking unverändert; ABER die Zeile wird fail-loud markiert, wenn `market_loads < 1` (Bestpreis-ISK/h dann optimistisch).
3. **Plattform:** Web + Flutter (APK separat gebaut).

## Berechnung

Nur wenn tatsächlich gemint wird (`oreM3h > 0`, also Erzraum `oreHoldM3 > 0` aufgelöst). Sei `unitsPerLoad = oreHoldM3 / oreVolumeM3` (Erz-Einheiten pro voller Erzraum-Füllung).

- **Roh:** `rawLoads = orderVolumeRemain / unitsPerLoad` (Order-`VolumeRemain` in Erz-Einheiten).
- **Refine:** pro Mineral `m` am gewählten Hub: produzierte Einheiten pro Ladung `mPerLoad = (unitsPerLoad / portionSize) * matQty_m * net`; `loads_m = mineralVolumeRemain_m / mPerLoad`. **`refineLoads = min über alle m`** (das zuerst erschöpfte Mineral bindet).
- Angezeigt wird der Cap des **`Best`-Pfads** (raw→rawLoads, refine→refineLoads) als `market_loads`.

## Datenfluss / Felder

- `systemBuy` + `bestReachableBuyOrder` führen zusätzlich `volumeRemain int` (der gewählten Order).
- Neues Feld `OreRankRow.MarketLoads float64 json:"market_loads,omitempty"` — 0/weggelassen, wenn nicht mining oder Pfad nicht erreichbar.
- Web + Flutter: in der aufgeklappten Zeile „Markt nimmt ~X.X Ladungen"; bei `0 < market_loads < 1` ein ⚠ „Bestpreis < 1 Ladung — ISK/h optimistisch".

## Fail-loud / Tests

- Service-Test: dünne Order (kleiner `VolumeRemain`) → `market_loads < 1`; fette Order → großer Wert; refine bindet am knappsten Mineral.
- Reine Zusatzinfo; bestehende ISK/h-/Verdict-Tests bleiben unberührt.

## Nicht im Scope

- Order-Book-Tiefe / Preis-Floor; echtes ISK/h-Capping; absolute Mengen pro Mineral (Materials zeigt schon Effektiv-Qty pro Portion).
