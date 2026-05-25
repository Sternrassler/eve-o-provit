/** Shared E2E constants. */

/** Backend base URL (matches NEXT_PUBLIC_API_URL default in lib/api-client.ts). */
export const API_BASE_URL = 'http://localhost:9001';

/** High-liquidity region guaranteed to have market data: The Forge (Jita). */
export const REGION_THE_FORGE = { id: 10000002, name: 'The Forge' };

/** Backend auth endpoints (cookie-based session). */
export const AUTH_ENDPOINTS = {
  session: `${API_BASE_URL}/auth/session`,
  logout: `${API_BASE_URL}/auth/logout`,
  refresh: `${API_BASE_URL}/auth/refresh`,
};

/** Autopilot waypoint endpoint (matches components/trading/TradingRouteCard.tsx). */
export const WAYPOINT_ENDPOINT_GLOB = '**/api/v1/esi/ui/autopilot/waypoint';
