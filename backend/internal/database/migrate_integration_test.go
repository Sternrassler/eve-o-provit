//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestApplyMigrations_Idempotent verifies the embedded migrations apply cleanly on a
// fresh database AND can be re-applied without error — the property that makes
// boot-time apply safe (single schema source, no init-db, no version table).
func TestApplyMigrations_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tc := SetupPostgresContainer(t)
	ctx := context.Background()

	// Apply twice; both must succeed.
	require.NoError(t, ApplyMigrations(ctx, tc.ConnStr), "first apply")
	require.NoError(t, ApplyMigrations(ctx, tc.ConnStr), "second apply (idempotent)")

	// Core tables exist after applying.
	for _, table := range []string{"market_orders", "price_history", "competition_tracked", "competition_metric"} {
		var exists bool
		err := tc.Pool.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = $1)", table,
		).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "table %s should exist after migrations", table)
	}
}
