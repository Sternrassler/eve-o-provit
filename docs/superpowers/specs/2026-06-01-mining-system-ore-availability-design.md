# System-genaue Erz-Verfügbarkeit (High + Low) + echte Erz-Namen — Design

**Datum:** 2026-06-01
**Status:** Approved (Design)
**Kontext:** Der Mining-Rechner rankt heute Erze nach einem groben Sec-Band
(`secBandOreGroups`), das im „high"-Band **auch Low-Sec-Erze** (Kernite, Omber,
Jaspet, Hemorphite, Hedbergite) enthält. Dadurch erscheinen im Hi-Sec Erze, die
es dort gar nicht gibt — und über die SDE-Namen „… II/III/IV-Grade" zusätzlich
falsche Namen. Dieses Feature macht die Erz-Auswahl **system-genau und
deterministisch** (für High- und Low-Sec) und zeigt die **echten Client-Namen**.

---

## 1. Befund: Erz-Verfügbarkeit ist deterministisch

EVE-University (autoritativ): Belt-Erz ist **fix pro System**, bestimmt durch
**Security-Status + Quarter (Empire-Fraktion)** (Null zusätzlich durch
Security-Class F–K). Alle Inputs liegen im SDE:
`mapSolarSystems.securityStatus`, `mapSolarSystems.regionID` →
`mapRegions.factionID` (Quarter). Quelle:
<https://wiki.eveuniversity.org/Asteroids_and_ore>.

**Scope dieses Specs: nur High-Sec + Low-Sec.** Null-Sec (#161), Class K (#162)
und exakte Belt-Inhalte per Scanner/Manuell (#163) sind separate Issues.

---

## 2. Sicherheits-Toggle (Bereitschaft)

Der bisherige High/Low/Null-Radio wird ein **Toggle: „High-only" vs „High+Low"**.
Er drückt die **Bereitschaft** aus, in Low-Sec zu operieren:
- **High-only:** Minen/Verkaufen/Aufbereiten/Routing nur in High-Sec.
- **High+Low:** Low-Sec-Systeme/-Stationen sind ebenfalls erlaubt.

Der Toggle steuert `avoidLowSec` im Routing (Feature #2: `avoidLowSec = !allowLow`).

**Backward-compatible API:** Das Request-Feld `sec_band` bleibt; interpretiert als
`allowLow := sec_band != "high"`. Die installierte Flutter-App (sendet
"high"/"low"/"null") funktioniert unverändert; das Web relabelt nur den Regler.

---

## 3. Erz-Verfügbarkeits-Regeln (wiki-belegt)

Erz-Gruppen-IDs (SDE „Asteroid"): Veldspar 462, Scordite 460, Plagioclase 458,
Pyroxeres 459, Omber 469, Kernite 457, Jaspet 456, Hemorphite 455, Hedbergite 454.

### High-Sec (securityStatus ≥ 0.5)
- **Immer (alle Quarters):** Veldspar (462), Scordite (460).
- **securityStatus ≤ 0.9:** + Pyroxeres (459).
- **Plagioclase (458):** Gallente/Minmatar bei ≤ 0.9 · Caldari bei ≤ 0.7 · Amarr **nie**.

### Low-Sec (0.0 < securityStatus < 0.5), nur wenn High+Low gewählt
Kein Veldspar/Scordite/Plagioclase. Pro Quarter:

| Quarter | bei ≤ 0.4 | zusätzlich bei ≤ 0.2 |
|---|---|---|
| Amarr | Pyroxeres (459), Kernite (457), Jaspet (456) | Hemorphite (455) |
| Caldari | Kernite (457), Pyroxeres (459) | Hedbergite (454) |
| Gallente | Omber (469), Jaspet (456) | Hemorphite (455) |
| Minmatar | Kernite (457), Omber (469) | Hedbergite (454) |

Die Schwellen sind **inklusive** (≤). „Quarter" wird aus der Region-Fraktion
bestimmt (s. §4). Das Erz-Set richtet sich nach dem **konkreten aktuellen System**
(dessen Security + Quarter).

**Toggle-Wirkung in `AvailableOreGroups(quarter, sec, allowLow)`:**
- `sec ≥ 0.5` → Hi-Sec-Set (unabhängig von `allowLow` — High ist immer erlaubt).
- `0 < sec < 0.5` **und** `allowLow` → Low-Sec-Set.
- `0 < sec < 0.5` **und nicht** `allowLow` → **leeres Set** (du bist in Low-Sec, willst aber nur High-Sec → hier wird nicht operiert; der Service zeigt einen Hinweis statt eines Rankings).
- `sec ≤ 0` → leeres Set (Null-Sec außerhalb des Scopes, → #161).

Zusätzlich steuert `allowLow` über `avoidLowSec = !allowLow` das **Routing/die
Kette** (Feature #2): bei High-only werden Low-Sec-Stationen/-Systeme als Verkaufs-/
Aufbereitungs-/Zwischenziele gemieden.

---

## 4. Quarter-Bestimmung (Region-Fraktion → Empire)

`mapRegions.factionID` → Quarter:
- 500001 → **Caldari**, 500002 → **Minmatar**, 500003 → **Amarr**, 500004 → **Gallente**.
- Minor-Empire-Fraktionen → Eltern-Empire: Ammatar Mandate (500007) → Amarr,
  Khanid Kingdom (500008) → Amarr. (Weitere Minor-Fraktionen bei Bedarf ergänzen.)
- **Quarter nicht auflösbar** (z. B. fraktionslose Region): es gelten nur die
  **quarter-unabhängigen** Erze (Veldspar, Scordite, ab ≤0.9 Pyroxeres). Plagioclase
  (quarter-spezifisch) wird dann weggelassen + die Zeile/Notiz vermerkt „Quarter
  unbekannt" (fail-loud, kein falsches Voll-Set).

---

## 5. Echte Erz-Namen (data-driven) + Filter

Unabhängig vom Sec-Thema: die SDE nennt Varianten „<Erz> II/III/IV-Grade"; im
Client heißen sie beschreibend (Concentrated/Dense Veldspar, Vivid/Radiant
Hemorphite …). Diese Namen sind **deterministisch aus CCPs eigenen Kompressions-
Blueprints** ableitbar: BP „Compressed <Descriptive> Blueprint" hat
`activities.manufacturing.materials[0].typeID` = die Roherz-Variante. Daraus:

- **Umbenennen** base/II/III → echter Name (data-driven Map aus `blueprints`).
- **Filtern** der IV-Grade/0-Grade-Erze: kein Kompressions-Blueprint → kein
  echter Name → kommen im Belt nicht vor → raus aus dem Ranking (analog zum
  Compressed-Filter #155).

Effekt: nach Rename+Filter trägt keine Belt-Zeile mehr ein „-Grade".

---

## 6. Komponenten & Datenfluss

- `backend/pkg/evedb/mining/availability.go` (neu):
  - `SystemQuarterAndSec(db, systemID int64) (quarter string, sec float64, err error)`
    — liest `securityStatus` + Region-`factionID`, mappt zum Quarter.
  - `AvailableOreGroups(quarter string, sec float64, allowLow bool) map[int64]bool`
    — die Regeltabellen aus §3.
- `backend/pkg/evedb/reprocessing/reprocessing.go`:
  - `ListOres` resolved Display-Namen via Blueprint-Map und filtert nicht-benennbare
    „-Grade"-Erze (§5).
- `backend/internal/services/mining_service.go`:
  - Aktuelles System (bereits via `GetCharacterLocation` vorhanden) →
    `SystemQuarterAndSec` → `AvailableOreGroups(quarter, sec, allowLow)` **ersetzt**
    `secBandOreGroups` für die Erz-Auswahl. `allowLow := req.SecBand != "high"`.
  - `avoidLowSec := !allowLow` (Routing, wie Feature #2).
- `backend/internal/services/ore_secband.go`: die alte band-basierte Map wird vom
  neuen Pfad ersetzt (für High/Low). Sie kann als Fallback für den (noch nicht
  umgestellten) Null-Pfad bleiben oder entfernt werden, falls ungenutzt.

Web/Flutter: keine Pflicht-Änderung (Namen + Erz-Set kommen aus der API). Das Web
relabelt den Regler optional zu „High-only / High+Low" (kosmetisch, eigener
kleiner Schritt im Plan).

---

## 7. Fehlerverhalten (fail-loud)

- Aktuelles System nicht auflösbar (kein Standort) → Ranking liefert eine klare
  Fehlermeldung statt eines falschen Voll-Sets.
- Quarter nicht auflösbar → nur quarter-unabhängige Erze + Notiz (§4).
- Kein neuer ESI-Scope, kein Deploy-Gate, kein Flutter-Pflicht-Build.

---

## 8. Tests

**`pkg/evedb/mining/availability_test.go` (reale SDE):**
- `SystemQuarterAndSec`: Jita (30000142) → Caldari, 0.95; ein Amarr-Hi-Sec-System
  → Amarr; ein Gallente-System → Gallente.
- `AvailableOreGroups`:
  - Caldari 0.95, allowLow=false → {Veldspar, Scordite} (kein Pyroxeres bei >0.9,
    kein Plagioclase bei >0.7).
  - Gallente 0.7, allowLow=false → {Veldspar, Scordite, Pyroxeres, Plagioclase}.
  - Amarr 0.6, allowLow=false → {Veldspar, Scordite, Pyroxeres} (kein Plagioclase).
  - Amarr 0.3, allowLow=true → {Pyroxeres, Kernite, Jaspet} (Low-Set).
  - Amarr 0.3, allowLow=false → **leeres Set** (Low-Sys, aber High-only).
  - 0.0/-0.5, allowLow=true → **leeres Set** (Null außerhalb des Scopes).
  - Quarter "" (unbekannt), 0.8, allowLow=false → {Veldspar, Scordite, Pyroxeres}.

**`pkg/evedb/reprocessing/reprocessing_test.go`:**
- `ListOres`: 17470 → Name „Concentrated Veldspar", 17444 → „Vivid Hemorphite",
  1230 → „Veldspar"; **keine** Zeile mit „-Grade" im Namen; 46689 (Veldspar
  IV-Grade) **nicht** enthalten.

**`internal/services/mining_service_test.go`:**
- Mit aktuellem System Jita (Fake) → nur Hi-Sec-Caldari-Erze im Ranking
  (Veldspar vorhanden, Hemorphite/Hedbergite **nicht**).
- `allowLow` aus `sec_band` korrekt abgeleitet; `avoidLowSec` entsprechend.

---

## 9. Betroffene Dateien

- `backend/pkg/evedb/mining/availability.go` (+ test) — neu.
- `backend/pkg/evedb/reprocessing/reprocessing.go` (+ test) — Rename + Filter.
- `backend/internal/services/mining_service.go` — Erz-Set aus System + allowLow.
- `backend/internal/services/ore_secband.go` — ersetzt/zurückgebaut.
- `frontend/src/components/.../mining` Regler-Label (kosmetisch, optional).

---

## 10. Nicht im Scope (Issues)

- **#161** Null-Sec (Security-Class F–J) + Null-Toggle.
- **#162** Class K (Drone-Regionen, regionsabhängig).
- **#163** Exakte Belt-Inhalte (manuelle Auswahl / Survey-Scan).
