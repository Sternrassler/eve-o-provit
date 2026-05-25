import { test, expect, request as playwrightRequest } from '@playwright/test';
import { AUTH_ENDPOINTS } from '../helpers/constants';

const STORAGE_STATE = 'playwright/.auth/user.json';

test.describe('Logout', () => {
  // Isolated context seeded from the captured state so logging out here does not
  // invalidate the cookies the other auth tests rely on.
  test('logout clears the session and restores the login button', async ({ browser }) => {
    const context = await browser.newContext({ storageState: STORAGE_STATE });
    const page = await context.newPage();
    try {
      await page.goto('http://localhost:9000/');
      const logout = page.getByRole('button', { name: 'Logout' }).first();
      await expect(logout).toBeVisible();
      await logout.click();

      await expect(
        page.getByRole('button', { name: /login with eve/i }).first(),
      ).toBeVisible({ timeout: 10000 });

      // /auth/session reports unauthenticated for this context's (now-cleared) cookies.
      const ctx = await playwrightRequest.newContext({ storageState: await context.storageState() });
      const res = await ctx.get(AUTH_ENDPOINTS.session);
      const data = await res.json();
      expect(data.authenticated).toBe(false);
      await ctx.dispose();
    } finally {
      await context.close();
    }
  });
});
