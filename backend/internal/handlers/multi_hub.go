package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
)

// MultiHubHandler serves the Multi-Hub Comparison endpoint (Issue #43).
type MultiHubHandler struct {
	service *services.MultiHubComparisonService
}

// NewMultiHubHandler creates a new multi-hub handler.
func NewMultiHubHandler(service *services.MultiHubComparisonService) *MultiHubHandler {
	return &MultiHubHandler{service: service}
}

// CompareHubs handles POST /api/v1/trading/hubs/compare
//
// @Summary Compare an item's station-trading profitability across major hubs
// @Description For a given item, compares buy/sell/spread, skill-adjusted net margin,
// @Description volume and competition across Jita, Amarr, Dodixie, Rens, Hek.
// @Tags Trading
// @Accept json
// @Produce json
// @Param request body models.HubComparisonRequest true "Item to compare"
// @Success 200 {object} models.HubComparisonResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/hubs/compare [post]
func (h *MultiHubHandler) CompareHubs(c *fiber.Ctx) error {
	var req models.HubComparisonRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.TypeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid type_id"})
	}

	cid, tok, ok := evesso.AuthFromContext(c)
	if !ok {
		return errUnauthorized(c)
	}

	result, err := h.service.CompareHubs(c.UserContext(), req.TypeID, cid, tok)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to compare hubs"})
	}
	return c.JSON(result)
}
