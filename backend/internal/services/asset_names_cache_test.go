package services

import "testing"

func TestAssetNamesCacheKey_OrderStableAndDistinct(t *testing.T) {
	a := assetNamesCacheKey(42, []int64{111, 222})
	b := assetNamesCacheKey(42, []int64{222, 111})
	if a != b {
		t.Errorf("order should not change key: %s vs %s", a, b)
	}
	c := assetNamesCacheKey(42, []int64{111, 333})
	if a == c {
		t.Errorf("different id sets must differ: %s", a)
	}
	d := assetNamesCacheKey(43, []int64{111, 222})
	if a == d {
		t.Errorf("different character must differ")
	}
}
