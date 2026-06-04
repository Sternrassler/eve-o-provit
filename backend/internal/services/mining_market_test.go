package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
)

type fakeMarketFetcher struct {
	orders []esi.ESIMarketOrder
	err    error
	calls  int
}

func (f *fakeMarketFetcher) FetchMarketOrdersForType(_ context.Context, _, _ int) ([]esi.ESIMarketOrder, error) {
	f.calls++
	return f.orders, f.err
}

func TestLiveMarketBuyProvider_MapsLiveOrders(t *testing.T) {
	f := &fakeMarketFetcher{orders: []esi.ESIMarketOrder{
		{OrderID: 1, TypeID: 1228, LocationID: 60003760, IsBuyOrder: true, Price: 15.76, VolumeTotal: 1_000_000, VolumeRemain: 985_407},
	}}
	p := NewLiveMarketBuyProvider(f)

	got, err := p.GetMarketOrders(context.Background(), 10000068, 1228)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	o := got[0]
	if !o.IsBuyOrder || o.Price != 15.76 || o.LocationID != 60003760 || o.VolumeRemain != 985_407 || o.RegionID != 10000068 {
		t.Errorf("bad mapping: %+v", o)
	}
}

// The fix's core invariant: an ESI failure is propagated (fail-loud), never
// swallowed into a fabricated/stale price.
func TestLiveMarketBuyProvider_PropagatesError(t *testing.T) {
	f := &fakeMarketFetcher{err: errors.New("esi unavailable")}
	p := NewLiveMarketBuyProvider(f)

	if _, err := p.GetMarketOrders(context.Background(), 1, 1); err == nil {
		t.Fatal("expected the ESI error to propagate, got nil")
	}
}
