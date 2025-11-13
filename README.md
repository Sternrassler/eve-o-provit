# EVE Online Profit Calculator

Web-App für Trading & Manufacturing Optimierung in EVE Online

## Was ist eve-o-provit?

Eine spezialisierte Web-Anwendung zur Gewinnmaximierung in EVE Online mit Fokus auf:

- 💰 **Trading & Market Analysis** - Intra-Region Routes, Inventory Sell Optimization, Live Market Data
- 🏭 **Manufacturing** *(geplant)* - T2/T3 Produktion, Blueprint Efficiency, Profit-Kalkulation

## Features

✅ **Intra-Region Trading Routes** - Optimale Buy-Low/Sell-High Opportunities innerhalb einer Region  
✅ **Inventory Sell Optimization** - Bester Verkaufsort für dein aktuelles Inventar  
✅ **EVE SSO Authentication** - Sichere Integration mit EVE Online Account  
✅ **Live Market Data** - Echtzeit-Daten via EVE ESI API (Parallel Fetching, <9s für The Forge)  
✅ **Dual-Database** - PostgreSQL (dynamic) + SQLite SDE (static)  
🚧 **Manufacturing Module** - In Planung

## Quick Start

**Voraussetzungen:** Docker & Docker Compose

```bash
# Repository klonen
git clone https://github.com/Sternrassler/eve-o-provit.git
cd eve-o-provit

# Environment konfigurieren
cd backend
cp .env.example .env
# Bearbeite .env (SDE_PATH, EVE SSO Credentials)

# Services starten
cd ..
make docker-up

# Datenbank migrieren
make migrate

# Fertig! Öffne http://localhost:9000
```

**Frontend:** http://localhost:9000  
**Backend API:** http://localhost:9001

## Architektur

**Frontend:** Next.js 14 (TypeScript, Radix UI, TanStack Table)  
**Backend:** Go 1.24+ (Fiber, PostgreSQL, Redis)  
**Static Data:** SQLite SDE (via [eve-sde](https://github.com/Sternrassler/eve-sde))  
**ESI Client:** [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) v0.3.0

Siehe [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) für Details

## Entwicklung

### Docker Commands

```bash
make docker-up      # Services starten
make docker-logs    # Logs anzeigen
make docker-down    # Services stoppen
```

### Backend lokal (ohne Docker)

```bash
cd backend
export DATABASE_URL="postgresql://eveprovit:dev@localhost:5432/eveprovit?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export SDE_PATH="../eve-sde/data/sqlite/sde.sqlite"
go run ./cmd/api
```

### Tests & Linting

```bash
make test           # Alle Tests
make lint           # Linting
make migrate-create NAME=add_table  # Neue Migration
```

## Dokumentation

- [Projekt-Struktur](docs/PROJECT_STRUCTURE.md)
- [Architektur](docs/ARCHITECTURE.md)
- [EVE SSO Integration](docs/EVE-SSO-INTEGRATION.md)
- [API Endpoints](docs/api/)
- [ADRs (Architektur-Entscheidungen)](docs/adr/)
- [Migration Testing](docs/testing/migrations.md)

## Verwandte Projekte

- [eve-sde](https://github.com/Sternrassler/eve-sde) - EVE Static Data Export Tools
- [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) - Go ESI API Client

## Lizenz

[MIT License](LICENSE)
