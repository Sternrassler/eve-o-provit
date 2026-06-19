# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- **Backend-Container läuft non-root (Hardening).** Das `eve-o-provit-backend`-Image lief als root; jetzt als unprivilegierter `appuser` (uid 10001). Die API schreibt nichts auf Platte (liest die read-only SDE-Mount, spricht Postgres/Redis; `/tmp` ist zur Laufzeit tmpfs) → keine beschreibbaren Pfade im Image nötig. Teil der Incident-2026-06-18-Härtung (Defense-in-Depth zusätzlich zu cap_drop/read-only-rootfs/no-new-privileges im hetzner-Compose). `docker build` + Non-root-Lauf (uid 10001) verifiziert.

## [0.35.0] - 2026-06-08

### Fixed

- **Mining: transienter Fitting-Abruf-Fehler wird nicht mehr als „kein Mining-Setup" fehlinterpretiert.** Schlug der ESI-Abruf des aktuellen Schiffs/der Module fehl, wurde still mit `moduleIDs=nil` weitergerechnet → `m3h=0` → `no_mining_setup=true`, also „du hast kein Mining-Schiff" statt „ESI-Hiccup". Jetzt: neues `fitting_degraded`-Flag, `no_mining_setup` wird nur noch bei **erfolgreichem** Abruf mit 0 m³/h gesetzt, und der Grund landet in `degraded_reason` (Banner zeigt ihn automatisch). Letzte verbliebene Silent-Fallback-Stelle im Mining-Pfad, Prinzip [[no-silent-fallbacks]]. `degraded_reason` ist jetzt kompositorisch (Skills/Standings/Fitting in einer Meldung).


### Changed

- **CI/Build härtet die Docker-Hub-Abhängigkeit.** Base-Image `alpine:latest` → `alpine:3.21` gepinnt (reproduzierbare Builds); `deploy.yml` loggt sich optional bei Docker Hub ein (hebt das Anonymous-Pull-Rate-Limit auf Shared-GH-Runnern, das den v0.34.0-Deploy mit `registry-1.docker.io context deadline exceeded` riss) — no-op bis `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN`-Secrets gesetzt sind.


## [0.34.0] - 2026-06-08

### Fixed

- **Mining: stille Skills-/Standings-Degradation jetzt sichtbar (no-silent-fallbacks).** Konnte das Erz-Ranking die Mining-/Reprocessing-Skills oder Standings nicht von ESI laden (z. B. abgelaufener Token), rechnete es **still** mit Null-Skills bzw. neutralen Standings weiter — der Owner sah massiv falsche ISK/h-/Yield-Werte ohne Hinweis (real: 3× Standings-401 am 2026-06-07). Die `OreRankingResponse` trägt jetzt `skills_degraded`/`standings_degraded` + eine erklärende `degraded_reason`; Web und Flutter zeigen einen Warn-Banner über der Tabelle (auch bei vorhandenen Zeilen). Das Ranking wird weiter geliefert, aber nicht mehr als verlässlich dargestellt. Spiegelt das `is_estimate`/`not_routable_reason`-Muster derselben Feature-Fläche.


## [0.33.0] - 2026-06-07

### Changed

- **eve-esi-client v0.6.0 → v0.7.0: Fresh-Serve.** Frische ESI-Cache-Einträge (innerhalb des `Expires`-Fensters) werden jetzt ohne Netzwerk-Roundtrip direkt aus Redis serviert statt pro Lookup per Conditional Request (304) revalidiert zu werden; Caching strikt GET-only. Effekt: Wiederholungs-Berechnungen (Hauling/Mining/ROI) im 5-Minuten-Markt-Fenster laufen in Sekundenbruchteilen statt ~30 s, ESI-Last und Rate-Limit-Token-Verbrauch sinken deutlich. Details → eve-esi-client-CHANGELOG v0.7.0.


## [0.32.0] - 2026-06-07

### Changed

- **eve-esi-client v0.5.1 → v0.6.0: Adoption des neuen ESI-Gruppen-Rate-Limitings (`X-Ratelimit-*`).** Die Lib trackt jetzt CCPs Token-Buckets pro Route-Gruppe (z. B. `market-order: 12000/15m`), drosselt proaktiv bei knappem Budget und respektiert `Retry-After` nach 429 (Retry statt Sofort-Fehler). Das Legacy-Error-Limit-Tracking bleibt für nicht migrierte Routen aktiv. Details → eve-esi-client-CHANGELOG v0.6.0.

## [0.31.1] - 2026-06-07

### Fixed

- **`/metrics` lieferte 500 (Label-Korruption durch Fiber-Zero-Copy).** Die HTTP-Metrics-Middleware speicherte `c.Method()`/`c.Route().Path` ungekopiert als Prometheus-Label; fasthttp recycelt die Request-Puffer, der nächste Request mutierte das gespeicherte Label (Prod: `method="POS"`) → `Gather()` schlug mit Duplikat-Fehlern fehl → `/metrics` 500 und Prometheus-Scrape tot. Fix: `strings.Clone` vor dem Speichern; Regressionstest simuliert die Puffer-Wiederverwendung (50× alternierende Requests + Gather).
- **ESI-Dauerdrossel durch vergifteten Rate-Limit-State (Hauling-Timeout).** eve-esi-client auf **v0.5.1** gehoben: ein stale `errors_remaining`-Wert in Redis (TTL -1, durch die abgeschafften `X-ESI-Error-Limit-*`-Header nie wieder angehoben) drosselte jeden ESI-Aufruf inkl. Cache-Hits mit 1 s Sleep — Hauling brauchte 73 s, der Client lief in den Timeout. Die Lib behandelt abgelaufene Reset-Fenster jetzt als healthy und versieht den State mit TTLs (Details im eve-esi-client-CHANGELOG).

## [0.31.0] - 2026-06-07

### Added

- **App-spezifische Prometheus-Metriken (Backend).** Drei neue Messschichten unter dem einheitlichen `eveoprovit_`-Namespace: (1) **HTTP** — `eveoprovit_http_requests_total{method,route,status}` + Latenz-Histogramm `eveoprovit_http_request_duration_seconds{method,route}` via Fiber-Middleware (Route-Pattern statt Roh-Pfad gegen Label-Explosion, unmatched → `(unmatched)`, `/metrics`/`/swagger`/Health ausgenommen; Buckets bis 60 s wegen langer Routen-Berechnungen). (2) **ESI** — `eveoprovit_esi_requests_total{status}` + Dauer-Histogramm über einen instrumentierten `http.RoundTripper` (`SetHTTPClient` am eve-esi-client): zählt jeden Transport-Versuch inkl. Retries — die ehrliche Messgröße fürs ESI-Error-Limit (420er sichtbar), `transport_error` für Verbindungsfehler. (3) Bestehende Trading-Metriken (Calc-Duration, Cache-Hits/Misses) bleiben erhalten.
- **Tooling: `make android-install` (Flutter-App, `app/Makefile`).** Baut die Release-APK mit den Prod-`--dart-define`s (`API_BASE_URL=https://eveonline.sternrassler.de`, `EVE_CLIENT_ID` aus `deployments/.env` → `EVE_MOBILE_CLIENT_ID`, fail-loud wenn Datei/Variable fehlt), installiert sie per `adb install -r` aufs angeschlossene Gerät und verifiziert den Install via `dumpsys` (INTERNET-Permission + `lastUpdateTime`). Macht die beiden bekannten Fehlerklassen mechanisch unmöglich: „APK ohne dart-defines gebaut → leere `client_id` → SSO-Login kaputt" und „APK gebaut, aber nie aufs Gerät deployed".

### Changed

- **Prometheus-Namespace vereinheitlicht:** `trading_*`-Metriken heißen jetzt `eveoprovit_trading_*` (Konvention wie depots `depot_`-Prefix). Kein Konsument betroffen — die Metriken wurden bis heute von keinem Prometheus gescraped.

### Removed

- **Tote Metriken entfernt:** `trading_cache_hit_ratio` und `trading_worker_pool_queue_size` wurden nirgends gesetzt und exportierten irreführende Konstanten (0).

## [0.30.0] - 2026-06-04

### Added

- **Mining: „Verkaufte Ladungen"-Strichliste (Web + Flutter).** Im aufgeklappten Erz-Detail kann man unter „Markt: ~X volle Ladungen" die bereits verkauften Ladungen abhaken und behält den Überblick: antippbare Pips (eine pro voller Ladung) mit Zähler „Verkauft: X / N"; bei > 24 Ladungen automatisch Zähler mit +/− und Fortschrittsbalken (statt zig Pips). Rein client-seitig/in-memory, **Reset bei jeder Neuberechnung**. Neue Komponente `SoldLoadsTally` (Web) bzw. `MiningLoadsTally` + `soldLoadsProvider` (Flutter).

## [0.29.0] - 2026-06-04

### Fixed

- **Mining: veraltete Marktpreise → Live-ESI-Abfrage (Backend).** Der Mining-Erz-Ranking las Order-Books aus der `market_orders`-DB-Momentaufnahme (`MarketRepository.GetMarketOrders`), die **nur on-demand** von Trading-Routen-Abfragen aufgefrischt wird (kein Hintergrund-Refresher) — für eine Region, gegen die länger keine Trading-Abfrage lief, war sie **tagealt** (real beobachtet: Verge Vendor **~3,5 Tage**). Folge: falsche Verkaufsstation/ISK (z. B. Alentene 10.50 statt der aktuellen Best-Order). Mining holt die Order-Books jetzt **live von ESI** (`FetchMarketOrdersForType`, ETag-/Redis-validiert) wie Trading/Multi-Hub/Sell-Assets — pro Request einmal je Typ memoisiert (wiederkehrende Mineral-Typen). ESI-Fehler werden **fail-loud** propagiert (kein stilles Zurückfallen auf alte Daten), Prinzip [[no-silent-fallbacks]].

### Added

- **Mining: aktueller Schiff-Standort in der Antwort (Backend).** `OreRankingResponse` liefert jetzt `origin_system_id`/`origin_system_name` (+ `origin_station_name` falls angedockt) — das System, von dem aus gerankt/geroutet wird. Grundlage für die Standort-Anzeige im UI.
- **Mining: Schiff-Standort im UI (Web + Flutter).** Die Mining-Ansicht zeigt im Ergebnis-Header „Aktueller Standort: \<System\>" (bzw. „\<Station\> (\<System\>)" falls angedockt) aus den neuen `origin_*`-Feldern — Web (`mining/page.tsx`) und Flutter (`mining_screen.dart`).

## [0.28.0] - 2026-06-04

### Added

- **Trading-Sub-Tabs (Frontend).** Die fünf Trading-Werkzeuge (Routes · Hauling · ROI · Multi-Hub · Sell Assets) haben jetzt eine gemeinsame **Tab-Leiste** auf jeder der Seiten — man wechselt das Werkzeug direkt, ohne über die Top-Nav zu gehen; die Trading-Sektion liest sich als ein Hub (analog zum Flutter-Client). Jeder Tab bleibt eine echte Route (Deep-Links funktionieren weiter), der aktive Tab ist routen-bewusst (`usePathname`). Neue Komponente `components/trading/TradingTabs`, eingebunden in die fünf Seiten; Top-Nav-Dropdown unverändert. Verifiziert via eslint/tsc/`next build`/vitest + Real-Browser-Check.

### Removed

- **Verwaiste `/calculations/*`-Endpunkte entfernt (Backend).** `POST /api/v1/calculations/cargo` und `POST /api/v1/calculations/warp` (samt `CalculationHandler`, ~260 LOC, und den nur dafür genutzten Modellen `Cargo/WarpCalculationRequest/Response` + Helfer) hatten **keinen Consumer** — weder Web noch Flutter rufen sie auf. Die zugrundeliegenden Domänen-Funktionen (`CalculateWarpTime`/`CalculateCargoFit` in `pkg/evedb/*`, vom Route-Calculator genutzt) bleiben unangetastet. Spec via `make swagger` mit-bereinigt. `deadcode`/`golangci-lint` weiterhin 0, alle BE-Tests grün.

### Changed

- **Swagger-Drift dauerhaft verhindert (Build/CI).** Neues `make swagger`-Target regeneriert die OpenAPI-Spec reproduzierbar via `go run github.com/swaggo/swag/cmd/swag@<pinned>` (Version in `SWAG_VERSION` einmalig gepinnt). `make release` zieht jetzt die Swagger-`@version` automatisch mit (`// @version` = Release-Version) und regeneriert die Spec. Ein CI-Schritt (`lint`-Job) führt `make swagger` aus und failt bei einem Diff in `backend/docs` — so kann eine veraltete Spec nicht mehr committet werden (gleicher Geist wie „nie zwei Schema-Quellen"). Verhindert die Wiederholung der 7-Monats-Drift aus dem vorigen Eintrag.
- **Swagger/OpenAPI-Doku neu generiert (Backend).** Die unter `/swagger` ausgelieferte API-Spec war seit 2025-11-13 nicht mehr regeneriert (`info.version` hing auf `0.1.0`) und beschrieb weder neuere Endpunkte noch Felder — u. a. fehlte `market_loads` in der Ore-Ranking-Antwort. `swag init -g cmd/api/main.go --parseInternal` neu erzeugt (`docs/{docs.go,swagger.json,swagger.yaml}`), `@version` auf `0.27.0` gesetzt. Reine Doku-Regeneration aus den vorhandenen Annotationen/Structs, kein Verhaltens-/API-Change.

## [0.27.0] - 2026-06-04

### Changed

- **Navigation gruppiert + tote Platzhalter aus der Nav entfernt (Frontend).** Die flache 9-Punkte-Leiste wurde auf **3 Top-Level-Einträge** verdichtet — ein **„Trading"-Dropdown** (Trading Routes · Hauling · ROI Calculator · Multi-Hub · Sell Assets) plus **Mining** und **Character** —, analog zur Gruppierung im Flutter-Client. Die statischen Phase-3-Platzhalter **Trends** und **Watchlist** (kein Backend, siehe #47/#48) sind nicht mehr in der Nav, damit sie keine funktionierenden Features vortäuschen; die Seiten selbst bleiben als UI-Spec erhalten. Reiner Frontend-Umbau (`navigation.tsx`); die shadcn-`NavigationMenu`-Komponente + Radix-Dep, in #170 als ungenutzt entfernt, sind für das Dropdown wieder eingebunden (jetzt mit echtem Zweck). Verifiziert via eslint/tsc/`next build`/vitest + Real-Browser-Check (Dropdown öffnet, Links korrekt, Platzhalter weg).
- **Auth-Kontext-Extraktion in den Handlern konsolidiert (Backend).** Die Identitäts-Locals (`character_id`/`access_token`), die `NewAuthMiddleware` auf jeder geschützten Route setzt, wurden in **7 Handlern** jeweils inline neu extrahiert und null-/typgeprüft — in mehreren Stilen mit divergierenden Fehlertexten („Authentication required" / „authentication required" / „Invalid authentication context"). Neuer typsicherer Accessor `evesso.AuthFromContext(c) (id, token, ok)` (mit der zugehörigen Locals-Key-Konstante als Single Source of Truth gegenüber `setAuthLocals`) plus ein einheitlicher `errUnauthorized`-401-Responder ersetzen ~12 Stellen. Verhalten unverändert (Status-Codes gleich; die zwei test-geprüften Meldungen in `character`/`fitting` bleiben erhalten); die generischen 401-Bodies sind jetzt einheitlich „Authentication required". Netto ~45 Zeilen weniger, alle BE-Tests + golangci-lint grün. Die zwei Token-only-ESI-UI-Handler bleiben bewusst unangetastet.
- **Toter Code entfernt (BE + FE, #170).** Reine Aufräumarbeit, **1758 Zeilen gelöscht, 0 hinzugefügt** — kein Verhaltens-, API- oder Wire-Effekt. **Backend** (verifiziert via `deadcode -test ./...`, das danach 0 meldet): toter `NavigationService` samt `NavigationServicer`-Interface (Pass-through, nie konstruiert), die hartkodierte Hauler-Tabelle `ShipType`/`CommonHaulers`/`GetShipType` + `ValidationError` (abgelöst durch die dynamischen Ship/Fitting-Services), die verwaiste `GetShipNavigationSkills`-Query nebst Nav-Typen, der `Deprecated`-Wrapper `NewWithConcrete` und die nie verdrahtete `NewOptionalAuthMiddleware`. **Frontend** (verifiziert via knip): Mock-Daten-Reste (`mock-data/{regions,trading-routes}.ts`), ungenutzte shadcn-Komponenten `navigation-menu`/`switch` samt der dann überflüssigen Radix-Deps, verwaiste `tests/helpers/auth.ts` und die toten Trading-Typen `LoadingState`/`InventorySellRequest`/`InventorySellRoute`.

### Security

- **Go-Toolchain auf 1.26.4 angehoben (11 erreichbare Stdlib-CVEs behoben).** `govulncheck` meldete 11 erreichbare Schwachstellen in der Go-Standardbibliothek von **go1.26.1** (u. a. `crypto/x509`, `net/http`, `html/template`, `net`, `net/textproto` — GO-2026-49xx/50xx), alle in **go1.26.4** gefixt. Ursache: keine `toolchain`-Direktive in `go.mod`, also Build mit dem installierten 1.26.1; CI-`setup-go "1.26"` löste auf 1.26.1 auf. Fix: `toolchain go1.26.4` in `backend/go.mod`, `go-version: "1.26.4"` in der CI (3×) und `golang:1.26.4-alpine` im Dockerfile. Reiner Toolchain-Bump, kein Code — `govulncheck ./...` danach **0**. Behebt den roten `vuln`-CI-Check auf `main`.

## [0.26.0] - 2026-06-02

### Added

- **Mining-Rechner: Markt-Mengen-Cap als „volle Ladungen" pro Zeile (#169).** Jede Erz-Zeile zeigt jetzt, **wie viele komplette Erzraum-Ladungen** (bezogen auf dein aktuelles Schiff) die gewählte Buy-Order aufnimmt — abgeleitet aus deren `VolumeRemain`. **Roh:** Kapazität der Erz-Order; **Refine:** das zuerst erschöpfte Mineral bindet (Minimum über alle Minerale). Trägt der **Bestpreis nicht mal eine volle Ladung** (`< 1`), wird die Zeile fail-loud markiert („⚠ Bestpreis nur ~X Ladungen — ISK/h optimistisch"), denn die ISK/h nimmt bislang an, du könntest unbegrenzt zum Bestpreis verkaufen. **Reine Anzeige** — ISK/h und Ranking unverändert. Neues Response-Feld `market_loads`. Web **und** Flutter. (Nicht im Scope: Order-Book-Tiefe, echtes ISK/h-Capping — bewusst zurückgestellt.)

## [0.25.4] - 2026-06-02

### Fixed

- **Mining-Rechner: Pyroxeres nur noch in Amarr-/Caldari-High-Sec (#168).** Das Ranking listete Pyroxeres fälschlich in **jedem** High-Sec-System (0.5–0.9). Tatsächlich kommt Pyroxeres im High-Sec **nur in Amarr- und Caldari-Quarter** vor — Gallente/Minmatar haben dort stattdessen Plagioclase (Quelle: EVE-University „Ore"-Tabelle, in-game gegen Bodenwahrheit geprüft: Alentene/Gallente hat kein Pyroxeres, Tar/Ourapheh/Amarr schon). Weder angezeigte Security noch `securityClass` trennen die Fälle — allein der **Empire-Quarter** tut es. `AvailableOreGroups` schärft die High-Sec-Regel entsprechend nach; Low-Sec war bereits korrekt. **Daten-Grenze (unverändert):** die exakte Gürtel-Zusammensetzung *innerhalb* eines Systems kennt nur der In-Game-Client („The Agency"/Survey-Scanner) — SDE/ESI enthalten keine Erz↔Gürtel-Zuordnung. Backend-only.

## [0.25.3] - 2026-06-01

### Changed

- **Mining-Rechner: Erz-Varianten zeigen In-Game-Namen + unsere Übersetzung (#167).** Varianten werden jetzt als **„Veldspar II-Grade (Concentrated)"** angezeigt — der **In-Game-Name** (den der EVE-Client zeigt) plus unser unterscheidendes Adjektiv in Klammern. Vorher zeigte die App nur den abgeleiteten Client-Namen („Concentrated Veldspar"), was sich nicht mit dem Übersichts-/Marktfenster im Spiel abgleichen ließ. Das Adjektiv ist das führende Wort des Kompressions-Blueprint-Namens; Basis-Erze (ohne „-Grade") bleiben unverändert. Reine Anzeige — Erz-Verfügbarkeit/Banding läuft über die GroupID, nicht den Namen. Backend-only (Web + Flutter rendern das Namensfeld unverändert).

## [0.25.2] - 2026-06-01

### Fixed

- **Mining-Rechner: erreichbarkeitsbewusste Haul-Downtime, entkoppelt (#166).** Unter „Nur High-Sec" waren **alle** Zeilen fälschlich „≈ Schätzung", weil die effektive ISK/h auf Routen zu Verkaufs-/Reprocess-Stationen baute, die im Low-Sec oder in Citadels liegen (Route scheitert → geschätzt). Zwei Ursachen behoben: (1) **Ziel-Wahl ist jetzt erreichbarkeitsbewusst** — Erz-Verkaufsort, Reprocess-Station und Mineral-Hub werden unter der Sicherheits-Bereitschaft des Spielers (`AvoidLowSec`) gewählt (Citadels ≥ 1e12 sind unerreichbar); bei gleichem Kaufpreis gewinnt die **näher** gelegene Station. (2) **Raw- und Refine-Pfad werden entkoppelt** berechnet: scheitert die Refine-Route, bleibt ein erreichbarer Raw-Pfad unangetastet (Zeile zeigt „roh", keine Schätzung). Eine Zeile wird nur noch **übersprungen**, wenn **kein** erreichbarer Verkaufs-/Reprocess-Ort existiert; `is_estimate` bleibt für Hull/Crystal-Auflösung (#165) reserviert, nicht für Routing. Per-m³-Zeilen erscheinen weiterhin ohne gefittetes Mining-Modul (ISK/h dann 0). Zusätzlich **fail-loud**: stille Fehlerpfade bei Erzraum- und Schiffs-Fitting-Auflösung loggen jetzt `Warn`. Backend-only, Response-Schema unverändert (Web + Flutter ohne Änderung). Zurückgestellt unverändert: Null-Sec (#161), Class K (#162), exakte Belt-Inhalte (#163), Max-Ratio-Zyklus (#158).

## [0.25.1] - 2026-06-01

### Fixed

- **Mining-Rechner: echte Erz-Namen statt „-Grade" + Pioneer-Schiffsbonus (#165).** Zwei in v0.25.0 entdeckte Bugs: (1) Das Ranking zeigte den rohen SDE-Namen („Veldspar II-Grade") statt des echten Client-Namens („Concentrated Veldspar") — der Rename aus `ListOres` steuerte nur den IV-Grade-Filter, nicht den angezeigten Namen (`GetOre` lieferte den rohen Namen). (2) Die **Pioneer** (Mining-Destroyer) wurde komplett als „≈ Schätzung" markiert, weil ihr Mining-Bonus (per-Level-Bonus über die Mining-Destroyer-Skill **plus** ein einmaliger Rollen-Bonus) nicht in der unterstützten Effekt-Menge stand; der Rollen-Bonus (einmalig, nicht per Skill-Level) wurde vom Modell gar nicht abgebildet. Beide behoben: echter Name wird angezeigt, Pioneer-Yield exakt (Mining Destroyer V → 2.25×). Andere Nicht-Barge/Exhumer-Hulls (Venture, Orca/Rorqual, Prospect/Endurance) bleiben weiterhin Schätzwert (bekannt aus #157).

## [0.25.0] - 2026-06-01

### Changed

- **Mining-Rechner: system-genaue Erz-Verfügbarkeit + echte Erz-Namen (#164).** Das Ranking zeigt jetzt **nur die Erze, die im aktuellen System wirklich vorkommen** — deterministisch aus Sicherheitsstatus + Empire-Quarter (EVE-University-Regeln), für **High- und Low-Sec**. Vorher lieferte der grobe „high"-Filter fälschlich auch Low-Sec-Erze (Hemorphite/Hedbergite/Kernite/Jaspet), die es im Hi-Sec gar nicht gibt. Neuer **Toggle „Nur High-Sec / High + Low-Sec"** (Bereitschaft; steuert auch das Routing). Erz-Varianten tragen jetzt ihre **echten Client-Namen** (Concentrated/Dense Veldspar, Vivid/Radiant Hemorphite …), data-driven aus CCPs Kompressions-Blueprints; nicht-abbaubare „IV-/0-Grade"-Geistervarianten (kein Blueprint) sind rausgefiltert. **Fail-loud:** kein Standort → Fehler; unbekanntes Quarter → nur quarter-unabhängige Erze; Low-Sec bei „Nur High-Sec" bzw. Null-Sec → Hinweis statt falscher Liste. API-Bruch: `sec_band` → `allow_low_sec`; Web + Flutter umgestellt. Zurückgestellt: Null-Sec (#161), Class K (#162), exakte Belt-Inhalte per Scanner/Manuell (#163).

## [0.24.0] - 2026-06-01

### Added

- **Mining-Rechner: In-Game-Markt- + Routen-Links (#160, nur Web).** Aus der Erz-Rangliste heraus direkt im EVE-Client agieren: **Erz- oder Mineral-Name anklicken** öffnet das Marktfenster (`openMarketDetails`); **Reprocess-Station oder einen Verkaufsort anklicken** setzt einen Autopilot-Waypoint (`setWaypoint`, ersetzt die Route). Der Erz-Name-Klick öffnet den Markt, ohne die Zeile auf-/zuzuklappen. Fehlt ein Routenziel, bleibt der Eintrag reiner Text; ein ESI-Fehler (EVE nicht offen, kein Docking-Zugriff) erscheint als Fehler-Toast. Backend: `SellLocation` trägt jetzt eine `location_id` als Waypoint-Ziel (NPC-Station und Citadel); Reprocess-Route nutzt `best_station_id`. Wiederverwendet die bestehenden ESI-UI-Calls; der Web-SSO hat die nötigen Scopes (`esi-ui.open_window.v1` + `esi-ui.write_waypoint.v1`) bereits — **kein neuer Scope, kein Re-Login**. Nur Web (kein Flutter).

## [0.23.0] - 2026-06-01

### Added

- **Mining-Rechner: Ranking nach effektiver ISK/h inkl. Haul-Downtime (#159).** Die Erz-Rangliste sortiert jetzt nach der **realistischen** ISK/h, die den Lade-Zyklus einrechnet: Erzraum vollmachen → zum Verkauf bzw. zur Aufbereitung fliegen → dort weiterminen (Einweg-Kette, kein Rückweg). Raw = 1 Etappe (Erz-Verkauf), Refine = 2 Etappen (Reprocess-Station → bestes In-Region-Verkaufs-Hub, das die ISK/h maximiert). Erzraum aus dem aktuellen Schiff (SDE-Attr 1556, sonst Frachtraum), Reisezeit aus Warp/Align des Schiffs, fixer Stopp-Overhead (75 s). Der **Verdict** (roh vs. refine) ergibt sich jetzt aus der höheren effektiven ISK/h; für den Refine-Pfad löst „alle Minerale am besten Einzel-Hub" das frühere „pro Mineral egal wo" ab (Verdict/Ranking können sich dadurch ändern). **Fail-loud:** ist Standort, Schiff-Warp/Align oder eine Route nicht auflösbar, bleibt die Brutto-ISK/h stehen und die Zeile trägt den ≈-Schätzwert-Marker — nie ein stilles „0 Downtime". Web + Flutter. Greedy-1-Schritt ab dem aktuellen System; das volle Optimum (Max-Ratio-Cycle / Tramp-Steamer) ist als Follow-up #158 angelegt.

## [0.22.0] - 2026-06-01

### Added

- **Mining-ISK/h: Schiffs-Boni + Erz-Crystals eingerechnet (#157).** Die ISK/h des Mining-Rechners ist jetzt genau statt skills-only Untergrenze: sie berücksichtigt den **Hull-Mining-Yield-Bonus** des aktuellen Schiffs (Rollen- + per-Skill-Level-Bonus, data-driven aus der SDE-Dogma; Mining Barge / Exhumers / Mining Frigate) und ein **best-case T2-Erz-Crystal** pro Erz (wenn ein crystal-fähiger Miner gefittet ist). Zeilen, deren Bonus/Crystal nicht auflösbar ist (unbekanntes Schiff, exotischer Hull-Bonus wie Venture/Orca/Rorqual, oder Erz ohne passendes Crystal wie Mercoxit), tragen einen **≈-Schätzwert-Marker** mit Grund — nie ein still angenommenes 1.0 (fail-loud). Der Hull-Bonus skaliert alle Erze gleich (Ranking unverändert), Crystals sind erz-spezifisch (können umsortieren); die Roh-vs-Refine-Entscheidung bleibt bei beiden unberührt. Web + Flutter.

## [0.21.0] - 2026-06-01

### Added

- **Mining-Rechner: Wo aufbereiten & wo verkaufen (#156).** Jede Erz-Zeile der Mining-Rangliste ist jetzt aufklappbar (Web + Flutter) und zeigt die Verkaufs-/Aufbereitungs-Orte: beim **Reprozessieren** die beste NPC-Aufbereitungs-Station (Name — System) plus eine **pro-Mineral-Aufschlüsselung** (Mineral · effektive Menge nach Net-Yield · Buy-Preis · Verkaufsort des besten Buy-Orders); beim **Roh-Verkauf** den Verkaufsort des Roherzes. Der Verkaufsort ist jeweils das beste Buy-Order **egal wo** (inkl. Citadels — diese erscheinen als „Player-Structure", da die SDE sie nicht benennen kann).

## [0.20.1] - 2026-05-31

### Fixed

- **Mining-Rechner: Compressed-Erze raus aus der Rangliste (#155).** „Compressed …" und „Batch Compressed …"-Varianten werden nicht abgebaut (Roh-Erz wird danach komprimiert) und hatten ein winziges Volumen (0,001 m³ statt 0,1 m³), das ISK/m³ und ISK/h ~100× verzerrte. Die Erz-Liste filtert sie jetzt aus (451→206 Erztypen).

## [0.20.0] - 2026-05-31

### Added

- **Mining-Rechner: Erz roh verkaufen vs. reprozessieren (#154).** Neue „Mining"-View (Web + Flutter), die Erze nach **ISK/Stunde** rankt und je Erz sagt, ob sich **roh verkaufen** oder **reprozessieren + Raffinate verkaufen** mehr lohnt. Berücksichtigt Reprocessing-, Mining-, Schiffs- und Trade/Broker-Skills: Reprocessing-Yield aus den Skills, beste NPC-Station der Region nach deinem Standing (neuer ESI-`/standings`-Call), Mining-Rate (m³/h) aus dem aktuellen Schiff + gefitteten Minern + Skills, Taker-Pricing (höchste Buy-Order, nur Sales Tax, Order-Book-Cap), Sicherheitszone wählbar. Endpoint `POST /api/v1/mining/ore-ranking`. Hinweis: ISK/h ist eine skills-basierte Untergrenze — Schiffs-Rollenboni (Mining Barge/Exhumer) und Erz-Crystals sind noch nicht eingerechnet (Verdict + Ranking sind davon unberührt; im UI vermerkt).

## [0.19.0] - 2026-05-31

### Changed

- **Nur noch das aktuelle Schiff — keine Schiffsauswahl mehr (#153).** ROI, Trading und Hauling zeigen und verwenden jetzt ausschließlich das **aktuell geflogene** Schiff (read-only Karte mit **Refresh**) statt eines Auswahl-Dropdowns — im **Web und in der Flutter-App**. Jede Berechnung schickt das exakte Effektiv-Cargo des aktuellen Schiffs als `cargo_capacity`-Override (Hauling bekam den Override neu; das Fitting wird weiterhin für Warp-/Align-Speed geholt, kein ISK/h-Regress). Das alte `ShipSelect`-Dropdown + Mock-Schiffsliste sind entfernt.

### Added

- **ESI-Asset-Namen werden in Redis gecacht (#153).** Der ungecachte ESI-Endpoint `/characters/{id}/assets/names/` wird jetzt pro Character (TTL 1 h) zwischengespeichert. Fail-loud: ein ESI-Fehler wird **nicht** gecacht (kein 1-h-Verschleiern fehlender Namen).

## [0.18.0] - 2026-05-31

### Added

- **Instanz-genaues Schiff-Dropdown im ROI-Rechner (#152).** Das Dropdown listet jetzt jedes besessene Schiff **einzeln** (nicht mehr nach Typ dedupliziert), mit eigenem Schiffsnamen (ESI `/assets/names`) und dem effektiven Frachtraum **seines** tatsächlichen Fittings (pro `item_id`). Die Auswahl trägt das exakte Cargo der Instanz als `cargo_capacity`-Override in den Optimizer.

### Fixed

- **Falscher Effektiv-Frachtraum bei mehreren gleichartigen Schiffen (#152).** Bisher wurde der Effektivwert eines Schiffstyps aus dem *ersten* Exemplar dieses Typs berechnet — bei mehreren gleichartigen Schiffen mit unterschiedlichem Fitting also ein beliebiges; ein frachtraum-reduzierendes Modul (z. B. Reinforced Bulkheads) ergab so einen Wert *unter* der Basis (gemeldet: Iteron Mark V 4.9k statt in-game 6.09k). Jetzt rechnet jede Instanz — inklusive des geflogenen Schiffs (ROI-Default) — mit ihrem eigenen Fitting. Die deterministische Cargo-Rechnung selbst war korrekt; nur die Instanz-Auswahl war es nicht.

## [0.17.3] - 2026-05-31

### Changed

- **golangci-lint-Config auf v2 migriert und alle Findings bereinigt (#151).** Die `backend/.golangci.yml` war v1-Format und lud unter golangci-lint v2 nie — der Linter lief faktisch nicht. Migration auf v2 + Behebung aller 20 aufgetauchten Findings: u. a. SA1029 (Context-Key-Kollision → geteilter typisierter `services.CtxKey…` statt doppelter String-Consts in handlers+services), errcheck (`tx.Rollback`), `math.Pow(x,2)`→`x*x` (numerisch identisch) und diverse Stil-/Simplify-Fixes. Verhaltensneutral; der Pre-Commit-Hook führt golangci-lint jetzt wieder aus.

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
