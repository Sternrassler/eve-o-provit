# Security Testing Skill

**Purpose:** Systematic security testing patterns to identify vulnerabilities, validate authentication/authorization, and ensure compliance with security best practices.

---

## Architecture Overview

### Security Test Types

1. **Authentication Testing**: Login, logout, session management
2. **Authorization Testing**: Role-based access control, permissions
3. **Input Validation**: Injection attacks, XSS, SQL injection
4. **Session Security**: Token validation, CSRF, session fixation
5. **API Security**: Rate limiting, CORS, security headers

### When to Use Each

- **Authentication**: Login flows, OAuth/SSO integration
- **Authorization**: Protected routes, role-based features
- **Input Validation**: Forms, API endpoints, user content
- **Session Security**: Token-based auth systems
- **API Security**: Public-facing APIs, third-party integrations

---

## Architecture Patterns

### 1. Authentication Flow Testing

**Principle:** Test complete auth lifecycle

```typescript
describe('Authentication Security', () => {
  test('successful login creates valid session', async () => {
    const user = userEvent.setup()
    render(<LoginForm />)
    
    await user.type(screen.getByLabelText(/email/i), 'user@example.com')
    await user.type(screen.getByLabelText(/password/i), 'SecurePass123!')
    await user.click(screen.getByRole('button', { name: /login/i }))
    
    // Verify token is stored securely
    expect(localStorage.getItem('token')).toBeNull() // NO localStorage!
    expect(document.cookie).toContain('session=') // httpOnly cookie
    
    // Verify redirects to protected page
    await waitFor(() => {
      expect(window.location.pathname).toBe('/dashboard')
    })
  })
  
  test('rejects invalid credentials', async () => {
    const user = userEvent.setup()
    render(<LoginForm />)
    
    await user.type(screen.getByLabelText(/email/i), 'user@example.com')
    await user.type(screen.getByLabelText(/password/i), 'WrongPassword')
    await user.click(screen.getByRole('button', { name: /login/i }))
    
    // Should NOT leak info about valid/invalid email
    expect(await screen.findByText(/invalid credentials/i)).toBeInTheDocument()
    expect(screen.queryByText(/user not found/i)).not.toBeInTheDocument()
  })
  
  test('prevents brute force attacks', async () => {
    const user = userEvent.setup()
    render(<LoginForm />)
    
    // Attempt multiple failed logins
    for (let i = 0; i < 5; i++) {
      await user.clear(screen.getByLabelText(/password/i))
      await user.type(screen.getByLabelText(/password/i), `wrong${i}`)
      await user.click(screen.getByRole('button', { name: /login/i }))
    }
    
    // Should be rate-limited
    expect(await screen.findByText(/too many attempts/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /login/i })).toBeDisabled()
  })
})
```

### 2. Authorization Testing (Go)

**Principle:** Verify access control at every level

```go
func TestAuthorizationMiddleware(t *testing.T) {
    tests := []struct {
        name           string
        token          string
        expectedStatus int
        expectedBody   string
    }{
        {
            name:           "valid admin token",
            token:          generateTestToken("admin", []string{"admin"}),
            expectedStatus: http.StatusOK,
        },
        {
            name:           "valid user token - insufficient permissions",
            token:          generateTestToken("user", []string{"user"}),
            expectedStatus: http.StatusForbidden,
            expectedBody:   "insufficient permissions",
        },
        {
            name:           "expired token",
            token:          generateExpiredToken(),
            expectedStatus: http.StatusUnauthorized,
            expectedBody:   "token expired",
        },
        {
            name:           "invalid signature",
            token:          "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid.signature",
            expectedStatus: http.StatusUnauthorized,
            expectedBody:   "invalid token",
        },
        {
            name:           "no token",
            token:          "",
            expectedStatus: http.StatusUnauthorized,
            expectedBody:   "missing authorization header",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            req := httptest.NewRequest("GET", "/api/admin/users", nil)
            if tt.token != "" {
                req.Header.Set("Authorization", "Bearer "+tt.token)
            }
            
            rec := httptest.NewRecorder()
            handler := AuthMiddleware(adminHandler)
            handler.ServeHTTP(rec, req)
            
            assert.Equal(t, tt.expectedStatus, rec.Code)
            if tt.expectedBody != "" {
                assert.Contains(t, rec.Body.String(), tt.expectedBody)
            }
        })
    }
}
```

### 3. Input Validation Testing

**Principle:** Test all injection vectors

```go
func TestInputValidation(t *testing.T) {
    tests := []struct {
        name          string
        input         string
        expectBlocked bool
    }{
        // SQL Injection attempts
        {name: "SQL injection - basic", input: "' OR '1'='1", expectBlocked: true},
        {name: "SQL injection - union", input: "' UNION SELECT * FROM users--", expectBlocked: true},
        
        // XSS attempts
        {name: "XSS - script tag", input: "<script>alert('XSS')</script>", expectBlocked: true},
        {name: "XSS - img onerror", input: "<img src=x onerror=alert(1)>", expectBlocked: true},
        {name: "XSS - event handler", input: "<div onclick='alert(1)'>Click</div>", expectBlocked: true},
        
        // Command injection
        {name: "Command injection", input: "; rm -rf /", expectBlocked: true},
        
        // Path traversal
        {name: "Path traversal", input: "../../etc/passwd", expectBlocked: true},
        
        // Valid inputs
        {name: "Valid text", input: "Hello World", expectBlocked: false},
        {name: "Valid email", input: "user@example.com", expectBlocked: false},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            sanitized := sanitizeInput(tt.input)
            
            if tt.expectBlocked {
                // Dangerous content should be escaped or rejected
                assert.NotEqual(t, tt.input, sanitized, "Dangerous input should be sanitized")
                assert.NotContains(t, sanitized, "<script>")
                assert.NotContains(t, sanitized, "' OR '")
            } else {
                // Valid content should pass through
                assert.Equal(t, tt.input, sanitized)
            }
        })
    }
}
```

### 4. OAuth/SSO Security Testing (Project-Specific: EVE SSO)

**Principle:** Validate complete OAuth flow security

```go
func TestEVESSOSecurity(t *testing.T) {
    t.Run("validates state parameter", func(t *testing.T) {
        // Attempt CSRF attack with invalid state
        req := httptest.NewRequest("GET", "/api/auth/callback?code=valid&state=malicious", nil)
        rec := httptest.NewRecorder()
        
        handler.ServeHTTP(rec, req)
        
        assert.Equal(t, http.StatusBadRequest, rec.Code)
        assert.Contains(t, rec.Body.String(), "invalid state parameter")
    })
    
    t.Run("rejects replay attacks", func(t *testing.T) {
        code := "authorization_code_123"
        
        // First use: should succeed
        req1 := createCallbackRequest(code, validState)
        rec1 := httptest.NewRecorder()
        handler.ServeHTTP(rec1, req1)
        assert.Equal(t, http.StatusOK, rec1.Code)
        
        // Second use: should fail (code already used)
        req2 := createCallbackRequest(code, validState)
        rec2 := httptest.NewRecorder()
        handler.ServeHTTP(rec2, req2)
        assert.Equal(t, http.StatusBadRequest, rec2.Code)
        assert.Contains(t, rec2.Body.String(), "authorization code already used")
    })
    
    t.Run("validates token signature", func(t *testing.T) {
        // Tampered JWT
        tamperedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkFkbWluIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature"
        
        _, err := validateEVESSOToken(tamperedToken)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "invalid signature")
    })
    
    t.Run("enforces token expiration", func(t *testing.T) {
        expiredToken := generateExpiredEVESSOToken()
        
        _, err := validateEVESSOToken(expiredToken)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "token expired")
    })
}
```

---

## Best Practices

### 1. Secure Token Storage

#### Never store tokens in localStorage

```typescript
// BAD: Vulnerable to XSS
localStorage.setItem('token', token)

// GOOD: httpOnly cookie (immune to XSS)
// Set by server with:
// Set-Cookie: session=abc123; HttpOnly; Secure; SameSite=Strict
```

### 2. Password Security Testing

```go
func TestPasswordRequirements(t *testing.T) {
    tests := []struct {
        name     string
        password string
        valid    bool
    }{
        {name: "too short", password: "Ab1!", valid: false},
        {name: "no uppercase", password: "password123!", valid: false},
        {name: "no lowercase", password: "PASSWORD123!", valid: false},
        {name: "no number", password: "Password!", valid: false},
        {name: "no special char", password: "Password123", valid: false},
        {name: "valid password", password: "SecurePass123!", valid: true},
        {name: "common password", password: "Password123!", valid: false}, // Check against common list
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validatePassword(tt.password)
            if tt.valid {
                assert.NoError(t, err)
            } else {
                assert.Error(t, err)
            }
        })
    }
}
```

### 3. CSRF Protection Testing

```typescript
test('CSRF token validation', async () => {
  // Mock API with CSRF protection
  server.use(
    http.post('/api/sensitive-action', async ({ request }) => {
      const csrfToken = request.headers.get('X-CSRF-Token')
      const cookieToken = request.headers.get('Cookie')?.match(/csrf=([^;]+)/)?.[1]
      
      if (!csrfToken || csrfToken !== cookieToken) {
        return new HttpResponse(null, { status: 403 })
      }
      
      return HttpResponse.json({ success: true })
    })
  )
  
  // Valid request should succeed
  const validResponse = await fetch('/api/sensitive-action', {
    method: 'POST',
    headers: {
      'X-CSRF-Token': 'valid-token',
      'Cookie': 'csrf=valid-token'
    }
  })
  expect(validResponse.status).toBe(200)
  
  // Mismatched token should fail
  const invalidResponse = await fetch('/api/sensitive-action', {
    method: 'POST',
    headers: {
      'X-CSRF-Token': 'different-token',
      'Cookie': 'csrf=valid-token'
    }
  })
  expect(invalidResponse.status).toBe(403)
})
```

### 4. Rate Limiting Testing

```go
func TestRateLimiting(t *testing.T) {
    handler := RateLimitMiddleware(testHandler, 5, time.Minute) // 5 requests per minute
    
    // First 5 requests should succeed
    for i := 0; i < 5; i++ {
        req := httptest.NewRequest("GET", "/api/endpoint", nil)
        req.RemoteAddr = "192.168.1.1:12345"
        rec := httptest.NewRecorder()
        
        handler.ServeHTTP(rec, req)
        assert.Equal(t, http.StatusOK, rec.Code, "Request %d should succeed", i+1)
    }
    
    // 6th request should be rate limited
    req := httptest.NewRequest("GET", "/api/endpoint", nil)
    req.RemoteAddr = "192.168.1.1:12345"
    rec := httptest.NewRecorder()
    
    handler.ServeHTTP(rec, req)
    assert.Equal(t, http.StatusTooManyRequests, rec.Code)
    assert.Contains(t, rec.Body.String(), "rate limit exceeded")
}
```

### 5. Security Headers Validation

```go
func TestSecurityHeaders(t *testing.T) {
    req := httptest.NewRequest("GET", "/", nil)
    rec := httptest.NewRecorder()
    
    handler := SecurityHeadersMiddleware(testHandler)
    handler.ServeHTTP(rec, req)
    
    // Verify required security headers
    assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
    assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
    assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
    assert.Contains(t, rec.Header().Get("Content-Security-Policy"), "default-src 'self'")
    assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}
```

### 6. Sensitive Data Exposure

```go
func TestNoSensitiveDataInLogs(t *testing.T) {
    var logBuffer bytes.Buffer
    logger := log.New(&logBuffer, "", 0)
    
    // Simulate login with password
    loginRequest := LoginRequest{
        Email:    "user@example.com",
        Password: "SecurePassword123!",
    }
    
    // Log request (should NOT include password)
    logger.Printf("Login attempt: %+v", sanitizeForLogging(loginRequest))
    
    logOutput := logBuffer.String()
    
    assert.Contains(t, logOutput, "user@example.com")
    assert.NotContains(t, logOutput, "SecurePassword123!", "Password leaked in logs")
    assert.Contains(t, logOutput, "[REDACTED]")
}
```

---

## Common Patterns

### Pattern 1: Protected Route Testing (Frontend)

```typescript
describe('Protected Routes', () => {
  test('redirects to login when not authenticated', async () => {
    render(<ProtectedPage />, { wrapper: AuthProvider })
    
    await waitFor(() => {
      expect(window.location.pathname).toBe('/login')
    })
  })
  
  test('allows access when authenticated', async () => {
    // Setup authenticated context
    const wrapper = ({ children }) => (
      <AuthProvider initialUser={{ id: 1, email: 'user@example.com' }}>
        {children}
      </AuthProvider>
    )
    
    render(<ProtectedPage />, { wrapper })
    
    expect(await screen.findByText(/dashboard/i)).toBeInTheDocument()
    expect(window.location.pathname).toBe('/dashboard')
  })
})
```

### Pattern 2: SQL Injection Prevention

```go
func TestSQLInjectionPrevention(t *testing.T) {
    db := setupTestDB(t)
    
    // Attempt SQL injection
    maliciousInput := "'; DROP TABLE users; --"
    
    // Using parameterized queries (SAFE)
    var count int
    err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = $1", maliciousInput).Scan(&count)
    
    // Should not execute injection
    assert.NoError(t, err)
    assert.Equal(t, 0, count) // No users with that "email"
    
    // Verify users table still exists
    var tableExists bool
    err = db.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_name = 'users'
        )
    `).Scan(&tableExists)
    
    assert.NoError(t, err)
    assert.True(t, tableExists, "Users table should still exist")
}
```

### Pattern 3: Session Fixation Prevention

```go
func TestSessionFixation(t *testing.T) {
    // Attacker sets session ID
    attackerSessionID := "attacker-controlled-session-id"
    
    req := httptest.NewRequest("POST", "/api/login", strings.NewReader(`{"email":"user@example.com","password":"password"}`))
    req.Header.Set("Cookie", fmt.Sprintf("session=%s", attackerSessionID))
    rec := httptest.NewRecorder()
    
    loginHandler(rec, req)
    
    // After successful login, session ID MUST change
    newSessionCookie := rec.Header().Get("Set-Cookie")
    assert.Contains(t, newSessionCookie, "session=")
    assert.NotContains(t, newSessionCookie, attackerSessionID, "Session ID should regenerate after login")
}
```

---

## Anti-Patterns

### 1. Storing Secrets in Code

❌ **Avoid:** Hardcoded credentials, API keys
✅ **Do:** Use environment variables, secret managers

### 2. Weak Password Validation

❌ **Avoid:** Only checking length
✅ **Do:** Enforce complexity, check against common passwords

### 3. Information Leakage

❌ **Avoid:** Detailed error messages ("User not found" vs "Password incorrect")
✅ **Do:** Generic messages ("Invalid credentials")

### 4. Insecure Token Storage

❌ **Avoid:** localStorage, sessionStorage
✅ **Do:** httpOnly cookies

### 5. Missing HTTPS

❌ **Avoid:** Allowing HTTP for auth endpoints
✅ **Do:** Enforce HTTPS, use HSTS headers

---

## Quick Reference

### OWASP Top 10 Test Coverage

| Vulnerability | Test Pattern |
|---------------|--------------|
| A01: Broken Access Control | Authorization middleware tests |
| A02: Cryptographic Failures | Token validation, HTTPS enforcement |
| A03: Injection | SQL/XSS/Command injection tests |
| A04: Insecure Design | Threat modeling, abuse cases |
| A05: Security Misconfiguration | Security headers validation |
| A06: Vulnerable Components | Dependency scanning (separate) |
| A07: Auth Failures | Login flow, session management |
| A08: Data Integrity | CSRF, request signing |
| A09: Logging Failures | Sensitive data in logs |
| A10: SSRF | External request validation |

### Security Testing Checklist

- [ ] Authentication flow tests (login, logout, session)
- [ ] Authorization tests (role-based access control)
- [ ] Input validation (SQL injection, XSS, command injection)
- [ ] CSRF protection
- [ ] Rate limiting
- [ ] Security headers
- [ ] Password requirements
- [ ] Token expiration
- [ ] Session fixation prevention
- [ ] Sensitive data redaction in logs
- [ ] HTTPS enforcement
- [ ] OAuth/SSO flow validation

---

This skill provides comprehensive security testing patterns. Prioritize based on your threat model and compliance requirements.
