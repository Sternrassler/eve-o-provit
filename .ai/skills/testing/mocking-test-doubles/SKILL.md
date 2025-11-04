# Mocking & Test Doubles Skill

**Purpose:** Systematic patterns for test doubles (mocks, stubs, fakes, spies) to isolate units under test and control external dependencies.

---

## Architecture Overview

### Test Double Types

1. **Stub**: Returns predefined values, no verification
2. **Mock**: Records calls, verifies interactions
3. **Fake**: Working implementation, simplified
4. **Spy**: Wraps real object, records calls

### When to Use Each

- **Stub**: Replace dependencies with fixed responses (database queries, API calls)
- **Mock**: Verify behavior interactions (message sending, event publishing)
- **Fake**: In-memory implementations (in-memory database, fake cache)
- **Spy**: Partial mocking (verify calls while using real implementation)

### Decision Tree

```
Need to verify calls? 
  ├─ Yes → Mock or Spy
  │   └─ Need real implementation? 
  │       ├─ Yes → Spy
  │       └─ No → Mock
  └─ No → Stub or Fake
      └─ Complex logic?
          ├─ Yes → Fake (working alternative)
          └─ No → Stub (simple return values)
```

---

## Architecture Patterns

### 1. Interface-Based Mocking (Go)

**Principle:** Define interfaces, inject dependencies

```go
// Production interface
type UserRepository interface {
    GetUser(ctx context.Context, id int) (*User, error)
    CreateUser(ctx context.Context, user *User) error
}

// Mock implementation
type MockUserRepository struct {
    GetUserFunc    func(ctx context.Context, id int) (*User, error)
    CreateUserFunc func(ctx context.Context, user *User) error
    Calls          struct {
        GetUser    []int
        CreateUser []*User
    }
}

func (m *MockUserRepository) GetUser(ctx context.Context, id int) (*User, error) {
    m.Calls.GetUser = append(m.Calls.GetUser, id)
    if m.GetUserFunc != nil {
        return m.GetUserFunc(ctx, id)
    }
    return nil, errors.New("not implemented")
}
```

### 2. HTTP Server Mocking (Go)

**Principle:** Use httptest.Server for controlled responses

```go
func TestAPIClient_FetchData(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request
        assert.Equal(t, "/api/data", r.URL.Path)
        assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
        
        // Return mock response
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]string{
            "result": "success"
        })
    }))
    defer server.Close()
    
    // Test with mock server URL
    client := NewAPIClient(server.URL)
    result, err := client.FetchData(context.Background())
    
    require.NoError(t, err)
    assert.Equal(t, "success", result)
}
```

### 3. Function Mocking (JavaScript/TypeScript)

**Principle:** Replace functions with Vitest mocks

```typescript
// Mock module
vi.mock('./api/client', () => ({
  fetchUser: vi.fn(),
  updateUser: vi.fn()
}))

import { fetchUser, updateUser } from './api/client'

test('component calls API with correct params', async () => {
  // Setup mock response
  vi.mocked(fetchUser).mockResolvedValue({
    id: 1,
    name: 'Test User'
  })
  
  // Test component
  render(<UserProfile userId={1} />)
  
  // Verify mock was called
  expect(fetchUser).toHaveBeenCalledWith(1)
  expect(await screen.findByText('Test User')).toBeInTheDocument()
})
```

---

## Best Practices

### 1. Minimal Mocking

**Mock only external boundaries, not internal logic**

```go
// GOOD: Mock external API
mockESI := &MockESIClient{
    FetchMarketOrdersFunc: func() ([]Order, error) {
        return testOrders, nil
    },
}

// BAD: Mock internal business logic
mockCalculator := &MockProfitCalculator{} // Should be real!
```

### 2. Behavior Verification vs. State Verification

**Prefer state verification over behavior verification**

```typescript
// GOOD: Verify state change
test('adds item to cart', () => {
  const cart = new ShoppingCart()
  cart.addItem({ id: 1, name: 'Product' })
  
  expect(cart.items).toHaveLength(1)
  expect(cart.total).toBe(10.00)
})

// BAD: Over-verify internal calls
test('calls internal method', () => {
  const spy = vi.spyOn(cart, 'calculateTotal')
  cart.addItem({ id: 1, name: 'Product' })
  
  expect(spy).toHaveBeenCalled() // Brittle!
})
```

### 3. Mock Lifecycle

**Setup in beforeEach, cleanup in afterEach**

```typescript
describe('UserService', () => {
  let mockRepository: MockRepository
  
  beforeEach(() => {
    mockRepository = {
      findUser: vi.fn(),
      saveUser: vi.fn()
    }
  })
  
  afterEach(() => {
    vi.clearAllMocks()
  })
  
  test('creates user', async () => {
    // Test uses fresh mock
  })
})
```

### 4. Realistic Test Data

**Use builders or fixtures for consistent test data**

```go
// Test data builder
func NewTestUser() *User {
    return &User{
        ID:    1,
        Email: "test@example.com",
        Name:  "Test User",
    }
}

func NewTestUserWithID(id int) *User {
    user := NewTestUser()
    user.ID = id
    return user
}

// Usage
func TestUserService(t *testing.T) {
    user := NewTestUser()
    // Test with consistent data
}
```

### 5. Avoid Over-Specification

**Don't verify every interaction**

```typescript
// GOOD: Verify critical behavior
expect(mockEmailService.sendWelcomeEmail).toHaveBeenCalledWith({
  to: 'user@example.com',
  subject: 'Welcome!'
})

// BAD: Over-specify
expect(mockLogger.debug).toHaveBeenCalledTimes(3)
expect(mockLogger.debug).toHaveBeenNthCalledWith(1, 'Starting')
expect(mockLogger.debug).toHaveBeenNthCalledWith(2, 'Processing')
// Too brittle!
```

### 6. Test Double Naming

**Clear naming conventions**

```go
// Prefix clearly indicates type
type MockUserRepository struct { }    // Mock: verifies interactions
type StubUserRepository struct { }    // Stub: returns fixed values
type FakeUserRepository struct { }    // Fake: working in-memory impl
type SpyUserRepository struct { }     // Spy: wraps real impl
```

---

## Common Patterns

### Pattern 1: ESI API Mocking (Project-Specific)

```go
// Mock ESI server for testing
func NewMockESIServer(t *testing.T) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch r.URL.Path {
        case "/markets/10000002/orders/":
            // Mock market orders response
            orders := []MarketOrder{
                {OrderID: 1, Price: 100.50, VolumeRemain: 10},
                {OrderID: 2, Price: 101.00, VolumeRemain: 5},
            }
            json.NewEncoder(w).Encode(orders)
            
        case "/markets/10000002/history/":
            // Mock price history
            w.WriteHeader(http.StatusOK)
            json.NewEncoder(w).Encode([]PriceHistory{})
            
        default:
            w.WriteHeader(http.StatusNotFound)
        }
    }))
}

// Usage in tests
func TestESIClient(t *testing.T) {
    server := NewMockESIServer(t)
    defer server.Close()
    
    client := NewESIClient(server.URL)
    orders, err := client.FetchMarketOrders(ctx, 10000002, 34)
    
    require.NoError(t, err)
    assert.Len(t, orders, 2)
}
```

### Pattern 2: Repository Stub Pattern

```go
// Stub with predefined responses
type StubMarketRepository struct {
    Orders []MarketOrder
    Err    error
}

func (s *StubMarketRepository) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]MarketOrder, error) {
    if s.Err != nil {
        return nil, s.Err
    }
    return s.Orders, nil
}

// Usage
func TestPriceCalculator(t *testing.T) {
    stub := &StubMarketRepository{
        Orders: []MarketOrder{
            {Price: 100.0, VolumeRemain: 10},
            {Price: 105.0, VolumeRemain: 5},
        },
    }
    
    calculator := NewPriceCalculator(stub)
    avgPrice := calculator.CalculateAveragePrice(ctx, 10000002, 34)
    
    assert.InDelta(t, 102.5, avgPrice, 0.1)
}
```

### Pattern 3: Fake In-Memory Implementation

```go
// Fake repository with in-memory storage
type FakeUserRepository struct {
    users map[int]*User
    mu    sync.RWMutex
}

func NewFakeUserRepository() *FakeUserRepository {
    return &FakeUserRepository{
        users: make(map[int]*User),
    }
}

func (f *FakeUserRepository) GetUser(ctx context.Context, id int) (*User, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()
    
    user, ok := f.users[id]
    if !ok {
        return nil, ErrUserNotFound
    }
    return user, nil
}

func (f *FakeUserRepository) CreateUser(ctx context.Context, user *User) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    f.users[user.ID] = user
    return nil
}

// Usage: Test complex multi-operation scenarios
func TestUserService_CompleteFlow(t *testing.T) {
    repo := NewFakeUserRepository()
    service := NewUserService(repo)
    
    // Create user
    user, err := service.Register(ctx, "test@example.com", "password")
    require.NoError(t, err)
    
    // Retrieve user
    retrieved, err := service.GetUser(ctx, user.ID)
    require.NoError(t, err)
    assert.Equal(t, user.Email, retrieved.Email)
}
```

### Pattern 4: Spy for Partial Mocking

```go
// Spy wraps real implementation
type SpyCache struct {
    realCache Cache
    getCalls  []string
    setCalls  []string
}

func (s *SpyCache) Get(key string) (interface{}, bool) {
    s.getCalls = append(s.getCalls, key)
    return s.realCache.Get(key)
}

func (s *SpyCache) Set(key string, value interface{}) {
    s.setCalls = append(s.setCalls, key)
    s.realCache.Set(key, value)
}

// Usage: Verify cache usage while testing real behavior
func TestService_UsesCache(t *testing.T) {
    spy := &SpyCache{realCache: NewRealCache()}
    service := NewService(spy)
    
    service.GetData("user:1")
    service.GetData("user:1") // Second call should hit cache
    
    assert.Equal(t, 2, len(spy.getCalls))
    // Real cache behavior still works
}
```

---

## Anti-Patterns

### 1. Mocking Everything

❌ **Avoid:** Mocking all dependencies
✅ **Do:** Mock external boundaries, use real objects for internal logic

### 2. Leaky Mocks

❌ **Avoid:** Mocks that know too much about implementation
✅ **Do:** Mocks that only verify interface contracts

### 3. Non-Deterministic Mocks

❌ **Avoid:** Mocks with random behavior
✅ **Do:** Predictable, repeatable mock responses

### 4. Global Mocks

❌ **Avoid:** Shared mocks between tests
✅ **Do:** Fresh mocks per test (setup in beforeEach)

### 5. Mock Verification Overload

❌ **Avoid:** Verifying every method call
✅ **Do:** Verify critical interactions only

---

## Integration Workflows

### With Testcontainers

```go
// Use real database for integration tests, mocks for unit tests
func TestUserRepository_Unit(t *testing.T) {
    // Use mock for fast unit test
    mockDB := &MockDatabase{}
    repo := NewUserRepository(mockDB)
    // Test repository logic
}

func TestUserRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Use real Testcontainer
    container := startPostgresContainer(t)
    defer container.Terminate(ctx)
    
    db := connectToContainer(container)
    repo := NewUserRepository(db)
    // Test with real database
}
```

### With MSW (Frontend)

```typescript
// Setup MSW handlers
export const handlers = [
  http.get('/api/users/:id', ({ params }) => {
    return HttpResponse.json({
      id: params.id,
      name: 'Test User'
    })
  }),
  
  http.post('/api/login', async ({ request }) => {
    const { email } = await request.json()
    
    if (email === 'valid@example.com') {
      return HttpResponse.json({ token: 'abc123' })
    }
    
    return new HttpResponse(null, { status: 401 })
  })
]

// Use in tests
test('login with valid credentials', async () => {
  const user = userEvent.setup()
  render(<LoginForm />)
  
  await user.type(screen.getByLabelText(/email/i), 'valid@example.com')
  await user.click(screen.getByRole('button', { name: /login/i }))
  
  expect(await screen.findByText('Welcome back')).toBeInTheDocument()
})
```

---

## Quick Reference

### Go Mock Patterns

```go
// Interface-based mock
type MockService struct {
    DoSomethingFunc func(ctx context.Context) error
}

// HTTP mock server
server := httptest.NewServer(handler)
defer server.Close()

// Fake in-memory
fake := NewFakeRepository()

// Spy with real implementation
spy := &Spy{real: NewRealService()}
```

### TypeScript Mock Patterns

```typescript
// Function mock
const mockFn = vi.fn()

// Module mock
vi.mock('./module')

// Spy on method
const spy = vi.spyOn(object, 'method')

// Mock implementation
vi.mocked(fetchUser).mockResolvedValue(data)

// Clear mocks
vi.clearAllMocks()
```

---

## Performance Considerations

### 1. Mock vs. Real Performance

- Mocks: Instant (no I/O)
- Fakes: Fast (in-memory)
- Real: Slow (actual I/O)

### 2. When to Use Real Dependencies

- Simple pure functions
- Critical integration paths
- Complex state machines

### 3. When to Mock

- External APIs
- Slow operations (DB, network)
- Non-deterministic behavior (time, random)

---

This skill provides systematic patterns for test isolation through mocking. Use the decision tree to choose the right test double type for each scenario.
