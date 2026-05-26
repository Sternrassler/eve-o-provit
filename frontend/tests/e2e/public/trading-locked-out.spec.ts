import { test, expect } from '@playwright/test';

/**
 * Logged-out policy: on the trading page nothing is operable without a session.
 * The region/ship selects and the calculate button stay disabled (the page
 * keeps the character-data query disabled while unauthenticated), so no route
 * calculation can be triggered.
 */
test.describe('Trading page is inert when logged out', () => {
  test('selects and calculate button are disabled', async ({ page }) => {
    await page.goto('/trading');
    await expect(page.locator('h1')).toContainText('Trading');

    // Both selects (Region, Schiff) render but are disabled.
    const comboboxes = page.getByRole('combobox');
    await expect(comboboxes).toHaveCount(2);
    await expect(comboboxes.nth(0)).toBeDisabled();
    await expect(comboboxes.nth(1)).toBeDisabled();

    // The calculate button is present but disabled (label stays "Lade
    // Character-Daten..." because the character query never runs logged out).
    const calc = page.getByRole('button', { name: /Berechnen|Lade Character-Daten/ });
    await expect(calc).toBeVisible();
    await expect(calc).toBeDisabled();
  });
});
