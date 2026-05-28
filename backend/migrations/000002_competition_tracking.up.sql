-- Migration: Competition tracking for Multi-Hub Comparison (Issue #43)
-- PostgreSQL schema. Powers the "order update frequency" competition indicator
-- (lazy-tracked per (type_id, region_id); live churn from periodic snapshots,
-- with a daily baseline from price_history.order_count as fallback).

-- Lazily registered (type, hub-region) pairs the collector should track.
CREATE TABLE IF NOT EXISTS competition_tracked (
    type_id        INTEGER NOT NULL,
    region_id      INTEGER NOT NULL,
    last_requested TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (type_id, region_id)
);

CREATE INDEX IF NOT EXISTS idx_competition_tracked_last_requested
    ON competition_tracked(last_requested);

COMMENT ON TABLE competition_tracked IS 'Lazily registered (type,region) pairs the competition collector snapshots periodically';

-- Rolling order snapshots; the collector diffs consecutive snapshots to derive churn.
-- fingerprint is a compact map order_id -> price of the active sell orders at snapshot time.
CREATE TABLE IF NOT EXISTS competition_snapshot (
    id          BIGSERIAL PRIMARY KEY,
    type_id     INTEGER NOT NULL,
    region_id   INTEGER NOT NULL,
    taken_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    fingerprint JSONB NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_competition_snapshot_lookup
    ON competition_snapshot(type_id, region_id, taken_at DESC);

COMMENT ON TABLE competition_snapshot IS 'Periodic order fingerprints; consecutive diffs yield order-change events for the churn metric';

-- Derived competition metric per (type, region). source = 'live' (from snapshots)
-- or 'baseline' (from price_history.order_count) until enough live data exists.
CREATE TABLE IF NOT EXISTS competition_metric (
    type_id          INTEGER NOT NULL,
    region_id        INTEGER NOT NULL,
    changes_per_hour DOUBLE PRECISION NOT NULL DEFAULT 0,
    window_start     TIMESTAMPTZ,
    window_end       TIMESTAMPTZ,
    source           VARCHAR(16) NOT NULL DEFAULT 'baseline',
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (type_id, region_id)
);

COMMENT ON TABLE competition_metric IS 'Derived competition score per (type,region); source live=snapshot churn, baseline=daily order_count';
