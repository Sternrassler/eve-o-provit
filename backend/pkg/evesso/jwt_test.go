package evesso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

const testClientID = "test-client-123"

func newTestValidator(t *testing.T) (*TokenValidator, jwk.Key) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	privJWK, err := jwk.FromRaw(priv)
	if err != nil {
		t.Fatalf("priv jwk: %v", err)
	}
	_ = privJWK.Set(jwk.KeyIDKey, "test-kid")
	_ = privJWK.Set(jwk.AlgorithmKey, jwa.RS256)
	pubJWK, err := privJWK.PublicKey()
	if err != nil {
		t.Fatalf("pub jwk: %v", err)
	}
	set := jwk.NewSet()
	_ = set.AddKey(pubJWK)
	v := NewTokenValidatorWithKeySet(testClientID, set)
	return v, privJWK
}

func signToken(t *testing.T, signer jwk.Key, iss string, aud []string, sub string, exp time.Time, extra map[string]interface{}) string {
	t.Helper()
	builder := jwt.NewBuilder().Issuer(iss).Subject(sub).Expiration(exp).IssuedAt(time.Now())
	// jwx v2 builder .Audience overwrites on each call, so set the full slice once.
	builder = builder.Audience(aud)
	for k, val := range extra {
		builder = builder.Claim(k, val)
	}
	tok, err := builder.Build()
	if err != nil {
		t.Fatalf("build token: %v", err)
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, signer))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return string(signed)
}

func validClaims() map[string]interface{} {
	return map[string]interface{}{
		"name":  "Test Pilot",
		"scp":   []string{"esi-location.read_location.v1", "esi-skills.read_skills.v1"},
		"owner": "ownerhash123",
	}
}

func TestValidate_ValidToken(t *testing.T) {
	v, signer := newTestValidator(t)
	tok := signToken(t, signer, "login.eveonline.com", []string{testClientID, "EVE Online"}, "CHARACTER:EVE:90000001", time.Now().Add(20*time.Minute), validClaims())
	info, err := v.Validate(context.Background(), tok)
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if info.CharacterID != 90000001 {
		t.Errorf("CharacterID = %d, want 90000001", info.CharacterID)
	}
	if info.CharacterName != "Test Pilot" {
		t.Errorf("CharacterName = %q", info.CharacterName)
	}
	if info.CharacterOwnerHash != "ownerhash123" {
		t.Errorf("OwnerHash = %q", info.CharacterOwnerHash)
	}
	if info.Scopes != "esi-location.read_location.v1 esi-skills.read_skills.v1" {
		t.Errorf("Scopes = %q", info.Scopes)
	}
}

func TestValidate_AcceptsBothIssuers(t *testing.T) {
	v, signer := newTestValidator(t)
	for _, iss := range []string{"login.eveonline.com", "https://login.eveonline.com"} {
		tok := signToken(t, signer, iss, []string{testClientID, "EVE Online"}, "CHARACTER:EVE:1", time.Now().Add(10*time.Minute), validClaims())
		if _, err := v.Validate(context.Background(), tok); err != nil {
			t.Errorf("issuer %q should be accepted: %v", iss, err)
		}
	}
}

func TestValidate_RejectsBadIssuer(t *testing.T) {
	v, signer := newTestValidator(t)
	tok := signToken(t, signer, "evil.example.com", []string{testClientID, "EVE Online"}, "CHARACTER:EVE:1", time.Now().Add(10*time.Minute), validClaims())
	if _, err := v.Validate(context.Background(), tok); err == nil {
		t.Error("bad issuer must be rejected")
	}
}

func TestValidate_RejectsExpired(t *testing.T) {
	v, signer := newTestValidator(t)
	tok := signToken(t, signer, "login.eveonline.com", []string{testClientID, "EVE Online"}, "CHARACTER:EVE:1", time.Now().Add(-1*time.Minute), validClaims())
	if _, err := v.Validate(context.Background(), tok); err == nil {
		t.Error("expired token must be rejected")
	}
}

func TestValidate_RejectsMissingAudience(t *testing.T) {
	v, signer := newTestValidator(t)
	tok1 := signToken(t, signer, "login.eveonline.com", []string{"EVE Online"}, "CHARACTER:EVE:1", time.Now().Add(10*time.Minute), validClaims())
	if _, err := v.Validate(context.Background(), tok1); err == nil {
		t.Error("aud ohne clientID muss abgelehnt werden")
	}
	tok2 := signToken(t, signer, "login.eveonline.com", []string{testClientID}, "CHARACTER:EVE:1", time.Now().Add(10*time.Minute), validClaims())
	if _, err := v.Validate(context.Background(), tok2); err == nil {
		t.Error("aud ohne \"EVE Online\" muss abgelehnt werden")
	}
}

func TestValidate_RejectsWrongSignature(t *testing.T) {
	v, _ := newTestValidator(t)
	_, otherSigner := newTestValidator(t)
	tok := signToken(t, otherSigner, "login.eveonline.com", []string{testClientID, "EVE Online"}, "CHARACTER:EVE:1", time.Now().Add(10*time.Minute), validClaims())
	if _, err := v.Validate(context.Background(), tok); err == nil {
		t.Error("falsch signiertes Token muss abgelehnt werden")
	}
}

func TestValidate_RejectsEmptyToken(t *testing.T) {
	v, _ := newTestValidator(t)
	if _, err := v.Validate(context.Background(), ""); err == nil {
		t.Error("empty token must be rejected")
	}
}

func TestValidate_AcceptsSecondaryClientID(t *testing.T) {
	v, signer := newTestValidator(t)
	v.AddAcceptedClientID("mobile-client")
	tok := signToken(t, signer, "login.eveonline.com", []string{"mobile-client", audienceEVE}, "CHARACTER:EVE:12345", time.Now().Add(10*time.Minute), validClaims())
	info, err := v.Validate(context.Background(), tok)
	if err != nil {
		t.Fatalf("Validate with secondary client ID: %v", err)
	}
	if info.CharacterID != 12345 {
		t.Errorf("CharacterID = %d, want 12345", info.CharacterID)
	}
}
