# API Contract Testing Skill

**Purpose:** Validate API contracts between services, ensure backward compatibility, and prevent breaking changes through systematic contract testing.

---

## Architecture Overview

### Contract Testing Types

1. **Consumer-Driven Contracts**: Consumers define expectations
2. **Provider Contracts**: Provider publishes contract
3. **Schema Validation**: Validate against OpenAPI/Swagger specs
4. **Snapshot Testing**: Detect unintended API changes

### When to Use Each

- **Consumer-Driven**: Microservices, multiple API consumers
- **Provider Contracts**: Single provider, multiple versions
- **Schema Validation**: REST APIs with OpenAPI specs
- **Snapshot Testing**: Detect accidental changes in stable APIs

---

## Architecture Patterns

### 1. OpenAPI Schema Validation

**Principle:** Validate requests/responses against OpenAPI spec

```go
import (
    "github.com/getkin/kin-openapi/openapi3"
    "github.com/getkin/kin-openapi/openapi3filter"
)

func TestAPIContractCompliance(t *testing.T) {
    // Load OpenAPI spec
    loader := openapi3.NewLoader()
    doc, err := loader.LoadFromFile("openapi.yaml")
    require.NoError(t, err)
    
    // Create router from spec
    router, err := openapi3filter.NewRouter().WithSwagger(doc)
    require.NoError(t, err)
    
    // Test request/response
    httpReq := httptest.NewRequest("GET", "/api/markets", nil)
    route, pathParams, err := router.FindRoute(httpReq)
    require.NoError(t, err)
    
    // Validate request
    requestValidationInput := &openapi3filter.RequestValidationInput{
        Request:    httpReq,
        PathParams: pathParams,
        Route:      route,
    }
    
    err = openapi3filter.ValidateRequest(context.Background(), requestValidationInput)
    assert.NoError(t, err, "Request should match OpenAPI spec")
}
```

### 2. Consumer-Driven Contract (Pact Pattern)

**Principle:** Consumer defines expectations, provider verifies

```typescript
// Consumer: Define expected interaction
import { pact } from '@pact-foundation/pact';

describe('Market API Consumer', () => {
  const provider = pact({
    consumer: 'Frontend',
    provider: 'MarketAPI',
  });
  
  beforeAll(() => provider.setup());
  afterAll(() => provider.finalize());
  
  test('gets market orders', async () => {
    // Define expectation
    await provider.addInteraction({
      state: 'market has orders',
      uponReceiving: 'a request for market orders',
      withRequest: {
        method: 'GET',
        path: '/api/markets/10000002/orders/34',
        headers: {
          'Accept': 'application/json',
        },
      },
      willRespondWith: {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
        body: {
          orders: pact.eachLike({
            order_id: 123456,
            price: 100.50,
            volume_remain: 10,
          }),
        },
      },
    });
    
    // Verify against real client
    const client = new MarketAPIClient(provider.mockService.baseUrl);
    const orders = await client.getOrders(10000002, 34);
    
    expect(orders).toHaveLength(1);
    expect(orders[0]).toHaveProperty('order_id');
  });
});
```

### 3. Response Schema Validation

**Principle:** Ensure response structure matches contract

```go
func TestMarketOrdersResponseSchema(t *testing.T) {
    // Define expected schema
    schema := map[string]interface{}{
        "type": "object",
        "required": []string{"orders", "total"},
        "properties": map[string]interface{}{
            "orders": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{
                    "type": "object",
                    "required": []string{"order_id", "price", "volume_remain"},
                    "properties": map[string]interface{}{
                        "order_id":      map[string]string{"type": "integer"},
                        "price":         map[string]string{"type": "number"},
                        "volume_remain": map[string]string{"type": "integer"},
                    },
                },
            },
            "total": map[string]string{"type": "integer"},
        },
    }
    
    // Make API request
    resp := makeAPIRequest(t, "GET", "/api/markets/10000002/orders/34")
    
    // Validate response against schema
    validator := jsonschema.NewValidator(schema)
    err := validator.Validate(resp.Body)
    assert.NoError(t, err, "Response should match schema")
}
```

### 4. Backward Compatibility Testing

**Principle:** Ensure new versions don't break old consumers

```go
func TestBackwardCompatibility(t *testing.T) {
    tests := []struct {
        name       string
        apiVersion string
        request    string
        validate   func(*testing.T, *http.Response)
    }{
        {
            name:       "v1 still works",
            apiVersion: "v1",
            request:    "/api/v1/markets",
            validate: func(t *testing.T, resp *http.Response) {
                assert.Equal(t, 200, resp.StatusCode)
                // Validate v1 response format
            },
        },
        {
            name:       "v2 is compatible",
            apiVersion: "v2",
            request:    "/api/v2/markets",
            validate: func(t *testing.T, resp *http.Response) {
                assert.Equal(t, 200, resp.StatusCode)
                // Validate v2 includes all v1 fields
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp := makeAPIRequest(t, "GET", tt.request)
            tt.validate(t, resp)
        })
    }
}
```

---

## Best Practices

### 1. Contract First Development

**Define contracts before implementation**

```yaml
# openapi.yaml
openapi: 3.0.0
info:
  title: Market API
  version: 1.0.0

paths:
  /api/markets/{region_id}/orders/{type_id}:
    get:
      summary: Get market orders
      parameters:
        - name: region_id
          in: path
          required: true
          schema:
            type: integer
        - name: type_id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/MarketOrders'
        '404':
          description: Market not found

components:
  schemas:
    MarketOrders:
      type: object
      required:
        - orders
        - total
      properties:
        orders:
          type: array
          items:
            $ref: '#/components/schemas/Order'
        total:
          type: integer
```

### 2. Version Your Contracts

**Use semantic versioning for contract changes**

- **MAJOR**: Breaking changes (remove field, change type)
- **MINOR**: Backward-compatible additions (new optional field)
- **PATCH**: Documentation, examples (no schema change)

### 3. Test All Response Codes

**Don't just test happy path**

```go
func TestAPIErrorContracts(t *testing.T) {
    tests := []struct {
        name           string
        request        *http.Request
        expectedStatus int
        validateError  func(*testing.T, map[string]interface{})
    }{
        {
            name:           "404 Not Found",
            request:        httptest.NewRequest("GET", "/api/markets/999999/orders/34", nil),
            expectedStatus: 404,
            validateError: func(t *testing.T, body map[string]interface{}) {
                assert.Equal(t, "market_not_found", body["error_code"])
                assert.Contains(t, body["message"], "Market 999999 not found")
            },
        },
        {
            name:           "400 Bad Request",
            request:        httptest.NewRequest("GET", "/api/markets/invalid/orders/34", nil),
            expectedStatus: 400,
            validateError: func(t *testing.T, body map[string]interface{}) {
                assert.Equal(t, "invalid_parameter", body["error_code"])
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            resp := executeRequest(tt.request)
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
            
            var body map[string]interface{}
            json.NewDecoder(resp.Body).Decode(&body)
            tt.validateError(t, body)
        })
    }
}
```

### 4. Validate Request Contracts

**Test that API rejects invalid requests**

```go
func TestRequestValidation(t *testing.T) {
    tests := []struct {
        name    string
        payload interface{}
        wantErr string
    }{
        {
            name:    "missing required field",
            payload: map[string]interface{}{"price": 100},
            wantErr: "volume_remain is required",
        },
        {
            name:    "invalid type",
            payload: map[string]interface{}{"price": "invalid", "volume_remain": 10},
            wantErr: "price must be a number",
        },
        {
            name:    "out of range",
            payload: map[string]interface{}{"price": -10, "volume_remain": 10},
            wantErr: "price must be positive",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            body, _ := json.Marshal(tt.payload)
            req := httptest.NewRequest("POST", "/api/orders", bytes.NewReader(body))
            resp := executeRequest(req)
            
            assert.Equal(t, 400, resp.StatusCode)
            
            var errResp map[string]interface{}
            json.NewDecoder(resp.Body).Decode(&errResp)
            assert.Contains(t, errResp["message"], tt.wantErr)
        })
    }
}
```

### 5. Document Breaking Changes

**Maintain changelog for contract changes**

```markdown
# API Contract Changelog

## [2.0.0] - 2025-01-15

### BREAKING CHANGES
- Removed `legacy_id` field from Order response
- Changed `price` type from string to number
- Renamed endpoint `/markets` to `/market-data`

### Migration Guide
- Use `order_id` instead of `legacy_id`
- Parse `price` as float64, not string
- Update API base path

## [1.1.0] - 2024-12-01

### Added (Backward Compatible)
- Optional `metadata` field in Order response
- New query parameter `include_history`
```

### 6. Cross-Service Contract Testing

**Test integration between services**

```go
func TestESIClientContract(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Test against real ESI API
    client := NewESIClient("https://esi.evetech.net/latest")
    
    // Verify contract expectations
    orders, resp, err := client.GetMarketOrders(10000002, 34)
    require.NoError(t, err)
    
    // Validate response structure
    assert.Equal(t, 200, resp.StatusCode)
    assert.NotEmpty(t, orders)
    
    // Validate fields exist
    for _, order := range orders {
        assert.NotZero(t, order.OrderID)
        assert.NotZero(t, order.Price)
        assert.NotZero(t, order.VolumeRemain)
        assert.NotEmpty(t, order.LocationID)
        assert.Contains(t, []string{"buy", "sell"}, order.IsBuyOrder)
    }
}
```

---

## Common Patterns

### Pattern 1: ESI API Contract Validation (Project-Specific)

```go
func TestESIMarketOrdersContract(t *testing.T) {
    // Define expected ESI contract
    expectedFields := []string{
        "order_id",
        "type_id",
        "location_id",
        "volume_total",
        "volume_remain",
        "min_volume",
        "price",
        "is_buy_order",
        "duration",
        "issued",
        "range",
    }
    
    // Mock ESI response
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Return contract-compliant response
        orders := []map[string]interface{}{
            {
                "order_id":      int64(123456),
                "type_id":       34,
                "location_id":   int64(60003760),
                "volume_total":  100,
                "volume_remain": 50,
                "min_volume":    1,
                "price":         100.50,
                "is_buy_order":  false,
                "duration":      90,
                "issued":        "2025-01-01T00:00:00Z",
                "range":         "region",
            },
        }
        json.NewEncoder(w).Encode(orders)
    }))
    defer mockServer.Close()
    
    // Test client against contract
    client := NewESIClient(mockServer.URL)
    orders, _, err := client.GetMarketOrders(10000002, 34)
    require.NoError(t, err)
    require.NotEmpty(t, orders)
    
    // Validate all expected fields are present
    orderJSON, _ := json.Marshal(orders[0])
    var orderMap map[string]interface{}
    json.Unmarshal(orderJSON, &orderMap)
    
    for _, field := range expectedFields {
        assert.Contains(t, orderMap, field, "ESI contract requires field: %s", field)
    }
}
```

### Pattern 2: GraphQL Contract Testing

```typescript
import { graphql } from 'graphql';
import { makeExecutableSchema } from '@graphql-tools/schema';

describe('GraphQL Schema Contract', () => {
  const typeDefs = `
    type Query {
      market(regionId: Int!, typeId: Int!): Market
    }
    
    type Market {
      orders: [Order!]!
      totalVolume: Int!
    }
    
    type Order {
      orderId: ID!
      price: Float!
      volumeRemain: Int!
    }
  `;
  
  const schema = makeExecutableSchema({ typeDefs, resolvers });
  
  test('query returns contract-compliant structure', async () => {
    const query = `
      query {
        market(regionId: 10000002, typeId: 34) {
          orders {
            orderId
            price
            volumeRemain
          }
          totalVolume
        }
      }
    `;
    
    const result = await graphql({ schema, source: query });
    
    expect(result.errors).toBeUndefined();
    expect(result.data.market).toBeDefined();
    expect(result.data.market.orders).toBeInstanceOf(Array);
    expect(result.data.market.totalVolume).toBeGreaterThan(0);
  });
});
```

### Pattern 3: Contract Snapshot Testing

```go
func TestAPIResponseSnapshot(t *testing.T) {
    // Load golden snapshot
    goldenFile := "testdata/market_orders.golden.json"
    
    // Make API request
    resp := makeAPIRequest(t, "GET", "/api/markets/10000002/orders/34")
    
    var actual map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&actual)
    
    // Compare with snapshot (using testify/require)
    if *updateGolden {
        // Update snapshot
        golden, _ := json.MarshalIndent(actual, "", "  ")
        os.WriteFile(goldenFile, golden, 0644)
    } else {
        // Validate against snapshot
        golden, err := os.ReadFile(goldenFile)
        require.NoError(t, err)
        
        var expected map[string]interface{}
        json.Unmarshal(golden, &expected)
        
        assert.Equal(t, expected, actual, "Response should match snapshot")
    }
}

// Run with: go test -update-golden to update snapshots
```

---

## Anti-Patterns

### 1. Testing Implementation, Not Contract
❌ **Avoid:** Testing internal logic instead of API surface
✅ **Do:** Test only public API contract

### 2. Brittle Contract Tests
❌ **Avoid:** Asserting exact response values
✅ **Do:** Validate structure, types, required fields

### 3. No Versioning Strategy
❌ **Avoid:** Breaking changes without version bump
✅ **Do:** Use semantic versioning, deprecation warnings

### 4. Missing Error Contract Tests
❌ **Avoid:** Only testing 200 OK responses
✅ **Do:** Test all error codes (400, 404, 500, etc.)

### 5. Ignoring Consumer Needs
❌ **Avoid:** Provider-only contract definition
✅ **Do:** Consumer-driven contracts where applicable

---

## Integration Workflows

### With CI/CD

```yaml
# GitHub Actions: Contract validation
name: Contract Tests

on: [pull_request]

jobs:
  validate-contracts:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Validate OpenAPI spec
        run: |
          npx @redocly/cli lint openapi.yaml
      
      - name: Run contract tests
        run: go test -tags=contract ./...
      
      - name: Check backward compatibility
        run: |
          git fetch origin main
          npx openapi-diff origin/main:openapi.yaml openapi.yaml
```

### With Pact Broker

```typescript
// Publish contract to Pact Broker
afterAll(async () => {
  await provider.finalize();
  
  const opts = {
    pactBroker: 'https://pact-broker.example.com',
    pactBrokerToken: process.env.PACT_BROKER_TOKEN,
    consumerVersion: process.env.GIT_SHA,
    tags: ['main', 'production'],
  };
  
  await publishPacts(opts);
});
```

---

## Quick Reference

### OpenAPI Validation Commands

```bash
# Validate OpenAPI spec
npx @redocly/cli lint openapi.yaml

# Generate docs from spec
npx @redocly/cli build-docs openapi.yaml

# Check breaking changes
npx openapi-diff old.yaml new.yaml

# Generate Go server stub
oapi-codegen -generate types,server openapi.yaml > api.gen.go
```

### Contract Testing Checklist

- [ ] All endpoints have OpenAPI definitions
- [ ] Request validation tests (400 errors)
- [ ] Response schema validation (200, 404, 500)
- [ ] Backward compatibility tests
- [ ] Error format consistency
- [ ] API versioning strategy
- [ ] Contract changelog maintained
- [ ] Consumer contracts verified (if applicable)

---

This skill ensures API reliability through systematic contract validation. Use appropriate patterns based on your architecture (REST, GraphQL, gRPC).
