# Dokumentation

**Version:** v0.1.0 | **Stand:** November 2025

## Schnellstart

- **[../README.md](../README.md)** - Quick Start & Features
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System-Architektur
- **[EVE-SSO-INTEGRATION.md](EVE-SSO-INTEGRATION.md)** - OAuth2 Setup

## Architecture Decision Records (ADRs)

Wichtige architektonische Entscheidungen:

| ADR | Titel | Status |
|-----|-------|--------|
| [ADR-001](adr/ADR-001-tech-stack.md) | Tech Stack | Proposed |
| [ADR-004](adr/ADR-004-frontend-oauth-pkce.md) | Frontend OAuth PKCE | Accepted |
| [ADR-009](adr/ADR-009-shared-redis-infrastructure.md) | Shared Redis Infrastructure | Proposed |
| [ADR-010](adr/ADR-010-sde-database-path-convention.md) | SDE Database Path Convention | Accepted |
| [ADR-012](adr/ADR-012-redis-caching-strategy.md) | Redis Caching Strategy | Accepted |
| [ADR-013](adr/ADR-013-timeout-handling-partial-content.md) | Timeout Handling | Accepted |
| [ADR-014](adr/ADR-014-esi-integration-pattern.md) | ESI Integration Pattern | Accepted |
| [ADR-015](adr/ADR-015-fitting-integration-architecture.md) | Fitting Integration | Accepted |

**Superseded:** ADR-011 (Worker Pool) → eve-esi-client BatchFetcher

## Testing

- **[testing/migrations.md](testing/migrations.md)** - Database Migration Tests (Testcontainers)

## Verwandte Projekte

- [eve-sde](https://github.com/Sternrassler/eve-sde) - EVE Static Data Export Tools
- [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) - Go ESI API Client

## Externe Ressourcen

- [EVE ESI Docs](https://esi.evetech.net/ui/) - API Reference
- [EVE SSO Guide](https://docs.esi.evetech.net/docs/sso/) - OAuth2
- [Next.js Docs](https://nextjs.org/docs) - Frontend Framework
- [Fiber Docs](https://docs.gofiber.io/) - Backend Framework
