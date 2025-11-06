// Package handlers provides HTTP request handlers
package handlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
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
func (h *Handler) Version(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version": "0.1.0",
		"service": "eve-o-provit-api",
	})
}

// GetType handles SDE type lookup requests
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
			MAX(fetched_at) as latest_fetch,
			EXTRACT(EPOCH FROM (NOW() - MAX(fetched_at)))/60 as age_minutes
		FROM market_orders
		WHERE region_id = $1
	`
	var totalOrders int
	var latestFetch time.Time
	var ageMinutes float64

	err = h.postgresQuery.QueryRow(c.Context(), query, regionID).Scan(&totalOrders, &latestFetch, &ageMinutes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to query market data age",
		})
	}

	return c.JSON(fiber.Map{
		"region_id":    regionID,
		"total_orders": totalOrders,
		"latest_fetch": latestFetch.Format(time.RFC3339),
		"age_minutes":  ageMinutes,
	})
}

// GetRegions handles SDE regions list requests
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
