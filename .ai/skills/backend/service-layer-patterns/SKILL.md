# Backend Skill: Service Layer Patterns

**Tech Stack:** Go 1.24 + Fiber v2.52.9 + Redis + PostgreSQL

**Project:** eve-o-provit Backend Services

---

## Architecture Context

Service Layer sitzt zwischen **Handler Layer** (HTTP) und **Repository Layer** (Data Access):

```
Handler → Service → Repository → Database
   ↓         ↓          ↓
  HTTP   Business    SQL/NoSQL
         Logic
```

**Verantwortlichkeiten:**
- Business Logic Orchestration
- Multi-Repository Coordination
- External API Integration (ESI, Redis)
- Caching Strategy
- Error Handling & Graceful Degradation

---

## Best Practices (Normative Requirements)

1. **Constructor Injection (MUST):** Alle Dependencies über Constructor, keine Globals
2. **Interface Definition (MUST):** Public Service Interface (`XXXServicer`) für Testing
3. **Context Propagation (MUST):** Alle Methoden akzeptieren `context.Context` als ersten Parameter
4. **Graceful Degradation (SHOULD):** Fallback-Strategie bei External Service Failures
5. **Caching Integration (SHOULD):** Redis für häufig abgefragte Daten (ADR-012)
6. **Logging (MUST):** Strukturiertes Logging (Debug, Info, Warn, Error)
7. **Metrics (SHOULD):** Prometheus Metrics für kritische Operationen

---

## Pattern 1: Service Constructor (Dependency Injection)

**Standard Pattern:**

```go
// Service Struct (Private Fields)
type SkillsService struct {
    esiClient   *esiclient.Client  // External API Client
    redisClient *redis.Client      // Caching Layer
    logger      *logger.Logger     // Structured Logging
}

// Constructor (Dependency Injection)
func NewSkillsService(
    esiClient *esiclient.Client,
    redisClient *redis.Client,
    logger *logger.Logger,
) SkillsServicer {  // Return Interface (not concrete type)
    return &SkillsService{
        esiClient:   esiClient,
        redisClient: redisClient,
        logger:      logger,
    }
}
```

**Why Interface Return?**
- Testing: Mock Interface in Handler Tests
- Decoupling: Handler doesn't depend on implementation
- Swappability: Easy to replace implementation

---

## Pattern 2: Public Interface Definition

**Interface in `interfaces.go`:**

```go
// SkillsServicer defines public API for Skills Service
type SkillsServicer interface {
    GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error)
    // Add methods as needed
}
```

**Why Separate File?**
- Clarity: Interface vs. Implementation separated
- Testing: Import interface for mocking
- Documentation: Public API at a glance

**Naming Convention:**
- Interface: `XXXServicer` (e.g., `SkillsServicer`, `TradingServicer`)
- Implementation: `XXXService` (e.g., `SkillsService`, `TradingService`)

---

## Pattern 3: Caching Strategy (Redis Integration)

**Standard Caching Flow:**

```go
func (s *SkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
    // 1. Check Redis Cache First
    cacheKey := fmt.Sprintf("character_skills:%d", characterID)
    cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
    if err == nil {
        s.logger.Debug("Cache hit", "key", cacheKey)
        var skills TradingSkills
        if err := json.Unmarshal(cachedData, &skills); err == nil {
            return &skills, nil
        }
        s.logger.Warn("Cache unmarshal failed", "error", err)
    }
    
    // 2. Cache Miss - Fetch from Source (ESI, Database, etc.)
    s.logger.Debug("Cache miss - fetching from source", "key", cacheKey)
    skills, err := s.fetchSkillsFromESI(ctx, characterID, accessToken)
    if err != nil {
        // Graceful Degradation (see Pattern 5)
        return s.getDefaultSkills(), nil
    }
    
    // 3. Cache Result (TTL: 5min for semi-static data)
    if skillsData, err := json.Marshal(skills); err == nil {
        if err := s.redisClient.Set(ctx, cacheKey, skillsData, 5*time.Minute).Err(); err != nil {
            s.logger.Warn("Cache set failed", "error", err)
        }
    }
    
    return skills, nil
}
```

**Cache Key Naming Convention:**
```
{entity}:{identifier}             → character_skills:123456
{entity}:{id}:{subresource}       → character_assets:123456:jita
{entity}:{param1}:{param2}        → market_orders:10000002:34
```

**TTL Guidelines (ADR-012):**
- **Character Skills:** 5min (semi-static)
- **Market Orders:** 5min (ESI updates ~5min)
- **Character Standing:** 1h (rarely changes)
- **Navigation Routes:** 1h (static data)

---

## Pattern 4: ESI Integration (External API)

**Following ADR-014 (ESI Integration Pattern):**

```go
func (s *SkillsService) fetchSkillsFromESI(ctx context.Context, characterID int, accessToken string) (*esiSkillsResponse, error) {
    // 1. Build HTTP Request
    endpoint := fmt.Sprintf("/v4/characters/%d/skills/", characterID)
    req, err := http.NewRequestWithContext(ctx, "GET", "https://esi.evetech.net"+endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    // 2. Add Authorization Header
    req.Header.Set("Authorization", "Bearer "+accessToken)
    
    // 3. Execute via eve-esi-client (Rate Limiting + Caching + Retries)
    resp, err := s.esiClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("esi request: %w", err)
    }
    defer resp.Body.Close()
    
    // 4. Handle HTTP Status Codes
    if resp.StatusCode == 401 || resp.StatusCode == 403 {
        return nil, fmt.Errorf("unauthorized: %d", resp.StatusCode)
    }
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("ESI status %d: %s", resp.StatusCode, string(body))
    }
    
    // 5. Parse JSON Response
    var result esiSkillsResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }
    
    return &result, nil
}
```

**Error Handling:**
- **401/403:** Return error (Frontend handles re-auth)
- **404:** Return `ErrNotFound` (resource doesn't exist)
- **5xx:** Retry handled by eve-esi-client, fallback to defaults
- **Timeout:** Context timeout propagates up

---

## Pattern 5: Graceful Degradation

**Fallback Strategy for External Service Failures:**

```go
func (s *SkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
    // ... Cache Check ...
    
    // Fetch from ESI
    esiSkills, err := s.fetchSkillsFromESI(ctx, characterID, accessToken)
    if err != nil {
        s.logger.Error("ESI fetch failed - using defaults", 
            "error", err, 
            "characterID", characterID)
        
        // Graceful Degradation: Return safe defaults
        // Ensures service continues working even if ESI is down
        return s.getDefaultSkills(), nil  // ⚠️ Return nil error (degraded mode)
    }
    
    // ... Process & Cache ...
    return extractedSkills, nil
}

// Default values for worst-case scenario
func (s *SkillsService) getDefaultSkills() *TradingSkills {
    return &TradingSkills{
        Accounting:              0,  // Worst-case: highest sales tax
        BrokerRelations:         0,  // Worst-case: highest broker fee
        AdvancedBrokerRelations: 0,
        FactionStanding:         0.0,
        // ... all skills = 0 (untrained)
    }
}
```

**When to Use Graceful Degradation:**
- ✅ Non-critical data (skills for fee calculation)
- ✅ Temporary ESI outages (service continues with defaults)
- ✅ User experience priority (degraded > unavailable)
- ❌ Critical operations (financial transactions, authentication)

---

## Pattern 6: Multi-Repository Orchestration

**Service coordinates multiple repositories:**

```go
type TradingService struct {
    marketRepo     database.MarketQuerier   // Market data
    sdeRepo        database.SDEQuerier      // Static game data
    navigationRepo database.NavigationQuerier // Route calculation
    esiClient      ESIClient                 // External API
}

func (s *TradingService) CalculateInventorySellRoutes(
    ctx context.Context,
    req models.InventorySellRequest,
) ([]models.InventorySellRoute, error) {
    // 1. Fetch from ESI (External API)
    orders, err := s.esiClient.GetMarketOrders(ctx, req.RegionID, req.TypeID)
    if err != nil {
        return nil, fmt.Errorf("fetch orders: %w", err)
    }
    
    // 2. Query SDE (Static Data)
    itemInfo, err := s.sdeRepo.GetItemInfo(ctx, req.TypeID)
    if err != nil {
        return nil, fmt.Errorf("fetch item info: %w", err)
    }
    
    // 3. Calculate Navigation Routes (Multi-Repo)
    for _, order := range buyOrders {
        route, err := s.navigationRepo.CalculateRoute(ctx, startSystemID, order.SystemID)
        if err != nil {
            s.logger.Warn("Route calc failed", "error", err)
            continue // Skip this order
        }
        // ... Profit calculation ...
        routes = append(routes, calculatedRoute)
    }
    
    return routes, nil
}
```

**Orchestration Best Practices:**
- Handle errors per repository (don't fail entire operation on one error)
- Use context timeout for long operations
- Log warnings for partial failures
- Return partial results when possible (ADR-013)

---

## Pattern 7: Structured Logging

**Logging Levels:**

```go
// DEBUG: Cache hits, internal state changes
s.logger.Debug("Cache hit", "key", cacheKey, "characterID", characterID)

// INFO: Successful operations, important state changes
s.logger.Info("Skills fetched from ESI", 
    "characterID", characterID, 
    "accounting", skills.Accounting)

// WARN: Recoverable errors, degraded mode
s.logger.Warn("Cache unmarshal failed - fetching from ESI", "error", err)

// ERROR: Unrecoverable errors, external service failures
s.logger.Error("ESI fetch failed - using defaults", 
    "error", err, 
    "characterID", characterID)
```

**Structured Fields (Key-Value Pairs):**
- Always include context: `characterID`, `regionID`, `typeID`
- Include error: `"error", err` (when logging errors)
- Avoid sensitive data: No tokens, passwords, PII

---

## Pattern 8: Testing (Mock Interfaces)

**Service Interface Mocking:**

```go
// Mock in handler tests
type MockSkillsService struct {
    GetCharacterSkillsFunc func(ctx context.Context, characterID int, accessToken string) (*services.TradingSkills, error)
}

func (m *MockSkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*services.TradingSkills, error) {
    if m.GetCharacterSkillsFunc != nil {
        return m.GetCharacterSkillsFunc(ctx, characterID, accessToken)
    }
    // Default mock behavior
    return &services.TradingSkills{Accounting: 5}, nil
}

// In handler test
func TestHandler(t *testing.T) {
    mockSkills := &MockSkillsService{
        GetCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*services.TradingSkills, error) {
            return &services.TradingSkills{Accounting: 3}, nil
        },
    }
    
    handler := NewHandler(mockSkills)
    // ... test handler ...
}
```

---

## Anti-Patterns

❌ **Global State:** Never use package-level variables for services

❌ **Direct Repository Access in Handlers:** Always go through service layer

❌ **Ignoring Cache Errors:** Cache failures should not break service (graceful degradation)

❌ **No Context Timeout:** Always set timeout for external API calls

❌ **Synchronous External Calls without Fallback:** ESI can fail, always have defaults

❌ **Logging Sensitive Data:** No tokens, passwords, or PII in logs

---

## Integration with Other Layers

### Handler Layer
- Handlers receive service via constructor injection
- Handlers call service methods with context from `fiber.Ctx`
- Handlers handle HTTP-specific errors (status codes, JSON encoding)

### Repository Layer
- Services call repository methods for data access
- Services orchestrate multiple repositories
- Services handle repository errors (retry, fallback)

### External APIs (ESI)
- Services use eve-esi-client for ESI integration (ADR-014)
- Services handle ESI errors gracefully (defaults, retries)
- Services cache ESI responses in Redis (ADR-012)

---

## Performance Considerations

- **Caching First:** Always check Redis before external API/Database
- **Context Timeout:** Set explicit timeouts for long operations (15s max for ESI, ADR-013)
- **Connection Pooling:** Reuse esiClient, redisClient (don't create per request)
- **Async Operations:** Use goroutines for non-critical operations (cache updates)
- **Partial Results:** Return partial data on timeout (ADR-013)

---

## References

- **ADRs:**
  - ADR-012: Redis Caching Strategy
  - ADR-013: Timeout Handling & Partial Content
  - ADR-014: ESI Integration Pattern
- **Examples:**
  - `internal/services/skills_service.go` (Issue #54)
  - `internal/services/trading_service.go`
- **Testing:**
  - `internal/services/skills_service_test.go`

---

**Last Updated:** 2025-11-06 (extracted from Issue #54)
