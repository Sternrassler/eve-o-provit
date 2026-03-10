/**
 * EVE Online SSO Client (Frontend)
 * Handles OAuth2 Authorization Code Flow (Backend handles token exchange)
 */

const EVE_SSO_AUTH_URL = "https://login.eveonline.com/v2/oauth/authorize";

/**
 * Generate random state for CSRF protection
 */
export function generateState(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * Build EVE SSO authorization URL (without PKCE — backend handles token exchange)
 */
export async function buildAuthorizationUrl(
  clientId: string,
  redirectUri: string,
  scopes: string[] = []
): Promise<string> {
  const state = generateState();
  sessionStorage.setItem("eve_oauth_state", state);
  const params = new URLSearchParams({
    response_type: "code",
    redirect_uri: redirectUri,
    client_id: clientId,
    scope: scopes.join(" "),
    state: state,
  });
  return `${EVE_SSO_AUTH_URL}?${params.toString()}`;
}

/**
 * Validate OAuth state parameter
 */
export function validateState(receivedState: string): boolean {
  const storedState = sessionStorage.getItem("eve_oauth_state");
  return storedState === receivedState;
}
