import { test, expect } from '@playwright/test';

/**
 * Multi-Hub Comparison workflow (Issue #43) — only operable when authenticated
 * (the compare endpoint requires the session cookie; logged out, the search
 * input is disabled by design). The auth project injects the captured session
 * via storageState, so this spec runs against the real backend.
 *
 * Structure assertions only — live market values vary. The compare call hits a
 * live market endpoint and can be slow, so timeouts are generous.
 */
test.describe('Authenticated multi-hub comparison', () => {
  test('search an item, compare hubs, see table + recommended badge', async ({
    page,
  }) => {
    // The compare endpoint runs a live multi-hub market lookup; give the whole
    // workflow a wide envelope while per-step timeouts still catch a hang.
    test.setTimeout(120_000);

    await page.goto('/multi-hub');
    await expect(page.locator('h1')).toContainText('Multi-Hub');

    // Search a high-liquidity item that reliably exists across all hubs.
    const search = page.getByLabel('Item suchen');
    await expect(search).toBeEnabled({ timeout: 30000 });
    await search.fill('Tritanium');

    // Pick the matching suggestion from the listbox.
    const suggestions = page.getByRole('listbox', { name: /Suchergebnisse/i });
    await expect(suggestions).toBeVisible({ timeout: 30000 });
    await suggestions
      .getByRole('option', { name: /Tritanium/i })
      .first()
      .click();

    // The compare table renders for the picked item.
    const table = page.getByRole('table', { name: /Hub-Vergleich/i });
    await expect(table).toBeVisible({ timeout: 60000 });
    await expect(page.getByText(/Hub-Vergleich: Tritanium/i)).toBeVisible();

    // The recommended-hub badge highlights the best hub.
    await expect(page.getByText('Empfohlen').first()).toBeVisible();

    // The skills-applied panel is shown alongside the table.
    await expect(page.getByText('Skills angewendet')).toBeVisible();
  });
});
