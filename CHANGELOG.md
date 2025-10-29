# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
- **Repository Pattern** - Clean architecture implementation
  - MarketRepository for PostgreSQL operations
  - SDERepository for SQLite read-only access
- **Documentation**
  - Updated README with complete setup instructions
  - API endpoint documentation
  - Docker Compose usage guide
  - Database migration guide

### Changed
- Updated `go.mod` with new dependencies (pgx/v5, go-redis/v9, eve-esi-client)
- Enhanced `.env.example` with all required environment variables
- Updated Makefile with database migration targets
- Refactored main.go for dual-database initialization

### Technical
- ADR-001: Tech Stack - PostgreSQL + SQLite dual-DB confirmed
- ADR-009: Shared Redis Infrastructure - Implemented with key-namespacing
- Dependencies: eve-esi-client v0.2.0, pgx/v5, go-redis/v9, golang-migrate/v4

## [0.1.0] - 2025-10-05

- Project initialization.
