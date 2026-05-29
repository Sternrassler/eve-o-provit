package services

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
)

func TestTakerUnitNet(t *testing.T) {
	// 100 ISK buy order, 2.5% sales tax -> 97.5 net, no broker fee.
	got := takerUnitNet(100.0, 0.025)
	if got != 97.5 {
		t.Fatalf("takerUnitNet = %v, want 97.5", got)
	}
}

func TestBestBuyInRegion(t *testing.T) {
	orders := []esi.ESIMarketOrder{
		{IsBuyOrder: true, Price: 90, VolumeRemain: 10, LocationID: 1},
		{IsBuyOrder: true, Price: 110, VolumeRemain: 5, LocationID: 2},
		{IsBuyOrder: false, Price: 200, VolumeRemain: 5, LocationID: 2}, // sell order ignored
		{IsBuyOrder: true, Price: 105, VolumeRemain: 0, LocationID: 3},  // empty ignored
	}
	price, stn, ok := bestBuyInRegion(orders)
	if !ok || price != 110 || stn != 2 {
		t.Fatalf("bestBuyInRegion = (%v,%v,%v), want (110,2,true)", price, stn, ok)
	}
}

func TestBestBuyInRegion_NoBuyOrders(t *testing.T) {
	orders := []esi.ESIMarketOrder{{IsBuyOrder: false, Price: 200, VolumeRemain: 5, LocationID: 1}}
	if _, _, ok := bestBuyInRegion(orders); ok {
		t.Fatalf("expected ok=false when no buy orders")
	}
}

func TestBestBuyByStation(t *testing.T) {
	orders := []esi.ESIMarketOrder{
		{IsBuyOrder: true, Price: 90, VolumeRemain: 10, LocationID: 1},
		{IsBuyOrder: true, Price: 95, VolumeRemain: 10, LocationID: 1}, // higher at stn 1
		{IsBuyOrder: true, Price: 80, VolumeRemain: 10, LocationID: 2},
		{IsBuyOrder: false, Price: 200, VolumeRemain: 5, LocationID: 1}, // sell ignored
	}
	got := bestBuyByStation(orders)
	if got[1] != 95 || got[2] != 80 || len(got) != 2 {
		t.Fatalf("bestBuyByStation = %v, want {1:95, 2:80}", got)
	}
}

func TestAggregateAssets(t *testing.T) {
	raw := []RawAsset{
		{TypeID: 34, LocationID: 60003760, Quantity: 100},
		{TypeID: 34, LocationID: 60003760, Quantity: 50}, // same type+loc -> summed
		{TypeID: 34, LocationID: 60008494, Quantity: 10}, // different loc -> separate
		{TypeID: 35, LocationID: 60003760, Quantity: 7},
	}
	got := aggregateAssets(raw)
	find := func(typeID int, loc int64) int {
		for _, a := range got {
			if a.typeID == typeID && a.locationID == loc {
				return a.quantity
			}
		}
		return -1
	}
	if find(34, 60003760) != 150 {
		t.Fatalf("type 34 @ 60003760 = %d, want 150", find(34, 60003760))
	}
	if find(34, 60008494) != 10 || find(35, 60003760) != 7 {
		t.Fatalf("unexpected aggregation: %+v", got)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 aggregated stacks, got %d", len(got))
	}
}
