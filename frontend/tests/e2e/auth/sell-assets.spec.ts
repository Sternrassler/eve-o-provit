import { test, expect } from '@playwright/test';

/**
 * Sell-from-Assets workflow (Issue #107) — only operable when authenticated
 * (the assets + sell-options endpoints require the session cookie and use the
 * character's owned assets / current region). The auth project injects the
 * captured session via storageState, so this spec runs against the real backend.
 *
 * Structure assertions only — live market values vary. The sell-options scan
 * hits a live multi-region market lookup and can be slow, so timeouts are
 * generous.
 */
test.describe('Authenticated sell-from-assets', () => {
  test('pick an asset, search, see sell options or empty-state', async ({
    page,
  }) => {
    test.setTimeout(120_000);

    await page.goto('/sell-assets');
    await expect(page.locator('h1')).toContainText('Sell from Assets');

    // The picker loads the character's assets once authenticated.
    const picker = page.getByTestId('asset-picker');
    await expect(picker).toBeVisible({ timeout: 30000 });

    // Either the asset list or the empty-state appears (both valid).
    const assetList = page.getByTestId('asset-list');
    const assetEmpty = page.getByTestId('asset-picker-empty');
    await expect(assetList.or(assetEmpty)).toBeVisible({ timeout: 30000 });

    // Without assets there is nothing to sell — done.
    if (!(await assetList.isVisible())) {
      return;
    }

    // Pick the first marketable asset (selectable rows are enabled).
    const marketableRow = page
      .getByTestId('asset-row')
      .filter({ hasNot: page.locator('[disabled]') })
      .first();
    const firstEnabled = page.locator('[data-testid="asset-row"]:not([disabled])').first();
    await expect(firstEnabled.or(marketableRow)).toBeVisible();
    await firstEnabled.click();

    // The sell form (quantity + search button) appears after selection.
    const submit = page.getByRole('button', { name: /Verkaufsorte suchen/i });
    await expect(submit).toBeEnabled({ timeout: 10000 });
    await submit.click();

    // Either option cards appear, or the empty-state hint — both are valid.
    const result = page.getByTestId('sell-options-result');
    const empty = page.getByTestId('sell-options-empty');
    await expect(result.or(empty)).toBeVisible({ timeout: 90000 });

    if (await result.isVisible()) {
      const firstOption = page.getByTestId('sell-option').first();
      await expect(firstOption).toBeVisible();
      // Security + scope badges render on the option card.
      await expect(firstOption.getByTestId('security-badge')).toBeVisible();
      await expect(firstOption.getByTestId('scope-badge')).toBeVisible();
    }

    // The skills-applied panel is shown alongside the results.
    await expect(page.getByText('Skills angewendet')).toBeVisible();
  });
});
