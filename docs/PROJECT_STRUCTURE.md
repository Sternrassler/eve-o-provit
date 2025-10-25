# Projekt-Struktur

```
eve-o-provit/
├── backend/                    # Go Backend API
│   ├── cmd/
│   │   └── api/               # API Server Entrypoint
│   │       └── main.go
│   ├── internal/              # Private Application Code
│   │   ├── database/         # PostgreSQL Migrations & Queries
│   │   ├── handlers/         # HTTP Request Handlers
│   │   ├── services/         # Business Logic Layer
│   │   └── esi/              # ESI API Client
│   ├── pkg/
│   │   └── evedb/            # SQLite SDE Access (aus eve-sde migriert)
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
│
├── frontend/                   # Next.js 14 Frontend (TODO: scaffold)
│   └── .gitkeep
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
- **Go 1.21+** mit Fiber Framework
- **SQLite SDE Access** via `pkg/evedb` (aus eve-sde migriert)
- **PostgreSQL** für Market/User Data
- **Redis** für Caching
- **ESI Client** für Live Market Data

### Frontend (`/frontend`)
- **Next.js 14** (App Router, Server Components)
- **TypeScript** + **shadcn/ui**
- **TanStack Table** für Data Tables
- **Recharts** für Charts

### Data (`/data`)
- **SDE SQLite DB** (Read-Only, aus eve-sde Projekt)

### Deployments (`/deployments`)
- **Docker Compose** für lokale Entwicklung
- PostgreSQL + Redis + API (+ Frontend später)

## Setup

Siehe README.md im Root-Verzeichnis.
