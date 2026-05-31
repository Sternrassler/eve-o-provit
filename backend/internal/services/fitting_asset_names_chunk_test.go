package services

import (
	"reflect"
	"testing"
)

// TestChunkInt64 verifies FetchAssetNames' batching helper respects ESI's 1000-id
// limit: ≤size stays a single batch, >size splits into consecutive ≤size batches
// with no ids lost or duplicated.
func TestChunkInt64(t *testing.T) {
	t.Run("at or below size is a single chunk", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		got := chunkInt64(ids, assetNamesMaxIDs)
		if len(got) != 1 || !reflect.DeepEqual(got[0], ids) {
			t.Fatalf("want single chunk %v, got %v", ids, got)
		}
	})

	t.Run("exactly size is a single chunk", func(t *testing.T) {
		ids := make([]int64, assetNamesMaxIDs)
		for i := range ids {
			ids[i] = int64(i)
		}
		got := chunkInt64(ids, assetNamesMaxIDs)
		if len(got) != 1 {
			t.Fatalf("1000 ids must be one batch, got %d batches", len(got))
		}
	})

	t.Run("above size splits into <=size batches", func(t *testing.T) {
		ids := make([]int64, 2300)
		for i := range ids {
			ids[i] = int64(i)
		}
		got := chunkInt64(ids, assetNamesMaxIDs)
		if len(got) != 3 {
			t.Fatalf("2300 ids → want 3 batches, got %d", len(got))
		}
		// No batch exceeds the limit; all ids preserved in order.
		var flat []int64
		for _, b := range got {
			if len(b) > assetNamesMaxIDs {
				t.Fatalf("batch of %d exceeds limit %d", len(b), assetNamesMaxIDs)
			}
			flat = append(flat, b...)
		}
		if !reflect.DeepEqual(flat, ids) {
			t.Fatalf("flattened batches must equal input ids")
		}
	})

	t.Run("size <= 0 yields one chunk", func(t *testing.T) {
		ids := []int64{1, 2, 3}
		got := chunkInt64(ids, 0)
		if len(got) != 1 || !reflect.DeepEqual(got[0], ids) {
			t.Fatalf("size<=0 must be single chunk, got %v", got)
		}
	})
}
