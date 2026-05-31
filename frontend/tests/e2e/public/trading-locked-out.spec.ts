import { test, expect } from '@playwright/test';

/**
 * Logged-out policy: on the trading page nothing is operable without a session.
 * The region Select and the calculate button stay disabled (the page keeps the
 * character-data query disabled while unauthenticated), so no route calculation
 * can be triggered. The ship is read-only (CurrentShipCard, no dropdown) and has
 * no current ship without a session.
 */
test.describe('Trading page is inert when logged out', () => {
  test('region select and calculate button are disabled', async ({ page }) => {
    await page.goto('/trading');
    await expect(page.locator('h1')).toContainText('Trading');

    // Only the Region Select remains a combobox; it renders but is disabled.
    const comboboxes = page.getByRole('combobox');
    await expect(comboboxes).toHaveCount(1);
    await expect(comboboxes.nth(0)).toBeDisabled();

    // The current-ship card renders read-only (label "Schiff", no dropdown).
    await expect(page.getByText('Schiff', { exact: true })).toBeVisible();

    // The calculate button is present but disabled — without a session there is
    // no current ship, so no ship_type_id can be sent.
    const calc = page.getByRole('button', { name: /Berechnen|Lade Character-Daten/ });
    await expect(calc).toBeVisible();
    await expect(calc).toBeDisabled();
  });
});
