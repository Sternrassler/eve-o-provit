//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompetitionRepository_Integration exercises the real schema (migration 000002)
// through the full register → snapshot → metric → prune lifecycle.
func TestCompetitionRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupPostgresContainer(t)
	tc.SetupSchema(t)

	repo := NewCompetitionRepository(tc.Pool)
	ctx := context.Background()
	const typeID, regionID = 34, 10000002

	// Register tracking (idempotent upsert).
	require.NoError(t, repo.RegisterTracked(ctx, typeID, regionID))
	require.NoError(t, repo.RegisterTracked(ctx, typeID, regionID))

	tracked, err := repo.ListTracked(ctx, time.Hour)
	require.NoError(t, err)
	require.Len(t, tracked, 1)
	assert.Equal(t, typeID, tracked[0].TypeID)
	assert.Equal(t, regionID, tracked[0].RegionID)

	// No snapshot yet.
	_, _, ok, err := repo.LatestSnapshot(ctx, typeID, regionID)
	require.NoError(t, err)
	assert.False(t, ok)

	// Save two snapshots; the latest must come back intact.
	require.NoError(t, repo.SaveSnapshot(ctx, typeID, regionID, map[int64]float64{1: 5.0, 2: 6.0}))
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, repo.SaveSnapshot(ctx, typeID, regionID, map[int64]float64{1: 5.5, 3: 7.0}))

	fp, takenAt, ok, err := repo.LatestSnapshot(ctx, typeID, regionID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.False(t, takenAt.IsZero())
	assert.Equal(t, map[int64]float64{1: 5.5, 3: 7.0}, fp)

	// No live metric yet.
	_, ok, err = repo.GetMetric(ctx, typeID, regionID)
	require.NoError(t, err)
	assert.False(t, ok)

	// Upsert a live metric and read it back.
	start := time.Now().Add(-time.Hour)
	require.NoError(t, repo.UpsertMetric(ctx, typeID, regionID, 42.5, start, time.Now(), "live"))
	m, ok, err := repo.GetMetric(ctx, typeID, regionID)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "live", m.Source)
	assert.InDelta(t, 42.5, m.ChangesPerHour, 1e-9)

	// Prune removes all snapshots older than 0.
	require.NoError(t, repo.PruneSnapshots(ctx, 0))
	_, _, ok, err = repo.LatestSnapshot(ctx, typeID, regionID)
	require.NoError(t, err)
	assert.False(t, ok)
}
