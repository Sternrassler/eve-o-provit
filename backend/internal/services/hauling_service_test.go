package services

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	applogger "github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// recordingFitting records whether GetShipFitting was called and returns a sentinel
// per-type cargo so we can prove which path produced the cargo value.
type recordingFitting struct {
	called          bool
	sentinelCargoM3 float64
}

func (f *recordingFitting) GetShipFitting(_ context.Context, _, _ int, _ string) (*FittingData, error) {
	f.called = true
	return &FittingData{Bonuses: FittingBonuses{EffectiveCargo: f.sentinelCargoM3}}, nil
}
func (f *recordingFitting) InvalidateFittingCache(_ context.Context, _, _ int) {}
func (f *recordingFitting) EnrichShipsEffectiveCargo(_ context.Context, _ int, _ []models.CharacterAssetShip, _ string) {
}
func (f *recordingFitting) EffectiveCargoForActiveShip(_ context.Context, _, _ int, _ int64, _ string) (float64, bool) {
	return 0, false
}

func TestResolveCargoAndSpeed_OverrideWinsAndSkipsFitting(t *testing.T) {
	fit := &recordingFitting{sentinelCargoM3: 999} // per-type path would yield 999
	svc := &HaulingService{fitting: fit, logger: applogger.New()}

	req := &models.HaulingRequest{ShipTypeID: 648, CargoCapacity: 5400}
	cargoM3, _ := svc.resolveCargoAndSpeed(context.Background(), req, 123, "token")

	if fit.called {
		t.Fatalf("expected GetShipFitting NOT to be called when CargoCapacity > 0")
	}
	if cargoM3 != 5400 {
		t.Fatalf("expected override cargo 5400, got %v", cargoM3)
	}
}

func TestResolveCargoAndSpeed_NoOverrideUsesFitting(t *testing.T) {
	fit := &recordingFitting{sentinelCargoM3: 5400}
	svc := &HaulingService{fitting: fit, logger: applogger.New()}

	req := &models.HaulingRequest{ShipTypeID: 648} // CargoCapacity == 0
	cargoM3, _ := svc.resolveCargoAndSpeed(context.Background(), req, 123, "token")

	if !fit.called {
		t.Fatalf("expected GetShipFitting to be called when CargoCapacity == 0")
	}
	if cargoM3 != 5400 {
		t.Fatalf("expected per-type cargo 5400, got %v", cargoM3)
	}
}
