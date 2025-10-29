// Package handlers provides HTTP request handlers
package handlers

import (
	"strconv"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/gofiber/fiber/v2"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	db         *database.DB
	sdeRepo    *database.SDERepository
	marketRepo *database.MarketRepository
	esiClient  *esi.Client
}

// New creates a new handler instance
func New(db *database.DB, sdeRepo *database.SDERepository, marketRepo *database.MarketRepository, esiClient *esi.Client) *Handler {
	return &Handler{
		db:         db,
		sdeRepo:    sdeRepo,
		marketRepo: marketRepo,
		esiClient:  esiClient,
	}
}

// Health handles health check requests
func (h *Handler) Health(c *fiber.Ctx) error {
	// Check database health
	if err := h.db.Health(c.Context()); err != nil {
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

	typeInfo, err := h.sdeRepo.GetTypeInfo(c.Context(), typeID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(typeInfo)
}

// GetMarketOrders handles market orders requests
func (h *Handler) GetMarketOrders(c *fiber.Ctx) error {
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
		// Fetch fresh data from ESI
		if err := h.esiClient.FetchMarketOrders(c.Context(), regionID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "failed to fetch market data",
			})
		}
	}

	// Get orders from database
	orders, err := h.esiClient.GetMarketOrders(c.Context(), regionID, typeID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"region_id": regionID,
		"type_id":   typeID,
		"orders":    orders,
		"count":     len(orders),
	})
}
