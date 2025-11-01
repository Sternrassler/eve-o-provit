import { Page } from '@playwright/test';

/**
 * Authentication Helper Functions for E2E Tests
 * 
 * These helpers manage EVE SSO login flow for tests
 */

/**
 * Performs full EVE SSO login flow
 * 
 * @param page - Playwright page object
 * @returns Promise<void>
 * 
 * Prerequisites:
 * - EVE_TEST_CHARACTER env var set
 * - EVE_TEST_USERNAME env var set
 * - EVE_TEST_PASSWORD env var set
 */
export async function loginWithEVE(page: Page): Promise<void> {
  const characterName = process.env.EVE_TEST_CHARACTER || 'Test Character';
  const username = process.env.EVE_TEST_USERNAME || '';
  const password = process.env.EVE_TEST_PASSWORD || '';
  
  if (!username || !password) {
    throw new Error('EVE_TEST_USERNAME and EVE_TEST_PASSWORD must be set');
  }
  
  console.log('[Auth Helper] Starting EVE SSO login flow...');
  
  // Go to home page
  await page.goto('/');
  
  // Click login button
  const loginButton = page.getByRole('button', { name: /login with eve/i });
  await loginButton.click();
  
  console.log('[Auth Helper] Clicked login button, waiting for EVE SSO...');
  
  // Wait for redirect to EVE SSO
  await page.waitForURL(/login\.eveonline\.com/, { timeout: 10000 });
  
  console.log('[Auth Helper] Redirected to EVE SSO');
  
  // Character selection (if multiple characters)
  const characterOption = page.locator(`text=${characterName}`).first();
  
  if (await characterOption.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log('[Auth Helper] Character selection page detected');
    await characterOption.click();
  }
  
  // Wait for login form
  await page.waitForSelector('input[name="UserName"], input[type="text"]', { timeout: 10000 });
  
  console.log('[Auth Helper] Login form detected, filling credentials...');
  
  // Fill credentials
  await page.fill('input[name="UserName"]', username);
  await page.fill('input[name="Password"]', password);
  
  // Submit
  await page.click('input[type="submit"], button[type="submit"]');
  
  // Wait for possible authorize page (first time only)
  await page.waitForTimeout(2000);
  
  const authorizeButton = page.locator('input[value="Authorize"], button:has-text("Authorize")');
  
  if (await authorizeButton.isVisible({ timeout: 5000 }).catch(() => false)) {
    console.log('[Auth Helper] Authorize page detected, clicking authorize...');
    await authorizeButton.click();
  }
  
  // Wait for redirect back to app
  await page.waitForURL('http://localhost:9000/**', { timeout: 15000 });
  
  console.log('[Auth Helper] Redirected back to app');
  
  // Wait for character info to appear
  await page.waitForTimeout(2000);
  
  console.log('[Auth Helper] Login complete');
}

/**
 * Logs out the current user
 * 
 * @param page - Playwright page object
 */
export async function logout(page: Page): Promise<void> {
  console.log('[Auth Helper] Logging out...');
  
  // Open character dropdown
  const characterNav = page.locator('[data-testid="character-info"]');
  
  if (await characterNav.isVisible({ timeout: 2000 }).catch(() => false)) {
    await characterNav.click();
    
    // Click logout button
    const logoutBtn = page.getByRole('button', { name: /logout/i });
    await logoutBtn.click();
    
    // Wait for logout
    await page.waitForTimeout(1000);
    
    console.log('[Auth Helper] Logout complete');
  } else {
    console.log('[Auth Helper] Not logged in, skipping logout');
  }
}

/**
 * Checks if user is currently authenticated
 * 
 * @param page - Playwright page object
 * @returns Promise<boolean>
 */
export async function isAuthenticated(page: Page): Promise<boolean> {
  const characterInfo = page.locator('[data-testid="character-info"]');
  return await characterInfo.isVisible({ timeout: 1000 }).catch(() => false);
}

/**
 * Gets the currently authenticated character name
 * 
 * @param page - Playwright page object
 * @returns Promise<string | null>
 */
export async function getCharacterName(page: Page): Promise<string | null> {
  const characterName = page.locator('[data-testid="character-name"]');
  
  if (await characterName.isVisible({ timeout: 1000 }).catch(() => false)) {
    return await characterName.textContent();
  }
  
  return null;
}

/**
 * Sets up authentication state for tests (fixture helper)
 * 
 * This can be used with Playwright's storageState to persist auth
 * across tests without re-logging in every time.
 * 
 * Usage in playwright.config.ts:
 * ```
 * projects: [
 *   {
 *     name: 'authenticated',
 *     use: {
 *       storageState: 'tests/.auth/user.json',
 *     },
 *   },
 * ],
 * ```
 */
export async function setupAuthState(page: Page, outputPath: string): Promise<void> {
  await loginWithEVE(page);
  
  // Save storage state
  await page.context().storageState({ path: outputPath });
  
  console.log(`[Auth Helper] Auth state saved to ${outputPath}`);
}
