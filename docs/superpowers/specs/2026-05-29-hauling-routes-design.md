# Umkreis-Hauling Routes (Issue #45) — Design

- **Datum:** 2026-05-29
- **Issue:** Sternrassler/eve-o-provit#45 (siehe Scope-Update im Issue-Body)
- **Status:** In Umsetzung

## Use Case

Inter-Region-Arbitrage durch Transport, **direkt kaufen/verkaufen** (Taker, kein Order-Platzieren): ein Item günstig in einer Station kaufen (aus deren Sell-Orders), in eine andere Station transportieren und dort teurer verkaufen (in deren Buy-Orders). Gewinn = Preisdifferenz − Transportkosten (Zeit/Risiko).

**Scope (abgestimmt):** „Umkreis-Hauling". Origin = die **aktuelle Region** des Characters (automatisch); Zielmenge = Origin **+ alle angrenzenden Regionen** (1 Hop). **Stationsgenau** (kein Hub-Zwang), **any→any** innerhalb des Sets. Bounded — kein globaler Markt-Ingest.

## Eingaben

- **Ship** (`ship_type_id` → effektives Cargo via Skills) — Pflicht.
- **Kapital** (`capital`, ISK-Budget) — Pflicht (zweiter Constraint neben Cargo).
- **Low-Sec meiden** (`avoid_low_sec`).
- **Origin-Region** (`origin_region_id`, optional) — Default = aktuelle Region aus `GetCharacterLocation`.
- **max_routes** (optional, Default 15).
- **Skills** automatisch (Cargo via Industrial/Hauler, Fees via Accounting/Broker).

## Zielfunktion / Modell

Pro **Route** = (Kauf-Station → Verkauf-Station) die **optimale Ladung** für **einen** Trip: Cargo füllen mit dem Item-Mix, der den Gewinn maximiert, unter den Constraints **Cargo-m³**, **Kapital-Budget** und **Liquidität** (verfügbare Menge je Item = min(Kauf-Angebot, Verkauf-Nachfrage)). Routen nach **ISK/h** (Routen-Netto-Gewinn / Round-Trip-Fahrtzeit) ranken.

## Architektur

### Backend
- **Regions-Adjazenz (neu):** `SDERepository.GetNeighborRegions(ctx, regionID) ([]int, error)` — aus `v_stargate_graph` + `mapSolarSystems.regionID`: Regionen, deren System per Stargate mit einem System der Origin-Region verbunden ist. Verifiziert: ⌀ ~4.8, max 10 Nachbarn.
- **`HaulingService`** `internal/services/hauling_service.go`:
  1. Origin-Region bestimmen (Param oder `GetCharacterLocation` → System → Region via SDE).
  2. `regions = {origin} ∪ GetNeighborRegions(origin)`.
  3. Pro Region Orderbuch laden (Wiederverwendung `RouteFinder.fetchMarketOrders(regionID)` — ESI-Batch + Redis-Cache + Upsert).
  4. **Cross-Region-Matching:** pro Item-Typ über alle Stationen des Sets den günstigsten aktiven **Sell-Order** (Kauf-Quelle: Station/System/Preis) und den höchsten aktiven **Buy-Order** (Verkauf-Ziel) bestimmen; profitabel, wenn an **verschiedenen** Stationen und `sellTargetBuyPrice > buySourceSellPrice` nach Fees. Verfügbare Menge = min(VolumeRemain Kauf, VolumeRemain Verkauf).
  5. **Routen bilden:** Item-Chancen nach `(Kauf-Station → Verkauf-Station)` gruppieren.
  6. Pro Route **optimale Ladung** packen — `HaulingOptimizer` (greedy nach Gewinn/m³, fügt Einheiten hinzu solange Cargo-m³ **und** Restkapital **und** Item-Liquidität reichen). Baut auf der `CargoService.KnapsackDP`-Logik auf, um die **Kapital-Dimension** erweitert.
  7. Pro Route Reisezeit + Sprünge + min-Sec via `navigation.ShortestPath(buySystem, sellSystem, avoidLowSec)` / `CalculateTravelTime` + `getMinRouteSecurityStatus`; `avoid_low_sec` filtert Routen mit Low/Null-Sprüngen. `security_risk` ∈ {safe, caution, danger} aus min-Sec. ISK/h = Routen-Netto / Round-Trip-Minuten.
  8. Nach ISK/h ranken, auf `max_routes` begrenzen. Skills einmal für `skills_applied`.
- **Fees:** Netto-Gewinn je Item via Sales-Tax + (Modell: Sofort-Arbitrage → Sales-Tax beim Verkauf; Broker-Fees wie im bestehenden Routen-Modell). Wiederverwendung `FeeService`/Rate-Helper.
- **Models** `internal/models/hauling.go`: `HaulingRequest`, `HaulingRoute`, `HaulingItem`, `HaulingResponse`.
- **Handler** `internal/handlers/hauling.go` → `POST /api/v1/trading/hauling/routes` (protected, rate-limited); Route in `cmd/api/main.go`.
- **Testbarkeit:** Cross-Region-Matching + Routen-Gruppierung + `HaulingOptimizer` arbeiten über reine Eingaben (Order-Listen / Candidate-Slices), damit unit-testbar ohne ESI/DB.

### Web (`frontend/src/app/hauling/page.tsx`, neu + Nav-Eintrag)
Eingabe-Formular (Ship, Kapital, Low-Sec-meiden; Origin-Region angezeigt, aus Character-Location) → `useMutation` auf `/hauling/routes`. Ergebnis: **Routen-Liste** (Kauf→Verkauf-System, Jumps, Reisezeit, Sec-Badge safe/caution/danger, ISK/h, Gesamtgewinn, Cargo-Füllstand), je Route aufklappbar zur **Ladungs-Tabelle** (Item, Menge, Kauf@Station, Verkauf@Station, Volumen, Gewinn, Gewinn/m³) + **„Route an EVE übertragen"-Button** (Wiederverwendung `setWaypoint`-Helper: clear+Kauf-Station, dann Verkauf-Station). Empty-State „Keine profitablen Routen im Umkreis".

### Flutter (`app/lib/features/trading/hauling_screen.dart`, neu + Route + Nav)
Adaptiver Screen via `isTwoPane(840)`: Eingaben + Routen-Liste, Route-Detail mit Ladungs-Tabelle + Waypoint-Button (Wiederverwendung `route_detail`-Pattern). DTOs `hauling_models.dart` (null/float-robust), Provider `hauling_providers.dart` (`AsyncNotifier`), `trading_api.findHaulingRoutes(...)`.

## Datenfluss

`Eingaben → POST /hauling/routes → Origin (CharacterLocation) → GetNeighborRegions → fetchMarketOrders je Region → Cross-Region-Matching je Item → Gruppierung nach (Kauf→Verkauf-Station) → HaulingOptimizer (Cargo+Kapital) je Route → Navigation (Zeit/Sec) → Ranking → Response`.

## Error-Handling
- Auth-pflichtig; Skills fehlen → Base-Werte (`skills_applied=false`).
- Keine profitablen Routen / leeres Set → leere Liste mit Hinweis.
- Region ohne Marktdaten → übersprungen (kein Hard-Fail). Keine geleakten Roh-Fehler. Mobile-DTO null/float-robust.
- Cold-Cache: Lade-Anzeige; Region-Fetch-Fehler einzeln tolerieren.

## Tests (Pyramide)
- **Backend Unit:** `hauling_service_test.go` — Cross-Region-Matching (bester Kauf/Verkauf je Item über Stationen), Routen-Gruppierung, `HaulingOptimizer` (respektiert Cargo-m³ **und** Kapital **und** Liquidität; ISK/h-Ranking; avoid-low-sec filtert). Reine Eingaben, keine ESI/DB.
- **Backend Integration** (`//go:build integration`): `GetNeighborRegions` gegen echte SDE (z.B. The Forge → 8 Nachbarn).
- **Web:** Vitest (Routen-Liste + Ladungs-Tabelle + Sec-Badge + Waypoint-Button-Call) + Playwright authed E2E.
- **Flutter:** Unit (Parsing), Widget (adaptiv 1-/2-Pane + Route-Detail), E2E-Gruppe (Provider-Override).

## Deployment
Backend tag-basiert (v0.x → GHCR → deploy-app.sh) + Frontend-Image (same-origin). Keine Migration nötig (read-only/berechnend). Flutter APK + On-Device-Happy-Path.

## Out of Scope (YAGNI)
- Global „alle Regionen überall" (eigenes Daten-Pipeline-Projekt).
- >1 Hop Reichweite (später konfigurierbar).
- Order-Platzieren / Market-Maker (#46).
- Multi-Trip-Logistik über mehrere Tage (ISK/h pro Route reicht).
