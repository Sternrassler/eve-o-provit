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

// RouteOptimizer handles route calculation and optimization
type RouteOptimizer struct {
	sdeRepo *database.SDERepository
	sdeDB   *sql.DB
}

// NewRouteOptimizer creates a new route optimizer instance
func NewRouteOptimizer(sdeRepo *database.SDERepository, sdeDB *sql.DB) *RouteOptimizer {
	return &RouteOptimizer{
		sdeRepo: sdeRepo,
		sdeDB:   sdeDB,
	}
}

// CalculateRoute calculates a complete trading route with travel time and profit
func (ro *RouteOptimizer) CalculateRoute(ctx context.Context, item models.ItemPair, cargoCapacity float64) (models.TradingRoute, error) {
	var route models.TradingRoute

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
	profitPerTour := profitPerUnit * float64(quantityPerTour)
	totalProfit := profitPerUnit * float64(totalQuantity)

	// Calculate travel time
	travelResult, err := navigation.ShortestPath(ro.sdeDB, item.BuySystemID, item.SellSystemID, false)
	if err != nil {
		return route, fmt.Errorf("failed to calculate route: %w", err)
	}

	// Get ship type for navigation calculations (use default ship type ID 0 for generic hauler)
	// TODO: In future, get actual ship type from request or character data
	ship := models.GetShipType(0) // Default hauler

	// Use RoutePlanner for accurate travel time calculation with navigation skills
	// For now, use default skills (0/0) - future enhancement: fetch character skills
	planner := &RoutePlanner{sdeDB: ro.sdeDB, sdeQuerier: ro.sdeRepo}
	
	// Calculate base travel time (without skills)
	baseOneWaySeconds := planner.CalculateJumpTime(travelResult.Jumps, ship.BaseWarpSpeed, ship.BaseAlignTime, 0, 0)
	
	// Calculate skilled travel time (with default skills for now)
	// TODO: Pass actual character skills when available from auth context
	navigationLevel := 0
	evasiveLevel := 0
	skilledOneWaySeconds := planner.CalculateJumpTime(travelResult.Jumps, ship.BaseWarpSpeed, ship.BaseAlignTime, navigationLevel, evasiveLevel)

	// Station Trading: Use minimum time for order cycling (5 minutes base time)
	// This prevents division by zero and provides realistic ISK/h for station trading
	if item.BuySystemID == item.SellSystemID || travelResult.Jumps == 0 {
		baseOneWaySeconds = 300.0    // 5 minutes for station trading order updates
		skilledOneWaySeconds = 300.0 // Same for station trading (no travel)
	}

	// Use skilled time for main calculations
	oneWaySeconds := skilledOneWaySeconds
	roundTripSeconds := oneWaySeconds * 2
	baseRoundTripSeconds := baseOneWaySeconds * 2

	// Multi-tour time calculation
	// (numberOfTours - 1) full roundtrips + 1 one-way trip
	var totalTimeSeconds float64
	var baseTotalTimeSeconds float64
	if numberOfTours > 1 {
		totalTimeSeconds = float64(numberOfTours-1)*roundTripSeconds + oneWaySeconds
		baseTotalTimeSeconds = float64(numberOfTours-1)*baseRoundTripSeconds + baseOneWaySeconds
	} else {
		totalTimeSeconds = roundTripSeconds
		baseTotalTimeSeconds = baseRoundTripSeconds
	}
	totalTimeMinutes := totalTimeSeconds / 60.0

	// Calculate ISK per hour (both base and skilled)
	var iskPerHour float64
	var baseISKPerHour float64
	if totalTimeSeconds > 0 {
		iskPerHour = (totalProfit / totalTimeSeconds) * 3600
	}
	if baseTotalTimeSeconds > 0 {
		baseISKPerHour = (totalProfit / baseTotalTimeSeconds) * 3600
	}

	// Calculate improvement percentage
	var timeImprovement float64
	if baseTotalTimeSeconds > 0 && baseTotalTimeSeconds != totalTimeSeconds {
		timeImprovement = ((baseTotalTimeSeconds - totalTimeSeconds) / baseTotalTimeSeconds) * 100
	}

	// Get system and station names
	buySystemName, buyStationName := ro.getLocationNames(ctx, item.BuySystemID, item.BuyStationID)
	sellSystemName, sellStationName := ro.getLocationNames(ctx, item.SellSystemID, item.SellStationID)

	// Get security status for both systems
	buySecurityStatus := ro.getSystemSecurityStatus(ctx, item.BuySystemID)
	sellSecurityStatus := ro.getSystemSecurityStatus(ctx, item.SellSystemID)

	// Calculate minimum security status across entire route
	minRouteSecurity := ro.getMinRouteSecurityStatus(ctx, travelResult.Route)

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
		// Navigation skills fields
		BaseTravelTimeSeconds:    baseOneWaySeconds,
		SkilledTravelTimeSeconds: skilledOneWaySeconds,
		BaseISKPerHour:           baseISKPerHour,
		TimeImprovementPercent:   timeImprovement,
	}

	return route, nil
}

// Helper functions

func (ro *RouteOptimizer) getLocationNames(ctx context.Context, systemID, stationID int64) (string, string) {
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
func (ro *RouteOptimizer) getSystemSecurityStatus(ctx context.Context, systemID int64) float64 {
	secStatus, err := ro.sdeRepo.GetSystemSecurityStatus(ctx, systemID)
	if err != nil {
		log.Printf("Warning: failed to get security status for system %d: %v", systemID, err)
		return 1.0 // Default to high-sec if lookup fails
	}
	return secStatus
}

// getMinRouteSecurityStatus finds the minimum security status across all systems in a route
func (ro *RouteOptimizer) getMinRouteSecurityStatus(ctx context.Context, route []int64) float64 {
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
