// Package database - Competition tracking repository (Issue #43)
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// CompetitionRepository persists the lazy-tracking, order snapshots and derived
// competition metrics that power the Multi-Hub Comparison "order update frequency"
// indicator.
type CompetitionRepository struct {
	db DBPool
}

// NewCompetitionRepository creates a new competition repository.
func NewCompetitionRepository(db DBPool) *CompetitionRepository {
	return &CompetitionRepository{db: db}
}

// TrackedPair is a (type, region) pair the collector should snapshot.
type TrackedPair struct {
	TypeID   int
	RegionID int
}

// CompetitionMetric is the derived per-(type,region) competition score.
type CompetitionMetric struct {
	ChangesPerHour float64
	Source         string // "live" | "baseline"
	UpdatedAt      time.Time
}

// RegisterTracked upserts a (type, region) pair, refreshing last_requested.
func (r *CompetitionRepository) RegisterTracked(ctx context.Context, typeID, regionID int) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO competition_tracked (type_id, region_id, last_requested)
		VALUES ($1, $2, NOW())
		ON CONFLICT (type_id, region_id) DO UPDATE SET last_requested = NOW()
	`, typeID, regionID)
	if err != nil {
		return fmt.Errorf("register tracked: %w", err)
	}
	return nil
}

// ListTracked returns pairs requested within the given retention window.
func (r *CompetitionRepository) ListTracked(ctx context.Context, within time.Duration) ([]TrackedPair, error) {
	rows, err := r.db.Query(ctx, `
		SELECT type_id, region_id FROM competition_tracked
		WHERE last_requested > NOW() - $1::interval
	`, fmt.Sprintf("%d seconds", int(within.Seconds())))
	if err != nil {
		return nil, fmt.Errorf("list tracked: %w", err)
	}
	defer rows.Close()
	var out []TrackedPair
	for rows.Next() {
		var p TrackedPair
		if err := rows.Scan(&p.TypeID, &p.RegionID); err != nil {
			return nil, fmt.Errorf("scan tracked: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SaveSnapshot stores an order fingerprint (order_id -> price of active sell orders).
func (r *CompetitionRepository) SaveSnapshot(ctx context.Context, typeID, regionID int, fingerprint map[int64]float64) error {
	data, err := json.Marshal(fingerprint)
	if err != nil {
		return fmt.Errorf("marshal fingerprint: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO competition_snapshot (type_id, region_id, taken_at, fingerprint)
		VALUES ($1, $2, NOW(), $3)
	`, typeID, regionID, string(data))
	if err != nil {
		return fmt.Errorf("save snapshot: %w", err)
	}
	return nil
}

// LatestSnapshot returns the most recent snapshot fingerprint + its time, or ok=false.
func (r *CompetitionRepository) LatestSnapshot(ctx context.Context, typeID, regionID int) (map[int64]float64, time.Time, bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT taken_at, fingerprint FROM competition_snapshot
		WHERE type_id = $1 AND region_id = $2
		ORDER BY taken_at DESC LIMIT 1
	`, typeID, regionID)
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("latest snapshot: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, time.Time{}, false, rows.Err()
	}
	var takenAt time.Time
	var raw []byte
	if err := rows.Scan(&takenAt, &raw); err != nil {
		return nil, time.Time{}, false, fmt.Errorf("scan snapshot: %w", err)
	}
	fp := map[int64]float64{}
	if err := json.Unmarshal(raw, &fp); err != nil {
		return nil, time.Time{}, false, fmt.Errorf("unmarshal fingerprint: %w", err)
	}
	return fp, takenAt, true, nil
}

// UpsertMetric stores the derived competition metric.
func (r *CompetitionRepository) UpsertMetric(ctx context.Context, typeID, regionID int, changesPerHour float64, windowStart, windowEnd time.Time, source string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO competition_metric (type_id, region_id, changes_per_hour, window_start, window_end, source, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (type_id, region_id) DO UPDATE SET
			changes_per_hour = EXCLUDED.changes_per_hour,
			window_start = EXCLUDED.window_start,
			window_end = EXCLUDED.window_end,
			source = EXCLUDED.source,
			updated_at = NOW()
	`, typeID, regionID, changesPerHour, windowStart, windowEnd, source)
	if err != nil {
		return fmt.Errorf("upsert metric: %w", err)
	}
	return nil
}

// GetMetric returns the live metric for (type, region), or ok=false if none exists.
func (r *CompetitionRepository) GetMetric(ctx context.Context, typeID, regionID int) (CompetitionMetric, bool, error) {
	rows, err := r.db.Query(ctx, `
		SELECT changes_per_hour, source, updated_at FROM competition_metric
		WHERE type_id = $1 AND region_id = $2
	`, typeID, regionID)
	if err != nil {
		return CompetitionMetric{}, false, fmt.Errorf("get metric: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return CompetitionMetric{}, false, rows.Err()
	}
	var m CompetitionMetric
	if err := rows.Scan(&m.ChangesPerHour, &m.Source, &m.UpdatedAt); err != nil {
		return CompetitionMetric{}, false, fmt.Errorf("scan metric: %w", err)
	}
	return m, true, nil
}

// PruneSnapshots deletes snapshots older than the cutoff.
func (r *CompetitionRepository) PruneSnapshots(ctx context.Context, olderThan time.Duration) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM competition_snapshot WHERE taken_at < NOW() - $1::interval
	`, fmt.Sprintf("%d seconds", int(olderThan.Seconds())))
	if err != nil {
		return fmt.Errorf("prune snapshots: %w", err)
	}
	return nil
}
