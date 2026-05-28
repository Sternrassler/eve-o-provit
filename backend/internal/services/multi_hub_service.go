package services

import (
	"context"
	"sort"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// Small dependency interfaces keep MultiHubComparisonService unit-testable.
type hubOrderFetcher interface {
	FetchMarketOrdersForType(ctx context.Context, regionID, typeID int) ([]esi.ESIMarketOrder, error)
}

type volumeProvider interface {
	GetVolumeMetrics(ctx context.Context, typeID, regionID int) (*models.VolumeMetrics, error)
}

type competitionProvider interface {
	Register(ctx context.Context, typeID, regionID int)
	GetCompetition(ctx context.Context, typeID, regionID int) models.CompetitionInfo
}

type typeNamer interface {
	GetTypeInfo(ctx context.Context, typeID int) (*database.TypeInfo, error)
}

// MultiHubComparisonService compares one item's station-trading profitability across the
// major trade hubs, applying the character's fee skills (Issue #43). The ranking logic is
// deliberately reusable: #107 (sell-from-assets) consumes the same per-hub net figures.
type MultiHubComparisonService struct {
	skills      SkillsServicer
	market      hubOrderFetcher
	volume      volumeProvider
	competition competitionProvider
	types       typeNamer
	logger      *logger.Logger
}

// NewMultiHubComparisonService creates a new service.
func NewMultiHubComparisonService(
	skills SkillsServicer,
	market hubOrderFetcher,
	volume volumeProvider,
	competition competitionProvider,
	types typeNamer,
	logger *logger.Logger,
) *MultiHubComparisonService {
	return &MultiHubComparisonService{
		skills:      skills,
		market:      market,
		volume:      volume,
		competition: competition,
		types:       types,
		logger:      logger,
	}
}

// CompareHubs builds the per-hub comparison for an item, sorted by net margin descending.
func (s *MultiHubComparisonService) CompareHubs(ctx context.Context, typeID, characterID int, accessToken string) (*models.HubComparisonResult, error) {
	// Skills once (graceful: GetCharacterSkills returns defaults on ESI failure).
	skills, err := s.skills.GetCharacterSkills(ctx, characterID, accessToken)
	if err != nil || skills == nil {
		skills = &TradingSkills{}
	}
	salesTaxRate := SalesTaxRate(skills.Accounting)
	brokerFeeRate := BrokerFeeRate(skills.BrokerRelations, skills.AdvancedBrokerRelations, skills.FactionStanding, skills.CorpStanding)

	itemName := ""
	if info, err := s.types.GetTypeInfo(ctx, typeID); err == nil && info != nil {
		itemName = info.Name
	}

	rows := make([]models.HubRow, 0, len(MajorHubs))
	for _, hub := range MajorHubs {
		s.competition.Register(ctx, typeID, hub.RegionID)
		rows = append(rows, s.buildRow(ctx, typeID, hub, salesTaxRate, brokerFeeRate))
	}

	// Sort: rows with data first, then by net margin descending.
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].HasData != rows[j].HasData {
			return rows[i].HasData
		}
		return rows[i].NetMarginPercent > rows[j].NetMarginPercent
	})

	// Only recommend a hub that is actually profitable. Rows are sorted by net
	// margin descending, so the first profitable row is the best one; if even the
	// top hub loses money (e.g. thin-margin commodities), recommend nothing.
	bestHubRegionID := 0
	for _, r := range rows {
		if r.HasData && r.NetMarginPercent > 0 {
			bestHubRegionID = r.RegionID
			break
		}
	}

	return &models.HubComparisonResult{
		TypeID:          typeID,
		ItemName:        itemName,
		Hubs:            rows,
		BestHubRegionID: bestHubRegionID,
		SkillsApplied: models.SkillsApplied{
			Applied:         err == nil,
			Accounting:      skills.Accounting,
			BrokerRelations: skills.BrokerRelations,
			SalesTaxRate:    salesTaxRate,
			BrokerFeeRate:   brokerFeeRate,
		},
	}, nil
}

// buildRow computes one hub's comparison row.
func (s *MultiHubComparisonService) buildRow(ctx context.Context, typeID int, hub Hub, salesTaxRate, brokerFeeRate float64) models.HubRow {
	row := models.HubRow{
		RegionID:   hub.RegionID,
		RegionName: hub.RegionName,
		HubName:    hub.Name,
		SystemID:   hub.SystemID,
		Tier:       hub.Tier,
	}

	// Volume first: GetVolumeMetrics lazy-populates price_history, which the
	// competition baseline then reads. Both are set even for hubs without current
	// orders.
	if vm, err := s.volume.GetVolumeMetrics(ctx, typeID, hub.RegionID); err == nil && vm != nil {
		row.DailyVolume = vm.DailyVolumeAvg
		row.LiquidityScore = vm.LiquidityScore
	}
	row.Competition = s.competition.GetCompetition(ctx, typeID, hub.RegionID)

	orders, err := s.market.FetchMarketOrdersForType(ctx, hub.RegionID, typeID)
	if err != nil {
		s.logger.Warn("multi-hub: fetch orders failed", "error", err, "type", typeID, "region", hub.RegionID)
		return row
	}

	bestBuy, bestSell, ok := bestPrices(orders)
	if !ok {
		return row
	}

	row.HasData = true
	row.BuyPrice = bestBuy
	row.SellPrice = bestSell
	if bestBuy > 0 {
		row.SpreadPercent = (bestSell - bestBuy) / bestBuy * 100
	}

	// Station-trading net per unit: buy at buy price (+broker), sell at sell price (-broker -tax).
	buyCost := bestBuy * (1 + brokerFeeRate)
	sellRevenue := bestSell * (1 - brokerFeeRate - salesTaxRate)
	row.NetProfitPerUnit = sellRevenue - buyCost
	if buyCost > 0 {
		row.NetMarginPercent = row.NetProfitPerUnit / buyCost * 100
	}

	return row
}

// bestPrices returns the highest active buy price and lowest active sell price.
func bestPrices(orders []esi.ESIMarketOrder) (bestBuy, bestSell float64, ok bool) {
	haveBuy, haveSell := false, false
	for _, o := range orders {
		if o.VolumeRemain <= 0 {
			continue
		}
		if o.IsBuyOrder {
			if !haveBuy || o.Price > bestBuy {
				bestBuy = o.Price
				haveBuy = true
			}
		} else {
			if !haveSell || o.Price < bestSell {
				bestSell = o.Price
				haveSell = true
			}
		}
	}
	return bestBuy, bestSell, haveBuy && haveSell
}
