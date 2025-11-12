// Package handlers provides HTTP request handlers
package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	_ "github.com/Sternrassler/eve-o-provit/backend/internal/models" // For OpenAPI
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/gofiber/fiber/v2"
)

// MarketServicer defines interface for market data operations (enables mocking)
type MarketServicer interface {
	FetchAndStoreMarketOrders(ctx context.Context, regionID int) (int, error)
	GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error)
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	healthChecker database.HealthChecker
	sdeQuerier    database.SDEQuerier
	marketQuerier database.MarketQuerier
	postgresQuery database.PostgresQuerier // Interface for raw Postgres queries
	regionQuerier database.RegionQuerier   // Interface for region data
	esiClient     *esi.Client
	marketService MarketServicer // Interface for testability
}

// New creates a new handler instance with interfaces
func New(healthChecker database.HealthChecker, sdeQuerier database.SDEQuerier, marketQuerier database.MarketQuerier, esiClient *esi.Client) *Handler {
	// Type assert to get interfaces from concrete types
	var postgresQuery database.PostgresQuerier
	var regionQuerier database.RegionQuerier
	if concreteDB, ok := healthChecker.(*database.DB); ok {
		postgresQuery = concreteDB // DB implements PostgresQuerier
	}
	if sdeRepo, ok := sdeQuerier.(*database.SDERepository); ok {
		regionQuerier = sdeRepo // SDERepository implements RegionQuerier
	}

	// Create MarketService
	marketService := services.NewMarketService(marketQuerier, esiClient)

	return &Handler{
		healthChecker: healthChecker,
		sdeQuerier:    sdeQuerier,
		marketQuerier: marketQuerier,
		postgresQuery: postgresQuery,
		regionQuerier: regionQuerier,
		esiClient:     esiClient,
		marketService: marketService,
	}
}

// NewWithConcrete creates a handler from concrete types (backward compatibility wrapper)
// Deprecated: Use New with interfaces instead
func NewWithConcrete(db *database.DB, sdeRepo *database.SDERepository, marketRepo *database.MarketRepository, esiClient *esi.Client) *Handler {
	marketService := services.NewMarketService(marketRepo, esiClient)

	return &Handler{
		healthChecker: db,
		sdeQuerier:    sdeRepo,
		marketQuerier: marketRepo,
		postgresQuery: db,      // DB implements PostgresQuerier
		regionQuerier: sdeRepo, // SDERepository implements RegionQuerier
		esiClient:     esiClient,
		marketService: marketService,
	}
}

// Health handles health check requests
//
// @Summary Health check
// @Description Check API and database health status
// @Tags Health
// @Produce json
// @Success 200 {object} models.HealthResponse
// @Failure 503 {object} models.ErrorResponse
// @Router /health [get]
func (h *Handler) Health(c *fiber.Ctx) error {
	// Check database health
	if err := h.healthChecker.Health(c.Context()); err != nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "eve-o-provit-api",
		"database": fiber.Map{
			"postgres": "ok",
			"sde":      "ok",
		},
	})
}

// Version handles version requests
//
// @Summary API version
// @Description Get API version information
// @Tags Health
// @Produce json
// @Success 200 {object} models.VersionResponse
// @Router /version [get]
func (h *Handler) Version(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version": "0.1.0",
		"service": "eve-o-provit-api",
	})
}

// GetType handles SDE type lookup requests
//
// @Summary Get item type information
// @Description Retrieve detailed information about an EVE Online item type from SDE
// @Tags SDE
// @Produce json
// @Param id path int true "Type ID"
// @Success 200 {object} models.TypeResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /types/{id} [get]
func (h *Handler) GetType(c *fiber.Ctx) error {
	typeIDStr := c.Params("id")
	typeID, err := strconv.Atoi(typeIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid type ID",
		})
	}

	typeInfo, err := h.sdeQuerier.GetTypeInfo(c.Context(), typeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(typeInfo)
}

// GetMarketOrders handles market orders requests
//
// @Summary Get market orders
// @Description Retrieve market orders for a specific item type in a region
// @Description Supports optional refresh from ESI (cached for 5 minutes)
// @Tags Market
// @Produce json
// @Param region path int true "Region ID" example(10000002)
// @Param type path int true "Type ID" example(34)
// @Param refresh query bool false "Force refresh from ESI" default(false)
// @Success 200 {array} models.MarketOrderResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /market/{region}/{type} [get]
func (h *Handler) GetMarketOrders(c *fiber.Ctx) error {
	// Parameter validation
	regionIDStr := c.Params("region")
	typeIDStr := c.Params("type")

	regionID, err := strconv.Atoi(regionIDStr)
	if err != nil || regionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid region ID",
		})
	}

	typeID, err := strconv.Atoi(typeIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid type ID",
		})
	}

	// Check if we should fetch fresh data
	refresh := c.QueryBool("refresh", false)
	if refresh {
		// Delegate to MarketService for fetching and storing
		count, err := h.marketService.FetchAndStoreMarketOrders(c.Context(), regionID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to fetch and store market data",
				"details": err.Error(),
			})
		}
		// Log success
		_ = count // Stored successfully
	}

	// Get orders from database via MarketService
	orders, err := h.marketService.GetMarketOrders(c.Context(), regionID, typeID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get market orders",
			"details": err.Error(),
		})
	}

	return c.JSON(orders)
}

// GetMarketDataStaleness returns age of market data for a region
//
// @Summary Get market data staleness
// @Description Check how old the cached market data is for a region
// @Description Status: fresh (<30min), stale (30-60min), very_stale (>60min)
// @Tags Market
// @Produce json
// @Param region path int true "Region ID" example(10000002)
// @Success 200 {object} models.MarketDataStalenessResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /market/staleness/{region} [get]
func (h *Handler) GetMarketDataStaleness(c *fiber.Ctx) error {
	regionIDStr := c.Params("region")
	if regionIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing region ID",
		})
	}

	regionID, err := strconv.Atoi(regionIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid region ID",
			"details": err.Error(),
		})
	}

	query := `
		SELECT 
			COUNT(*) as total_orders,
			MAX(cached_at) as latest_fetch,
			EXTRACT(EPOCH FROM (NOW() - MAX(cached_at)))/60 as age_minutes
		FROM market_orders
		WHERE region_id = $1
	`
	var totalOrders int
	var latestFetch *time.Time // Nullable for empty regions
	var ageMinutes *float64    // Nullable for empty regions

	err = h.postgresQuery.QueryRow(c.Context(), query, regionID).Scan(&totalOrders, &latestFetch, &ageMinutes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to query market data age",
		})
	}

	response := fiber.Map{
		"region_id":    regionID,
		"total_orders": totalOrders,
	}

	if latestFetch != nil {
		response["latest_fetch"] = latestFetch.Format(time.RFC3339)
	} else {
		response["latest_fetch"] = nil
	}

	if ageMinutes != nil {
		response["age_minutes"] = *ageMinutes
	} else {
		response["age_minutes"] = nil
	}

	return c.JSON(response)
}

// GetRegions handles SDE regions list requests
//
// @Summary List all EVE regions
// @Description Get list of all EVE Online regions from SDE
// @Tags SDE
// @Produce json
// @Success 200 {array} models.RegionResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /sde/regions [get]
func (h *Handler) GetRegions(c *fiber.Ctx) error {
	if h.regionQuerier == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Region querier not initialized",
		})
	}

	regions, err := h.regionQuerier.GetAllRegions(c.Context())
	if err != nil {
		fmt.Printf("ERROR: Failed to fetch regions: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch regions",
			"details": err.Error(),
		})
	}

	// Convert RegionData to models.Region
	result := make([]models.Region, len(regions))
	for i, rd := range regions {
		result[i] = models.Region{
			ID:   rd.ID,
			Name: rd.Name,
		}
	}

	return c.JSON(models.RegionsResponse{
		Regions: result,
		Count:   len(result),
	})
}
