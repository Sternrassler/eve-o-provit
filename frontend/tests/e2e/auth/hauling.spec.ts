import { test, expect } from '@playwright/test';

/**
 * Neighborhood Hauling Routes workflow (Issue #45) — only operable when
 * authenticated (the hauling endpoint requires the session cookie and uses the
 * character's current region). The auth project injects the captured session
 * via storageState, so this spec runs against the real backend.
 *
 * Structure assertions only — live market values vary. The scan hits a live
 * multi-region market lookup and can be slow, so timeouts are generous.
 */
test.describe('Authenticated hauling routes', () => {
  test('fill the form, search, see route list or empty-state', async ({
    page,
  }) => {
    // The hauling endpoint scans the current + adjacent regions live; give the
    // whole workflow a wide envelope while per-step timeouts still catch a hang.
    test.setTimeout(120_000);

    await page.goto('/hauling');
    await expect(page.locator('h1')).toContainText('Hauling Routes');

    // Submit button is enabled once authenticated (ship defaults filled).
    const submit = page.getByRole('button', { name: /Routen finden/i });
    await expect(submit).toBeEnabled({ timeout: 30000 });

    // Adjust capital to a healthy budget and submit.
    const capital = page.getByLabel('Kapital (ISK)');
    await capital.fill('500000000');
    await submit.click();

    // Either route cards appear, or the empty-state hint (no profitable routes
    // nearby) — both are valid backend outcomes.
    const routeList = page.getByTestId('hauling-route-list');
    const empty = page.getByTestId('hauling-empty');
    await expect(routeList.or(empty)).toBeVisible({ timeout: 90000 });

    if (await routeList.isVisible()) {
      const firstRoute = page.getByTestId('hauling-route').first();
      await expect(firstRoute).toBeVisible();
      // Security badge renders on the route card.
      await expect(
        firstRoute.getByTestId('security-badge')
      ).toBeVisible();
      // Expanding reveals the cargo table.
      await firstRoute.getByTestId('hauling-expand').click();
      await expect(page.getByTestId('cargo-table').first()).toBeVisible();
    }

    // The skills-applied panel is shown alongside the results.
    await expect(page.getByText('Skills angewendet')).toBeVisible();
  });
});
