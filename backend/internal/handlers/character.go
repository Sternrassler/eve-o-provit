// Package handlers - Character endpoints
package handlers

import (
	"strconv"

	_ "github.com/Sternrassler/eve-o-provit/backend/internal/models" // For OpenAPI
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// CharacterHandler handles character-related HTTP requests
type CharacterHandler struct {
	skillsService services.SkillsServicer
}

// NewCharacterHandler creates a new character handler instance
func NewCharacterHandler(skillsService services.SkillsServicer) *CharacterHandler {
	return &CharacterHandler{
		skillsService: skillsService,
	}
}

// GetCharacterSkills handles GET /api/v1/characters/:characterId/skills
// Fetches and returns character skills from ESI with caching
// Returns default skills (all = 0) if ESI fails (graceful degradation)
//
// @Summary Get character skills
// @Description Retrieve character skills from ESI with Redis caching
// @Description Graceful degradation: Returns default skills (level 0) if ESI fails
// @Tags Character
// @Security BearerAuth
// @Produce json
// @Param characterId path int true "Character ID" example(12345678)
// @Success 200 {object} models.CharacterSkillsResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/characters/{characterId}/skills [get]
func (h *CharacterHandler) GetCharacterSkills(c *fiber.Ctx) error {
	// Get character ID from path parameter
	characterIDParam := c.Params("characterId")
	characterID, err := strconv.Atoi(characterIDParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid character_id",
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
	// This prevents users from querying other characters' skills
	authenticatedCharID, ok := c.Locals("character_id").(int)
	if !ok || authenticatedCharID != characterID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Cannot access skills for other characters",
		})
	}

	// Fetch skills from ESI (with caching)
	skills, err := h.skillsService.GetCharacterSkills(c.Context(), characterID, accessToken)
	if err != nil {
		// SkillsService already handles graceful degradation
		// This error should only occur on critical failures
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch character skills",
			"details": err.Error(),
		})
	}

	// Return skills
	return c.JSON(fiber.Map{
		"character_id": characterID,
		"skills":       skills,
	})
}
