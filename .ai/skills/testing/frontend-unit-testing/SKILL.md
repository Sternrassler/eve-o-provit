# Frontend Unit Testing Skill

**Tech Stack:** React Testing Library + Vitest + Next.js 16 + React 19

**Purpose:** Comprehensive unit testing patterns for React components, hooks, and utilities using modern testing tools.

---

## Architecture Overview

### Test Types Distribution

- **Component Tests (60%)**: UI components, user interactions, accessibility
- **Hook Tests (20%)**: Custom React hooks, state management
- **Utility Tests (15%)**: Helper functions, data transformations
- **Integration Tests (5%)**: Component compositions, context providers

### When to Use Each

- **Component Tests**: Any UI component with user interactions
- **Hook Tests**: Custom hooks with complex state logic
- **Utility Tests**: Pure functions, calculations, formatters
- **Integration Tests**: Multi-component flows, context consumers

---

## Architecture Patterns

### 1. Component Testing Pattern

**Principle:** Test user behavior, not implementation details

```typescript
// GOOD: Test what user sees and does
test('displays error message when form submission fails', async () => {
  const user = userEvent.setup()
  render(<LoginForm />)
  
  await user.type(screen.getByLabelText(/email/i), 'invalid@test')
  await user.click(screen.getByRole('button', { name: /login/i }))
  
  expect(await screen.findByRole('alert')).toHaveTextContent('Invalid credentials')
})

// BAD: Test implementation details
test('sets error state to true', () => {
  const { rerender } = render(<LoginForm />)
  // Testing internal state - brittle!
})
```

### 2. Hook Testing Pattern

**Principle:** Test hooks in isolation with renderHook

```typescript
test('useAuth returns user after successful login', async () => {
  const { result } = renderHook(() => useAuth())
  
  await act(async () => {
    await result.current.login('user@test.com', 'password')
  })
  
  expect(result.current.user).toEqual({
    email: 'user@test.com',
    name: 'Test User'
  })
})
```

### 3. Accessibility Testing Pattern

**Principle:** Test with screen reader queries, validate ARIA

```typescript
test('navigation menu is keyboard accessible', async () => {
  const user = userEvent.setup()
  render(<NavigationMenu />)
  
  const trigger = screen.getByRole('button', { name: /menu/i })
  await user.click(trigger)
  
  const menu = screen.getByRole('menu')
  expect(menu).toBeVisible()
  
  // Test keyboard navigation
  await user.keyboard('{ArrowDown}')
  expect(screen.getByRole('menuitem', { name: /home/i })).toHaveFocus()
})
```

---

## Best Practices

### 1. Query Priority (Testing Library)

**Use in this order:**

1. `getByRole` - Accessibility-first (preferred)
2. `getByLabelText` - Forms (user-centric)
3. `getByPlaceholderText` - Form inputs (last resort)
4. `getByText` - Non-interactive content
5. `getByTestId` - Only when nothing else works

**Avoid:** `getByClassName`, `querySelector` - Implementation details

### 2. User Interaction Simulation

**Always use `@testing-library/user-event`**, not `fireEvent`

```typescript
// GOOD: Realistic user interaction
const user = userEvent.setup()
await user.type(input, 'Hello')
await user.click(button)

// BAD: Synthetic events
fireEvent.change(input, { target: { value: 'Hello' }})
fireEvent.click(button)
```

### 3. Async Testing

#### Wait for elements, don't use arbitrary timeouts

```typescript
// GOOD: Wait for specific condition
expect(await screen.findByText('Success')).toBeInTheDocument()

// BAD: Arbitrary wait
await waitFor(() => {}, { timeout: 3000 })
```

### 4. Mock External Dependencies

#### Mock API calls, not React components

```typescript
// Setup MSW (Mock Service Worker)
const server = setupServer(
  http.get('/api/user', () => {
    return HttpResponse.json({ name: 'Test User' })
  })
)

beforeAll(() => server.listen())
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
```

### 5. Component Isolation

#### Provide necessary context/providers

```typescript
function renderWithProviders(ui: ReactElement) {
  return render(
    <AuthProvider>
      <ToastProvider>
        {ui}
      </ToastProvider>
    </AuthProvider>
  )
}

test('component uses auth context', () => {
  renderWithProviders(<ProtectedPage />)
  // Test with full context
})
```

### 6. Test Organization

**Structure:** Arrange-Act-Assert (AAA)

```typescript
test('adds item to cart', async () => {
  // Arrange: Setup test state
  const user = userEvent.setup()
  render(<ProductCard product={mockProduct} />)
  
  // Act: Perform user action
  await user.click(screen.getByRole('button', { name: /add to cart/i }))
  
  // Assert: Verify outcome
  expect(screen.getByText('1 item in cart')).toBeInTheDocument()
})
```

### 7. Snapshot Testing (Use Sparingly)

**Only for:**

- Static content (error pages, legal text)
- Complex data structures
- Generated markup validation

**Never for:** Interactive components, business logic

---

## Common Patterns

### Pattern 1: Form Validation Testing

```typescript
describe('ContactForm', () => {
  test('shows validation errors for empty required fields', async () => {
    const user = userEvent.setup()
    render(<ContactForm />)
    
    await user.click(screen.getByRole('button', { name: /submit/i }))
    
    expect(screen.getByText(/email is required/i)).toBeInTheDocument()
    expect(screen.getByText(/message is required/i)).toBeInTheDocument()
  })
  
  test('submits form with valid data', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()
    render(<ContactForm onSubmit={onSubmit} />)
    
    await user.type(screen.getByLabelText(/email/i), 'test@example.com')
    await user.type(screen.getByLabelText(/message/i), 'Hello world')
    await user.click(screen.getByRole('button', { name: /submit/i }))
    
    expect(onSubmit).toHaveBeenCalledWith({
      email: 'test@example.com',
      message: 'Hello world'
    })
  })
})
```

### Pattern 2: Custom Hook Testing

```typescript
describe('useLocalStorage', () => {
  beforeEach(() => {
    localStorage.clear()
  })
  
  test('stores and retrieves value', () => {
    const { result } = renderHook(() => useLocalStorage('key', 'default'))
    
    expect(result.current[0]).toBe('default')
    
    act(() => {
      result.current[1]('new value')
    })
    
    expect(result.current[0]).toBe('new value')
    expect(localStorage.getItem('key')).toBe('"new value"')
  })
  
  test('handles invalid JSON gracefully', () => {
    localStorage.setItem('key', 'invalid-json')
    
    const { result } = renderHook(() => useLocalStorage('key', 'default'))
    
    expect(result.current[0]).toBe('default')
  })
})
```

### Pattern 3: Radix UI Component Testing

```typescript
describe('Dialog Component', () => {
  test('opens and closes dialog', async () => {
    const user = userEvent.setup()
    render(<DialogDemo />)
    
    // Dialog closed initially
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    
    // Open dialog
    await user.click(screen.getByRole('button', { name: /open/i }))
    expect(screen.getByRole('dialog')).toBeVisible()
    
    // Close with button
    await user.click(screen.getByRole('button', { name: /close/i }))
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })
  
  test('closes on escape key', async () => {
    const user = userEvent.setup()
    render(<DialogDemo />)
    
    await user.click(screen.getByRole('button', { name: /open/i }))
    await user.keyboard('{Escape}')
    
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })
})
```

---

## Anti-Patterns

### 1. Testing Implementation Details

❌ **Avoid:** Testing component internal state, private methods
✅ **Do:** Test observable behavior from user perspective

### 2. Brittle Selectors

❌ **Avoid:** `container.querySelector('.css-class')`
✅ **Do:** `screen.getByRole('button', { name: /submit/i })`

### 3. Not Cleaning Up

❌ **Avoid:** Leaving mocks/timers active between tests
✅ **Do:** Use `afterEach(() => { vi.clearAllMocks() })`

### 4. Testing Library Internals

❌ **Avoid:** Testing React Testing Library helpers
✅ **Do:** Test your application code

### 5. Over-Mocking

❌ **Avoid:** Mocking everything including UI components
✅ **Do:** Mock external dependencies (API, localStorage, etc.)

### 6. Shallow Rendering

❌ **Avoid:** Using enzyme-style shallow rendering
✅ **Do:** Full render with React Testing Library

---

## Integration Workflows

### With Next.js App Router

```typescript
// Test Server Component (render in Node.js environment)
test('ServerComponent fetches and displays data', async () => {
  const component = await ServerComponent()
  const { container } = render(component)
  
  expect(container).toHaveTextContent('Server Data')
})

// Test Client Component
test('ClientComponent handles user interaction', async () => {
  render(<ClientComponent />)
  // Standard client testing
})
```

### With Vitest Configuration

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./tests/setup.ts'],
    globals: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      exclude: ['node_modules/', 'tests/']
    }
  }
})
```

### With MSW (Mock Service Worker)

```typescript
// tests/setup.ts
import { afterAll, afterEach, beforeAll } from 'vitest'
import { setupServer } from 'msw/node'
import { handlers } from './handlers'

export const server = setupServer(...handlers)

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())
```

---

## Coverage Goals & Strategy

### Target Coverage (Frontend)

| Area | Current | Q1 Target | Q2 Target |
|------|---------|-----------|-----------|
| Components | 0% | 70% | 85% |
| Hooks | 0% | 80% | 90% |
| Utils | 0% | 85% | 95% |
| Overall | 0% | 70% | 85% |

### Priority Order

1. **Authentication Components** (Login, Register, Protected Routes)
2. **Core UI Components** (Navigation, Dialogs, Forms)
3. **Custom Hooks** (useAuth, useToast, useLocalStorage)
4. **Business Logic Utils** (formatters, validators, calculators)

---

## Quick Reference

### Essential Commands

```bash
# Run all unit tests
npm run test

# Run with coverage
npm run test:coverage

# Run in watch mode
npm run test:watch

# Run specific test file
npm run test src/components/Button.test.tsx

# Update snapshots
npm run test -- -u
```

### Common Assertions

```typescript
// Presence
expect(element).toBeInTheDocument()
expect(element).toBeVisible()

// Text content
expect(element).toHaveTextContent('Hello')
expect(element).toHaveAccessibleName('Submit')

// Attributes
expect(element).toHaveAttribute('aria-expanded', 'true')
expect(input).toHaveValue('test@example.com')

// Focus
expect(element).toHaveFocus()

// Disability
expect(button).toBeDisabled()
expect(button).toBeEnabled()
```

### Debugging Tips

```typescript
// Print current DOM
screen.debug()

// Print specific element
screen.debug(screen.getByRole('button'))

// Log available roles
screen.logTestingPlaygroundURL()

// Check what queries are available
screen.getByRole('') // Shows error with all roles
```

---

## Performance Considerations

### 1. Test Isolation

- Each test runs in clean environment
- No shared state between tests
- Fast test execution (< 100ms per test)

### 2. Parallel Execution

- Vitest runs tests in parallel by default
- Use `test.concurrent` for independent tests

### 3. Setup Optimization

- Shared test utilities in `tests/utils.tsx`
- Reusable mock handlers in `tests/handlers.ts`
- Common fixtures in `tests/fixtures.ts`

---

## Security Guidelines

### 1. Test Authentication Flows

- Login/logout cycles
- Token refresh handling
- Session expiration

### 2. Test Authorization

- Protected route access
- Role-based rendering
- Permission checks

### 3. Input Validation

- XSS prevention (escaped output)
- SQL injection (if direct queries)
- CSRF token handling

---

This skill provides comprehensive patterns for testing React applications with modern tools. All examples follow Testing Library principles: test user behavior, not implementation details.
