package services

import "testing"

func mo(typeID int, station int64, buy bool, price float64, vol int) matchOrder {
	return matchOrder{TypeID: typeID, StationID: station, IsBuyOrder: buy, Price: price, VolumeRemain: vol}
}

func TestMatchHauls_GroupsByRoute(t *testing.T) {
	orders := []matchOrder{
		mo(1, 100, false, 5.0, 50), mo(1, 100, false, 6.0, 10),
		mo(1, 200, true, 8.0, 40),
		mo(2, 100, false, 3.0, 100), mo(2, 200, true, 4.0, 100),
		mo(3, 300, false, 10, 5), mo(3, 300, true, 12, 5), // same station → no haul
	}
	routes := MatchHauls(orders, 0.0)
	if len(routes) != 1 {
		t.Fatalf("expected 1 route (100→200), got %d", len(routes))
	}
	r := routes[0]
	if r.BuyStationID != 100 || r.SellStationID != 200 {
		t.Errorf("route stations wrong: %d→%d", r.BuyStationID, r.SellStationID)
	}
	if len(r.Items) != 2 {
		t.Errorf("expected 2 items on route, got %d", len(r.Items))
	}
	var i1 *HaulItem
	for k := range r.Items {
		if r.Items[k].TypeID == 1 {
			i1 = &r.Items[k]
		}
	}
	if i1 == nil || i1.ProfitPerUnit != 3.0 || i1.AvailableQty != 40 || i1.BuyPrice != 5.0 {
		t.Errorf("type1 economics wrong: %+v", i1)
	}
}

func TestMatchHauls_SalesTaxRemovesThinMargin(t *testing.T) {
	orders := []matchOrder{
		mo(1, 100, false, 100, 10), mo(1, 200, true, 101, 10),
	}
	routes := MatchHauls(orders, 0.05)
	if len(routes) != 0 {
		t.Errorf("expected no profitable route after tax, got %d", len(routes))
	}
}
