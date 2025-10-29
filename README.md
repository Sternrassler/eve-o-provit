tech stack festlegen# EVE Online Profit Calculator

**Web-App für Trading & Manufacturing Optimierung in EVE Online**

## Vision

`eve-o-provit` ist eine spezialisierte Web-Anwendung zur Gewinnmaximierung in EVE Online, fokussiert auf:

- 💰 **Trading & Market Analysis** - Station Trading, Margin Trading, Buy/Sell-Order Optimierung
- 🏭 **Manufacturing & Industry** - T2/T3 Produktion, Capital Ship Building, Profit-Kalkulation

## Kernfunktionen

### Trading Module
- **Profit-Margin Analyse** - Echtzeit-Berechnung von Buy/Sell-Order Spreads
- **Market Hub Vergleiche** - Jita, Amarr, Dodixie, Rens Preisvergleich
- **Trade Route Finder** - Optimale Inter-Hub Arbitrage-Routes
- **Historical Price Trends** - Marktentwicklung und Volatilitäts-Analyse
- **Live Market Data** - Integration mit EVE ESI API

### Manufacturing Module
- **Blueprint Efficiency Calculator** - Material-/Zeit-/Kosten-Optimierung
- **Profit Calculator** - Material Cost vs. Market Price Analyse
- **T2/T3 Manufacturing Chains** - Komplette Produktionsketten-Planung
- **Capital Ship Production** - High-Investment Planning Tools
- **Industry Job Optimizer** - Multi-Charakter Job-Scheduling

## Tech Stack

### Frontend
- **Framework:** Next.js 14+ (App Router, Server Components)
- **Language:** TypeScript
- **UI Library:** shadcn/ui (Radix UI + Tailwind CSS)
- **State:** Zustand
- **Charts:** Recharts / Apache ECharts
- **Tables:** TanStack Table

### Backend
- **Language:** Go 1.24+
- **Framework:** Fiber (Fast HTTP Router)
- **Database:** 
  - PostgreSQL 16+ (Dynamic Market Data)
  - SQLite (Read-Only SDE from eve-sde)
- **ORM/Query:** pgx/v5, database/sql
- **API:** REST (tRPC/OpenAPI geplant)
- **Caching:** Redis (ESI Cache & Rate Limiting)
- **ESI Client:** [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) v0.2.0
- **Migrations:** golang-migrate
- **Auth:** JWT (EVE SSO Integration)

### Datenbank
- **Dynamic Data:** PostgreSQL 16+ (Market Orders, Price History, User Data)
- **Static Data:** SQLite (Read-Only SDE from [eve-sde](https://github.com/Sternrassler/eve-sde))
- **Optional:** TimescaleDB Extension (Time-Series Market Data)

### Infrastructure
- **Containerization:** Docker + Docker Compose
- **Reverse Proxy:** Caddy (Auto-HTTPS) - geplant
- **Monitoring:** Prometheus + Grafana - geplant

### Datenquellen
- **EVE SDE:** SQLite DB (via [eve-sde](https://github.com/Sternrassler/eve-sde) Projekt, Read-Only)
- **EVE ESI API:** Live Market Orders/History via [eve-esi-client](https://github.com/Sternrassler/eve-esi-client)
- **Cache Layer:** Redis (ESI Response Cache + Rate Limit Tracking)

> Siehe [ADR-001](docs/adr/ADR-001-tech-stack.md) für detaillierte Entscheidungsbegründung

## Projekt-Status

🚀 **Backend Foundation Complete** 
- ✅ Dual-DB Architecture (PostgreSQL + SQLite SDE)
- ✅ ESI Client Integration (eve-esi-client v0.2.0)
- ✅ Basic API Endpoints (Health, Version, Types, Market)
- ✅ Docker Compose Setup
- ✅ Database Migrations
- 🚧 Frontend (Next.js) - In Planung
- 🚧 Advanced Trading Features - In Planung

## Verwandte Projekte

- [eve-sde](https://github.com/Sternrassler/eve-sde) - EVE Static Data Export Tools
- [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) - Go Client Library for EVE ESI API

## API Endpoints

### Public Endpoints

- `GET /health` - Health check (with database status)
- `GET /version` - API version information
- `GET /api/v1/types/:id` - SDE type lookup (Items, Ships, etc.)
- `GET /api/v1/market/:region/:type` - Market orders for region and type
  - Query param: `?refresh=true` to fetch fresh data from ESI

### Protected Endpoints (EVE SSO required)

- `GET /api/v1/character` - Character information
- `GET /api/v1/trading/profit-margins` - Profit margin calculations (TODO)
- `GET /api/v1/manufacturing/blueprints` - Blueprint data (TODO)

Weitere Details zur API-Nutzung und Authentifizierung siehe [docs/EVE-SSO-INTEGRATION.md](docs/EVE-SSO-INTEGRATION.md)

## Getting Started

### Prerequisites

- **Go 1.24+**
- **Docker & Docker Compose**
- **golang-migrate** (für Datenbank-Migrationen)
- **Node.js 18+** (für Frontend, später)
- **eve-sde** SQLite Database (optional für lokale Entwicklung)

**golang-migrate Installation:**
```bash
# macOS (Homebrew)
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/

# Windows (Scoop)
scoop install migrate

# Oder via Go
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Quick Start

1. **Repository clonen**
   ```bash
   git clone https://github.com/Sternrassler/eve-o-provit.git
   cd eve-o-provit
   ```

2. **Git Hooks aktivieren**
   ```bash
   git config core.hooksPath .githooks
   ```

3. **Environment-Datei erstellen**
   ```bash
   cd backend
   cp .env.example .env
   # Bearbeite .env und passe Werte an (vor allem SDE_PATH)
   ```

4. **Docker Services starten**
   ```bash
   # Von Repository-Root
   make docker-up
   ```
   
   Dies startet:
   - PostgreSQL (Port 5432)
   - Redis (Port 6379)
   - Backend API (Port 9001)
   - Frontend (Port 9000)

5. **Datenbank Migrations ausführen**
   ```bash
   make migrate
   ```

6. **SDE Database verlinken (optional für lokale Entwicklung)**
   ```bash
   # Falls du das Backend lokal (ohne Docker) entwickeln möchtest
   ln -s /path/to/eve-sde/data/sqlite/sde.sqlite backend/data/sde/sde.sqlite
   ```

7. **API testen**
   ```bash
   # Health Check
   curl http://localhost:9001/health
   
   # Version Info
   curl http://localhost:9001/version
   
   # SDE Type Lookup (Tritanium = 34)
   curl http://localhost:9001/api/v1/types/34
   
   # Market Orders (Jita = 10000002, Tritanium = 34)
   curl "http://localhost:9001/api/v1/market/10000002/34?refresh=true"
   ```

### Development

**Backend lokal entwickeln (ohne Docker):**
```bash
cd backend

# Stelle sicher, dass PostgreSQL und Redis laufen
make docker-up  # Oder starte sie separat

# Environment-Variablen setzen (siehe .env.example)
export DATABASE_URL="postgresql://eveprovit:dev@localhost:5432/eveprovit?sslmode=disable"
export REDIS_URL="redis://localhost:6379/0"
export SDE_PATH="../eve-sde/data/sqlite/sde.sqlite"
export ESI_USER_AGENT="eve-o-provit/0.1.0 (your-email@example.com)"

# API starten
go run ./cmd/api
```

**Tests ausführen:**
```bash
# Alle Tests
make test

# Nur Backend Unit-Tests
make test-be-unit

# Linting
make lint
```

**Docker Commands:**
```bash
# Alle Services starten
make docker-up

# Logs anzeigen
make docker-logs

# Services stoppen
make docker-down

# In Container-Shell wechseln
make docker-shell-api    # Backend API
make docker-shell-db     # PostgreSQL
make docker-shell-redis  # Redis CLI
```

**Datenbank Migrations:**
```bash
# Migrations ausführen
make migrate

# Neue Migration erstellen
make migrate-create NAME=add_new_table

# Letzte Migration zurückrollen
make migrate-down
```

**Frontend entwickeln (TODO):**
```bash
cd frontend
npm run dev
```

### Projekt-Struktur

Siehe [docs/PROJECT_STRUCTURE.md](docs/PROJECT_STRUCTURE.md)

## Lizenz

Dieses Projekt steht unter der [MIT-Lizenz](LICENSE).
