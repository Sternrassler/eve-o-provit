# ADR-014: ESI Integration Pattern

Status: Accepted
Datum: 2025-11-06
Autoren: GitHub Copilot (aus Issue #54 Skills Service extrahiert)

> Ablageort: ADR-Dateien werden im Verzeichnis `docs/adr/` gepflegt.

## Kontext

EVE ESI API Integration ist zentral für eve-o-provit (Market Orders, Character Skills, Assets, etc.). Issue #54 (Skills Service) hat ein wiederverwendbares Pattern etabliert, das bereits an mehreren Stellen verwendet wird:

- **Skills Service:** Character Skills Fetching (`/v4/characters/{id}/skills/`)
- **Trading Service:** Market Orders (implizit über `esiClient`)
- **Zukünftig:** Fee Service (#55), Cargo Service (#56), Assets, Standings

**Fragen:**

- Wie integrieren wir ESI API konsistent über alle Services?
- Wie handhaben wir Authorization (Bearer Token)?
- Wie nutzen wir `eve-esi-client` Package optimal (Rate Limiting, Caching, Retries)?
- Wie behandeln wir Fehler (401/403 vs. 5xx)?
- Wie implementieren wir Graceful Degradation bei ESI Ausfällen?

**Constraints:**

- ESI Rate Limit: 150 req/s burst, 20 req/s sustained
- OAuth Token Management: Bearer Token aus Frontend (PKCE Flow, ADR-004)
- Shared Redis Infrastructure (ADR-009) für Caching
- Timeout Handling (ADR-013): Max 15s pro ESI Request
- Graceful Degradation: Service muss auch bei ESI Ausfall funktionieren

**Stakeholder:**

- Backend Services (Skills, Fees, Cargo, Assets)
- Frontend (API Contract Stability)
- Operations (Rate Limiting Monitoring)

## Betrachtete Optionen

### Option 1: Direkter HTTP Client (stdlib net/http)

- **Vorteile:**
  - Keine zusätzliche Dependency
  - Volle Kontrolle über HTTP Details
  - Einfaches Debugging
- **Nachteile:**
  - Kein automatisches Rate Limiting
  - Kein integriertes Caching
  - Kein Retry Handling
  - Boilerplate Code pro Service
- **Risiken:**
  - Rate Limit Überschreitung (ESI Ban)
  - Inkonsistente Error Handling

### Option 2: eve-esi-client Package (Generic Client)

- **Vorteile:**
  - Rate Limiting eingebaut (Thread-Safe)
  - Caching Support (Redis Integration)
  - Retry Logic (Exponential Backoff)
  - Konsolidierte Metrics (Prometheus)
  - Wartung zentral (ein Package)
- **Nachteile:**
  - Externe Dependency (eve-esi-client)
  - Zusätzliche Abstraktionsschicht
  - Erfordert HTTP Request Konstruktion
- **Risiken:**
  - eve-esi-client Breaking Changes
  - Overhead bei einfachen Requests

### Option 3: Generierter ESI Client (OpenAPI/Swagger Codegen)

- **Vorteile:**
  - Type-Safe API Calls
  - Automatisch generiert aus ESI Spec
  - Keine manuelle Request Konstruktion
- **Nachteile:**
  - Sehr großer generierter Code
  - Schwierig anzupassen (Custom Logic)
  - ESI Spec Updates erfordern Re-Generation
  - Keine Rate Limiting/Caching Integration
- **Risiken:**
  - Maintenance Overhead (Spec Updates)
  - Vendor Lock-in (Generator Tool)

## Entscheidung

**Gewählte Option:** Option 2 - eve-esi-client Package (Generic Client)

**Begründung:**

1. **Rate Limiting Critical:** ESI Ban bei Überschreitung (150 req/s) - eve-esi-client implementiert Thread-Safe Rate Limiter
2. **Caching Integriert:** Nutzt Redis (ADR-009) automatisch für ESI Cache Headers
3. **Retry Logic:** Exponential Backoff bei 5xx Errors (ESI Instabilität)
4. **Konsolidierte Metrics:** Prometheus Metrics für alle ESI Requests (Monitoring)
5. **Wartbarkeit:** Ein Package für alle Services - Bugfixes zentral

**Pattern Details:**

```go
// 1. Create HTTP Request with Context
req, err := http.NewRequestWithContext(ctx, "GET", esiURL, nil)

// 2. Add Authorization Header (Bearer Token from Frontend)
req.Header.Set("Authorization", "Bearer "+accessToken)

// 3. Execute via eve-esi-client (handles Rate Limiting, Caching, Retries)
resp, err := s.esiClient.Do(req)
defer resp.Body.Close()

// 4. Handle HTTP Status Codes
if resp.StatusCode == 401 || resp.StatusCode == 403 {
    return ErrUnauthorized // Specific error for auth failures
}
if resp.StatusCode != http.StatusOK {
    return fmt.Errorf("ESI status %d", resp.StatusCode)
}

// 5. Parse JSON Response
var result ESIResponse
json.NewDecoder(resp.Body).Decode(&result)
```

**Trade-offs:**

- **Akzeptiert:** eve-esi-client Dependency (stabil, aktiv maintained)
- **Akzeptiert:** HTTP Request Konstruktion (minimaler Overhead)
- **Abgelehnt:** Generierter Client (zu viel Code, schwer wartbar)

**Annahmen:**

- eve-esi-client bleibt API-stabil (SemVer Compliance)
- ESI API Spec ändert sich selten (Breaking Changes)
- OAuth Token Management erfolgt im Frontend (ADR-004)

## Konsequenzen

### Positiv

- **Konsistenz:** Alle Services nutzen gleiches ESI Pattern
- **Rate Limiting Enforcement:** Automatisch über eve-esi-client
- **Caching Efficiency:** Redis Integration für ESI Cache Headers (ADR-009)
- **Monitoring:** Konsolidierte Metrics für ESI Performance
- **Graceful Degradation:** Standardisierte Fallback-Strategie (Default Values)
- **Maintainability:** Ein Package für Bugfixes/Improvements

### Negativ

- **Dependency:** eve-esi-client muss gepflegt werden
- **HTTP Konstruktion:** Manuelles Request Building (nicht generiert)
- **Abstraktionsschicht:** Zusätzliche Indirektion gegenüber stdlib

### Risiken

- **eve-esi-client Breaking Changes:**
  - **Mitigation:** SemVer Pinning in go.mod, langsame Upgrades
- **ESI API Changes:**
  - **Mitigation:** Versionierte Endpoints (/v4/), graduelle Migration
- **Rate Limit Exhaustion:**
  - **Mitigation:** Monitoring via Prometheus, Alerts bei >80% Rate Limit

## Implementierung

**Standard Pattern (Alle Services):**

```go
// Service Struct
type SkillsService struct {
    esiClient   *esiclient.Client  // eve-esi-client
    redisClient *redis.Client      // Caching Layer
    logger      *logger.Logger
}

// Constructor
func NewSkillsService(
    esiClient *esiclient.Client,
    redisClient *redis.Client,
    logger *logger.Logger,
) SkillsServicer {
    return &SkillsService{...}
}

// ESI Fetch Method
func (s *SkillsService) fetchFromESI(ctx context.Context, characterID int, accessToken string) (*ESIResponse, error) {
    // 1. Build Request
    endpoint := fmt.Sprintf("/v4/characters/%d/skills/", characterID)
    req, err := http.NewRequestWithContext(ctx, "GET", "https://esi.evetech.net"+endpoint, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }
    
    // 2. Authorization Header
    req.Header.Set("Authorization", "Bearer "+accessToken)
    
    // 3. Execute via eve-esi-client (Rate Limiting + Caching + Retries)
    resp, err := s.esiClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("esi request failed: %w", err)
    }
    defer resp.Body.Close()
    
    // 4. Handle Auth Errors (401/403)
    if resp.StatusCode == 401 || resp.StatusCode == 403 {
        return nil, fmt.Errorf("unauthorized: status %d", resp.StatusCode)
    }
    
    // 5. Handle Other Errors
    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("ESI status %d: %s", resp.StatusCode, string(body))
    }
    
    // 6. Parse JSON
    var result ESIResponse
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }
    
    return &result, nil
}
```

**Graceful Degradation Pattern:**

```go
func (s *SkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
    // 1. Try Redis Cache
    cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
    if err == nil {
        return unmarshalCached(cachedData), nil
    }
    
    // 2. Fetch from ESI
    esiData, err := s.fetchFromESI(ctx, characterID, accessToken)
    if err != nil {
        s.logger.Error("ESI fetch failed - using defaults", "error", err)
        // Graceful Degradation: Return default values (worst-case scenario)
        return s.getDefaultSkills(), nil
    }
    
    // 3. Cache Result (5min TTL)
    s.redisClient.Set(ctx, cacheKey, marshal(esiData), 5*time.Minute)
    
    return esiData, nil
}

// Default values ensure service continues working
func (s *SkillsService) getDefaultSkills() *TradingSkills {
    return &TradingSkills{
        Accounting:      0,  // Worst-case: highest tax
        BrokerRelations: 0,  // Worst-case: highest broker fee
        // ... all skills = 0
    }
}
```

**Error Categorization:**

| HTTP Status | Interpretation | Action |
|-------------|----------------|--------|
| 200 OK | Success | Parse response |
| 304 Not Modified | Cache valid | Use cached data (eve-esi-client handles this) |
| 401 Unauthorized | Invalid/Expired Token | Return `ErrUnauthorized`, Frontend handles re-auth |
| 403 Forbidden | Insufficient Scopes | Return `ErrForbidden`, log scope issue |
| 404 Not Found | Resource doesn't exist | Return `ErrNotFound` (e.g., deleted character) |
| 420 Error Limited | Rate Limit Exceeded | eve-esi-client blocks, wait for rate limit reset |
| 5xx Server Error | ESI Instability | Retry with backoff (eve-esi-client), fallback to defaults |

**Aufwand:** 1 PT (Pattern bereits etabliert, nur Dokumentation)

**Abhängigkeiten:**

- ADR-004: Frontend OAuth PKCE Flow (Bearer Token)
- ADR-009: Shared Redis Infrastructure (Caching)
- ADR-013: Timeout Handling (15s max per ESI request)
- eve-esi-client Package (externe Dependency)

**Validierung:**

- Erfolg gemessen an:
  - Alle Services nutzen gleiches Pattern (Code Reviews)
  - Rate Limit nie überschritten (Prometheus Metrics)
  - Graceful Degradation bei ESI Ausfällen (Integration Tests)

**Migrations-Pfad:**

1. ✅ Skills Service (#54) - Pattern etabliert
2. 🔜 Fee Service (#55) - Pattern übernehmen
3. 🔜 Cargo Service (#56) - Pattern übernehmen
4. 🔜 Assets Service (zukünftig) - Pattern übernehmen

## Referenzen

- **Issues:** #54 (Skills Service), #55 (Fee Service), #56 (Cargo Service)
- **ADRs:**
  - ADR-004 (Frontend OAuth PKCE Flow)
  - ADR-009 (Shared Redis Infrastructure)
  - ADR-012 (Redis Caching Strategy)
  - ADR-013 (Timeout Handling)
- **Externe Docs:**
  - [EVE ESI API](https://esi.evetech.net/ui/)
  - [eve-esi-client Package](https://github.com/Sternrassler/eve-esi-client)
  - [ESI Rate Limiting](https://docs.esi.evetech.net/docs/rate_limiting)

## Notizen

**Rate Limiting Details:**

- **ESI Limits:** 150 req/s burst, 20 req/s sustained
- **eve-esi-client Enforcement:** Token Bucket Algorithm (Thread-Safe)
- **Monitoring:** Prometheus `esi_requests_total`, `esi_rate_limit_exceeded_total`

**Caching Strategy:**

- **Character Skills:** 5min TTL (semi-static data)
- **Market Orders:** 5min TTL (ESI updates every ~5min)
- **Static Data (SDE):** No ESI cache (SQLite local)

**Known Deviations:**

- Trading Service nutzt abstrahierten `ESIMarketClient` Interface (statt direktem eve-esi-client) - erwägen Migration auf direktes Pattern
- Zukünftige Services MÜSSEN direktes eve-esi-client Pattern verwenden

**Security:**

- Bearer Token nur im Memory (nie persistiert)
- Redis Connection über TLS (ADR-009)
- ESI Requests über HTTPS (enforced)

**Testing:**

- Unit Tests: Mock `eve-esi-client.Client` Interface
- Integration Tests: Mock ESI Endpoints (httptest)
- E2E Tests: Nutzen ESI Test Server (Singularity)

---

**Change Log:**

- 2025-11-06: Status auf Accepted gesetzt (GitHub Copilot, aus Issue #54 extrahiert)
