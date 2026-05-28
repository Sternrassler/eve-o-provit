# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
