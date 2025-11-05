# Architektur-Review: Testbarkeit & Technische Schuld

**Datum:** 5. November 2025  
**Kontext:** Coverage-Ziele (ESI 35% ✅, Services 30%, Handlers 40%)  
**Aktueller Stand:** Services 29.8%, Handlers 17.1%

## Executive Summary

**Kernproblem:** Die schwierige Testbarkeit ist primär auf **fehlende Abstraktionen** und **konkrete Typ-Abhängigkeiten** zurückzuführen, nicht auf grundlegend schlechte Architektur.

**Hauptursachen:**

1. ❌ **Keine Interfaces für Dependencies** → Unmöglich zu mocken
2. ❌ **Konkrete Typen in Konstruktoren** (`*database.DB`, `*esi.Client`, `*sql.DB`)
3. ❌ **Tight Coupling** zwischen Layern (Handler → Services → Database → ESI)
4. ⚠️ **God Objects** (RouteCalculator mit 6 Dependencies)
5. ⚠️ **Mixed Concerns** (Handler machen ESI-Calls + DB-Queries + Business Logic)

**Schweregrad:** **Mittel** - Architektur ist funktional solide, aber **nicht test-freundlich**.

---

## 1. Dependency Injection: Missing Interfaces

### Problem: Konkrete Typen statt Interfaces

**Handler Constructor:**

```go
// handlers.go
type Handler struct {
    db         *database.DB              // ❌ Konkreter Typ
    sdeRepo    *database.SDERepository   // ❌ Konkreter Typ
    marketRepo *database.MarketRepository // ❌ Konkreter Typ
    esiClient  *esi.Client               // ❌ Konkreter Typ
}

func New(db *database.DB, sdeRepo *database.SDERepository, 
         marketRepo *database.MarketRepository, esiClient *esi.Client) *Handler {
    return &Handler{...}
}
```

**RouteCalculator Constructor:**

```go
// services/route_calculator.go
type RouteCalculator struct {
    esiClient   *esi.Client               // ❌ Konkreter Typ
    marketRepo  *database.MarketRepository // ❌ Konkreter Typ
    sdeDB       *sql.DB                    // ❌ Standard library, nicht mockbar
    sdeRepo     *database.SDERepository    // ❌ Konkreter Typ
    cache       map[string]*models.CachedData
    redisClient *redis.Client              // ❌ Konkreter Typ (aber optional)
    // ...6 weitere Fields
}
```

### Impact auf Tests

**Unit Tests unmöglich:**

```go
// ❌ UNMÖGLICH: Handler braucht echte DB, ESI, Redis
func TestHealth(t *testing.T) {
    handler := New(nil, nil, nil, nil) // Panic bei h.db.Health()
}

// ❌ UNMÖGLICH: RouteCalculator braucht echte Dependencies
func TestCalculate(t *testing.T) {
    calc := NewRouteCalculator(nil, nil, nil, nil, nil) // Panic überall
}
```

**Nur Integration Tests möglich:**

```go
// ✅ FUNKTIONIERT: Aber langsam (2-3s PostgreSQL Container startup)
func TestMarketRepository_Integration(t *testing.T) {
    pgContainer := SetupPostgresContainer(t)
    repo := database.NewMarketRepository(pgContainer.Pool)
    // Real database operations...
}
```

### Lösungsvorschlag: Interface-basierte DI

**Refactoring-Plan:**

```go
// internal/database/interfaces.go (NEU)
package database

type HealthChecker interface {
    Health(ctx context.Context) error
}

type SDEQuerier interface {
    GetTypeInfo(ctx context.Context, typeID int) (*TypeInfo, error)
    GetSystemInfo(ctx context.Context, systemID int) (*SystemInfo, error)
    GetRegionInfo(ctx context.Context, regionID int) (*RegionInfo, error)
    SearchTypes(ctx context.Context, query string, limit int) ([]TypeSearchResult, error)
}

type MarketRepository interface {
    GetMarketOrders(ctx context.Context, regionID, typeID int) ([]MarketOrder, error)
    UpsertMarketOrders(ctx context.Context, orders []MarketOrder) error
    GetLatestFetchTime(ctx context.Context, regionID int) (time.Time, error)
}

// internal/handlers/handlers.go (REFACTORED)
type Handler struct {
    healthChecker HealthChecker    // ✅ Interface statt *database.DB
    sdeRepo       SDEQuerier        // ✅ Interface
    marketRepo    MarketRepository  // ✅ Interface
    esiClient     ESIClient         // ✅ Interface (zu definieren)
}

func New(hc HealthChecker, sde SDEQuerier, market MarketRepository, esi ESIClient) *Handler {
    return &Handler{
        healthChecker: hc,
        sdeRepo:       sde,
        marketRepo:    market,
        esiClient:     esi,
    }
}
```

**Mock-Test wird möglich:**

```go
// handlers_test.go
type mockSDEQuerier struct {
    typeInfo *TypeInfo
    err      error
}

func (m *mockSDEQuerier) GetTypeInfo(ctx context.Context, typeID int) (*TypeInfo, error) {
    return m.typeInfo, m.err
}

func TestGetType_Success(t *testing.T) {
    mockSDE := &mockSDEQuerier{
        typeInfo: &TypeInfo{ID: 34, Name: "Tritanium"},
    }
    handler := New(nil, mockSDE, nil, nil)
    
    // Test ohne echte Database!
}
```

**Aufwand:** ~2-4 Stunden pro Layer (Database, Services, Handlers)  
**Breaking Changes:** Ja - Konstruktor-Signaturen ändern sich  
**Migrations-Path:** Wrapper-Funktionen für Rückwärtskompatibilität während Migration

---

## 2. God Objects & Single Responsibility

### Problem: RouteCalculator hat zu viele Verantwortlichkeiten

**Aktuell (534 Zeilen, 12 Dependencies):**

```go
type RouteCalculator struct {
    // ESI Integration
    esiClient   *esi.Client
    rateLimiter *ESIRateLimiter
    
    // Database Access
    sdeDB       *sql.DB
    sdeRepo     *database.SDERepository
    marketRepo  *database.MarketRepository
    
    // Caching
    cache       map[string]*models.CachedData
    cacheMu     sync.RWMutex
    marketCache *MarketOrderCache
    navCache    *NavigationCache
    redisClient *redis.Client
    
    // Async Processing
    workerPool  *RouteWorkerPool
}

// Funktionen (in einer Klasse):
// - Calculate() - Orchestrierung
// - fetchMarketOrders() - ESI Integration
// - findProfitableItems() - Business Logic
// - calculateRoute() - Navigation Logic
// - getRegionName() - SDE Lookup
// - getSystemIDFromLocation() - SDE Lookup
// - getSystemSecurityStatus() - SDE Lookup
// - ... 8+ weitere Hilfsfunktionen
```

### Impact

- **Testbarkeit:** 0% Coverage (zu komplex für Unit Tests)
- **Maintenance:** Änderung an ESI betrifft Route-Calculation-Logik
- **Debugging:** 534 Zeilen Code in einer Datei
- **Parallelisierung:** State-Management in `cache` ist fehleranfällig

### Lösungsvorschlag: Service Decomposition

**Nach Single Responsibility Principle:**

```go
// services/market_fetcher.go (NEU)
type MarketFetcher struct {
    esiClient *esi.Client
    cache     *MarketOrderCache
}

func (f *MarketFetcher) FetchOrders(ctx context.Context, regionID int) ([]MarketOrder, error) {
    // ESI fetch logic
}

// services/profit_analyzer.go (NEU)
type ProfitAnalyzer struct {
    taxCalculator *TaxCalculator
}

func (a *ProfitAnalyzer) FindProfitable(orders []MarketOrder, minSpread float64) ([]ItemPair, error) {
    // Business logic only
}

// services/route_planner.go (NEU)
type RoutePlanner struct {
    sdeRepo     SDEQuerier
    navCache    *NavigationCache
}

func (p *RoutePlanner) PlanRoute(ctx context.Context, from, to int) (*Route, error) {
    // Navigation logic only
}

// services/route_calculator.go (REFACTORED - Orchestrator only)
type RouteCalculator struct {
    fetcher  *MarketFetcher
    analyzer *ProfitAnalyzer
    planner  *RoutePlanner
}

func (rc *RouteCalculator) Calculate(ctx context.Context, req Request) (*Response, error) {
    // Orchestration only - delegates to specialized services
    orders, err := rc.fetcher.FetchOrders(ctx, req.RegionID)
    items, err := rc.analyzer.FindProfitable(orders, MinSpread)
    routes, err := rc.planner.PlanRoutes(ctx, items)
    return &Response{Routes: routes}, nil
}
```

**Benefits:**

- ✅ Jeder Service testbar (isolierte Concerns)
- ✅ Mocking wird einfach (klare Interfaces)
- ✅ Parallelisierung sicher (kein shared State)
- ✅ Wiederverwendbarkeit (Fetcher auch für andere Features nutzbar)

**Aufwand:** ~1-2 Tage Refactoring  
**Risk:** Mittel (viel Logik muss verschoben werden)

---

## 3. Mixed Concerns in Handlers

### Problem: Handler machen zu viel

**Beispiel: GetMarketOrders (100 Zeilen)**

```go
func (h *Handler) GetMarketOrders(c *fiber.Ctx) error {
    // 1. Parameter Parsing (OK)
    regionID, err := strconv.Atoi(c.Params("region"))
    
    // 2. Business Logic Validation (OK)
    if regionID <= 0 { ... }
    
    // 3. ESI Direct Call (❌ Sollte in Service sein)
    refresh := c.QueryBool("refresh", false)
    if refresh {
        config := pagination.DefaultConfig()
        fetcher := pagination.NewBatchFetcher(h.esiClient.GetRawClient(), config)
        results, err := fetcher.FetchAllPages(c.Context(), endpoint)
        
        // 4. Data Transformation (❌ Sollte in Service sein)
        allOrders := make([]database.MarketOrder, 0)
        for pageNum := 1; pageNum <= len(results); pageNum++ {
            var orders []database.MarketOrder
            json.Unmarshal(pageData, &orders)
            // ...
        }
        
        // 5. Database Write (❌ Sollte in Service sein)
        h.marketRepo.UpsertMarketOrders(c.Context(), allOrders)
    }
    
    // 6. Database Read (❌ Sollte in Service sein)
    orders, err := h.esiClient.GetMarketOrders(c.Context(), regionID, typeID)
    
    // 7. Response Formatting (OK)
    return c.JSON(fiber.Map{...})
}
```

**Layers verletzt:**

- Handler → ESI (sollte über Service gehen)
- Handler → Database (sollte über Service gehen)
- Handler macht Business Logic (Pagination, Transformation)

### Lösungsvorschlag: Thin Handlers

```go
// internal/services/market_service.go (NEU)
type MarketService struct {
    esiClient  ESIClient
    marketRepo MarketRepository
}

func (s *MarketService) GetOrders(ctx context.Context, regionID, typeID int, refresh bool) ([]MarketOrder, error) {
    if refresh {
        // ESI fetch + pagination + transformation + DB write
        return s.fetchAndStoreOrders(ctx, regionID)
    }
    // DB read only
    return s.marketRepo.GetMarketOrders(ctx, regionID, typeID)
}

// internal/handlers/handlers.go (REFACTORED)
func (h *Handler) GetMarketOrders(c *fiber.Ctx) error {
    regionID, _ := strconv.Atoi(c.Params("region"))
    typeID, _ := strconv.Atoi(c.Params("type"))
    refresh := c.QueryBool("refresh", false)
    
    // Thin controller - just coordinate
    orders, err := h.marketService.GetOrders(c.Context(), regionID, typeID, refresh)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
    
    return c.JSON(fiber.Map{
        "region_id": regionID,
        "type_id":   typeID,
        "orders":    orders,
        "count":     len(orders),
    })
}
```

**Benefits:**

- ✅ Handler-Tests nur für HTTP-Layer (Parameter, Status Codes)
- ✅ Service-Tests für Business Logic (mockbar)
- ✅ Klare Layer-Trennung

**Aufwand:** ~1 Tag pro Handler-Datei  
**Breaking Changes:** Nein (interne Reorganisation)

---

## 4. Fehlende Test-Utilities

### Problem: Keine Shared Test Infrastructure

**Jeder Test reimplementiert Setup:**

```go
// handlers_integration_test.go
func setupRedisContainer(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
    s := miniredis.RunT(t)
    client := redis.NewClient(&redis.Options{Addr: s.Addr()})
    return s, client
}

// market_integration_test.go (DUPLIKAT)
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
    container := SetupPostgresContainer(t)
    return container.Pool, container.Cleanup
}
```

### Lösungsvorschlag: Test Helpers Package

```go
// internal/testutil/fixtures.go (NEU)
package testutil

func NewMockSDERepo(t *testing.T) *MockSDERepo {
    return &MockSDERepo{
        types: map[int]*TypeInfo{
            34: {ID: 34, Name: "Tritanium"},
        },
    }
}

func NewTestDB(t *testing.T) (*pgxpool.Pool, func()) {
    container := SetupPostgresContainer(t)
    return container.Pool, container.Cleanup
}

func NewTestRedis(t *testing.T) (*redis.Client, func()) {
    s := miniredis.RunT(t)
    client := redis.NewClient(&redis.Options{Addr: s.Addr()})
    return client, s.Close
}

// internal/testutil/mocks.go (NEU)
type MockMarketRepo struct {
    Orders []MarketOrder
    Err    error
}

func (m *MockMarketRepo) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]MarketOrder, error) {
    return m.Orders, m.Err
}
```

**Usage:**

```go
// handlers_test.go
func TestGetMarketOrders(t *testing.T) {
    mockMarket := testutil.NewMockMarketRepo(Orders: testOrders)
    handler := New(nil, nil, mockMarket, nil)
    // ...
}
```

**Aufwand:** ~4 Stunden  
**Impact:** Massiv reduziert Boilerplate in allen Tests

---

## 5. Technische Schuld-Übersicht

### Critical (Blockiert Tests komplett)

| Issue | Severity | Effort | Impact |
|-------|----------|--------|--------|
| Fehlende Interfaces für DI | 🔴 Critical | 2-4h/Layer | Unmöglich Unit Tests zu schreiben |
| Konkrete Typen in Konstruktoren | 🔴 Critical | 2-4h | Mocking nicht möglich |

### High (Erschwert Tests erheblich)

| Issue | Severity | Effort | Impact |
|-------|----------|--------|--------|
| God Object RouteCalculator | 🟠 High | 1-2 Tage | 534 Zeilen untestbar |
| Mixed Concerns in Handlers | 🟠 High | 1 Tag | Handler-Tests brauchen ESI/DB |
| Fehlende Test Utilities | 🟠 High | 4h | Viel Boilerplate |

### Medium (Reduziert Testqualität)

| Issue | Severity | Effort | Impact |
|-------|----------|--------|--------|
| Error Handling inkonsistent | 🟡 Medium | 1 Tag | Tests schwer zu verifizieren |
| Keine Context Propagation Checks | 🟡 Medium | 4h | Timeout-Tests fehlen |
| Hardcoded Constants (MinSpread, MaxRoutes) | 🟡 Medium | 2h | Parameter-Tests unmöglich |

### Low (Best Practice)

| Issue | Severity | Effort | Impact |
|-------|----------|--------|--------|
| Fehlende Table-Driven Tests | 🟢 Low | 2h | Redundanter Test-Code |
| Keine Benchmark Tests | 🟢 Low | 4h | Performance-Regression möglich |

---

## 6. Migrations-Roadmap (Priorisiert)

### Phase 1: Foundation (1 Woche)

**Ziel:** Interfaces einführen ohne Breaking Changes

1. **Database Layer Interfaces** (8h)
   - `SDEQuerier`, `MarketRepository`, `HealthChecker` Interfaces definieren
   - Bestehende Structs implementieren Interfaces
   - Wrapper-Functions für Backward Compatibility

2. **Test Utilities Package** (4h)
   - `internal/testutil` Package erstellen
   - Mock-Implementierungen für alle Interfaces
   - Shared Test Fixtures (Redis, PostgreSQL Containers)

3. **Handler Refactoring - Phase 1** (8h)
   - Konstruktoren auf Interfaces umstellen (mit Wrappers)
   - Erste Handler auf `MarketService` umziehen
   - Unit Tests für einfache Handler (Health, Version, GetType)

**Validierung:** Services Coverage sollte auf **>35%** steigen

### Phase 2: Service Decomposition (1 Woche)

**Ziel:** RouteCalculator aufteilen

4. **Service Extraction** (16h)
   - `MarketFetcher` Service (ESI + Cache)
   - `ProfitAnalyzer` Service (Business Logic)
   - `RoutePlanner` Service (Navigation)
   - `RouteCalculator` wird Orchestrator

5. **Unit Tests für Services** (8h)
   - Mock-basierte Tests für jeden Service
   - Integration Tests behalten (E2E Validation)

**Validierung:** Services Coverage sollte auf **>50%** steigen

### Phase 3: Handler Cleanup (3 Tage)

**Ziel:** Thin Controllers

6. **Business Logic aus Handlers** (12h)
   - `MarketService` vollständig
   - `TradingService` vollständig
   - Handler sind nur noch HTTP-Layer

7. **Handler Unit Tests** (8h)
   - Parameter Validation Tests
   - Status Code Tests
   - Error Handling Tests

**Validierung:** Handlers Coverage sollte auf **>40%** steigen ✅

### Phase 4: Polish (2 Tage)

**Ziel:** Best Practices

8. **Error Handling Standardization** (4h)
9. **Table-Driven Tests Migration** (4h)
10. **Benchmark Tests** (4h)

---

## 7. Konkrete Empfehlungen

### Sofort (Keine Breaking Changes)

1. ✅ **Test Utilities Package erstellen**
   - Aufwand: 4h
   - Impact: Alle neuen Tests profitieren
   - Files: `internal/testutil/fixtures.go`, `internal/testutil/mocks.go`

2. ✅ **MarketService extrahieren**
   - Aufwand: 8h
   - Impact: Handler-Tests werden möglich
   - Breaking: Nein (interne Reorganisation)

### Kurzfristig (1-2 Wochen)

3. ⚠️ **Interfaces für Database Layer**
   - Aufwand: 8h
   - Impact: Unit Tests für Handler möglich
   - Breaking: Ja - aber mit Wrappers migrierbar

4. ⚠️ **RouteCalculator decomposition**
   - Aufwand: 16h
   - Impact: Services werden testbar
   - Breaking: Nein (interne API stabil)

### Langfristig (1-2 Monate)

5. ⚠️ **ESI Client Interface**
   - Aufwand: 16h (betrifft eve-esi-client Repository)
   - Impact: ESI-abhängige Tests mockbar

6. ⚠️ **Navigation/Cargo Package Interfaces**
   - Aufwand: 8h
   - Impact: SDE-abhängige Tests mockbar

---

## 8. Fazit

### Ist die Architektur "schlecht"?

**Nein.** Die Architektur folgt **Layered Architecture** und **Domain-Driven Design** Prinzipien:

- ✅ Klare Layer-Trennung (Handler → Services → Database → External APIs)
- ✅ Domain Models getrennt von Infrastruktur
- ✅ Dependency Injection (Konstruktor-basiert)
- ✅ Context Propagation
- ✅ Error Handling konsistent

### Was ist das Problem?

**Testbarkeit wurde nicht priorisiert:**

- ❌ Keine Interfaces → Konkrete Dependencies
- ❌ Zu große Services → God Objects
- ❌ Mixed Concerns → Handler machen zu viel
- ❌ Keine Test-Infrastruktur → Viel Boilerplate

### Ist es "technische Schuld"?

**Ja, aber bewusst eingegangen:**

- Die Architektur war **Delivery-optimiert** (schnelle Features)
- Testbarkeit wurde **aufgeschoben** (keine Tests = kein Testbarkeits-Druck)
- Jetzt zahlen wir den **Zins** (schwierig Coverage zu erhöhen)

### Sollten wir refactoren?

**Ja, schrittweise:**

**Effort vs. Impact:**

```
High Impact, Low Effort (DO FIRST):
├─ Test Utilities Package ⭐⭐⭐⭐⭐ (4h)
├─ MarketService Extraction ⭐⭐⭐⭐ (8h)
└─ Handler Thin Controllers ⭐⭐⭐⭐ (12h)

High Impact, High Effort (DO NEXT):
├─ Database Layer Interfaces ⭐⭐⭐⭐ (8h + Migration)
└─ RouteCalculator Decomposition ⭐⭐⭐⭐ (16h)

Medium Impact (BACKLOG):
├─ ESI Client Interface ⭐⭐⭐ (16h + Cross-Repo)
└─ Error Handling Standardization ⭐⭐ (4h)
```

**ROI Berechnung:**

- **Investment:** ~3-4 Wochen Refactoring
- **Return:** Handlers 40% Coverage ✅, Services 50%+ Coverage ✅
- **Long-term:** Neue Features sind per Default testbar

---

## 9. Nächste Schritte

### Empfohlene Reihenfolge

1. **Issue erstellen:** "Refactor: Improve testability - Phase 1 (Interfaces)"
2. **ADR schreiben:** "ADR-XXX: Service Layer Interfaces for Testability"
3. **Branch erstellen:** `feat/testability-phase1-interfaces`
4. **Incrementally Refactor:**
   - Tag 1-2: Test Utilities Package
   - Tag 3-4: Database Interfaces
   - Tag 5-6: MarketService Extraction
   - Tag 7-8: Handler Tests
5. **PR Review** mit Coverage-Nachweis

**Coverage-Ziel nach Phase 1:** Services 35%, Handlers 25%

---

## 10. Referenzen

- **Aktueller Code:**
  - `internal/handlers/handlers.go` (274 Zeilen)
  - `internal/services/route_calculator.go` (534 Zeilen)
  - `internal/database/sde.go` (277 Zeilen)

- **Bestehende Tests:**
  - `internal/handlers/handlers_integration_test.go` (Integration, funktioniert)
  - `internal/database/market_integration_test.go` (Integration, funktioniert)
  - `pkg/esi/client_test.go` (Unit, funktioniert mit miniredis)

- **Coverage Reports:**
  - ESI: 35.2% ✅
  - Services: 29.8% (0.2% bis Ziel)
  - Handlers: 17.1% (22.9% bis Ziel)

**Fazit:** Die Architektur ist **funktional solide**, aber **nicht test-freundlich**. Refactoring ist **lohnenswert** und **schrittweise machbar**.
