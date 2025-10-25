package evesso

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

const (
	AuthURL   = "https://login.eveonline.com/v2/oauth/authorize"
	TokenURL  = "https://login.eveonline.com/v2/oauth/token"
	VerifyURL = "https://esi.evetech.net/verify/"
)

// Config holds OAuth2 configuration for EVE SSO
type Config struct {
	ClientID     string
	ClientSecret string
	CallbackURL  string
	Scopes       []string
}

// Client represents an EVE SSO OAuth2 client
type Client struct {
	config       *oauth2.Config
	clientID     string
	clientSecret string
}

// CharacterInfo represents verified character information from ESI
type CharacterInfo struct {
	CharacterID          int    `json:"CharacterID"`
	CharacterName        string `json:"CharacterName"`
	ExpiresOn            string `json:"ExpiresOn"`
	Scopes               string `json:"Scopes"`
	TokenType            string `json:"TokenType"`
	CharacterOwnerHash   string `json:"CharacterOwnerHash"`
	IntellectualProperty string `json:"IntellectualProperty"`
}

// NewClient creates a new EVE SSO OAuth2 client
func NewClient(cfg *Config) *Client {
	return &Client{
		config: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  AuthURL,
				TokenURL: TokenURL,
			},
			RedirectURL: cfg.CallbackURL,
			Scopes:      cfg.Scopes,
		},
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}
}

// GenerateState creates a random state parameter for CSRF protection
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthURL returns the authorization URL for EVE SSO
func (c *Client) GetAuthURL(state string) string {
	return c.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// ExchangeCode exchanges an authorization code for an access token
func (c *Client) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	if code == "" {
		return nil, errors.New("authorization code is empty")
	}

	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// VerifyCharacter verifies the access token and returns character information
func (c *Client) VerifyCharacter(ctx context.Context, accessToken string) (*CharacterInfo, error) {
	if accessToken == "" {
		return nil, errors.New("access token is empty")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", VerifyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create verify request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify character: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("verify request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var charInfo CharacterInfo
	if err := json.NewDecoder(resp.Body).Decode(&charInfo); err != nil {
		return nil, fmt.Errorf("failed to decode character info: %w", err)
	}

	return &charInfo, nil
}

// RefreshToken refreshes an access token using the refresh token
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is empty")
	}

	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := c.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// ParseScopes parses the scopes string from character info into a slice
func ParseScopes(scopesStr string) []string {
	if scopesStr == "" {
		return []string{}
	}
	return strings.Split(scopesStr, " ")
}
