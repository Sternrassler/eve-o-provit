package evesso

import (
	"context"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		CallbackURL:  "http://localhost:9000/callback",
		Scopes:       []string{"publicData"},
	}

	client := NewClient(cfg)
	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.clientID != cfg.ClientID {
		t.Errorf("Expected clientID %s, got %s", cfg.ClientID, client.clientID)
	}

	if client.config.RedirectURL != cfg.CallbackURL {
		t.Errorf("Expected redirect URL %s, got %s", cfg.CallbackURL, client.config.RedirectURL)
	}
}

func TestGenerateState(t *testing.T) {
	state1, err := GenerateState()
	if err != nil {
		t.Fatalf("Failed to generate state: %v", err)
	}

	if state1 == "" {
		t.Error("Expected non-empty state")
	}

	state2, err := GenerateState()
	if err != nil {
		t.Fatalf("Failed to generate second state: %v", err)
	}

	if state1 == state2 {
		t.Error("Expected different state values for different calls")
	}
}

func TestGetAuthURL(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		CallbackURL:  "http://localhost:9000/callback",
		Scopes:       []string{"publicData", "esi-location.read_location.v1"},
	}

	client := NewClient(cfg)
	state := "test-state-123"

	authURL := client.GetAuthURL(state)

	if authURL == "" {
		t.Error("Expected non-empty auth URL")
	}

	// Check that URL contains expected components (some may be URL encoded)
	expectedComponents := []string{
		AuthURL,
		"test-client-id",
		"test-state-123",
	}

	for _, component := range expectedComponents {
		if !contains(authURL, component) {
			t.Errorf("Expected auth URL to contain %s", component)
		}
	}
}

func TestExchangeCode_EmptyCode(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		CallbackURL:  "http://localhost:9000/callback",
		Scopes:       []string{"publicData"},
	}

	client := NewClient(cfg)
	ctx := context.Background()

	_, err := client.ExchangeCode(ctx, "")
	if err == nil {
		t.Error("Expected error for empty code")
	}
}

func TestVerifyCharacter_EmptyToken(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		CallbackURL:  "http://localhost:9000/callback",
		Scopes:       []string{"publicData"},
	}

	client := NewClient(cfg)
	ctx := context.Background()

	_, err := client.VerifyCharacter(ctx, "")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestRefreshToken_EmptyToken(t *testing.T) {
	cfg := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		CallbackURL:  "http://localhost:9000/callback",
		Scopes:       []string{"publicData"},
	}

	client := NewClient(cfg)
	ctx := context.Background()

	_, err := client.RefreshToken(ctx, "")
	if err == nil {
		t.Error("Expected error for empty refresh token")
	}
}

func TestParseScopes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "Single scope",
			input:    "publicData",
			expected: []string{"publicData"},
		},
		{
			name:     "Multiple scopes",
			input:    "publicData esi-location.read_location.v1 esi-skills.read_skills.v1",
			expected: []string{"publicData", "esi-location.read_location.v1", "esi-skills.read_skills.v1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseScopes(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d scopes, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected scope[%d] to be %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestNewSessionManager(t *testing.T) {
	secretKey := "test-secret-key-12345678"
	duration := 24 * time.Hour

	sm := NewSessionManager(secretKey, duration)
	if sm == nil {
		t.Fatal("Expected session manager to be created")
	}

	if sm.tokenDuration != duration {
		t.Errorf("Expected token duration %v, got %v", duration, sm.tokenDuration)
	}
}

func TestSessionManager_CreateToken(t *testing.T) {
	sm := NewSessionManager("test-secret-key-12345678", 24*time.Hour)

	charInfo := &CharacterInfo{
		CharacterID:   123456789,
		CharacterName: "Test Pilot",
		Scopes:        "publicData esi-location.read_location.v1",
	}

	token, err := sm.CreateToken(charInfo)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	if token == "" {
		t.Error("Expected non-empty token")
	}
}

func TestSessionManager_CreateToken_NilCharInfo(t *testing.T) {
	sm := NewSessionManager("test-secret-key-12345678", 24*time.Hour)

	_, err := sm.CreateToken(nil)
	if err == nil {
		t.Error("Expected error for nil character info")
	}
}

func TestSessionManager_CreateToken_InvalidCharacterID(t *testing.T) {
	sm := NewSessionManager("test-secret-key-12345678", 24*time.Hour)

	charInfo := &CharacterInfo{
		CharacterID:   0,
		CharacterName: "Test Pilot",
		Scopes:        "publicData",
	}

	_, err := sm.CreateToken(charInfo)
	if err == nil {
		t.Error("Expected error for invalid character ID")
	}
}

func TestSessionManager_ValidateToken(t *testing.T) {
	sm := NewSessionManager("test-secret-key-12345678", 24*time.Hour)

	charInfo := &CharacterInfo{
		CharacterID:   123456789,
		CharacterName: "Test Pilot",
		Scopes:        "publicData esi-location.read_location.v1",
	}

	token, err := sm.CreateToken(charInfo)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	claims, err := sm.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.CharacterID != charInfo.CharacterID {
		t.Errorf("Expected character ID %d, got %d", charInfo.CharacterID, claims.CharacterID)
	}

	if claims.CharacterName != charInfo.CharacterName {
		t.Errorf("Expected character name %s, got %s", charInfo.CharacterName, claims.CharacterName)
	}

	expectedScopes := ParseScopes(charInfo.Scopes)
	if len(claims.Scopes) != len(expectedScopes) {
		t.Errorf("Expected %d scopes, got %d", len(expectedScopes), len(claims.Scopes))
	}
}

func TestSessionManager_ValidateToken_EmptyToken(t *testing.T) {
	sm := NewSessionManager("test-secret-key-12345678", 24*time.Hour)

	_, err := sm.ValidateToken("")
	if err == nil {
		t.Error("Expected error for empty token")
	}
}

func TestSessionManager_ValidateToken_InvalidToken(t *testing.T) {
	sm := NewSessionManager("test-secret-key-12345678", 24*time.Hour)

	_, err := sm.ValidateToken("invalid.token.here")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestSessionManager_ValidateToken_WrongSecret(t *testing.T) {
	sm1 := NewSessionManager("secret-key-1", 24*time.Hour)
	sm2 := NewSessionManager("secret-key-2", 24*time.Hour)

	charInfo := &CharacterInfo{
		CharacterID:   123456789,
		CharacterName: "Test Pilot",
		Scopes:        "publicData",
	}

	token, err := sm1.CreateToken(charInfo)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	_, err = sm2.ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating token with wrong secret")
	}
}

func TestSessionManager_RefreshToken(t *testing.T) {
	sm := NewSessionManager("test-secret-key-12345678", 1*time.Hour)

	charInfo := &CharacterInfo{
		CharacterID:   123456789,
		CharacterName: "Test Pilot",
		Scopes:        "publicData",
	}

	oldToken, err := sm.CreateToken(charInfo)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Wait to ensure new token has different timestamp (JWT uses second precision)
	time.Sleep(1100 * time.Millisecond)

	newToken, err := sm.RefreshToken(oldToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if newToken == oldToken {
		t.Error("Expected new token to be different from old token")
	}

	// Validate new token
	claims, err := sm.ValidateToken(newToken)
	if err != nil {
		t.Fatalf("Failed to validate refreshed token: %v", err)
	}

	if claims.CharacterID != charInfo.CharacterID {
		t.Errorf("Expected character ID %d, got %d", charInfo.CharacterID, claims.CharacterID)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(containsAt(s, substr, 0) || containsAfter(s, substr)))
}

func containsAt(s, substr string, pos int) bool {
	if pos+len(substr) > len(s) {
		return false
	}
	return s[pos:pos+len(substr)] == substr
}

func containsAfter(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if containsAt(s, substr, i) {
			return true
		}
	}
	return false
}
