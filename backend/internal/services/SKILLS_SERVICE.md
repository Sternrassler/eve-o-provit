# Skills Service Documentation

## Overview

The Skills Service provides centralized character skills fetching and caching for all trading-related features in EVE-O-Provit. It fetches character skills from ESI, extracts trading-relevant skills, and caches results in Redis.

## Purpose (Phase 0 - Issue #54)

**Goal:** Create a single source of truth for character skills to be used by:
- Fee Service (Issue #55)
- Cargo Service (Issue #56)
- All future features requiring skill-based calculations

## Architecture

### Interface

```go
type SkillsServicer interface {
    GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error)
}
```

### Implementation

**File:** `internal/services/skills_service.go`

**Dependencies:**
- ESI Client (temporary interface until `eve-esi-client` implements `GetCharacterSkills`)
- Redis Client (for caching)
- Logger (structured logging)

**Key Features:**
1. **Redis Caching (5min TTL)**
   - Cache key: `character_skills:{characterID}`
   - JSON-serialized `TradingSkills` struct
   - Reduces ESI load, improves response times

2. **Graceful Degradation**
   - ESI failure → Returns default skills (all = 0)
   - Allows trading features to continue with worst-case assumptions
   - No blocking errors for skill fetch failures

3. **Skill Extraction**
   - Extracts 12 trading-relevant skills from ESI response
   - Maps EVE skill IDs to `TradingSkills` struct fields
   - Ignores unknown/irrelevant skills

## Skill Mappings

| Skill Name                  | Skill ID | Field                     | Impact                                 |
|-----------------------------|----------|---------------------------|----------------------------------------|
| Accounting                  | 16622    | `Accounting`              | -10% sales tax per level (max -50%)   |
| Broker Relations            | 3446     | `BrokerRelations`         | -0.3% broker fee per level (max -1.5%) |
| Advanced Broker Relations   | 3447     | `AdvancedBrokerRelations` | -0.3% broker fee per level (max -1.5%) |
| Navigation                  | 3449     | `Navigation`              | +5% warp speed per level (max +25%)    |
| Evasive Maneuvering         | 3452     | `EvasiveManeuvering`      | -5% align time per level (max -25%)    |
| Spaceship Command           | 3327     | `SpaceshipCommand`        | +5% cargo capacity per level (max +25%)|
| (Ship-specific Cargo Skill) | varies   | `CargoOptimization`       | +5% cargo per level (max +25%)         |
| Gallente Industrial         | 3348     | `GallenteIndustrial`      | +5% cargo per level (Gallente ships)   |
| Caldari Industrial          | 3346     | `CaldarIndustrial`        | +5% cargo per level (Caldari ships)    |
| Amarr Industrial            | 3347     | `AmarrIndustrial`         | +5% cargo per level (Amarr ships)      |
| Minmatar Industrial         | 3349     | `MinmatarIndustrial`      | +5% cargo per level (Minmatar ships)   |

**Note:** Faction Standing (0.0-10.0) affects broker fees but is not a skill (fetched separately).

## TradingSkills Struct

```go
type TradingSkills struct {
    // Trading Skills
    Accounting              int     // 0-5 (0 = untrained, 5 = max)
    BrokerRelations         int     // 0-5
    AdvancedBrokerRelations int     // 0-5
    FactionStanding         float64 // 0.0-10.0 (separate from skills)

    // Cargo Skills
    SpaceshipCommand  int // 0-5
    CargoOptimization int // 0-5

    // Navigation Skills
    Navigation         int // 0-5
    EvasiveManeuvering int // 0-5

    // Ship-specific Industrial Skills
    GallenteIndustrial int // 0-5
    CaldarIndustrial   int // 0-5
    AmarrIndustrial    int // 0-5
    MinmatarIndustrial int // 0-5
}
```

## Usage Example

```go
// In Fee Service (Issue #55)
skills, err := skillsService.GetCharacterSkills(ctx, characterID, accessToken)
if err != nil {
    // Should never happen (graceful degradation returns default skills)
    return err
}

// Calculate tax rate based on Accounting skill
taxReduction := float64(skills.Accounting) * 0.01 // -1% per level
baseTax := 0.055 // 5.5% base sales tax
effectiveTax := baseTax * (1 - taxReduction)
```

## Error Handling

**No Hard Failures:**
- ESI timeout → Default skills (all = 0)
- Redis unavailable → ESI fetch every time (no cache)
- Cache corruption → Log warning, re-fetch from ESI

**Rationale:**
Trading features should degrade gracefully. Worst-case skills mean higher fees/lower cargo, but features remain functional.

## Testing

**Test Suite:** `internal/services/skills_service_test.go`

**Coverage:**
- ✅ Cache hit scenario (ESI not called)
- ✅ Cache miss scenario (ESI called, result cached)
- ✅ ESI error scenario (graceful fallback to defaults)
- ✅ Individual skill extraction (6 test cases)
- ✅ Multiple skills extraction
- ✅ Unknown skill IDs ignored
- ✅ Default skills validation

**Test Framework:** `miniredis` + `testify` (consistent with existing services)

## Integration Status

**Current State (v0.1.x):**
- ✅ Service implemented
- ✅ Tests passing
- ✅ Logger package created
- ✅ Interface defined in `interfaces.go`
- ⏳ Integration commented out in `main.go` (waiting for ESI client implementation)

**Blocked By:**
- `eve-esi-client` package missing `GetCharacterSkills` method
- Temporary `ESIClient` interface created in `skills_service.go` as workaround

**Next Steps:**
1. Implement `GetCharacterSkills` in `eve-esi-client` package
2. Move `CharacterSkillsResponse` and `Skill` types to `eve-esi-client`
3. Uncomment integration in `main.go`
4. Use in Fee Service (#55) and Cargo Service (#56)

## Cache Strategy

**TTL:** 5 minutes

**Rationale:**
- Skills change infrequently (training takes hours/days)
- Active training → want updates within reasonable time
- Balances freshness vs. ESI load

**Key Pattern:** `character_skills:{characterID}`

**Invalidation:**
- Automatic (TTL expiry)
- Manual invalidation not implemented (not needed for skill training timescales)

## Dependencies

**Internal:**
- `pkg/logger` (simple structured logger)
- `redis/go-redis/v9` (Redis client)

**External (Temporary):**
- `ESIClient` interface (local definition until `eve-esi-client` implements)
- `CharacterSkillsResponse`, `Skill` types (local until moved to `eve-esi-client`)

**Future:**
- `github.com/Sternrassler/eve-esi-client` (when `GetCharacterSkills` implemented)

## Performance

**Cache Hit:** ~1ms (Redis GET + JSON unmarshal)
**Cache Miss:** ~200ms (ESI roundtrip + cache write)
**ESI Failure:** ~5s (timeout + fallback to defaults)

**Expected Hit Rate:** >95% (5min TTL, skills change rarely)

## Monitoring

**Logs:**
- `Debug`: Cache hit/miss
- `Info`: ESI fetch success
- `Warn`: Cache unmarshal failure, cache write failure
- `Error`: ESI fetch failure (before fallback)

**Metrics (Future):**
- Cache hit rate
- ESI fetch latency
- Default skills fallback count

## Migration Path (TODO)

**Phase 1: Implement in eve-esi-client**
1. Add `GetCharacterSkills` method to `eve-esi-client/pkg/client/client.go`
2. Move types to `eve-esi-client/pkg/client/types.go`
3. Use ESI endpoint: `GET /characters/{character_id}/skills/`
4. Include rate limiting, caching, error handling (consistent with existing methods)

**Phase 2: Update skills_service.go**
1. Remove temporary `ESIClient` interface
2. Remove temporary types (`CharacterSkillsResponse`, `Skill`)
3. Import from `eve-esi-client` package
4. Update `NewSkillsService` parameter type

**Phase 3: Activate in main.go**
1. Uncomment `skillsService` instantiation
2. Pass to Fee Service, Cargo Service constructors

## Related Issues

- **Issue #54** (This implementation)
- **Issue #55** Fee Service (consumer)
- **Issue #56** Cargo Service (consumer)
- **Issue #57** Handler Cleanup (uses Fee/Cargo services)

## References

- [EVE ESI Documentation - Skills](https://esi.evetech.net/ui/#/Skills)
- [EVE Skill IDs Reference](https://www.fuzzwork.co.uk/dump/latest/invTypes.csv)
- [Phase 0 Dependency Validation](../../docs/phase0-dependency-validation.md)
