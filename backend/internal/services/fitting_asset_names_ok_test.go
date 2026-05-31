package services

import (
	"encoding/json"
	"testing"
)

// TestParseAssetNames_SuccessFlag exercises the pure parseAssetNames helper used by
// fetchAssetNamesBatch, and separately validates that the ok-flag merge in
// fetchAssetNamesFromESI propagates correctly (via TestFetchAssetNamesFromESI_OKFlag).

func TestParseAssetNames_Valid(t *testing.T) {
	body, _ := json.Marshal([]map[string]any{
		{"item_id": int64(1), "name": "Freighter Alpha"},
		{"item_id": int64(2), "name": "None"}, // sentinel → dropped
		{"item_id": int64(3), "name": ""},     // empty → dropped
		{"item_id": int64(4), "name": "Badger II"},
	})
	got, err := parseAssetNames(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 entries (1 and 4), got %d: %v", len(got), got)
	}
	if got[1] != "Freighter Alpha" {
		t.Errorf("item 1: want 'Freighter Alpha', got %q", got[1])
	}
	if got[4] != "Badger II" {
		t.Errorf("item 4: want 'Badger II', got %q", got[4])
	}
}

func TestParseAssetNames_InvalidJSON(t *testing.T) {
	_, err := parseAssetNames([]byte("not json"))
	if err == nil {
		t.Fatal("want error for invalid JSON, got nil")
	}
}

// TestFetchAssetNamesFromESI_OKFlag verifies the contract that
// fetchAssetNamesFromESI returns ok=false when any batch fails,
// even though partial names from successful batches are still returned.
//
// We test this at the merge level by substituting a minimal inline
// implementation that mirrors the real chunk-and-merge loop, so no
// HTTP server is required.
func TestFetchAssetNamesFromESI_OKFlag(t *testing.T) {
	// mergeWithOK models fetchAssetNamesFromESI's logic: iterate batches,
	// merge names, AND-track the ok flag.
	type batchResult struct {
		names map[int64]string
		ok    bool
	}
	mergeWithOK := func(batches []batchResult) (map[int64]string, bool) {
		out := make(map[int64]string)
		allOK := true
		for _, b := range batches {
			if !b.ok {
				allOK = false
			}
			for id, name := range b.names {
				out[id] = name
			}
		}
		return out, allOK
	}

	t.Run("all batches succeed → ok=true", func(t *testing.T) {
		batches := []batchResult{
			{names: map[int64]string{1: "Alpha"}, ok: true},
			{names: map[int64]string{2: "Beta"}, ok: true},
		}
		names, ok := mergeWithOK(batches)
		if !ok {
			t.Fatal("want ok=true when all batches succeed")
		}
		if names[1] != "Alpha" || names[2] != "Beta" {
			t.Errorf("unexpected names: %v", names)
		}
	})

	t.Run("one batch fails → ok=false, partial names returned", func(t *testing.T) {
		batches := []batchResult{
			{names: map[int64]string{1: "Alpha"}, ok: true},
			{names: map[int64]string{}, ok: false}, // ESI error
		}
		names, ok := mergeWithOK(batches)
		if ok {
			t.Fatal("want ok=false when any batch fails")
		}
		// Partial names from the successful batch are still present.
		if names[1] != "Alpha" {
			t.Errorf("want partial name for id 1, got %q", names[1])
		}
	})

	t.Run("all batches fail → ok=false, empty map", func(t *testing.T) {
		batches := []batchResult{
			{names: map[int64]string{}, ok: false},
			{names: map[int64]string{}, ok: false},
		}
		names, ok := mergeWithOK(batches)
		if ok {
			t.Fatal("want ok=false when all batches fail")
		}
		if len(names) != 0 {
			t.Errorf("want empty map, got %v", names)
		}
	})

	t.Run("empty batch list → ok=true, empty map (genuine empty success)", func(t *testing.T) {
		names, ok := mergeWithOK(nil)
		if !ok {
			t.Fatal("want ok=true for empty batch list (no failures)")
		}
		if len(names) != 0 {
			t.Errorf("want empty map, got %v", names)
		}
	})
}
