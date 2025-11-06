/**
 * EVE Online SSO Client (Frontend)
 * Handles OAuth2 Authorization Code Flow with PKCE
 */

const EVE_SSO_AUTH_URL = "https://login.eveonline.com/v2/oauth/authorize";
const EVE_SSO_TOKEN_URL = "https://login.eveonline.com/v2/oauth/token";
const EVE_ESI_VERIFY_URL = "https://esi.evetech.net/verify/";

interface EVETokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  refresh_token?: string;
}

interface EVECharacterInfo {
  CharacterID: number;
  CharacterName: string;
  ExpiresOn: string;
  Scopes: string;
  TokenType: string;
  CharacterOwnerHash: string;
  IntellectualProperty: string;
}

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
 * Generate PKCE code verifier
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
 * Generate PKCE code challenge from verifier
 */
export async function generateCodeChallenge(
  verifier: string
): Promise<string> {
  const encoder = new TextEncoder();
  const data = encoder.encode(verifier);
  const hash = await crypto.subtle.digest("SHA-256", data);
  return btoa(String.fromCharCode(...new Uint8Array(hash)))
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * Build EVE SSO authorization URL
 */
export async function buildAuthorizationUrl(
  clientId: string,
  redirectUri: string,
  scopes: string[] = []
): Promise<string> {
  const state = generateState();
  const codeVerifier = generateCodeVerifier();
  const codeChallenge = await generateCodeChallenge(codeVerifier);

  // Store state and verifier for validation
  sessionStorage.setItem("eve_oauth_state", state);
  sessionStorage.setItem("eve_oauth_verifier", codeVerifier);

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
 * Exchange authorization code for access token
 */
export async function exchangeCodeForToken(
  code: string,
  clientId: string
): Promise<EVETokenResponse> {
  const codeVerifier = sessionStorage.getItem("eve_oauth_verifier");
  
  if (!codeVerifier) {
    throw new Error("Missing code verifier - possible CSRF attack");
  }

  const params = new URLSearchParams({
    grant_type: "authorization_code",
    code: code,
    client_id: clientId,
    code_verifier: codeVerifier,
  });

  const response = await fetch(EVE_SSO_TOKEN_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: params.toString(),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Token exchange failed: ${response.status} - ${error}`);
  }

  const token = await response.json();
  
  // Clean up session storage
  sessionStorage.removeItem("eve_oauth_state");
  sessionStorage.removeItem("eve_oauth_verifier");
  
  return token;
}

/**
 * Verify access token and get character info
 */
export async function verifyToken(
  accessToken: string
): Promise<EVECharacterInfo> {
  const response = await fetch(EVE_ESI_VERIFY_URL, {
    headers: {
      Authorization: `Bearer ${accessToken}`,
    },
  });

  if (!response.ok) {
    throw new Error(`Token verification failed: ${response.status}`);
  }

  return response.json();
}

/**
 * Refresh access token using refresh token
 */
export async function refreshAccessToken(
  refreshToken: string,
  clientId: string
): Promise<EVETokenResponse> {
  const params = new URLSearchParams({
    grant_type: "refresh_token",
    refresh_token: refreshToken,
    client_id: clientId,
  });

  const response = await fetch(EVE_SSO_TOKEN_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: params.toString(),
  });

  if (!response.ok) {
    const error = await response.text();
    throw new Error(`Token refresh failed: ${response.status} - ${error}`);
  }

  return response.json();
}

/**
 * Validate OAuth state parameter
 */
export function validateState(receivedState: string): boolean {
  const storedState = sessionStorage.getItem("eve_oauth_state");
  return storedState === receivedState;
}

/**
 * Token storage utilities
 */
export const TokenStorage = {
  save(token: EVETokenResponse): void {
    localStorage.setItem("eve_access_token", token.access_token);
    if (token.refresh_token) {
      localStorage.setItem("eve_refresh_token", token.refresh_token);
    }
    const expiresAt = Date.now() + token.expires_in * 1000;
    localStorage.setItem("eve_token_expires_at", expiresAt.toString());
  },

  getAccessToken(): string | null {
    return localStorage.getItem("eve_access_token");
  },

  getRefreshToken(): string | null {
    return localStorage.getItem("eve_refresh_token");
  },

  isExpired(): boolean {
    const expiresAt = localStorage.getItem("eve_token_expires_at");
    if (!expiresAt) return true;
    return Date.now() >= parseInt(expiresAt, 10);
  },

  getTimeUntilExpiry(): number {
    const expiresAt = localStorage.getItem("eve_token_expires_at");
    if (!expiresAt) return 0;
    return parseInt(expiresAt, 10) - Date.now();
  },

  shouldRefresh(): boolean {
    const timeUntilExpiry = this.getTimeUntilExpiry();
    // Refresh 3 minutes before expiry
    return timeUntilExpiry > 0 && timeUntilExpiry < 3 * 60 * 1000;
  },

  clear(): void {
    localStorage.removeItem("eve_access_token");
    localStorage.removeItem("eve_refresh_token");
    localStorage.removeItem("eve_token_expires_at");
    localStorage.removeItem("eve_character_info");
  },

  saveCharacterInfo(info: EVECharacterInfo): void {
    localStorage.setItem("eve_character_info", JSON.stringify(info));
  },

  getCharacterInfo(): EVECharacterInfo | null {
    const data = localStorage.getItem("eve_character_info");
    return data ? JSON.parse(data) : null;
  },
};
