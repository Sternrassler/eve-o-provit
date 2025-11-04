# EVE SSO Integration Guide

This guide explains how to set up and use the EVE Online SSO authentication in the EVE-O-Provit application.

## Overview

The application implements a **Frontend-only OAuth2 flow with PKCE** (ADR-004) using EVE Online's Single Sign-On (SSO) system. This allows users to log in with their EVE characters and grants the application access to ESI (EVE Swagger Interface) endpoints.

**Security Model:** Authorization Code Flow with PKCE (Proof Key for Code Exchange) - no Client Secret needed, tokens stored in browser.

## Architecture

### Frontend (Next.js) - PKCE Flow

- **Framework**: Next.js 14 (App Router)
- **Authentication**: OAuth2 Authorization Code Flow with PKCE (ADR-004)
- **State Management**: React Context API (`auth-context.tsx`)
- **Token Storage**: Browser localStorage (encrypted with PKCE)
- **Session Duration**: Based on ESI token expiry (~20 minutes)
- **Components**:
  - `AuthProvider` - Manages authentication state
  - `EveLoginButton` - Initiates login flow
  - `CharacterInfo` - Displays logged-in character
  - `/callback` page - Handles OAuth callback

## Setup Instructions

### 1. EVE Developer Application

**Note**: The Client ID shown below is the actual application ID from the issue requirements. Client IDs are public and safe to share (they're used in browser-side OAuth flows). Only the Client Secret must be kept private.

1. Go to [EVE Developer Portal](https://developers.eveonline.com/)
2. Create a new application or use existing:
   - **Name**: EVE Profit Maximizer
   - **Client ID**: `0828b4bcd20242aeb9b8be10f5451094` (from issue requirements)
   - **Callback URL**: `http://localhost:9000/callback` ⚠️ **Frontend Port!**
   - **Connection Type**: Authentication & API Access
   - **Scopes**: Enable the following:
     - `publicData`
     - `esi-location.read_location.v1`
     - `esi-location.read_ship_type.v1`
     - `esi-skills.read_skills.v1`
     - `esi-wallet.read_character_wallet.v1`
     - `esi-universe.read_structures.v1`
     - `esi-assets.read_assets.v1`
     - `esi-fittings.read_fittings.v1`
     - `esi-characters.read_standings.v1`
     - `esi-ui.write_waypoint.v1` (for autopilot route setting)
     - `esi-markets.read_character_orders.v1` (for character market orders)
3. **No Client Secret needed** - PKCE flow uses code_verifier instead!

### 2. Frontend Configuration

Create `frontend/.env.local` file:

```env
# EVE SSO (Public - used in browser)
NEXT_PUBLIC_EVE_CLIENT_ID=0828b4bcd20242aeb9b8be10f5451094
NEXT_PUBLIC_EVE_CALLBACK_URL=http://localhost:9000/callback

# Backend API URL
NEXT_PUBLIC_API_URL=http://localhost:9001

# Optional: Override scopes (defaults in auth-context.tsx)
NEXT_PUBLIC_EVE_SCOPES=esi-location.read_location.v1,esi-location.read_ship_type.v1,esi-clones.read_clones.v1,esi-assets.read_assets.v1,esi-ui.write_waypoint.v1
```

**Note:** No secrets needed! PKCE flow uses dynamically generated code_verifier.

### 3. Start the Application

**Backend (for Market Data API):**

```bash
cd backend
go run ./cmd/api/main.go
```

**Frontend (handles OAuth):**

```bash
cd frontend
npm run dev
```

## Authentication Flow

1. User clicks "Login with EVE" button
2. Frontend redirects to `GET /api/v1/auth/login`
3. Backend generates a state parameter and redirects to EVE SSO
4. User authorizes the application on EVE's login page
5. EVE redirects back to `GET /api/v1/auth/callback?code=...&state=...`
6. Backend:
   - Validates state parameter (CSRF protection)
   - Exchanges authorization code for access token
   - Calls ESI `/verify` to get character information
   - Creates JWT session token
   - Sets HttpOnly cookie with session token
7. Frontend `/callback` page verifies session and redirects to home
8. Navigation component displays character information

## API Endpoints

### `GET /api/v1/auth/login`

Initiates OAuth2 flow by redirecting to EVE SSO.

**Response:** HTTP 307 Redirect to EVE SSO

---

### `GET /api/v1/auth/callback`

Handles OAuth2 callback from EVE SSO.

**Query Parameters:**

- `code` - Authorization code
- `state` - CSRF protection token

**Response:** HTTP 307 Redirect to frontend with session cookie set

---

### `POST /api/v1/auth/logout`

Invalidates the current session.

**Response:**

```json
{
  "message": "Logged out successfully"
}
```

---

### `GET /api/v1/auth/verify`

Verifies the current session and returns session information.

**Response:**

```json
{
  "authenticated": true,
  "character_id": 123456789,
  "character_name": "Example Pilot",
  "scopes": ["publicData", "esi-location.read_location.v1"],
  "expires_at": "2025-10-26T12:00:00Z"
}
```

---

### `POST /api/v1/auth/refresh`

Refreshes the session token with a new expiration time.

**Response:**

```json
{
  "message": "Session refreshed successfully"
}
```

---

### `GET /api/v1/auth/character`

Returns detailed character information.

**Response:**

```json
{
  "character_id": 123456789,
  "character_name": "Example Pilot",
  "scopes": ["publicData", "esi-location.read_location.v1"],
  "portrait_url": "https://images.evetech.net/characters/123456789/portrait?size=128"
}
```

## Frontend Integration

### Using the Auth Context

```tsx
import { useAuth } from "@/lib/auth-context";

function MyComponent() {
  const { isAuthenticated, character, login, logout } = useAuth();

  if (!isAuthenticated) {
    return <button onClick={login}>Login</button>;
  }

  return (
    <div>
      <p>Welcome, {character?.character_name}!</p>
      <button onClick={logout}>Logout</button>
    </div>
  );
}
```

### Protected Routes (Optional)

Create a wrapper component:

```tsx
"use client";

import { useAuth } from "@/lib/auth-context";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.push("/");
    }
  }, [isAuthenticated, isLoading, router]);

  if (isLoading) return <div>Loading...</div>;
  if (!isAuthenticated) return null;

  return <>{children}</>;
}
```

## Security Considerations

### Development vs Production

**Development (HTTP):**

- Cookies use `Secure: false` to work with HTTP
- Callback URL: `http://localhost:9001/api/v1/auth/callback`
- Frontend: `http://localhost:9000`
- Backend: `http://localhost:9001`

**Production (HTTPS):**

- Cookies must use `Secure: true`
- Callback URL: `https://your-domain.com/api/v1/auth/callback`
- Update EVE application callback URL in developer portal

### Cookie Settings

- **HttpOnly**: Prevents JavaScript access (XSS protection)
- **Secure**: Only sent over HTTPS (production)
- **SameSite**: `Strict` for session, `Lax` for state (CSRF protection)
- **MaxAge**: 24 hours for session, 5 minutes for state

### State Parameter

The state parameter is a random 32-byte value that:

- Prevents CSRF attacks
- Is stored in a short-lived cookie (5 minutes)
- Must match between the initial request and callback

### JWT Tokens

- Signed with HMAC-SHA256
- Include character ID, name, and scopes
- Have a configurable expiration time
- Cannot be modified without invalidating the signature

## Troubleshooting

### Login Fails with "Invalid state parameter"

- Check that cookies are enabled in your browser
- Verify CORS is configured to allow credentials
- Ensure the frontend and backend URLs match the configuration

### "Failed to exchange authorization code"

- Verify `EVE_CLIENT_ID` and `EVE_CLIENT_SECRET` are correct
- Check that the callback URL matches exactly in EVE developer portal
- Ensure scopes match between code and EVE application

### Session Expires Too Quickly

- Check `SESSION_DURATION` in backend `.env`
- Default is 24 hours
- Frontend auto-refreshes every 15 minutes

### Character Portrait Not Loading

- Verify the character ID is correct
- Check browser console for CORS errors
- EVE image server: `https://images.evetech.net/`

## Next Steps

After successful authentication, you can:

1. **Access ESI Endpoints**: Use the access token to call EVE ESI API
2. **Store Character Data**: Save character information to database
3. **Multi-Character Support**: Allow users to add multiple characters
4. **Persistent Sessions**: Store sessions in PostgreSQL/Redis for multi-instance deployments

## References

- [EVE SSO Documentation](https://docs.esi.evetech.net/docs/sso/)
- [ESI Swagger Interface](https://esi.evetech.net/ui/)
- [OAuth2 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)
- [EVE Developer Portal](https://developers.eveonline.com/)
