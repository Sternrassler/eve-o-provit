package services

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	_ "github.com/mattn/go-sqlite3"
)

// newSecTestSDE builds an in-memory SDE with a single known system. Any other
// system ID will produce a "not found" lookup error from GetSystemSecurityStatus.
func newSecTestSDE(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		CREATE TABLE mapSolarSystems (_key INTEGER PRIMARY KEY, securityStatus REAL NOT NULL);
		INSERT INTO mapSolarSystems VALUES (30000142, 0.9);  -- Jita (high-sec)
		INSERT INTO mapSolarSystems VALUES (30000049, -0.4); -- a low/null-sec system
	`); err != nil {
		t.Fatal(err)
	}
	return db
}

// TestGetSystemSecurityStatus_LookupErrorDoesNotFakeHighSec is the issue #147 A4
// regression: a failed security-status lookup MUST return an error, NEVER a
// fabricated 1.0 (high-sec). A 1.0 fallback would let a null-/low-sec route pass a
// high-sec security filter — the exact failure mode of the old COALESCE bug.
func TestGetSystemSecurityStatus_LookupErrorDoesNotFakeHighSec(t *testing.T) {
	db := newSecTestSDE(t)
	defer db.Close()
	ro := NewRouteCalculator(database.NewSDERepository(db), db, nil)
	ctx := context.Background()

	// Unknown system → lookup fails.
	sec, err := ro.getSystemSecurityStatus(ctx, 99999999)
	if err == nil {
		t.Fatalf("expected error for unknown system, got sec=%v err=nil (fail-loud violated)", sec)
	}
	if sec == 1.0 {
		t.Fatalf("lookup failure must not return a fabricated high-sec 1.0")
	}
}

// TestGetSystemSecurityStatus_RealValuesPassThrough verifies the happy path still
// returns the true security status for known systems.
func TestGetSystemSecurityStatus_RealValuesPassThrough(t *testing.T) {
	db := newSecTestSDE(t)
	defer db.Close()
	ro := NewRouteCalculator(database.NewSDERepository(db), db, nil)
	ctx := context.Background()

	hi, err := ro.getSystemSecurityStatus(ctx, 30000142)
	if err != nil || hi != 0.9 {
		t.Fatalf("Jita: got (%v,%v), want (0.9,nil)", hi, err)
	}

	lo, err := ro.getSystemSecurityStatus(ctx, 30000049)
	if err != nil || lo != -0.4 {
		t.Fatalf("low-sec: got (%v,%v), want (-0.4,nil)", lo, err)
	}
}
