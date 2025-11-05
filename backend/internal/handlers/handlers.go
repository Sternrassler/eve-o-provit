// Package handlers provides HTTP request handlers
package handlers

import (
	"context"
	"encoding/json"
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
	esiClient     *esi.Client
	marketService MarketServicer // Interface for testability
	// TODO(Phase 2): Remove raw DB access, use services instead
	db *database.DB // Temporary: for GetRegions (SDE access)
}

// New creates a new handler instance with interfaces
func New(healthChecker database.HealthChecker, sdeQuerier database.SDEQuerier, marketQuerier database.MarketQuerier, esiClient *esi.Client) *Handler {
	// For Phase 1: Accept both interfaces and concrete DB
	// Type assert to get raw DB access (temporary)
	var rawDB *database.DB
	var postgresQuery database.PostgresQuerier
	if concreteDB, ok := healthChecker.(*database.DB); ok {
		rawDB = concreteDB
		postgresQuery = concreteDB // DB implements PostgresQuerier
	}

	// Create MarketService
	marketService := services.NewMarketService(marketQuerier, esiClient)

	return &Handler{
		healthChecker: healthChecker,
		sdeQuerier:    sdeQuerier,
		marketQuerier: marketQuerier,
		postgresQuery: postgresQuery,
		esiClient:     esiClient,
		marketService: marketService,
		db:            rawDB, // Temporary for Phase 1
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
		postgresQuery: db, // DB implements PostgresQuerier
		esiClient:     esiClient,
		marketService: marketService,
		db:            db,
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
	if err != nil {
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
	query := `
		SELECT _key, name
		FROM mapRegions
		ORDER BY name ASC
	`

	rows, err := h.db.SDE.QueryContext(c.Context(), query)
	if err != nil {
		fmt.Printf("ERROR: Failed to query SDE regions: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch regions",
			"details": err.Error(),
		})
	}
	defer rows.Close()

	var regions []models.Region
	for rows.Next() {
		var r models.Region
		var nameJSON string
		if err := rows.Scan(&r.ID, &nameJSON); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to parse region data",
				"details": err.Error(),
			})
		}

		// Parse JSON name and extract English version
		var names map[string]string
		if err := json.Unmarshal([]byte(nameJSON), &names); err != nil {
			// Fallback: use raw string if JSON parsing fails
			r.Name = nameJSON
		} else if enName, ok := names["en"]; ok {
			r.Name = enName
		} else {
			// Fallback: use first available name
			for _, name := range names {
				r.Name = name
				break
			}
		}

		regions = append(regions, r)
	}

	if err := rows.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Error reading regions",
			"details": err.Error(),
		})
	}

	return c.JSON(models.RegionsResponse{
		Regions: regions,
		Count:   len(regions),
	})
}
