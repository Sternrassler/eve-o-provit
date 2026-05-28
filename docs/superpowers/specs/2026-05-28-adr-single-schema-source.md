# ADR: Single schema source — apply migrations at startup, drop init-db

- **Datum:** 2026-05-28
- **Status:** Accepted (implemented in v0.9.4)

## Context

The Postgres schema had **two hand-maintained sources that drifted unnoticed**:

- `backend/migrations/*.up.sql` — golang-migrate-style files, used by integration tests (`testcontainer.go` applies them) and `make migrate-up`.
- `deployments/init-db/01-init.sql` — the file actually used in production (applied manually via `psql` per the deploy runbook).

They diverged twice:
1. The market-history table was named `price_history` in the migrations/code but `market_history` in init-db. The backend queries `price_history`, which **never existed in prod** → daily volume + competition baseline were silently 0.
2. The competition tables (#43) only existed in the migrations until they were hand-mirrored into init-db.

Tests passed (they use `migrations/`), prod was broken — the classic dual-source drift. Additional smell: `FetchAndStoreMarketHistory` had a unit test but no caller, so "history never populated" was invisible.

## Decision

`backend/migrations/*.up.sql` is the **single source of truth** for the Postgres schema.

- The backend **applies the embedded migrations at startup** (`database.ApplyMigrations`, invoked from `db.New` after the Postgres connection). Migrations are embedded via `//go:embed` (`backend/migrations/embed.go`).
- **All up-migrations are idempotent** (`CREATE … IF NOT EXISTS`). Applying them on every boot is therefore safe and needs no version table — re-runs are no-ops on an already-migrated database; the `pgdata` volume preserves data.
- `deployments/init-db/` is **removed**. The local `docker-compose.yml` no longer mounts it; the prod runbook no longer has a manual `psql` schema step.

### Why not golang-migrate version tracking?
The existing prod DB was bootstrapped via init-db, so it has no `schema_migrations` table — golang-migrate would need a one-time `force`/baseline. Idempotent boot-apply avoids that entirely and also auto-applies future migrations on deploy (closing the earlier "migrations weren't auto-applied" gap).

## Consequences

- A deploy can no longer drift from the committed schema; new migrations ship + apply automatically.
- Integration tests already run against `migrations/` (now the same source as prod), so a missing/renamed table fails CI.
- New migrations **must be idempotent** (`IF NOT EXISTS`, `ON CONFLICT`, etc.) since they re-run on every boot.
- Legacy init-db-only tables (`users`, `watchlists`, `watchlist_items`, `profit_calculations`) were unused by code and are intentionally not carried into migrations.

## Guardrails (process)

- **Wire-or-delete:** a service method with no non-test caller is either wired up or removed (Go's `unused` linter does not catch exported methods — enforce in review).
- **Feature smoke beyond `/health`:** post-deploy, check a representative endpoint returns sane non-empty data, not just that the process is up.
- See brain memory [[eve-o-provit-schema-source-drift]] for the cross-cutting lesson.
