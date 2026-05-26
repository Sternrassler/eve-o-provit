import type { Page } from '@playwright/test';
import { WAYPOINT_ENDPOINT_GLOB } from './constants';

/**
 * Intercept the autopilot waypoint POST so the test exercises the full UI path
 * without writing a real waypoint into the live game. Returns a counter of how
 * many times the endpoint was hit (the card calls it twice: buy then sell).
 */
export function mockWaypoint(page: Page): { count: () => number } {
  let hits = 0;
  void page.route(WAYPOINT_ENDPOINT_GLOB, async (route) => {
    hits += 1;
    await route.fulfill({
      status: 204,
      contentType: 'application/json',
      body: '',
    });
  });
  return { count: () => hits };
}
