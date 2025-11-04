# Go Testing Skill

**Tech Stack:** Go 1.24 + testing package + Testcontainers  
**Purpose:** Comprehensive testing strategies for backend services  
**Project Context:** 13.6% coverage → Target: 70%+ coverage

---

## Architecture Overview

**Go Testing Pyramid:**

- **Unit Tests (70%):** Fast, isolated, no external dependencies
- **Integration Tests (20%):** Database, Redis, external APIs via Testcontainers
- **Benchmark Tests (10%):** Performance validation

**When to Use Each:**

- **Unit:** Business logic, calculations, data transformations
- **Integration:** Database operations, cache interactions, migration tests
- **Benchmark:** Performance-critical paths (navigation, cargo calculations)

**Critical Workflow:**

- Tests BEFORE implementation (TDD)
- `make docker-rebuild` before integration tests
- Coverage reports guide test priorities

---

## Architecture Patterns

### 1. Table-Driven Tests

**Pattern:** Define test cases as structs, iterate with `t.Run()`.

```go
func TestCalculateAlignTime(t *testing.T) {
    tests := []struct {
        name            string
        mass            float64
        inertiaModifier float64
        want            float64
    }{
        {
            name:            "Interceptor",
            mass:            1200000,
            inertiaModifier: 0.3,
            want:            1.0,
        },
        {
            name:            "Cruiser",
            mass:            12000000,
            inertiaModifier: 0.4,
            want:            13.3,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CalculateAlignTime(tt.mass, tt.inertiaModifier)
            if math.Abs(got-tt.want) > 0.1 {
                t.Errorf("got %v, want %v", got, tt.want)
            }
        })
    }
}
```

**Benefits:**

- Easy to add new test cases (just add struct)
- Clear test intent (name field describes scenario)
- Parallel execution support (`t.Parallel()`)
- Isolated failures (one case fails, others continue)

### 2. Testcontainers Integration

**Pattern:** Spin up real PostgreSQL/Redis in Docker, run tests, cleanup automatically.

```go
func setupPostgresContainer(t *testing.T, ctx context.Context) (*postgres.PostgresContainer, string) {
    t.Helper()

    pgContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("testuser"),
        postgres.WithPassword("testpass"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready").
                WithOccurrence(2).
                WithStartupTimeout(60*time.Second)),
    )
    if err != nil {
        t.Fatalf("Failed to start PostgreSQL: %v", err)
    }

    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    return pgContainer, connStr
}
```

**Benefits:**

- Real database behavior (not mocks)
- Isolated test environment (no shared state)
- Automatic cleanup via `defer`
- Production-like testing

### 3. Test Build Tags

**Pattern:** Separate fast unit tests from slow integration tests.

```go
// cache_integration_test.go
//go:build integration
// +build integration

package services

func TestRedisCache(t *testing.T) { /* ... */ }
```

**Usage:**

```bash
# Run only unit tests (fast)
go test -short ./...

# Run only integration tests
go test -tags=integration ./...

# Run all tests
go test ./...
```

**Benefits:**

- Fast feedback loop (unit tests < 1s)
- CI can run integration tests separately
- Developers choose test scope locally

---

## Best Practices

1. **Test Naming Convention**
   - File: `package_test.go` (same package) or `package_integration_test.go`
   - Function: `TestFunctionName` (unit) or `TestFunctionName_Integration` (integration)
   - Subtests: Descriptive names ("Interceptor", "Empty database", "Invalid input")

2. **Use t.Helper() for Test Utilities**
   - Marks function as helper (correct line numbers in failures)
   - Example: `setupPostgresContainer`, `connectDB`, `validateSchema`

3. **Prefer testify/assert for Readability**
   - `assert.Equal(t, expected, actual)` clearer than `if got != want`
   - `require.NoError(t, err)` stops test immediately on error
   - `assert.NotNil(t, obj)` for existence checks

4. **Test Cleanup with defer**
   - Always cleanup resources (containers, connections, files)
   - Use `defer` immediately after resource creation
   - Example: `defer pool.Close()`, `defer redisClient.Close()`

5. **Coverage-Driven Test Priorities**
   - Start with 0% modules (handlers, services, ESI client)
   - Then increase low coverage (database 31% → 60%+)
   - Maintain high coverage (cargo 91%, navigation 78%)

6. **Mock External Dependencies**
   - ESI API: Mock HTTP responses (httptest package)
   - Time-dependent: Inject time.Now() via interface
   - File system: Use afero for virtual FS

7. **Benchmark Performance-Critical Code**
   - Navigation algorithms (Dijkstra, warp calculations)
   - Cargo fit calculations
   - Market data processing
   - Run: `go test -bench=. -benchmem ./pkg/evedb/navigation/`

---

## Common Patterns

### Pattern 1: Unit Test with Table-Driven Cases

**Scenario:** Test pure function with multiple input/output combinations.

```go
func TestGetItemVolume(t *testing.T) {
    db := setupTestDB(t) // In-memory SQLite
    defer db.Close()

    tests := []struct {
        name     string
        typeID   int
        wantVol  float64
        wantErr  bool
    }{
        {"Tritanium", 34, 0.01, false},
        {"Capsule", 670, 500.0, false},
        {"Invalid", -1, 0, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := GetItemVolume(db, tt.typeID)
            if (err != nil) != tt.wantErr {
                t.Errorf("unexpected error: %v", err)
            }
            if got != tt.wantVol {
                t.Errorf("got %v, want %v", got, tt.wantVol)
            }
        })
    }
}
```

### Pattern 2: Integration Test with Testcontainers

**Scenario:** Test database repository operations with real PostgreSQL.

```go
func TestMarketRepository_UpsertOrders(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    ctx := context.Background()

    // Start PostgreSQL container
    pgContainer, connStr := setupPostgresContainer(t, ctx)
    defer pgContainer.Terminate(ctx)

    // Run migrations
    runMigration(t, connStr, "up")

    // Connect to database
    pool := connectDB(t, ctx, connStr)
    defer pool.Close()

    // Create repository
    repo := database.NewMarketRepository(pool)

    // Test data
    orders := []MarketOrder{
        {OrderID: 123, TypeID: 34, Price: 5.5, /* ... */},
    }

    // Execute test
    err := repo.UpsertMarketOrders(ctx, orders)
    assert.NoError(t, err)

    // Verify
    fetched, err := repo.GetMarketOrders(ctx, 34, 10000002)
    require.NoError(t, err)
    assert.Len(t, fetched, 1)
    assert.Equal(t, 5.5, fetched[0].Price)
}
```

### Pattern 3: Redis Cache Integration Test

**Scenario:** Test caching logic with real Redis instance.

```go
func TestMarketOrderCache_SetAndGet(t *testing.T) {
    redisClient, cleanup := setupRedisContainer(t)
    defer cleanup()

    ctx := context.Background()
    cache := NewMarketOrderCache(redisClient)

    // Test data
    orders := []MarketOrder{{OrderID: 123, Price: 5.5}}
    cacheKey := "market:34:10000002"

    // Set cache
    err := cache.Set(ctx, cacheKey, orders, 10*time.Minute)
    require.NoError(t, err)

    // Get from cache
    cached, err := cache.Get(ctx, cacheKey)
    require.NoError(t, err)
    assert.Len(t, cached, 1)
    assert.Equal(t, 5.5, cached[0].Price)

    // Verify TTL
    ttl, err := redisClient.TTL(ctx, cacheKey).Result()
    require.NoError(t, err)
    assert.Greater(t, ttl, 9*time.Minute)
}
```

### Pattern 4: Migration Testing

**Scenario:** Verify migrations create correct schema (UP) and clean up properly (DOWN).

```go
func TestMigrationUpDown(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()
    pgContainer, connStr := setupPostgresContainer(t, ctx)
    defer pgContainer.Terminate(ctx)

    // Run UP migrations
    runMigration(t, connStr, "up")

    pool := connectDB(t, ctx, connStr)
    defer pool.Close()

    // Verify tables exist
    assert.True(t, tableExists(t, ctx, pool, "market_orders"))

    pool.Close()

    // Run DOWN migrations
    runMigration(t, connStr, "down", "1")

    pool = connectDB(t, ctx, connStr)
    defer pool.Close()

    // Verify tables removed
    assert.False(t, tableExists(t, ctx, pool, "market_orders"))
}
```

### Pattern 5: Benchmark Test

**Scenario:** Measure performance of navigation algorithm.

```go
func BenchmarkShortestPath(b *testing.B) {
    db := setupBenchmarkDB(b)
    defer db.Close()

    nav := NewNavigator(db)
    
    // Pre-load graph
    nav.LoadGraph()

    b.ResetTimer() // Don't count setup time

    for i := 0; i < b.N; i++ {
        _, err := nav.ShortestPath(30000142, 30002187)
        if err != nil {
            b.Fatal(err)
        }
    }
}

// Run: go test -bench=BenchmarkShortestPath -benchmem
// Output: BenchmarkShortestPath-16    5000    245000 ns/op    12000 B/op    150 allocs/op
```

---

## Anti-Patterns

### Skipping Tests in Production Code

**Why:** Tests without assertions are useless.  
**Instead:** Every test MUST have assertions (`assert.*`, `require.*`, or `t.Error*`).

### Testing Implementation Details

**Why:** Brittle tests that break on refactoring.  
**Instead:** Test public API behavior, not internal implementation.

### Shared Mutable State Between Tests

**Why:** Test order dependency, flaky tests.  
**Instead:** Isolate each test (`t.Run()`), use fresh fixtures.

### Not Using t.Helper() for Utilities

**Why:** Wrong line numbers in test failures.  
**Instead:** Mark all helper functions with `t.Helper()`.

### Ignoring Test Cleanup

**Why:** Resource leaks, interference with other tests.  
**Instead:** Always `defer` cleanup immediately after resource creation.

### Over-Mocking Integration Points

**Why:** Miss real integration issues.  
**Instead:** Use Testcontainers for database/cache tests, only mock external APIs.

---

## Integration with Development Workflow

### With Docker Services

**Workflow:**

```
make docker-up → make test-be-int → Testcontainers start fresh instances → Tests run → Cleanup
```

**Note:** Integration tests start their OWN containers (don't rely on `docker-up` services).

### With Code Changes

**Workflow:**

```
Write Test (Red) → Implement (Green) → make docker-rebuild → make test-be → Refactor
```

**Critical:** Always rebuild Docker after code changes before integration tests.

### With Coverage Reports

**Workflow:**

```
make test-be → go tool cover -func=coverage.out → Identify gaps → Write tests for 0% modules
```

**Priority:** Handlers (1.8%) → Services (0%) → ESI (0%) → Database (31% → 60%)

### With CI/CD

**Workflow:**

```
make pr-check → Runs: lint + test-be + scan → All green → Merge allowed
```

**CI runs:** Unit tests (fast) + Integration tests (Testcontainers in CI) + Benchmarks (optional).

---

## Performance Considerations

1. **Test Speed**
   - Unit tests: < 100ms per test
   - Integration tests: < 5s per test (Testcontainers startup)
   - Use `-short` flag for fast local feedback

2. **Testcontainers Optimization**
   - Reuse containers across tests in same package (if stateless)
   - Use alpine images (postgres:16-alpine, redis:7-alpine)
   - Parallel execution: `t.Parallel()` for isolated tests

3. **Benchmark Accuracy**
   - Use `b.ResetTimer()` after setup
   - Run multiple iterations (`-benchtime=10s` or `-benchtime=100x`)
   - Profile with `-cpuprofile` / `-memprofile` for bottlenecks

---

## Security Guidelines

1. **Test Data**
   - Never use production credentials in tests
   - Generate random test data (avoid hardcoded IDs that might leak)
   - Clean up test databases after runs

2. **Secret Handling**
   - Use environment variables for test configuration
   - Don't commit test credentials to repository
   - Use `.env.test` (gitignored) for local test config

3. **Integration Test Isolation**
   - Each test gets fresh container (no shared state)
   - Use unique database names per test run
   - Verify cleanup actually happens (defer checks)

---

## Quick Reference

| Operation | Command | Use Case |
|-----------|---------|----------|
| Run all tests | `make test-be` | Complete backend testing |
| Run unit tests only | `make test-be-unit` | Fast feedback (short mode) |
| Run integration tests | `make test-be-int` | Database/Redis tests |
| Run benchmarks | `make test-be-bench` | Performance validation |
| Check coverage | `go tool cover -func=coverage.out` | Identify gaps |
| Coverage HTML | `go tool cover -html=coverage.out` | Visual coverage report |
| Run specific test | `go test -run TestName ./pkg/...` | Focused testing |
| Verbose output | `go test -v ./...` | See all test output |
| Race detection | `go test -race ./...` | Find concurrency bugs |

---

## Common Debugging Scenarios

### Scenario: Test Fails Only in CI

**Steps:**

1. Check if test has external dependencies (network, time)
2. Verify test cleanup (defer statements)
3. Add `-race` flag to detect concurrency issues
4. Check Testcontainers timeout (CI slower than local)

### Scenario: Flaky Integration Test

**Steps:**

1. Check for shared state between tests
2. Verify container startup (wait strategy correct?)
3. Add explicit waits for async operations
4. Use `t.Parallel()` to detect race conditions

### Scenario: Low Coverage Despite Tests

**Steps:**

1. Run `go tool cover -html=coverage.out` to visualize
2. Check if tests skip error paths
3. Add table-driven cases for edge cases
4. Verify test actually calls the function (not just setup)

### Scenario: Testcontainers Timeout

**Steps:**

1. Increase `WithStartupTimeout(90*time.Second)`
2. Check Docker daemon is running
3. Verify wait strategy matches container logs
4. Use `wait.ForLog()` with correct log message

---

## Makefile Integration

All test operations available via Makefile:

```makefile
# From project Makefile:
test-be:       # All backend tests (unit + integration)
test-be-unit:  # Fast unit tests only (-short flag)
test-be-int:   # Integration tests only (-tags=integration)
test-be-bench: # Performance benchmarks
test-load:     # Load tests (requires Redis + SDE)
```

**Usage:**

```bash
make test-be                    # Complete test suite
make test-be-unit              # Fast feedback (< 1s)
make test-be-int               # Full integration (~ 30s)
make test-be-bench             # Performance baseline
go tool cover -func=backend/coverage.out  # Coverage report
```

---

## Coverage Goals & Strategy

**Current State:** 13.6% overall coverage

**Critical Gaps (Priority Order):**

1. **internal/handlers (1.8%)** → Target: 70%
2. **internal/services (0%)** → Target: 75%
3. **pkg/esi (0%)** → Target: 80%
4. **pkg/evesso (0%)** → Target: 85% (security-critical)
5. **internal/database (31%)** → Target: 60%

**Maintain High Coverage:**

- pkg/evedb/cargo (91.4%) → Keep > 90%
- pkg/evedb/navigation (77.9%) → Keep > 75%

**Test Pyramid Target:**

- Unit Tests: 70% of all tests
- Integration Tests: 20% of all tests
- Benchmarks: 10% of all tests

---

**Last Updated:** 2025-11-04  
**Maintained By:** skill-creator agent  
**Critical Integration:** Use with Docker skill (`make docker-rebuild` before integration tests!)
