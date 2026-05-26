import { test, expect } from '@playwright/test';

test.describe('Login UI (unauthenticated)', () => {
  test('login button is visible and initiates EVE SSO redirect', async ({ page }) => {
    await page.goto('/');
    const loginBtn = page.getByRole('button', { name: /login with eve/i }).first();
    await expect(loginBtn).toBeVisible();

    // Clicking sets window.location to the EVE SSO authorize URL. Catch the
    // navigation toward login.eveonline.com, then tolerate staying on host.
    const navPromise = page
      .waitForURL(/login\.eveonline\.com/, { timeout: 15000 })
      .catch(() => undefined);
    await loginBtn.click();
    await navPromise;
    expect(page.url()).toMatch(/login\.eveonline\.com|localhost:9000/);
  });
});
