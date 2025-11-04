# Database Skill: SQLite

**Tech Stack:** SQLite 3 (mattn/go-sqlite3)

**Project:** eve-o-provit Static Data Export (SDE) Read-Only Access

---

## Architecture Patterns

### Read-Only Mode
- **Immutable Access:** Database opened with `mode=ro&immutable=1` flags
- **Static Reference Data:** EVE SDE (items, regions, systems, stations)
- **No Writes:** Application never modifies SDE database

### Dual-Database Architecture
- **SQLite:** Static EVE game data (SDE)
- **PostgreSQL:** Dynamic market data, user preferences
- **Separation of Concerns:** Static vs. dynamic data in separate databases

---

## Best Practices

1. **Read-Only URI:** Always use `file:path?mode=ro&immutable=1` for SDE access
2. **Connection Per Application:** Single shared connection (not per request)
3. **No Transactions:** Read-only mode doesn't need transactions
4. **Fast Lookups:** Use `PRAGMA query_only = ON` for safety
5. **Index Optimization:** Ensure SDE has indexes on frequently queried columns
6. **Error Handling:** Handle "database is locked" gracefully (rare in read-only)
7. **Version Control:** SDE database versioned, updated via download script

---

## Common Patterns

### 1. Read-Only Connection

```go
import _ "github.com/mattn/go-sqlite3"

sdeURI := fmt.Sprintf("file:%s?mode=ro&immutable=1", sdePath)
sdeDB, err := sql.Open("sqlite3", sdeURI)
if err != nil {
    return fmt.Errorf("failed to open SDE: %w", err)
}

// Test connection
if err := sdeDB.Ping(); err != nil {
    return fmt.Errorf("failed to ping SDE: %w", err)
}
```

### 2. Repository Pattern for SDE

```go
type SDERepository struct {
    db *sql.DB
}

func NewSDERepository(db *sql.DB) *SDERepository {
    return &SDERepository{db: db}
}

func (r *SDERepository) GetTypeInfo(ctx context.Context, typeID int) (*TypeInfo, error) {
    var info TypeInfo
    err := r.db.QueryRowContext(ctx, 
        "SELECT type_id, type_name, group_id FROM invTypes WHERE type_id = ?", 
        typeID).Scan(&info.TypeID, &info.Name, &info.GroupID)
    
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("type %d not found", typeID)
    }
    return &info, err
}
```

### 3. List Query with Limit

```go
func (r *SDERepository) SearchItems(ctx context.Context, query string, limit int) ([]Item, error) {
    rows, err := r.db.QueryContext(ctx, 
        "SELECT type_id, type_name FROM invTypes WHERE type_name LIKE ? LIMIT ?",
        "%"+query+"%", limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var items []Item
    for rows.Next() {
        var item Item
        rows.Scan(&item.TypeID, &item.Name)
        items = append(items, item)
    }
    return items, rows.Err()
}
```

---

## Anti-Patterns

❌ **Write Attempts:** Never `INSERT`, `UPDATE`, or `DELETE` on SDE (read-only mode enforces this)

❌ **Connection Per Request:** Create one connection at startup, reuse across requests

❌ **Missing Indexes:** Ensure SDE has indexes on `type_id`, `group_id`, `region_id`

❌ **Unbounded Queries:** Always use `LIMIT` for list queries (prevent large result sets)

❌ **Ignoring Context:** Use `QueryContext()`, not `Query()` (for cancellation support)

---

## Integration with Backend

### Dual-Database Manager
- **DB Struct:** Holds both `*pgxpool.Pool` (PostgreSQL) and `*sql.DB` (SQLite)
- **Repository Injection:** SDERepository receives SQLite connection, MarketRepository receives PostgreSQL pool
- **Service Coordination:** Services can query both databases (SDE for names, PostgreSQL for prices)

### Typical Flow
1. User requests market data for "Tritanium" (item name)
2. Service queries SDE for `type_id` by name
3. Service queries PostgreSQL for market orders by `type_id`
4. Response includes item name (SDE) + prices (PostgreSQL)

---

## Performance Considerations

- **No Connection Pooling:** SQLite read-only uses single connection (safe with WAL mode)
- **Query Optimization:** Use `EXPLAIN QUERY PLAN` to verify index usage
- **Memory Mode:** Consider loading small tables into memory (`PRAGMA cache_size`)
- **Prepared Statements:** Reuse prepared statements for frequently executed queries

---

## Security Guidelines

- **File Permissions:** SDE file should be world-readable, not writable by application user
- **Path Validation:** Sanitize SDE path from environment variable (prevent path traversal)
- **Read-Only Enforcement:** Use `mode=ro&immutable=1` flags (prevents accidental writes)

---

## Quick Reference

| Task | Pattern |
|------|---------|
| Open read-only | `sql.Open("sqlite3", "file:path?mode=ro&immutable=1")` |
| Single row | `db.QueryRowContext(ctx, "SELECT ...", args).Scan(&var)` |
| Multiple rows | `rows, _ := db.QueryContext(ctx, "SELECT ...", args)` |
| LIKE search | `WHERE name LIKE ? LIMIT 100` (always use LIMIT) |
| No rows error | `err == sql.ErrNoRows` |
| Close | `db.Close()` (on application shutdown) |
