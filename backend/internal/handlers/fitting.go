// Package handlers - Fitting endpoints
package handlers

import (
	"strconv"

	_ "github.com/Sternrassler/eve-o-provit/backend/internal/models" // For OpenAPI
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// FittingHandler handles fitting-related HTTP requests
type FittingHandler struct {
	fittingService services.FittingServicer
}

// NewFittingHandler creates a new fitting handler instance
func NewFittingHandler(fittingService services.FittingServicer) *FittingHandler {
	return &FittingHandler{
		fittingService: fittingService,
	}
}

// GetCharacterFitting handles GET /api/v1/characters/:characterId/fitting/:shipTypeId
// Fetches and returns character's ship fitting with bonus calculations
// Returns fitting data including:
// - List of fitted modules with dogma attributes
// - Aggregated bonuses (cargo, warp speed, inertia)
// - Cache status (5min TTL)
//
// @Summary Get character ship fitting
// @Description Retrieve character's ship fitting with deterministic bonus calculations
// @Description Calculates effective cargo capacity, warp speed with skills and modules
// @Description Uses EVE dogma engine with stacking penalties
// @Tags Fitting
// @Security BearerAuth
// @Produce json
// @Param characterId path int true "Character ID" example(12345678)
// @Param shipTypeId path int true "Ship Type ID" example(650)
// @Param refresh query bool false "Force cache refresh" default(false)
// @Success 200 {object} models.CharacterFittingResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/characters/{characterId}/fitting/{shipTypeId} [get]
func (h *FittingHandler) GetCharacterFitting(c *fiber.Ctx) error {
	// Get character ID from path parameter
	characterIDParam := c.Params("characterId")
	characterID, err := strconv.Atoi(characterIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid character_id",
		})
	}

	// Get ship type ID from path parameter
	shipTypeIDParam := c.Params("shipTypeId")
	shipTypeID, err := strconv.Atoi(shipTypeIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ship_type_id",
		})
	}

	// Get access token from locals (set by AuthMiddleware)
	accessToken, ok := c.Locals("access_token").(string)
	if !ok || accessToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Missing access token",
		})
	}

	// Verify that the requested character ID matches the authenticated character
	authenticatedCharID, ok := c.Locals("character_id").(int)
	if !ok || authenticatedCharID != characterID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot access fitting for other characters",
		})
	}

	// Check if cache refresh is requested via query parameter
	refresh := c.Query("refresh") == "true"
	if refresh {
		h.fittingService.InvalidateFittingCache(c.Context(), characterID, shipTypeID)
	}

	// Fetch fitting from ESI (with caching)
	fitting, err := h.fittingService.GetShipFitting(c.Context(), characterID, shipTypeID, accessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch character fitting",
			"details": err.Error(),
		})
	}

	// Return fitting data
	return c.JSON(fiber.Map{
		"character_id":   characterID,
		"ship_type_id":   shipTypeID,
		"fitted_modules": fitting.FittedModules,
		"bonuses": fiber.Map{
			"cargo_bonus_m3":        fitting.Bonuses.CargoBonus,
			"warp_speed_multiplier": fitting.Bonuses.WarpSpeedMultiplier,
			"inertia_modifier":      fitting.Bonuses.InertiaModifier,
		},
		"cached": fitting.Cached,
	})
}
