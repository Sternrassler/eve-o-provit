package services

// matchOrder is the minimal order shape the matcher needs.
type matchOrder struct {
	TypeID       int
	StationID    int64
	IsBuyOrder   bool
	Price        float64
	VolumeRemain int
}

// RouteCandidate is one (buy station → sell station) leg with its profitable items.
type RouteCandidate struct {
	BuyStationID  int64
	SellStationID int64
	Items         []HaulItem
}

// MatchHauls finds, per item type, the cheapest sell order (buy source) and highest
// buy order (sell target) across all stations, keeps profitable cross-station hauls
// (net of salesTaxRate on the sell), and groups them by (buy station → sell station).
func MatchHauls(orders []matchOrder, salesTaxRate float64) []RouteCandidate {
	type best struct {
		sellPrice float64
		sellQty   int
		sellStn   int64
		hasSell   bool
		buyPrice  float64
		buyQty    int
		buyStn    int64
		hasBuy    bool
	}
	byType := map[int]*best{}
	for _, o := range orders {
		if o.VolumeRemain <= 0 || o.Price <= 0 {
			continue
		}
		b := byType[o.TypeID]
		if b == nil {
			b = &best{}
			byType[o.TypeID] = b
		}
		if o.IsBuyOrder { // someone buys here → a SELL target for us
			if !b.hasBuy || o.Price > b.buyPrice {
				b.buyPrice, b.buyQty, b.buyStn, b.hasBuy = o.Price, o.VolumeRemain, o.StationID, true
			}
		} else { // someone sells here → a BUY source for us
			if !b.hasSell || o.Price < b.sellPrice {
				b.sellPrice, b.sellQty, b.sellStn, b.hasSell = o.Price, o.VolumeRemain, o.StationID, true
			}
		}
	}

	grouped := map[[2]int64]*RouteCandidate{}
	for typeID, b := range byType {
		if !b.hasSell || !b.hasBuy || b.sellStn == b.buyStn {
			continue
		}
		net := b.buyPrice*(1-salesTaxRate) - b.sellPrice
		if net <= 0 {
			continue
		}
		qty := b.sellQty
		if b.buyQty < qty {
			qty = b.buyQty
		}
		key := [2]int64{b.sellStn, b.buyStn} // buy at sellStn (cheapest sell order), sell at buyStn
		rc := grouped[key]
		if rc == nil {
			rc = &RouteCandidate{BuyStationID: b.sellStn, SellStationID: b.buyStn}
			grouped[key] = rc
		}
		rc.Items = append(rc.Items, HaulItem{
			TypeID:        typeID,
			BuyPrice:      b.sellPrice,
			ProfitPerUnit: net,
			AvailableQty:  qty,
			// Name + UnitVolume filled by the service from SDE (matcher is price-only).
		})
	}

	out := make([]RouteCandidate, 0, len(grouped))
	for _, rc := range grouped {
		out = append(out, *rc)
	}
	return out
}
