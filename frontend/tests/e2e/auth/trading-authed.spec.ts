import { test, expect, type Page, type Locator } from '@playwright/test';
import { mockWaypoint } from '../helpers/waypoint-mock';
import { REGION_THE_FORGE } from '../helpers/constants';

/**
 * Full trading workflow — only operable when authenticated (logged out, the
 * controls are disabled by design). Region/ship are prefilled from character
 * data; this test explicitly switches the region to The Forge (Jita) via its
 * Select so the workflow runs against a high-liquidity market that reliably
 * yields routes. Structure assertions only — live market values vary.
 *
 * Select layout: Region is the first combobox on the page, Schiff the second.
 */
function countFromText(text: string | null): number {
  return Number(text?.match(/(\d+)/)?.[1] ?? '0');
}

// Pick an option in the region Select (the first combobox on the trading page).
async function selectRegion(page: Page, optionRe: RegExp) {
  await page.getByRole('combobox').first().click();
  await page.getByRole('option', { name: optionRe }).first().click();
}

// Click "Berechnen" and return the route count, tolerating a cold-cache miss:
// the first heavy calculation right after a fresh ~890k-order market refresh can
// hit the backend's calculation-timeout and return zero (partial) routes with a
// warning. That is a transient, not a regression — re-run the calculation a
// bounded number of times until The Forge yields its reliably-present routes.
// Returns 0 if every attempt comes back empty (a genuine regression then fails
// the caller's `> 0` assertion).
async function calculateRoutes(page: Page, calc: Locator, counter: Locator, attempts = 3): Promise<number> {
  const empty = page.getByText(/Keine Routen gefunden/i);
  for (let attempt = 1; attempt <= attempts; attempt++) {
    await expect(calc).toBeEnabled({ timeout: 30000 });
    await calc.click();
    // The calculation resolves to either the route counter or the empty state.
    await expect(counter.or(empty).first()).toBeVisible({ timeout: 45000 });
    if (await counter.isVisible()) return countFromText(await counter.textContent());
  }
  return 0;
}

test.describe('Authenticated trading workflow', () => {
  test('calculate, inspect, set waypoint (mocked), filter, paginate', async ({ page }) => {
    // Legitimately slow: this single workflow runs a live market refresh plus two
    // heavy route calculations against The Forge sequentially, each with its own
    // calculate-retry budget (see calculateRoutes). Raise the wall-clock envelope
    // well past the 60s default so the cumulative live work fits; per-step timeouts
    // below still catch a genuinely hung operation.
    test.setTimeout(240_000);

    const waypoint = mockWaypoint(page);
    await page.goto('/trading');
    await expect(page.locator('h1')).toContainText('Trading');

    // Once character data settles, the calculate button enables and selects unlock.
    const calc = page.getByRole('button', { name: 'Berechnen' });
    await expect(calc).toBeEnabled({ timeout: 30000 });

    // ShipFittingCard renders for an authenticated character with a selected ship.
    await expect(page.getByText('Schiff-Fitting')).toBeVisible();

    // Select a high-liquidity region (The Forge / Jita) so the workflow exercises
    // real routes rather than an empty result for the character's home region.
    await selectRegion(page, new RegExp(REGION_THE_FORGE.name, 'i'));

    // The staleness indicator is shown for the selected region. Its text lives in
    // the element's title attribute, so match by title rather than text.
    await expect(page.getByTitle(/Letzte Aktualisierung/i).first()).toBeVisible();

    // Market-data refresh: disabled during refresh, then re-enabled.
    const refresh = page.getByRole('button', {
      name: /Market-Daten für diese Region aktualisieren/i,
    });
    await expect(refresh).toBeEnabled();
    await refresh.click();
    await expect(refresh).toBeEnabled({ timeout: 30000 });

    // Calculate routes. The Forge reliably has profitable routes — require them
    // (an empty result after the bounded retries signals a regression, not a
    // tolerable data gap).
    const counter = page.getByText(/\d+ Routen gefunden/);
    const total = await calculateRoutes(page, calc, counter);
    expect(total).toBeGreaterThan(0);

    // Authenticated: the "Route in EVE setzen" action is enabled. Clicking fires
    // up to two waypoint POSTs (buy + sell), both intercepted by the mock.
    const setRoute = page.getByRole('button', { name: /Route in EVE setzen/i }).first();
    await expect(setRoute).toBeEnabled();
    await setRoute.click();
    await expect.poll(() => waypoint.count(), { timeout: 10000 }).toBeGreaterThanOrEqual(1);
    await expect(page.getByText(/Waypoints in EVE gesetzt/i)).toBeVisible({ timeout: 10000 });

    // Filter effect: widening allowed security zones cannot reduce the count.
    await page.getByRole('checkbox', { name: /Low Sec/i }).check();
    await page.getByRole('checkbox', { name: /Null Sec/i }).check();
    const widened = await calculateRoutes(page, calc, counter);
    expect(widened).toBeGreaterThanOrEqual(total);

    // Pagination: "Mehr anzeigen" reveals more route cards when present.
    const more = page.getByRole('button', { name: /Mehr anzeigen/i });
    if (await more.isVisible().catch(() => false)) {
      const before = await page.getByRole('button', { name: /Route in EVE setzen/i }).count();
      await more.click();
      await expect
        .poll(async () => page.getByRole('button', { name: /Route in EVE setzen/i }).count())
        .toBeGreaterThan(before);
    }
  });
});
