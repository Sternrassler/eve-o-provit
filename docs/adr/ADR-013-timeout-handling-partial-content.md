# ADR-013: Timeout Handling mit HTTP 206 Partial Content

Status: Accepted
Datum: 2025-11-01
Autoren: GitHub Copilot (Performance Optimization Phase 3)

> Ablageort: ADR-Dateien werden im Verzeichnis `docs/adr/` gepflegt.

## Kontext

Trading Route Berechnung kann bei großen Regionen länger dauern als gewünscht. User Experience leidet, wenn Requests nach 30 Sekunden ohne Ergebnis abgebrochen werden.

**Fragen:**

- Wie kommunizieren wir Timeout transparent an Frontend?
- Sollen wir Partial Results zurückgeben oder 504 Gateway Timeout?
- Wie verhindern wir Timeout Cascades (Request Retry Storms)?
- Welche HTTP Status Codes sind semantisch korrekt?

**Constraints:**

- Total Timeout: 30 Sekunden (User Experience)
- Market Fetch: max 15 Sekunden
- Route Calculation: max 25 Sekunden (inkl. Sorting)
- Frontend muss Partial Results anzeigen können

## Betrachtete Optionen

### Option 1: HTTP 504 Gateway Timeout (No Results)

- **Vorteile:** 
  - Semantisch korrekt für Timeout
  - Standard HTTP Status
  - Einfache Implementierung
- **Nachteile:** 
  - User bekommt keine Ergebnisse
  - Verschwendete CPU Zeit (Partial Results verworfen)
  - Schlechte UX
- **Risiken:** 
  - User frustriert bei wiederholten Timeouts
  - Retry Storms möglich

### Option 2: HTTP 200 OK mit Warning (Implicit Partial)

- **Vorteile:** 
  - User bekommt Ergebnisse
  - Einfache Frontend-Integration
  - Keine Status Code Änderung
- **Nachteile:** 
  - Semantisch unklar (200 = Success, aber Timeout?)
  - Frontend kann Warning übersehen
  - Nicht HTTP-Standard-konform
- **Risiken:** 
  - Misleading für API Clients
  - Schwer zu monitoren (Logs zeigen nur 200)

### Option 3: HTTP 206 Partial Content (Explicit Partial)

- **Vorteile:** 
  - Semantisch korrekt für Partial Results
  - Standard HTTP Status (Range Requests)
  - Frontend kann Status Code prüfen
  - Warning Header für Details
  - Monitoring unterscheidet Success/Partial
- **Nachteile:** 
  - 206 ursprünglich für Range Requests
  - Nicht alle HTTP Clients unterstützen 206
  - Komplexer zu implementieren
- **Risiken:** 
  - Misinterpretation von 206 (nicht im ursprünglichen Kontext)

## Entscheidung

**Gewählte Option:** Option 3 - HTTP 206 Partial Content mit Warning Header

**Begründung:**

1. **User Experience:** Partial Results sind besser als keine Results
2. **Semantik:** 206 = "Incomplete Response" ist akzeptabel für Timeout
3. **Monitoring:** Status Code 206 ermöglicht Alerting
4. **Transparency:** Warning Header erklärt Grund

**Response Format:**

```http
HTTP/1.1 206 Partial Content
Warning: 199 - "Calculation timeout after 30s, showing partial results"
Content-Type: application/json

{
  "warning": "Calculation timeout after 30s, showing partial results",
  "region_id": 10000002,
  "routes": [ /* partial results */ ],
  "calculation_time_ms": 30000
}
```

**Trade-offs:**

- 206 außerhalb ursprünglicher Verwendung (Range Requests) akzeptiert
- Komplexität akzeptiert für bessere UX und Monitoring

**Annahmen:**

- Frontend kann 206 Status Code verarbeiten
- Partial Results sind nützlich (Top N Routes bereits sortiert)
- Context Cancellation unterstützt Partial Results

## Konsequenzen

### Positiv

- **User Experience:** User bekommt Top Routes auch bei Timeout
- **Monitoring:** 206 Status ermöglicht SLI/SLO Tracking
- **Transparency:** Warning Header erklärt Situation
- **Performance:** Partial Results = Resourcen nicht verschwendet
- **Graceful Degradation:** System funktioniert auch bei Überlast

### Negativ

- **Komplexität:** Context Handling in Worker Pools erforderlich
- **Semantik:** 206 nicht original Intent (aber akzeptabel)
- **Testing:** Timeout Scenarios schwer zu testen

### Risiken

- **Frontend Breakage:** Alte Clients erwarten nur 200/500
- **Cache Poisoning:** Partial Results dürfen nicht gecacht werden
- **Retry Storms:** Frontend muss 206 anders behandeln als 504

## Implementierung

**Backend (Handler):**

```go
func (h *TradingHandler) CalculateRoutes(c *fiber.Ctx) error {
    result, err := h.calculator.Calculate(c.Context(), regionID, shipTypeID, cargoCapacity)
    if err != nil {
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "error": "Failed to calculate routes",
        })
    }
    
    // Check for timeout warning (partial results)
    if result.Warning != "" {
        c.Set("Warning", `199 - "`+result.Warning+`"`)
        return c.Status(fiber.StatusPartialContent).JSON(result)
    }
    
    return c.JSON(result) // 200 OK
}
```

**RouteCalculator (Context Timeout):**

```go
func (rc *RouteCalculator) Calculate(ctx context.Context, ...) (*models.RouteCalculationResponse, error) {
    calcCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    
    // Market Fetch (max 15s)
    marketCtx, _ := context.WithTimeout(calcCtx, 15*time.Second)
    orders, err := rc.fetchMarketOrders(marketCtx, regionID)
    if errors.Is(err, context.DeadlineExceeded) {
        return nil, err // Hard fail if market fetch times out
    }
    
    // Route Calculation (max 25s)
    routeCtx, _ := context.WithTimeout(calcCtx, 25*time.Second)
    routes, err := rc.workerPool.ProcessItems(routeCtx, profitableItems, cargo)
    
    // Check if we timed out
    timedOut := errors.Is(routeCtx.Err(), context.DeadlineExceeded)
    
    response := &models.RouteCalculationResponse{
        Routes: routes, // Partial if timeout
    }
    
    if timedOut {
        response.Warning = fmt.Sprintf("Calculation timeout after %v, showing partial results", CalculationTimeout)
    }
    
    return response, nil
}
```

**Worker Pool (Context Aware):**

```go
func (p *RouteWorkerPool) worker(ctx context.Context, itemQueue <-chan models.ItemPair, results chan<- models.TradingRoute) {
    for item := range itemQueue {
        // Check for context cancellation
        select {
        case <-ctx.Done():
            return // Stop processing on timeout
        default:
        }
        
        route, err := p.calculator.calculateRoute(ctx, item, cargo)
        if err != nil {
            continue
        }
        
        select {
        case results <- route:
        case <-ctx.Done():
            return
        }
    }
}
```

**Frontend (React):**

```typescript
const response = await fetch('/api/v1/trading/routes/calculate', { ... });

if (response.status === 206) {
  const warning = response.headers.get('Warning');
  const data = await response.json();
  
  // Show partial results with warning banner
  showWarningBanner(data.warning);
  displayRoutes(data.routes);
} else if (response.status === 200) {
  const data = await response.json();
  displayRoutes(data.routes);
} else {
  // Error handling
}
```

**Aufwand:** 4 PT (Implementation + Frontend Integration + Tests)

**Abhängigkeiten:**
- ADR-011: Worker Pool Pattern (Context Support erforderlich)

**Validierung:**
- Integration Test: Mock Slow ESI → Verify 206
- Load Test: Concurrent Requests mit Timeout
- Frontend Test: UI zeigt Warning Banner

## Referenzen

- **Issues:** #16c (Phase 3)
- **ADRs:** ADR-011 (Worker Pool Pattern)
- **Externe Docs:**
  - HTTP 206: https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/206
  - Warning Header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Warning

## Notizen

**Timeout Budget:**

```
Total: 30s
├─ Market Fetch: 15s (max)
├─ Route Calculation: 25s (max, parallel with Market)
└─ Sorting: 5s Reserve
```

**Why 206 and not Custom Status?**

- Custom Status Codes (599) sind nicht Standard
- 206 ist HTTP Standard und wird von Proxies/CDNs verstanden
- Semantische Nähe: "Partial Content" = "Partial Results"

**Alternative Considered:**

- HTTP 202 Accepted (Async Processing) → Nicht geeignet (keine Async API)
- HTTP 203 Non-Authoritative → Nicht semantisch passend

**Security:**

- Partial Results dürfen nicht gecacht werden (Cache-Control: no-store)
- Metrics: Count 206 vs 200 (SLI für Performance)

**Known Deviations:**

- 206 ursprünglich für Range Requests (Content-Range Header)
- Wir nutzen 206 ohne Content-Range (akzeptabel für API)

---

**Change Log:**

- 2025-11-01: Status auf Accepted gesetzt (GitHub Copilot, Phase 3 Implementation)
