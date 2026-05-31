package services

// MaterialValue is one reprocessing output material with its current taker price.
type MaterialValue struct {
	Qty      int64   // material output per ONE portionSize batch (from typeMaterials)
	BuyPrice float64 // ISK/unit, highest buy order
}

// OreCompareInput holds the resolved inputs for one ore's raw-vs-refine taker comparison.
type OreCompareInput struct {
	PortionSize  int64
	OreVolumeM3  float64
	OreBuyPrice  float64 // ISK/unit, highest buy order for the raw ore
	Materials    []MaterialValue
	NetYield     float64 // reprocessing.NetYield(...) for the best station
	StationTake  float64 // ReprocessTax(...) for the best station
	SalesTaxRate float64 // from Accounting skill
}

// OreCompareResult is the per-ore verdict, normalized to net ISK per m³.
type OreCompareResult struct {
	RawNetPerM3    float64
	RefineNetPerM3 float64
	Best           string // "raw" | "refine"
	DeltaPerM3     float64
}

// CompareOre compares selling the raw ore vs. reprocessing + selling the materials,
// both as a taker (sales tax only, no broker fee), normalized to net ISK per m³.
func CompareOre(in OreCompareInput) OreCompareResult {
	salesMul := 1 - in.SalesTaxRate
	rawPerM3 := in.OreBuyPrice * salesMul / in.OreVolumeM3
	var batchGross float64
	for _, m := range in.Materials {
		batchGross += float64(m.Qty) * in.NetYield * m.BuyPrice
	}
	batchNet := batchGross * (1 - in.StationTake) * salesMul
	refinePerUnit := batchNet / float64(in.PortionSize)
	refinePerM3 := refinePerUnit / in.OreVolumeM3
	r := OreCompareResult{RawNetPerM3: rawPerM3, RefineNetPerM3: refinePerM3}
	if refinePerM3 >= rawPerM3 {
		r.Best = "refine"
	} else {
		r.Best = "raw"
	}
	r.DeltaPerM3 = absFloat(refinePerM3 - rawPerM3)
	return r
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
