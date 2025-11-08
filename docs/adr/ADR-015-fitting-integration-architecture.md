# ADR-015: Ship Fitting Integration Architecture

Status: Proposed  
Datum: 2025-11-08  
Autoren: Development Team

> Ablageort: ADR-Dateien werden im Verzeichnis `docs/adr/` gepflegt.

## Kontext

**Problem:** Aktuelle Cargo- und Navigationsberechnungen nutzen nur Skills, aber **keine Fitting-Bonuses** (Module, Rigs). Dies führt zu erheblichen Ungenauigkeiten:

- **Cargo Capacity:** 30-50% Ungenauigkeit (fehlende Expanded Cargoholds, Optimization Rigs)
- **Warp Speed:** 20-40% Ungenauigkeit (fehlende Hyperspatial Rigs)
- **Align Time:** 50% Ungenauigkeit (fehlende Inertial Stabilizers)
- **ISK/h Calculations:** 50-100% Ungenauigkeit (kombinierter Effekt)

**User Impact:**

Ein Spieler fitted einen Badger mit 2x Expanded Cargohold II:

- **Reale Kapazität:** 11,094 m³
- **Angezeigte Kapazität:** 6,094 m³ (nur Skills, keine Module)
- **Fehler:** -45% (5,000 m³ fehlen)

→ Route sagt "3 Trips nötig", tatsächlich nur 2 → Zeitverschwendung + falsches ISK/h

**Constraints:**

- **ESI Assets API:** Bereits in `trading.go:367` integriert, aber nur für `LocationFlag == "Hangar"` gefiltert
- **SDE Dogma System:** Benötigt `type_dogma` Tabelle für Modul-Attribute (capacity, warpSpeed, inertia)
- **Performance:** Cache-First Strategie erforderlich (ESI Rate Limits)
- **Graceful Degradation:** System muss ohne Fitting-Daten funktionieren (Fallback auf Skills)

**Stakeholder:**

- **Trading Calculator Users:** Benötigen präzise ISK/h Berechnungen
- **Hauler Pilots:** Nutzen gefittete Schiffe mit Cargo-Expandern
- **Route Planners:** Nutzen Hyperspatial Rigs für schnellere Warps

---

## Betrachtete Optionen

### Option 1: Manual Fitting Input (DNA/EFT Format)

**Beschreibung:** User kopiert Fitting aus EVE Client (DNA String oder EFT Format) und fügt es manuell ein.

**Vorteile:**

- ✅ Keine ESI Assets API benötigt (weniger Scopes)
- ✅ Funktioniert offline (keine API-Calls)
- ✅ Theorycrafting möglich (Fittings testen ohne sie zu besitzen)

**Nachteile:**

- ❌ Manueller Schritt erforderlich (schlechte UX)
- ❌ User muss Fitting aktuell halten (Sync-Problem)
- ❌ Fehleranfällig (falsche Copy-Paste)
- ❌ Keine Echtzeit-Validierung (User könnte unmögliche Fittings eingeben)

**Risiken:**

- User vergisst Fitting zu aktualisieren nach Modul-Wechsel
- DNA/EFT Parsing komplex (Format-Variationen)

**Bewertung:** ❌ Abgelehnt (User wollte "exakte, asset-basierte" Lösung)

---

### Option 2: Asset-Based Fitting Detection (ESI Assets API)

**Beschreibung:** Automatische Erkennung gefitteter Module via ESI `/v5/characters/{id}/assets/` Endpoint mit `location_flag` Filterung.

**Vorteile:**

- ✅ **Echtzeit-Genauigkeit:** Spiegelt tatsächliches In-Game Fitting wider
- ✅ **Zero-Input UX:** Keine manuelle Eingabe erforderlich
- ✅ **Automatische Synchronisation:** Fitting-Änderungen sofort sichtbar
- ✅ **ESI API bereits integriert:** `trading.go:367` nutzt Assets API bereits
- ✅ **location_flag bereits erfasst:** Nur Filter-Erweiterung nötig

**Nachteile:**

- ❌ ESI Scope erforderlich (`esi-assets.read_assets.v1`)
- ❌ ESI Rate Limits (aber: Cache-First Strategie mit 5min TTL)
- ❌ Offline nicht nutzbar (aber: Cache als Fallback)

**Risiken:**

- ESI Assets Pagination (bei Spielern mit vielen Items)
- SDE Dogma Daten unvollständig/fehlend

**Bewertung:** ✅ **Gewählt** (User-Anforderung: "asset-basiert", beste UX)

---

### Option 3: Hybrid Approach (Assets + Manual Override)

**Beschreibung:** Asset-basierte Erkennung als Standard, manuelle Override-Möglichkeit für Theorycrafting.

**Vorteile:**

- ✅ Kombiniert Vorteile beider Ansätze
- ✅ Theorycrafting möglich für geplante Fittings

**Nachteile:**

- ❌ Höhere Komplexität (2 Systeme parallel)
- ❌ UI komplizierter (Toggle zwischen Auto/Manual)
- ❌ Mehr Test-Aufwand

**Bewertung:** 🔄 **Deferred** (Phase 6+, erst MVP mit Option 2)

---

## Entscheidung

**Gewählte Option:** **Option 2 - Asset-Based Fitting Detection**

**Begründung:**

1. **User-Anforderung erfüllt:** "ich möchte es schon exakt haben also asset-basiert"
2. **Technische Feasibility:** ESI Assets API bereits integriert, nur Filter-Extension nötig
3. **Beste UX:** Zero-Input, automatische Synchronisation, Echtzeit-Genauigkeit
4. **Performance machbar:** Cache-First (Redis 5min TTL) verhindert ESI Rate Limit Issues
5. **Follows Existing Patterns:** SkillsService Pattern (ADR-014) wiederverwendbar

**Akzeptierte Trade-offs:**

- ✅ ESI Scope erforderlich (`esi-assets.read_assets.v1`) → User muss zustimmen
- ✅ Offline nicht nutzbar → Cache als Fallback (5min stale data akzeptabel)
- ✅ Theorycrafting nicht möglich → Future Feature (Hybrid Approach in Phase 6+)

**Annahmen:**

- SDE `type_dogma` Tabelle existiert oder wird migriert
- ESI Assets API stabil (keine Breaking Changes)
- Dogma Attribute IDs 38 (capacity), 20 (warpSpeed), 70 (inertia) korrekt

---

## Konsequenzen

### Positiv

**Accuracy Improvements:**

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Cargo Capacity (Badger) | 6,094 m³ | 11,094 m³ | **+82%** |
| Warp Time (20 AU) | 27.0s (skills) | 21.6s (skills+rigs) | **-20%** |
| ISK/h Calculation | 10M (inaccurate) | 17M (accurate) | **+70%** |

**User Experience:**

- ✅ Automatische Fitting-Erkennung (keine manuelle Eingabe)
- ✅ Echtzeit-Synchronisation mit In-Game Fitting
- ✅ Präzise Route-Planung (korrekte Cargo + Travel Time)

**Technical Benefits:**

- ✅ Wiederverwendung existierender ESI Integration (`trading.go:367`)
- ✅ Konsistenz mit SkillsService Pattern (ADR-014)
- ✅ Cache-First Design (geringe ESI Last)

### Negativ

**ESI Dependency:**

- ❌ Zusätzlicher ESI Scope erforderlich (`esi-assets.read_assets.v1`)
- ❌ Offline nicht nutzbar (Cache als Fallback)
- ❌ ESI Rate Limits (aber: Cache-First mitigiert)

**Complexity:**

- ❌ Dogma Attribute Mapping erforderlich (SDE Integration)
- ❌ Stacking Penalty Calculation (komplexe Formel für Module)
- ❌ ESI Assets Pagination handling (bei vielen Items)

**Technical Debt:**

- ❌ Kein Theorycrafting (manuelles Fitting-Input fehlt)
- ❌ Keine Fitting-Import/Export (EFT/DNA Format)

### Risiken

**Risk 1: SDE Dogma Table Missing**

- **Probability:** Medium
- **Impact:** High (Implementation blockiert)
- **Mitigation:** Verify `type_dogma` schema in `eve-sde` project before starting
- **Contingency:** Migrate YAML SDE data to SQLite (slower but functional)

**Risk 2: ESI Rate Limits**

- **Probability:** Low
- **Impact:** Medium (User experience degraded)
- **Mitigation:** Aggressive caching (5min TTL), Circuit Breaker pattern
- **Contingency:** Extend cache TTL to 15min if rate limits become issue

**Risk 3: ESI Assets Pagination**

- **Probability:** Medium (players with many items)
- **Impact:** Medium (incomplete fitting data)
- **Mitigation:** Fetch all pages until `X-Pages` header indicates end
- **Contingency:** Document pagination in API guide

**Risk 4: Stacking Penalty Complexity**

- **Probability:** Low
- **Impact:** Low (rare edge case)
- **Mitigation:** Phase 2 defers stacking if complex
- **Contingency:** Most modules don't stack (cargo expanders additive, not stacking)

---

## Implementierung

### Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                  FittingService (NEW)                   │
│  • GetShipFitting(characterID, shipItemID, token)       │
│  • Fetches: ESI Assets + SDE Dogma Attributes           │
│  • Calculates: Cargo/Warp/Inertia Bonuses               │
│  • Caches: Redis 5min TTL                               │
└────────────────────┬────────────────────────────────────┘
                     │
         ┌───────────┴───────────┐
         │                       │
         ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│ CargoService    │     │ NavigationSvc   │
│ (EXTEND)        │     │ (EXTEND)        │
│ + CalculateEffec│     │ + CalculateWarp │
│   tiveCapacity()│     │   Time()        │
└─────────────────┘     └─────────────────┘
         │                       │
         └───────────┬───────────┘
                     ▼
         ┌─────────────────────┐
         │   RouteService      │
         │ • Uses accurate     │
         │   cargo + warp time │
         │ • ISK/h calculation │
         └─────────────────────┘
```

### Service Structure

**FittingService:**

```go
type FittingService struct {
    esiClient   *esiclient.Client  // Rate limiting + caching
    redisClient *redis.Client      // 5min TTL cache
    sdeQuerier  SDEQuerier          // Dogma attributes
    logger      zerolog.Logger
}

func (s *FittingService) GetShipFitting(
    ctx context.Context,
    characterID int,
    shipItemID int64,
    accessToken string,
) (*ShipFitting, error)
```

**Data Flow:**

1. Check Redis cache (`fitting:{characterID}:{shipItemID}`)
2. Fetch ESI Assets (reuse `trading.go` pattern)
3. Filter by `location_flag` (HiSlot0-7, MedSlot0-7, LoSlot0-7, RigSlot0-2)
4. Query SDE for dogma attributes (batch query)
5. Calculate bonuses (cargo additive, warp/inertia multiplicative)
6. Cache result (5min TTL)
7. Return ShipFitting struct

**ESI Location Flags:**

- `"HiSlot0"` - `"HiSlot7"` → High slot modules
- `"MedSlot0"` - `"MedSlot7"` → Medium slot modules
- `"LoSlot0"` - `"LoSlot7"` → Low slot modules
- `"RigSlot0"` - `"RigSlot2"` → Rigs

**SDE Dogma Attributes:**

| ID | Name | Effect | Example |
|----|------|--------|---------|
| 38 | capacity | Cargo volume (+m³) | Expanded Cargohold II: +2,500 m³ |
| 20 | warpSpeedMultiplier | Warp speed (%) | Hyperspatial Rig: +20% |
| 70 | inertiaModifier | Align time (%) | Inertial Stabilizer II: -20% |

### Integration Points

**CargoService Extension:**

```go
// OLD (only skills)
effectiveCapacity = baseCapacity 
                  × (1 + SpaceshipCommand × 0.05)
                  × (1 + RacialIndustrial × 0.05)

// NEW (skills + fitting)
effectiveCapacity = baseCapacity 
                  × (1 + SpaceshipCommand × 0.05)
                  × (1 + RacialIndustrial × 0.05)
                  + fittingCargoBonus  // ADDITIVE
```

**NavigationService Extension:**

```go
effective_warp_speed = base_warp_speed 
                     × (1 + Navigation_skill × 0.05)
                     × fitting.WarpSpeedBonus

effective_inertia = base_inertia
                  × (1 - EvasiveManeuvering × 0.05)
                  × fitting.InertiaBonus
```

### Effort Estimation

| Phase | Tasks | Effort |
|-------|-------|--------|
| 1. FittingService Foundation | ESI + SDE + Cache + Tests | 12h |
| 2. Bonus Calculations | Cargo/Warp/Inertia + Stacking | 8h |
| 3. Service Integration | CargoService + NavigationService | 12h |
| 4. Frontend Integration | ShipFittingCard + UI updates | 8h |
| 5. Testing & Documentation | E2E + ADR + API Docs | 8h |
| **Total** | | **48h** |

### Abhängigkeiten

**Required:**

- ✅ #40: ESI Skills Integration (completed)
- ✅ #67: ESI Character Standings (completed)
- 🔄 #52: Cargo Skills Integration (in progress)
- 🔄 #53: Navigation Skills Integration (in progress)
- SDE `type_dogma` table (to be verified)

**Blocks:**

- #42: Volume Filter (needs accurate cargo)
- #38: Profit Calculator (needs accurate ISK/h)

### Validierung

**Success Criteria:**

**Accuracy:**

- [ ] Cargo capacity error < 5% (vs. current ~50%)
- [ ] Warp time error < 10% (vs. current ~30%)
- [ ] ISK/h accuracy > 90% (vs. current ~50%)

**Performance:**

- [ ] FittingService cache hit rate > 80%
- [ ] API response time p95 < 300ms
- [ ] ESI error rate < 5%

**Adoption:**

- [ ] % of routes using fitted ships > 50%
- [ ] User feedback: "Accurate cargo" mentions +50%

---

## Referenzen

**Issues:**

- #76: Ship Fitting Integration (this ADR)
- #40: ESI Skills Integration (✅ completed)
- #52: Cargo Skills Integration (🔄 in progress)
- #53: Navigation Skills Integration (🔄 in progress)

**ADRs:**

- ADR-001: Tech Stack (Go + Next.js + PostgreSQL + Redis)
- ADR-014: ESI Integration Pattern (SkillsService template)
- ADR-012: Redis Caching Strategy (5min TTL)

**Externe Docs:**

- EVE University: Warp Time Calculation (<https://wiki.eveuniversity.org/Warp_time_calculation>)
- ESI Documentation: <https://esi.evetech.net/ui/>
- Community Tools: Pyfa, EVEShip.fit, Theorycrafter

**Code References:**

- `backend/internal/services/skills_service.go` (pattern template)
- `backend/internal/services/cargo_service.go` (extension target)
- `backend/internal/handlers/trading.go:367` (ESI assets usage)

**Detailed Plan:**

- `tmp/ship-fitting-integration-plan.md` (600+ lines, complete architecture)

---

## Notizen

**ESI Assets API Discovery:**

Während der Recherche entdeckt: Assets API bereits in `trading.go:367` integriert!

```go
type esiAssetResponse struct {
    ItemID       int64  `json:"item_id"`
    TypeID       int64  `json:"type_id"`
    LocationID   int64  `json:"location_id"`
    LocationFlag string `json:"location_flag"`  // ⭐ CRITICAL
    IsSingleton  bool   `json:"is_singleton"`
}

// Current filter (line 390):
if asset.LocationFlag != "Hangar" { continue }

// New filters needed:
if isFittedSlot(asset.LocationFlag) {
    // HiSlot0-7, MedSlot0-7, LoSlot0-7, RigSlot0-2
}
```

**Impact:** Significantly reduced implementation effort (no new ESI endpoint, just filter extension).

**Stacking Penalties:**

EVE's stacking penalty formula for modules with same effect:

```
penalty = 1 - (1 - bonus) × 0.5^((n-1)^2)
```

**Example:** 2x Inertial Stabilizer II (-20% each):

- First module: -20% (full effect)
- Second module: -20% × 0.5^1 = -10% (50% effective)
- Total: -30% (not -40%)

**Decision:** Phase 2 defers stacking if complex (most cargo modules are additive, not stacking).

**Future Enhancements (Post-MVP):**

1. **Manual Fitting Override** (Phase 6+)
   - Theorycrafting for planned fittings
   - Fitting comparison tool
   - Hybrid approach (auto + manual toggle)

2. **Fitting Import/Export** (Phase 7+)
   - DNA String parsing
   - EFT Format parsing
   - XML Format support

3. **Fitting Recommendations** (Phase 8+)
   - Optimal cargo hauler fittings
   - Fastest warp fittings
   - ISK/h optimized fittings

---

**Change Log:**

- 2025-11-08: Status auf Proposed gesetzt (Development Team)
- 2025-11-08: Initial draft nach Research Phase (Issue #76 erstellt)
