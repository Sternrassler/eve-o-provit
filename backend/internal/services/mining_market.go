package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
)

// miningMarketFetcher is the live ESI market source (subset of *esi.Client).
type miningMarketFetcher interface {
	FetchMarketOrdersForType(ctx context.Context, regionID, typeID int) ([]esi.ESIMarketOrder, error)
}

// LiveMarketBuyProvider implements MarketBuyProvider by fetching LIVE order books
// from ESI (ETag/Redis-validated at the ESI layer) instead of reading the
// market_orders DB snapshot.
//
// Why: the DB snapshot is only refreshed on-demand by trading-route queries
// (route_finder upsert) — there is no background refresher — so for a region no
// one has recently run a trading query against, it can be days stale. Mining then
// ranks ores against stale prices and recommends the wrong (stale-best) sell
// location. All other market consumers (trading, multi-hub, sell-assets,
// competition) already fetch live; mining was the lone DB reader.
type LiveMarketBuyProvider struct {
	esi miningMarketFetcher
}

// NewLiveMarketBuyProvider returns a MarketBuyProvider backed by live ESI fetches.
func NewLiveMarketBuyProvider(f miningMarketFetcher) LiveMarketBuyProvider {
	return LiveMarketBuyProvider{esi: f}
}

// GetMarketOrders fetches the live order book for (region, type) and maps it to
// the database.MarketOrder shape the mining ranking consumes. An ESI error is
// propagated (fail-loud) — the caller skips the ore rather than fabricating a
// price from stale data.
func (p LiveMarketBuyProvider) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
	eos, err := p.esi.FetchMarketOrdersForType(ctx, regionID, typeID)
	if err != nil {
		return nil, err
	}
	out := make([]database.MarketOrder, len(eos))
	for i, o := range eos {
		out[i] = database.MarketOrder{
			OrderID:      o.OrderID,
			TypeID:       o.TypeID,
			RegionID:     regionID,
			LocationID:   o.LocationID,
			IsBuyOrder:   o.IsBuyOrder,
			Price:        o.Price,
			VolumeTotal:  o.VolumeTotal,
			VolumeRemain: o.VolumeRemain,
			Issued:       o.Issued,
			Duration:     o.Duration,
		}
	}
	return out, nil
}
