# Handler Refactoring Quick Reference

## Problem Statement

**Current State:**
- Handlers contain 70% business logic, 30% HTTP mapping
- Tests require 4+ service mocks (50+ lines setup)
- Difficult to test handlers in isolation
- Business logic not reusable outside HTTP context

**Goal:**
- Handlers = 100% HTTP mapping (thin handlers)
- Tests require 1 orchestrator mock (15 lines setup)
- Easy to test handlers with simple mocks
- Business logic reusable in any context

---

## Architectural Patterns

### ❌ Anti-Pattern: Fat Handler

```go
type TradingHandler struct {
    characterHelper *services.CharacterHelper      // Concrete
    sdeQuerier      database.SDEQuerier            // Interface
    tradingService  *services.TradingService       // Concrete
    handler         *Handler                       // Nested concrete
}

func (h *TradingHandler) Handle(c *fiber.Ctx) error {
    // Parse request
    // Validate request
    // Call service 1 ❌
    // Call service 2 ❌
    // Call service 3 ❌
    // Call service 4 ❌
    // Return response
}
```

**Problems:**
- 4 dependencies (hard to mock)
- Business logic in handler (not reusable)
- Test complexity: O(n) where n = number of dependencies

---

### ✅ Recommended: Thin Handler + Orchestrator

```go
type TradingHandler struct {
    inventorySellOrchestrator services.InventorySellOrchestrator  // Interface
}

func (h *TradingHandler) Handle(c *fiber.Ctx) error {
    // Parse request
    // Validate request (delegated to model)
    routes, err := h.orchestrator.CalculateSellRoutes(...) // Single call ✅
    // Handle error
    // Return response
}
```

**Benefits:**
- 1 dependency (easy to mock)
- Business logic in service (reusable)
- Test complexity: O(1)

---

## Interface Design

### Service Interfaces (Create These)

```go
// Character operations
type CharacterService interface {
    GetLocation(ctx, charID, token) (*CharacterLocation, error)
    CalculateTaxRate(ctx, charID, token) (float64, error)
    GetSkills(ctx, charID, token) (*CharacterSkills, error)
}

// Navigation operations
type NavigationService interface {
    GetSystemIDForLocation(ctx, locationID) (int64, error)
    GetSystemName(ctx, systemID) (string, error)
    CalculateRoute(ctx, from, to, avoidLowsec) (*RouteResult, error)
}

// Trading operations
type TradingServiceInterface interface {
    CalculateInventorySellRoutes(ctx, req, systemID, taxRate) ([]Route, error)
}

// Orchestrator (Facade)
type InventorySellOrchestrator interface {
    CalculateSellRoutes(ctx, req, charID, token) ([]Route, error)
}
```

---

## Request Validation Pattern

### ❌ Before: Validation in Handler

```go
func (h *Handler) Handle(c *fiber.Ctx) error {
    var req Request
    c.BodyParser(&req)
    
    if req.TypeID <= 0 {
        return c.Status(400).JSON(...)
    }
    if req.Quantity <= 0 {
        return c.Status(400).JSON(...)
    }
    // ... more validation (20 lines)
}
```

### ✅ After: Validation in Model

```go
// In model
func (r *Request) Validate() error {
    if r.TypeID <= 0 {
        return errors.New("invalid type_id")
    }
    if r.Quantity <= 0 {
        return errors.New("invalid quantity")
    }
    return nil
}

// In handler (clean)
func (h *Handler) Handle(c *fiber.Ctx) error {
    var req Request
    c.BodyParser(&req)
    
    if err := req.Validate(); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    }
    // ... business logic
}
```

---

## Error Handling Pattern

### ❌ Before: Scattered Error Mapping

```go
if err := service.DoSomething(); err != nil {
    if err.Error() == "not found" {
        return c.Status(404).JSON(...)
    }
    return c.Status(500).JSON(...)
}
```

### ✅ After: Centralized Error Mapping

```go
// Define business errors
var (
    ErrNotDocked = errors.New("character not docked")
    ErrLocationNotFound = errors.New("location not found")
)

// Error mapping helper
func handleBusinessError(c *fiber.Ctx, err error) error {
    switch {
    case errors.Is(err, ErrNotDocked):
        return c.Status(400).JSON(fiber.Map{"error": "Character must be docked"})
    case errors.Is(err, ErrLocationNotFound):
        return c.Status(500).JSON(fiber.Map{"error": "Failed to determine location"})
    default:
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }
}

// In handler (clean)
if err := service.DoSomething(); err != nil {
    return handleBusinessError(c, err)
}
```

---

## Testing Patterns

### ❌ Before: Complex Mock Setup

```go
func TestHandler(t *testing.T) {
    // Mock 1: CharacterHelper
    mockChar := &MockChar{...}
    
    // Mock 2: SDEQuerier
    mockSDE := &MockSDE{...}
    
    // Mock 3: TradingService
    mockTrading := &MockTrading{...}
    
    // Mock 4: Base Handler
    baseHandler := &Handler{sdeQuerier: mockSDE}
    
    handler := &TradingHandler{
        characterHelper: mockChar,
        tradingService: mockTrading,
        handler: baseHandler,
    }
    
    // ... 50+ lines total
}
```

### ✅ After: Simple Mock Setup

```go
func TestHandler(t *testing.T) {
    // Single mock
    mockOrch := &MockOrchestrator{
        CalculateSellRoutesFunc: func(...) ([]Route, error) {
            return []Route{{ProfitPerUnit: 100}}, nil
        },
    }
    
    handler := NewTradingHandler(mockOrch, nil)
    
    // ... 15 lines total
}
```

---

## Migration Checklist

### Phase 1: Interface Extraction ✅
- [ ] Create `internal/services/interfaces.go`
- [ ] Define `CharacterService` interface
- [ ] Define `NavigationService` interface
- [ ] Define `TradingServiceInterface` interface
- [ ] Define `InventorySellOrchestrator` interface
- [ ] Add compile-time checks: `var _ Interface = (*Concrete)(nil)`
- [ ] Update handler constructors to accept interfaces
- [ ] Verify tests still pass

### Phase 2: Orchestrator Creation ⚠️
- [ ] Create `internal/services/inventory_sell_orchestrator.go`
- [ ] Implement `InventorySellOrchestrator` interface
- [ ] Move business logic from handler to orchestrator
- [ ] Define business errors (`ErrNotDocked`, etc.)
- [ ] Write orchestrator unit tests
- [ ] Update handler to use orchestrator
- [ ] Verify integration tests still pass

### Phase 3: Handler Cleanup ✅
- [ ] Add `Validate()` method to `InventorySellRequest`
- [ ] Add `Validate()` method to `RouteCalculationRequest`
- [ ] Create `handleBusinessError()` helper
- [ ] Remove nested handler access (h.handler.sdeQuerier → h.navService)
- [ ] Document handler responsibilities
- [ ] Verify handler LOC reduced by ~70%

### Phase 4: Test Migration ✅
- [ ] Create mock orchestrators in test files
- [ ] Refactor existing handler tests to use orchestrator mocks
- [ ] Write new orchestrator unit tests
- [ ] Remove obsolete multi-service mocks
- [ ] Measure test coverage (target: >80%)
- [ ] Verify test setup reduced by ~70%

---

## Code Metrics

### Success Criteria

| Metric | Before | After | Target |
|--------|--------|-------|--------|
| Handler LOC | 80 | ? | **≤ 30** |
| Business Logic % | 70% | ? | **0%** |
| Dependencies | 4 | ? | **≤ 2** |
| Mocks per Test | 4 | ? | **1** |
| Test Setup LOC | 50+ | ? | **≤ 20** |

### How to Measure

```bash
# Handler complexity
wc -l internal/handlers/trading.go

# Test setup lines
grep -A 50 "func Test" internal/handlers/trading_test.go | head -50 | wc -l

# Number of mocks
grep "Mock.*:=" internal/handlers/trading_test.go | wc -l

# Test coverage
go test -cover ./internal/handlers/...
```

---

## Common Pitfalls

### ❌ Don't: Create Too Many Orchestrators

```go
// Bad: One orchestrator per endpoint
type GetLocationOrchestrator interface {}
type CalculateTaxOrchestrator interface {}
type GetSystemIDOrchestrator interface {}
```

**Instead:** One orchestrator per **use case** (which may span multiple endpoints)

```go
// Good: One orchestrator for complex workflow
type InventorySellOrchestrator interface {
    CalculateSellRoutes(...) ([]Route, error)
}
```

---

### ❌ Don't: Put HTTP Logic in Orchestrator

```go
// Bad: HTTP dependencies in service
func (o *Orchestrator) DoSomething(c *fiber.Ctx) error {
    c.JSON(...) // ❌ Service shouldn't know about Fiber
}
```

**Instead:** Return domain errors, let handler map to HTTP

```go
// Good: Domain errors
func (o *Orchestrator) DoSomething(...) error {
    if condition {
        return ErrNotDocked // Let handler decide HTTP code
    }
}
```

---

### ❌ Don't: Mix Orchestrator with Repository Logic

```go
// Bad: SQL queries in orchestrator
func (o *Orchestrator) DoSomething() error {
    rows, err := db.Query("SELECT ...") // ❌
}
```

**Instead:** Delegate to repositories/services

```go
// Good: Use service interfaces
func (o *Orchestrator) DoSomething() error {
    data, err := o.tradingService.GetData(...) // ✅
}
```

---

## Quick Decision Tree

**When to create an Orchestrator?**

```
Is the handler calling 3+ services?
├─ YES → Create Orchestrator ✅
└─ NO  → Use direct service call

Is the workflow complex (5+ steps)?
├─ YES → Create Orchestrator ✅
└─ NO  → Use direct service call

Is the business logic reused in multiple contexts?
├─ YES → Create Orchestrator ✅
└─ NO  → Consider if extraction makes sense
```

**Examples:**

| Use Case | Services Needed | Orchestrator? |
|----------|-----------------|---------------|
| Get Type Info | 1 (SDEQuerier) | ❌ NO (simple) |
| Get Market Orders | 1 (MarketService) | ❌ NO (simple) |
| Calculate Routes | 1 (RouteCalculator) | ❌ NO (simple) |
| Inventory Sell | 3+ (Character, Navigation, Trading) | ✅ YES (complex) |

---

## Resources

- **Main Analysis:** `/tmp/handler-architecture-analysis.md`
- **Code Examples:** `/tmp/refactoring-examples/`
- **Visual Comparison:** `/tmp/refactoring-examples/VISUAL_COMPARISON.md`

---

## Questions?

**Q: Do I need to refactor all handlers at once?**  
A: No. Start with the most complex handler (e.g., InventorySell), prove the pattern, then iterate.

**Q: What if a service only has 1 method?**  
A: Still use an interface for testability. Single-method interfaces are common (e.g., `io.Reader`).

**Q: Should orchestrators call other orchestrators?**  
A: Generally no. Orchestrators should coordinate domain services, not other orchestrators.

**Q: How do I handle validation that needs database access?**  
A: Validation that requires DB = business rule, not input validation. Put it in the service/orchestrator.

---

**Last Updated:** 2025-11-05  
**Status:** Ready for Review
