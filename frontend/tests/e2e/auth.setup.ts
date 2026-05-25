import { test as setup, expect, request as playwrightRequest } from '@playwright/test';
import { existsSync } from 'node:fs';
import { AUTH_ENDPOINTS } from './helpers/constants';

const STORAGE_STATE = 'playwright/.auth/user.json';
const RECAPTURE_HINT =
  'No valid captured session. Re-run the one-time login capture via the Playwright MCP ' +
  'to regenerate playwright/.auth/user.json (see docs/superpowers/specs/2026-05-25-e2e-test-suite-design.md).';

setup('validate cached session', async () => {
  expect(existsSync(STORAGE_STATE), RECAPTURE_HINT).toBe(true);

  const ctx = await playwrightRequest.newContext({ storageState: STORAGE_STATE });
  try {
    const response = await ctx.get(AUTH_ENDPOINTS.session);
    expect(response.ok(), RECAPTURE_HINT).toBeTruthy();
    const data = await response.json();
    expect(data.authenticated, RECAPTURE_HINT).toBe(true);
    expect(data.character, RECAPTURE_HINT).toBeTruthy();
  } finally {
    await ctx.dispose();
  }
});
