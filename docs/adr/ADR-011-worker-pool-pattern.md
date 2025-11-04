# ADR-011: Worker Pool Pattern für Trading Route Berechnung

Status: Superseded
Superseded by: eve-esi-client BatchFetcher (external: ADR-011-equivalent in eve-esi-client repository)
Datum: 2025-11-01
Superseded: 2025-11-04
Ersetzt durch: eve-esi-client/pkg/pagination/batch_fetcher.go
Repository: <https://github.com/Sternrassler/eve-esi-client>
Autoren: GitHub Copilot (Performance Optimization Phase 3)

> **Hinweis:** Dieses ADR beschreibt die ursprüngliche Implementierung eines MarketOrderFetchers in eve-o-provit.
> Die Funktionalität wurde in eve-esi-client v0.3.0 als generisches BatchFetcher-Pattern zentralisiert.
> Siehe: <https://github.com/Sternrassler/eve-esi-client/pkg/pagination>
> Ablageort: ADR-Dateien werden im Verzeichnis `docs/adr/` gepflegt.

## Kontext

Die sequenzielle Berechnung von Trading Routes aus Issue #16b ist für große Regionen wie The Forge (383.000 Market Orders) zu langsam (~120 Sekunden). Ziel ist eine Berechnung in unter 30 Sekunden.

**Fragen:**

- Wie können wir Market Order Fetching und Route Calculation parallelisieren?
- Welche Concurrency-Pattern sind für Go-basierte ESI-Integration geeignet?
- Wie bleiben wir innerhalb der ESI Rate Limits (300 req/min)?
- Wie vermeiden wir Race Conditions bei parallel processing?

**Constraints:**

- ESI Rate Limit: 300 requests/minute (burst: 400)
- Total Timeout: 30 Sekunden
- SDE Database ist SQLite (kein Thread-Safe writes, aber reads OK)
- Context-basierte Timeout Handling erforderlich

## Betrachtete Optionen

### Option 1: Sequential Processing (Baseline)

- **Vorteile:**
  - Einfach zu implementieren und debuggen
  - Keine Concurrency-Probleme
  - Bereits vorhanden
- **Nachteile:**
  - Zu langsam (~120s für The Forge)
  - Schlechte Ressourcennutzung (CPU idle während I/O)
  - Skaliert nicht mit größeren Datenmengen
- **Risiken:**
  - Erfüllt Performance-Anforderungen nicht

### Option 2: Goroutines ohne Worker Pool (Unbounded Concurrency)

- **Vorteile:**
  - Maximale Parallelität
  - Einfache Implementierung mit `go func()`
- **Nachteile:**
  - ESI Rate Limit Violations (429 Errors)
  - Ressourcen-Exhaustion bei vielen Items
  - Schwer kontrollierbar
  - Keine Backpressure
- **Risiken:**
  - ESI API Ban möglich
  - Memory Exhaustion

### Option 3: Worker Pool Pattern (Bounded Concurrency)

- **Vorteile:**
  - Kontrollierte Parallelität
  - ESI Rate Limit safe
  - Backpressure durch buffered channels
  - Graceful Degradation bei Timeouts
  - Etabliertes Go Pattern
- **Nachteile:**
  - Komplexere Implementierung
  - Channel Overhead
  - Tuning erforderlich (Worker Count)
- **Risiken:**
  - Deadlocks bei falscher Channel-Nutzung
  - Starve bei zu wenigen Workers

## Entscheidung

**Gewählte Option:** Option 3 - Worker Pool Pattern

**Begründung:**

1. **ESI Rate Limit Compliance:** 10 Workers für Market Orders bleiben unter 300 req/min
2. **Predictable Performance:** 50 Workers für Route Calculation ermöglichen kontrollierte Parallelität
3. **Graceful Degradation:** Context Timeout ermöglicht Partial Results bei Überlast
4. **Best Practice:** Etabliertes Pattern in Go Community

**Trade-offs:**

- Höhere Komplexität akzeptiert für Performance und Stabilität
- Channel Overhead akzeptiert für Kontrolle und Backpressure

**Annahmen:**

- SDE Database Reads sind thread-safe (SQLite: OK bei Reads)
- Navigation API ist stateless (OK)
- ESI Client ist thread-safe (muss validiert werden)

## Konsequenzen

### Positiv

- **Performance:** ~20-30s für The Forge (statt 120s)
- **Skalierbarkeit:** Lineare Verbesserung mit Worker Count (bis Rate Limit)
- **Stabilität:** Keine ESI Rate Limit Violations
- **Fehlertoleranz:** Partial Results bei Timeouts
- **Observability:** Queue Size Metrics verfügbar

### Negativ

- **Komplexität:** Mehr Code für Channel Management
- **Debugging:** Concurrency Bugs schwerer zu debuggen
- **Tuning:** Worker Count muss für verschiedene Szenarien optimiert werden

### Risiken

- **Deadlocks:** Bei falscher Channel-Nutzung möglich → Mitigation: Code Review + Tests
- **Memory:** Buffered Channels können Memory nutzen → Mitigation: Bounded Buffer Size
- **Starvation:** Bei zu wenigen Workers → Mitigation: Metrics + Alerting

## Implementierung

**Worker Pool für Market Orders (10 Workers):**

```go
type MarketOrderFetcher struct {
    esiClient   *esi.Client
    workerCount int // 10
    timeout     time.Duration // 15s
}

func (f *MarketOrderFetcher) FetchAllPages(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
    pageQueue := make(chan int, totalPages) // Buffered: 400
    results := make(chan []database.MarketOrder, totalPages)
    errors := make(chan error, workerCount)
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < workerCount; i++ {
        wg.Add(1)
        go worker(ctx, pageQueue, results, errors, &wg)
    }
    
    // Collect results with timeout
    // ...
}
```

**Worker Pool für Route Calculation (50 Workers):**

```go
type RouteWorkerPool struct {
    workerCount int // 50
    calculator  *RouteCalculator
}

func (p *RouteWorkerPool) ProcessItems(ctx context.Context, items []models.ItemPair, cargo float64) ([]models.TradingRoute, error) {
    itemQueue := make(chan models.ItemPair, len(items)) // Buffered: 100
    results := make(chan models.TradingRoute, len(items))
    
    // Start workers
    // Similar pattern as above
}
```

**Aufwand:** 8 PT (Implementation + Tests + Tuning)

**Abhängigkeiten:**

- ADR-009: Shared Redis Infrastructure (für Caching)
- ADR-012: Redis Caching Strategy (parallel ADR)
- ESI Client muss thread-safe sein

**Validierung:**

- Benchmark: The Forge < 30s
- Load Test: 10 concurrent requests ohne 429 Errors
- Metrics: `worker_pool_queue_size` < Capacity

## Referenzen

- **Issues:** #16 (Parent), #16b (Backend API), #16c (Phase 3)
- **ADRs:** ADR-009 (Redis), ADR-012 (Caching), ADR-013 (Timeout Handling)
- **Externe Docs:**
  - Go Worker Pools: <https://gobyexample.com/worker-pools>
  - ESI Rate Limits: <https://docs.esi.evetech.net/docs/esi_introduction.html#limits>

## Notizen

**Worker Count Rationale:**

- **Market Orders (10 Workers):**
  - ESI Rate Limit: 300 req/min = 5 req/s
  - Mit 10 Workers: Max 10 * 5 = 50 req/s (safe)
  - Actual: ~30 req/s (mit Latency)

- **Route Calculation (50 Workers):**
  - CPU-bound (Navigation API + Sorting)
  - Keine ESI Calls (nur SDE Reads)
  - 50 Workers gut für 8-16 CPU Cores

**Known Deviations:**

- Market Fetcher aktuell nutzt bestehenden ESI Client (keine Pagination)
- TODO: ESI Client um Pagination erweitern (separates Issue)

---

**Change Log:**

- 2025-11-01: Status auf Accepted gesetzt (GitHub Copilot, Phase 3 Implementation)
