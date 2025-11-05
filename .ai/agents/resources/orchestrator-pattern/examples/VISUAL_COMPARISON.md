# Handler Architecture: Before vs. After

## BEFORE: Complex Handler Architecture

```
┌────────────────────────────────────────────────────────────────┐
│  TradingHandler (80 LOC)                                      │
│                                                                │
│  ❌ HTTP Parsing (10 LOC)                                     │
│  ❌ Validation (20 LOC)                                       │
│  ❌ Business Logic (40 LOC)                                   │
│     - Get Character Location                                  │
│     - Validate Docked State                                   │
│     - Resolve System ID                                       │
│     - Calculate Tax Rate                                      │
│     - Calculate Routes                                        │
│  ❌ HTTP Response (10 LOC)                                    │
└────────────┬───────────────────────────────────────────────────┘
             │
             │ Depends on 4+ Concrete Services
             ▼
┌─────────────────────────────────────────────────────────────────┐
│  *CharacterHelper    (Concrete)                                │
│  *SDEQuerier         (Interface, but nested via *Handler)      │
│  *TradingService     (Concrete)                                │
│  *Handler            (Nested base handler with DB access)      │
└─────────────────────────────────────────────────────────────────┘

Testing Pain Points:
❌ 4 Mocks benötigt (CharacterHelper, SDEQuerier, TradingService, Handler)
❌ 50+ Zeilen Test-Setup
❌ Business Logic in Handler = schwer testbar
❌ Änderungen in Services → Tests brechen
```

---

## AFTER: Thin Handler + Orchestrator Architecture

```
┌────────────────────────────────────────────────────────────────┐
│  TradingHandler (25 LOC)                                      │
│                                                                │
│  ✅ HTTP Parsing (5 LOC)                                      │
│  ✅ Validation (delegated to model.Validate())                │
│  ✅ Single Service Call (5 LOC)                               │
│  ✅ HTTP Response (5 LOC)                                     │
└────────────┬───────────────────────────────────────────────────┘
             │
             │ Single Interface Dependency
             ▼
┌─────────────────────────────────────────────────────────────────┐
│  InventorySellOrchestrator (Interface)                         │
│                                                                 │
│  ├─ CalculateSellRoutes(req, charID, token) → []Routes        │
│                                                                 │
│  Implementation:                                                │
│  ├─ Step 1: Get Character Location (CharacterService)         │
│  ├─ Step 2: Validate Docked State                             │
│  ├─ Step 3: Resolve System ID (NavigationService)             │
│  ├─ Step 4: Calculate Tax Rate (CharacterService)             │
│  └─ Step 5: Calculate Routes (TradingService)                 │
└────────────┬────────────────────────────────────────────────────┘
             │
             │ Delegates to Domain Services (Interfaces)
             ▼
┌─────────────────────────────────────────────────────────────────┐
│  CharacterService (Interface)                                  │
│  NavigationService (Interface)                                 │
│  TradingServiceInterface (Interface)                           │
└─────────────────────────────────────────────────────────────────┘

Testing Benefits:
✅ 1 Mock benötigt (InventorySellOrchestrator)
✅ 15 Zeilen Test-Setup
✅ Business Logic in Orchestrator = einfach testbar
✅ Änderungen in Services → Handler-Tests bleiben grün
```

---

## Code Comparison

### Before: Complex Handler

```go
func (h *TradingHandler) CalculateInventorySellRoutes(c *fiber.Ctx) error {
    // Parse request
    var req models.InventorySellRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
    }

    // Validate parameters (20 lines)
    if req.TypeID <= 0 {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid type_id"})
    }
    if req.Quantity <= 0 {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid quantity"})
    }
    // ... more validation

    // Business logic (40 lines) ❌ SHOULD BE IN SERVICE
    location, err := h.characterHelper.GetCharacterLocation(ctx, characterID, accessToken)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to fetch location"})
    }

    if location.StationID == nil {
        return c.Status(400).JSON(fiber.Map{"error": "Character must be docked"})
    }

    startSystemID, err := h.handler.sdeQuerier.GetSystemIDForLocation(ctx, *location.StationID)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to determine system"})
    }

    taxRate, err := h.characterHelper.CalculateTaxRate(ctx, characterID, accessToken)
    if err != nil {
        taxRate = 0.055
    }

    routes, err := h.tradingService.CalculateInventorySellRoutes(ctx, req, startSystemID, taxRate)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to calculate routes"})
    }

    return c.JSON(fiber.Map{"routes": routes, "count": len(routes)})
}

// Total: ~80 lines
// Business Logic: 70% of handler
// Dependencies: 4 services
```

### After: Thin Handler

```go
func (h *TradingHandler) CalculateInventorySellRoutes(c *fiber.Ctx) error {
    // 1. Parse Request
    var req models.InventorySellRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
    }

    // 2. Validate Request (delegated)
    if err := req.Validate(); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": err.Error()})
    }

    // 3. Extract Auth Context
    characterID := c.Locals("character_id").(int)
    accessToken := c.Locals("access_token").(string)

    // 4. Delegate to Orchestrator (single call) ✅
    routes, err := h.inventorySellOrchestrator.CalculateSellRoutes(
        c.Context(), req, characterID, accessToken,
    )
    if err != nil {
        return handleBusinessError(c, err)
    }

    // 5. Return Response
    return c.JSON(fiber.Map{"routes": routes, "count": len(routes)})
}

// Total: ~25 lines
// Business Logic: 0% (delegated to service)
// Dependencies: 1 orchestrator
```

---

## Test Comparison

### Before: Complex Test Setup

```go
func TestCalculateInventorySellRoutes(t *testing.T) {
    // Mock 1: CharacterHelper
    mockCharHelper := &MockCharacterHelper{
        GetCharacterLocationFunc: func(...) (*CharacterLocation, error) {
            return &CharacterLocation{StationID: ptr(60003760)}, nil
        },
        CalculateTaxRateFunc: func(...) (float64, error) {
            return 0.03, nil
        },
    }

    // Mock 2: SDEQuerier
    mockSDEQuerier := &MockSDEQuerier{
        GetSystemIDForLocationFunc: func(...) (int64, error) {
            return 30000142, nil
        },
    }

    // Mock 3: TradingService
    mockTradingService := &MockTradingService{
        CalculateInventorySellRoutesFunc: func(...) ([]Route, error) {
            return []Route{{ProfitPerUnit: 100.0}}, nil
        },
    }

    // Mock 4: Base Handler
    baseHandler := &Handler{sdeQuerier: mockSDEQuerier}

    // Construct handler with all dependencies
    handler := &TradingHandler{
        characterHelper: mockCharHelper,
        tradingService:  mockTradingService,
        handler:         baseHandler,
    }

    // Setup Fiber app
    app := fiber.New()
    app.Use(mockAuthMiddleware)
    app.Post("/inventory-sell", handler.CalculateInventorySellRoutes)

    // Create request
    reqBody := models.InventorySellRequest{...}
    bodyBytes, _ := json.Marshal(reqBody)
    req := httptest.NewRequest("POST", "/inventory-sell", bytes.NewReader(bodyBytes))

    // Execute test
    resp, err := app.Test(req)
    // ... assertions

    // Total: 50+ lines of setup
}
```

### After: Simple Test Setup

```go
func TestCalculateInventorySellRoutes(t *testing.T) {
    // Single Mock: Orchestrator
    mockOrch := &MockInventorySellOrchestrator{
        CalculateSellRoutesFunc: func(ctx, req, charID, token) ([]Route, error) {
            return []Route{{ProfitPerUnit: 100.0}}, nil
        },
    }

    handler := NewTradingHandler(mockOrch, nil)

    app := fiber.New()
    app.Use(mockAuthMiddleware)
    app.Post("/inventory-sell", handler.CalculateInventorySellRoutes)

    reqBody := models.InventorySellRequest{...}
    resp := sendRequest(app, reqBody)

    assert.Equal(t, 200, resp.StatusCode)
    assert.True(t, mockOrch.WasCalled)

    // Total: 15 lines of setup
}
```

---

## Benefits Matrix

| Aspect | Before | After | Improvement |
|--------|--------|-------|-------------|
| **Handler LOC** | 80 | 25 | **-69%** |
| **Business Logic in Handler** | 70% | 0% | **✅ Moved to Service** |
| **Service Dependencies** | 4 | 1 | **-75%** |
| **Mocks per Test** | 4 | 1 | **-75%** |
| **Test Setup LOC** | 50+ | 15 | **-70%** |
| **Testability** | Schwer | Einfach | **✅ Unit-testbar** |
| **Reusability** | HTTP-only | Any context | **✅ CLI, Jobs, etc.** |
| **Maintainability** | Brittle | Robust | **✅ Loose coupling** |

---

## Migration Strategy

```
Phase 1: Interface Extraction (1-2 days)
├─ Define service interfaces
├─ Update handler constructors
└─ Verify compile-time checks
   Risk: ✅ Low

Phase 2: Orchestrator Creation (2-3 days)
├─ Create InventorySellOrchestrator
├─ Move business logic from handler
├─ Write orchestrator unit tests
└─ Update handler to use orchestrator
   Risk: ⚠️ Medium

Phase 3: Handler Cleanup (1 day)
├─ Add Validate() to request models
├─ Centralize error handling
└─ Remove nested handler access
   Risk: ✅ Low

Phase 4: Test Migration (1-2 days)
├─ Create mock orchestrators
├─ Refactor handler tests
├─ Add orchestrator unit tests
└─ Measure coverage
   Risk: ✅ Low

Total Time: 5-8 days (over 2 weeks)
Total Risk: ⚠️ Medium (controlled, incremental)
```

---

## Recommended Next Steps

1. ✅ **Review** dieses Dokument + Code-Beispiele
2. ✅ **Entscheidung:** Full Migration oder Proof-of-Concept?
3. ✅ **Start:** Phase 1 (Interfaces) in Feature-Branch
4. ✅ **Validate:** Bestehende Tests bleiben grün
5. ✅ **Iterate:** Phase 2-4 schrittweise mit Code Reviews

---

## Files Created

- `/tmp/handler-architecture-analysis.md` - Detaillierte Analyse
- `/tmp/refactoring-examples/01_service_interfaces.go` - Interface-Definitionen
- `/tmp/refactoring-examples/02_orchestrator_implementation.go` - Orchestrator-Implementierung
- `/tmp/refactoring-examples/03_refactored_handler.go` - Thin Handler Beispiel
- `/tmp/refactoring-examples/04_request_validation.go` - Validierungs-Extraktion
- `/tmp/refactoring-examples/05_simplified_test.go` - Handler-Test Vereinfachung
- `/tmp/refactoring-examples/06_orchestrator_unit_test.go` - Orchestrator-Tests
- `/tmp/refactoring-examples/README.md` - Übersicht der Beispiele
