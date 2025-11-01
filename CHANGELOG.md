# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Performance Optimization Infrastructure (Phase 3)** - Intra-Region Trading Route Calculation
  - Worker pool for market order fetching (10 concurrent workers, ESI rate limit safe)
  - Worker pool for route calculation (50 concurrent workers for parallel processing)
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
- **ESI Client Integration** - eve-esi-client v0.2.0
  - Market orders fetching from ESI API
  - Redis-based caching (ADR-009 compliant)
  - Automatic rate limiting and error handling
- **Database Migrations** - golang-migrate setup
  - Migration 001: market_orders and price_history tables
  - Makefile targets for migration management
- **API Endpoints**
  - `GET /health` - Health check with database status
  - `GET /version` - API version information
  - `GET /api/v1/types/:id` - SDE type information lookup
  - `GET /api/v1/market/:region/:type` - Market orders with ESI integration
  - `POST /api/v1/trading/routes/calculate` - Trading route calculation with timeout support
- **Repository Pattern** - Clean architecture implementation
  - MarketRepository for PostgreSQL operations
  - SDERepository for SQLite read-only access
- **Documentation**
  - Updated README with complete setup instructions
  - API endpoint documentation
  - Docker Compose usage guide
  - Database migration guide
  - ADR-011: Worker Pool Pattern
  - ADR-012: Redis Caching Strategy
  - ADR-013: Timeout Handling (HTTP 206 Partial Content)

### Changed
- Updated `go.mod` with new dependencies (pgx/v5, go-redis/v9, eve-esi-client, golang.org/x/time/rate)
- Enhanced `.env.example` with all required environment variables
- Updated Makefile with database migration targets
- Refactored main.go for dual-database initialization
- RouteCalculator now supports Redis caching and parallel processing

### Technical
- ADR-001: Tech Stack - PostgreSQL + SQLite dual-DB confirmed
- ADR-009: Shared Redis Infrastructure - Implemented with key-namespacing
- ADR-011: Worker Pool Pattern - 10 workers for ESI, 50 for route calculation
- ADR-012: Redis Caching Strategy - 5min TTL, gzip compression
- ADR-013: Timeout Handling - HTTP 206 Partial Content pattern
- Dependencies: eve-esi-client v0.2.0, pgx/v5, go-redis/v9, golang-migrate/v4, golang.org/x/time/rate

### Performance
- Target: The Forge (383k orders) calculation < 30 seconds
- Cache hit ratio: > 95% after warmup
- Worker pools enable concurrent processing while respecting ESI rate limits
- Gzip compression reduces cache memory usage by ~80%

## [0.1.0] - 2025-10-05

- Project initialization.
