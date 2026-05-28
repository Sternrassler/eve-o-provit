# Multi-Hub Comparison (Issue #43) — Design

- **Datum:** 2026-05-28
- **Issue:** Sternrassler/eve-o-provit#43
- **Status:** In Umsetzung

## Use Case

Ein kleiner Trader wählt ein Item und sieht für **denselben Artikel** einen Side-by-Side-Vergleich über die großen Handels-Hubs (Jita, Dodixie, Amarr, Rens, Hek): pro Hub Buy-Preis, Sell-Preis, Spread%, **skill-korrigierte Netto-Marge**, Liquidität/Volumen und ein **Konkurrenz-Indikator**. Eine Empfehlungs-Engine markiert den besten Hub. Ziel: Jita-Überfüllung meiden, profitablere Sekundär-Hubs finden — mit den Character-Skills (Fees, Cargo) korrekt eingerechnet.

Abgrenzung: #43 ist **handels-getrieben** (generisch gewähltes Item, Station-Trading-Marge je Hub). #107 (Bestand verkaufen) ist bestands-getrieben und **wiederverwendet** die hier gebaute Hub-Ranking-Logik (siehe Design-Note in #43).

## Scope

Backend (Go) + Web-Frontend (Next.js) + Flutter-Client + volle Testpyramide (Unit/Integration/E2E je Frontend) + Deployment (Backend tag-basiert; Flutter APK + On-Device-Happy-Path).

## Entscheidungen

1. **Pro Hub anzeigen:** Buy-Preis, Sell-Preis, Spread%, **Netto-Marge nach Fees/Skills** (Round-Trip Station-Trading: Buy-Order kaufen → Sell-Order verkaufen, minus Sales-Tax + Broker-Fee).
2. **Konkurrenz = A+C kombiniert:**
   - **C (Sofort-Baseline):** ESI-Market-History `order_count`/Tag (vom `VolumeService` ohnehin geholt) → sofortiger Tages-Indikator, kein Cold-Start-Leerzustand.
   - **A (Live-Upgrade):** Lazy-Tracking — beim ersten Vergleich werden `(type, 5 Hubs)` registriert; ein Scheduler snapshottet die Orders der getrackten Paare periodisch und akkumuliert Änderungs-Events/Stunde (neu/entfernt/umpreist). Sobald genug Snapshots da sind, ersetzt der Live-Churn die Baseline.
   - Response-Feld `competition.source ∈ {baseline, live}` + `score`. C bleibt dauerhafter Fallback.
3. **Hubs (fix, 5 Major-Hubs):** Jita/The Forge, Dodixie/Sinq Laison, Amarr/Domain, Rens/Heimatar, Hek/Metropolis — als Konstanten-Registry (Region-ID + Station-ID + primary/secondary-Flag).
4. **Reusable Service:** `MultiHubComparisonService` komponiert die vorhandenen Services (Fee/Skills/Cargo/Volume/Market) — getrennt von HTTP-Handler/UI, damit #107 ihn wiederverwendet (siehe #43 Design-Note).
5. **Skills:** Fees via `FeeService` (Accounting/Broker), Cargo via `cargo.GetShipCapacities` — auf **alle** Hub-Zeilen angewandt; ein „Skills Applied"-Panel zeigt die Effekte.

## Architektur

### Backend

Neue/erweiterte Bausteine (alle unter `backend/`):

- **Hub-Registry** `internal/services/hubs.go` — Konstanten der 5 Hubs (region_id, station_id, name, tier).
- **Service** `internal/services/multi_hub_service.go` — `MultiHubComparisonService` mit
  `CompareHubs(ctx, typeID, characterID, accessToken) (*MultiHubComparisonResult, error)`.
  Komponiert: `SkillsService` (einmal, gecached), `FeeService.CalculateFees` (je Hub), `VolumeService.GetVolumeMetrics` (je Hub-Region), Market-Orders (je Hub-Region, best buy/best sell), `CompetitionService` (Score je Hub). Ranking nach Netto-Marge (Tie-Break: Netto-Tagesprofit via Volumen). Setzt `BestHub`.
- **Competition** `internal/services/competition_service.go` + Collector:
  - `competition_collector.go` — periodischer Job (Ticker, im `main.go` gestartet wie bestehende Background-Worker), iteriert getrackte `(type, hub)`-Paare, fetcht Orders, difft gegen letzten Snapshot, schreibt Churn-Metrik.
  - Lazy-Registrierung: `CompareHubs` registriert die Paare (`upsert tracked`).
  - Baseline aus Market-History `order_count` (C) wenn noch keine Live-Daten.
- **Migration** `migrations/000002_competition_tracking.{up,down}.sql`:
  - `competition_tracked(type_id, region_id, last_seen, PRIMARY KEY(type_id, region_id))`
  - `competition_snapshot(type_id, region_id, taken_at, order_fingerprint JSONB)` (rolling, geprunt)
  - `competition_metric(type_id, region_id, changes_per_hour, window_start, window_end, source, updated_at, PRIMARY KEY(type_id, region_id))`
- **Repository** `internal/database/competition_repository.go` — Upserts/Reads für die drei Tabellen (pgx, wie `MarketRepository`).
- **Handler** `internal/handlers/trading.go` → neue Methode `CompareHubs` (analog `CalculateRoutes`), Route `POST /api/v1/trading/hubs/compare` in `cmd/api/main.go` (protected, rate-limited).
- **Models** `internal/models/hub_comparison.go` — `HubComparisonRequest{TypeID}`, `HubComparisonResult{ItemTypeID, ItemName, Hubs[], BestHubRegionID, SkillsApplied}`, `HubRow{RegionID, RegionName, StationName, Tier, BuyPrice, SellPrice, SpreadPercent, NetMarginPercent, NetProfitPerUnit, Volume *VolumeMetrics, Competition{Score, Source}}`.

### Web (Next.js)

- `frontend/src/app/multi-hub/page.tsx` — Placeholder ersetzen: Item-Suche (re-use `GET /api/v1/items/search`) → bei Auswahl `POST /hubs/compare` (React-Query `useMutation`, `credentials:"include"`).
- Komponenten `frontend/src/components/trading/`: `HubComparisonTable.tsx`, `RecommendedHubBadge.tsx`, `SkillsAppliedPanel.tsx`, `CompetitionIndicator.tsx` (zeigt baseline/live).
- `frontend/src/lib/api-client.ts` → `compareHubs(typeId)`, `searchItems(q)` (falls nicht vorhanden).
- `frontend/src/types/trading.ts` → `HubComparisonResult`, `HubRow`, `CompetitionInfo`.

### Flutter

- `app/lib/api/hub_comparison_models.dart` — DTOs (null/float-robust `fromJson`).
- `app/lib/features/trading/hub_comparison_screen.dart` — adaptiv via `isTwoPane(840)`: Item-Suchfeld + Ergebnis-Tabelle (1-Pane gestapelt / 2-Pane nebeneinander).
- `app/lib/features/trading/hub_comparison_providers.dart` — `AsyncNotifier` (item → result).
- `app/lib/api/trading_api.dart` → `compareHubs(typeId)`, `searchItems(q)`.
- `app/lib/core/router.dart` → Route `/hub-comparison` + 4. NavigationDestination.

## Datenfluss

`Item-Suche (UI) → /items/search → Auswahl typeId → POST /hubs/compare → MultiHubComparisonService: Skills(1×) → für jeden Hub [Market best buy/sell, Fees, Volume, Competition] → Ranking → BestHub → Response → Tabelle + Empfehlung`.

## Error-Handling

- Auth-Pflicht (wie Trading); fehlender Skill-Scope → Fallback auf Base-Fees (kein Hard-Fail), `SkillsApplied=false`.
- Hub ohne Orders/Volumen → Zeile mit Hinweis „keine Daten", nicht aus dem Vergleich kippen.
- Keine geleakten Roh-Fehler (bestehende Konvention aus Handler-Tests).
- Mobile-DTO null/float-robust (bekannte Lesson).

## Tests (Pyramide)

- **Backend Unit:** `multi_hub_service_test.go` (Ranking, Spread/Marge-Rechnung, Best-Hub-Wahl, baseline-vs-live-Auswahl) mit gemockten Servicern; `competition_service_test.go` (Churn-Diff aus zwei Snapshots).
- **Backend Integration** (`//go:build integration`): `multi_hub_integration_test.go` — echte SDE + Markt-Migrationen, mehrere Hub-Regionen; Collector-Diff gegen DB.
- **Web Unit (Vitest):** `HubComparisonTable.test.tsx`, `CompetitionIndicator.test.tsx`.
- **Web E2E (Playwright, authed):** `tests/e2e/auth/multi-hub.spec.ts` — Login-Session, Item suchen, Vergleich anzeigen, Empfehlung sichtbar.
- **Flutter Unit/Widget:** `test/hub_comparison_models_test.dart` (Parsing), `test/hub_comparison_screen_layout_test.dart` (1-Pane/2-Pane via View-Size).
- **Flutter E2E:** Gruppe in `test/e2e/app_flow_test.dart` mit Provider-Override (fake result) — Tabelle + Empfehlung gerendert.

## Deployment

- **Backend:** SemVer-Bump in `CHANGELOG.md`, Migration `000002`, `make release VERSION=x.y.z` → Tag `v*` → CI (lint/test/blocking-govulncheck) → GHCR → `deploy-app.sh eveoprovit` → Smoke. Migration läuft beim Start (`migrate-up`/Startup-Apply).
- **Flutter:** `flutter build apk --release --dart-define=API_BASE_URL=https://eveonline.sternrassler.de ...`, On-Device-Happy-Path auf dem Galaxy Tab gegen Prod (Item suchen → Hub-Vergleich → Empfehlung).

## Out of Scope

- #107 (Bestand verkaufen) — baut hierauf auf, separat.
- Persistente Langzeit-Konkurrenz-Historie/Charts (#47).
- Hub-Set konfigurierbar machen (vorerst fixe 5).
