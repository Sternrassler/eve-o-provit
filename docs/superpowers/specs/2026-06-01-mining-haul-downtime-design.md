# Mining ISK/h inkl. Haul-Downtime (Greedy 1-Schritt) — Design

**Datum:** 2026-06-01
**Status:** Approved (Design)
**Kontext:** Feature #2 der Mining-Reihe. Die Mining-Rangliste sortiert heute nach
roher Mining-Rate (`m3h · netPerM3`). Das ignoriert die Realität, dass man den
Erzraum vollmacht, zum Verkaufen/Aufbereiten fliegt und dort weitermint. Dieses
Feature modelliert den **Lade-Zyklus** und rankt nach **effektiver ISK/h**.

**Optimierungs-Niveau:** Greedy-1-Schritt (bester nächster Zyklus ab dem
aktuellen System). Das volle Optimum (Maximum Profit-to-Time Ratio Cycle /
Tramp-Steamer) ist als Follow-up **Issue #158** angelegt — nicht in diesem Scope.

---

## 1. Workflow (vom Nutzer vorgegeben)

Immer im **aktuellen System** das empfohlene Erz abbauen, dann:
- **(Raw)** zur empfohlenen Verkaufsstation fliegen, Roherz verkaufen, dort
  weitermint.
- **(Refine)** zur empfohlenen Aufbereitungsstation fliegen, aufbereiten, dann
  zur besten Verkaufsstation der Raffinate fliegen, verkaufen, dort weitermint.

Es gibt **keinen Rückweg** zum Start-Belt — man mint dort weiter, wo man zuletzt
verkauft hat. Pro Zyklus zählen also nur die **Einweg-Etappen**.

---

## 2. Zyklus-Modell

```
t_fill  = oreHold / m3h[erz]                         // Erzraum vollmachen (h)
isk_load = oreHold · netPerM3[erz]                   // Wert einer vollen Ladung

Raw:    cycle = t_fill + travel(origin → ore_sell_sys)                       + 1·t_stop
Refine: cycle = t_fill + travel(origin → reprocess_sys) + travel(reprocess_sys → sell_hub_sys) + 2·t_stop

eff_isk_h = isk_load / cycle
```

- `oreHold` = Erzraum-Kapazität des aktuellen Schiffs (SDE-Attribut **1556**,
  „Mining Hold Capacity"). Schiff ohne Erzraum (aber mit Minern) → reguläre
  `types.capacity` (korrekt, kein Fallback).
- `m3h[erz]` ist die ore-spezifische Mining-Rate aus Feature #1 (Hull-Bonus +
  Crystal). `netPerM3[erz]` aus der bestehenden CompareOre-Logik.
- `t_stop` = fixer, **im UI sichtbarer** Overhead pro Stopp (Docken, Aktion,
  Undock; Default 75 s). Deckt auch den Same-System-Fall (0 Jumps) ab.
- **Ranking sortiert nach `eff_isk_h`.** Die rohe Brutto-ISK/h
  (`m3h · netPerM3`) bleibt als Detailwert erhalten.

---

## 3. Routing

- **Origin** = aktuelles Solar-System des Characters (`GetCharacterLocation`,
  schon vorhanden; die Region-Auflösung nutzt es bereits).
- **Reisezeit:** `navigation.CalculateTravelTime(db, fromSys, toSys, params, false)`
  → `RouteResult.TotalSeconds` (einfach) + `Jumps`. `params` aus Warp/Align des
  **aktuellen Schiffs** (Fitting-Bonuses `WarpSpeedAUS` / `AlignTime`, wie beim
  Hauling über `GetShipFitting`). `AvoidLowSec = (sec_band == "high")`.
- **Memoisierung:** Reisezeiten je Zielsystem cachen — zwei Tabellen pro Request:
  ab `origin` und ab `reprocess_sys` (das ist für alle Erze konstant). So bleibt
  die Anzahl `CalculateTravelTime`-Aufrufe an die Zahl distinkter Zielsysteme
  gebunden, nicht an die Zahl der Erze.

---

## 4. Sell-Hub-Wahl (Refine) — „bestes Einzel-Hub netto"

Pro Erz wird **das eine In-Region-Verkaufssystem** `S` gewählt, das `eff_isk_h`
maximiert (= fraktionale Wahl über endlich viele Kandidaten; Dinkelbach nicht
nötig, da klein):

```
value(S)     = Σ_mineral  qty_eff_mineral · bestBuyPrice(mineral, S) · (1 − salesTax)
cycle(S)     = t_fill + travel(origin → reprocess_sys) + travel(reprocess_sys → S) + 2·t_stop
choose S* = argmax_S  value(S) / cycle(S)
```

- `qty_eff_mineral` = Menge nach Net-Yield (wie in Feature #1, pro voller Ladung
  hochgerechnet auf `oreHold`).
- `bestBuyPrice(mineral, S)` = höchstes Buy-Order dieses Minerals **im System S**.
  Dafür werden die (ohnehin je Mineral-Typ geholten) Region-Market-Orders **nach
  System gruppiert** (Order-Location → System via `GetSystemIDForLocation`),
  Buy-Orders, bestes je System.
- **Kandidaten-Systeme** = alle Systeme, in denen mindestens ein Mineral dieses
  Erzes ein Buy-Order hat. (Erz-spezifisch, da Mineralmix variiert.)

**Interaktion mit Feature #1:** Für den Refine-Pfad ersetzt das die bisherige
„bestes Buy egal wo, pro Mineral einzeln"-Wertbasis durch „alle Minerale am
besten Einzel-Hub `S*`". Die Pro-Mineral-Aufschlüsselung (Feature #1, Detail-
Panel) zeigt dann die Preise **an `S*`** statt je Mineral woanders. Folge:
**Refine-Wert, Verdict und Ranking können sich gegenüber Feature #1 ändern** —
das ist gewollt (Routing-Realität). **Raw-Pfad** bleibt: Erz an seinem
Region-Best-Buy (ein Ort, eine Etappe).

---

## 5. Fehlerverhalten (fail-loud, keine stillen Fallbacks)

`eff_isk_h` wird **nur** berechnet, wenn Origin-System, Schiff-Warp/Align und die
benötigten Routen auflösbar sind. Andernfalls **kein** stilles „0 Downtime":
- Die Zeile behält die Brutto-ISK/h als Anzeigewert.
- Sie wird mit dem bestehenden `is_estimate`-Marker (Feature #1) + Grund
  versehen (z. B. „Route nicht berechenbar", „aktuelles Schiff/Warp unbekannt").
- Sortierung: Estimate-Zeilen nach `eff_isk_h`, wo vorhanden, sonst nach Brutto;
  Marker macht den Unterschied sichtbar.

Kein neuer ESI-Scope, kein Deploy-Gate.

---

## 6. Response-Schema (Ergänzung `OreRankRow`)

| Feld | Typ | Bedeutung |
|------|-----|-----------|
| `effective_isk_per_hour` | float | ISK/h inkl. Zyklus (Füllen + Etappen + Stops). Sortier-Key. |
| `load_volume_m3` | float | genutzte Erzraum-Kapazität pro Ladung. |
| `fill_minutes` | float | Zeit, den Erzraum zu füllen. |
| `cycle_minutes` | float | Gesamt-Zykluszeit (Füllen + Etappen + Stops). |
| `route_jumps` | int | Summe der Jumps über die Etappen des Zyklus. |
| `sell_system_name` | string (omitempty) | gewähltes Sell-Hub-System (Refine) bzw. Roherz-Sell-System (Raw). |

Bestehende `is_estimate`/`estimate_reason` werden wiederverwendet. `oreHold`/
`t_stop` werden als einmaliger Kontext (nicht je Zeile) im UI angezeigt.

---

## 7. Tests

**Backend (`pkg/evedb/mining` + Service):**
- Zyklus-Mathe: `eff_isk_h == isk_load / (t_fill + Σ legs + n·t_stop)` für Raw
  (1 Etappe) und Refine (2 Etappen) gegen feste Inputs.
- Same-System (0 Jumps): `travel ≈ 0` → cycle = `t_fill + n·t_stop`.
- Sell-Hub-Auswahl: bei zwei Kandidatensystemen wird das mit max `value/cycle`
  gewählt; per-System-Preise korrekt gruppiert.
- Memoisierung: gleiche Zielsysteme lösen nur einen Routing-Aufruf aus
  (Spy-Fake zählt Aufrufe).
- Fail-loud: Routenfehler / unbekanntes Origin/Schiff → `is_estimate` gesetzt,
  Brutto-ISK/h bleibt, `effective_isk_per_hour` = 0. Sortier-Key der Zeile =
  `effective_isk_per_hour` falls > 0, sonst die Brutto-ISK/h (Fallback nur fürs
  Sortieren, sichtbar durch den Marker).

**Web (`OreRankingTable`) & Flutter (`mining_*`):**
- Neue Felder parsen null/float-robust.
- Tabelle sortiert nach `effective_isk_per_hour`.
- Detail zeigt Ladung, Füllzeit, Zyklus, Jumps, Sell-Hub; `t_stop`-Hinweis.
- Estimate-Marker bleibt funktional.

---

## 8. Betroffene Dateien (Überblick)

- `backend/pkg/evedb/mining/cycle.go` (neu) — Zyklus-Mathe + Sell-Hub-Auswahl
  (pure Funktionen, ohne ESI/DB-IO; bekommt Reisezeiten + per-System-Preise als
  Inputs) + Tests.
- `backend/internal/services/mining_service.go` — Origin/Schiff-Params holen,
  Market-Orders je System gruppieren, Routing memoisiert, Zyklus je Erz, Felder
  setzen, Sortierung umstellen.
- `backend/internal/services/mining_service.go` braucht Schiff-Warp/Align →
  schmales Interface auf `FittingService.GetShipFitting` (neuer Provider-Param
  oder Erweiterung eines bestehenden).
- `backend/internal/models/mining.go` — neue `OreRankRow`-Felder.
- `frontend/src/types/trading.ts` + `OreRankingTable.tsx` + Test.
- `app/lib/api/mining_models.dart` + `features/mining/mining_screen.dart` + Tests.

Kein neuer ESI-Scope, keine `deployments`-Änderung. Follow-up: Issue #158.
