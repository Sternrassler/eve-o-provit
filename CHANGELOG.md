# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.17.2] - 2026-05-31

### Changed

- **eve-esi-client auf v0.5.0 angehoben (#150).** Backend-Dependency von der alten gepinnten Pseudo-Version (`v0.2.1-…df66ffe`, Nov 2025) auf den getaggten Release v0.5.0 — bringt fail-loud Fehlerbehandlung in der ESI-Lib (GetBody-Retry, Cache-TTL-Parse, Ratelimit-Teilread, Close-Errors; eve-esi-client #27) sowie die Entfernung des Prometheus-Metrics-Features und des esi-proxy-cmd.

### Fixed

- **Pre-Commit-Hook blockierte nicht mehr jeden Backend-Commit.** `golangci-lint` v2 konnte die v1-formatige `backend/.golangci.yml` (ohne `version:`-Key) nicht laden und hard-failte; der Schritt wird jetzt via `golangci-lint config verify` sauber übersprungen (go vet läuft weiter; CI lintet über gofmt+vet). Self-healing nach einer v2-Migration der Config.

## [0.17.1] - 2026-05-31

### Fixed

- **Keine stillen Fallbacks mehr, die echte Fehler verschleiern (#147).** Audit über Frontend + Backend: degradierte Ersatzwerte, die einen Fehler überdeckten, werden jetzt sichtbar gemacht (fail-loud) statt als korrekt ausgegeben.
  - **Ship-Dropdown:** zeigt wieder das gefittete (effektive) Volumen statt „Basis" für alle Schiffe; der Client hatte `effective_cargo_capacity` verworfen. Schlägt die Fitting-Anreicherung mit einem Fehler fehl, steht jetzt „fitted unbekannt" statt stillschweigend das Basis-Volumen.
  - **Sicherheitsstatus:** ein fehlgeschlagener Sec-Status-Lookup klassifiziert eine Route nicht mehr fälschlich als High-Sec (Default 1.0) — die Route wird übersprungen statt als sicher angezeigt.
  - **Wallet/Skills:** ein fehlgeschlagener Wallet-Fetch zeigt eine sichtbare Warnung (Kapital ist Platzhalter, nicht das echte Guthaben) statt still 500M anzunehmen; der Skills-Endpoint liefert bei ESI-Fehler HTTP 500 statt 200 mit Null-Skills, das UI meldet den Fehler.
  - **Regionen/Schiffe:** bei Ladefehler keine Mock-Liste mehr als echte Daten — sichtbarer Fehlerzustand.
  - **Backend:** Cargo-Capacity-, Fitting-, Skills- und Tax-Rate-Fehler werden propagiert statt fabrizierter Werte (0, Default-Fitting, 5,5 %); malformte API-Antworten (`regions`/`ships`/`items`/`routes`) werfen jetzt statt leerer Listen.

## [0.17.0] - 2026-05-30

### Added

- **ROI zeigt + sortiert nach ISK/h.** `PortfolioItem.isk_per_hour` neu (= `daily_profit / (trips_per_day × trip_minutes / 60)`). Outcome wird nach ISK/h sortiert (Tiebreak Tagesgewinn); Allokation läuft weiterhin nach Capital-Efficiency, nur die Anzeige ändert sich. Web + Flutter zeigen neue Spalte „ISK/h".

### Verified

- **ROI cappt am Sell-Side-Limit.** Bereits seit v0.15.3 cappt `Candidate.MaxAvailableUnits` (= `route_finder`s `min(cheapest-sell-VolumeRemain, highest-buy-VolumeRemain)`) die empfohlene Stückzahl — das heißt die Allokation überschreitet nie das, was der höchste Buy-Order tatsächlich abnimmt. Neuer Regressionstest `TestOptimize_RespectsSellSideAvailability` prüft den Sell-Side-Fall isoliert.

## [0.16.2] - 2026-05-30

### Added

- **Sell-Assets zeigt den ESI-Cache-Countdown.** Reporter: Item längst verkauft, taucht aber nach Refresh weiter in der Liste auf — Ursache ist der ESI-Server-Cache (~1 h TTL). Backend liest jetzt den `Expires`-Header aus jeder Asset-Antwort und reicht ihn als `cache_expires_at` durch. Web + Flutter zeigen unter dem Suchfeld einen Live-Countdown („ESI-Cache läuft bis HH:MM — Refresh vorher zeigt denselben Stand"), der nach Ablauf zu „Refresh holt jetzt frische Daten" wechselt. Auto-Refresh alle 30 s.

## [0.16.1] - 2026-05-30

### Added

- **Sell-Assets: Refresh-Button für die Asset-Liste.** Nach Items-Verschieben im Spiel möchte man die Liste manuell neu laden statt auf einen App-Reload zu warten. Web (Icon neben „Asset suchen") + Flutter (Icon rechts vom Suchfeld); zeigt Spinner während des Ladens. ESI cached Assets serverseitig mit ~1 h TTL — ein Refresh kann also trotzdem den alten Stand zeigen, wenn der EVE-Cache noch nicht abgelaufen ist.

## [0.16.0] - 2026-05-30

### Added

- **Sell-Assets sortiert nach ISK/h und zeigt den Wert pro Verkaufsort.** Bisher wurden Optionen nur nach Netto-Total sortiert — nahe Hubs sahen damit gleichwertig zu fernen Hubs aus, obwohl die Reisezeit den effektiven Stundenlohn massiv verändert (z.B. Dodixie 6 Sprünge vs. Amarr 37 Sprünge bei gleichem Netto). Neue `SellOption.isk_per_hour` (net ISK pro Reise-Stunde, 0 bei lokalen Verkäufen). Ranking: lokale Verkäufe zuerst (instant), dann Remotes nach ISK/h, Tiebreak Netto-Total. Web + Flutter zeigen den Wert als zusätzliche Metric („sofort" bei lokalem Verkauf).

## [0.15.5] - 2026-05-30

### Fixed

- **Sell-Assets: Items in Containern innerhalb NPC-Stationen werden jetzt korrekt geroutet.** ESI gibt für ein Item in einem Container `location_id` = Container-ItemID (≥ 1×10¹², sieht wie Citadel-ID aus), nicht die Station-ID — daher rutschte der Fall in den Citadel-Pfad und wurde mit „verlagere in NPC-Station" abgewiesen, obwohl der Container ja bereits in einer NPC-Station lag. Backend baut jetzt aus der Asset-Liste eine `ItemID→LocationID`-Map und läuft die Container-Kette bottom-up (max. 8 Hops); sobald ein Hop SDE-resolvable ist, wird das System verwendet. Echte Citadels (Kette endet ohne Treffer) zeigen weiter den `origin_in_player_structure`-Hinweis. Regressionstest deckt beide Pfade ab.

## [0.15.4] - 2026-05-30

### Fixed

- **Sell-Assets erklärt jetzt, warum Citadel-Items keine Verkaufsorte ergeben.** Wenn der Bestand in einer Player-Structure liegt, kann das Backend keinen Origin-System auflösen (SDE kennt nur NPC-Stationen) → früh-Return mit leerer Optionsliste. Bisher zeigte das UI nur „Keine Verkaufsorte gefunden" — ununterscheidbar von „wirklich keine Käufer". Backend setzt jetzt `not_routable_reason: "origin_in_player_structure"`; Web + Flutter rendern eine konkrete Empfehlung („Items liegen in einer Player-Structure — verlagere den Bestand in eine NPC-Station").

## [0.15.3] - 2026-05-30

### Fixed

- **ROI: empfohlene Stückzahl wird jetzt vom Order-Book-Limit gecappt.** Der Optimizer extrapolierte den Cheapest-Tier-Preis über Mengen, die zu dem Preis gar nicht existierten — z.B. 154 Carbon zu 85 ISK, obwohl der EVE-Markt nur 2 zu diesem Preis hat (nächster Tier 332,51 ISK). `Candidate.MaxAvailableUnits` wird jetzt aus `TradingRoute.Quantity` (route_finder's `AvailableQuantity = min(cheapest-sell-VolumeRemain, highest-buy-VolumeRemain)`) befüllt und cappt `maxUnits` zusätzlich zu den bestehenden Liquiditäts- und Kapital-Caps. Regressionstest deckt den Fall ab.

## [0.15.2] - 2026-05-30

### Fixed

- **ROI-Item-Klick erklärt jetzt das Photon-UI-Verhalten.** Bei ESI-Markt-Open gibt der EVE-Client 204 zurück, öffnet aber kein Fenster, wenn keines bereits offen ist (bekannter Photon-UI-Bug, [esi-issues #1349](https://github.com/esi/esi-issues/issues/1349)). Toast formuliert das jetzt aus („Falls nichts passiert: Alt+R, dann nochmal klicken"); zusätzlich ein statischer Tipp über der ROI-Allokations-Tabelle.

## [0.15.1] - 2026-05-30

### Removed

- **Flutter: alle Spiele-Aktions-Buttons entfernt.** Die Tablet-App ist Planungs-/Analyse-Client, keine Spiele-Fernbedienung. „Route an EVE übertragen" (ROI/Hauling/Route-Detail/Sell-Assets) und der ROI-Item-Tap → Marktdetails fliegen raus; die Scopes `esi-ui.write_waypoint.v1` und `esi-ui.open_window.v1` werden vom Flutter-Client nicht mehr angefragt. Backend-Endpoints + Web nutzen sie unverändert weiter.

## [0.15.0] - 2026-05-30

### Added

- **ROI: Klick/Tap auf den Item-Namen öffnet das Markt-Fenster im EVE-Client.** Neuer Endpoint `POST /api/v1/esi/ui/openwindow/marketdetails` (analog zur Waypoint-API). Web + Flutter zeigen Toast/SnackBar bei Erfolg oder Fehler.

### Changed

- **SSO fragt zusätzlich `esi-ui.open_window.v1` an** (Web + Mobile). Der Scope muss in beiden EVE-App-Registrierungen aktiv sein (vom Owner vorab erledigt); bestehende Sessions müssen neu autorisieren.

## [0.14.1] - 2026-05-30

### Fixed

- **Character-Screen (Flutter) zeigt für das aktive Schiff den effektiven Laderaum.** Folge-Patch zu v0.14.0: Die „Laderaum"-Zeile auf dem Character-Screen blieb auf der Basis-Hülle stehen, während die Dropdowns bereits den Effektiv-Wert aus dem Fitting nutzen. Jetzt konsistent — Effektiv wenn vorhanden, sonst Basis-Fallback („X m³ (Basis)").

## [0.14.0] - 2026-05-30

### Added

- **Schiff-Dropdown zeigt jetzt die effektive Cargo-Kapazität** — denselben Wert, mit dem der Optimizer seit v0.12.4 rechnet und den EVE in-game anzeigt (z.B. ~9,6k m³ für eine Nereus mit Cargo Expandern + Skills statt 2,7k Basis). `/character/ships` und `/character/ship` werden mit `effective_cargo_capacity` pro Schiff angereichert (`FittingService.GetShipFitting`, parallel mit Concurrency-Cap 4, 5 Min Redis-Cache). Erste Dropdown-Öffnung kostet ~2–5 s je Hangar-Größe, danach instant. Best-Effort: bei Fitting-Fehler weiter „Basis Xk m³" als Fallback.

## [0.13.3] - 2026-05-30

### Fixed

- **Schiff-Dropdown-Label sagt jetzt „Basis", damit es nicht als Rechenwert missverstanden wird.** Das Label zeigt den Basis-Hull-Cargo (z.B. „Nereus (2,7k m³)") aus `/character/ships`; die Routen-/ROI-Rechnung nutzt seit v0.12.4 aber die echte effektive Cargo-Kapazität aus dem Fitting (z.B. ~9,6k m³ inkl. Cargo Expander + Skills). Label nun „Basis 2,7k m³" (Web + Flutter) — inhaltlich ändert sich nichts am Rechner.

## [0.13.2] - 2026-05-30

### Fixed

- **Sell-Assets crasht nicht mehr, wenn das Item in einer Player-Structure/Citadel liegt.** Bei unauflösbarem Origin (SDE kennt nur NPC-Stationen) blieb `Options` ein Go-nil-Slice → JSON `null` → Web crashte mit „Cannot read properties of null (reading 'length')". Backend initialisiert `Options` jetzt immer als leeres Array; Web ist zusätzlich gegen `null` defensiv. UX-Verhalten: leerer Empty-State „Keine Verkaufsorte gefunden" (dokumentierte Citadel-Limitierung bleibt).

## [0.13.1] - 2026-05-30

### Removed

- **Veraltete „Phase 2"-/„Phase 3"-Badges aus der Web-Navigation entfernt.** Multi-Hub, ROI Calculator, Hauling und Sell Assets sind seit v0.9.0–v0.13.0 live. Phase-3-Badges (Trends, Watchlist) ebenfalls raus; die Pages haben sichtbare „Coming in Phase 3"-Platzhalter, der Badge war redundant.

## [0.13.0] - 2026-05-29

### Added

- **ROI-Kapital wird mit dem echten Wallet-Kontostand vorbefüllt.** Neuer Endpoint `GET /api/v1/character/wallet` (ESI `/characters/{id}/wallet/`, 60 s gecacht) liefert den ISK-Stand; ROI-Rechner (Web + Flutter) befüllt das Kapital-Feld damit vor — überschreibbar nach dem `override ?? wallet ?? default`-Muster, das Region/Schiff schon nutzen. Erfordert den neuen Scope `esi-wallet.read_character_wallet.v1`; der Wallet-Abruf ist fehlertolerant (Charaktere, die vor der Scope-Erweiterung autorisiert haben, fallen auf den Default zurück).

### Changed

- **SSO fragt zusätzlich `esi-wallet.read_character_wallet.v1` an** (Web + Mobile). **Achtung:** Der Scope muss in beiden EVE-App-Registrierungen auf developers.eveonline.com aktiviert sein, bevor diese Version deployt wird — sonst schlägt der Login fehl. Bestehende Nutzer müssen die App neu autorisieren.

## [0.12.7] - 2026-05-29

### Changed

- **Frontend-Aufräumarbeiten aus dem Parameter-Review (#10):** Geteilte Zahlenformatter (`app/lib/core/format.dart`: `fmtIsk`/`fmtUnits`/`fmtVolume`) und die Security-Risk-Farb-/Label-Zuordnung (`app/lib/core/security_risk.dart`) sind nun je einmal definiert und werden von allen Flutter-Trading-Screens (Hauling, Sell-Assets, ROI, Hub-Vergleich) genutzt — vorher pro Screen dupliziert, mit Drift-Risiko bei der Darstellung.

### Fixed

- **Schiffsliste-Ladefehler im Web wird nicht mehr stillschweigend verschluckt (#5).** `ShipSelect` zeigt bei fehlgeschlagenem Laden der Charakter-Schiffe einen Hinweis ("Standardliste angezeigt") statt unbemerkt auf die Default-Liste zurückzufallen.

### Removed

- Toter Web-Code: ungenutzte Komponenten `ShipRefreshButton` und `VolumeFilters` entfernt (#7).

## [0.12.6] - 2026-05-29

### Fixed

- **Warp-Geschwindigkeit & Align-Zeit jetzt auch in Hauling (#45) und Sell-from-Assets (#107) wirksam.** Beide riefen die Reisezeit-Berechnung mit Default-Geschwindigkeit auf (3,0 AU/s / 6 s) statt der echten Schiffswerte. Hauling reicht jetzt Warp/Align aus dem (ohnehin geladenen) Fitting durch; Sell-from-Assets nutzt das aktuell geflogene Schiff (neuer `CharacterHelper.GetActiveShipTypeID`). Damit ist die schiffsspezifische Reisezeit über alle Routen-Features konsistent.
- **Verkaufssteuer in Trading-Routen & ROI ist jetzt skill-bereinigt.** Der Routen-Rechner verwendete hart `Accounting=0` (5% worst-case), während alle anderen Features das echte Accounting-Skill anwenden — und das ROI-Panel die echte (niedrigere) Rate anzeigte. Der Routen-Rechner nutzt jetzt das Accounting-Level des Characters (Anzeige = Rechnung).

### Changed

- **Geteilte Helfer gegen Inkonsistenz/Redundanz:** Min-Route-Security (`SDERepository.MinRouteSecurityStatus`, mit Logging bei Lookup-Fehler statt stillem high-sec-Default), Schiff-Navigationsparameter (`shipNavParams`) und Fee/Skills-Auflösung (`resolveTradingRates` → `SkillsApplied`) sind nun je einmal implementiert und werden von allen Trading-Services genutzt (vorher 3×/4× dupliziert).

### Removed

- Toter Code: `FeeService.CalculateFees` + `Fees`-Struct + Interface-Methode (keine Nicht-Test-Aufrufer; Routen-Rechner nutzt direkt `CalculateSalesTax`).

## [0.12.5] - 2026-05-29

### Changed

- **Routen-/ROI-Reisezeit nutzt jetzt die schiff-/fitting-/skill-spezifische Warp-Geschwindigkeit und Align-Zeit.** Bisher rechnete der Routen-/ROI-Pfad die Reisezeit mit festen Default-Werten (3,0 AU/s Warp, 6 s Align) — die Agilität/Geschwindigkeit des Schiffs spielte keine Rolle. `applyCharacterSkills` liefert jetzt zusätzlich die deterministisch berechnete effektive Warp-Geschwindigkeit (AU/s) und Align-Zeit aus dem Fitting (Skills + Module) und speist sie in `CalculateTravelTime` ein; `getDefaultFitting` (nicht besessene Schiffe) liefert die Basis-Werte der Hülle. Dadurch schaffen schnellere, agilere Schiffe korrekt mehr Fahrten im Zeitbudget (z.B. Iteron Mark V align ~16 s vs. Imicus ~6 s). Explizite Request-Parameter haben weiterhin Vorrang.

## [0.12.4] - 2026-05-29

### Fixed

- **Schiffswahl ohne Wirkung bei nicht besessenen Schiffen (ROI/Trading).** Wählte man ein Schiff, das der Character nicht im Besitz hat (kein abrufbares Fitting), lieferte `getDefaultFitting` eine Cargo-Kapazität von **0** statt der Hüllen-Basiskapazität. Dadurch rechnete der Route-/ROI-Optimizer für *jedes* solche Schiff mit 0 m³ → identisches (degeneriertes) Ergebnis, egal ob 400 m³ oder 5800 m³. `getDefaultFitting` nutzt jetzt das schiffsspezifische Basis-Cargo aus der SDE, sodass die Schiffsgröße wieder korrekt in Einheiten/Fahrt → Fahrten/Tag → Tagesgewinn eingeht.

## [0.12.3] - 2026-05-29

### Fixed

- **Aktuelle Region/Schiff-Vorbelegung griff nach frischem Login nicht.** Die Prefill-Provider liefen während des SSO-Login-Übergangs (Token noch nicht da), cachten ein leeres Ergebnis und blieben für die Session leer (Region leer, Schiff fiel auf den 648-Fallback). Jetzt sind `currentRegionIdProvider` (Flutter, neu) an den Auth-Zustand gekoppelt und das `CurrentSelectionPrefill`-Mixin wartet auf `Authenticated` + invalidiert die Quell-Provider einmalig, sodass die echten Werte unter Auth geladen werden.
- **Aktuelles Schiff war nicht auswählbar, wenn es nicht im Hangar liegt.** Das aktive Schiff (geflogen) steht nicht zwingend in der Hangar-Liste, die der Schiff-Selektor anbietet — dadurch konnte „aktuelles Schiff" nicht angezeigt werden. Der `ShipSelect` (Flutter + Web) merged das aktive Schiff jetzt als erste (deduplizierte) Option in die Liste.

## [0.12.2] - 2026-05-29

### Security

- Logout leert jetzt den React-Query-Cache (`queryClient.clear()`), damit charakterbezogene Daten (Standort, Schiff, …) der vorigen Session nicht an eine nachfolgende Anmeldung im selben Browser ausgeliefert werden können (Cache-Key war nur auf `isAuthenticated` gekeyt).

### Changed

- **Region- und Schiff-Auswahl werden überall mit dem aktuellen Wert vorbelegt.** Alle Masken mit Region-/Schiff-Selektor (Web + Flutter: Trading, ROI, Hauling) belegen beim Öffnen die Region mit der aktuellen Region des Characters und das Schiff mit dem aktiven Schiff vor (einmalig, danach bleibt die eigene Auswahl). Fallback wie bisher (Forge / Schiff 648) wenn nicht eingeloggt oder ESI nicht erreichbar. Web-Trading konnte das bereits; jetzt konsistent über alle Masken. Flutter: neuer `currentRegionIdProvider` (aus `GET /character/location`) + geteiltes `CurrentSelectionPrefill`-Mixin. (Hauling nutzt die aktuelle Region weiterhin backend-seitig via `origin_region_id:0`; hier wird nur das Schiff vorbelegt.)

## [0.12.1] - 2026-05-29

### Added

- **Sell-from-Assets:** the asset picker is now sortable by name or quantity (ascending/descending), in both the web and Flutter clients. Client-side only — no API change. Items stored in player structures/citadels remain a documented limitation (no route resolvable → "Keine Verkaufsorte").

## [0.12.0] - 2026-05-29

### Added

- **Sell-from-Assets (#107):** `GET /api/v1/trading/assets` (owned items aggregated by type + location) and `POST /api/v1/trading/assets/sell-options` — for a selected item, ranks taker sell locations (instant sell into a buy order, net of sales tax) across the 5 major hubs + every station in the item's current region, each with route (jumps, travel time, security risk). Ranked by total net proceeds. New paginated character-assets fetch; reuses hub price fetch, fee/skills, and navigation.

## [0.11.0] - 2026-05-29

### Added

- **Neighborhood Hauling Routes (#45):** `POST /api/v1/trading/hauling/routes` — from the character's current region + adjacent regions (1 hop, derived from the stargate graph), finds profitable station→station arbitrage (buy cheap, haul, sell dear; direct/taker, no order placing). Per route an optimal cargo load (greedy by profit/m³ under cargo m³ + capital + liquidity), travel time + jumps + security risk (safe/caution/danger), ranked by ISK/h. New `SDERepository.GetNeighborRegions`; reuses the region order fetch, navigation, fitting cargo and skills.

## [0.10.1] - 2026-05-29

## [0.10.0] - 2026-05-28

### Added

- **ROI Calculator & Capital Allocation Optimizer (#44):** `POST /api/v1/trading/portfolio/optimize` — given region, ship, capital and a daily time budget, suggests how to allocate capital across items for maximum expected daily profit. Greedy allocator (most-efficient profit/ISK first) over the existing route-engine candidates, under per-item liquidity and capital caps; returns per-item allocation (capital, units, trips/day, daily profit, ROI%), totals and a Herfindahl-based diversification score. Sec-zone filter applied to candidate routes; skills applied to all figures.

## [0.9.4] - 2026-05-28

### Changed

- **Single schema source (ADR `2026-05-28-single-schema-source`).** `backend/migrations/*.up.sql` is now the only schema definition. The backend applies the embedded, idempotent migrations at startup (`database.ApplyMigrations` in `db.New`), so deploys can no longer drift from the committed schema and new migrations auto-apply. Removed `deployments/init-db/` (the hand-maintained prod schema that had drifted — root cause of the `price_history`/`market_history` bug); local compose no longer mounts it and the deploy runbook drops the manual `psql` step. Integration test asserts the migrations apply idempotently (twice) on a fresh DB.

## [0.9.3] - 2026-05-28

### Fixed

- Market history / daily volume + competition baseline were always 0 in production. Two causes: (1) `FetchAndStoreMarketHistory` was never called, so `price_history` was never populated — `GetVolumeMetrics` now lazy-populates from ESI on a cold (type, region) and re-reads; (2) the prod `init-db` created the table as `market_history`, but the backend queries `price_history` — `init-db` corrected to `price_history` (matching migration 000001) and the dead empty `market_history` dropped. Multi-Hub: volume is fetched before the competition baseline so the baseline reads freshly-populated history.

## [0.9.2] - 2026-05-28

### Changed

- Multi-Hub Comparison (#43): recommend a hub only when its net margin is **positive**. For thin-margin commodities where every hub loses money after fees, no hub is starred and the UI shows a "kein profitabler Hub" hint instead of recommending the least-bad loss. Web table now renders prices with full ISK precision (2 decimals + separators) instead of rounding sub-1000 values to whole ISK.

## [0.9.1] - 2026-05-28

### Added

- Web frontend now served in production **same-origin** at `https://eveonline.sternrassler.de` (Next.js container behind `edge-caddy`; API/auth/swagger/metrics paths route to the backend, everything else to the frontend). Unblocks the web SSO `/callback` and the `/multi-hub` UI in prod without cross-origin CORS. The release pipeline builds + pushes the frontend image alongside the backend.

## [0.9.0] - 2026-05-28

### Added

- **Multi-Hub Comparison (#43):** `POST /api/v1/trading/hubs/compare` compares an item's station-trading profitability across the five major hubs (Jita, Amarr, Dodixie, Rens, Hek) — buy/sell/spread, skill-adjusted net margin (sales tax + broker fee via Accounting/Broker Relations/standings), volume and an order-update-frequency competition indicator. The competition indicator combines a live snapshot-churn metric (periodic background collector over lazily-tracked `(type, region)` pairs) with a daily `order_count` baseline fallback (`source: live|baseline`). New migration `000002_competition_tracking`. Hub price/ranking logic lives in a reusable `MultiHubComparisonService` (consumed later by #107).

### Changed

- Gate all testcontainers-based tests + the `testcontainer.go` helper behind the `integration` build tag (previously a mix of `integration || !unit` and untagged). The default build graph no longer pulls in `github.com/docker/docker`, so `govulncheck` is now a **blocking** CI gate. No change to the production binary.

## [0.8.0] - 2026-05-27

### Security

- Build the backend on Go 1.26 (Dockerfile `golang:1.26-alpine`, CI `setup-go 1.26`) to pull in stdlib CVE fixes; bump `golang.org/x/crypto` to v0.52.0 (GO-2026-5017..5020) and `github.com/gofiber/fiber/v2` to v2.52.12 (GO-2026-4543). `govulncheck` runs in CI (report-only): the sole remaining advisory is `github.com/docker/docker` ("Fixed in: N/A") pulled in by testcontainers — test-path only, not reachable from `cmd/api`.

## [0.7.2] - 2026-05-27

### Added

- CI/CD GitHub Actions: `ci.yml` (gofmt + go vet, docker-free unit lane, report-only govulncheck), `deploy.yml` (SemVer-tag → build+push backend image to GHCR → `deploy-app.sh eveoprovit` on the backend host), `smoke-test.yml` (post-deploy + daily health/TLS check).
- Production deployment behind `edge-caddy` at `https://eveonline.sternrassler.de` with the dedicated prod EVE SSO application; deploy runbook + `deployments/.env.example`.

### Changed

- SemVer-Quelle ist jetzt ausschließlich `CHANGELOG.md` + git-Tags. `release-check`/`release` lesen die Version aus dem CHANGELOG.
- `/api/v1/version` liefert nicht mehr hardcoded `0.1.0`, sondern die zur Build-Zeit injizierte Version (`internal/version.Version` via `-ldflags`). `APP_VERSION` wird im Makefile aus dem obersten veröffentlichten CHANGELOG-Eintrag abgeleitet und über den Docker-Build-Arg an das Backend gereicht; Fallback `dev` für lokale Builds.

### Removed

- GitHub-Workflows (`codeql-analysis`, `integration-tests`, `lint-test`, `load-tests`, `pr-quality-gates`, `test-migrations`, `workflow-run-cleanup`)
- Issue-Templates (`bug_issue`, `feature_issue`) und Template-Config
- `CODEOWNERS`
- `copilot-instructions.md` und `copilot-instructions-old.md`
- `VERSION`-Dateien (Root + Backend) — SemVer lebt nur noch im CHANGELOG
- Copilot-Governance-Reste: `scripts/common/check-normative.sh` (inkl. pre-commit/CI-Aufruf), Copilot-Bot-Skip in der Commit-Message-Prüfung und `copilot-instructions.md`-Verweise

## [0.5.0] - 2025-11-13

### Added

- **Deterministic Calculation Engine** - Complete ship fitting calculations
  - `pkg/evedb/cargo`: Deterministic cargo capacity with skills + module stacking penalties
  - `pkg/evedb/navigation`: Deterministic warp speed + inertia/align time calculations
  - Stacking penalty formula for passive modules (exponential diminishing returns)
  - FittingService integration with all three deterministic functions
  - API endpoints: `/api/v1/calculations/cargo`, `/api/v1/calculations/warp`, `/api/v1/calculations/align`
  - Complete test coverage with SDE database integration tests

### Changed

- **Service Layer Refactoring** - Eliminated ~530 lines of redundant wrapper code
  - Phase 1: Removed NavigationService wrappers (GetEffectiveWarpSpeed, GetEffectiveInertia)
  - Phase 2: Consolidated travel time calculations (unified CalculateTravelTime with useExactFormula parameter)
  - Phase 3: Removed all deprecated wrappers (no backward compatibility)
  - CargoService simplified to single responsibility (Knapsack DP only)
  - RouteService refactored to use FittingService directly
  - NavigationService reduced to system/location resolution only
- **Architecture Simplification** - Frontend-first deterministic values
  - Frontend calculates deterministic values once (warp_speed, align_time, cargo_capacity)
  - Backend reuses these values for route calculation (eliminates redundant calculations)
  - FittingService now returns complete fitting data including navigation values
- **Documentation Consolidation** - 67% reduction in documentation size
  - README simplified for new visitors (68% reduction: 350 → 110 lines)
  - docs/ folder streamlined (64% reduction: 1,100 → 400 lines)
  - 6 implementation summaries archived (~1,500 lines)
  - ASCII-Art diagrams converted to Mermaid (3 diagrams)
  - ARCHITECTURE.md updated with Service Layer Patterns section

### Fixed

- Market orders schema corrections (issued_at/cached_at nullable time.Time)
- ON CONFLICT clause for market_orders table (order_id, cached_at)
- SDE database path configuration via SDE_DB_PATH environment variable
- Docker health check path corrected to /api/v1/health
- OpenAPI BasePath and route paths for Swagger UI
- ESLint errors (unescaped entities, unused variables)
- Test suite improvements: Skip SDE-dependent tests when database unavailable (CI compatibility)

### Technical

- OpenAPI/Swagger documentation for all 17 API endpoints
- Centralized test database utilities in testutil package
- Service Layer Patterns documented in ARCHITECTURE.md
- All deterministic core functions preserved in pkg/evedb (guaranteed single source of truth)

## [0.4.0] - 2025-11-08

### Removed

- **Legacy Service Cleanup (Issue #75)** - Removed 6 service files containing ~730 LOC of unused/redundant code
  - Deleted `MarketFetcher` service (functionality integrated in RouteFinder)
  - Deleted `ProfitAnalyzer` service (100% code duplication with RouteFinder.FindProfitableItems)
  - Deleted `RoutePlanner` service (navigation logic integrated in RouteCalculator)
  - Deleted `TradingService` (only used for removed `/inventory-sell` endpoint)
  - Deleted `/api/v1/trading/inventory-sell` endpoint and all related code (InventorySellOrchestrator, handlers, models)

### Changed

- **Service Architecture Simplification** - Reduced service count from 11 to 7 (-36%)
  - Renamed `RouteOptimizer` → `RouteCalculator` for clarity (consistent with interface naming)
  - Inlined `CalculateJumpTime` logic in RouteCalculator (previously delegated to RoutePlanner)
  - Improved code organization and removed naming confusion

### Technical

- Service reduction improves build times, test execution, and code maintainability
- Clearer separation of concerns between remaining services
- No functionality lost - all features remain intact through consolidation

## [0.3.0] - 2025-11-08

### Added

- **Configurable Route Service Timeouts** - Environment variable support for timeout configuration
  - `Config` struct pattern in RouteService with three configurable timeouts
  - Environment variables: `ROUTE_CALCULATION_TIMEOUT` (default: 120s), `ROUTE_MARKET_FETCH_TIMEOUT` (default: 60s), `ROUTE_ROUTE_CALC_TIMEOUT` (default: 90s)
  - `DefaultConfig()` function for standard timeout values
  - Updated all tests to use DefaultConfig() pattern

### Changed

- Refactored RouteService from hardcoded timeout constants to configurable Config struct
- Increased default timeouts to support large regions like The Forge (388 market order pages)
- Updated `.env.example` and `deployments/.env` with timeout configuration documentation

### Fixed

- The Forge region route calculation timeout issues (30s → 60s market fetch timeout)

## [Unreleased - Skills Service]

### Added

- **Skills Service (Phase 0 - Issue #54)** - Centralized character skills management
  - `SkillsService` implementation with ESI integration and Redis caching (5min TTL)
  - `TradingSkills` struct covering 12 trading-relevant skills (fees, cargo, navigation)
  - Graceful degradation: ESI failure → default skills (all = 0) instead of blocking errors
  - Comprehensive test suite (8 test cases, miniredis-based)
  - `pkg/logger` package (simple structured logger for services)
  - Documentation: `internal/services/SKILLS_SERVICE.md`
  - Skill extraction logic for: Accounting, Broker Relations, Navigation, Evasive Maneuvering, 4x Racial Industrials
  - Temporary ESI interface (workaround until `eve-esi-client` implements `GetCharacterSkills`)
  - Foundation for Fee Service (#55) and Cargo Service (#56)

## [0.2.0] - 2025-11-04

### Added

- **Market Data Management** - Region Staleness & Manual Refresh
  - `RegionStalenessIndicator` Component (color-coded: 🟢 <5min, 🟡 5-15min, 🟠 >15min)
  - `RegionRefreshButton` Component (manual market data refresh per region)
  - Backend endpoint: `GET /api/v1/market/staleness/:region`
  - Auto-refresh staleness indicator every 60s
  - Integrated in `RegionSelect`, Intra-Region Trading, Inventory Sell pages
- **Performance Optimization Infrastructure (Phase 3)** - Intra-Region Trading Route Calculation
  - **BatchFetcher Integration** (eve-esi-client v0.3.0) - Automatischer paralleler Abruf aller Market Order Seiten
  - Redis caching for market orders (5min TTL, gzip compression, ~80% size reduction)
  - Redis caching for navigation data (1h TTL)
  - Context-based timeout handling (30s total: 15s market fetch, 25s route calculation)
  - HTTP 206 Partial Content support for timeout scenarios
  - ESI rate limiter (Token Bucket pattern, 300 req/min)
  - Exponential backoff retry for ESI 429 errors
  - In-memory volume filtering (reduces candidates by ~80%)
  - Prometheus metrics for trading operations
- **Backend Foundation** - Complete dual-database architecture implementation
  - PostgreSQL integration for dynamic market data
  - SQLite SDE integration for read-only static data
  - Dual-DB connection management with health checks
- **ESI Client Integration** - eve-esi-client v0.3.0
  - Market orders fetching via BatchFetcher pattern
  - Redis-based caching (ADR-009 compliant)
  - Automatic rate limiting and error handling
  - Concurrent page fetching mit Worker Pool (ADR-011 in eve-esi-client)
- **Database Migrations** - golang-migrate setup
  - Migration 001: market_orders and price_history tables
  - Makefile targets for migration management
- **API Endpoints**
  - `GET /health` - Health check with database status
  - `GET /version` - API version information
  - `GET /api/v1/types/:id` - SDE type information lookup
  - `GET /api/v1/market/:region/:type` - Market orders with ESI integration
  - `GET /api/v1/market/staleness/:region` - Market data freshness indicator
  - `POST /api/v1/trading/routes/calculate` - Trading route calculation with timeout support
- **Repository Pattern** - Clean architecture implementation
  - MarketRepository for PostgreSQL operations
  - SDERepository for SQLite read-only access
- **Documentation**
  - Comprehensive ARCHITECTURE.md (500+ lines system overview)
  - Updated README with v0.1.0 Production Ready status
  - API endpoint documentation
  - Docker Compose usage guide
  - Database migration guide
  - ADR-011: Worker Pool Pattern (Superseded - moved to eve-esi-client)
  - ADR-012: Redis Caching Strategy
  - ADR-013: Timeout Handling (HTTP 206 Partial Content)
  - Frontend README with component documentation
  - Archive for obsolete implementation summaries

### Changed

- Updated `go.mod` with new dependencies (pgx/v5, go-redis/v9, eve-esi-client, golang.org/x/time/rate)
- Enhanced `.env.example` with all required environment variables
- Updated Makefile with database migration targets
- Refactored main.go for dual-database initialization
- RouteCalculator now supports Redis caching and parallel processing

### Technical

- ADR-001: Tech Stack - PostgreSQL + SQLite dual-DB confirmed
- ADR-009: Shared Redis Infrastructure - Implemented with key-namespacing
- ADR-011: Worker Pool Pattern - **Superseded** (2025-11-04), Pattern verschoben nach eve-esi-client BatchFetcher
- ADR-012: Redis Caching Strategy - 5min TTL, gzip compression
- ADR-013: Timeout Handling - HTTP 206 Partial Content pattern
- Dependencies: **eve-esi-client v0.3.0**, pgx/v5, go-redis/v9, golang-migrate/v4, golang.org/x/time/rate

### Performance

- Target: The Forge (383k orders) calculation < 30 seconds
- Cache hit ratio: > 95% after warmup
- Worker pools enable concurrent processing while respecting ESI rate limits
- Gzip compression reduces cache memory usage by ~80%

## [0.1.0] - 2025-10-05

- Project initialization.
