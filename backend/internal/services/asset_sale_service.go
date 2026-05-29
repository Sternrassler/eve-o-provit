package services

import (
	"context"
	"database/sql"
	"sort"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// takerUnitNet is the per-unit proceeds when selling instantly into a buy order:
// buy price minus sales tax. No broker fee (that applies only to placed orders).
func takerUnitNet(buyPrice, salesTaxRate float64) float64 {
	return buyPrice * (1 - salesTaxRate)
}

// bestBuyInRegion returns the highest active buy-order price across all stations in a
// region, plus the station it sits at.
func bestBuyInRegion(orders []esi.ESIMarketOrder) (price float64, stationID int64, ok bool) {
	for _, o := range orders {
		if !o.IsBuyOrder || o.VolumeRemain <= 0 {
			continue
		}
		if !ok || o.Price > price {
			price, stationID, ok = o.Price, o.LocationID, true
		}
	}
	return price, stationID, ok
}

// bestBuyByStation returns the highest active buy-order price per station.
func bestBuyByStation(orders []esi.ESIMarketOrder) map[int64]float64 {
	best := map[int64]float64{}
	for _, o := range orders {
		if !o.IsBuyOrder || o.VolumeRemain <= 0 {
			continue
		}
		if cur, seen := best[o.LocationID]; !seen || o.Price > cur {
			best[o.LocationID] = o.Price
		}
	}
	return best
}

// aggStack is an aggregated (type, location) quantity, pre-enrichment.
type aggStack struct {
	typeID     int
	locationID int64
	quantity   int
}

// aggregateAssets sums quantities for identical (type, location) pairs.
func aggregateAssets(raw []RawAsset) []aggStack {
	idx := map[[2]int64]int{} // (typeID, locationID) -> index in out
	var out []aggStack
	for _, r := range raw {
		key := [2]int64{int64(r.TypeID), r.LocationID}
		if i, ok := idx[key]; ok {
			out[i].quantity += r.Quantity
			continue
		}
		idx[key] = len(out)
		out = append(out, aggStack{typeID: r.TypeID, locationID: r.LocationID, quantity: r.Quantity})
	}
	return out
}

// AssetSaleService lists owned items and ranks taker sell locations (Issue #107).
type AssetSaleService struct {
	skills  SkillsServicer
	market  hubOrderFetcher
	types   typeNamer
	assets  assetFetcher
	sdeRepo *database.SDERepository
	sdeDB   *sql.DB
	logger  *logger.Logger
}

// NewAssetSaleService creates a new service.
func NewAssetSaleService(
	skills SkillsServicer,
	market hubOrderFetcher,
	types typeNamer,
	assets assetFetcher,
	sdeRepo *database.SDERepository,
	sdeDB *sql.DB,
	logger *logger.Logger,
) *AssetSaleService {
	return &AssetSaleService{skills: skills, market: market, types: types, assets: assets, sdeRepo: sdeRepo, sdeDB: sdeDB, logger: logger}
}

// ListAssets returns the character's owned items aggregated by (type, location).
func (s *AssetSaleService) ListAssets(ctx context.Context, characterID int, accessToken string) (*models.AssetsResponse, error) {
	raw, err := s.assets.FetchCharacterAssets(ctx, characterID, accessToken)
	if err != nil {
		return nil, err
	}
	stacks := aggregateAssets(raw)
	items := make([]models.AssetItem, 0, len(stacks))
	for _, st := range stacks {
		item := models.AssetItem{TypeID: st.typeID, Quantity: st.quantity, LocationID: st.locationID}
		if info, err := s.types.GetTypeInfo(ctx, st.typeID); err == nil && info != nil {
			item.Name = info.Name
			item.Marketable = info.MarketGroup != nil
		}
		if sysID, err := s.sdeRepo.GetSystemIDForLocation(ctx, st.locationID); err == nil {
			item.SystemID = int(sysID)
			if rid, err := s.sdeRepo.GetRegionIDForSystem(ctx, sysID); err == nil {
				item.RegionID = rid
			}
		}
		if name, err := s.sdeRepo.GetStationName(ctx, st.locationID); err == nil {
			item.LocationName = name
		}
		items = append(items, item)
	}
	return &models.AssetsResponse{Assets: items, Count: len(items)}, nil
}

// SellOptions ranks taker sell locations for one owned item.
func (s *AssetSaleService) SellOptions(ctx context.Context, req *models.SellOptionsRequest, characterID int, accessToken string) (*models.SellOptionsResponse, error) {
	skills, serr := s.skills.GetCharacterSkills(ctx, characterID, accessToken)
	if serr != nil || skills == nil {
		skills = &TradingSkills{}
	}
	salesTaxRate := SalesTaxRate(skills.Accounting)

	name := ""
	if info, err := s.types.GetTypeInfo(ctx, req.TypeID); err == nil && info != nil {
		name = info.Name
	}

	resp := &models.SellOptionsResponse{
		TypeID: req.TypeID, Name: name, Quantity: req.Quantity,
		SkillsApplied: models.SkillsApplied{
			Applied: serr == nil, Accounting: skills.Accounting, BrokerRelations: skills.BrokerRelations,
			SalesTaxRate:  salesTaxRate,
			BrokerFeeRate: BrokerFeeRate(skills.BrokerRelations, skills.AdvancedBrokerRelations, skills.FactionStanding, skills.CorpStanding),
		},
	}

	originSys, err := s.sdeRepo.GetSystemIDForLocation(ctx, req.LocationID)
	if err != nil {
		return resp, nil // origin unresolvable -> empty options
	}
	resp.OriginSystemID = int(originSys)
	currentRegion, _ := s.sdeRepo.GetRegionIDForSystem(ctx, originSys)

	var options []models.SellOption

	// 5 major hubs: region-best buy order, route to hub system.
	for _, hub := range MajorHubs {
		orders, err := s.market.FetchMarketOrdersForType(ctx, hub.RegionID, req.TypeID)
		if err != nil {
			s.logger.Warn("sell-options: hub fetch failed", "error", err, "region", hub.RegionID)
			options = append(options, models.SellOption{Scope: "hub", RegionID: hub.RegionID, RegionName: hub.RegionName, SystemName: hub.Name, HasData: false})
			continue
		}
		price, stationID, ok := bestBuyInRegion(orders)
		opt := s.buildOption(ctx, "hub", hub.RegionID, hub.RegionName, stationID, int64(hub.SystemID), originSys, price, ok, req, salesTaxRate)
		if opt.SystemName == "" {
			opt.SystemName = hub.Name
		}
		options = append(options, opt)
	}

	// Current region: per-station best buy, route to each station's system.
	if currentRegion != 0 {
		if orders, err := s.market.FetchMarketOrdersForType(ctx, currentRegion, req.TypeID); err == nil {
			for stationID, price := range bestBuyByStation(orders) {
				sysID, err := s.sdeRepo.GetSystemIDForLocation(ctx, stationID)
				if err != nil {
					continue // unresolvable (player structure) -> skip
				}
				opt := s.buildOption(ctx, "current_region", currentRegion, "", stationID, sysID, originSys, price, true, req, salesTaxRate)
				options = append(options, opt)
			}
		}
	}

	sort.SliceStable(options, func(i, j int) bool { return options[i].TotalNet > options[j].TotalNet })
	resp.Options = options
	for i := range options {
		if options[i].HasData && options[i].TotalNet > 0 {
			best := options[i]
			resp.Best = &best
			break
		}
	}
	return resp, nil
}

// buildOption computes net + route for one sell location. destSys is the system to route
// to; stationID is the buy-order station (for the waypoint + name).
func (s *AssetSaleService) buildOption(ctx context.Context, scope string, regionID int, regionName string, stationID, destSys, originSys int64, buyPrice float64, hasData bool, req *models.SellOptionsRequest, salesTaxRate float64) models.SellOption {
	opt := models.SellOption{Scope: scope, RegionID: regionID, RegionName: regionName, StationID: stationID, HasData: hasData}
	if name, err := s.sdeRepo.GetStationName(ctx, stationID); err == nil {
		opt.StationName = name
	}
	if name, err := s.sdeRepo.GetSystemName(ctx, destSys); err == nil {
		opt.SystemName = name
	}
	if !hasData {
		return opt
	}
	opt.BuyPrice = buyPrice
	opt.UnitNet = takerUnitNet(buyPrice, salesTaxRate)
	opt.TotalNet = opt.UnitNet * float64(req.Quantity)

	if destSys == originSys {
		opt.Jumps = 0
		opt.TravelTimeMin = 0
		opt.SecurityRisk = securityRisk(minRouteSecurityStatus(s.sdeDB, []int64{originSys}))
		return opt
	}
	travel, err := navigation.CalculateTravelTime(s.sdeDB, originSys, destSys, &navigation.NavigationParams{AvoidLowSec: req.AvoidLowSec}, false)
	if err != nil {
		opt.HasData = false // no route (e.g. high-sec-only filter) -> not actionable
		return opt
	}
	opt.Jumps = travel.Jumps
	opt.TravelTimeMin = travel.TotalMinutes
	opt.SecurityRisk = securityRisk(minRouteSecurityStatus(s.sdeDB, travel.Route))
	return opt
}

// minRouteSecurityStatus returns the lowest securityStatus along a route (1.0 if empty/unknown).
func minRouteSecurityStatus(db *sql.DB, route []int64) float64 {
	if len(route) == 0 {
		return 1.0
	}
	minSec := 1.0
	for _, sysID := range route {
		var sec float64
		if err := db.QueryRowContext(context.Background(), "SELECT securityStatus FROM mapSolarSystems WHERE _key = ?", sysID).Scan(&sec); err != nil {
			continue
		}
		if sec < minSec {
			minSec = sec
		}
	}
	return minSec
}
