package esiconfig

import "testing"

func TestBaseURL(t *testing.T) {
	if BaseURL != "https://esi.evetech.net" {
		t.Errorf("BaseURL = %q", BaseURL)
	}
}
