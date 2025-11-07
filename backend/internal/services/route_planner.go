// Package services - Route planning and navigation
package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
	"github.com/redis/go-redis/v9"
)

const (
	// JumpTimeSeconds is average time per jump
	JumpTimeSeconds = 30.0
	// MaxTours is maximum number of round trips to calculate
	MaxTours = 10
)

// RoutePlanner handles navigation and route calculation
type RoutePlanner struct {
	sdeDB      *sql.DB
	sdeQuerier database.SDEQuerier
	navCache   *NavigationCache
}

// NewRoutePlanner creates a new route planner
func NewRoutePlanner(sdeDB *sql.DB, sdeQuerier database.SDEQuerier, redisClient *redis.Client) *RoutePlanner {
	rp := &RoutePlanner{
		sdeDB:      sdeDB,
		sdeQuerier: sdeQuerier,
	}

	// Initialize cache if Redis available
	if redisClient != nil {
		rp.navCache = NewNavigationCache(redisClient)
	}

	return rp
}

// CalculateRoute builds a complete trading route with navigation and profit details
func (rp *RoutePlanner) CalculateRoute(ctx context.Context, item models.ItemPair, cargoCapacity float64, numberOfTours int, quantityPerTour int) (models.TradingRoute, error) {
	return rp.CalculateRouteWithSkills(ctx, item, cargoCapacity, numberOfTours, quantityPerTour, nil, 0)
}

// CalculateRouteWithSkills builds a complete trading route with navigation skills applied
func (rp *RoutePlanner) CalculateRouteWithSkills(ctx context.Context, item models.ItemPair, cargoCapacity float64, numberOfTours int, quantityPerTour int, skills *TradingSkills, shipTypeID int) (models.TradingRoute, error) {
	var route models.TradingRoute

	// Validate inputs
	if quantityPerTour <= 0 {
		return route, fmt.Errorf("invalid quantity per tour: %d", quantityPerTour)
	}

	// Calculate total quantity
	totalQuantity := item.AvailableQuantity
	if totalQuantity > quantityPerTour*numberOfTours {
		totalQuantity = quantityPerTour * numberOfTours
	}

	// Calculate profit
	profitPerUnit := item.SellPrice - item.BuyPrice
	profitPerTour := profitPerUnit * float64(quantityPerTour)
	totalProfit := profitPerUnit * float64(totalQuantity)

	// Calculate travel time
	travelResult, err := navigation.ShortestPath(rp.sdeDB, item.BuySystemID, item.SellSystemID, false)
	if err != nil {
		return route, fmt.Errorf("failed to calculate route: %w", err)
	}

	// Get ship type for navigation calculations
	ship := models.GetShipType(shipTypeID)

	// Calculate base travel time (without skills)
	baseTravelTime := rp.CalculateJumpTime(travelResult.Jumps, ship.BaseWarpSpeed, ship.BaseAlignTime, 0, 0)
	oneWaySecondsBase := baseTravelTime
	roundTripSecondsBase := oneWaySecondsBase * 2

	// Calculate skilled travel time (with navigation skills)
	navigationLevel := 0
	evasiveLevel := 0
	if skills != nil {
		navigationLevel = skills.Navigation
		evasiveLevel = skills.EvasiveManeuvering
	}
	skilledTravelTime := rp.CalculateJumpTime(travelResult.Jumps, ship.BaseWarpSpeed, ship.BaseAlignTime, navigationLevel, evasiveLevel)
	oneWaySecondsSkilled := skilledTravelTime
	roundTripSecondsSkilled := oneWaySecondsSkilled * 2

	// Multi-tour time: (numberOfTours - 1) full roundtrips + 1 one-way trip
	var totalTimeSecondsBase float64
	var totalTimeSecondsSkilled float64
	if numberOfTours > 1 {
		totalTimeSecondsBase = float64(numberOfTours-1)*roundTripSecondsBase + oneWaySecondsBase
		totalTimeSecondsSkilled = float64(numberOfTours-1)*roundTripSecondsSkilled + oneWaySecondsSkilled
	} else {
		totalTimeSecondsBase = roundTripSecondsBase
		totalTimeSecondsSkilled = roundTripSecondsSkilled
	}
	totalTimeMinutes := totalTimeSecondsSkilled / 60.0

	// Calculate ISK per hour (both base and skilled)
	var iskPerHourBase float64
	var iskPerHourSkilled float64
	if totalTimeSecondsBase > 0 {
		iskPerHourBase = (totalProfit / totalTimeSecondsBase) * 3600
	}
	if totalTimeSecondsSkilled > 0 {
		iskPerHourSkilled = (totalProfit / totalTimeSecondsSkilled) * 3600
	}

	// Calculate improvement percentage
	var timeImprovement float64
	if totalTimeSecondsBase > 0 {
		timeImprovement = ((totalTimeSecondsBase - totalTimeSecondsSkilled) / totalTimeSecondsBase) * 100
	}

	// Get location names
	buySystemName, buyStationName := rp.getLocationNames(ctx, item.BuySystemID, item.BuyStationID)
	sellSystemName, sellStationName := rp.getLocationNames(ctx, item.SellSystemID, item.SellStationID)

	// Get security status
	buySecurityStatus := rp.getSystemSecurityStatus(ctx, item.BuySystemID)
	sellSecurityStatus := rp.getSystemSecurityStatus(ctx, item.SellSystemID)
	minRouteSecurity := rp.getMinRouteSecurityStatus(ctx, travelResult.Route)

	// Build route
	route = models.TradingRoute{
		ItemTypeID:             item.TypeID,
		ItemName:               item.ItemName,
		ItemVolume:             item.ItemVolume,
		BuyStationID:           item.BuyStationID,
		BuyStationName:         buyStationName,
		BuySystemID:            item.BuySystemID,
		BuySystemName:          buySystemName,
		BuyPrice:               item.BuyPrice,
		BuySecurityStatus:      buySecurityStatus,
		SellStationID:          item.SellStationID,
		SellStationName:        sellStationName,
		SellSystemID:           item.SellSystemID,
		SellSystemName:         sellSystemName,
		SellPrice:              item.SellPrice,
		SellSecurityStatus:     sellSecurityStatus,
		Jumps:                  travelResult.Jumps,
		MinRouteSecurityStatus: minRouteSecurity,
		Quantity:               quantityPerTour,
		NumberOfTours:          numberOfTours,
		ProfitPerUnit:          item.SellPrice - item.BuyPrice,
		ProfitPerTour:          profitPerTour,
		TotalProfit:            totalProfit,
		TotalTimeMinutes:       totalTimeMinutes,
		TravelTimeSeconds:      oneWaySecondsSkilled,
		RoundTripSeconds:       roundTripSecondsSkilled,
		ISKPerHour:             iskPerHourSkilled,
		SpreadPercent:          item.SpreadPercent,
		// Navigation skills fields
		BaseTravelTimeSeconds:    oneWaySecondsBase,
		SkilledTravelTimeSeconds: oneWaySecondsSkilled,
		BaseISKPerHour:           iskPerHourBase,
		TimeImprovementPercent:   timeImprovement,
	}

	return route, nil
}

// CalculateJumpTime calculates total travel time for jumps with navigation skills
// baseWarpSpeed: AU/s (e.g., 3.0 for haulers)
// baseAlignTime: seconds (e.g., 8.0 for haulers)
// navigationLevel: 0-5 (+5% warp speed per level)
// evasiveLevel: 0-5 (-5% align time per level)
// Exported for testing purposes
func (rp *RoutePlanner) CalculateJumpTime(jumps int, baseWarpSpeed, baseAlignTime float64, navigationLevel, evasiveLevel int) float64 {
	if jumps == 0 {
		return 0
	}

	// Apply Navigation skill bonus (+5% warp speed per level)
	warpSpeed := baseWarpSpeed * (1.0 + 0.05*float64(navigationLevel))

	// Apply Evasive Maneuvering skill bonus (-5% align time per level)
	alignTime := baseAlignTime * (1.0 - 0.05*float64(evasiveLevel))

	// Average distance per jump (AU) - simplified model
	// In reality, distances vary, but we use a constant for simplicity
	const avgDistanceAU = 9.0 // Average distance between gates

	// Calculate time per jump
	warpTime := avgDistanceAU / warpSpeed
	dockingTime := 10.0 // Time for undocking/docking/gate activation

	timePerJump := alignTime + warpTime + dockingTime

	return float64(jumps) * timePerJump
}

// GetSystemIDFromLocation resolves a location ID to its system ID
func (rp *RoutePlanner) GetSystemIDFromLocation(ctx context.Context, locationID int64) int64 {
	systemID, err := rp.sdeQuerier.GetSystemIDForLocation(ctx, locationID)
	if err != nil {
		return locationID // Fallback: assume it's already a system ID
	}
	return systemID
}

// getLocationNames retrieves system and station names
func (rp *RoutePlanner) getLocationNames(ctx context.Context, systemID, stationID int64) (string, string) {
	systemName, err := rp.sdeQuerier.GetSystemName(ctx, systemID)
	if err != nil {
		systemName = fmt.Sprintf("System-%d", systemID)
	}

	stationName, err := rp.sdeQuerier.GetStationName(ctx, stationID)
	if err != nil {
		stationName = fmt.Sprintf("Station-%d", stationID)
	}

	return systemName, stationName
}

// getSystemSecurityStatus retrieves security status for a system
func (rp *RoutePlanner) getSystemSecurityStatus(ctx context.Context, systemID int64) float64 {
	security, err := rp.sdeQuerier.GetSystemSecurityStatus(ctx, systemID)
	if err != nil {
		// Default to high-sec (1.0) on error for safety
		// Changed from 0.0 (nullsec) which was incorrect and could cause
		// routes to incorrectly include dangerous systems
		return 1.0
	}
	return security
}

// getMinRouteSecurityStatus finds the lowest security status along a route
func (rp *RoutePlanner) getMinRouteSecurityStatus(ctx context.Context, route []int64) float64 {
	if len(route) == 0 {
		return 0.0
	}

	minSec := 1.0
	for _, systemID := range route {
		sec := rp.getSystemSecurityStatus(ctx, systemID)
		if sec < minSec {
			minSec = sec
		}
	}
	return minSec
}

// CalculateTravelTime calculates travel time for a route
func (rp *RoutePlanner) CalculateTravelTime(jumps int, numberOfTours int) time.Duration {
	oneWaySeconds := float64(jumps) * JumpTimeSeconds
	roundTripSeconds := oneWaySeconds * 2

	var totalSeconds float64
	if numberOfTours > 1 {
		totalSeconds = float64(numberOfTours-1)*roundTripSeconds + oneWaySeconds
	} else {
		totalSeconds = roundTripSeconds
	}

	return time.Duration(totalSeconds) * time.Second
}
