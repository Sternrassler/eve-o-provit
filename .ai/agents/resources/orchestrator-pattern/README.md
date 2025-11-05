# Orchestrator Pattern Resources

This directory contains reference materials for applying the **Orchestrator Pattern** to refactor fat handlers into thin handlers with dedicated orchestrator services.

## Pattern Overview

**Problem:** Handlers contain too much business logic (70%+) and have too many service dependencies (4+), making them difficult to test.

**Solution:** Extract business logic into a dedicated Orchestrator service that coordinates multiple domain services. Handler becomes thin HTTP mapper (0% business logic).

**Benefits:**
- **Testability:** 1 mock instead of 4+ mocks (-75%)
- **Test Setup:** 15 lines instead of 50+ lines (-70%)
- **Handler LOC:** Reduced by ~50% (e.g., 80 → 40 lines)
- **Reusability:** Business logic usable outside HTTP context
- **Maintainability:** Clear separation of HTTP vs. domain logic

## Quick Start

**Load the agent:**
```
@workspace load .ai/agents/orchestrator-pattern-refactorer.md
```

**Request format:**
```
"Handler XYZ has 4 service dependencies and 80 lines of business logic - can you apply the orchestrator pattern?"
```

## Directory Structure

```
orchestrator-pattern/
├── README.md                          # This file
└── examples/                          # Real implementation examples
    ├── QUICK_REFERENCE.md             # Quick lookup guide
    ├── VISUAL_COMPARISON.md           # Before/After visual comparison
    ├── 01_service_interfaces.go       # Phase 1: Interface extraction
    ├── 02_orchestrator_implementation.go  # Phase 2: Orchestrator creation
    ├── 03_refactored_handler.go       # Phase 3: Handler simplification
    ├── 04_request_validation.go       # Validation pattern
    ├── 05_simplified_test.go          # Handler test with single mock
    └── 06_orchestrator_unit_test.go   # Orchestrator unit test
```

## When to Apply

### ✅ Good Candidates

- Handler calls **3+ services** sequentially
- Handler contains **50%+ business logic**
- Tests require **4+ mocks**
- Workflow has **5+ steps**
- Business logic **reusable** in other contexts

**Example:** `CalculateInventorySellRoutes`
- Services: CharacterService, NavigationService, TradingService
- Steps: Get location → Validate docked → Resolve system → Calculate tax → Calculate routes
- Result: 80 LOC → 40 LOC handler, 1 mock instead of 4

### ❌ Bad Candidates

- Handler calls **1-2 simple services**
- Handler is **CRUD-only**
- Total handler **< 40 lines**
- No **reusability** requirement

**Example:** `GetTypeInfo` (single SDE query)

## Refactoring Process (4 Phases)

### Phase 1: Interface Extraction (1-2h)
- Create `internal/services/interfaces.go`
- Define service contracts (CharacterServicer, NavigationServicer, etc.)
- Add compile-time checks

### Phase 2: Orchestrator Implementation (2-3h)
- Create `internal/services/<use_case>_orchestrator.go`
- Move business logic from handler to orchestrator
- Define BusinessError type for domain errors

### Phase 3: Handler Simplification (1-2h)
- Add Validate() to request models
- Update handler constructor to create orchestrator
- Reduce handler to HTTP-only concerns

### Phase 4: Test Simplification (2-3h)
- Create orchestrator mock
- Write simplified handler tests (1 mock)
- Write comprehensive orchestrator unit tests

## Expected Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Handler LOC | 80 | 40 | **-50%** |
| Business Logic % | 70% | 0% | **-100%** |
| Dependencies | 4 | 1 | **-75%** |
| Test Setup LOC | 50+ | 15 | **-70%** |
| Mocks per Test | 4 | 1 | **-75%** |
| Test Coverage | +2-5% | +2-5% | Improved |

## Real Implementation Example

**From:** CalculateInventorySellRoutes refactoring (2025-11-05)

**Commits:**
- `44d28eb` - Orchestrator Pattern implementation (Phases 1-3)
- `a338dcc` - Orchestrator unit tests (Phase 4)

**Results:**
- Handler: 80 LOC → 40 LOC (-50%)
- Tests: 50+ lines setup → 15 lines (-70%)
- Mocks: 4 → 1 (-75%)
- Coverage: 26.7% → 28.5% (+1.8%)
- Tests: 0 → 10 (4 handler + 6 orchestrator), all passing ✅

## Code Examples

See `examples/` directory for:
- Complete interface definitions
- Full orchestrator implementation
- Refactored handler code
- Simplified test examples
- Orchestrator unit tests

## Related Agents

- **code-refactor-master** - File organization refactoring
- **test-implementer** - Comprehensive test coverage
- **code-architecture-reviewer** - Architecture validation

## Usage Pattern

1. **Identify** fat handler (3+ services, 50%+ business logic)
2. **Load agent:** `@workspace load .ai/agents/orchestrator-pattern-refactorer.md`
3. **Request:** "Apply orchestrator pattern to XYZ handler"
4. **Review** plan and metrics
5. **Execute** 4 phases
6. **Validate** tests pass, coverage improved

## References

- **Quick Reference:** `examples/QUICK_REFERENCE.md`
- **Visual Comparison:** `examples/VISUAL_COMPARISON.md`
- **Agent Definition:** `../.ai/agents/orchestrator-pattern-refactorer.md`

---

**Last Updated:** 2025-11-05  
**Status:** Production-ready (validated on CalculateInventorySellRoutes)  
**ROI:** ~6h investment, break-even at 2-3 handlers
