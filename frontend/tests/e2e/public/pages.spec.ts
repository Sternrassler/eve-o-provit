import { test, expect } from '@playwright/test';

test.describe('Public pages load', () => {
  test('home shows hero and CTAs', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/EVE-O-Provit/i);
    await expect(page.locator('h1')).toContainText('EVE-O-Provit');
    await expect(page.getByRole('link', { name: /start trading/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /character skills/i })).toBeVisible();
  });

  test('main navigation: Trading group + top-level links, no dead placeholders', async ({ page }) => {
    await page.goto('/');
    const nav = page.locator('nav').first();

    // Top-level links stay direct.
    await expect(nav.getByRole('link', { name: 'Mining' })).toHaveAttribute('href', '/mining');
    await expect(nav.getByRole('link', { name: 'Character' })).toHaveAttribute('href', '/character');

    // The static Phase-3 placeholders are no longer advertised in the nav.
    await expect(nav.getByRole('link', { name: 'Trends' })).toHaveCount(0);
    await expect(nav.getByRole('link', { name: 'Watchlist' })).toHaveCount(0);

    // The five buy→sell→profit tools now live under the "Trading" dropdown.
    await nav.getByRole('button', { name: 'Trading' }).click();
    for (const [name, path] of [
      ['Trading Routes', '/trading'],
      ['Hauling', '/hauling'],
      ['ROI Calculator', '/roi-calculator'],
      ['Multi-Hub', '/multi-hub'],
      ['Sell Assets', '/sell-assets'],
    ] as const) {
      await expect(page.getByRole('link', { name }).first()).toHaveAttribute('href', path);
    }
  });

  test('trading page loads with controls', async ({ page }) => {
    await page.goto('/trading');
    await expect(page.locator('h1')).toContainText('Trading');
    await expect(page.getByText('Region', { exact: true })).toBeVisible();
    await expect(page.getByText('Schiff', { exact: true })).toBeVisible();
    // The calculate button reads "Lade Character-Daten..." while the page probes
    // character data on load, then settles to "Berechnen".
    await expect(
      page.getByRole('button', { name: /Berechnen|Lade Character-Daten/ }),
    ).toBeVisible();
  });

  test('multi-hub loads', async ({ page }) => {
    await page.goto('/multi-hub');
    await expect(page.locator('h1')).toContainText('Multi-Hub Comparison');
  });

  test('roi-calculator loads', async ({ page }) => {
    await page.goto('/roi-calculator');
    await expect(page.locator('h1')).toContainText('ROI Calculator');
  });

  test('trends loads', async ({ page }) => {
    await page.goto('/trends');
    await expect(page.locator('h1')).toContainText('Price Trend Analysis');
  });

  test('watchlist loads', async ({ page }) => {
    await page.goto('/watchlist');
    await expect(page.locator('h1')).toContainText('Watchlist & Alerts');
  });

  test('character page prompts login when unauthenticated', async ({ page }) => {
    await page.goto('/character');
    // Unauthenticated: a "Login with EVE" affordance is reachable (nav or page body).
    await expect(page.getByRole('button', { name: /login with eve/i }).first()).toBeVisible();
  });

  test('trading sub-tabs link the five tools, active tab is route-aware', async ({ page }) => {
    await page.goto('/roi-calculator');
    const tabs = page.getByRole('navigation', { name: 'Trading-Werkzeuge' });
    for (const [name, path] of [
      ['Routes', '/trading'],
      ['Hauling', '/hauling'],
      ['ROI', '/roi-calculator'],
      ['Multi-Hub', '/multi-hub'],
      ['Sell Assets', '/sell-assets'],
    ] as const) {
      await expect(tabs.getByRole('link', { name, exact: true })).toHaveAttribute('href', path);
    }
    // The tab for the current route is marked active.
    await expect(
      tabs.getByRole('link', { name: 'ROI', exact: true }),
    ).toHaveAttribute('aria-current', 'page');
  });
});
