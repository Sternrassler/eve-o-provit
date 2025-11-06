# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
