package services

import (
	"context"
	"database/sql"
	"sort"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

const defaultMaxHaulingRoutes = 15

// HaulingService finds neighborhood hauling routes (#45): origin region + adjacent
// regions, cross-region station→station arbitrage, optimal cargo per route.
type HaulingService struct {
	sdeRepo     *database.SDERepository
	routeFinder *RouteFinder
	fitting     FittingServicer
	skills      SkillsServicer
	charHelper  *CharacterHelper
	sdeDB       *sql.DB
	optimizer   *HaulingOptimizer
	logger      *logger.Logger
}

// NewHaulingService creates a new hauling service.
func NewHaulingService(
	sdeRepo *database.SDERepository,
	routeFinder *RouteFinder,
	fitting FittingServicer,
	skills SkillsServicer,
	charHelper *CharacterHelper,
	sdeDB *sql.DB,
	logger *logger.Logger,
) *HaulingService {
	return &HaulingService{
		sdeRepo: sdeRepo, routeFinder: routeFinder, fitting: fitting, skills: skills,
		charHelper: charHelper, sdeDB: sdeDB, optimizer: NewHaulingOptimizer(), logger: logger,
	}
}

// FindRoutes computes ranked hauling routes for the request.
func (s *HaulingService) FindRoutes(ctx context.Context, req *models.HaulingRequest, characterID int, accessToken string) (*models.HaulingResponse, error) {
	// 1. Origin region.
	origin := req.OriginRegionID
	if origin <= 0 {
		loc, err := s.charHelper.GetCharacterLocation(ctx, characterID, accessToken)
		if err != nil {
			return nil, err
		}
		region, err := s.sdeRepo.GetRegionIDForSystem(ctx, loc.SolarSystemID)
		if err != nil {
			return nil, err
		}
		origin = region
	}

	// 2. Region set = origin + neighbors.
	neighbors, err := s.sdeRepo.GetNeighborRegions(ctx, origin)
	if err != nil {
		return nil, err
	}
	regions := append([]int{origin}, neighbors...)

	// 3. Skills + ship cargo.
	skills, serr := s.skills.GetCharacterSkills(ctx, characterID, accessToken)
	if serr != nil || skills == nil {
		skills = &TradingSkills{}
	}
	salesTaxRate := SalesTaxRate(skills.Accounting)
	cargoM3 := 0.0
	if fit, err := s.fitting.GetShipFitting(ctx, characterID, req.ShipTypeID, accessToken); err == nil && fit != nil {
		cargoM3 = fit.Bonuses.EffectiveCargo
	}

	// 4. Load orders for the region set.
	var orders []matchOrder
	for _, region := range regions {
		regionOrders, err := s.routeFinder.fetchMarketOrders(ctx, region)
		if err != nil {
			s.logger.Warn("hauling: region order fetch failed", "error", err, "region", region)
			continue
		}
		for _, o := range regionOrders {
			orders = append(orders, matchOrder{
				TypeID: o.TypeID, StationID: o.LocationID, IsBuyOrder: o.IsBuyOrder,
				Price: o.Price, VolumeRemain: o.VolumeRemain,
			})
		}
	}

	// 5. Match + build routes.
	cands := MatchHauls(orders, salesTaxRate)
	maxRoutes := req.MaxRoutes
	if maxRoutes <= 0 {
		maxRoutes = defaultMaxHaulingRoutes
	}

	routes := make([]models.HaulingRoute, 0, len(cands))
	for _, c := range cands {
		route, ok := s.buildRoute(ctx, c, cargoM3, req.Capital, req.AvoidLowSec)
		if ok {
			routes = append(routes, route)
		}
	}

	sort.SliceStable(routes, func(i, j int) bool { return routes[i].ISKPerHour > routes[j].ISKPerHour })
	if len(routes) > maxRoutes {
		routes = routes[:maxRoutes]
	}

	return &models.HaulingResponse{
		OriginRegionID: origin,
		RegionsScanned: regions,
		Routes:         routes,
		SkillsApplied: models.SkillsApplied{
			Applied:         serr == nil,
			Accounting:      skills.Accounting,
			BrokerRelations: skills.BrokerRelations,
			SalesTaxRate:    salesTaxRate,
			BrokerFeeRate:   BrokerFeeRate(skills.BrokerRelations, skills.AdvancedBrokerRelations, skills.FactionStanding, skills.CorpStanding),
		},
	}, nil
}

// buildRoute enriches a candidate with volumes, packs the cargo, resolves the route
// and computes time/risk/ISK-per-hour. Returns ok=false if not viable.
func (s *HaulingService) buildRoute(ctx context.Context, c RouteCandidate, cargoM3, capital float64, avoidLowSec bool) (models.HaulingRoute, bool) {
	// Enrich items with name + unit volume.
	items := make([]HaulItem, 0, len(c.Items))
	for _, it := range c.Items {
		vol, err := cargo.GetItemVolume(s.sdeDB, int64(it.TypeID))
		if err != nil || vol == nil || vol.Volume <= 0 {
			continue
		}
		it.UnitVolume = vol.Volume
		if info, err := s.sdeRepo.GetTypeInfo(ctx, it.TypeID); err == nil && info != nil {
			it.Name = info.Name
		}
		items = append(items, it)
	}
	load := s.optimizer.FillCargo(items, cargoM3, capital)
	if len(load.Items) == 0 {
		return models.HaulingRoute{}, false
	}

	buySystem, err := s.sdeRepo.GetSystemIDForLocation(ctx, c.BuyStationID)
	if err != nil {
		return models.HaulingRoute{}, false
	}
	sellSystem, err := s.sdeRepo.GetSystemIDForLocation(ctx, c.SellStationID)
	if err != nil {
		return models.HaulingRoute{}, false
	}

	travel, err := navigation.CalculateTravelTime(s.sdeDB, buySystem, sellSystem, &navigation.NavigationParams{AvoidLowSec: avoidLowSec}, false)
	if err != nil {
		return models.HaulingRoute{}, false // e.g. no high-sec route when avoidLowSec
	}
	minSec := s.minRouteSecurity(travel.Route)
	if avoidLowSec && minSec < 0.5 {
		return models.HaulingRoute{}, false
	}

	iskPerHour := 0.0
	if travel.TotalMinutes > 0 {
		iskPerHour = load.TotalProfit / (travel.TotalMinutes / 60.0)
	}

	profitByType := map[int]float64{}
	buyByType := map[int]float64{}
	for _, in := range items {
		profitByType[in.TypeID] = in.ProfitPerUnit
		buyByType[in.TypeID] = in.BuyPrice
	}
	outItems := make([]models.HaulingItem, 0, len(load.Items))
	for _, li := range load.Items {
		ppu := profitByType[li.TypeID]
		buy := buyByType[li.TypeID]
		var vol float64
		if v, err := cargo.GetItemVolume(s.sdeDB, int64(li.TypeID)); err == nil && v != nil {
			vol = v.Volume
		}
		ppm3 := 0.0
		if vol > 0 {
			ppm3 = ppu / vol
		}
		outItems = append(outItems, models.HaulingItem{
			TypeID: li.TypeID, Name: li.Name, Quantity: li.Quantity,
			BuyPrice: buy, SellPrice: buy + ppu, UnitVolume: vol,
			Profit: li.Profit, ProfitPerM3: ppm3,
		})
	}

	return models.HaulingRoute{
		BuyStationID:    c.BuyStationID,
		BuyStationName:  s.stationName(ctx, c.BuyStationID),
		BuySystemName:   s.systemName(ctx, buySystem),
		SellStationID:   c.SellStationID,
		SellStationName: s.stationName(ctx, c.SellStationID),
		SellSystemName:  s.systemName(ctx, sellSystem),
		Jumps:           travel.Jumps,
		TravelTimeMin:   travel.TotalMinutes,
		SecurityRisk:    securityRisk(minSec),
		TotalProfit:     load.TotalProfit,
		TotalCapital:    load.TotalCapital,
		TotalVolume:     load.TotalVolume,
		ISKPerHour:      iskPerHour,
		Items:           outItems,
	}, true
}

func (s *HaulingService) minRouteSecurity(route []int64) float64 {
	if len(route) == 0 {
		return 1.0
	}
	min := 1.0
	for _, sysID := range route {
		var sec float64
		if err := s.sdeDB.QueryRowContext(context.Background(),
			"SELECT securityStatus FROM mapSolarSystems WHERE _key = ?", sysID).Scan(&sec); err != nil {
			continue
		}
		if sec < min {
			min = sec
		}
	}
	return min
}

func (s *HaulingService) stationName(ctx context.Context, stationID int64) string {
	if name, err := s.sdeRepo.GetStationName(ctx, stationID); err == nil {
		return name
	}
	return ""
}

func (s *HaulingService) systemName(ctx context.Context, systemID int64) string {
	if name, err := s.sdeRepo.GetSystemName(ctx, systemID); err == nil {
		return name
	}
	return ""
}

func securityRisk(minSec float64) string {
	switch {
	case minSec >= 0.5:
		return "safe"
	case minSec > 0.0:
		return "caution"
	default:
		return "danger"
	}
}
