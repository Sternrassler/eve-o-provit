/**
 * EVE Online SSO Client (Frontend)
 * Handles OAuth2 Authorization Code Flow with PKCE (public client)
 * Backend receives code + code_verifier to exchange tokens without client_secret
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
 * Generate a cryptographically random PKCE code_verifier (43–128 chars, base64url)
 */
export function generateCodeVerifier(): string {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return btoa(String.fromCharCode(...array))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * Derive PKCE code_challenge (S256) from code_verifier
 */
export async function generateCodeChallenge(verifier: string): Promise<string> {
  const data = new TextEncoder().encode(verifier);
  const digest = await crypto.subtle.digest("SHA-256", data);
  return btoa(String.fromCharCode(...new Uint8Array(digest)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * Build EVE SSO authorization URL with PKCE (code_challenge_method=S256)
 */
export async function buildAuthorizationUrl(
  clientId: string,
  redirectUri: string,
  scopes: string[] = []
): Promise<string> {
  const state = generateState();
  const codeVerifier = generateCodeVerifier();
  const codeChallenge = await generateCodeChallenge(codeVerifier);

  sessionStorage.setItem("eve_oauth_state", state);
  sessionStorage.setItem("eve_code_verifier", codeVerifier);

  const params = new URLSearchParams({
    response_type: "code",
    redirect_uri: redirectUri,
    client_id: clientId,
    scope: scopes.join(" "),
    state: state,
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
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
