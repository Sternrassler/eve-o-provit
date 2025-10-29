# Migration Testing Documentation

## Überblick

Dieses Dokument beschreibt die automatisierten Tests für Database Migrations im eve-o-provit Projekt. Die Tests verwenden **Testcontainers** für vollständige Integration Tests mit PostgreSQL und Redis.

## Architektur

### Testcontainers Integration

Die Migration-Tests verwenden [Testcontainers for Go](https://golang.testcontainers.org/) um isolierte, reproduzierbare Testumgebungen zu erstellen:

- **PostgreSQL 16 Alpine** - Für Datenbank-Migrations-Tests
- **Redis 7 Alpine** - Für ESI Client Cache-Tests
- **Automatische Cleanup** - Container werden nach Tests automatisch entfernt

### Test-Kategorien

#### 1. Migration Tests (`migrations_test.go`)

| Test | Beschreibung | Validierung |
|------|-------------|-------------|
| `TestMigrationUp` | Führt UP Migration aus | Tabellen & Indizes existieren |
| `TestMigrationDown` | Führt DOWN Migration aus | Tabellen wurden entfernt |
| `TestMigrationReUp` | UP → DOWN → UP Cycle | Schema nach Re-UP korrekt |
| `TestMigrationIdempotency` | UP 2x ausführen | Keine Fehler bei wiederholter Ausführung |
| `TestSchemaValidation` | Schema-Struktur prüfen | Spalten, Indizes, Constraints |
| `TestDataIntegrity` | INSERT/SELECT nach Migration | Daten korrekt gespeichert |
| `TestMigrationStatus` | `schema_migrations` Tabelle | Version korrekt, dirty=false |

#### 2. Market Repository Tests (`market_test.go`)

| Test | Beschreibung |
|------|-------------|
| `TestMarketRepository_UpsertMarketOrders` | Einfügen/Aktualisieren von Market Orders |
| `TestMarketRepository_GetMarketOrders` | Abrufen von Market Orders nach Region/Type |
| `TestMarketRepository_CleanOldMarketOrders` | Löschen alter Orders |
| `TestMarketRepository_UpsertConflict` | ON CONFLICT Verhalten testen |
| `TestMarketRepository_BatchInsert` | Performance bei 100+ Orders |

#### 3. ESI Client Tests (`client_test.go`)

| Test | Status | Beschreibung |
|------|--------|-------------|
| `TestClient_FetchMarketOrders_Mock` | ⏳ Geplant | ESI API Mock mit Redis Cache |
| `TestClient_FetchMarketOrders_ErrorHandling` | ⏳ Geplant | Error Handling & Retries |
| `TestRedisConfig` | ⏳ Geplant | Redis Konfiguration & Expiration |

## Lokal Ausführen

### Voraussetzungen

```bash
# Docker muss laufen (für Testcontainers)
docker --version

# golang-migrate installieren
cd backend
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### Alle Migration Tests ausführen

```bash
# Via Make Target (empfohlen)
make test-migrations

# Direkt mit go test
cd backend
go test -v -run TestMigration ./internal/database/
```

### Einzelne Tests ausführen

```bash
cd backend

# Nur Migration UP/DOWN Tests
go test -v -run TestMigrationUp ./internal/database/
go test -v -run TestMigrationDown ./internal/database/

# Nur Schema Validation
go test -v -run TestSchemaValidation ./internal/database/

# Market Repository Tests
go test -v -run TestMarketRepository ./internal/database/
```

### Alle Integration Tests

```bash
cd backend
go test -v ./internal/database/...
```

### Performance Benchmarks

```bash
cd backend
go test -bench=. -benchmem ./internal/database/
```

## Beispiel-Ausgaben

### Erfolgreicher Migration UP Test

```
=== RUN   TestMigrationUp
2025/10/29 16:53:51 🐳 Creating container for image postgres:16-alpine
2025/10/29 16:53:59 ✅ Container created: 1b7c3a7d9218
2025/10/29 16:54:01 🔔 Container is ready: 1b7c3a7d9218
    migrations_test.go:34: Migration output: 1/u create_market_tables (19.623329ms)
2025/10/29 16:54:02 🚫 Container terminated: 1b7c3a7d9218
--- PASS: TestMigrationUp (11.20s)
PASS
```

### Schema Validation Output

```
=== RUN   TestSchemaValidation
    migrations_test.go:175: Migration output: 1/u create_market_tables (19.317441ms)
--- PASS: TestSchemaValidation (2.24s)
```

### Market Repository Batch Insert

```
=== RUN   TestMarketRepository_BatchInsert
    market_test.go:381: Migration output: 1/u create_market_tables (19.324875ms)
    market_test.go:415: Batch insert of 100 orders took 35.166883ms
--- PASS: TestMarketRepository_BatchInsert (1.96s)
```

## GitHub Actions Workflow

### Trigger

Der Migration Tests Workflow läuft automatisch bei:

- **Pull Requests** die folgende Dateien ändern:
  - `backend/migrations/**`
  - `backend/internal/database/**`
  - `backend/go.mod` / `backend/go.sum`
  - `.github/workflows/test-migrations.yml`
- **Manuell** via Workflow Dispatch

### Workflow Steps

1. **Checkout Code**
2. **Setup Go 1.24**
3. **Install golang-migrate**
4. **Run Migration UP**
5. **Run Integration Tests** (Testcontainers)
6. **Validate Schema** (SQL Queries)
7. **Test Rollback** (DOWN)
8. **Test Re-Apply** (UP)
9. **Generate Test Report** (JSON + Markdown)
10. **Upload Artifact**
11. **Comment PR** mit Ergebnissen

### Test Report Download

Nach jedem Workflow-Run ist ein Test Report als Artifact verfügbar:

- **Name:** `migration-test-report-{run_number}`
- **Dateien:**
  - `migration-test-report.md` - Human-readable Markdown
  - `migration-test-report.json` - Machine-readable JSON
  - `test-output.log` - Vollständige Test-Ausgabe
- **Retention:** 30 Tage

## Troubleshooting

### Docker nicht verfügbar

**Problem:** `Cannot connect to the Docker daemon`

**Lösung:**
```bash
# Docker starten
sudo systemctl start docker

# Berechtigung prüfen
sudo usermod -aG docker $USER
newgrp docker
```

### migrate Binary nicht gefunden

**Problem:** `migrate: command not found`

**Lösung:**
```bash
cd backend
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# PATH erweitern
export PATH="$HOME/go/bin:$PATH"

# Dauerhaft in ~/.bashrc oder ~/.zshrc
echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc
```

### Testcontainer Pull Timeout

**Problem:** `Error pulling image postgres:16-alpine`

**Lösung:**
```bash
# Image vorher pullen
docker pull postgres:16-alpine
docker pull redis:7-alpine
docker pull testcontainers/ryuk:0.13.0
```

### Port bereits in Verwendung

**Problem:** `bind: address already in use`

**Lösung:**
Testcontainers wählt automatisch freie Ports. Falls das Problem auftritt:

```bash
# Laufende Container stoppen
docker ps -q | xargs docker stop

# Testcontainers Cleanup
docker container prune -f
```

### Migration dirty=true

**Problem:** Migration ist als "dirty" markiert

**Lösung:**
```bash
# Manuell zurücksetzen
migrate -path backend/migrations \
        -database "postgresql://user:pass@localhost:5432/db?sslmode=disable" \
        force 1

# Danach erneut UP
migrate -path backend/migrations \
        -database "postgresql://user:pass@localhost:5432/db?sslmode=disable" \
        up
```

### Test-Timeout

**Problem:** `test timed out after 10m0s`

**Lösung:**
```bash
# Timeout erhöhen
go test -v -timeout 20m ./internal/database/...

# Oder in Go Test Datei:
// testing.Short() Skip hinzufügen
if testing.Short() {
    t.Skip("Skipping integration test in short mode")
}
```

## Best Practices

### 1. Isolation

Jeder Test startet einen eigenen PostgreSQL Container:
- ✅ Keine Shared State
- ✅ Parallele Ausführung möglich
- ✅ Deterministisch

### 2. Cleanup

Tests nutzen `defer` für automatischen Cleanup:
```go
defer func() {
    if err := pgContainer.Terminate(ctx); err != nil {
        t.Logf("Failed to terminate container: %v", err)
    }
}()
```

### 3. Helper Functions

Wiederverwendbare Helper in `migrations_test.go`:
- `setupPostgresContainer()` - Container starten
- `connectDB()` - Datenbank-Verbindung
- `runMigration()` - Migration ausführen
- `tableExists()` - Tabelle prüfen
- `validateMarketOrdersTable()` - Schema validieren

### 4. Test Data

Verwende realistische Test-Daten:
```go
orders := []MarketOrder{
    {
        OrderID:      123456,
        TypeID:       34,        // Tritanium
        RegionID:     10000002,  // The Forge (Jita)
        LocationID:   60003760,  // Jita 4-4
        Price:        5.50,
        VolumeTotal:  1000,
        VolumeRemain: 500,
    },
}
```

### 5. Performance

Batch Insert Performance Target:
- **100 Orders:** < 100ms ✅ (aktuell: ~35ms)
- **1000 Orders:** < 1s (geplant)

## Migration Schema Referenz

### market_orders

| Spalte | Typ | Constraint | Beschreibung |
|--------|-----|------------|--------------|
| order_id | BIGINT | PRIMARY KEY | ESI Order ID |
| type_id | INTEGER | NOT NULL | EVE Item Type ID |
| region_id | INTEGER | NOT NULL | EVE Region ID |
| location_id | BIGINT | NOT NULL | EVE Location ID |
| is_buy_order | BOOLEAN | NOT NULL | Buy (true) / Sell (false) |
| price | DECIMAL(19,2) | NOT NULL | ISK Preis |
| volume_total | INTEGER | NOT NULL | Ursprüngliches Volumen |
| volume_remain | INTEGER | NOT NULL | Verbleibendes Volumen |
| min_volume | INTEGER | NULL | Mindestvolumen |
| issued | TIMESTAMPTZ | NOT NULL | Order erstellt |
| duration | INTEGER | NOT NULL | Laufzeit in Tagen |
| fetched_at | TIMESTAMPTZ | DEFAULT NOW() | Abrufzeitpunkt |

**Indizes:**
- `PRIMARY KEY (order_id)`
- `UNIQUE (order_id, fetched_at)`
- `idx_market_orders_type_region (type_id, region_id)`
- `idx_market_orders_fetched (fetched_at)`
- `idx_market_orders_location (location_id)`

### price_history

| Spalte | Typ | Constraint | Beschreibung |
|--------|-----|------------|--------------|
| id | SERIAL | PRIMARY KEY | Auto-increment ID |
| type_id | INTEGER | NOT NULL | EVE Item Type ID |
| region_id | INTEGER | NOT NULL | EVE Region ID |
| date | DATE | NOT NULL | Datum |
| highest | DECIMAL(19,2) | NULL | Höchster Preis |
| lowest | DECIMAL(19,2) | NULL | Niedrigster Preis |
| average | DECIMAL(19,2) | NULL | Durchschnittspreis |
| volume | BIGINT | NULL | Handelsvolumen |
| order_count | INTEGER | NULL | Anzahl Orders |

**Indizes:**
- `PRIMARY KEY (id)`
- `UNIQUE (type_id, region_id, date)`
- `idx_price_history_lookup (type_id, region_id, date DESC)`

## Weiterführende Ressourcen

- [Testcontainers Go Documentation](https://golang.testcontainers.org/)
- [golang-migrate GitHub](https://github.com/golang-migrate/migrate)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/16/)
- [EVE ESI Documentation](https://esi.evetech.net/ui/)

## Lizenz

Siehe [LICENSE](../../LICENSE) im Repository Root.
