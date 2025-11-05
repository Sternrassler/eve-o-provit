# Chrome DevTools MCP Skill

**Tech Stack:** Chrome DevTools MCP Server  
**Purpose:** Browser automation, testing, debugging, and screenshot capabilities  
**Version:** MCP Protocol (2024+)

---

## Architecture Overview

**Chrome DevTools MCP Server** provides programmatic access to browser automation through the Model Context Protocol. It enables:

- **Page Management:** Create, navigate, and manage browser pages
- **Element Interaction:** Click, fill forms, hover, drag-and-drop
- **Snapshot Capabilities:** Text-based accessibility tree snapshots + visual screenshots
- **Network & Console Inspection:** Monitor requests, responses, and console messages
- **Performance Analysis:** Trace recordings, Core Web Vitals, LCP/FCP metrics

**When to Use:**

- E2E testing of authenticated routes
- Visual regression testing
- Debugging frontend issues (screenshots + console logs)
- Performance profiling
- Accessibility validation

---

## Architecture Patterns

### 1. Snapshot-First Approach

**Pattern:** Always take a text snapshot before screenshots to minimize token usage.

```txt
1. Take text snapshot (a11y tree) → Identify elements by UID
2. Interact with elements using UIDs
3. Take screenshot only when visual confirmation needed
```

**Benefits:**

- Lower token consumption (text < images)
- Faster interaction (no image processing)
- Accessibility-first mindset

### 2. Page Lifecycle Management

**Pattern:** Explicit page creation → navigation → interaction → cleanup.

```txt
Create Page → Navigate to URL → Wait for Load → Interact → Close Page
```

**Anti-Pattern:** Reusing pages without cleanup (stale state, memory leaks)

### 3. Element Selection Strategy

**Priority Order:**

1. Accessibility selectors (role, label) → Best for stability
2. UID from snapshot → Precise targeting
3. CSS selectors → Last resort (brittle)

---

## Best Practices (Normative Requirements)

1. **Snapshot Before Screenshot (MUST)**
   - Text snapshots are cheaper and faster
   - Use screenshots only for visual validation or debugging
   - Combine both: snapshot for interaction, screenshot for verification

2. **Use Accessibility Selectors (MUST)**
   - More stable than CSS selectors
   - Promotes accessible markup
   - Example: `role=button[name="Submit"]` vs `.btn-primary`

3. **Wait for Network Idle (MUST)**
   - Don't interact immediately after navigation
   - Use `wait_for` with expected text/elements
   - Monitor network requests to confirm API responses

4. **Handle Dynamic Content (SHOULD)**
   - Poll for element visibility before interaction
   - Use `wait_for` instead of arbitrary delays
   - Check console for JavaScript errors

5. **Isolate Tests (MUST)**
   - Each test should create its own page
   - Clean up pages after test completion
   - Avoid shared state between tests

6. **Monitor Console & Network (SHOULD)**
   - Check console for errors after critical actions
   - Validate network requests for expected API calls
   - Use network inspection for debugging 401/403 errors

7. **Performance Testing (MAY)**
   - Use performance traces for Core Web Vitals
   - Identify slow page loads (LCP, FCP)
   - Analyze performance insights for optimization

---

## Common Patterns

### Pattern 1: Authenticated Route Testing

**Scenario:** Test protected routes with authentication cookies.

```txt
1. Navigate to login page
2. Fill credentials (email, password)
3. Submit form
4. Wait for redirect to dashboard
5. Take snapshot to verify authenticated state
6. Navigate to protected route
7. Verify access granted (no 401/403)
```

**Key Points:**

- Cookies persist across navigation within same page
- Use `wait_for` to confirm successful authentication
- Check network tab for auth token in requests

### Pattern 2: Form Submission Testing

**Scenario:** Submit form and verify database changes.

```txt
1. Take snapshot → Identify form fields by UID
2. Fill form using `fill` tool
3. Click submit button
4. Wait for success message
5. Verify network request succeeded (200 status)
6. Check database for created record
```

**Key Points:**

- Use `fill_form` for multiple fields simultaneously
- Monitor network requests for API response
- Combine with backend verification (database query)

### Pattern 3: Visual Regression Detection

**Scenario:** Detect unexpected UI changes.

```txt
1. Navigate to page
2. Take full-page screenshot
3. Compare with baseline screenshot
4. Flag differences exceeding threshold
```

**Key Points:**

- Use `fullPage: true` for complete screenshots
- Store baseline images in version control
- Automate comparison in CI/CD pipeline

### Pattern 4: Debugging Frontend Errors

**Scenario:** User reports "page not loading correctly."

```txt
1. Navigate to reported URL
2. Take snapshot to check rendered elements
3. Take screenshot for visual confirmation
4. Check console messages for errors
5. Inspect network requests for failed API calls
6. Identify root cause (JavaScript error, API failure, etc.)
```

**Key Points:**

- Console errors often reveal JavaScript issues
- Network tab shows API failures (500, 404)
- Screenshot confirms visual state vs expected

### Pattern 5: Performance Profiling

**Scenario:** Page feels slow, need metrics.

```txt
1. Start performance trace (with reload)
2. Navigate to page
3. Wait for page load
4. Stop trace
5. Analyze Core Web Vitals (LCP, FCP, CLS)
6. Review performance insights
```

**Key Points:**

- LCP < 2.5s (good), > 4s (poor)
- FCP < 1.8s (good), > 3s (poor)
- Use insights to identify bottlenecks

---

## Anti-Patterns

### ❌ Taking Screenshots for Everything

**Why:** High token cost, slow, unnecessary for most interactions.  
**Instead:** Use text snapshots for element identification, screenshots only for visual validation.

### ❌ Using Arbitrary `sleep()` Delays

**Why:** Flaky tests, slower execution, doesn't handle variable load times.  
**Instead:** Use `wait_for` with expected text/elements or network idle.

### ❌ Hardcoded CSS Selectors

**Why:** Brittle, breaks with UI changes, not accessibility-friendly.  
**Instead:** Use accessibility selectors (role, label) or UIDs from snapshots.

### ❌ Ignoring Console Errors

**Why:** Silent failures, JavaScript errors cause broken functionality.  
**Instead:** Always check console after critical actions (page load, form submit).

### ❌ Reusing Pages Across Tests

**Why:** Stale state, cookies, memory leaks, test interdependence.  
**Instead:** Create fresh page for each test, close after completion.

---

## Integration with Testing Workflow

### With Playwright Skill

**Complementary, not overlapping:**

- **Playwright Skill:** Test structure, assertions, page object models
- **Chrome DevTools Skill:** Browser automation mechanics, debugging tools

**Example:** Use Playwright patterns for test organization, Chrome DevTools MCP for actual browser interactions.

### With Backend Testing

**Verification Chain:**

```text
1. Chrome DevTools → Submit form via browser
2. Backend → Verify database record created
3. Chrome DevTools → Verify success message displayed
```

**Benefits:** Full stack validation (frontend + backend).

---

## Performance Considerations

1. **Snapshot vs Screenshot Token Usage**
   - Text snapshot: ~500-1000 tokens
   - Screenshot (viewport): ~2000-5000 tokens
   - Screenshot (full page): ~5000-15000 tokens

2. **Parallel Execution**
   - MCP supports multiple pages simultaneously
   - Isolate tests to different pages for parallelism
   - Avoid shared state (cookies, localStorage)

3. **Network Monitoring Overhead**
   - Minimal impact on performance
   - Essential for debugging API issues
   - Use selectively (not every test)

---

## Security Guidelines

1. **Credential Handling**
   - Never hardcode passwords in tests
   - Use environment variables for sensitive data
   - Clear cookies/storage after authentication tests

2. **Screenshot Privacy**
   - Screenshots may contain sensitive user data
   - Mask PII before storing/sharing screenshots
   - Don't commit screenshots with production data

3. **Network Inspection**
   - Network logs may contain auth tokens
   - Don't log full request/response bodies in production
   - Sanitize logs before sharing

---

## Quick Reference

| Operation | Tool | Use Case |
|-----------|------|----------|
| Text-based page view | `take_snapshot` | Element identification, interaction |
| Visual confirmation | `take_screenshot` | Debugging, visual regression |
| Click element | `click` | Button clicks, link navigation |
| Fill form field | `fill` | Single input field |
| Fill entire form | `fill_form` | Multiple fields at once |
| Check console | `list_console_messages` | JavaScript error debugging |
| Check network | `list_network_requests` | API failure debugging |
| Wait for element | `wait_for` | Dynamic content loading |
| Performance metrics | `performance_start_trace` | Core Web Vitals, bottleneck analysis |

---

## Common Debugging Scenarios

### Scenario: 401 Error on Authenticated Route

**Steps:**

1. Take snapshot → Check if user is logged in (auth indicators)
2. Check console → Look for auth token errors
3. Check network → Verify `Authorization` header in request
4. Check cookies → Confirm auth cookie exists

### Scenario: Form Not Submitting

**Steps:**

1. Take snapshot → Verify form fields are filled
2. Check console → Look for JavaScript errors
3. Check network → See if submit request was sent
4. Take screenshot → Confirm visual state

### Scenario: Page Load Timeout

**Steps:**

1. Start performance trace
2. Navigate to page
3. Analyze performance insights → Identify slow resources
4. Check network → Find requests taking too long

---

**Last Updated:** 2025-11-04  
**Maintained By:** skill-creator agent
