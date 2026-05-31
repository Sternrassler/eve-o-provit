package services

import "testing"

func TestParseAssetNames_MapsAndFallsBack(t *testing.T) {
	raw := []byte(`[{"item_id":111,"name":"Ix-ITE-1"},{"item_id":222,"name":"None"}]`)
	names, err := parseAssetNames(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if names[111] != "Ix-ITE-1" {
		t.Errorf("want Ix-ITE-1, got %q", names[111])
	}
	if _, ok := names[222]; ok {
		t.Errorf(`"None" should be dropped so the caller falls back to the type name`)
	}
}
