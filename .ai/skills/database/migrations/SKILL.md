# Database Migrations Skill

**Tech Stack:** golang-migrate/migrate v4 + PostgreSQL  
**Purpose:** Version-controlled database schema changes with rollback support  
**Migration Tool:** golang-migrate (CLI + Library)

---

## Architecture Overview

**golang-migrate** provides systematic database schema evolution:

- **Sequential Versioning:** Migrations numbered sequentially (001, 002, 003...)
- **Up/Down Pairs:** Each migration has UP (apply) and DOWN (rollback) SQL files
- **Schema Tracking:** `schema_migrations` table tracks applied versions
- **CLI + Library:** Can be run manually (CLI) or in tests (library)

**When to Use:**

- Adding/modifying database tables
- Creating/dropping indexes
- Altering column types
- Adding constraints
- Data migrations (carefully!)

**Migration Directory:** `backend/migrations/`

---

## Architecture Patterns

### 1. Sequential Numbered Migrations

**Pattern:** Each migration is a pair of files: `{version}_{name}.up.sql` and `{version}_{name}.down.sql`

```txt
migrations/
├── 000001_initial_schema.up.sql
├── 000001_initial_schema.down.sql
├── 000002_add_users_table.up.sql
├── 000002_add_users_table.down.sql
├── 000003_add_user_email_index.up.sql
└── 000003_add_user_email_index.down.sql
```

**Benefits:**

- Deterministic execution order
- Clear migration history
- Easy to identify current version
- Automatic version tracking

### 2. Idempotent Migrations

**Pattern:** Migrations can be run multiple times without breaking.

```sql
-- Good: Idempotent
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE
);

-- Bad: Not idempotent (fails on second run)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE
);
```

**Benefits:**

- Safe to re-run in case of partial failures
- Supports development workflow (reset → re-migrate)
- Reduces migration errors

### 3. Safe Rollback Strategy

**Pattern:** Every UP migration has corresponding DOWN that reverses changes.

```sql
-- 000002_add_users_table.up.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);

-- 000002_add_users_table.down.sql
DROP INDEX IF EXISTS idx_users_email;
DROP TABLE IF EXISTS users;
```

**Benefits:**

- Can undo changes if issues arise
- Supports feature branch testing (apply → test → rollback)
- Clean slate for re-testing

---

## Best Practices (Normative Requirements)

1. **Never Modify Applied Migrations (MUST NOT)**
   - Once merged to main, migrations are immutable
   - Create new migration to fix issues
   - Reason: Other developers/environments already applied

2. **One Logical Change Per Migration (SHOULD)**
   - Single table creation
   - Single index addition
   - Single column modification
   - Easier to rollback granularly

3. **Use Transactions Where Supported (MUST)**
   - PostgreSQL supports DDL transactions
   - Wrap migration in `BEGIN; ... COMMIT;`
   - All-or-nothing application

4. **Test Migrations Locally First (MUST)**
   - Run `make migrate-up` on local database
   - Verify schema changes
   - Test rollback: `make migrate-down`
   - Ensure no data loss

5. **Data Migrations Require Extra Care (MUST)**
   - Separate from schema migrations
   - Test with production-like data volumes
   - Consider backfilling in application code instead
   - Always have rollback plan for data

6. **Use Testcontainers for Integration Tests (SHOULD)**
   - Spin up temporary PostgreSQL instance
   - Apply migrations programmatically
   - Test against real database
   - Cleanup automatic after tests

7. **Version Control Schema State (MUST)**
   - Commit migration files to git
   - One PR per migration (or related set)
   - Review migrations like code
   - Test in CI before merging

---

## Common Patterns

### Pattern 1: Create New Migration

**Scenario:** Need to add `profiles` table.

```bash
# Create migration files
make migrate-create NAME=add_profiles_table

# Created files:
# migrations/000004_add_profiles_table.up.sql
# migrations/000004_add_profiles_table.down.sql

# Edit UP migration:
cat > backend/migrations/000004_add_profiles_table.up.sql << 'EOF'
BEGIN;

CREATE TABLE IF NOT EXISTS profiles (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(100),
    bio TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_profiles_user_id ON profiles(user_id);

COMMIT;
EOF

# Edit DOWN migration:
cat > backend/migrations/000004_add_profiles_table.down.sql << 'EOF'
BEGIN;

DROP INDEX IF EXISTS idx_profiles_user_id;
DROP TABLE IF EXISTS profiles;

COMMIT;
EOF

# Apply migration
make migrate-up

# Verify
make docker-shell-db
\dt profiles
\d profiles
```

### Pattern 2: Add Column to Existing Table

**Scenario:** Add `last_login` timestamp to users.

```sql
-- UP migration
BEGIN;

ALTER TABLE users 
ADD COLUMN IF NOT EXISTS last_login TIMESTAMPTZ;

-- Optional: Create index if querying on this column
CREATE INDEX IF NOT EXISTS idx_users_last_login ON users(last_login);

COMMIT;

-- DOWN migration
BEGIN;

DROP INDEX IF EXISTS idx_users_last_login;
ALTER TABLE users DROP COLUMN IF EXISTS last_login;

COMMIT;
```

**Key Points:**

- Use `IF NOT EXISTS` / `IF EXISTS` for idempotency
- Consider index if column will be queried
- Rollback removes both index and column

### Pattern 3: Modify Column Type (Safe Pattern)

**Scenario:** Change `email` from VARCHAR(255) to VARCHAR(320).

```sql
-- UP migration
BEGIN;

-- Safe: Widening a VARCHAR is non-destructive
ALTER TABLE users 
ALTER COLUMN email TYPE VARCHAR(320);

COMMIT;

-- DOWN migration
BEGIN;

-- Rollback to original size
ALTER TABLE users 
ALTER COLUMN email TYPE VARCHAR(255);

COMMIT;
```

**Warning:** Narrowing types can cause data loss! Test thoroughly.

### Pattern 4: Data Migration with Backfill

**Scenario:** Add `username` column, populate from email.

```sql
-- UP migration
BEGIN;

-- 1. Add column (nullable initially)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS username VARCHAR(50);

-- 2. Backfill existing data
UPDATE users 
SET username = SPLIT_PART(email, '@', 1)
WHERE username IS NULL;

-- 3. Make column NOT NULL after backfill
ALTER TABLE users 
ALTER COLUMN username SET NOT NULL;

-- 4. Add unique constraint
ALTER TABLE users 
ADD CONSTRAINT users_username_unique UNIQUE (username);

COMMIT;

-- DOWN migration
BEGIN;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_unique;
ALTER TABLE users DROP COLUMN IF EXISTS username;

COMMIT;
```

**Key Points:**

- Add column nullable first
- Backfill data
- Then add constraints
- Prevents NOT NULL violation on existing rows

### Pattern 5: Testing Migrations with Testcontainers

**Scenario:** Integration test that verifies migrations work correctly.

```go
// backend/internal/database/migrations_test.go
package database_test

import (
    "context"
    "database/sql"
    "testing"
    "time"

    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
    _ "github.com/lib/pq"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

func TestMigrations(t *testing.T) {
    ctx := context.Background()

    // 1. Start PostgreSQL container
    req := testcontainers.ContainerRequest{
        Image:        "postgres:16-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_USER":     "test",
            "POSTGRES_PASSWORD": "test",
            "POSTGRES_DB":       "testdb",
        },
        WaitingFor: wait.ForLog("database system is ready to accept connections").
            WithStartupTimeout(60 * time.Second),
    }

    postgresC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        t.Fatalf("Failed to start container: %v", err)
    }
    defer postgresC.Terminate(ctx)

    // 2. Get connection details
    host, _ := postgresC.Host(ctx)
    port, _ := postgresC.MappedPort(ctx, "5432")
    dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

    // 3. Connect to database
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        t.Fatalf("Failed to connect: %v", err)
    }
    defer db.Close()

    // 4. Run migrations
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    if err != nil {
        t.Fatalf("Failed to create driver: %v", err)
    }

    m, err := migrate.NewWithDatabaseInstance(
        "file://../migrations",
        "postgres",
        driver,
    )
    if err != nil {
        t.Fatalf("Failed to create migrate instance: %v", err)
    }

    // 5. Apply all migrations
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        t.Fatalf("Migration failed: %v", err)
    }

    // 6. Verify schema
    var tableExists bool
    err = db.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'users'
        )
    `).Scan(&tableExists)
    
    if err != nil {
        t.Fatalf("Failed to check table: %v", err)
    }
    
    if !tableExists {
        t.Error("Expected users table to exist after migration")
    }

    // 7. Test rollback
    if err := m.Down(); err != nil && err != migrate.ErrNoChange {
        t.Fatalf("Rollback failed: %v", err)
    }

    // 8. Verify table removed
    err = db.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'users'
        )
    `).Scan(&tableExists)
    
    if tableExists {
        t.Error("Expected users table to be removed after rollback")
    }
}
```

**Run test:**

```bash
make test-migrations
```

### Pattern 6: Migration Workflow in Development

**Complete workflow from creation to production:**

```bash
# 1. Create migration
make migrate-create NAME=add_sessions_table

# 2. Edit migration files
vim backend/migrations/000005_add_sessions_table.up.sql
vim backend/migrations/000005_add_sessions_table.down.sql

# 3. Ensure Docker services running
make docker-up

# 4. Apply migration locally
make migrate-up

# 5. Verify in database
make docker-shell-db
\dt sessions
\d sessions

# 6. Test application with new schema
make docker-rebuild
make test-be

# 7. Test rollback
make migrate-down
make docker-shell-db
\dt sessions  # Should not exist

# 8. Re-apply
make migrate-up

# 9. Commit migration files
git add backend/migrations/
git commit -m "feat(db): add sessions table for user authentication"

# 10. Push and create PR
git push origin feat/add-sessions-table
gh pr create --title "feat(db): add sessions table" --body "..."
```

---

## Anti-Patterns

### Modifying Applied Migrations

**Why:** Other environments already applied old version, causes inconsistency.  
**Instead:** Create new migration to fix/modify.

### Skipping DOWN Migrations

**Why:** Can't rollback if issues arise.  
**Instead:** Always write reversible DOWN migration.

### Data Loss in Rollback

**Why:** DOWN migration drops table with data.  
**Instead:** Backup data or make rollback non-destructive where possible.

### Non-Idempotent Migrations

**Why:** Re-running migration fails.  
**Instead:** Use `IF EXISTS` / `IF NOT EXISTS` clauses.

### Mixing Schema and Data Migrations

**Why:** Hard to rollback, complex dependencies.  
**Instead:** Separate migrations or use application-level backfill.

### Not Testing Migrations Locally

**Why:** Production failures, schema corruption.  
**Instead:** Test locally with `make migrate-up/down` before committing.

---

## Integration with Development Workflow

### With Docker Services

**Workflow:**

```text
make docker-up → make migrate-up → make docker-rebuild → make test-be
```

**Critical:** Docker PostgreSQL must be running before migrations.

### With Code Changes

**Workflow:**

```text
Create Migration → Apply Migration → Update Models/Queries → make docker-rebuild → Test
```

**Rebuild ensures:** Code changes + schema changes both reflected.

### With Testing

**Workflow:**

```text
make test-migrations → Verify schema → make test-be-int → Full integration
```

**Testcontainers:** Provides isolated PostgreSQL for migration tests.

---

## Performance Considerations

1. **Index Creation**
   - Create indexes CONCURRENTLY to avoid table locks
   - Example: `CREATE INDEX CONCURRENTLY idx_name ON table(column);`
   - Longer creation time, but no downtime

2. **Large Table Alterations**
   - ALTER TABLE can lock table during modification
   - Test on production-size data first
   - Consider zero-downtime strategies (new table + swap)

3. **Data Migrations**
   - Batch large updates (LIMIT + loop)
   - Avoid full table scans
   - Monitor query performance

---

## Security Guidelines

1. **Migration Permissions**
   - Use dedicated migration user (not application user)
   - Grant only necessary DDL permissions
   - Restrict in production

2. **Sensitive Data**
   - Don't hardcode sensitive data in migrations
   - Use application seeding for initial data
   - Never commit production data

3. **Rollback Safety**
   - Test rollback in staging first
   - Have database backups before production migrations
   - Document rollback procedure

---

## Quick Reference

| Operation | Make Target | Use Case |
|-----------|-------------|----------|
| Create migration | `make migrate-create NAME=xxx` | New schema change |
| Apply migrations | `make migrate-up` | Apply pending |
| Rollback one | `make migrate-down` | Undo last |
| Check version | `SELECT * FROM schema_migrations;` | Current state |
| Test migrations | `make test-migrations` | Testcontainers test |
| Force version | `migrate -database $URL force 5` | Fix dirty state |

---

## Common Debugging Scenarios

### Scenario: Dirty Migration State

**Steps:**

1. Check schema_migrations: `SELECT * FROM schema_migrations;`
2. If `dirty = true`: Migration partially applied
3. Manually fix database to match expected state
4. Force version: `migrate -database $DATABASE_URL force <version>`

### Scenario: Migration Fails Mid-Apply

**Steps:**

1. Check error in logs
2. Verify transaction support (PostgreSQL has it)
3. If transaction failed: State rolled back automatically
4. Fix migration SQL
5. Re-run `make migrate-up`

### Scenario: Rollback Not Working

**Steps:**

1. Check DOWN migration syntax
2. Verify table/index names match
3. Test locally: `make migrate-down`
4. If data loss concern: Backup first

---

## Makefile Integration

```makefile
# From project Makefile:
migrate-up:     # Apply all pending migrations
migrate-down:   # Rollback last migration
migrate-create: # Create new migration (NAME=xxx required)
migrate:        # Alias for migrate-up
test-migrations: # Run Testcontainers integration tests
```

**Usage:**

```bash
make migrate-create NAME=add_api_keys_table
make migrate-up
make migrate-down
make test-migrations
```

---

**Last Updated:** 2025-11-04  
**Maintained By:** skill-creator agent  
**Critical Integration:** Always run `make docker-rebuild` after schema changes + code changes!
