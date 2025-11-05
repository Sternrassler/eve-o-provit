# Handler Refactoring Examples

Dieser Ordner enthält Code-Beispiele für die vorgeschlagene Handler-Architektur-Refaktorierung.

## Dateien

### 1. `01_service_interfaces.go`
**Zweck:** Definiert Interfaces für alle Services  
**Vorteil:** Dependency Injection, einfaches Mocking  
**Migration:** Phase 1 - Low Risk

Interfaces:
- `CharacterService` (statt `*CharacterHelper`)
- `NavigationService` (SDE-Queries gekapselt)
- `TradingServiceInterface` (statt `*TradingService`)
- `InventorySellOrchestrator` (Facade für komplexe Workflows)
- `RouteCalculationService` (statt `*RouteCalculator`)

### 2. `02_orchestrator_implementation.go`
**Zweck:** Implementiert Facade-Service für Inventory-Sell Use Case  
**Vorteil:** Business-Logik aus Handler extrahiert  
**Migration:** Phase 2 - Medium Risk

Features:
- Multi-Service Orchestration (Character, Navigation, Trading)
- Business Error Handling (`ErrNotDocked`, `ErrLocationNotFound`)
- Fallback-Logik (Tax Rate)

### 3. `03_refactored_handler.go`
**Zweck:** Zeigt "Thin Handler" Pattern  
**Vorteil:** Handler = nur HTTP Mapping (25 LOC statt 80)  
**Migration:** Phase 2-3

Änderungen:
- Single Orchestrator Dependency (statt 4+ Services)
- Request Validation delegiert an Model
- Business Error Mapping zentralisiert

### 4. `04_request_validation.go`
**Zweck:** Validierungs-Logik in Request Models  
**Vorteil:** Wiederverwendbar, testbar ohne Handler  
**Migration:** Phase 3 - Low Risk

Features:
- `Validate()` Methods für Request-Structs
- Helper Functions (`ValidatePositiveInt`, `ValidateEnum`)
- Konsistente Fehlermeldungen

### 5. `05_simplified_test.go`
**Zweck:** Handler Unit Tests mit Orchestrator-Mocks  
**Vorteil:** 70% weniger Code, 75% weniger Mocks  
**Migration:** Phase 4

Vergleich:
- **Before:** 50+ Zeilen Setup, 4 Mocks
- **After:** 15 Zeilen Setup, 1 Mock

### 6. `06_orchestrator_unit_test.go`
**Zweck:** Business-Logik Tests auf Orchestrator-Ebene  
**Vorteil:** Keine HTTP-Dependencies, schnelle Tests

Test Cases:
- Success Path
- Not Docked Error
- Tax Rate Fallback
- Location Not Found Error

## Migration Path

### Phase 1: Interfaces (1-2 Tage)
1. Erstelle `internal/services/interfaces.go`
2. Definiere Interfaces für existierende Services
3. Verifiziere Implementierung mit `var _ Interface = (*ConcreteType)(nil)`
4. Update Handler-Konstruktoren: Akzeptiere Interfaces

**Risiko:** ✅ Niedrig (nur Typ-Änderungen, keine Logik)

### Phase 2: Orchestrator (2-3 Tage)
1. Erstelle `internal/services/inventory_sell_orchestrator.go`
2. Extrahiere Business-Logik aus Handler
3. Schreibe Orchestrator Unit Tests
4. Update Handler: Nutze Orchestrator

**Risiko:** ⚠️ Mittel (Logik-Migration)

### Phase 3: Handler Cleanup (1 Tag)
1. Füge `Validate()` zu Request Models hinzu
2. Zentralisiere Error Handling (`handleBusinessError`)
3. Entferne verschachtelte Handler-Zugriffe

**Risiko:** ✅ Niedrig (Refactoring, keine Feature-Änderung)

### Phase 4: Test Migration (1-2 Tage)
1. Erstelle Mock Orchestrators
2. Refactore Handler Tests: Nutze Orchestrator-Mocks
3. Schreibe Orchestrator Unit Tests
4. Measure Coverage

**Risiko:** ✅ Niedrig (nur Tests)

## Vorteile Zusammenfassung

| Metrik | Before | After | Verbesserung |
|--------|--------|-------|--------------|
| Handler LOC | 80 | 25 | **-69%** |
| Test Setup LOC | 50+ | 15 | **-70%** |
| Mocks pro Test | 4 | 1 | **-75%** |
| Testbarkeit | Schwer | Einfach | **✅** |
| Business Logic Location | Handler | Service | **✅ Proper Layering** |

## Nächste Schritte

1. **Review** dieser Beispiele
2. **Entscheidung:** Full Migration oder Proof-of-Concept?
3. **Start:** Phase 1 (Interfaces) in einem separaten Branch
4. **Validierung:** Bestehende Tests bleiben grün
5. **Iteration:** Phase 2-4 schrittweise

## Fragen?

Siehe Hauptdokument: `../handler-architecture-analysis.md`
