# Backend Skill: Fiber Web Framework

**Tech Stack:** Go 1.24 + Fiber v2.52.9

**Project:** eve-o-provit Backend API

---

## Architecture Patterns

### Layered Architecture

- **Handler Layer:** HTTP request/response handling (Fiber controllers)
- **Service Layer:** Business logic, orchestration (RouteCalculator, CharacterHelper)
- **Repository Layer:** Data access abstraction (MarketRepository, SDERepository)
- **Database Layer:** Dual-DB architecture (PostgreSQL + SQLite SDE)

### Dependency Injection

- Handlers receive repositories and services via constructor injection
- No global state, all dependencies explicit
- Testable through interface mocking

---

## Best Practices

1. **Handler Responsibility:** Parse request → Call service → Return response (no business logic)
2. **Context Propagation:** Always pass `context.Context` down the call chain
3. **Timeout Management:** Set explicit timeouts for long-running operations
4. **Error Handling:** Return structured errors, log details, return generic messages to client
5. **Middleware Usage:** Authentication, logging, metrics at middleware level
6. **Resource Cleanup:** Always defer closing resources (transactions, connections)
7. **Type Safety:** Strong types for requests/responses, avoid `map[string]interface{}`

---

## Common Patterns

### 1. Handler with Dependency Injection

```go
type Handler struct {
    marketRepo *database.MarketRepository
    sdeRepo    *database.SDERepository
    esiClient  *esi.Client
}

func New(marketRepo *database.MarketRepository, sdeRepo *database.SDERepository, esiClient *esi.Client) *Handler {
    return &Handler{
        marketRepo: marketRepo,
        sdeRepo:    sdeRepo,
        esiClient:  esiClient,
    }
}
```

### 2. Request Parsing with Validation

```go
func (h *Handler) GetMarketOrders(c *fiber.Ctx) error {
    regionID, err := c.ParamsInt("region")
    if err != nil {
        return fiber.NewError(fiber.StatusBadRequest, "invalid region ID")
    }
    
    refresh := c.QueryBool("refresh", false)
    
    // Call service layer
    orders, err := h.marketService.FetchOrders(c.Context(), regionID, refresh)
    if err != nil {
        return err // Fiber handles error responses
    }
    
    return c.JSON(orders)
}
```

### 3. Context Timeout Pattern

```go
func (h *Handler) CalculateRoutes(c *fiber.Ctx) error {
    ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
    defer cancel()
    
    routes, err := h.calculator.Calculate(ctx, regionID, shipTypeID, capacity)
    if errors.Is(err, context.DeadlineExceeded) {
        return fiber.NewError(fiber.StatusPartialContent, "calculation timeout, partial results")
    }
    return c.JSON(routes)
}
```

### 4. Structured Error Responses

```go
type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
}

func (h *Handler) errorResponse(c *fiber.Ctx, status int, msg string) error {
    return c.Status(status).JSON(ErrorResponse{
        Error:   fiber.ErrBadRequest.Error(),
        Message: msg,
    })
}
```

---

## Anti-Patterns

❌ **Business Logic in Handlers:** Handlers should not calculate, validate complex rules, or orchestrate multi-step workflows

❌ **Direct Database Access in Handlers:** Always go through repository layer

❌ **Ignoring Context Cancellation:** Not checking `ctx.Done()` in long operations leads to wasted resources

❌ **Panic in Handlers:** Use proper error returns, Fiber's middleware catches panics but shouldn't rely on it

❌ **Blocking Operations:** Avoid synchronous calls without timeouts (ESI API calls, database queries)

---

## Integration with Other Layers

### Service Layer Integration

- Handlers call service methods with context
- Services orchestrate multiple repositories
- Services handle retries, caching, and complex logic

### Repository Layer Integration

- Services call repository methods for data access
- Repositories return domain models, not raw SQL results
- Use transactions when modifying multiple tables

### ESI Client Integration

- Always use context with timeout
- Handle rate limiting at service level
- Cache ESI responses in Redis

---

## Performance Considerations

- **Connection Pooling:** Fiber reuses connections, don't create new clients per request
- **JSON Encoding:** Fiber's `c.JSON()` is optimized, avoid manual `json.Marshal`
- **Prefork Mode:** For CPU-bound operations, enable prefork in production
- **Middleware Order:** Place expensive middleware (auth checks) after cheap ones (CORS)

---

## Security Guidelines

- **Input Validation:** Always validate path params, query params, and request bodies
- **Authentication:** Use middleware for protected routes, extract token from Authorization header
- **Rate Limiting:** Implement per-IP or per-user rate limits
- **CORS Configuration:** Restrict origins in production, don't use wildcard `*`

---

## Quick Reference

| Task | Method |
|------|--------|
| Path parameter | `c.Params("id")` or `c.ParamsInt("id")` |
| Query parameter | `c.Query("key")` or `c.QueryBool("refresh")` |
| JSON body | `c.BodyParser(&struct{})` |
| JSON response | `c.JSON(data)` |
| Error response | `fiber.NewError(status, msg)` |
| Context | `c.Context()` (pass to services) |
| Middleware | `app.Use(middleware)` |
