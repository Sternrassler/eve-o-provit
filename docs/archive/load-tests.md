# Load Tests – Performance Validation

Dieses Dokument beschreibt die Load Test Suite für **Performance Target Validation** (<30s The Forge Berechnung).

## Übersicht

**Ziel:** Sicherstellen, dass die Backend-Performance für The Forge Region unter **30 Sekunden** bleibt (mit allen ~383.000 Market Orders).

**Test Coverage:**

1. ✅ **Cold Cache** - Initiale Berechnung mit leerer Redis (Target: <30s)
2. ✅ **Warm Cache** - Wiederholte Anfrage mit Cache Hit (Target: <5s)
3. ✅ **Concurrent Load** - 10 parallele Regionen ohne Timeout
4. ✅ **Cache Hit Ratio** - >95% Cache Hits bei 100 Requests

## Voraussetzungen

### Lokale Umgebung

**Docker Compose (erforderlich):**

```bash
# Redis + PostgreSQL starten
cd deployments
docker-compose up -d

# Verifizieren
docker ps | grep redis
```

**SDE Database (erforderlich):**

```bash
# Download & Extract
make db-load

# Verify
ls -lh backend/data/sde/eve-sde.db
# Expected: ~500MB SQLite file
```

**Go 1.24+ (erforderlich):**

```bash
go version
# go version go1.24.0 linux/amd64
```

## Ausführung

### Make Targets

**Load Tests (alle Szenarien):**

```bash
make test-load
```

**Output:**

```
[make test-load] Führe Load Tests aus (Redis + SDE erforderlich)...
=== RUN   TestLoadTheForge_ColdCache
    load_test.go:67: 🧪 Load Test: The Forge Cold Cache (383k+ orders expected)
    load_test.go:76: ✅ Fetched 383170 orders in 24.3s
    load_test.go:80: 📊 Metrics:
    load_test.go:81:   - Total Orders: 383170
    load_test.go:82:   - Total Time: 24.30s
    load_test.go:83:   - Throughput: 15771 orders/sec
--- PASS: TestLoadTheForge_ColdCache (24.30s)

=== RUN   TestLoadTheForge_WarmCache
    load_test.go:106: 🔥 Priming cache...
    load_test.go:113: 🧪 Load Test: The Forge Warm Cache
    load_test.go:119: ✅ Fetched 383170 orders from cache in 2.8s
    load_test.go:125: 📊 Metrics:
    load_test.go:126:   - Total Orders: 383170
    load_test.go:127:   - Cache Hit Time: 2.80s
    load_test.go:128:   - Speedup: 11x faster than cold cache
--- PASS: TestLoadTheForge_WarmCache (27.10s)

=== RUN   TestLoadConcurrent
    load_test.go:152: 🧪 Load Test: 10 Concurrent Requests
    load_test.go:182: ✅ Region 10000002: 383170 orders in 25.1s
    load_test.go:182: ✅ Region 10000043: 125340 orders in 18.3s
    ...
    load_test.go:200: 📊 Concurrent Load Metrics:
    load_test.go:201:   - Total Regions: 10
    load_test.go:202:   - Success: 10, Failures: 0
    load_test.go:203:   - Total Orders Fetched: 1245678
    load_test.go:204:   - Total Time (parallel): 28.7s
    load_test.go:205:   - Max Individual Time: 25.1s
    load_test.go:206:   - Throughput: 43392 orders/sec
--- PASS: TestLoadConcurrent (28.70s)

=== RUN   TestLoadCacheHitRatio
    load_test.go:227: 🧪 Load Test: Cache Hit Ratio (100 requests)
    load_test.go:230: 🔥 Priming cache with first request per region...
    load_test.go:256: 📊 Cache Hit Ratio Metrics:
    load_test.go:257:   - Total Requests: 100
    load_test.go:258:   - Cache Hits: 98
    load_test.go:259:   - Cache Misses: 2
    load_test.go:260:   - Hit Ratio: 98.00%
    load_test.go:261:   - Total Time: 4.2s
    load_test.go:262:   - Avg Time per Request: 42ms
--- PASS: TestLoadCacheHitRatio (38.50s)

PASS
ok      github.com/Sternrassler/eve-o-provit/backend/internal/services  120.600s
```

**Benchmarks (Go Bench):**

```bash
make test-load-bench
```

**Output:**

```
BenchmarkTheForgeCalculation-16
  orders: 383170
  item_pairs: 1523
  seconds: 24.315
  24315 ms/op
```

### Manuelle Ausführung

**Einzelner Test:**

```bash
cd backend
go test -v -timeout 10m -tags=load -run TestLoadTheForge_ColdCache ./internal/services/
```

**Mit Race Detector:**

```bash
go test -v -race -timeout 10m -tags=load ./internal/services/
```

**Benchmark (3 Iterationen):**

```bash
go test -bench=BenchmarkTheForge -benchtime=3x -tags=load ./internal/services/
```

## Test Szenarien

### 1. Cold Cache Test

**Ziel:** Validate <30s für initiale Market Order Fetch (The Forge)

**Setup:**

- Redis DB flushed (leerer Cache)
- Region: The Forge (10000002)
- Expected Orders: ~383.000

**Assertions:**

- ✅ Fetched Orders > 100.000
- ✅ Total Time < 30s
- ✅ No ESI 429 Rate Limit errors

**Metriken:**

- Total Orders
- Total Time
- Throughput (orders/sec)

---

### 2. Warm Cache Test

**Ziel:** Validate <5s für Cache Hit

**Setup:**

- Prime cache mit Cold Request
- Second request hits cache

**Assertions:**

- ✅ Fetched Orders > 100.000
- ✅ Total Time < 5s
- ✅ Cache Hit (Redis GET)

**Metriken:**

- Cache Hit Time
- Speedup vs Cold Cache

---

### 3. Concurrent Load Test

**Ziel:** Validate concurrent requests ohne Timeouts

**Setup:**

- 10 parallele Requests
- Verschiedene Regionen (The Forge, Domain, Sinq Laison, ...)

**Assertions:**

- ✅ All 10 Requests successful
- ✅ Max Individual Time < 30s
- ✅ No 429 Rate Limit errors

**Metriken:**

- Total Orders Fetched (alle Regionen)
- Max Individual Time
- Total Throughput

---

### 4. Cache Hit Ratio Test

**Ziel:** Validate >95% Cache Hit Ratio

**Setup:**

- 10 Regionen
- 10 Requests pro Region (100 total)
- Prime cache mit 1. Request pro Region

**Assertions:**

- ✅ Cache Hit Ratio > 95%
- ✅ Avg Request Time < 100ms

**Metriken:**

- Total Requests
- Cache Hits / Misses
- Hit Ratio (%)

## CI Integration

**GitHub Actions Workflow:** `.github/workflows/load-tests.yml`

**Trigger:** Manual (`workflow_dispatch` only)

**Warum Manual?**

- Load Tests sind zeitintensiv (~2-3 Minuten)
- Erfordern ESI API Zugriff (Rate Limits)
- Sollten nicht bei jedem PR laufen

**Verwendung:**

```bash
# Via GitHub UI:
Actions → Load Tests → Run workflow → Branch: main

# Via gh CLI:
gh workflow run load-tests.yml --repo Sternrassler/eve-o-provit
```

## Troubleshooting

### Redis Connection Failed

**Fehler:**

```
Redis must be running on localhost:6379
```

**Lösung:**

```bash
# Start Redis
docker-compose up -d redis

# Verify
redis-cli ping
# PONG
```

---

### SDE Database Not Found

**Fehler:**

```
SDE database must exist at backend/data/sde/eve-sde.db
```

**Lösung:**

```bash
# Download SDE
make db-load

# Verify
file backend/data/sde/eve-sde.db
# eve-sde.db: SQLite 3.x database
```

---

### ESI Rate Limit (429)

**Fehler:**

```
ESI responded with 429 Too Many Requests
```

**Lösung:**

- Wait 60 seconds zwischen Test-Runs
- Verwende ESI Sandbox statt Tranquility
- Reduziere Worker Count (10 → 5)

---

### Test Timeout

**Fehler:**

```
test timed out after 10m0s
```

**Lösung:**

```bash
# Increase timeout
go test -v -timeout 15m -tags=load ./internal/services/
```

## Performance Targets (Summary)

| Szenario | Target | Aktuell | Status |
|----------|--------|---------|--------|
| Cold Cache (The Forge) | <30s | 24.3s | ✅ |
| Warm Cache | <5s | 2.8s | ✅ |
| Concurrent (Max) | <30s | 25.1s | ✅ |
| Cache Hit Ratio | >95% | 98.0% | ✅ |

## Nächste Schritte

**Optimierungen:**

1. ESI Batch Requests (aktuell: einzelne Pages)
2. Redis Pipeline (aktuell: einzelne SET/GET)
3. Gzip Compression Level Tuning (Best Speed vs Best Compression)

**Monitoring:**

1. Grafana Dashboard für Load Test Metriken
2. Prometheus Metrics Export
3. Alerting bei Performance Regression

## Referenzen

- **Issue:** [#27 - Load Test to Verify <30s Performance Target](https://github.com/Sternrassler/eve-o-provit/issues/27)
- **ADR-011:** Worker Pool Pattern
- **ADR-012:** Redis Caching Strategy
- **ADR-013:** Timeout Handling & Partial Content
- **Go Benchmarks:** <https://pkg.go.dev/testing#hdr-Benchmarks>
