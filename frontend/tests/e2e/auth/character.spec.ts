import { test, expect } from '@playwright/test';

test.describe('Authenticated character + navigation', () => {
  test('character page shows the logged-in character', async ({ page }) => {
    await page.goto('/character');
    // Authenticated: the "Character Information" card with portrait + name,
    // NOT the "Please login" prompt.
    await expect(page.getByText('Character Information')).toBeVisible();
    await expect(page.getByText(/please login/i)).toHaveCount(0);
    await expect(page.getByRole('img').first()).toBeVisible();
  });

  test('navigation shows CharacterInfo (logout affordance) instead of login', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('button', { name: /login with eve/i })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Logout' }).first()).toBeVisible();
  });
});
