# Testing Guide

## Environment Variables

### SDE_DB_PATH

Path to the EVE SDE SQLite database for tests.

**Default values:**

- `pkg/evedb/navigation/`: `../../../data/sde/eve-sde.db`
- `internal/services/`: `../../data/sde/eve-sde.db`

**Usage:**

```bash
# Local testing (default)
go test ./pkg/evedb/navigation -v

# Container testing (override)
export SDE_DB_PATH=/app/data/sde/eve-sde.db
go test ./pkg/evedb/navigation -v

# CI/CD (absolute path)
SDE_DB_PATH=/workspace/backend/data/sde/eve-sde.db go test ./...
```

**Docker Compose:**

```yaml
services:
  backend:
    environment:
      - SDE_DB_PATH=/app/data/sde/eve-sde.db
```

## Running Tests

### Unit Tests (no database required)

```bash
go test ./pkg/evedb/dogma -v
```

### Integration Tests (require SDE database)

```bash
# Local (default paths)
go test ./pkg/evedb/navigation -v
go test ./internal/services -v -run "Integration"

# Container (custom path)
export SDE_DB_PATH=/app/data/sde/eve-sde.db
go test ./pkg/evedb/navigation -v
```

### All Tests

```bash
go test ./... -v
```

## Test Database Setup

### Local Development

1. Download SDE database: `scripts/download-sde.sh`
2. Place at: `backend/data/sde/eve-sde.db`
3. Run tests: `go test ./...`

### Container

1. Mount volume: `-v ./data/sde:/app/data/sde`
2. Set ENV: `SDE_DB_PATH=/app/data/sde/eve-sde.db`
3. Run tests inside container

### CI/CD

```yaml
env:
  SDE_DB_PATH: ${{ github.workspace }}/backend/data/sde/eve-sde.db

steps:
  - name: Download SDE
    run: ./scripts/download-sde.sh
  
  - name: Run Tests
    run: go test ./...
```
