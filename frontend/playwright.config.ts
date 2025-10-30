import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Test Configuration
 * Siehe https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests/e2e',
  
  // Timeout für einzelne Tests
  timeout: 60 * 1000, // 60 Sekunden (OAuth Flow kann länger dauern)
  
  // Erwarte dass Services laufen (make docker-rebuild)
  fullyParallel: false,
  
  // Keine Retries in CI (echte OAuth Tests sollten stabil sein)
  retries: 0,
  
  // Reporter
  use: {
    // Base URL (Frontend)
    baseURL: 'http://localhost:9000',
    
    // Screenshot bei Fehler
    screenshot: 'only-on-failure',
    
    // Video bei Fehler
    video: 'retain-on-failure',
    
    // Trace bei Fehler
    trace: 'retain-on-failure',
  },

  // Chromium headless für CI
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // Web Server (optional - falls noch nicht gestartet)
  // webServer: {
  //   command: 'make docker-up',
  //   url: 'http://localhost:9000',
  //   reuseExistingServer: true,
  //   timeout: 120 * 1000,
  // },
});
