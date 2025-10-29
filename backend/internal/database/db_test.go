package database

import (
	"context"
	"testing"
)

func TestNewDB_InvalidPostgresURL(t *testing.T) {
	ctx := context.Background()

	cfg := Config{
		PostgresURL: "invalid://url",
		SDEPath:     "test.db",
	}

	_, err := New(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid PostgreSQL URL")
	}
}

func TestNewDB_InvalidSDEPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	ctx := context.Background()

	cfg := Config{
		PostgresURL: "postgresql://user:pass@localhost:5432/testdb?sslmode=disable",
		SDEPath:     "/nonexistent/path/to/sde.db",
	}

	_, err := New(ctx, cfg)
	if err == nil {
		t.Error("Expected error for invalid SDE path")
	}
}
