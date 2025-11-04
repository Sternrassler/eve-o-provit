# Backend Commands

This directory contains standalone command-line tools and services.

## Available Commands

### Production Services

- **`api/`** - Main HTTP API server (Fiber framework)
  - Port: 9001 (configurable via `PORT` env var)
  - Routes: `/api/v1/...`

### Development Tools

- **`test-sde/`** - Database connectivity test
  - Tests basic SDE database access
  - Validates database schema

## Example Programs

For interactive CLI tools and usage examples, see:

- **`../examples/cargo/`** - Cargo capacity calculator with skill modifiers
- **`../examples/navigation/`** - Route planning and travel time calculator

Run examples with:

```bash
go run ../examples/cargo --help
go run ../examples/navigation --help
```

## Building

```bash
# Build API server
go build -o bin/api ./cmd/api

# Build test tool
go build -o bin/test-sde ./cmd/test-sde
```

## Running

```bash
# Run API server
./bin/api

# Or directly
go run ./cmd/api
```
