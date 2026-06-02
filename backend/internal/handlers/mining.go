package handlers

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
)

// MiningHandler serves the mining ore-ranking endpoint (raw-vs-refine per region+sec-band).
type MiningHandler struct {
	service *services.MiningService
}

// NewMiningHandler creates a new mining handler.
func NewMiningHandler(service *services.MiningService) *MiningHandler {
	return &MiningHandler{service: service}
}

// OreRanking handles POST /api/v1/mining/ore-ranking
//
// @Summary Rank asteroid ores by raw-vs-refine net ISK for a region + security band
// @Description Given a region (or the character's current region) and a security band,
// @Description ranks ores by net ISK per m³ and — when a mining ship is fitted — per hour,
// @Description comparing selling the raw ore vs reprocessing + selling the materials.
// @Tags Mining
// @Accept json
// @Produce json
// @Param request body models.OreRankingRequest true "Ore ranking parameters"
// @Success 200 {object} models.OreRankingResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/mining/ore-ranking [post]
func (h *MiningHandler) OreRanking(c *fiber.Ctx) error {
	var req models.OreRankingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	cid, tok, ok := evesso.AuthFromContext(c)
	if !ok {
		return errUnauthorized(c)
	}

	result, err := h.service.OreRanking(c.UserContext(), cid, tok, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to rank ores"})
	}
	return c.JSON(result)
}
