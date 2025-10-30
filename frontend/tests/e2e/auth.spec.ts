import { test, expect } from '@playwright/test';

/**
 * EVE SSO OAuth2 Login Flow E2E Test
 * 
 * Voraussetzungen:
 * - make docker-rebuild (Services müssen laufen)
 * - .env mit EVE_TEST_CHARACTER und EVE_TEST_PASSWORD
 * 
 * Flow:
 * 1. App öffnen
 * 2. Login Button klicken → Redirect zu EVE SSO
 * 3. Character auswählen (erste Stufe)
 * 4. Login mit Username/Password (zweite Stufe)
 * 5. Authorize (falls erster Login)
 * 6. Zurück zur App → Character Info sichtbar
 */

test.describe('EVE SSO Authentication', () => {
  
  test.skip('EVE SSO Login Flow - Character Selection & Login', async ({ page }) => {
    // Debug: Alle Navigationen loggen
    page.on('console', msg => console.log('Browser:', msg.text()));
    
    // 1. App öffnen
    await page.goto('/');
    await expect(page).toHaveTitle(/EVE-O-Provit/);
    
    // 2. Login Button suchen und klicken
    // TODO: Implement Login Button in Navigation first
    const loginButton = page.getByRole('button', { name: /login/i });
    await expect(loginButton).toBeVisible();
    await loginButton.click();
    
    // 3. Redirect zu EVE SSO (login.eveonline.com)
    await page.waitForURL(/login\.eveonline\.com/, { timeout: 10000 });
    console.log('✅ Redirected to EVE SSO');
    
    // EXPERIMENTIEREN: Character Selection Page
    // EVE zeigt zuerst alle Characters des Accounts
    await page.screenshot({ path: 'tests/screenshots/01-character-selection.png' });
    
    // Mögliche Selektoren (experimentell):
    // - div mit Character Name
    // - Button/Link mit "Select" oder Character Name
    const characterName = process.env.EVE_TEST_CHARACTER || 'Test Character';
    
    // Variante 1: Nach Text suchen
    const characterOption = page.locator(`text=${characterName}`).first();
    await characterOption.click();
    
    await page.screenshot({ path: 'tests/screenshots/02-after-character-click.png' });
    
    // 4. Login Page (Username/Password)
    // Nach Character-Auswahl kommt das eigentliche Login-Formular
    await page.waitForSelector('input[name="UserName"], input[type="text"]', { timeout: 10000 });
    
    await page.screenshot({ path: 'tests/screenshots/03-login-form.png' });
    
    const username = process.env.EVE_TEST_USERNAME || '';
    const password = process.env.EVE_TEST_PASSWORD || '';
    
    await page.fill('input[name="UserName"]', username);
    await page.fill('input[name="Password"]', password);
    
    await page.screenshot({ path: 'tests/screenshots/04-credentials-filled.png' });
    
    // Submit
    await page.click('input[type="submit"], button[type="submit"]');
    
    // 5. Authorize Page (nur beim ersten Mal)
    await page.waitForTimeout(2000); // Kurz warten
    
    const authorizeButton = page.locator('input[value="Authorize"], button:has-text("Authorize")');
    if (await authorizeButton.isVisible({ timeout: 5000 }).catch(() => false)) {
      console.log('✅ Authorize page detected');
      await page.screenshot({ path: 'tests/screenshots/05-authorize-page.png' });
      await authorizeButton.click();
    } else {
      console.log('⏭️  Authorize page skipped (already authorized)');
    }
    
    // 6. Zurück zur App
    await page.waitForURL('http://localhost:9000/**', { timeout: 15000 });
    console.log('✅ Redirected back to app');
    
    await page.screenshot({ path: 'tests/screenshots/06-back-to-app.png' });
    
    // 7. Character Info sollte sichtbar sein
    // TODO: Check for character name/portrait in UI
    await expect(page.locator(`text=${characterName}`)).toBeVisible({ timeout: 5000 });
    
    console.log('✅ Login successful - Character visible');
  });
  
  // Explorativer Test: EVE SSO Selektoren finden
  test('Explore EVE SSO Page Structure (headed mode)', async ({ page }) => {
    // Dieser Test läuft im headed mode (Browser sichtbar)
    // Zum Experimentieren: npx playwright test --headed --project=chromium
    
    await page.goto('/');
    
    // STOPP: Hier manuell Login Button klicken
    // und dann Page Structure analysieren
    await page.pause(); // Öffnet Playwright Inspector
  });
});
