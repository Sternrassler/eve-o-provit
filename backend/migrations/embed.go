// Package migrations embeds the SQL migration files so the backend can apply
// them at startup. These files are the single source of truth for the Postgres
// schema (no separate init-db). All up-migrations are idempotent (IF NOT EXISTS),
// so applying them on every boot is safe.
package migrations

import "embed"

// FS holds the embedded *.up.sql migration files.
//
//go:embed *.up.sql
var FS embed.FS
