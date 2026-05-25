import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E configuration.
 * Three projects:
 *  - setup: validates the cached real session (playwright/.auth/user.json)
 *  - public: unauthenticated use cases (CI-capable)
 *  - auth: authenticated use cases, reuses the captured session, depends on setup
 * See docs/superpowers/specs/2026-05-25-e2e-test-suite-design.md
 */
export default defineConfig({
  testDir: './tests/e2e',
  timeout: 60 * 1000,
  fullyParallel: false,
  // One retry absorbs transient hiccups from the live backend / market data
  // (refresh + recalc) without masking real, reproducible failures.
  retries: 1,
  use: {
    baseURL: 'http://localhost:9000',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    trace: 'retain-on-failure',
  },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'public',
      testMatch: /public\/.*\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'auth',
      testMatch: /auth\/.*\.spec\.ts/,
      dependencies: ['setup'],
      use: {
        ...devices['Desktop Chrome'],
        storageState: 'playwright/.auth/user.json',
      },
    },
  ],
  webServer: {
    command: 'make -C ../ docker-up',
    url: 'http://localhost:9000',
    reuseExistingServer: true,
    timeout: 180 * 1000,
  },
});
