# ADR-001: Tech Stack für eve-o-provit Web-App

Status: Proposed
Datum: 2025-10-25
Autoren: Team

## Kontext

Das Projekt eve-o-provit benötigt einen modernen, wartbaren Tech Stack für eine Web-Anwendung, die:

- **Trading & Market Analysis** mit Echtzeit-Daten (EVE ESI API)
- **Manufacturing & Industry Calculators** mit komplexen Berechnungen
- Integration mit **EVE SDE** statischen Daten (Blueprint-Daten, Items, Materials)
- Skalierbare Datenhaltung für **Market History & Calculations**

**Constraints:**
- Synergien mit bestehendem eve-sde Projekt (Go-basiert)
- Performante Berechnungen für Manufacturing-Chains
- Moderne Developer Experience
- Self-hosted Deployment-Fähigkeit

**Stakeholder:**
- Entwickler (DX, Wartbarkeit)
- End-User (Performance, UX)
- Operations (Deployment, Monitoring)

## Betrachtete Optionen

### Option 1: Go Full-Stack (Templ + HTMX)

- **Vorteile:**
  - Maximale Synergien mit eve-sde Projekt
  - Single Language (Go) für Frontend + Backend
  - Minimale JavaScript-Dependencies
  - Performante Server-Side Rendering
  - Einfaches Deployment (Single Binary)
- **Nachteile:**
  - Weniger interaktive UI-Patterns
  - HTMX Lernkurve für komplexe Interaktionen
  - Kleinere Community für Templ
- **Risiken:**
  - Eingeschränkte UI-Komplexität für Dashboard-Features
  - Weniger Third-Party UI-Komponenten

### Option 2: Next.js (React) + Go Backend

- **Vorteile:**
  - Moderne React-Ecosystem (UI Libraries, Charts)
  - TypeScript für Type Safety
  - Server Components für Performance
  - Große Community & Dokumentation
  - Go Backend nutzt eve-sde Code direkt
- **Nachteile:**
  - Zwei Sprachen (TypeScript + Go)
  - Komplexerer Build-Prozess
  - Node.js Deployment zusätzlich zu Go
- **Risiken:**
  - Höhere Infrastruktur-Komplexität
  - Größere Bundle-Sizes

### Option 3: SvelteKit + Go Backend

- **Vorteile:**
  - Kleinere Bundle-Sizes als React
  - Weniger Framework-Overhead
  - Moderne DX (Reactivity)
  - Go Backend nutzt eve-sde Code
- **Nachteile:**
  - Kleinere Community als React
  - Weniger Third-Party UI-Komponenten
  - Zwei Sprachen
- **Risiken:**
  - Längerfristige Ecosystem-Stabilität

## Entscheidung

**Gewählte Option:** **Option 2 - Next.js (React) + Go Backend**

**Begründung:**

1. **Komplexe UI-Anforderungen:** Trading Dashboards und Manufacturing-Calculators benötigen interaktive Charts, Tabellen, Filters → React Ecosystem bietet beste Tooling (Recharts, TanStack Table, etc.)
2. **TypeScript Safety:** Type-safe API-Integration zwischen Frontend/Backend über OpenAPI/tRPC
3. **Go Backend Integration:** Direkter Code-Reuse von eve-sde Projekt für SDE-Datenverarbeitung
4. **Developer Experience:** Große Community, gute Dokumentation, moderne Tooling
5. **Performance:** Next.js Server Components reduzieren Client-Side JavaScript wo möglich

**Tech Stack Details:**

**Frontend:**
- Framework: **Next.js 14+** (App Router, Server Components)
- Language: **TypeScript**
- UI Library: **shadcn/ui** (Radix UI + Tailwind CSS)
- State Management: **Zustand** (leichtgewichtig)
- Charts: **Recharts** / **Apache ECharts**
- Tables: **TanStack Table**
- API Client: **tRPC** (Type-safe API)

**Backend:**
- Language: **Go 1.21+**
- Framework: **Fiber** (Fast HTTP Router)
- API: **tRPC-Go** oder **OpenAPI/Swagger**
- Database ORM: **sqlc** (Type-safe SQL)
- Auth: **JWT** (evtl. EVE SSO Integration)
- Caching: **Redis** (Market Data Cache)

**Datenbank:**
- **SDE (Read-Only):** SQLite (direkt von eve-sde, inkl. Views)
- **Dynamic Data:** PostgreSQL 16+ (Market History, User Data)
- **Extensions:** TimescaleDB (optional für Time-Series Market Data)

**Infrastructure:**
- Containerization: **Docker** + **Docker Compose**
- Reverse Proxy: **Caddy** (Auto-HTTPS)
- Monitoring: **Prometheus** + **Grafana**

**Datenquellen:**
- **EVE SDE:** SQLite DB + Views (via eve-sde Projekt, Read-Only)
- **EVE ESI API:** Live Market Orders/History
- **Cache Layer:** Redis (ESI Rate Limiting)

**Code-Migration:**
- API-Funktionen aus eve-sde nach eve-o-provit verschieben
- Anpassung für Web-API Kontext (REST/tRPC Endpoints)

## Konsequenzen

### Positiv

- Moderne, wartbare Codebasis mit starkem Typing (TS + Go)
- Beste UI/UX-Möglichkeiten für komplexe Dashboards
- **Direkter Code-Reuse:** API-Funktionen aus eve-sde migrieren
- **SQLite Views nutzen:** Keine Duplikation statischer Daten
- Skalierbare Architektur (Frontend/Backend Separation)
- Self-Hosting freundlich (Docker Compose)
- **Hybrid-DB Ansatz:** SQLite (SDE) + PostgreSQL (Dynamic)

### Negativ

- Zwei Sprachen/Runtimes erhöhen Komplexität
- Node.js Build-Pipeline zusätzlich zu Go
- Größere Deployment-Footprint als Single Binary
- TypeScript-Go Type-Sync muss gepflegt werden
- **Zwei Datenbanksysteme:** SQLite + PostgreSQL (erhöht Ops-Komplexität)
- **Keine DB-übergreifenden JOINs:** SDE-Market JOINs nur in App-Logic

### Risiken

- **API Contract Drift:** Frontend/Backend Types müssen synchron bleiben
  - **Mitigation:** tRPC oder Code-Gen aus OpenAPI
- **Deployment Komplexität:** Zwei Services orchestrieren
  - **Mitigation:** Docker Compose für lokale Dev, orchestriert für Prod
- **Performance:** React-Overhead bei großen Datasets
  - **Mitigation:** Server Components, Virtualized Tables, Lazy Loading

## Implementierung

**Aufwand:** ~3-5 PT für Grundstruktur-Setup

**Abhängigkeiten:**
- ADR-002: API Design (tRPC vs. OpenAPI) (TODO)
- ADR-003: Deployment-Strategie (TODO)
- Integration mit eve-sde SQLite Schema

**Validierung:**
- Erfolg gemessen an:
  - Lokaler Dev-Setup in < 5 Minuten
  - Frontend-Build in < 30 Sekunden
  - API Response Time < 100ms (lokal)

**Migrations-Pfad:**
1. Next.js Projekt-Scaffold erstellen
2. Go Backend mit Fiber + sqlc aufsetzen
3. Docker Compose für Lokale Dev (PostgreSQL + Redis)
4. **SQLite SDE Integration:** 
   - eve-sde SQLite DB einbinden (Read-Only Mount)
   - Views aus eve-sde nutzen (Cargo-Volumen, Material-Chains, etc.)
5. **Code-Migration aus eve-sde:**
   - `pkg/evedb` Package nach eve-o-provit portieren
   - API-Funktionen anpassen für Web-Endpoints
   - Go Client-Code für ESI API aus eve-sde übernehmen
6. **PostgreSQL Schema:** Nur für dynamische Daten (Market, User)
7. ESI API Client implementieren/anpassen

## Referenzen

- **Projekte:** eve-sde (SQLite Schema, SDE Import Logic)
- **APIs:**
  - [EVE ESI API](https://esi.evetech.net/ui/)
  - [EVE SDE Download](https://developers.eveonline.com/resource/resources)
- **Tech Docs:**
  - [Next.js Docs](https://nextjs.org/docs)
  - [Go Fiber](https://docs.gofiber.io/)
  - [sqlc](https://docs.sqlc.dev/)
  - [shadcn/ui](https://ui.shadcn.com/)

## Notizen

**Alternative Stack (Falls Komplexität zu hoch):**
- Fallback zu Option 1 (Templ + HTMX) für MVP, später zu React migrieren
- Hybrid: HTMX für einfache Seiten, React für Dashboards

**EVE ESI Rate Limits:**
- 150 req/s burst, 20 req/s sustained
- Redis Cache Layer MUSS implementiert werden

**Datenbank-Strategie (Hybrid):**
```
┌─────────────────────┐
│ SQLite (SDE)        │  ← eve-sde Projekt (Read-Only)
│ - blueprints        │
│ - invTypes          │
│ - materials         │
│ - VIEWS (Cargo, etc)│
└─────────────────────┘
         ↓ Read-Only
┌─────────────────────┐
│ Go Backend          │  ← Code aus eve-sde migriert
│ - evedb Package     │
│ - ESI Client        │
│ - Business Logic    │
└─────────────────────┘
         ↓ Read/Write
┌─────────────────────┐
│ PostgreSQL          │
│ - market_orders     │
│ - price_history     │
│ - user_calculations │
│ - sessions          │
└─────────────────────┘
```

**Code-Migration aus eve-sde:**
- `pkg/evedb` → eve-o-provit (SQLite Access Layer)
- ESI Client Code (falls vorhanden)
- Navigation/Routing Logik (für Trade Routes)
- Cargo-Volume Berechnungen (aus Views)

---

**Change Log:**

- 2025-10-25: Status auf Proposed gesetzt (Team)
