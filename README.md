# EVE Online Profit Calculator

## Web-App für Trading & Manufacturing Optimierung in EVE Online

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
- **ESI Client:** [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) v0.3.0 (BatchFetcher Pattern)
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

🚀 **Production Ready - v0.1.0**

- ✅ Dual-DB Architecture (PostgreSQL + SQLite SDE)
- ✅ ESI Client Integration (eve-esi-client v0.3.0 mit BatchFetcher)
- ✅ Frontend (Next.js 14 mit Trading UI)
- ✅ EVE SSO Authentication (OAuth2)
- ✅ Intra-Region Trading Routes
- ✅ Inventory Sell Optimization
- ✅ Market Data Refresh (Parallel Fetching, 8.7s für ~1.2M Orders)
- ✅ Docker Compose Setup
- ✅ Database Migrations
- 🚧 Manufacturing Module - Geplant
- 🚧 Multi-Region Trading - Geplant

## Verwandte Projekte

- [eve-sde](https://github.com/Sternrassler/eve-sde) - EVE Static Data Export Tools
- [eve-esi-client](https://github.com/Sternrassler/eve-esi-client) - Go Client Library for EVE ESI API

## API Endpoints

### Public Endpoints

**Health & Info:**

- `GET /health` - Health check (database status)
- `GET /version` - API version information

**SDE Data:**

- `GET /api/v1/types/:id` - SDE type lookup (Items, Ships, etc.)
- `GET /api/v1/sde/regions` - List all regions

**Market Data:**

- `GET /api/v1/market/:region/:type` - Market orders for region and type
  - Query param: `?refresh=true` - Fetch fresh data from ESI (parallel, ~8.7s für The Forge)
- `GET /api/v1/market/staleness/:region` - Market data age/staleness indicator

**Trading Routes:**

- `POST /api/v1/trading/routes/calculate` - Calculate intra-region trading routes
- `GET /api/v1/items/search` - Search items by name (autocomplete)

### Protected Endpoints (EVE SSO required)

**Authentication:**

- `GET /api/v1/auth/login` - Initiate EVE SSO login
- `GET /api/v1/auth/callback` - OAuth callback handler
- `GET /api/v1/auth/verify` - Verify current session
- `POST /api/v1/auth/logout` - Logout

**Character:**

- `GET /api/v1/character` - Character information
- `GET /api/v1/character/location` - Current character location
- `GET /api/v1/character/ship` - Current ship
- `GET /api/v1/character/ships` - Available ships

**Trading:**

- `POST /api/v1/trading/inventory-sell` - Calculate best sell locations for inventory
- `POST /api/v1/esi/ui/autopilot/waypoint` - Set autopilot waypoint

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
   - PostgreSQL (Port 5432) - Persistent market data
   - Redis (Port 6379) - ESI caching
   - Backend API (Port 9001) - Go/Fiber REST API
   - Frontend (Port 9000) - Next.js 14 Web UI

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
   
   # Market Orders (The Forge = 10000002, Tritanium = 34)
   # Normal query (from cache/DB)
   curl http://localhost:9001/api/v1/market/10000002/34
   
   # Refresh all market data for region (parallel fetch, ~45s)
   curl "http://localhost:9001/api/v1/market/10000002/34?refresh=true"
   
   # Check market data age
   curl http://localhost:9001/api/v1/market/staleness/10000002
   ```

8. **Frontend öffnen**

   ```bash
   # Intra-Region Trading
   http://localhost:9000/intra-region
   
   # Inventory Sell Optimization
   http://localhost:9000/inventory-sell
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

# Migration Integration Tests (mit Testcontainers)
make test-migrations

# Linting
make lint
```

> **Migration Testing:** Siehe [docs/testing/migrations.md](docs/testing/migrations.md) für ausführliche Dokumentation zu Migration Tests mit Testcontainers.

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
