package services

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	applogger "github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// recordingFitting records whether GetShipFitting was called and returns sentinel
// per-type cargo + speed so we can prove which path produced each value.
type recordingFitting struct {
	called          bool
	sentinelCargoM3 float64
	sentinelWarpAUS float64
	sentinelAlign   float64
}

func (f *recordingFitting) GetShipFitting(_ context.Context, _, _ int, _ string) (*FittingData, error) {
	f.called = true
	return &FittingData{Bonuses: FittingBonuses{
		EffectiveCargo: f.sentinelCargoM3,
		WarpSpeedAUS:   f.sentinelWarpAUS,
		AlignTime:      f.sentinelAlign,
	}}, nil
}
func (f *recordingFitting) InvalidateFittingCache(_ context.Context, _, _ int) {}
func (f *recordingFitting) EnrichShipsEffectiveCargo(_ context.Context, _ int, _ []models.CharacterAssetShip, _ string) {
}
func (f *recordingFitting) EffectiveCargoForActiveShip(_ context.Context, _, _ int, _ int64, _ string) (float64, bool) {
	return 0, false
}
func (f *recordingFitting) ActiveShipFittedModuleTypeIDs(_ context.Context, _ int, _ string) ([]int64, error) {
	return nil, nil
}

// When CargoCapacity > 0 the override replaces ONLY the cargo m³; the fitting is
// still fetched so the warp/align SPEED reflects the actual fitted ship.
func TestResolveCargoAndSpeed_OverrideReplacesCargoOnly_SpeedFromFitting(t *testing.T) {
	fit := &recordingFitting{
		sentinelCargoM3: 999, // per-type cargo — must be overridden
		sentinelWarpAUS: 4.5,
		sentinelAlign:   6.0,
	}
	svc := &HaulingService{fitting: fit, logger: applogger.New()}

	req := &models.HaulingRequest{ShipTypeID: 648, CargoCapacity: 5400}
	cargoM3, ship := svc.resolveCargoAndSpeed(context.Background(), req, 123, "token")

	if !fit.called {
		t.Fatalf("expected GetShipFitting to be called for speed even with a cargo override")
	}
	if cargoM3 != 5400 {
		t.Fatalf("expected override cargo 5400, got %v", cargoM3)
	}
	if ship.warpAUS != 4.5 || ship.alignTime != 6.0 {
		t.Fatalf("expected speed from fitting (warp=4.5, align=6.0), got warp=%v align=%v", ship.warpAUS, ship.alignTime)
	}
}

func TestResolveCargoAndSpeed_NoOverrideUsesFitting(t *testing.T) {
	fit := &recordingFitting{
		sentinelCargoM3: 5400,
		sentinelWarpAUS: 4.5,
		sentinelAlign:   6.0,
	}
	svc := &HaulingService{fitting: fit, logger: applogger.New()}

	req := &models.HaulingRequest{ShipTypeID: 648} // CargoCapacity == 0
	cargoM3, ship := svc.resolveCargoAndSpeed(context.Background(), req, 123, "token")

	if !fit.called {
		t.Fatalf("expected GetShipFitting to be called when CargoCapacity == 0")
	}
	if cargoM3 != 5400 {
		t.Fatalf("expected per-type cargo 5400, got %v", cargoM3)
	}
	if ship.warpAUS != 4.5 || ship.alignTime != 6.0 {
		t.Fatalf("expected speed from fitting (warp=4.5, align=6.0), got warp=%v align=%v", ship.warpAUS, ship.alignTime)
	}
}
