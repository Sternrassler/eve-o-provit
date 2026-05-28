package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver "pgx" for migration apply

	"github.com/Sternrassler/eve-o-provit/backend/migrations"
)

// ApplyMigrations applies every embedded *.up.sql migration in lexical order
// against the Postgres database. The migrations are idempotent (IF NOT EXISTS),
// so this is safe to run on every startup — it is the single source of truth for
// the schema (there is no separate init-db). Applying on boot means a deploy can
// never drift from the committed migrations.
func ApplyMigrations(ctx context.Context, postgresURL string) error {
	db, err := sql.Open("pgx", postgresURL)
	if err != nil {
		return fmt.Errorf("open db for migrations: %w", err)
	}
	defer db.Close()

	files, err := fs.Glob(migrations.FS, "*.up.sql")
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no embedded *.up.sql migrations found")
	}
	sort.Strings(files)

	for _, name := range files {
		content, err := migrations.FS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}
	return nil
}
