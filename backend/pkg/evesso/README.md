# EVE SSO OAuth2 Package

This package provides EVE Online Single Sign-On (SSO) authentication using OAuth2.

## Features

- **OAuth2 Authorization Code Flow**: Complete implementation of EVE SSO OAuth2 flow
- **JWT Session Management**: Secure session tokens with configurable duration
- **Character Verification**: Validates access tokens and retrieves character information
- **Token Refresh**: Automatic token refresh support
- **Fiber HTTP Handlers**: Ready-to-use handlers for Fiber framework

## Usage

### Configuration

```go
import "github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"

// Create EVE SSO client
ssoClient := evesso.NewClient(&evesso.Config{
    ClientID:     "your-eve-client-id",
    ClientSecret: "your-eve-client-secret",
    CallbackURL:  "http://localhost:8082/api/v1/auth/callback",
    Scopes:       []string{"publicData", "esi-location.read_location.v1"},
})

// Create session manager
sessionManager := evesso.NewSessionManager("your-jwt-secret", 24*time.Hour)

// Create HTTP handler
authHandler := evesso.NewHandler(ssoClient, sessionManager)
```

### HTTP Endpoints

The package provides the following handlers:

- `GET /auth/login` - Initiates OAuth2 flow and redirects to EVE SSO
- `GET /auth/callback` - Processes OAuth2 callback and creates session
- `POST /auth/logout` - Invalidates session
- `GET /auth/verify` - Verifies current session
- `POST /auth/refresh` - Refreshes session token
- `GET /auth/character` - Returns current character information

### Middleware

Use the `AuthMiddleware` to protect routes:

```go
protected := api.Group("/protected")
protected.Use(authHandler.AuthMiddleware)
protected.Get("/data", func(c *fiber.Ctx) error {
    characterID := c.Locals("character_id").(int)
    return c.JSON(fiber.Map{"character_id": characterID})
})
```

## Environment Variables

- `EVE_CLIENT_ID` - EVE application client ID
- `EVE_CLIENT_SECRET` - EVE application client secret
- `EVE_CALLBACK_URL` - OAuth2 callback URL
- `EVE_SCOPES` - Space or comma-separated list of ESI scopes
- `JWT_SECRET` - Secret key for JWT signing
- `SESSION_DURATION` - Session token duration (e.g., "24h")

## Security

- Session tokens are stored in **HttpOnly cookies** to prevent XSS attacks
- **State parameter** validation prevents CSRF attacks
- JWT tokens are signed with HMAC-SHA256
- Supports **Secure** and **SameSite** cookie attributes for production

## Testing

Run tests with:

```bash
go test -v ./pkg/evesso/...
```

## References

- [EVE SSO Documentation](https://docs.esi.evetech.net/docs/sso/)
- [OAuth2 RFC 6749](https://datatracker.ietf.org/doc/html/rfc6749)
