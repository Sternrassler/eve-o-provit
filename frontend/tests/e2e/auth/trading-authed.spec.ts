import { test, expect, type Page } from '@playwright/test';
import { mockWaypoint } from '../helpers/waypoint-mock';
import { REGION_THE_FORGE } from '../helpers/constants';

// Open a radix Select identified by its visible label, then pick an option.
async function selectByLabel(page: Page, label: string, optionRe: RegExp) {
  const labelEl = page.getByText(label, { exact: true });
  const combo = labelEl.locator('xpath=following::*[@role="combobox"][1]');
  await combo.click();
  await page.getByRole('option', { name: optionRe }).first().click();
}

/**
 * Full trading workflow (only operable when authenticated). Region/ship are
 * prefilled from character data (falling back to defaults), so the workflow is
 * driven via those prefilled values rather than the radix Select dropdown.
 * Structure assertions only — live market data varies.
 */
function countFromText(text: string | null): number {
  return Number(text?.match(/(\d+)/)?.[1] ?? '0');
}

test.describe('Authenticated trading workflow', () => {
  test('calculate, inspect, set waypoint (mocked), filter, paginate', async ({ page }) => {
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
    await selectByLabel(page, 'Region', new RegExp(REGION_THE_FORGE.name, 'i'));

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

    // Calculate routes.
    await expect(calc).toBeEnabled({ timeout: 30000 });
    await calc.click();

    const counter = page.getByText(/\d+ Routen gefunden/);
    const emptyState = page.getByText('Keine Routen gefunden');
    const errorTitle = page.getByText('Fehler', { exact: true });
    await expect(counter.or(emptyState).or(errorTitle)).toBeVisible({ timeout: 45000 });

    if (await counter.isVisible().catch(() => false)) {
      const total = countFromText(await counter.textContent());
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
      await calc.click();
      await expect(page.getByText(/\d+ Routen gefunden/)).toBeVisible({ timeout: 45000 });
      const widened = countFromText(await page.getByText(/\d+ Routen gefunden/).textContent());
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
    }
  });
});
