package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
)

// PortfolioHandler serves the ROI Calculator / capital-allocation endpoint (Issue #44).
type PortfolioHandler struct {
	service *services.PortfolioService
}

// NewPortfolioHandler creates a new portfolio handler.
func NewPortfolioHandler(service *services.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{service: service}
}

// Optimize handles POST /api/v1/trading/portfolio/optimize
//
// @Summary Optimize capital allocation across items for max daily profit
// @Description Given region, ship, capital and time budget, suggests how to allocate
// @Description capital across items (greedy, under liquidity + per-item caps).
// @Tags Trading
// @Accept json
// @Produce json
// @Param request body models.PortfolioRequest true "Optimization parameters"
// @Success 200 {object} models.PortfolioResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/portfolio/optimize [post]
func (h *PortfolioHandler) Optimize(c *fiber.Ctx) error {
	var req models.PortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.RegionID <= 0 || req.ShipTypeID <= 0 || req.Capital <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "region_id, ship_type_id and capital are required"})
	}

	characterID := c.Locals("character_id")
	accessToken := c.Locals("access_token")
	if characterID == nil || accessToken == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authentication required"})
	}
	cid, ok := characterID.(int)
	tok, ok2 := accessToken.(string)
	if !ok || !ok2 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authentication context"})
	}

	result, err := h.service.Optimize(c.UserContext(), &req, cid, tok)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to optimize portfolio"})
	}
	return c.JSON(result)
}
