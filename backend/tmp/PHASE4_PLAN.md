# Phase 4: Polish & Documentation - Plan

**Status:** STARTING  
**Datum:** 2025-11-05  
**Baseline Coverage:**

- pkg/esi: 35.2% ✅ (Ziel: 35%)
- internal/services: 29.1% 🟡 (Ziel: 30%, bei 97%)
- internal/handlers: **44.0% ✅** (Ziel: 40%, erreicht!)

---

## Ziele Phase 4

### Primärziel: Code-Qualität & Wartbarkeit

1. Error Handling vereinheitlichen
2. Test-Dokumentation verbessern
3. Code Cleanup (tote Pfade entfernen)
4. Performance Baseline etablieren

### Sekundärziel: Coverage Gaps schließen

- Services: 29.1% → 30%+ (nur 0.9pp fehlen)
- Optional: Weitere Handler-Tests für 45%+

---

## Task-Übersicht (Priorität)

### 🔥 HIGH: Quick Wins (2-3h)

**Task 4.1: Services Coverage auf 30%+ bringen**

- Character Helper Error Tests (6 edge cases)
- RouteCalculator Constructor Test
- Erwartete Coverage: +1.0-1.5pp
- **Aufwand:** 1h

**Task 4.2: Test Report Update**

- TEST_COVERAGE_REPORT.md aktualisieren
- Coverage Metriken dokumentieren
- Phase 3.2 Achievements dokumentieren
- **Aufwand:** 30min

**Task 4.3: Code Cleanup**

- `NewWithConcrete` als deprecated markieren (bereits in Code)
- Ungenutzte Test-Mocks identifizieren
- TODO-Kommentare konsolidieren
- **Aufwand:** 1h

### 🟡 MEDIUM: Verbesserungen (3-4h)

**Task 4.4: Error Handling Review**

- Konsistente Error-Response-Struktur prüfen
- HTTP Status Codes standardisieren
- Error Logging Pattern dokumentieren
- **Aufwand:** 2h

**Task 4.5: Test Documentation**

- Test Patterns dokumentieren (Mock-Strategy)
- Integration vs Unit Test Guidelines
- Best Practices für Handler-Tests
- **Aufwand:** 1h

**Task 4.6: Performance Baseline**

- Benchmark für RouteCalculator.Calculate()
- Benchmark für MarketOrderCache compress/decompress
- Baseline Metriken dokumentieren
- **Aufwand:** 1.5h

### 🔵 LOW: Nice-to-Have (Optional)

**Task 4.7: Table-Driven Test Migration**

- Bestehende Tests zu Table-Driven konvertieren
- **Aufwand:** 3-4h (nur wenn Zeit übrig)

**Task 4.8: Integration Test Suite**

- Testcontainers Setup dokumentieren
- E2E Test Strategy definieren
- **Aufwand:** 4-6h (separates Projekt)

---

## Execution Plan

### Session 1: Quick Wins (2-3h)

1. ✅ Services Coverage: Character Helper Error Tests
2. ✅ RouteCalculator Constructor Test
3. ✅ Coverage Verification (>30%)
4. ✅ TEST_COVERAGE_REPORT.md Update
5. ✅ Code Cleanup (deprecated markers, TODOs)

### Session 2: Quality Improvements (3-4h)

1. Error Handling Review
2. Test Documentation
3. Performance Benchmarks
4. Final Report

---

## Success Criteria

**MUST (Breaking Criteria):**

- ✅ internal/services Coverage >= 30%
- ✅ internal/handlers Coverage >= 40%
- ✅ All tests passing
- ✅ No compiler warnings

**SHOULD (Quality Criteria):**

- Error responses follow consistent pattern
- Test patterns documented
- Performance baseline established
- Code cleanup done

**NICE-TO-HAVE:**

- Table-driven tests migrated
- Integration test strategy defined

---

## Deliverables

1. **Code:**
   - 6+ neue Character Helper Error Tests
   - 1 RouteCalculator Constructor Test
   - 3+ Benchmarks (RouteCalculator, Cache)

2. **Documentation:**
   - Updated TEST_COVERAGE_REPORT.md
   - Test Patterns Guide (neu)
   - Error Handling Guidelines (neu)
   - Performance Baseline Report (neu)

3. **Cleanup:**
   - Deprecated Code markiert
   - TODOs konsolidiert
   - Unused Mocks entfernt

---

## Timeline

**Geschätzt:** 5-7 Stunden Gesamt  
**Realistisch:** 2 Sessions à 3-4h

**Session 1 (heute):** Quick Wins (Tasks 4.1-4.3)  
**Session 2 (später):** Quality Improvements (Tasks 4.4-4.6)

---

## Risiken & Mitigation

**Risiko 1:** Coverage steigt nicht auf 30%

- **Mitigation:** Fokus auf Character Helper (6 Tests = ~1.5pp)

**Risiko 2:** Zeit-Budget überschritten

- **Mitigation:** LOW-Priority Tasks optional

**Risiko 3:** Breaking Changes nötig

- **Mitigation:** Nur additive Changes, keine API-Breaks

---

## Next Steps (jetzt starten)

```bash
# 1. Services Coverage Gap identifizieren
cd backend
go test -coverprofile=coverage.out ./internal/services/...
go tool cover -func=coverage.out | grep -E "character|route_calculator"

# 2. Character Helper Error Tests schreiben
# File: internal/services/character_helpers_test.go
# Tests: GetCharacterInfo error cases (6 neue Tests)

# 3. Coverage messen
go test -coverprofile=coverage.out ./internal/services/...
go tool cover -func=coverage.out | tail -1

# 4. Bei 30%+ → Dokumentation updaten
```

---

**Status:** READY TO START  
**Start:** Jetzt (2025-11-05 11:50 UTC)
