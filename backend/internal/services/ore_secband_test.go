package services

import "testing"

func TestSecBandOreGroups(t *testing.T) {
	high := secBandOreGroups("high")
	if !high[462] || !high[460] { // Veldspar, Scordite
		t.Fatal("highsec must include Veldspar (462) + Scordite (460)")
	}
	if high[450] { // Arkonor
		t.Fatal("highsec must NOT include Arkonor (450)")
	}

	null := secBandOreGroups("null")
	if !null[450] {
		t.Fatal("nullsec must include Arkonor (450)")
	}
	if !null[462] || !null[452] { // cumulative: highsec Veldspar + lowsec Crokite
		t.Fatal("nullsec must include highsec + lowsec ores")
	}

	if len(secBandOreGroups("bogus")) != 0 {
		t.Fatal("unknown band must be empty")
	}
}
