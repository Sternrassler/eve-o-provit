# Projekt-Struktur

```
eve-o-provit/
├── backend/                    # Go Backend API
│   ├── cmd/
│   │   └── api/               # API Server Entrypoint
│   │       └── main.go
│   ├── internal/              # Private Application Code
│   │   ├── database/         # PostgreSQL Repository Layer
│   │   ├── handlers/         # HTTP Request Handlers
│   │   ├── models/           # Domain Models
│   │   └── services/         # Business Logic Layer
│   │       ├── route_calculator.go  # Trading Route Calculator
│   │       └── cache.go            # Market Data Cache (TODO: refactor)
│   ├── pkg/
│   │   ├── evedb/            # SQLite SDE Access Library
│   │   │   ├── cargo/        # Cargo/Hauling Calculations
│   │   │   └── navigation/   # Pathfinding & Travel Time
│   │   ├── esi/              # ESI Client Wrapper
│   │   └── evesso/           # EVE SSO OAuth2
│   ├── examples/             # CLI Examples
│   │   ├── cargo/            # Cargo Calculator Demo
│   │   └── navigation/       # Route Planning Demo
│   ├── migrations/           # Database Schema Migrations
│   ├── sql/views/            # PostgreSQL View Definitions
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/                   # Next.js 14 Frontend
│   ├── src/
│   │   ├── app/              # App Router Pages
│   │   │   ├── intra-region/         # Intra-Region Trading
│   │   │   ├── inventory-sell/       # Inventory Sell Optimizer
│   │   │   ├── character/            # Character Management
│   │   │   └── callback/             # OAuth Callback Handler
│   │   ├── components/
│   │   │   ├── ui/                   # shadcn/ui Components
│   │   │   ├── trading/              # Trading-Specific Components
│   │   │   │   ├── RegionSelect.tsx
│   │   │   │   ├── RegionRefreshButton.tsx
│   │   │   │   ├── RegionStalenessIndicator.tsx
│   │   │   │   ├── TradingRouteList.tsx
│   │   │   │   └── TradingFilters.tsx
│   │   │   └── item-autocomplete.tsx
│   │   ├── lib/
│   │   │   ├── api-client.ts         # Backend API Client
│   │   │   └── auth-context.tsx      # Auth State Management
│   │   └── types/
│   │       └── trading.ts            # TypeScript Type Definitions
│   ├── tests/                # Playwright E2E Tests
│   ├── package.json
│   └── Dockerfile
│
├── data/
│   └── sde/                   # SQLite SDE Database (Read-Only Mount)
│       └── .gitkeep
│
├── deployments/
│   └── docker-compose.yml     # Lokale Dev-Umgebung
│
├── docs/
│   └── adr/                   # Architecture Decision Records
│       ├── 000-template.md
│       └── ADR-001-tech-stack.md
│
├── scripts/                    # Engineering Scripts (aus eve-sde synchronisiert)
│   ├── common/
│   ├── github/
│   └── workflows/
│
├── .github/
│   ├── workflows/             # CI/CD Workflows
│   ├── copilot-instructions.md
│   └── ISSUE_TEMPLATE/
│
├── .githooks/                 # Pre-commit Hooks
│
├── Makefile                   # Build & Automation Targets
├── README.md
├── CHANGELOG.md
├── VERSION
└── LICENSE
```

## Komponenten

### Backend (`/backend`)

- **Go 1.24+** mit Fiber Framework
- **SQLite SDE Access** via `pkg/evedb` (migriert von eve-sde)
- **PostgreSQL** für Market/User Data (pgx/v5)
- **Redis** für ESI Response Caching
- **eve-esi-client v0.3.0** für Live Market Data (BatchFetcher Pattern)
- **EVE SSO** OAuth2 Authentication (pkg/evesso)

**Key Features:**

- Parallel Market Data Fetching (10 workers, ~8.7s für 387 Seiten)
- Intra-Region Trading Route Calculator
- Inventory Sell Optimizer
- Market Data Staleness Tracking
- Character Location & Ship Integration

### Frontend (`/frontend`)

- **Next.js 14** (App Router, Server Components)
- **TypeScript** + **shadcn/ui** (Radix UI + Tailwind)
- **Auth:** EVE SSO Integration (OAuth2, JWT Sessions)
- **State:** React Context API
- **Components:**
  - Region Selection mit Refresh & Staleness Indicator
  - Trading Route Visualization
  - Item Autocomplete Search
  - Inventory Sell Calculator
  - Character Integration

### Data (`/data`)

- **SDE SQLite DB** (Read-Only, aus eve-sde Projekt)
- Enthält: Types, Regions, Systems, Stargates, Ships, Blueprints

### Deployments (`/deployments`)

- **Docker Compose** für lokale Entwicklung
- Services: PostgreSQL + Redis + API + Frontend
- Ports: 5432, 6379, 9001, 9000

## Setup

Siehe README.md im Root-Verzeichnis.
