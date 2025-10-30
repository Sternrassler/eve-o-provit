# EVE SSO Token Verification Package

This package provides **server-side token verification** for EVE Online SSO access tokens.

> **Architecture Note:** OAuth2 flow and session management happen **client-side** (Next.js frontend with PKCE).
> This backend package **only** verifies Bearer tokens received from the frontend.

## Features

- **Token Verification**: Validates EVE SSO access tokens via ESI `/verify/` endpoint
- **Auth Middleware**: Protects routes and extracts character information from Bearer tokens
- **No Session Management**: Stateless verification only

## Usage

### Token Verification

```go
import "github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"

// Verify a Bearer token
charInfo, err := evesso.VerifyToken(ctx, accessToken)
if err != nil {
    // Token invalid or expired
}

// Access character information
fmt.Printf("Character: %s (ID: %d)\n", charInfo.CharacterName, charInfo.CharacterID)
```

### Middleware

Protect routes with the `AuthMiddleware`:

```go
import "github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"

protected := api.Group("/api/v1")
protected.Use(evesso.AuthMiddleware)

protected.Get("/character", func(c *fiber.Ctx) error {
    characterID := c.Locals("character_id").(int)
    characterName := c.Locals("character_name").(string)
    
    return c.JSON(fiber.Map{
        "character_id": characterID,
        "character_name": characterName,
    })
})
```

## How It Works

1. **Frontend** handles OAuth2 PKCE flow with EVE SSO
2. **Frontend** stores access token (localStorage)
3. **Frontend** sends `Authorization: Bearer <token>` to backend
4. **Backend** verifies token via ESI `/verify/` endpoint
5. **Backend** extracts character info and allows/denies access

## Security

- **Stateless**: No session storage on backend
- **Bearer Token Validation**: Each request verified via ESI
- **Character Info Extraction**: ID, Name, Scopes available in handlers via `c.Locals()`

## Testing

Run tests with:

```bash
go test -v ./pkg/evesso/...
```

## References

- [EVE SSO Documentation](https://docs.esi.evetech.net/docs/sso/)
- [ESI Verification Endpoint](https://esi.evetech.net/verify/)
