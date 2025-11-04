# Load & Performance Testing Skill

**Purpose:** Systematic performance testing, benchmarking, and load simulation to ensure application scalability and identify bottlenecks.

---

## Architecture Overview

### Test Types

1. **Benchmark Tests**: Measure function/operation performance
2. **Load Tests**: Simulate concurrent user load
3. **Stress Tests**: Find breaking points under extreme load
4. **Spike Tests**: Sudden traffic surges
5. **Soak Tests**: Long-duration stability testing

### When to Use Each

- **Benchmark**: Optimize algorithms, detect regressions
- **Load**: Validate capacity planning, find bottlenecks
- **Stress**: Determine limits, test failure modes
- **Spike**: Test auto-scaling, cache behavior
- **Soak**: Find memory leaks, resource exhaustion

---

## Architecture Patterns

### 1. Go Benchmark Pattern

**Principle:** Use `testing.B` for micro-benchmarks

```go
func BenchmarkCalculateRoute(b *testing.B) {
    graph := setupTestGraph() // Setup outside timer
    start := 30000001
    end := 30000142
    
    b.ResetTimer() // Start timing here
    
    for i := 0; i < b.N; i++ {
        ShortestPath(graph, start, end)
    }
}

// Parallel benchmark
func BenchmarkCalculateRouteParallel(b *testing.B) {
    graph := setupTestGraph()
    start := 30000001
    end := 30000142
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            ShortestPath(graph, start, end)
        }
    })
}
```

### 2. Load Testing Pattern (k6)

**Principle:** Progressive load increase with thresholds

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '2m', target: 10 },   // Ramp-up to 10 users
    { duration: '5m', target: 10 },   // Stay at 10 for 5 minutes
    { duration: '2m', target: 50 },   // Ramp-up to 50 users
    { duration: '5m', target: 50 },   // Stay at 50 for 5 minutes
    { duration: '2m', target: 0 },    // Ramp-down to 0
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],  // 95% of requests < 500ms
    http_req_failed: ['rate<0.01'],    // Error rate < 1%
  },
};

export default function () {
  const res = http.get('http://localhost:3000/api/routes');
  
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  sleep(1);
}
```

### 3. Database Query Benchmarking

**Principle:** Isolate and measure query performance

```go
func BenchmarkMarketOrderQuery(b *testing.B) {
    db := setupTestDB(b)
    defer db.Close()
    
    ctx := context.Background()
    regionID := 10000002
    typeID := 34
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        _, err := db.Query(ctx, `
            SELECT order_id, price, volume_remain
            FROM market_orders
            WHERE region_id = $1 AND type_id = $2
            ORDER BY price ASC
            LIMIT 100
        `, regionID, typeID)
        
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

---

## Best Practices

### 1. Benchmark Setup Isolation

**Exclude setup from measurements**

```go
func BenchmarkCacheOperation(b *testing.B) {
    // Setup (not measured)
    cache := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer cache.Close()
    
    testData := generateTestData(1000)
    
    b.ResetTimer() // Reset timer before measurement
    
    for i := 0; i < b.N; i++ {
        cache.Set(ctx, fmt.Sprintf("key:%d", i), testData, 0)
    }
    
    b.StopTimer() // Stop timer for cleanup if needed
}
```

### 2. Realistic Load Patterns

**Model actual user behavior**

```javascript
// k6: Realistic user flow
export default function () {
  // 1. Login
  const loginRes = http.post('http://localhost:3000/api/login', {
    email: 'user@example.com',
    password: 'password'
  });
  
  const token = loginRes.json('token');
  sleep(2); // Think time
  
  // 2. Browse markets
  http.get('http://localhost:3000/api/markets', {
    headers: { Authorization: `Bearer ${token}` }
  });
  sleep(3);
  
  // 3. Calculate routes
  http.post('http://localhost:3000/api/routes', JSON.stringify({
    regions: [10000002, 10000043]
  }), {
    headers: { 
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json'
    }
  });
  
  sleep(5); // User reviews results
}
```

### 3. Performance Baselines

**Establish and track baselines**

```go
// Store baseline results
const (
    BaselineRouteCalculation = 50 * time.Millisecond
    BaselineDBQuery          = 10 * time.Millisecond
    BaselineCacheHit         = 1 * time.Millisecond
)

func BenchmarkWithBaseline(b *testing.B) {
    var totalDuration time.Duration
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        doWork()
        totalDuration += time.Since(start)
    }
    
    avgDuration := totalDuration / time.Duration(b.N)
    
    if avgDuration > BaselineRouteCalculation*2 {
        b.Fatalf("Performance regression: %v (baseline: %v)", 
            avgDuration, BaselineRouteCalculation)
    }
}
```

### 4. Memory Profiling

**Track allocations and memory usage**

```go
func BenchmarkWithMemory(b *testing.B) {
    b.ReportAllocs() // Report allocations
    
    for i := 0; i < b.N; i++ {
        processLargeDataset()
    }
}

// Run with: go test -bench=. -benchmem
// Output includes: allocs/op and B/op
```

### 5. Stress Testing Boundaries

**Find breaking points safely**

```javascript
// k6: Stress test with gradual ramp-up
export const options = {
  stages: [
    { duration: '5m', target: 100 },
    { duration: '5m', target: 200 },
    { duration: '5m', target: 300 },
    { duration: '5m', target: 400 },  // Find where it breaks
    { duration: '10m', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.1'],  // Allow up to 10% errors
  },
};
```

### 6. Continuous Performance Monitoring

**Integrate into CI/CD**

```yaml
# GitHub Actions: Benchmark comparison
- name: Run benchmarks
  run: |
    go test -bench=. -benchmem > new.txt
    
- name: Compare with baseline
  run: |
    benchstat baseline.txt new.txt
```

---

## Common Patterns

### Pattern 1: Worker Pool Benchmarking

```go
func BenchmarkWorkerPool(b *testing.B) {
    tests := []struct {
        name    string
        workers int
    }{
        {"1 worker", 1},
        {"5 workers", 5},
        {"10 workers", 10},
        {"20 workers", 20},
    }
    
    for _, tt := range tests {
        b.Run(tt.name, func(b *testing.B) {
            pool := NewWorkerPool(tt.workers)
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                pool.Submit(func() { doWork() })
            }
            pool.Wait()
        })
    }
}
```

### Pattern 2: Cache Performance Testing

```go
func BenchmarkCacheHitRate(b *testing.B) {
    cache := NewCache()
    
    // Pre-populate cache
    for i := 0; i < 1000; i++ {
        cache.Set(fmt.Sprintf("key:%d", i), fmt.Sprintf("value:%d", i))
    }
    
    b.ResetTimer()
    
    hits := 0
    misses := 0
    
    for i := 0; i < b.N; i++ {
        key := fmt.Sprintf("key:%d", i%1500) // 66% hit rate
        _, found := cache.Get(key)
        if found {
            hits++
        } else {
            misses++
        }
    }
    
    b.ReportMetric(float64(hits)/float64(hits+misses)*100, "hit-rate%")
}
```

### Pattern 3: API Endpoint Load Test

```javascript
// k6: Test multiple endpoints
import { group } from 'k6';

export default function () {
  group('API Endpoints', function () {
    group('Markets', function () {
      http.get('http://localhost:3000/api/markets');
    });
    
    group('Orders', function () {
      http.get('http://localhost:3000/api/orders/10000002/34');
    });
    
    group('Routes', function () {
      http.post('http://localhost:3000/api/routes', JSON.stringify({
        regions: [10000002, 10000043]
      }));
    });
  });
  
  sleep(1);
}
```

### Pattern 4: Database Connection Pool Sizing

```go
func BenchmarkConnectionPool(b *testing.B) {
    poolSizes := []int{5, 10, 20, 50, 100}
    
    for _, size := range poolSizes {
        b.Run(fmt.Sprintf("pool-size-%d", size), func(b *testing.B) {
            db := setupDBWithPoolSize(b, size)
            defer db.Close()
            
            b.RunParallel(func(pb *testing.PB) {
                for pb.Next() {
                    _, err := db.Query(ctx, "SELECT 1")
                    if err != nil {
                        b.Fatal(err)
                    }
                }
            })
        })
    }
}
```

---

## Anti-Patterns

### 1. Benchmarking with Side Effects
❌ **Avoid:** Benchmarks that modify global state
✅ **Do:** Isolate benchmarks, use local state

### 2. Ignoring Warmup
❌ **Avoid:** Measuring cold start performance
✅ **Do:** Run warmup iterations before measurement

### 3. Non-Representative Load
❌ **Avoid:** Uniform traffic patterns
✅ **Do:** Model realistic user behavior, think times

### 4. No Performance Budgets
❌ **Avoid:** Benchmarking without thresholds
✅ **Do:** Set and enforce performance budgets

### 5. Testing in Development Environment
❌ **Avoid:** Load testing on local machine
✅ **Do:** Use production-like environment

---

## Integration Workflows

### With CI/CD

```yaml
# GitHub Actions: Performance regression detection
name: Performance Tests

on: [pull_request]

jobs:
  benchmark:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v4
        
      - name: Run benchmarks
        run: go test -bench=. -benchmem -run=^$ ./... | tee new.txt
        
      - name: Download baseline
        run: gh release download latest -p benchmark-baseline.txt
        
      - name: Compare performance
        run: |
          go install golang.org/x/perf/cmd/benchstat@latest
          benchstat benchmark-baseline.txt new.txt
```

### With Monitoring

```go
// Export benchmark metrics to Prometheus
func BenchmarkWithMetrics(b *testing.B) {
    registry := prometheus.NewRegistry()
    duration := prometheus.NewHistogram(prometheus.HistogramOpts{
        Name: "benchmark_duration_seconds",
        Help: "Benchmark execution duration",
    })
    registry.MustRegister(duration)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        start := time.Now()
        doWork()
        duration.Observe(time.Since(start).Seconds())
    }
}
```

---

## Performance Budgets

### API Response Times

| Endpoint | p50 | p95 | p99 |
|----------|-----|-----|-----|
| GET /markets | 50ms | 100ms | 200ms |
| POST /routes | 200ms | 500ms | 1000ms |
| GET /orders | 30ms | 80ms | 150ms |

### Resource Limits

| Resource | Threshold | Action |
|----------|-----------|--------|
| CPU Usage | >80% sustained | Scale horizontally |
| Memory | >85% | Optimize/Scale |
| DB Connections | >80% pool | Increase pool size |
| Response Errors | >1% | Alert/Investigate |

---

## Quick Reference

### Go Benchmark Commands

```bash
# Run all benchmarks
go test -bench=.

# Run with memory stats
go test -bench=. -benchmem

# Run specific benchmark
go test -bench=BenchmarkRoute

# Save baseline
go test -bench=. > baseline.txt

# Compare with baseline
benchstat baseline.txt new.txt

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof

# Memory profiling
go test -bench=. -memprofile=mem.prof

# View profile
go tool pprof cpu.prof
```

### k6 Commands

```bash
# Run load test
k6 run script.js

# Run with custom VUs
k6 run --vus 10 --duration 30s script.js

# Output to InfluxDB
k6 run --out influxdb=http://localhost:8086/k6 script.js

# Run in cloud
k6 cloud script.js
```

---

## Performance Considerations

### 1. Benchmark Accuracy
- Run multiple iterations (default: automatic)
- Isolate CPU cores (`GOMAXPROCS=1`)
- Disable frequency scaling
- Close background applications

### 2. Load Test Infrastructure
- Dedicated load generator machines
- Network latency consideration
- Database in production-like state
- Monitoring overhead

### 3. Results Interpretation
- Look for patterns, not single values
- Compare distributions (p50, p95, p99)
- Watch for outliers
- Track trends over time

---

This skill provides systematic approaches to performance testing and benchmarking. Use appropriate test types based on your goals (optimization, capacity planning, stability).
