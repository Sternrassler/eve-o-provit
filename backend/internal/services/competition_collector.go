package services

import (
	"context"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// Collector tuning. Snapshots every CollectInterval; only pairs requested within
// TrackingWindow are tracked; snapshots older than SnapshotRetention are pruned.
const (
	CollectInterval   = 20 * time.Minute
	TrackingWindow    = 48 * time.Hour
	SnapshotRetention = 72 * time.Hour
)

// CompetitionCollector periodically snapshots the sell-order book of lazily-tracked
// (type, region) pairs and derives the live order-update-frequency metric (Issue #43).
type CompetitionCollector struct {
	repo      *database.CompetitionRepository
	esiClient *esi.Client
	logger    *logger.Logger
}

// NewCompetitionCollector creates a new collector.
func NewCompetitionCollector(repo *database.CompetitionRepository, esiClient *esi.Client, logger *logger.Logger) *CompetitionCollector {
	return &CompetitionCollector{repo: repo, esiClient: esiClient, logger: logger}
}

// Start runs the collector loop until ctx is cancelled. Intended to run in a goroutine.
func (c *CompetitionCollector) Start(ctx context.Context) {
	ticker := time.NewTicker(CollectInterval)
	defer ticker.Stop()
	c.logger.Info("competition collector started", "interval", CollectInterval.String())
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("competition collector stopped")
			return
		case <-ticker.C:
			c.tick(ctx)
		}
	}
}

// tick snapshots all tracked pairs once and updates their live metric.
func (c *CompetitionCollector) tick(ctx context.Context) {
	pairs, err := c.repo.ListTracked(ctx, TrackingWindow)
	if err != nil {
		c.logger.Warn("competition collector: list tracked failed", "error", err)
		return
	}
	for _, p := range pairs {
		c.collectPair(ctx, p.TypeID, p.RegionID)
	}
	if err := c.repo.PruneSnapshots(ctx, SnapshotRetention); err != nil {
		c.logger.Warn("competition collector: prune failed", "error", err)
	}
}

// collectPair fetches the current sell-order book, diffs it against the previous
// snapshot to derive churn, persists the live metric and stores the new snapshot.
func (c *CompetitionCollector) collectPair(ctx context.Context, typeID, regionID int) {
	orders, err := c.esiClient.FetchMarketOrdersForType(ctx, regionID, typeID)
	if err != nil {
		c.logger.Warn("competition collector: fetch orders failed", "error", err, "type", typeID, "region", regionID)
		return
	}
	curr := sellFingerprint(orders)

	if prev, takenAt, ok, err := c.repo.LatestSnapshot(ctx, typeID, regionID); err == nil && ok {
		hours := time.Since(takenAt).Hours()
		churn := ComputeChurn(prev, curr, hours)
		if err := c.repo.UpsertMetric(ctx, typeID, regionID, churn, takenAt, time.Now(), "live"); err != nil {
			c.logger.Warn("competition collector: upsert metric failed", "error", err)
		}
	}

	if err := c.repo.SaveSnapshot(ctx, typeID, regionID, curr); err != nil {
		c.logger.Warn("competition collector: save snapshot failed", "error", err)
	}
}

// sellFingerprint maps order_id -> price for active sell orders.
func sellFingerprint(orders []esi.ESIMarketOrder) map[int64]float64 {
	fp := make(map[int64]float64, len(orders))
	for _, o := range orders {
		if o.IsBuyOrder || o.VolumeRemain <= 0 {
			continue
		}
		fp[o.OrderID] = o.Price
	}
	return fp
}
