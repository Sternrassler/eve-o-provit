package services

import "testing"

func TestCompareOre_RefineWins(t *testing.T) {
	in := OreCompareInput{
		PortionSize: 100, OreVolumeM3: 0.1, OreBuyPrice: 8.0,
		Materials:    []MaterialValue{{Qty: 400, BuyPrice: 5.0}},
		NetYield:     0.90,
		StationTake:  0.02,
		SalesTaxRate: 0.04,
	}
	r := CompareOre(in)
	// raw: 8 × 0.96 / 0.1 = 76.8 per m³
	if r.RawNetPerM3 < 76.7 || r.RawNetPerM3 > 76.9 {
		t.Fatalf("raw/m3 %.4f", r.RawNetPerM3)
	}
	// refine per 100 ore: 400×0.90×5 = 1800; ×0.98 ×0.96 = 1693.44; /100 /0.1 = 169.344 per m³
	if r.RefineNetPerM3 < 169.3 || r.RefineNetPerM3 > 169.4 {
		t.Fatalf("refine/m3 %.4f", r.RefineNetPerM3)
	}
	if r.Best != "refine" {
		t.Fatalf("best %s", r.Best)
	}
}

func TestCompareOre_RawWins(t *testing.T) {
	in := OreCompareInput{
		PortionSize: 100, OreVolumeM3: 0.1, OreBuyPrice: 100,
		Materials: []MaterialValue{{Qty: 400, BuyPrice: 1}},
		NetYield:  0.5, StationTake: 0.05, SalesTaxRate: 0.04,
	}
	if CompareOre(in).Best != "raw" {
		t.Fatal("raw should win")
	}
}
