// Package services provides business logic for trading operations
package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
)

// RouteCalculator handles route calculation and optimization
type RouteCalculator struct {
	sdeRepo    *database.SDERepository
	sdeDB      *sql.DB
	feeService FeeServicer
}

// NewRouteCalculator creates a new route optimizer instance
func NewRouteCalculator(sdeRepo *database.SDERepository, sdeDB *sql.DB, feeService FeeServicer) *RouteCalculator {
	return &RouteCalculator{
		sdeRepo:    sdeRepo,
		sdeDB:      sdeDB,
		feeService: feeService,
	}
}

// CalculateRoute calculates a complete trading route with travel time and profit
// cargoCapacity is the effective capacity (with skills already applied)
// baseCapacity and skillBonus are optional - if 0, they'll match cargoCapacity
func (ro *RouteCalculator) CalculateRoute(ctx context.Context, item models.ItemPair, cargoCapacity float64) (models.TradingRoute, error) {
	return ro.CalculateRouteWithCapacityInfo(ctx, item, cargoCapacity, cargoCapacity, 0, 0, nil, nil)
}

// CalculateRouteWithCapacityInfo calculates a route with detailed capacity and navigation information
// warpSpeed and alignTime are optional pointers - if nil, navigation package uses defaults
func (ro *RouteCalculator) CalculateRouteWithCapacityInfo(ctx context.Context, item models.ItemPair, effectiveCapacity, baseCapacity, skillBonusPercent, fittingBonusM3 float64, warpSpeed, alignTime *float64) (models.TradingRoute, error) {
	var route models.TradingRoute

	// Use effective capacity for calculations
	cargoCapacity := effectiveCapacity

	// Calculate quantity that fits in cargo (per tour)
	if item.ItemVolume <= 0 {
		return route, fmt.Errorf("invalid item volume: %f", item.ItemVolume)
	}
	quantityPerTour := int(cargoCapacity / item.ItemVolume)
	if quantityPerTour <= 0 {
		return route, fmt.Errorf("item too large for cargo")
	}

	// Multi-tour calculation
	// Calculate number of tours based on available volume
	var numberOfTours int
	var totalQuantity int

	if item.AvailableQuantity > 0 && item.AvailableVolumeM3 > 0 {
		// Calculate max tours based on available volume
		maxToursFromVolume := int((item.AvailableVolumeM3 / cargoCapacity) + 0.5) // Round up
		if maxToursFromVolume < 1 {
			maxToursFromVolume = 1
		}

		// Limit to max 10 tours (practical limit)
		numberOfTours = maxToursFromVolume
		if numberOfTours > 10 {
			numberOfTours = 10
		}

		// Calculate total quantity across all tours
		totalQuantity = item.AvailableQuantity
		if totalQuantity > quantityPerTour*numberOfTours {
			totalQuantity = quantityPerTour * numberOfTours
		}
	} else {
		// Fallback: single tour
		numberOfTours = 1
		totalQuantity = quantityPerTour
	}

	// Calculate profit per tour and total profit
	profitPerUnit := item.SellPrice - item.BuyPrice
	totalProfit := profitPerUnit * float64(totalQuantity)
	profitPerTour := totalProfit / float64(numberOfTours)

	// Build navigation parameters from provided deterministic values
	var navParams *navigation.NavigationParams
	if warpSpeed != nil || alignTime != nil {
		navParams = &navigation.NavigationParams{
			WarpSpeed: warpSpeed,
			AlignTime: alignTime,
		}
	}

	// Calculate travel time with navigation parameters (uses defaults if navParams is nil)
	travelResult, err := navigation.CalculateTravelTime(ro.sdeDB, item.BuySystemID, item.SellSystemID, navParams)
	if err != nil {
		return route, fmt.Errorf("failed to calculate route: %w", err)
	}

	// Extract travel times
	oneWaySeconds := travelResult.TotalSeconds
	roundTripSeconds := oneWaySeconds * 2

	// Station Trading: Use minimum time for order cycling (5 minutes base time)
	if item.BuySystemID == item.SellSystemID || travelResult.Jumps == 0 {
		oneWaySeconds = 300.0    // 5 minutes for station trading
		roundTripSeconds = 600.0 // Same for roundtrip
	}

	// Multi-tour time calculation
	// (numberOfTours - 1) full roundtrips + 1 one-way trip
	var totalTimeSeconds float64
	if numberOfTours > 1 {
		totalTimeSeconds = float64(numberOfTours-1)*roundTripSeconds + oneWaySeconds
	} else {
		totalTimeSeconds = roundTripSeconds
	}
	totalTimeMinutes := totalTimeSeconds / 60.0

	// ISK/h calculation moved after fee calculation to use net profit

	// Get system and station names
	buySystemName, buyStationName := ro.getLocationNames(ctx, item.BuySystemID, item.BuyStationID)
	sellSystemName, sellStationName := ro.getLocationNames(ctx, item.SellSystemID, item.SellStationID)

	// Get security status for both systems
	buySecurityStatus := ro.getSystemSecurityStatus(ctx, item.BuySystemID)
	sellSecurityStatus := ro.getSystemSecurityStatus(ctx, item.SellSystemID)

	// Calculate minimum security status across entire route
	minRouteSecurity := ro.getMinRouteSecurityStatus(ctx, travelResult.Route)

	// Calculate trading fees (Issue #39)
	// Use worst-case assumptions (all skills = 0) for conservative estimates
	// Fees are calculated based on total buy/sell order values
	buyValue := item.BuyPrice * float64(totalQuantity)
	sellValue := item.SellPrice * float64(totalQuantity)

	// Calculate individual fees using worst-case skills (all = 0)
	buyBrokerFee := ro.feeService.CalculateBrokerFee(
		0, // BrokerRelations = 0
		0, // AdvancedBrokerRelations = 0
		0, // FactionStanding = 0
		0, // CorpStanding = 0
		buyValue,
	)
	sellBrokerFee := ro.feeService.CalculateBrokerFee(
		0, // BrokerRelations = 0
		0, // AdvancedBrokerRelations = 0
		0, // FactionStanding = 0
		0, // CorpStanding = 0
		sellValue,
	)
	salesTax := ro.feeService.CalculateSalesTax(
		0, // Accounting = 0
		sellValue,
	)

	// Sum all fees
	totalFees := buyBrokerFee + sellBrokerFee + salesTax

	// Calculate broker fees (combined)
	brokerFees := buyBrokerFee + sellBrokerFee

	// Estimated relist fee is the sell broker fee
	// (represents the cost if the order needs to be modified/relisted)
	estimatedRelistFee := sellBrokerFee

	// Calculate net profit (total profit minus all fees)
	netProfit := totalProfit - totalFees

	// Calculate gross profit (this is totalProfit before fees)
	grossProfit := totalProfit

	// Calculate ISK per hour using NET profit (after fees)
	var iskPerHour float64
	if totalTimeSeconds > 0 {
		// Calculate theoretical ISK/h (assuming infinite supply)
		theoreticalISKPerHour := (netProfit / totalTimeSeconds) * 3600

		// Calculate realistic ISK/h based on available quantity
		// If the trip takes >1 hour, cap ISK/h to actual profit achievable
		maxTripsPerHour := 3600.0 / totalTimeSeconds

		// If we can't complete even one full trip set per hour, use proportional profit
		if maxTripsPerHour < 1.0 {
			// Less than 1 full trip set per hour - use proportional profit
			iskPerHour = netProfit * maxTripsPerHour
		} else {
			// Can do multiple trip sets - use theoretical ISK/h
			iskPerHour = theoreticalISKPerHour
		}
	}

	// Calculate investment (total cost to buy)
	totalInvestment := item.BuyPrice * float64(totalQuantity)

	// Calculate margin percentages
	var grossMarginPercent float64
	var netProfitPercent float64
	if totalInvestment > 0 {
		grossMarginPercent = (grossProfit / totalInvestment) * 100
		netProfitPercent = (netProfit / totalInvestment) * 100
	}

	// Calculate cargo utilization
	cargoUsed := item.ItemVolume * float64(quantityPerTour)
	cargoUtilization := 0.0
	if cargoCapacity > 0 {
		cargoUtilization = (cargoUsed / cargoCapacity) * 100
	}

	route = models.TradingRoute{
		ItemTypeID:             item.TypeID,
		ItemName:               item.ItemName,
		BuySystemID:            item.BuySystemID,
		BuySystemName:          buySystemName,
		BuyStationID:           item.BuyStationID,
		BuyStationName:         buyStationName,
		BuyPrice:               item.BuyPrice,
		SellSystemID:           item.SellSystemID,
		SellSystemName:         sellSystemName,
		SellStationID:          item.SellStationID,
		SellStationName:        sellStationName,
		SellPrice:              item.SellPrice,
		BuySecurityStatus:      buySecurityStatus,
		SellSecurityStatus:     sellSecurityStatus,
		MinRouteSecurityStatus: minRouteSecurity,
		Quantity:               totalQuantity,
		ProfitPerUnit:          profitPerUnit,
		TotalProfit:            totalProfit,
		SpreadPercent:          item.SpreadPercent,
		TravelTimeSeconds:      oneWaySeconds,
		RoundTripSeconds:       roundTripSeconds,
		ISKPerHour:             iskPerHour,
		Jumps:                  travelResult.Jumps,
		ItemVolume:             item.ItemVolume,
		// Multi-tour fields
		NumberOfTours:    numberOfTours,
		ProfitPerTour:    profitPerTour,
		TotalTimeMinutes: totalTimeMinutes,
		// Navigation skills fields (deprecated - keeping for backward compatibility)
		BaseTravelTimeSeconds:    oneWaySeconds, // Now same as TravelTimeSeconds
		SkilledTravelTimeSeconds: oneWaySeconds, // Now same as TravelTimeSeconds
		BaseISKPerHour:           iskPerHour,    // Now same as ISKPerHour
		TimeImprovementPercent:   0,             // No longer calculated (deterministic values from frontend)
		// Trading fees fields (Issue #39)
		BuyBrokerFee:       buyBrokerFee,
		SellBrokerFee:      sellBrokerFee,
		BrokerFees:         brokerFees,
		SalesTax:           salesTax,
		EstimatedRelistFee: estimatedRelistFee,
		TotalFees:          totalFees,
		GrossProfit:        grossProfit,
		GrossMarginPercent: grossMarginPercent,
		NetProfit:          netProfit,
		NetProfitPercent:   netProfitPercent,
		// Cargo fields
		CargoUsed:         cargoUsed,
		CargoCapacity:     cargoCapacity,
		CargoUtilization:  cargoUtilization,
		BaseCargoCapacity: baseCapacity,
		SkillBonusPercent: skillBonusPercent,
		FittingBonusM3:    fittingBonusM3,
	}

	return route, nil
}

// Helper functions

func (ro *RouteCalculator) getLocationNames(ctx context.Context, systemID, stationID int64) (string, string) {
	// Get system name from SDE
	systemName, err := ro.sdeRepo.GetSystemName(ctx, systemID)
	if err != nil {
		log.Printf("Warning: failed to get system name for %d: %v", systemID, err)
		systemName = fmt.Sprintf("System-%d", systemID)
	}

	// Get station name from SDE
	stationName, err := ro.sdeRepo.GetStationName(ctx, stationID)
	if err != nil {
		log.Printf("Warning: failed to get station name for %d: %v", stationID, err)
		stationName = fmt.Sprintf("Station-%d", stationID)
	}

	return systemName, stationName
}

// getSystemSecurityStatus retrieves the security status of a solar system from SDE
func (ro *RouteCalculator) getSystemSecurityStatus(ctx context.Context, systemID int64) float64 {
	secStatus, err := ro.sdeRepo.GetSystemSecurityStatus(ctx, systemID)
	if err != nil {
		log.Printf("Warning: failed to get security status for system %d: %v", systemID, err)
		return 1.0 // Default to high-sec if lookup fails
	}
	return secStatus
}

// getMinRouteSecurityStatus finds the minimum security status across all systems in a route
func (ro *RouteCalculator) getMinRouteSecurityStatus(ctx context.Context, route []int64) float64 {
	if len(route) == 0 {
		return 1.0 // Default to high-sec if no route
	}

	minSecurity := 1.0
	for _, systemID := range route {
		security := ro.getSystemSecurityStatus(ctx, systemID)
		if security < minSecurity {
			minSecurity = security
		}
	}

	return minSecurity
}
