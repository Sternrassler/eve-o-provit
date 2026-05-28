# ROI Calculator & Capital Allocation Optimizer (Issue #44) — Design

- **Datum:** 2026-05-28
- **Issue:** Sternrassler/eve-o-provit#44
- **Status:** In Umsetzung

## Use Case

Ein Trader hat ein Budget (z.B. 500 Mio. ISK) und begrenzte Spielzeit. Frage: **Welche Items kaufe ich in welcher Stückzahl, um meinen Tagesgewinn zu maximieren?** #44 ist eine **Portfolio-/Kapital-Allokations-Schicht** auf der bestehenden Routen-Engine — nicht *wo* handle ich ein Item (#43), sondern *was und wie viel* mit meinem Kapital.

Modell-Scope (wie das bestehende Trading): **eine Region, Station→Station-Arbitrage** (an Station A aus Sell-Orders kaufen, haulen, an Station B in Buy-Orders verkaufen), Cargo- und reisezeit-begrenzt, skill-bereinigt.

## Eingaben

- **Region** (`region_id`)
- **Schiff** (`ship_type_id` → effektives Cargo via Skills, Warp/Align für Reisezeit)
- **Kapital** (verfügbares ISK-Budget)
- **Verfügbare Zeit** (`time_budget_min` — hartes Fahrten/Tag-Budget)
- **Liquiditäts-Cap** (`liquidity_cap_pct` — max. Anteil am Tages-Volumen, den man realistisch handelt)
- **Sec-Zonen-Filter** (High/Low/Null)
- **Max. % Kapital pro Item** (`max_item_pct` — Diversifikations-Constraint)
- **Skills** — automatisch aus dem Login (Accounting/Broker → Fees, Industrial/Hauler → Cargo, Navigation → Fahrten/Tag)

## Zielfunktion

**Maximiere erwarteten Gesamt-Tagesgewinn** unter den Constraints: Kapital-Budget, Zeit-Budget (Fahrten/Tag), Cargo/Fahrt, Liquiditäts-Cap pro Item, Max-% Kapital pro Item. Zeit geht als hartes Constraint ein (langsame Routen → weniger Fahrten/Tag), Kapital ist die Allokations-Achse. „Profit je Zeit" ist das implizite Effizienzmaß für die Auswahl.

## Architektur

### Backend
- **Kandidaten-Generierung (Wiederverwendung):** `RouteService.CalculateWithFilters(region, ship, sec-zones, volume)` liefert die profitablen Station→Station-Routen der Region mit Netto-Gewinn, Cargo, Fahrten, ISK/h und Volumen-Metriken (skill-bereinigt). Kein neuer Markt-/Trading-Code.
- **`PortfolioOptimizerService`** `internal/services/portfolio_service.go` — greedy 2-Ressourcen-Allokation (Kapital + Zeit):
  1. Pro Kandidat realistische Kapazität: `tripsPerDay = min(reisezeit-erlaubte Fahrten, Zeit-Budget)`; `maxUnitsPerDay = liquidity_cap_pct × daily_volume`; `cargo/Fahrt`; `perItemCapital = max_item_pct × capital`.
  2. **Effizienz** = erwarteter Tagesgewinn pro eingesetztem ISK (unter obigen Caps).
  3. **Greedy:** nach Effizienz sortiert Kapital + geteiltes Zeit-Budget auffüllen, bis Kapital oder Zeit erschöpft; Per-Item-Kapital-Cap erzwingt Streuung.
  4. Ergebnis je Item `{type_id, name, capital_used, units, trips_per_day, daily_profit, roi_pct}`; Totals `{total_capital_used, total_daily_profit, diversification_score, time_used_min}`.
  - **Diversifikations-Score:** aus Kapital-Konzentration, Herfindahl-artig — `score = round((1 − Σ(anteil_i²)) × 100)` (0 = alles in 1 Item, 100 = maximal gestreut).
- **Models** `internal/models/portfolio.go` — `PortfolioRequest`, `PortfolioResult`, `PortfolioItem`.
- **Handler** `internal/handlers/portfolio.go` → `POST /api/v1/trading/portfolio/optimize` (protected, rate-limited), Route in `cmd/api/main.go`.
- **Testbarkeit:** Optimizer bekommt die Kandidaten + Constraints als reine Eingaben (Interface für die Routen-Quelle), damit die Greedy-Logik ohne echte Engine unit-testbar ist.

### Web (`frontend/src/app/roi-calculator/page.tsx`)
Placeholder ersetzen. Eingabe-Formular (Region, Schiff, Kapital, Zeit, Liquiditäts-Cap, Max-%/Item, Sec-Zonen — re-use `RegionSelect`/`ShipSelect`/Filter-Muster) → `useMutation` auf `/portfolio/optimize`. Ergebnis: **ROI-Ranking-Tabelle** + **Allokations-Tabelle** (Item, Kapital, Units, Tagesgewinn, Fahrten/Tag, ROI%) + **Gesamt-Tagesgewinn** + **Diversifikations-Score** + Skills-Applied-Panel. Komponenten unter `components/trading/`.

### Flutter (`app/lib/features/trading/roi_calculator_screen.dart`)
Adaptiver Screen via `isTwoPane(840)`: Eingaben links/oben, Ergebnis-Tabelle rechts/unten. DTOs in `api/portfolio_models.dart` (null/float-robust), Provider `roi_providers.dart` (`AsyncNotifier`), `trading_api.optimizePortfolio(...)`, Route `/roi-calculator` + Nav-Destination.

## Datenfluss

`Eingaben (UI) → POST /portfolio/optimize → RouteService.CalculateWithFilters (Kandidaten) → PortfolioOptimizerService (greedy Allokation unter Kapital+Zeit+Liquidität+Per-Item-Cap) → Result → Ranking + Allokation + Totals`.

## Error-Handling
- Auth-pflichtig; fehlende Skills → Base-Werte (kein Hard-Fail), `skills_applied=false`.
- Keine Kandidaten / Kapital zu klein für günstigstes Item → leeres Portfolio mit Hinweis „kein sinnvolles Portfolio für dieses Budget".
- Keine geleakten Roh-Fehler (Handler-Konvention). Mobile-DTO null/float-robust.

## Tests (Pyramide)
- **Backend Unit:** `portfolio_service_test.go` — greedy respektiert Kapital-/Zeit-/Liquiditäts-/Per-Item-Cap; Diversifikations-Score (1 Item → ~0, gleichverteilt → hoch); leeres Budget → leeres Portfolio.
- **Web:** Vitest (Allokations-/Ranking-Tabelle, Diversifikations-Score-Anzeige) + Playwright authed E2E (`tests/e2e/auth/roi-calculator.spec.ts`).
- **Flutter:** Unit (Parsing) + Widget (adaptiv 1-/2-Pane) + E2E-Gruppe in `test/e2e/app_flow_test.dart` (Provider-Override).

## Deployment
Backend tag-basiert (v0.x → GHCR → deploy-app.sh) + Frontend-Image (same-origin, bereits in der Pipeline). Migrationen werden beim Boot angewandt (keine neue nötig — #44 ist read-only/berechnend). Flutter: APK-Build + On-Device-Happy-Path.

## Out of Scope (YAGNI)
- Exakter ILP-Solver (greedy reicht, erklärbar).
- Multi-Region / Inter-Hub-Hauling (#45).
- Persistente Portfolios/Watchlist (#48).
- Eigener Markt-Fetch (Routen-Engine wird wiederverwendet).
