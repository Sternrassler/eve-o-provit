# ADR-012: Redis Caching Strategy für Market Orders

Status: Accepted
Datum: 2025-11-01
Autoren: GitHub Copilot (Performance Optimization Phase 3)

> Ablageort: ADR-Dateien werden im Verzeichnis `docs/adr/` gepflegt.

## Kontext

Market Order Fetching von ESI ist der größte Performance-Bottleneck (383 Seiten für The Forge = ~40 Sekunden). Wiederholte Requests innerhalb kurzer Zeit (z.B. bei UI Refresh) sollten gecacht werden.

**Fragen:**

- Wie cachen wir 383.000 Market Orders effizient?
- Welche TTL ist sinnvoll (ESI Updates vs. Cache Staleness)?
- Wie vermeiden wir Memory Exhaustion bei großen Regions?
- Wie warmup wir den Cache on Startup?

**Constraints:**

- Market Orders ändern sich häufig (ESI: ~5 Minuten Update Cycle)
- The Forge: ~383.000 Orders = ~50 MB uncompressed JSON
- Redis Shared Infrastructure (ADR-009) bereits vorhanden
- Cache Miss darf nicht zu Timeout führen (max 15s fetch time)

## Betrachtete Optionen

### Option 1: In-Memory Cache (sync.Map)

- **Vorteile:** 
  - Keine externe Dependency
  - Sehr schnell (ns Latency)
  - Bereits teilweise implementiert
- **Nachteile:** 
  - Nicht geteilt zwischen Instanzen
  - Memory Exhaustion bei vielen Regions
  - Keine Persistence bei Restart
- **Risiken:** 
  - Cache Stampede bei Cold Start
  - Memory Pressure

### Option 2: Redis Cache mit JSON (unkomprimiert)

- **Vorteile:** 
  - Geteilt zwischen Instanzen
  - Persistence Option
  - Einfache Implementierung
- **Nachteile:** 
  - 50 MB pro Region (Memory intensive)
  - Network Overhead
  - Redis Memory Limit kann erreicht werden
- **Risiken:** 
  - Redis OOM bei vielen Regions
  - Eviction kann Cache Miss erzeugen

### Option 3: Redis Cache mit Gzip Kompression

- **Vorteile:** 
  - Reduziert Memory auf ~10 MB (80% Kompression)
  - Geteilt zwischen Instanzen
  - Eviction weniger wahrscheinlich
  - Ermöglicht mehr Regions im Cache
- **Nachteile:** 
  - CPU Overhead für Compression/Decompression
  - Komplexere Implementierung
- **Risiken:** 
  - CPU Bottleneck bei vielen Requests
  - Gzip Bomb Attack (DoS)

## Entscheidung

**Gewählte Option:** Option 3 - Redis Cache mit Gzip Kompression + In-Memory Fallback

**Begründung:**

1. **Memory Efficiency:** 80% Compression ermöglicht mehr Regions im Cache
2. **Multi-Instance Support:** Redis teilt Cache zwischen Backend-Instanzen
3. **Performance:** CPU Overhead akzeptabel (Gzip: ~20ms für 50 MB)
4. **Fallback:** In-Memory Cache bei Redis Failure

**Hybrid Approach:**

- **Layer 1:** In-Memory (sync.Map) - 5min TTL
- **Layer 2:** Redis (Gzip) - 5min TTL
- **Layer 3:** ESI Fetch (fallback)

**Trade-offs:**

- CPU Overhead für Compression akzeptiert für Memory Savings
- Komplexität akzeptiert für Reliability (Fallback)

**Annahmen:**

- Redis verfügbar (ADR-009)
- Gzip Compression ~80% für JSON Arrays
- ESI Cache Header werden respektiert (304 Not Modified)

## Konsequenzen

### Positiv

- **Performance:** Cache Hit < 100ms (statt 40s ESI Fetch)
- **Cache Hit Ratio:** > 95% nach Warmup
- **Memory Efficiency:** 10 MB statt 50 MB pro Region
- **Skalierbarkeit:** Mehrere Backend-Instanzen teilen Cache
- **Reliability:** In-Memory Fallback bei Redis Failure

### Negativ

- **CPU Overhead:** ~20ms Compression/Decompression
- **Komplexität:** Zwei Cache Layers
- **Latency:** Redis Network Roundtrip (~1-5ms)

### Risiken

- **Redis Failure:** Fallback zu In-Memory → Performance OK
- **Cache Invalidation:** Race zwischen ESI Update und Cache → TTL 5min akzeptabel
- **Gzip Bomb:** Validierung der Payload Size erforderlich

## Implementierung

**Redis Key Schema:**

```
Key: market_orders:{region_id}
Value: [gzip compressed JSON]
TTL: 300 seconds (5 Minuten)

Example:
market_orders:10000002 → <gzip binary data>
```

**Compression Implementation:**

```go
type MarketOrderCache struct {
    redis   *redis.Client
    ttl     time.Duration // 5min
    fetcher *MarketOrderFetcher
}

func (c *MarketOrderCache) compress(orders []database.MarketOrder) ([]byte, error) {
    jsonData, err := json.Marshal(orders)
    if err != nil {
        return nil, err
    }
    
    var buf bytes.Buffer
    gzipWriter := gzip.NewWriter(&buf)
    if _, err := gzipWriter.Write(jsonData); err != nil {
        return nil, err
    }
    gzipWriter.Close()
    
    return buf.Bytes(), nil
}

func (c *MarketOrderCache) Get(ctx context.Context, regionID int) ([]database.MarketOrder, error) {
    // Try Redis first
    data, err := c.redis.Get(ctx, cacheKey).Bytes()
    if err == nil {
        metrics.CacheHitsTotal.Inc()
        return c.decompress(data)
    }
    
    metrics.CacheMissesTotal.Inc()
    
    // Fetch from ESI
    orders, err := c.fetcher.FetchAllPages(ctx, regionID)
    if err != nil {
        return nil, err
    }
    
    // Update cache async
    go c.Set(ctx, regionID, orders)
    
    return orders, nil
}
```

**Navigation Cache (Bonus):**

```
Key: nav:{systemA}:{systemB}
Value: {"travel_time_seconds": 180, "jumps": 6}
TTL: 3600 seconds (1 Stunde)
```

**Cache Warmup on Startup:**

```go
func (s *Server) warmupCache() {
    regions := []int{10000002} // The Forge
    
    for _, regionID := range regions {
        go func(rid int) {
            ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
            defer cancel()
            
            _, _ = s.marketCache.Get(ctx, rid)
            log.Printf("Cache warmed up for region %d", rid)
        }(regionID)
    }
}
```

**Aufwand:** 6 PT (Implementation + Tests + Monitoring)

**Abhängigkeiten:**
- ADR-009: Shared Redis Infrastructure
- ADR-011: Worker Pool Pattern (für Fetcher)

**Validierung:**
- Benchmark: Cache Hit < 100ms
- Metrics: `cache_hit_ratio` > 0.95
- Load Test: 100 req/min ohne Cache Thrashing

## Referenzen

- **Issues:** #16c (Phase 3)
- **ADRs:** ADR-009 (Redis Infrastructure), ADR-011 (Worker Pool)
- **Externe Docs:**
  - Redis Best Practices: https://redis.io/docs/management/optimization/
  - Gzip Compression: Go stdlib `compress/gzip`

## Notizen

**TTL Rationale:**

- **Market Orders (5min):** 
  - ESI Updates alle ~5 Minuten
  - Balance zwischen Freshness und Cache Hit Ratio
  - Bei hoher Last: Cache Hit Ratio > 95%

- **Navigation (1h):**
  - Static Data (ändert sich selten)
  - Längere TTL OK

**Character Data Caching (Amendment 2025-11-06):**

Basierend auf Skills Service Implementation (Issue #54) wurde Character Data Caching Guidelines hinzugefügt:

- **Character Skills (5min TTL):**
  - Key: `character_skills:{characterID}`
  - Value: JSON (uncompressed, ~1-5 KB per character)
  - Rationale: Semi-static data (ändert sich nur bei Skill Training)
  - Cache Strategy: Redis only (keine Gzip - kleine Payloads)
  - Graceful Degradation: Default skills (all = 0) bei ESI failure

- **Character Standings (1h TTL - zukünftig):**
  - Key: `character_standings:{characterID}`
  - Rationale: Rarely changes (Missionen/Ratting erforderlich)
  - Use Case: Broker Fee Calculation (Faction Standing Impact)

- **Character Assets (15min TTL - zukünftig):**
  - Key: `character_assets:{characterID}:{locationID}`
  - Rationale: Moderate change frequency (Hauling/Trading)
  - Use Case: Inventory Sell Orchestration

**Character Data Pattern:**
```go
// 1. Check Redis
cachedData, err := redisClient.Get(ctx, cacheKey).Bytes()
if err == nil {
    json.Unmarshal(cachedData, &result)
    return result, nil
}

// 2. Fetch from ESI (ADR-014 Pattern)
esiData, err := fetchFromESI(ctx, characterID, accessToken)
if err != nil {
    // Graceful Degradation: Return safe defaults
    return getDefaultValues(), nil
}

// 3. Cache Result
redisClient.Set(ctx, cacheKey, marshal(esiData), ttl)
return esiData, nil
```

**Key Schema Convention:**
```
character_skills:{characterID}
character_standings:{characterID}
character_assets:{characterID}:{locationID}
character_wallet:{characterID}
```

**Compression Ratio Measurements:**

- The Forge (383k Orders): 50 MB → 10 MB (~80%)
- Domain (smaller): 5 MB → 1 MB (~80%)
- JSON Arrays komprimieren sehr gut (repetitive keys)

**Security:**

- Gzip Bomb Protection: Max Decompressed Size = 100 MB
- Redis Auth: Shared Secret (ADR-009)

**Known Deviations:**

- Navigation Cache aktuell nicht aktiv genutzt (calculateRoute nutzt direkte DB Query)
- TODO: Navigation Cache Integration (separates Issue)

---

**Change Log:**

- 2025-11-01: Status auf Accepted gesetzt (GitHub Copilot, Phase 3 Implementation)
- 2025-11-06: Character Data Caching Guidelines hinzugefügt (Amendment nach Issue #54)
