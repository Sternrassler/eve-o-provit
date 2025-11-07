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
	// CalculationTimeout is the total timeout for route calculation
	CalculationTimeout = 30 * time.Second
	// MarketFetchTimeout is the timeout for market order fetching
	MarketFetchTimeout = 15 * time.Second
	// RouteCalculationTimeout is the timeout for route calculation phase
	RouteCalculationTimeout = 25 * time.Second
)

// Context keys for character information (must match handler keys)
type contextKey string

const (
	contextKeyCharacterID contextKey = "character_id"
	contextKeyAccessToken contextKey = "access_token"
)

// RouteService orchestrates route calculation workflow
type RouteService struct {
	esiClient      *esi.Client
	sdeRepo        *database.SDERepository
	sdeDB          *sql.DB
	routeFinder    *RouteFinder
	routeOptimizer *RouteOptimizer
	workerPool     *RouteWorkerPool
	redisClient    *redis.Client
	cargoService   CargoServicer  // For skill-aware cargo calculations
	skillsService  SkillsServicer // For fetching character skills
	feeService     FeeServicer    // For fee calculations
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
) *RouteService {
	rs := &RouteService{
		esiClient:     esiClient,
		sdeRepo:       sdeRepo,
		sdeDB:         sdeDB,
		redisClient:   redisClient,
		cargoService:  cargoService,
		skillsService: skillsService,
		feeService:    feeService,
	}

	// Initialize sub-services
	rs.routeFinder = NewRouteFinder(esiClient, marketRepo, sdeRepo, sdeDB, redisClient)
	rs.routeOptimizer = NewRouteOptimizer(sdeRepo, sdeDB, feeService)

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
	calcCtx, cancel := context.WithTimeout(ctx, CalculationTimeout)
	defer cancel()

	// Variables to track capacity calculation
	var baseCapacity float64
	var effectiveCapacity float64
	var skillBonusPercent float64

	// Get ship info if cargo capacity not provided
	if cargoCapacity == 0 {
		shipCap, err := cargo.GetShipCapacities(rs.sdeDB, int64(shipTypeID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get ship capacities: %w", err)
		}
		baseCapacity = shipCap.BaseCargoHold
		effectiveCapacity = baseCapacity // Default: no skills

		// Apply character skills if available in context
		effectiveCapacity, skillBonusPercent = rs.applyCharacterSkills(calcCtx, baseCapacity)

		cargoCapacity = effectiveCapacity
	} else {
		// Capacity was provided explicitly - use as both base and effective
		baseCapacity = cargoCapacity
		effectiveCapacity = cargoCapacity
		skillBonusPercent = 0
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
	marketCtx, marketCancel := context.WithTimeout(calcCtx, MarketFetchTimeout)
	defer marketCancel()

	profitableItems, err := rs.routeFinder.FindProfitableItems(marketCtx, regionID, cargoCapacity)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Market order fetch timeout after %v", MarketFetchTimeout)
			return nil, err
		}
		return nil, fmt.Errorf("failed to find profitable items: %w", err)
	}
	log.Printf("Found %d profitable items", len(profitableItems))

	// Calculate routes using worker pool with timeout
	routeCtx, routeCancel := context.WithTimeout(calcCtx, RouteCalculationTimeout)
	defer routeCancel()

	routes, err := rs.workerPool.ProcessItemsWithCapacityInfo(routeCtx, profitableItems, effectiveCapacity, baseCapacity, skillBonusPercent)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return nil, fmt.Errorf("failed to calculate routes: %w", err)
	}

	// Check if we timed out
	timedOut := errors.Is(routeCtx.Err(), context.DeadlineExceeded) || errors.Is(calcCtx.Err(), context.DeadlineExceeded)

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
		response.Warning = fmt.Sprintf("Calculation timeout after %v, showing partial results", CalculationTimeout)
		log.Printf("WARNING: %s", response.Warning)
	}

	return response, nil
}

// Helper functions

func (rs *RouteService) getRegionName(ctx context.Context, regionID int) (string, error) {
	return rs.sdeRepo.GetRegionName(ctx, regionID)
}

// applyCharacterSkills extracts character context and applies skills to cargo capacity
// Returns (effectiveCapacity, skillBonusPercent)
// Falls back to base capacity if skills unavailable
func (rs *RouteService) applyCharacterSkills(ctx context.Context, baseCapacity float64) (float64, float64) {
	effectiveCapacity := baseCapacity
	skillBonusPercent := 0.0

	// Extract character_id if available in context metadata
	characterID := ctx.Value(contextKeyCharacterID)
	accessToken := ctx.Value(contextKeyAccessToken)

	if characterID != nil && accessToken != nil {
		charID, ok1 := characterID.(int)
		token, ok2 := accessToken.(string)

		if ok1 && ok2 && charID > 0 && token != "" {
			// Fetch skills and apply to capacity
			if skills, err := rs.skillsService.GetCharacterSkills(ctx, charID, token); err == nil {
				effectiveCapacity, skillBonusPercent = rs.cargoService.CalculateCargoCapacity(baseCapacity, skills)
				log.Printf("Applied cargo skills: base=%.2f, effective=%.2f, bonus=%.2f%%",
					baseCapacity, effectiveCapacity, skillBonusPercent)
			}
		}
	}

	return effectiveCapacity, skillBonusPercent
}
