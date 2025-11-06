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

	// Calculate travel time in seconds
	oneWaySeconds := float64(travelResult.Jumps) * JumpTimeSeconds
	roundTripSeconds := oneWaySeconds * 2

	// Multi-tour time: (numberOfTours - 1) full roundtrips + 1 one-way trip
	var totalTimeSeconds float64
	if numberOfTours > 1 {
		totalTimeSeconds = float64(numberOfTours-1)*roundTripSeconds + oneWaySeconds
	} else {
		totalTimeSeconds = roundTripSeconds
	}
	totalTimeMinutes := totalTimeSeconds / 60.0

	// Calculate ISK per hour
	var iskPerHour float64
	if totalTimeSeconds > 0 {
		iskPerHour = (totalProfit / totalTimeSeconds) * 3600
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
		TravelTimeSeconds:      oneWaySeconds,
		RoundTripSeconds:       roundTripSeconds,
		ISKPerHour:             iskPerHour,
		SpreadPercent:          item.SpreadPercent,
	}

	return route, nil
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
		return 1.0 // Default to high-sec on error
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
