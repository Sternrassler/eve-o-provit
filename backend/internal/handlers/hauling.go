package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
)

// HaulingHandler serves the neighborhood hauling-routes endpoint (Issue #45).
type HaulingHandler struct {
	service *services.HaulingService
}

// NewHaulingHandler creates a new hauling handler.
func NewHaulingHandler(service *services.HaulingService) *HaulingHandler {
	return &HaulingHandler{service: service}
}

// FindRoutes handles POST /api/v1/trading/hauling/routes
//
// @Summary Find profitable station→station hauling routes in the neighborhood
// @Description From the character's current region + adjacent regions, finds cross-region
// @Description arbitrage routes with an optimal per-trip cargo load (cargo + capital).
// @Tags Trading
// @Accept json
// @Produce json
// @Param request body models.HaulingRequest true "Hauling parameters"
// @Success 200 {object} models.HaulingResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/hauling/routes [post]
func (h *HaulingHandler) FindRoutes(c *fiber.Ctx) error {
	var req models.HaulingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.ShipTypeID <= 0 || req.Capital <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "ship_type_id and capital are required"})
	}

	cid, tok, ok := evesso.AuthFromContext(c)
	if !ok {
		return errUnauthorized(c)
	}

	result, err := h.service.FindRoutes(c.UserContext(), &req, cid, tok)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to find hauling routes"})
	}
	return c.JSON(result)
}
