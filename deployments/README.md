# Docker Compose Setup

Lokales Entwicklungs-Setup mit PostgreSQL, Redis und Backend API.

## Services

### PostgreSQL 16
- **Port**: 5432
- **User**: `eveprovit`
- **Password**: `devpassword`
- **Database**: `eveprovit`
- **Volume**: `postgres_data` (persistent)

**Schema**:
- Users & Authentication
- Market Orders (cached from ESI)
- Market History
- Watchlists
- Profit Calculations

### Redis 7
- **Port**: 6379
- **Persistence**: AOF enabled
- **Volume**: `redis_data` (persistent)

**Use Cases**:
- API Response Caching
- Session Storage
- Rate Limiting
- Market Data Cache

### Backend API (Go + Fiber)
- **Port**: 8080
- **Health Check**: `http://localhost:8080/health`
- **Dependencies**:
  - PostgreSQL (market/user data)
  - Redis (caching)
  - SDE SQLite (static game data via symlink)

**Environment Variables**:
```bash
PORT=8080
DATABASE_URL=postgres://eveprovit:devpassword@postgres:5432/eveprovit?sslmode=disable
REDIS_URL=redis://redis:6379/0
SDE_DB_PATH=/data/sde/eve-sde.db
CORS_ORIGINS=http://localhost:3000,http://localhost:5173
LOG_LEVEL=debug
ENV=development
```

## Quick Start

### 1. Starte alle Services
```bash
make docker-up
```

**Output**:
```
Services verfügbar unter:
  - Backend API:  http://localhost:8080
  - PostgreSQL:   localhost:5432 (User: eveprovit, DB: eveprovit)
  - Redis:        localhost:6379
```

### 2. Prüfe Status
```bash
make docker-ps
```

### 3. Zeige Logs
```bash
# Alle Services
make docker-logs

# Nur Backend
make docker-logs SERVICE=api

# Nur PostgreSQL
make docker-logs SERVICE=postgres
```

### 4. Teste Backend Health
```bash
curl http://localhost:8080/health
```

**Expected Response**:
```json
{
  "status": "ok",
  "service": "eve-o-provit-api"
}
```

## Development Workflow

### Rebuild nach Code-Änderungen
```bash
make docker-restart
```

### Database Shell
```bash
make docker-shell-db
```

**SQL Queries**:
```sql
-- Alle Tabellen anzeigen
\dt

-- Market Orders prüfen
SELECT COUNT(*) FROM market_orders;

-- Watchlists anzeigen
SELECT * FROM watchlists;
```

### Redis CLI
```bash
make docker-shell-redis
```

**Redis Commands**:
```bash
# Keys anzeigen
KEYS *

# Cache Stats
INFO stats

# Flushall (Development only!)
FLUSHALL
```

### Backend Container Shell
```bash
make docker-shell-api
```

## Cleanup

### Services stoppen
```bash
make docker-down
```

### Alles entfernen (inkl. Volumes!)
```bash
make docker-clean
```

⚠️ **Warnung**: `docker-clean` löscht alle Daten (PostgreSQL + Redis)!

## Troubleshooting

### Port bereits belegt
```bash
# Prüfe welche Ports belegt sind
lsof -i :8080  # Backend
lsof -i :5432  # PostgreSQL
lsof -i :6379  # Redis
```

### Container startet nicht
```bash
# Logs prüfen
make docker-logs SERVICE=api

# Image neu bauen
make docker-build
make docker-up
```

### Database Migration fehlgeschlagen
```bash
# PostgreSQL Logs prüfen
make docker-logs SERVICE=postgres

# Manuell SQL ausführen
make docker-shell-db
\i /docker-entrypoint-initdb.d/01-init.sql
```

### SDE Database nicht gefunden
```bash
# Symlink prüfen
ls -lh backend/data/sde/eve-sde.db

# Sollte zeigen:
# eve-sde.db -> ../../../eve-sde/data/sqlite/eve-sde.db
```

## Network

Alle Services laufen im gleichen Docker Network `eve-network`:
- Interne DNS-Resolution funktioniert (z.B. `postgres:5432`)
- Services können sich gegenseitig erreichen
- Frontend (wenn aktiviert) kann Backend über `api:8080` erreichen

## Production Differences

**Development (docker-compose.yml)**:
- Ports exposed (für lokalen Zugriff)
- Debug Logging
- Schwache Passwords
- Volume Mounts für Live-Reload

**Production (TODO: docker-compose.prod.yml)**:
- Keine Port-Exposition außer Reverse Proxy
- Error Logging only
- Starke Secrets (via Environment/Secrets)
- Read-only Filesystems wo möglich
- Resource Limits (CPU/Memory)
- Health Checks mit Timeouts
- Restart Policies

## Makefile Targets

| Target | Beschreibung |
|--------|--------------|
| `make docker-up` | Startet alle Services |
| `make docker-down` | Stoppt alle Services |
| `make docker-logs` | Zeigt Logs (SERVICE=name für einzelnen) |
| `make docker-ps` | Status aller Container |
| `make docker-build` | Rebuild alle Images |
| `make docker-clean` | Entfernt Container + Volumes + Images |
| `make docker-restart` | Down → Up |
| `make docker-shell-api` | Shell im Backend Container |
| `make docker-shell-db` | PostgreSQL psql CLI |
| `make docker-shell-redis` | Redis CLI |

## Next Steps

1. ✅ PostgreSQL Schema initialisiert
2. ✅ Redis Cache Layer bereit
3. ✅ Backend API startet
4. 🔲 API Endpoints für Cargo/Navigation implementieren
5. 🔲 Frontend Container hinzufügen
6. 🔲 ESI Market Data Import Cron
7. 🔲 Production docker-compose.yml
