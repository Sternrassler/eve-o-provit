package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
)

// AssetsHandler serves the sell-from-assets endpoints (Issue #107).
type AssetsHandler struct {
	service *services.AssetSaleService
}

// NewAssetsHandler creates a new assets handler.
func NewAssetsHandler(service *services.AssetSaleService) *AssetsHandler {
	return &AssetsHandler{service: service}
}

// ListAssets handles GET /api/v1/trading/assets
//
// @Summary List owned items aggregated by type and location
// @Description Returns the character's assets aggregated by (type, location) with quantity.
// @Tags Trading
// @Produce json
// @Success 200 {object} models.AssetsResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/assets [get]
func (h *AssetsHandler) ListAssets(c *fiber.Ctx) error {
	cid, tok, ok := evesso.AuthFromContext(c)
	if !ok {
		return errUnauthorized(c)
	}
	res, err := h.service.ListAssets(c.UserContext(), cid, tok)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to list assets"})
	}
	return c.JSON(res)
}

// SellOptions handles POST /api/v1/trading/assets/sell-options
//
// @Summary Rank taker sell locations for one owned item
// @Description For a selected owned item, ranks instant-sell (taker) locations across the
// @Description major hubs + the item's current region by net proceeds, with route info.
// @Tags Trading
// @Accept json
// @Produce json
// @Param request body models.SellOptionsRequest true "Selected item"
// @Success 200 {object} models.SellOptionsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/assets/sell-options [post]
func (h *AssetsHandler) SellOptions(c *fiber.Ctx) error {
	var req models.SellOptionsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.TypeID <= 0 || req.Quantity <= 0 || req.LocationID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "type_id, location_id and quantity are required"})
	}
	cid, tok, ok := evesso.AuthFromContext(c)
	if !ok {
		return errUnauthorized(c)
	}
	res, err := h.service.SellOptions(c.UserContext(), &req, cid, tok)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to compute sell options"})
	}
	return c.JSON(res)
}
