// Package services provides business logic for trading operations
package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/metrics"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/redis/go-redis/v9"
)

const (
	// MinSpreadPercent is the minimum spread percentage to consider profitable
	MinSpreadPercent = 5.0
	// MaxRoutes is the maximum number of routes to return
	MaxRoutes = 50
)

// Config holds route service configuration
type Config struct {
	// CalculationTimeout is the total timeout for route calculation (default: 120s)
	CalculationTimeout time.Duration
	// MarketFetchTimeout is the timeout for market order fetching (default: 60s)
	MarketFetchTimeout time.Duration
	// RouteCalculationTimeout is the timeout for route calculation phase (default: 90s)
	RouteCalculationTimeout time.Duration
}

// DefaultConfig returns default configuration values
func DefaultConfig() Config {
	return Config{
		CalculationTimeout:      120 * time.Second,
		MarketFetchTimeout:      60 * time.Second,
		RouteCalculationTimeout: 90 * time.Second,
	}
}

// Context keys for character information (must match handler keys)
const (
	contextKeyCharacterID = "character_id"
	contextKeyAccessToken = "access_token"
)

// RouteService orchestrates route calculation workflow
type RouteService struct {
	esiClient      *esi.Client
	sdeRepo        *database.SDERepository
	sdeDB          *sql.DB
	routeFinder    *RouteFinder
	routeOptimizer *RouteCalculator
	workerPool     *RouteWorkerPool
	redisClient    *redis.Client
	cargoService   CargoServicer  // For skill-aware cargo calculations
	skillsService  SkillsServicer // For fetching character skills
	feeService     FeeServicer    // For fee calculations
	volumeService  VolumeServicer // For volume metrics and liquidity analysis
	config         Config         // Timeouts and configuration
}

// NewRouteService creates a new route service instance
func NewRouteService(
	esiClient *esi.Client,
	sdeDB *sql.DB,
	sdeRepo *database.SDERepository,
	marketRepo *database.MarketRepository,
	redisClient *redis.Client,
	cargoService CargoServicer,
	skillsService SkillsServicer,
	feeService FeeServicer,
	config Config,
) *RouteService {
	rs := &RouteService{
		esiClient:     esiClient,
		sdeRepo:       sdeRepo,
		sdeDB:         sdeDB,
		redisClient:   redisClient,
		cargoService:  cargoService,
		skillsService: skillsService,
		feeService:    feeService,
		config:        config,
	}

	rs.routeFinder = NewRouteFinder(esiClient, marketRepo, sdeRepo, sdeDB, redisClient)
	rs.routeOptimizer = NewRouteCalculator(sdeRepo, sdeDB, feeService)
	rs.volumeService = NewVolumeService(marketRepo, esiClient)

	// Initialize worker pool
	rs.workerPool = NewRouteWorkerPool(rs.routeOptimizer)

	return rs
}

// Compile-time interface compliance check
var _ RouteCalculatorServicer = (*RouteService)(nil)

// Calculate computes profitable trading routes for a region with timeout support
// If cargoCapacity is provided in the request, it's used directly
// Otherwise, ship capacity is fetched from SDE and skills are applied if available in context
func (rs *RouteService) Calculate(ctx context.Context, regionID, shipTypeID int, cargoCapacity float64) (*models.RouteCalculationResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.TradingCalculationDuration.Observe(duration)
		log.Printf("Route calculation completed in %.2fs", duration)
	}()

	// Create context with timeout
	calcCtx, cancel := context.WithTimeout(ctx, rs.config.CalculationTimeout)
	defer cancel()

	// Variables to track capacity calculation
	var baseCapacity float64
	var effectiveCapacity float64
	var skillBonusPercent float64
	var fittingBonusM3 float64

	// Get ship info if cargo capacity not provided
	if cargoCapacity == 0 {
		shipCap, err := cargo.GetShipCapacities(rs.sdeDB, int64(shipTypeID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get ship capacities: %w", err)
		}
		baseCapacity = shipCap.BaseCargoHold

		// Apply character skills and fitting (required - no fallback)
		effectiveCapacity, skillBonusPercent, fittingBonusM3 = rs.applyCharacterSkills(calcCtx, baseCapacity, shipTypeID)

		cargoCapacity = effectiveCapacity
	} else {
		// Capacity was provided explicitly - use as both base and effective
		baseCapacity = cargoCapacity
		effectiveCapacity = cargoCapacity
		skillBonusPercent = 0
		fittingBonusM3 = 0
	}

	// Get ship name
	shipInfo, err := rs.sdeRepo.GetTypeInfo(calcCtx, shipTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ship info: %w", err)
	}

	// Get region name
	regionName, err := rs.getRegionName(calcCtx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get region name: %w", err)
	}

	// Find profitable items with timeout
	marketCtx, marketCancel := context.WithTimeout(calcCtx, rs.config.MarketFetchTimeout)
	defer marketCancel()

	profitableItems, err := rs.routeFinder.FindProfitableItems(marketCtx, regionID, cargoCapacity)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Market order fetch timeout after %v", rs.config.MarketFetchTimeout)
			return nil, err
		}
		return nil, fmt.Errorf("failed to find profitable items: %w", err)
	}
	log.Printf("Found %d profitable items", len(profitableItems))

	// Calculate routes using worker pool with timeout
	routeCtx, routeCancel := context.WithTimeout(calcCtx, rs.config.RouteCalculationTimeout)
	defer routeCancel()

	routes, err := rs.workerPool.ProcessItemsWithCapacityInfo(routeCtx, profitableItems, effectiveCapacity, baseCapacity, skillBonusPercent, fittingBonusM3)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("failed to calculate routes: %w", err)
	}

	// Check if we timed out
	timedOut := errors.Is(routeCtx.Err(), context.DeadlineExceeded) || errors.Is(calcCtx.Err(), context.DeadlineExceeded)

	// Filter out routes with negative net profit (unprofitable after fees)
	profitableRoutes := make([]models.TradingRoute, 0, len(routes))
	for _, route := range routes {
		if route.NetProfit > 0 {
			profitableRoutes = append(profitableRoutes, route)
		}
	}
	routes = profitableRoutes

	// Sort by ISK per hour (descending)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].ISKPerHour > routes[j].ISKPerHour
	})

	// Limit to top 50
	if len(routes) > MaxRoutes {
		routes = routes[:MaxRoutes]
	}

	calculationTime := time.Since(startTime).Milliseconds()

	response := &models.RouteCalculationResponse{
		RegionID:          regionID,
		RegionName:        regionName,
		ShipTypeID:        shipTypeID,
		ShipName:          shipInfo.Name,
		CargoCapacity:     cargoCapacity,
		CalculationTimeMS: calculationTime,
		Routes:            routes,
	}

	// Add timeout warning if applicable
	if timedOut {
		response.Warning = fmt.Sprintf("Calculation timeout after %v, showing partial results", rs.config.CalculationTimeout)
		log.Printf("WARNING: %s", response.Warning)
	}

	return response, nil
}

// CalculateWithFilters computes profitable trading routes with volume filtering support
func (rs *RouteService) CalculateWithFilters(ctx context.Context, req *models.RouteCalculationRequest) (*models.RouteCalculationResponse, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.TradingCalculationDuration.Observe(duration)
		log.Printf("Route calculation with volume filters completed in %.2fs", duration)
	}()

	// Call base Calculate method to get routes
	response, err := rs.Calculate(ctx, req.RegionID, req.ShipTypeID, req.CargoCapacity)
	if err != nil {
		return nil, err
	}

	// Early return if volume metrics not requested
	if !req.IncludeVolumeMetrics {
		return response, nil
	}

	// Filter out routes with negative net profit first
	profitableRoutes := make([]models.TradingRoute, 0, len(response.Routes))
	for _, route := range response.Routes {
		if route.NetProfit > 0 {
			profitableRoutes = append(profitableRoutes, route)
		}
	}

	// Enrich routes with volume metrics and apply filters
	filteredRoutes := make([]models.TradingRoute, 0, len(profitableRoutes))

	for _, route := range profitableRoutes {
		// Get volume metrics for this item
		volumeMetrics, err := rs.volumeService.GetVolumeMetrics(ctx, route.ItemTypeID, req.RegionID)
		if err != nil {
			log.Printf("Warning: failed to get volume metrics for type %d: %v", route.ItemTypeID, err)
			// Continue without volume metrics for this route
			filteredRoutes = append(filteredRoutes, route)
			continue
		}

		// Calculate liquidation time
		liquidationDays := rs.volumeService.CalculateLiquidationTime(route.Quantity, volumeMetrics.DailyVolumeAvg)

		// Apply volume filters
		if req.MinDailyVolume > 0 && volumeMetrics.DailyVolumeAvg < req.MinDailyVolume {
			continue // Skip routes with too low volume
		}

		if req.MaxLiquidationDays > 0 && liquidationDays > req.MaxLiquidationDays {
			continue // Skip routes with too long liquidation time
		}

		// Calculate daily profit (use net profit if available, otherwise total profit)
		dailyProfit := 0.0
		if liquidationDays > 0 {
			profitToUse := route.NetProfit
			if profitToUse == 0 {
				profitToUse = route.TotalProfit
			}
			dailyProfit = profitToUse / liquidationDays
		}

		// Enrich route with volume metrics
		route.VolumeMetrics = volumeMetrics
		route.LiquidationDays = liquidationDays
		route.DailyProfit = dailyProfit

		filteredRoutes = append(filteredRoutes, route)
	}

	// Sort by daily profit if volume metrics are included
	if len(filteredRoutes) > 0 && req.IncludeVolumeMetrics {
		sort.Slice(filteredRoutes, func(i, j int) bool {
			return filteredRoutes[i].DailyProfit > filteredRoutes[j].DailyProfit
		})
	}

	// Update response with filtered routes
	response.Routes = filteredRoutes

	return response, nil
}

// Helper functions

func (rs *RouteService) getRegionName(ctx context.Context, regionID int) (string, error) {
	return rs.sdeRepo.GetRegionName(ctx, regionID)
}

// applyCharacterSkills extracts character context and applies skills to cargo capacity
// Returns (effectiveCapacity, skillBonusPercent, fittingBonusM3)
// Requires character authentication in context
func (rs *RouteService) applyCharacterSkills(ctx context.Context, baseCapacity float64, shipTypeID int) (float64, float64, float64) {
	// Extract character_id (required - no fallback)
	characterID := ctx.Value(contextKeyCharacterID)
	accessToken := ctx.Value(contextKeyAccessToken)

	if characterID == nil || accessToken == nil {
		// This should never happen if AuthMiddleware is properly configured
		log.Printf("ERROR: Missing character context in applyCharacterSkills")
		return baseCapacity, 0.0, 0.0
	}

	charID, ok1 := characterID.(int)
	token, ok2 := accessToken.(string)

	if !ok1 || !ok2 || charID <= 0 || token == "" {
		log.Printf("ERROR: Invalid character context types")
		return baseCapacity, 0.0, 0.0
	}

	// Get deterministic cargo capacity with breakdown
	// CargoService internally uses FittingService for deterministic calculation
	totalCapacity, err := rs.cargoService.GetEffectiveCargoCapacity(ctx, charID, shipTypeID, baseCapacity, token)
	if err != nil {
		log.Printf("ERROR: Failed to get effective cargo capacity: %v", err)
		return baseCapacity, 0.0, 0.0
	}

	// Note: Detailed breakdown (skills vs modules) is available via FittingService.GetShipFitting
	// For now, we return total capacity only
	// TODO: Add breakdown if needed for display

	log.Printf("Applied cargo capacity: base=%.2f, total=%.2f m³",
		baseCapacity, totalCapacity)

	return totalCapacity, 0.0, 0.0
}
