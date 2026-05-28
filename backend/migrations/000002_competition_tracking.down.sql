-- Rollback: Competition tracking (Issue #43)
DROP TABLE IF EXISTS competition_metric;
DROP TABLE IF EXISTS competition_snapshot;
DROP TABLE IF EXISTS competition_tracked;
