# Testing Skill: Playwright E2E

**Tech Stack:** Playwright v1.56.1

**Project:** eve-o-provit Frontend E2E Tests

---

## Architecture Patterns

### Test Organization

- **Feature-Based:** Tests grouped by feature (`auth.spec.ts`, `trading.spec.ts`)
- **Page Object Model:** Encapsulate page interactions in classes
- **Fixtures:** Reusable setup/teardown logic

### Test Execution

- **Parallel Execution:** Tests run concurrently across multiple browsers
- **Isolation:** Each test gets fresh browser context
- **Retry Strategy:** Failed tests retry automatically in CI

---

## Best Practices

1. **Accessibility Selectors:** Use `getByRole()`, `getByLabel()` (not CSS classes)
2. **Auto-Waiting:** Playwright waits for elements automatically (no manual waits)
3. **Test Isolation:** No shared state between tests
4. **Descriptive Names:** Test names describe user behavior, not implementation
5. **Error Debugging:** Take screenshots/videos on failure
6. **API Mocking:** Mock external APIs for deterministic tests
7. **Test Data:** Use fixtures for consistent test data

---

## Common Patterns

### 1. Basic Test Structure

```ts
import { test, expect } from "@playwright/test";

test.describe("Intra-Region Trading", () => {
  test("should display market orders", async ({ page }) => {
    await page.goto("/intra-region");
    
    await expect(page.getByRole("heading", { name: /trading/i })).toBeVisible();
    await expect(page.getByRole("combobox")).toBeVisible();
  });
});
```

### 2. User Interaction

```ts
test("should select region and load data", async ({ page }) => {
  await page.goto("/intra-region");
  
  // Select from dropdown
  await page.getByRole("combobox").click();
  await page.getByText("The Forge").click();
  
  // Wait for data to load
  await expect(page.getByRole("table")).toBeVisible();
});
```

### 3. API Response Waiting

```ts
test("should refresh market data", async ({ page }) => {
  await page.goto("/intra-region");
  
  const responsePromise = page.waitForResponse(
    (res) => res.url().includes("/api/v1/market") && res.status() === 200
  );
  
  await page.getByRole("button", { name: /refresh/i }).click();
  
  const response = await responsePromise;
  expect(response.ok()).toBeTruthy();
});
```

### 4. Page Object Model

```ts
// tests/pages/TradingPage.ts
export class TradingPage {
  constructor(private page: Page) {}
  
  async goto() {
    await this.page.goto("/intra-region");
  }
  
  async selectRegion(regionName: string) {
    await this.page.getByRole("combobox").click();
    await this.page.getByText(regionName).click();
  }
  
  async isTableVisible() {
    return await this.page.getByRole("table").isVisible();
  }
}

// Usage
test("trading flow", async ({ page }) => {
  const tradingPage = new TradingPage(page);
  await tradingPage.goto();
  await tradingPage.selectRegion("The Forge");
  expect(await tradingPage.isTableVisible()).toBe(true);
});
```

---

## Anti-Patterns

❌ **Hardcoded Waits:** Don't use `waitForTimeout()` (Playwright auto-waits)

❌ **CSS Selectors:** Avoid `.btn-primary` (brittle, use `getByRole()`)

❌ **Test Interdependence:** Tests should not rely on execution order

❌ **No Cleanup:** Always clean up test data (use `afterEach`)

❌ **Testing Implementation:** Test user behavior, not internal state

---

## Integration with CI/CD

### GitHub Actions Configuration

```yaml
- name: Run Playwright tests
  run: npm run test:e2e
  
- name: Upload test results
  if: failure()
  uses: actions/upload-artifact@v3
  with:
    name: playwright-report
    path: playwright-report/
```

---

## Performance Considerations

- **Parallel Execution:** Configure workers based on CPU cores
- **Selective Testing:** Use `test.only()` for focused debugging
- **Headless Mode:** Faster execution in CI (no GUI rendering)
- **Browser Reuse:** Share browser context when possible

---

## Security Guidelines

- **No Secrets in Tests:** Use environment variables for credentials
- **Mock Sensitive APIs:** Don't hit real authentication endpoints in tests

---

## Quick Reference

| Task | Pattern |
|------|---------|
| Click element | `await page.getByRole("button", { name: /text/i }).click()` |
| Fill input | `await page.getByLabel("Username").fill("john")` |
| Assert visibility | `await expect(page.getByText("Success")).toBeVisible()` |
| Wait for API | `await page.waitForResponse((res) => res.url().includes("/api/"))` |
| Take screenshot | `await page.screenshot({ path: "screenshot.png" })` |
| Mock API | `await page.route("/api/**", (route) => route.fulfill({ body: "..." }))` |
| Debug | `await page.pause()` (opens Playwright Inspector) |
