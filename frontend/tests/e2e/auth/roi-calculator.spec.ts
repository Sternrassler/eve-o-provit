import { test, expect } from '@playwright/test';

/**
 * ROI Calculator / Capital Allocation Optimizer workflow (Issue #44) — only
 * operable when authenticated (the optimize endpoint requires the session
 * cookie). The auth project injects the captured session via storageState, so
 * this spec runs against the real backend.
 *
 * Structure assertions only — live market values vary. The optimize call runs a
 * live portfolio optimization and can be slow, so timeouts are generous.
 */
test.describe('Authenticated ROI calculator', () => {
  test('fill the form, optimize, see allocation table + diversification score', async ({
    page,
  }) => {
    // The optimize endpoint runs a live multi-item market lookup; give the whole
    // workflow a wide envelope while per-step timeouts still catch a hang.
    test.setTimeout(120_000);

    await page.goto('/roi-calculator');
    await expect(page.locator('h1')).toContainText('ROI Calculator');

    // Submit button is enabled once authenticated (region/ship default-filled,
    // High-Sec checked by default).
    const submit = page.getByRole('button', { name: /Portfolio optimieren/i });
    await expect(submit).toBeEnabled({ timeout: 30000 });

    // Adjust capital to a healthy budget and submit.
    const capital = page.getByLabel('Kapital (ISK)');
    await capital.fill('500000000');
    await submit.click();

    // Either an allocation table appears, or the empty-state hint (no viable
    // portfolio for the budget) — both are valid backend outcomes.
    const table = page.getByRole('table', { name: /Kapital-Allokation/i });
    const empty = page.getByTestId('portfolio-empty');
    await expect(table.or(empty)).toBeVisible({ timeout: 90000 });

    if (await table.isVisible()) {
      // Allocation rows render, plus the diversification score and skills panel.
      await expect(page.getByTestId('portfolio-row').first()).toBeVisible();
      await expect(page.getByTestId('diversification-score')).toBeVisible();
      await expect(page.getByText('Skills angewendet')).toBeVisible();
    }
  });
});
