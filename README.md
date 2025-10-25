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
- **Language:** Go 1.21+
- **Framework:** Fiber (Fast HTTP Router)
- **Database ORM:** sqlc (Type-safe SQL)
- **API:** tRPC-Go / OpenAPI
- **Caching:** Redis (Market Data Cache)
- **Auth:** JWT (EVE SSO Integration)

### Datenbank
- **Primary:** PostgreSQL 16+
- **Optional:** TimescaleDB (Time-Series Market Data)

### Infrastructure
- **Containerization:** Docker + Docker Compose
- **Reverse Proxy:** Caddy (Auto-HTTPS)
- **Monitoring:** Prometheus + Grafana

### Datenquellen
- **EVE SDE:** SQLite DB (via eve-sde Projekt)
- **EVE ESI API:** Live Market Orders/History
- **Cache Layer:** Redis (ESI Rate Limiting)

> Siehe [ADR-001](docs/adr/ADR-001-tech-stack.md) für detaillierte Entscheidungsbegründung

## Projekt-Status

🚧 **Early Development** - Grundstruktur wird aufgebaut

## Verwandte Projekte

- [eve-sde](https://github.com/Sternrassler/eve-sde) - EVE Static Data Export Tools

## Getting Started

### Prerequisites

- **Go 1.21+**
- **Docker & Docker Compose**
- **Node.js 18+** (für Frontend, später)
- **eve-sde** SQLite Database

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

3. **SDE Database verlinken**
   ```bash
   # Aus eve-sde Projekt
   ln -s /path/to/eve-sde/data/sqlite/sde.sqlite data/sde/sde.sqlite
   ```

4. **Backend Dependencies installieren**
   ```bash
   cd backend
   go mod download
   cd ..
   ```

5. **Dev-Umgebung starten**
   ```bash
   cd deployments
   docker-compose up -d
   ```

6. **API testen**
   ```bash
   curl http://localhost:8080/health
   ```

### Development

**Backend entwickeln:**
```bash
cd backend
go run ./cmd/api
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
