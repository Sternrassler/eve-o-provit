package services

import "testing"

// TestParseStandings verifies the pure standings parser against an ESI-shaped fixture.
func TestParseStandings(t *testing.T) {
	fixture := []byte(`[
		{"from_type":"npc_corp","from_id":1000035,"standing":7.5},
		{"from_type":"faction","from_id":500003,"standing":2.25},
		{"from_type":"agent","from_id":3008416,"standing":-1.0}
	]`)

	got, err := parseStandings(fixture)
	if err != nil {
		t.Fatalf("parseStandings returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[1000035] != 7.5 {
		t.Errorf("npc_corp standing: got %v, want 7.5", got[1000035])
	}
	if got[500003] != 2.25 {
		t.Errorf("faction standing: got %v, want 2.25", got[500003])
	}
	if got[3008416] != -1.0 {
		t.Errorf("agent standing: got %v, want -1.0", got[3008416])
	}
}

// TestParseStandings_Empty verifies an empty array yields an empty (non-nil) map.
func TestParseStandings_Empty(t *testing.T) {
	got, err := parseStandings([]byte(`[]`))
	if err != nil {
		t.Fatalf("parseStandings returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil map")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %d entries", len(got))
	}
}

// TestParseStandings_Invalid verifies malformed JSON returns an error.
func TestParseStandings_Invalid(t *testing.T) {
	if _, err := parseStandings([]byte(`{not json`)); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
