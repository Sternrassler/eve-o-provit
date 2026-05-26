package evesso

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const (
	jwksURL       = "https://login.eveonline.com/oauth/jwks"
	audienceEVE   = "EVE Online"
	subjectPrefix = "CHARACTER:EVE:"
)

var validIssuers = map[string]bool{
	"login.eveonline.com":         true,
	"https://login.eveonline.com": true,
}

// TokenValidator validates EVE SSO access tokens (JWTs) locally against the SSO JWKS.
type TokenValidator struct {
	clientID        string
	acceptedClients []string
	keys            jwk.Set
}

// NewTokenValidator creates a validator that fetches and auto-refreshes the EVE SSO JWKS.
func NewTokenValidator(ctx context.Context, clientID string) (*TokenValidator, error) {
	cache := jwk.NewCache(ctx)
	if err := cache.Register(jwksURL); err != nil {
		return nil, fmt.Errorf("register jwks: %w", err)
	}
	if _, err := cache.Refresh(ctx, jwksURL); err != nil {
		return nil, fmt.Errorf("initial jwks fetch: %w", err)
	}
	return &TokenValidator{clientID: clientID, acceptedClients: []string{clientID}, keys: jwk.NewCachedSet(cache, jwksURL)}, nil
}

// NewTokenValidatorWithKeySet creates a validator using a provided key set (useful for tests).
func NewTokenValidatorWithKeySet(clientID string, keys jwk.Set) *TokenValidator {
	return &TokenValidator{clientID: clientID, acceptedClients: []string{clientID}, keys: keys}
}

// AddAcceptedClientID lets the validator accept tokens whose audience contains an
// additional client ID (e.g. the mobile app), alongside the primary one.
func (v *TokenValidator) AddAcceptedClientID(id string) {
	if id == "" {
		return
	}
	v.acceptedClients = append(v.acceptedClients, id)
}

// Validate verifies the token signature, expiration, issuer and audience, then extracts character info.
func (v *TokenValidator) Validate(ctx context.Context, token string) (*CharacterInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("access token is empty")
	}
	tok, err := jwt.Parse([]byte(token), jwt.WithKeySet(v.keys), jwt.WithValidate(true))
	if err != nil {
		return nil, fmt.Errorf("parse/verify token: %w", err)
	}
	if !validIssuers[tok.Issuer()] {
		return nil, fmt.Errorf("invalid issuer: %q", tok.Issuer())
	}
	aud := tok.Audience()
	if !containsString(aud, audienceEVE) || !audienceHasAcceptedClient(aud, v.acceptedClients) {
		return nil, fmt.Errorf("invalid audience: %v", aud)
	}
	charID, err := parseCharacterID(tok.Subject())
	if err != nil {
		return nil, err
	}
	info := &CharacterInfo{CharacterID: charID}
	if name, ok := getStringClaim(tok, "name"); ok {
		info.CharacterName = name
	}
	if owner, ok := getStringClaim(tok, "owner"); ok {
		info.CharacterOwnerHash = owner
	}
	info.Scopes = scopesToString(tok)
	return info, nil
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// audienceHasAcceptedClient returns true if aud contains at least one entry from accepted.
func audienceHasAcceptedClient(aud, accepted []string) bool {
	for _, id := range accepted {
		if containsString(aud, id) {
			return true
		}
	}
	return false
}

func parseCharacterID(sub string) (int, error) {
	if !strings.HasPrefix(sub, subjectPrefix) {
		return 0, fmt.Errorf("unexpected subject format: %q", sub)
	}
	id, err := strconv.Atoi(strings.TrimPrefix(sub, subjectPrefix))
	if err != nil {
		return 0, fmt.Errorf("parse character id from %q: %w", sub, err)
	}
	return id, nil
}

func getStringClaim(tok jwt.Token, key string) (string, bool) {
	if raw, ok := tok.Get(key); ok {
		if s, ok := raw.(string); ok {
			return s, true
		}
	}
	return "", false
}

func scopesToString(tok jwt.Token) string {
	raw, ok := tok.Get("scp")
	if !ok {
		return ""
	}
	switch val := raw.(type) {
	case string:
		return val
	case []string:
		return strings.Join(val, " ")
	case []interface{}:
		parts := make([]string, 0, len(val))
		for _, e := range val {
			if s, ok := e.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}
