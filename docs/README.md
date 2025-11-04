# Dokumentations-Index

**Stand:** November 2025 | **Version:** v0.1.0

Dieser Index bietet eine Übersicht über die gesamte Projekt-Dokumentation.

## 📚 Hauptdokumentation

### Getting Started

- **[README.md](../README.md)** - Projekt-Übersicht, Quick Start, Features
- **[CHANGELOG.md](../CHANGELOG.md)** - Versions-Historie und Änderungen
- **[LICENSE](../LICENSE)** - MIT Lizenz

### Architektur & Design

- **[ARCHITECTURE.md](ARCHITECTURE.md)** ⭐ - Vollständige System-Architektur
- **[PROJECT_STRUCTURE.md](PROJECT_STRUCTURE.md)** - Code-Organisation & Package-Struktur
- **[EVE-SSO-INTEGRATION.md](EVE-SSO-INTEGRATION.md)** - OAuth2 Authentication Guide

### Entwicklung

- **[backend/MIGRATION.md](../backend/MIGRATION.md)** - evedb Package Migration von eve-sde
- **[frontend/README.md](../frontend/README.md)** - Frontend-Dokumentation
- **[deployments/README.md](../deployments/README.md)** - Docker Deployment Guide

## 🏛️ Architecture Decision Records (ADRs)

Aktuelle ADRs:

| ADR | Titel | Status | Datum |
|-----|-------|--------|-------|
| [ADR-001](adr/ADR-001-tech-stack.md) | Tech Stack | Proposed | 2025-10-24 |
| [ADR-004](adr/ADR-004-frontend-oauth-pkce.md) | Frontend OAuth PKCE | Accepted | 2025-10-25 |
| [ADR-009](adr/ADR-009-shared-redis-infrastructure.md) | Shared Redis Infrastructure | Proposed | 2025-10-31 |
| [ADR-010](adr/ADR-010-sde-database-path-convention.md) | SDE Database Path Convention | Accepted | 2025-10-31 |
| [ADR-011](adr/ADR-011-worker-pool-pattern.md) | Worker Pool Pattern | **Superseded** | 2025-11-01 |
| [ADR-012](adr/ADR-012-redis-caching-strategy.md) | Redis Caching Strategy | Accepted | 2025-11-01 |
| [ADR-013](adr/ADR-013-timeout-handling-partial-content.md) | Timeout Handling & Partial Content | Accepted | 2025-11-01 |

**Superseded ADRs:**

- ADR-011: Worker Pool Pattern → ersetzt durch eve-esi-client BatchFetcher (v0.3.0)

**Template:**

- [000-template.md](adr/000-template.md) - ADR Template für neue Entscheidungen

## 📦 Package-Dokumentation

### Backend Packages

- **[backend/cmd/README.md](../backend/cmd/README.md)** - Command-Line Tools
- **[backend/examples/README.md](../backend/examples/README.md)** - Example Programs
- **[backend/pkg/evedb/README.md](../backend/pkg/evedb/README.md)** - SDE Library (veraltet, siehe MIGRATION.md)
- **[backend/pkg/evesso/README.md](../backend/pkg/evesso/README.md)** - OAuth2 Package
- **[backend/internal/esi/README.md](../backend/internal/esi/README.md)** - ESI Client Wrapper

### Frontend

- **[frontend/tests/README.md](../frontend/tests/README.md)** - Playwright Test Guide

## 🧪 Testing

- **[docs/testing/migrations.md](testing/migrations.md)** - Database Migration Testing

## 🗄️ Archiv

Veraltete Dokumente (nur historische Referenz):

- **[archive/README.md](archive/README.md)** - Archiv-Index
- **[archive/IMPLEMENTATION_SUMMARY.md](archive/IMPLEMENTATION_SUMMARY.md)** - Frühe Backend-Implementierung
- **[archive/PHASE3_IMPLEMENTATION_SUMMARY.md](archive/PHASE3_IMPLEMENTATION_SUMMARY.md)** - Worker Pool Pattern (veraltet)

## 🔗 Externe Referenzen

### Verwandte Projekte

- **[eve-sde](https://github.com/Sternrassler/eve-sde)** - EVE Static Data Export Tools
- **[eve-esi-client](https://github.com/Sternrassler/eve-esi-client)** - Go Client Library für EVE ESI API

### EVE Online APIs

- **[EVE ESI Documentation](https://esi.evetech.net/ui/)** - ESI API Swagger UI
- **[EVE SSO Guide](https://docs.esi.evetech.net/docs/sso/)** - OAuth2 Integration
- **[EVE Developer Portal](https://developers.eveonline.com/)** - App Registration

### Technologie-Dokumentation

- **[Next.js 14](https://nextjs.org/docs)** - React Framework
- **[Go Fiber](https://docs.gofiber.io/)** - Web Framework
- **[PostgreSQL](https://www.postgresql.org/docs/)** - Database
- **[Redis](https://redis.io/docs/)** - Cache & Session Store

## 📝 Dokumentations-Richtlinien

### Neue Dokumentation erstellen

1. **ADRs:** Nutze Template `adr/000-template.md`
2. **README:** Markdown mit klarer Struktur (Headings, Code-Blocks)
3. **Code-Kommentare:** Deutsch (außer API/Lib-Namen)
4. **Commit Messages:** Conventional Commits (feat/fix/docs/chore)

### Veraltete Dokumente

- Status: Markiere als **Superseded** oder **Deprecated**
- Archivierung: Verschiebe nach `docs/archive/`
- Verweis: Füge Link zum neuen Dokument hinzu

### Review-Prozess

- **Minor Updates:** Direkt committen (Typos, Links)
- **Major Changes:** PR mit Review (Architektur, ADRs)
- **ADR Status:** Proposed → Accepted/Rejected → ggf. Superseded

## 🆘 Hilfe & Support

Bei Fragen oder Unklarheiten:

1. Prüfe relevante Dokumentation (siehe Index oben)
2. Suche in GitHub Issues: [eve-o-provit/issues](https://github.com/Sternrassler/eve-o-provit/issues)
3. Erstelle neues Issue mit Template (Bug/Feature)

## 🔄 Aktualisierungs-Historie

| Datum | Änderung | Autor |
|-------|----------|-------|
| 2025-11-04 | Dokumentation konsolidiert, ADR-011 superseded, ARCHITECTURE.md erstellt | GitHub Copilot |
| 2025-11-01 | ADR-011, ADR-012, ADR-013 hinzugefügt | GitHub Copilot |
| 2025-10-31 | ADR-009, ADR-010 hinzugefügt | GitHub Copilot |
| 2025-10-25 | EVE-SSO-INTEGRATION.md, ADR-004 hinzugefügt | GitHub Copilot |
| 2025-10-24 | Initial Projekt-Setup | GitHub Copilot |
