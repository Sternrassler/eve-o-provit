// One-time interactive session capture for the E2E `auth` project.
//
// Opens a headed browser at the app, waits for you to complete the real EVE SSO
// login (Steam route if your account is Steam-linked, incl. 2FA), then saves the
// browser storage state — including the HttpOnly session cookies — to
// playwright/.auth/user.json (gitignored). Re-run whenever the session expires.
//
// Usage (from eve-o-provit/frontend, with the stack running on :9000/:9001):
//   node tests/e2e/capture-session.mjs

import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import { dirname } from 'node:path';

const APP_URL = 'http://localhost:9000';
const SESSION_URL = 'http://localhost:9001/auth/session';
const STORAGE_STATE = 'playwright/.auth/user.json';
const TIMEOUT_MS = 5 * 60 * 1000;

mkdirSync(dirname(STORAGE_STATE), { recursive: true });

const browser = await chromium.launch({ headless: false });
const context = await browser.newContext();
const page = await context.newPage();

await page.goto(APP_URL);
console.log('\n→ Bitte im geöffneten Browser einloggen ("Login with EVE").');
console.log('  Warte auf authentifizierte Session (max. 5 Minuten)...\n');

const deadline = Date.now() + TIMEOUT_MS;
let authenticated = false;
while (Date.now() < deadline) {
  // context.request shares cookies with the browser context.
  const res = await context.request.get(SESSION_URL).catch(() => null);
  const data = res ? await res.json().catch(() => ({})) : {};
  if (data.authenticated) {
    authenticated = true;
    console.log(`✓ Eingeloggt als: ${data.character?.character_name ?? 'unbekannt'}`);
    break;
  }
  await new Promise((r) => setTimeout(r, 2000));
}

if (!authenticated) {
  console.error('✗ Timeout: keine authentifizierte Session erkannt.');
  await browser.close();
  process.exit(1);
}

await context.storageState({ path: STORAGE_STATE });
console.log(`✓ Session gespeichert: ${STORAGE_STATE}`);
await browser.close();
