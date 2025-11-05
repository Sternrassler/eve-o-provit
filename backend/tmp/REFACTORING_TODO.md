# Testability Refactoring Roadmap - TODO Tracker

**Quelle:** `docs/ARCHITECTURE_REVIEW_TESTABILITY.md`  
**Start:** 5. November 2025  
**Ziel:** Services 35%, Handlers 40% Coverage

---

## Phase 1: Foundation (1 Woche) - IN PROGRESS

### ✅ Completed

- [x] Architecture Review erstellt
- [x] Handler validation tests (17.1% Coverage)
- [x] Services cache tests (29.8% Coverage)

### 🔄 In Progress

- [x] **Task 1.1: Database Layer Interfaces** (8h) - ✅ COMPLETE
  - [x] `internal/database/interfaces.go` erstellen
  - [x] `HealthChecker` Interface
  - [x] `SDEQuerier` Interface  
  - [x] `MarketQuerier` Interface
  - [x] Bestehende Structs implementieren Interfaces
  - [x] Compile-Time Checks (var _ Interface = (*Struct)(nil))

- [x] **Task 1.2: Test Utilities Package** (4h) - ✅ COMPLETE
  - [x] `internal/testutil/fixtures.go` erstellt (7 fixture functions)
  - [x] `internal/testutil/mocks.go` erstellt (3 mock implementations)
  - [x] MockSDEQuerier mit allen Methoden
  - [x] MockMarketQuerier mit stateful storage
  - [x] MockHealthChecker mit error variant
  - [x] Helper functions: NewMockSDEWithDefaults, NewMockMarketWithDefaults
  - [x] Alle Tests bestehen (7/7 passed)

- [ ] **Task 1.3: Handler Refactoring - Phase 1** (8h) - NEXT
  - [ ] Handler Konstruktor auf Interfaces umstellen
  - [ ] Wrapper-Funktionen für alte API
  - [ ] MarketService extrahieren
  - [ ] Unit Tests für Health endpoint
  - [ ] Unit Tests für GetType endpoint
  - [ ] Unit Tests für GetRegions endpoint

**Validierung:** Services Coverage >35%, Handlers Coverage >25%

---

## Phase 2: Service Decomposition (1 Woche) - PENDING

### Task 2.1: Service Extraction (16h)

- [ ] `services/market_fetcher.go` erstellen (ESI + Cache)
- [ ] `services/profit_analyzer.go` erstellen (Business Logic)
- [ ] `services/route_planner.go` erstellen (Navigation)
- [ ] `RouteCalculator` zu Orchestrator refactoren
- [ ] Interfaces für neue Services

### Task 2.2: Unit Tests für Services (8h)

- [ ] MarketFetcher Mock-Tests
- [ ] ProfitAnalyzer Mock-Tests
- [ ] RoutePlanner Mock-Tests
- [ ] RouteCalculator Orchestration-Tests
- [ ] Integration Tests anpassen

**Validierung:** Services Coverage >50%

---

## Phase 3: Handler Cleanup (3 Tage) - PENDING

### Task 3.1: Business Logic aus Handlers (12h)

- [ ] `MarketService` vollständig implementieren
- [ ] `TradingService` vollständig implementieren
- [ ] `GetMarketOrders` Handler refactoren
- [ ] `CalculateRoutes` Handler refactoren
- [ ] `SetAutopilotWaypoint` Handler refactoren

### Task 3.2: Handler Unit Tests (8h)

- [ ] Parameter Validation Tests (vollständig)
- [ ] Status Code Tests (alle Endpoints)
- [ ] Error Handling Tests (alle Endpoints)
- [ ] Mock-basierte Service Tests

**Validierung:** Handlers Coverage >40% ✅

---

## Phase 4: Polish (2 Tage) - PENDING

### Task 4.1: Error Handling Standardization (4h)

- [ ] Error Types definieren
- [ ] Konsistente Error Responses
- [ ] Error Logging standardisieren

### Task 4.2: Table-Driven Tests Migration (4h)

- [ ] Bestehende Tests konvertieren
- [ ] Test Coverage Report

### Task 4.3: Benchmark Tests (4h)

- [ ] RouteCalculator Benchmarks
- [ ] MarketFetcher Benchmarks
- [ ] Handler Benchmarks

---

## Metrics Tracking

| Phase | Start Coverage | Target | Actual | Status |
|-------|---------------|--------|--------|--------|
| **Baseline** | Services: 29.8%, Handlers: 17.1% | - | - | ✅ |
| **Phase 1** | - | Services: 35%, Handlers: 25% | - | 🔄 |
| **Phase 2** | - | Services: 50%, Handlers: 30% | - | ⏳ |
| **Phase 3** | - | Services: 50%, Handlers: 40% | - | ⏳ |
| **Phase 4** | - | Polish + Benchmarks | - | ⏳ |

---

## Notes

- Breaking Changes nur mit Migration-Path
- Alle Änderungen mit Tests validieren
- Coverage nach jeder Phase messen
- Integration Tests parallel behalten
