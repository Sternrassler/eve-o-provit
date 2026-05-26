// Non-interactive session refresh for the E2E `auth` project.
//
// When only the short-lived access token has expired but the refresh token
// (~30d) in playwright/.auth/user.json is still valid, this renews the session
// via /auth/refresh and re-saves the storage state — no browser login needed.
// If the refresh token is also expired, re-run capture-session.mjs instead.
//
// Usage (from eve-o-provit/frontend, with the backend running on :9001):
//   node tests/e2e/refresh-session.mjs

import { request } from '@playwright/test';

const SESSION_URL = 'http://localhost:9001/auth/session';
const REFRESH_URL = 'http://localhost:9001/auth/refresh';
const STORAGE_STATE = 'playwright/.auth/user.json';

const ctx = await request.newContext({ storageState: STORAGE_STATE });
try {
  await ctx.post(REFRESH_URL);
  const res = await ctx.get(SESSION_URL);
  const data = await res.json().catch(() => ({}));
  if (!data.authenticated) {
    console.error('✗ Refresh fehlgeschlagen — Refresh-Token vermutlich abgelaufen.');
    console.error('  Bitte neu einloggen: node tests/e2e/capture-session.mjs');
    process.exit(1);
  }
  await ctx.storageState({ path: STORAGE_STATE });
  console.log(`✓ Session erneuert für: ${data.character?.character_name ?? 'unbekannt'}`);
} finally {
  await ctx.dispose();
}
