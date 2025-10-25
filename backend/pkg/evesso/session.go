package evesso

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// SessionClaims represents JWT claims for EVE character session
type SessionClaims struct {
	CharacterID   int      `json:"character_id"`
	CharacterName string   `json:"character_name"`
	Scopes        []string `json:"scopes"`
	jwt.RegisteredClaims
}

// SessionManager handles JWT session token creation and validation
type SessionManager struct {
	secretKey     []byte
	tokenDuration time.Duration
	issuer        string
}

// NewSessionManager creates a new session manager
func NewSessionManager(secretKey string, duration time.Duration) *SessionManager {
	return &SessionManager{
		secretKey:     []byte(secretKey),
		tokenDuration: duration,
		issuer:        "eve-o-provit",
	}
}

// CreateToken creates a new JWT session token for a character
func (sm *SessionManager) CreateToken(charInfo *CharacterInfo) (string, error) {
	if charInfo == nil {
		return "", errors.New("character info is nil")
	}

	if charInfo.CharacterID == 0 {
		return "", errors.New("invalid character ID")
	}

	scopes := ParseScopes(charInfo.Scopes)

	claims := SessionClaims{
		CharacterID:   charInfo.CharacterID,
		CharacterName: charInfo.CharacterName,
		Scopes:        scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(sm.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    sm.issuer,
			Subject:   fmt.Sprintf("%d", charInfo.CharacterID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(sm.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT session token and returns the claims
func (sm *SessionManager) ValidateToken(tokenString string) (*SessionClaims, error) {
	if tokenString == "" {
		return nil, errors.New("token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return sm.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is invalid")
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok {
		return nil, errors.New("failed to extract claims")
	}

	return claims, nil
}

// RefreshToken creates a new token with extended expiration based on existing claims
func (sm *SessionManager) RefreshToken(oldTokenString string) (string, error) {
	claims, err := sm.ValidateToken(oldTokenString)
	if err != nil {
		return "", fmt.Errorf("failed to validate old token: %w", err)
	}

	// Create new token with updated expiration
	newClaims := SessionClaims{
		CharacterID:   claims.CharacterID,
		CharacterName: claims.CharacterName,
		Scopes:        claims.Scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(sm.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    sm.issuer,
			Subject:   fmt.Sprintf("%d", claims.CharacterID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err := token.SignedString(sm.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refreshed token: %w", err)
	}

	return tokenString, nil
}
