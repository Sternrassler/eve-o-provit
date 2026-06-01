# Erreichbarkeitsbewusstes Haul-Downtime (entkoppelt) — Design

**Datum:** 2026-06-01
**Status:** Approved (Design)
**Kontext:** Der Haul-Downtime-Rechner (effektive ISK/h, v0.23.0+) markiert unter
„Nur High-Sec" **alle** Zeilen als „≈ Schätzung", weil die Routen zu den
gewählten Verkaufs-/Reprocess-Stationen scheitern (die liegen im Low-Sec oder in
Citadels). Root Cause bestätigt per Toggle-Test: mit „High + Low-Sec" lösen fast
alle Zeilen auf. Zwei Schwächen im Modell:

1. **Geteiltes `rowResolved`:** scheitert eine Etappe (z. B. Refine-Route), wird
   die **ganze** Zeile geschätzt — obwohl der Raw-Pfad routbar wäre.
2. **Ziel-Wahl ignoriert Erreichbarkeit:** Reprocess/Verkauf wird „best egal wo"
   gewählt (Feature #1), auch wenn nur über Low-Sec oder in einer Citadel.

---

## 1. Ziel

Die Verkaufs-/Reprocess-Ziele werden **erreichbarkeitsbewusst** unter der
Security-Bereitschaft des Spielers gewählt, und Raw-/Refine-Pfad werden
**entkoppelt** berechnet. Eine Zeile ist nur noch „Schätzung", wenn der
Hull/Crystal-Bonus nicht auflösbar ist (Feature #1) **oder** **kein** Pfad einen
erreichbaren Verkaufs-/Reprocess-Ort hat.

**Backend-only.** Response-Schema unverändert (nur Werte/Gründe ändern sich) →
Web + Flutter brauchen keine Änderung, kein APK-Rebuild.

---

## 2. Erreichbarkeit

Eine Station/ein Buy-Order-Ort gilt als **erreichbar**, wenn:
- die `location_id` zu einem Solar-System auflösbar ist (`GetSystemIDForLocation`;
  Citadels ≥ 1e12 sind es **nicht** → unerreichbar), **und**
- `navigation.CalculateTravelTime(origin → System, params)` unter
  `params.AvoidLowSec = !allow_low_sec` eine Route findet (`travelSecs` resolved).

Helper (Service): `reachableSystem(origin, destSys, params, travelMemo) (secs, jumps, ok)`
— ist `s.travelSecs` (existiert bereits; `from==to` → 0/0/true). Order-Location →
System wird über die bestehende `sysOf`-Map memoisiert; unauflösbare (Citadel)
Locations werden mit Sentinel `0` als „unerreichbar" gecacht.

---

## 3. Ziel-Wahl = bestes ERREICHBARES (ersetzt „best egal wo")

### 3.1 Raw-Verkauf (pro Erz)
`highestBuyOrder` wird zu `bestReachableBuyOrder(ctx, regionID, typeID, origin,
params, travelMemo, sysOf) (price float64, locationID int64, sellSys int64, secs
float64, jumps int, ok bool)`: iteriert die Buy-Orders des Typs in der Region,
löst je Order die Location → System auf, prüft Erreichbarkeit, und behält den
**höchsten Preis unter den erreichbaren** Orders (inkl. dessen System +
Reisezeit). `ok=false`, wenn kein erreichbares Buy-Order existiert.

### 3.2 Reprocess-Station (einmal pro Request)
Die Kandidaten (`StationStanding` aus den Region-Reprocess-Stationen) werden auf
**erreichbare** gefiltert (Station-System auflösen + `reachableSystem`), dann
`BestStation` über die erreichbaren. Ergebnis: `reprocessSys`, `reprocessSecs`,
`reprocessJumps`, Rate/Take. Gibt es keine erreichbare Station → Refine-Pfad ist
für **alle** Erze nicht verfügbar.

### 3.3 Mineral-Verkaufs-Hub (pro Erz, Refine)
Wie Feature #2, aber die Kandidaten-Systeme werden auf **erreichbare** (von
`reprocessSys` aus) gefiltert, bevor das beste Hub nach `eff` gewählt wird.

### 3.4 Auswirkung auf Anzeige + Brutto-ISK/h
Da Raw-Preis und Refine-Hub-Preise nun die **erreichbaren** Bests sind, fließen
sie in `CompareOre` → `RawNetPerM3`/`RefineNetPerM3` und damit in **Brutto-ISK/h
(roh/refine)**, in `RawSell`/`Materials`-Anzeige und in die effektive ISK/h.
Alles ist konsistent „was du unter deiner Sicherheits-Bereitschaft erreichst".
Die Materials-Aufschlüsselung zeigt die Preise/Orte des gewählten **erreichbaren
Hubs**.

---

## 4. Entkoppelte Pfad-Berechnung

Pro Erz, unabhängig:
- **Raw:** wenn `bestReachableBuyOrder` ok → `rawNet` (CompareOre raw-Seite mit dem
  erreichbaren Erz-Preis) und `rawEff = EffectiveISKPerHour(oreHold, m3h, rawNet,
  raw.secs, 1·t_stop)`. Sonst Raw nicht verfügbar.
- **Refine:** wenn erreichbare Reprocess-Station **und** ein erreichbares Hub
  existieren → `refNet`(Hub) und `refEff = EffectiveISKPerHour(oreHold, m3h,
  refNet, reprocessSecs + hubSecs, 2·t_stop)`. Sonst Refine nicht verfügbar.

**Zeilen-Ergebnis:**
- `available := rawAvailable || refineAvailable`.
- Wenn `available`: `Best` = Pfad mit höherer effektiver ISK/h; setze
  `EffectiveISKPerHour`, `CycleMinutes`, `FillMinutes`, `RouteJumps`,
  `SellSystemName`, `RawNetPerM3`/`RefineNetPerM3` (nur die verfügbaren Pfade),
  Brutto-ISK/h, `RawSell`/`Materials` entsprechend.
- Wenn **nicht** `available` (kein erreichbarer Verkaufs- **und** kein
  erreichbarer Reprocess-Ort): die Zeile wird **übersprungen** (nicht gerankt) —
  das Erz ist unter deiner Sicherheits-Bereitschaft nicht verkauf-/aufbereitbar.
  Da `bestReachableBuyOrder` das beste **erreichbare** Order wählt (meist eine
  NPC-Station, auch wenn das global beste in einer Citadel liegt), ist „gar
  nichts erreichbar" ein seltener Rand. `is_estimate` bleibt damit **nur** für
  Hull/Crystal (Feature #1) reserviert.

Das behebt (1): ein nicht erreichbarer Refine-Pfad lässt einen erreichbaren
Raw-Pfad **unangetastet** (Zeile zeigt Raw, keine Schätzung).

---

## 5. Fail-loud / Logging

Die bisher stillen `rowResolved = false`/`cycleResolved = false`-Pfade bekommen
`s.logger.Warn` mit Grund (Schiff-Fitting, Erzraum, Erz-Sell-Route, Reprocess-
Route, Hub-Route). Estimate-Gründe sind spezifisch (s. §4). Kein stiller Fallback.

---

## 6. Komponenten & Dateien

- `backend/internal/services/mining_service.go`:
  - `bestReachableBuyOrder` (ersetzt `highestBuyOrder` im Erz-/Mineral-Kontext, wo
    Erreichbarkeit zählt).
  - Reprocess-Station-Auswahl auf erreichbare filtern.
  - Refine-Hub-Kandidaten auf erreichbare filtern.
  - Per-Erz-Schleife: Raw/Refine entkoppelt, `available`-Logik, granulare Gründe.
  - Logging auf den Fehlerpfaden.
- Keine Model-/Web-/Flutter-Änderung (Schema gleich).

---

## 7. Tests

**Service (`mining_service_test.go`, reale SDE + Fakes):**
- Erreichbarkeit: ein Fake-Buy-Order in einer Citadel-Location (≥1e12) → als
  unerreichbar behandelt; ein NPC-Order im selben System (Origin) → erreichbar,
  travel 0.
- Entkopplung: Raw erreichbar, Refine-Reprocess unerreichbar → Zeile resolved als
  „raw" mit `EffectiveISKPerHour > 0`, **kein** `IsEstimate`.
- Kein erreichbarer Ort (alle Orders Citadel) → `IsEstimate=true` + Grund.
- Bestehende Veldspar/Estimate/Variant-Namens-Tests bleiben grün (ggf. Fakes um
  Locations/Erreichbarkeit ergänzen).

---

## 8. Nicht im Scope

- Reachability-aware Trading/Hauling/ROI (nur Mining-Ranking).
- Null-Sec/Class-K-Erz-Sets (#161/#162), exakte Belt-Inhalte (#163).
- Multi-Stop-Mineral-Verkauf (bleibt Einzel-Hub; Issue #158-Kontext unberührt).
