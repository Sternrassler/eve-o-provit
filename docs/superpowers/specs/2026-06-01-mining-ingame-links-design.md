# In-Game-Markt- + Routen-Links in der Mining-Rangliste (nur Web) — Design

**Datum:** 2026-06-01
**Status:** Approved (Design)
**Kontext:** Feature #3 der Mining-Reihe. Aus der Mining-Rangliste heraus soll man
direkt im EVE-Client (a) das **Marktfenster** eines Erzes/Minerals öffnen und
(b) eine **Autopilot-Route** zum Aufbereitungs-/Verkaufsort setzen. **Nur Web** —
ausdrücklich **kein** Flutter (vermeidet das Mobile-ESI-Scope-Deploy-Gate).

---

## 1. Reuse & Scope

Beide Aktionen existieren bereits und werden im ROI-Table genutzt:
- `openMarketDetails(typeId)` → `POST /api/v1/esi/ui/openwindow/marketdetails`
  (Scope `esi-ui.open_window.v1`).
- `setWaypoint(destinationId, opts)` → `POST /api/v1/esi/ui/autopilot/waypoint`
  (Scope `esi-ui.write_waypoint.v1`).

Beide Scopes fragt der **Web**-SSO bereits ab (`frontend/src/lib/auth-context.tsx`
Zeilen 33 + 36) → **kein Re-Login, kein neuer Scope, kein Deploy-Gate**. Keine
Flutter-Änderung.

---

## 2. Mentales Modell

- **Name anklicken → Markt öffnen** (`openMarketDetails(type_id)`): Erz-Name und
  jeder Mineral-Name in der Refine-Aufschlüsselung.
- **Ort anklicken → Route setzen** (`setWaypoint(destination_id, {clearOtherWaypoints: true})`,
  Route ersetzen): Reprocess-Station, Roherz-Verkaufsort und je Mineral der
  Verkaufsort.

---

## 3. Backend-Change (web-dienend, kein Scope)

Routen brauchen ein Ziel-ID. Heute trägt `models.SellLocation` nur Namen:

```go
type SellLocation struct {
	StationName string `json:"station_name,omitempty"`
	SystemName  string `json:"system_name,omitempty"`
	IsStructure bool   `json:"is_structure"`
}
```

**Ergänzung:** ein Feld `LocationID int64 `json:"location_id,omitempty"``. Es wird
in `locResolver.resolve(ctx, locationID)` befüllt — die `locationID` liegt dort
bereits vor und ist genau die Station-/Struktur-ID des Buy-Orders (gültiges
`setWaypoint`-Ziel für NPC-Stationen **und** Citadels). Wird einmal vor dem
Cachen gesetzt, für beide Zweige (Station und Struktur).

Damit haben `raw_sell.location_id` und jede `material.sell.location_id` ein
Routen-Ziel. Die **Reprocess-Route** nutzt das bereits vorhandene
`OreRankRow.BestStationID` (`best_station_id`).

Kein anderer Backend-Change; keine neuen Endpunkte; keine `deployments`-Änderung.

---

## 4. Web-UI (`frontend/src/components/trading/OreRankingTable.tsx`)

Zwei geteilte Handler innerhalb der Tabelle, mit ROI-identischer Toast-Sprache
(`useToast`):

- `openMarket(typeId, name)`:
  - Erfolg → Toast „Markt-Detail an EVE gesendet" / „{name} — falls nichts
    passiert: Markt-Fenster im Spiel (Alt+R) öffnen und nochmal klicken."
  - Fehler → Toast „Fehler" / `err.message` (variant `destructive`).
- `setRoute(destId, label)`:
  - Erfolg → Toast „Route gesetzt" / „Waypoint in EVE gesetzt: {label}".
  - Fehler → Toast „Fehler" / `err.message` (variant `destructive`).

`openMarketDetails`/`setWaypoint` werfen schon sprechende Meldungen
(„EVE client not running", „Missing scope …").

**Platzierung:**
- **Erz-Summenzeile:** der Erzname wird ein klickbarer Button (Hover-Underline,
  `title` „Marktdetails im EVE-Client öffnen") → `openMarket(ore_type_id, ore_name)`.
  `onClick` ruft `e.stopPropagation()`, damit der Klick **nicht** die Zeile
  auf-/zuklappt.
- **Refine-Detail-Panel:**
  - Zeile „Aufbereiten bei: {Station} — {System}": die Station ist ein
    Routen-Link → `setRoute(best_station_id, "<Station> — <System>")`. Nur
    gezeigt, wenn `best_station_id` (>0) vorhanden.
  - Pro Mineral-Zeile: der **Mineral-Name** ist Markt-Link
    (`openMarket(material_type_id, material_name)`); der **Verkaufsort-Text** ist
    Routen-Link (`setRoute(material.sell.location_id, "<sell label>")`), nur wenn
    `material.sell.location_id` (>0) vorhanden.
- **Raw-Detail-Panel:**
  - Zeile „Roh verkaufen bei: {Ort}": der Ort ist Routen-Link
    (`setRoute(raw_sell.location_id, "<raw sell label>")`), nur wenn vorhanden.

Fehlt eine ID (Reprocess-Station 0, Citadel ohne auflösbares Ziel) → der
jeweilige Link wird nicht gerendert. Schlägt der ESI-Call fehl (EVE nicht offen,
Citadel ohne Zugriff) → Fehler-Toast, kein stiller No-Op.

**Auth:** Kein extra Login-Guard — ohne Login lädt die Mining-View keine Zeilen;
ein 401/403 landet als sprechender Fehler-Toast.

---

## 5. Frontend-Typ-Ergänzung

`frontend/src/types/trading.ts` → `SellLocation` bekommt `location_id?: number`.
Keine weiteren Typänderungen (`ore_type_id`, `material_type_id`, `best_station_id`
existieren bereits).

---

## 6. Tests

**Backend:**
- `mining_location_test.go`: `resolve` setzt `LocationID` auf die übergebene
  `locationID` — für NPC-Station und Citadel.
- Service-Wire-Test: eine `raw_sell`/`material.sell` trägt `location_id` im JSON.

**Web (vitest, `OreRankingTable.test.tsx`):**
- `@/lib/api-client` mocken (`openMarketDetails`, `setWaypoint`).
- Klick auf Erz-Name → `openMarketDetails(ore_type_id)` **und** Zeile bleibt
  eingeklappt (stopPropagation).
- Nach Aufklappen: Klick auf Mineral-Name → `openMarketDetails(material_type_id)`;
  Klick auf Reprocess-Station → `setWaypoint(best_station_id, {clearOtherWaypoints:true})`;
  Klick auf einen Verkaufsort → `setWaypoint(location_id, {clearOtherWaypoints:true})`.
- Fehlerfall (Mock wirft) → Fehler-Toast (`useToast` gemockt).

---

## 7. Nicht im Scope

- Flutter (bewusst ausgelassen).
- Neue ESI-Scopes / Backend-Endpunkte.
- Routen zum Refine-„Sell-Hub"-System als eigenes Ziel (die per-Mineral-
  Verkaufsorte decken das ab).

---

## 8. Betroffene Dateien

- `backend/internal/models/mining.go` — `SellLocation.LocationID`.
- `backend/internal/services/mining_location.go` — `resolve` befüllt `LocationID`.
- `backend/internal/services/mining_location_test.go` — Assertion.
- `frontend/src/types/trading.ts` — `SellLocation.location_id?`.
- `frontend/src/components/trading/OreRankingTable.tsx` — Handler + Links.
- `frontend/tests/components/OreRankingTable.test.tsx` — Tests.
