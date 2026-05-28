package services

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	applogger "github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

type fakeSkills struct{ s *TradingSkills }

func (f fakeSkills) GetCharacterSkills(_ context.Context, _ int, _ string) (*TradingSkills, error) {
	return f.s, nil
}

type fakeMarket struct{ byRegion map[int][]esi.ESIMarketOrder }

func (f fakeMarket) FetchMarketOrdersForType(_ context.Context, regionID, _ int) ([]esi.ESIMarketOrder, error) {
	return f.byRegion[regionID], nil
}

type fakeVolume struct{}

func (fakeVolume) GetVolumeMetrics(_ context.Context, _, _ int) (*models.VolumeMetrics, error) {
	return &models.VolumeMetrics{DailyVolumeAvg: 1000, LiquidityScore: 50}, nil
}

type fakeCompetition struct{ registered int }

func (f *fakeCompetition) Register(_ context.Context, _, _ int) { f.registered++ }
func (f *fakeCompetition) GetCompetition(_ context.Context, _, _ int) models.CompetitionInfo {
	return models.CompetitionInfo{ChangesPerHour: 1, Source: "baseline"}
}

type fakeTypes struct{}

func (fakeTypes) GetTypeInfo(_ context.Context, _ int) (*database.TypeInfo, error) {
	return &database.TypeInfo{TypeID: 34, Name: "Tritanium"}, nil
}

func sell(price float64) esi.ESIMarketOrder {
	return esi.ESIMarketOrder{OrderID: int64(price * 10), Price: price, VolumeRemain: 100, IsBuyOrder: false}
}
func buy(price float64) esi.ESIMarketOrder {
	return esi.ESIMarketOrder{OrderID: int64(price*10) + 1, Price: price, VolumeRemain: 100, IsBuyOrder: true}
}

func TestCompareHubs_RanksByNetMargin(t *testing.T) {
	const jita, dodixie = 10000002, 10000032
	market := fakeMarket{byRegion: map[int][]esi.ESIMarketOrder{
		jita:    {buy(100), sell(102)}, // thin margin → loss after fees
		dodixie: {buy(100), sell(110)}, // healthy margin
		// Amarr/Rens/Hek: no orders → HasData=false
	}}
	svc := NewMultiHubComparisonService(
		fakeSkills{&TradingSkills{Accounting: 5, BrokerRelations: 5}},
		market, fakeVolume{}, &fakeCompetition{}, fakeTypes{}, applogger.New(),
	)

	res, err := svc.CompareHubs(context.Background(), 34, 123, "token")
	if err != nil {
		t.Fatalf("CompareHubs error: %v", err)
	}
	if res.ItemName != "Tritanium" {
		t.Errorf("item name = %q", res.ItemName)
	}
	if len(res.Hubs) != len(MajorHubs) {
		t.Fatalf("expected %d hub rows, got %d", len(MajorHubs), len(res.Hubs))
	}
	// Best hub must be Dodixie (higher net margin than thin Jita).
	if res.BestHubRegionID != dodixie {
		t.Errorf("best hub = %d, want %d (Dodixie)", res.BestHubRegionID, dodixie)
	}
	// First row is the recommended (highest-margin) hub with data.
	if !res.Hubs[0].HasData || res.Hubs[0].RegionID != dodixie {
		t.Errorf("first row = region %d hasData %v, want Dodixie with data", res.Hubs[0].RegionID, res.Hubs[0].HasData)
	}
	// Spread for Dodixie = (110-100)/100*100 = 10%.
	if !almostEqual(res.Hubs[0].SpreadPercent, 10) {
		t.Errorf("Dodixie spread = %v, want 10", res.Hubs[0].SpreadPercent)
	}
	// Dodixie net margin positive; the thin Jita row negative.
	if res.Hubs[0].NetMarginPercent <= 0 {
		t.Errorf("Dodixie net margin should be positive, got %v", res.Hubs[0].NetMarginPercent)
	}
	// Hubs without data sort last and are flagged.
	last := res.Hubs[len(res.Hubs)-1]
	if last.HasData {
		t.Errorf("expected a no-data hub last, got region %d with data", last.RegionID)
	}
	// Skills applied reflected.
	if !res.SkillsApplied.Applied || !almostEqual(res.SkillsApplied.SalesTaxRate, 0.025) {
		t.Errorf("skills applied = %+v", res.SkillsApplied)
	}
}

func TestCompareHubs_NoProfitableHub_RecommendsNothing(t *testing.T) {
	const jita = 10000002
	// Thin commodity: buy ≈ sell, fees push every hub negative.
	market := fakeMarket{byRegion: map[int][]esi.ESIMarketOrder{
		jita:     {buy(100), sell(101)},
		10000032: {buy(100), sell(100.5)},
	}}
	svc := NewMultiHubComparisonService(
		fakeSkills{&TradingSkills{Accounting: 3, BrokerRelations: 2}},
		market, fakeVolume{}, &fakeCompetition{}, fakeTypes{}, applogger.New(),
	)
	res, err := svc.CompareHubs(context.Background(), 34, 123, "token")
	if err != nil {
		t.Fatalf("CompareHubs error: %v", err)
	}
	// All hubs lose money → no recommendation.
	if res.BestHubRegionID != 0 {
		t.Errorf("best hub = %d, want 0 (no profitable hub)", res.BestHubRegionID)
	}
	// Sanity: the rows with data do have negative margins.
	if res.Hubs[0].HasData && res.Hubs[0].NetMarginPercent > 0 {
		t.Errorf("expected non-positive top margin, got %v", res.Hubs[0].NetMarginPercent)
	}
}
