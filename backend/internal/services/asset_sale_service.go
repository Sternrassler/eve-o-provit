package services

import (
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
)

// takerUnitNet is the per-unit proceeds when selling instantly into a buy order:
// buy price minus sales tax. No broker fee (that applies only to placed orders).
func takerUnitNet(buyPrice, salesTaxRate float64) float64 {
	return buyPrice * (1 - salesTaxRate)
}

// bestBuyInRegion returns the highest active buy-order price across all stations in a
// region, plus the station it sits at.
func bestBuyInRegion(orders []esi.ESIMarketOrder) (price float64, stationID int64, ok bool) {
	for _, o := range orders {
		if !o.IsBuyOrder || o.VolumeRemain <= 0 {
			continue
		}
		if !ok || o.Price > price {
			price, stationID, ok = o.Price, o.LocationID, true
		}
	}
	return price, stationID, ok
}

// bestBuyByStation returns the highest active buy-order price per station.
func bestBuyByStation(orders []esi.ESIMarketOrder) map[int64]float64 {
	best := map[int64]float64{}
	for _, o := range orders {
		if !o.IsBuyOrder || o.VolumeRemain <= 0 {
			continue
		}
		if cur, seen := best[o.LocationID]; !seen || o.Price > cur {
			best[o.LocationID] = o.Price
		}
	}
	return best
}

// aggStack is an aggregated (type, location) quantity, pre-enrichment.
type aggStack struct {
	typeID     int
	locationID int64
	quantity   int
}

// aggregateAssets sums quantities for identical (type, location) pairs.
func aggregateAssets(raw []RawAsset) []aggStack {
	idx := map[[2]int64]int{} // (typeID, locationID) -> index in out
	var out []aggStack
	for _, r := range raw {
		key := [2]int64{int64(r.TypeID), r.LocationID}
		if i, ok := idx[key]; ok {
			out[i].quantity += r.Quantity
			continue
		}
		idx[key] = len(out)
		out = append(out, aggStack{typeID: r.TypeID, locationID: r.LocationID, quantity: r.Quantity})
	}
	return out
}
