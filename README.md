# EVE Online Profit Calculator

Web-App für Trading & Manufacturing Optimierung in EVE Online

## Was ist eve-o-provit?

Eine spezialisierte Web-Anwendung zur Gewinnmaximierung in EVE Online mit Fokus auf:

- 💰 **Trading & Market Analysis** - Intra-Region Routes, Inventory Sell Optimization, Live Market Data
- 🏭 **Manufacturing** *(geplant)* - T2/T3 Produktion, Blueprint Efficiency, Profit-Kalkulation

## Features

✅ **Trading Routes** - Intra-Region & Multi-Hub Trading Optimizer  
✅ **ROI Calculator** - Return on Investment für Trading-Opportunitäten  
✅ **EVE SSO Authentication** - PKCE-Flow (Backend-Token-Exchange, HttpOnly-Cookies)  
✅ **Live Market Data** - Echtzeit-Daten via EVE ESI API (Parallel Fetching, <9s für The Forge)  
✅ **Dual-Database** - PostgreSQL (dynamic) + SQLite SDE (static)  
🚧 **Manufacturing Module** - In Planung

## Quick Start

**Voraussetzungen:** Docker & Docker Compose

```bash
# Monorepo klonen
git clone https://github.com/Sternrassler/eveonline.git
cd eveonline/eve-o-provit

# Environment konfigurieren
cd backend
cp .env.example .env
# Bearbeite .env (SDE_PATH)
# EVE SSO wird im Frontend konfiguriert (.env.local)

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

**Frontend:** Next.js 16 (React 19, TypeScript, shadcn/ui)  
**Backend:** Go 1.24+ (Fiber, PostgreSQL, Redis)  
**Static Data:** SQLite SDE (via [eve-sde](https://github.com/Sternrassler/eve-sde))  
**ESI Client:** [eve-esi-client](https://github.com/Sternrassler/eve-esi-client)

Siehe [../docs/eve-o-provit.md](../docs/eve-o-provit.md) für Details

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

- [Architektur, API, DB-Schema & EVE-SSO-Flow](../docs/eve-o-provit.md) (kanonisch)
- [docs/](docs/) — Subprojekt-Index
- [Testing Guide](docs/testing/README.md)
- [Migration Testing](docs/testing/migrations.md)

## Verwandte Projekte

- [eve-sde](https://github.com/Sternrassler/eve-sde) - EVE Static Data Export Tools
- [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) - Go ESI API Client

## Lizenz

[MIT License](LICENSE)
