# Database Skill: PostgreSQL

**Tech Stack:** PostgreSQL (via pgx/v5 driver)

**Project:** eve-o-provit Dynamic Market Data Storage

---

## Architecture Patterns

### Connection Pooling

- **pgxpool.Pool:** Production-ready connection pool with automatic management
- **Configuration:** Max connections, idle timeout, health checks
- **Context-aware:** All operations support context cancellation

### Repository Pattern

- **Encapsulation:** Database logic isolated in repository structs
- **Domain Models:** Repositories return business domain types, not raw SQL types
- **Transaction Management:** Explicit transaction boundaries in repositories

---

## Best Practices (Normative Requirements)

1. **Use Connection Pools (MUST):** `pgxpool.Pool` instead of single `pgx.Conn`
2. **Context Timeouts (MUST):** Every query has context with timeout
3. **Prepared Statements (MUST):** Use `$1, $2` placeholders (SQL injection prevention)
4. **Transaction Boundaries (SHOULD):** Short-lived transactions, commit or rollback explicitly
5. **Batch Operations (SHOULD):** Use `pgx.Batch` for bulk inserts/updates (10x faster)
6. **NULL Handling (MUST):** Use pointers for nullable columns (`*int`, `*float64`)
7. **Migration Strategy (MUST):** Version-controlled SQL migrations, never alter production directly

---

## Common Patterns

### 1. Connection Pool Setup

```go
ctx := context.Background()
pool, err := pgxpool.New(ctx, "postgres://user:pass@localhost:5432/db")
if err != nil {
    return fmt.Errorf("failed to create pool: %w", err)
}
defer pool.Close()

// Test connection
if err := pool.Ping(ctx); err != nil {
    return fmt.Errorf("failed to ping: %w", err)
}
```

### 2. Repository Pattern

```go
type MarketRepository struct {
    db *pgxpool.Pool
}

func NewMarketRepository(db *pgxpool.Pool) *MarketRepository {
    return &MarketRepository{db: db}
}

func (r *MarketRepository) GetOrders(ctx context.Context, regionID int) ([]Order, error) {
    rows, err := r.db.Query(ctx, "SELECT * FROM orders WHERE region_id = $1", regionID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    return pgx.CollectRows(rows, pgx.RowToStructByName[Order])
}
```

### 3. Transaction Pattern

```go
func (r *MarketRepository) UpsertOrders(ctx context.Context, orders []Order) error {
    tx, err := r.db.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback(ctx) // Safe if already committed
    
    for _, order := range orders {
        _, err := tx.Exec(ctx, `INSERT INTO orders (...) VALUES ($1, $2) ON CONFLICT DO UPDATE ...`, order.ID, order.Price)
        if err != nil {
            return err // Rollback via defer
        }
    }
    
    return tx.Commit(ctx)
}
```

### 4. Batch Insert (High Performance)

```go
func (r *MarketRepository) BulkInsert(ctx context.Context, orders []Order) error {
    batch := &pgx.Batch{}
    
    for _, order := range orders {
        batch.Queue("INSERT INTO orders (...) VALUES ($1, $2, $3)", order.ID, order.Price, order.Volume)
    }
    
    results := r.db.SendBatch(ctx, batch)
    defer results.Close()
    
    for i := 0; i < batch.Len(); i++ {
        if _, err := results.Exec(); err != nil {
            return fmt.Errorf("batch insert failed at %d: %w", i, err)
        }
    }
    
    return nil
}
```

---

## Anti-Patterns

❌ **String Concatenation for Queries:** Always use parameterized queries (`$1, $2`)

❌ **Long-Running Transactions:** Hold locks minimal time, commit quickly

❌ **Missing Context:** Every query needs context for cancellation support

❌ **Ignoring NULL Values:** Use pointers (`*int`) for nullable columns

❌ **One Connection Per Request:** Use connection pool, not individual connections

---

## Integration with Backend

### Dependency Injection

- Pass `*pgxpool.Pool` to repository constructors
- Repositories are fields in service structs
- Services orchestrate multiple repositories

### Error Handling

- Check `pgx.ErrNoRows` for not-found scenarios
- Log full error details, return generic errors to API clients
- Use `errors.Is()` and `errors.As()` for error checking

---

## Performance Considerations

- **Connection Pool Size:** Set based on CPU cores (recommended: `num_cpus * 2`)
- **Query Timeouts:** Use context with timeout (3-5 seconds for most queries)
- **Indexes:** Create indexes on frequently queried columns (`region_id`, `type_id`, `fetched_at`)
- **EXPLAIN ANALYZE:** Profile slow queries, optimize joins
- **Batch Operations:** Use `pgx.Batch` for >100 inserts (10-50x faster)

---

## Security Guidelines

- **Least Privilege:** Application user has only needed permissions (SELECT, INSERT, UPDATE)
- **SSL/TLS:** Use `sslmode=require` in connection string for production
- **Connection String:** Store in environment variable, never hardcode
- **Prepared Statements:** pgx uses them automatically with `$1` placeholders

---

## Quick Reference

| Task | Pattern |
|------|---------|
| Create pool | `pgxpool.New(ctx, "postgres://...")` |
| Single row | `db.QueryRow(ctx, "SELECT ...", args).Scan(&var)` |
| Multiple rows | `rows, _ := db.Query(ctx, "SELECT ...", args)` |
| Insert/Update | `db.Exec(ctx, "INSERT ...", args)` |
| Transaction | `tx, _ := db.Begin(ctx); defer tx.Rollback(ctx); tx.Commit(ctx)` |
| Batch insert | `batch := &pgx.Batch{}; batch.Queue(...); db.SendBatch(ctx, batch)` |
| No rows error | `errors.Is(err, pgx.ErrNoRows)` |
