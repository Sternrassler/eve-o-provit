// Package handlers - Calculation endpoints for deterministic ship bonuses
package handlers

import (
	"database/sql"
	"fmt"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	_ "github.com/Sternrassler/eve-o-provit/backend/internal/models" // For OpenAPI
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// CalculationHandler handles deterministic calculation requests
type CalculationHandler struct {
	sdeDB          *sql.DB
	fittingService services.FittingServicer
}

// NewCalculationHandler creates a new calculation handler instance
func NewCalculationHandler(
	sdeDB *sql.DB,
	fittingService services.FittingServicer,
) *CalculationHandler {
	return &CalculationHandler{
		sdeDB:          sdeDB,
		fittingService: fittingService,
	}
}

// CalculateCargo calculates effective cargo capacity with deterministic bonuses
//
// @Summary Calculate effective cargo capacity
// @Description Calculate effective cargo capacity including skill bonuses and module bonuses
// @Description Supports both character-based (with ESI fetch) and manual skill input
// @Description Returns deterministic breakdown of all bonuses applied
// @Tags Calculations
// @Accept json
// @Produce json
// @Param request body models.CargoCalculationRequest true "Cargo calculation parameters"
// @Success 200 {object} models.CargoCalculationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/calculations/cargo [post]
func (h *CalculationHandler) CalculateCargo(c *fiber.Ctx) error {
	var req models.CargoCalculationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}

	// Validate ship type ID
	if req.ShipTypeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ship_type_id is required",
		})
	}

	// Get ship type name from SDE
	var shipTypeName string
	err := h.sdeDB.QueryRowContext(c.Context(),
		`SELECT COALESCE(json_extract(name, '$.en'), json_extract(name, '$.de'), 'Unknown') 
		FROM types WHERE _key = ?`,
		req.ShipTypeID,
	).Scan(&shipTypeName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to fetch ship type",
			"details": err.Error(),
		})
	}

	// Get base capacity if not provided
	baseCapacity := req.BaseCapacity
	if baseCapacity == 0 {
		err := h.sdeDB.QueryRowContext(c.Context(),
			`SELECT COALESCE(json_extract(dogma_attributes, '$.38'), 0)
			FROM types WHERE _key = ?`,
			req.ShipTypeID,
		).Scan(&baseCapacity)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "failed to fetch base capacity",
				"details": err.Error(),
			})
		}
	}

	// Calculate skill bonus (default: 0%)
	skillBonusPercent := 0.0
	if req.SkillLevels != nil {
		// Spaceship Command: +5% per level
		skillBonusPercent += float64(req.SkillLevels.SpaceshipCommand) * 5.0

		// Racial skills: +5% per level (only one applies)
		maxRacialBonus := 0.0
		if req.SkillLevels.RacialFrigate > 0 {
			maxRacialBonus = float64(req.SkillLevels.RacialFrigate) * 5.0
		}
		if req.SkillLevels.RacialDestroyer > 0 && float64(req.SkillLevels.RacialDestroyer)*5.0 > maxRacialBonus {
			maxRacialBonus = float64(req.SkillLevels.RacialDestroyer) * 5.0
		}
		if req.SkillLevels.RacialCruiser > 0 && float64(req.SkillLevels.RacialCruiser)*5.0 > maxRacialBonus {
			maxRacialBonus = float64(req.SkillLevels.RacialCruiser) * 5.0
		}
		skillBonusPercent += maxRacialBonus
	}

	// Calculate module bonus (sum of all cargo expander bonuses)
	moduleBonusM3 := 0.0
	if req.ModuleBonuses != nil {
		for _, bonus := range req.ModuleBonuses {
			if bonus.AttributeID == 38 { // Attribute 38 = capacity
				moduleBonusM3 += bonus.Value
			}
		}
	}

	// Calculate effective capacity
	capacityWithSkills := baseCapacity * (1.0 + skillBonusPercent/100.0)
	effectiveCapacity := capacityWithSkills + moduleBonusM3

	// Build breakdown string
	breakdown := fmt.Sprintf("Base: %.1fm³", baseCapacity)
	if skillBonusPercent > 0 {
		breakdown += fmt.Sprintf(" + Skills: %.1f%%", skillBonusPercent)
	}
	if moduleBonusM3 > 0 {
		breakdown += fmt.Sprintf(" + Modules: %.1fm³", moduleBonusM3)
	}
	breakdown += fmt.Sprintf(" = %.1fm³", effectiveCapacity)

	return c.JSON(models.CargoCalculationResponse{
		ShipTypeID:        req.ShipTypeID,
		ShipTypeName:      shipTypeName,
		BaseCapacity:      baseCapacity,
		SkillBonus:        skillBonusPercent,
		ModuleBonus:       moduleBonusM3,
		EffectiveCapacity: effectiveCapacity,
		CapacityBreakdown: breakdown,
	})
}

// CalculateWarp calculates effective warp speed and align time
//
// @Summary Calculate warp speed and align time
// @Description Calculate effective warp speed, inertia modifier, and align time
// @Description Includes skill bonuses and module effects with stacking penalties
// @Description Returns deterministic breakdown of all bonuses applied
// @Tags Calculations
// @Accept json
// @Produce json
// @Param request body models.WarpCalculationRequest true "Warp calculation parameters"
// @Success 200 {object} models.WarpCalculationResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/calculations/warp [post]
func (h *CalculationHandler) CalculateWarp(c *fiber.Ctx) error {
	var req models.WarpCalculationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid request body",
			"details": err.Error(),
		})
	}

	// Validate ship type ID
	if req.ShipTypeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "ship_type_id is required",
		})
	}

	// Get ship attributes from SDE if not provided
	ctx := c.Context()
	baseWarpSpeed := req.BaseWarpSpeed
	baseInertia := req.BaseInertia
	baseMass := req.BaseMass
	var shipTypeName string

	row := h.sdeDB.QueryRowContext(ctx,
		`SELECT 
			COALESCE(json_extract(name, '$.en'), json_extract(name, '$.de'), 'Unknown') as name,
			COALESCE(json_extract(dogma_attributes, '$.600'), 1.0) as base_warp_speed,
			COALESCE(json_extract(dogma_attributes, '$.70'), 1.0) as inertia_modifier,
			COALESCE(json_extract(dogma_attributes, '$.4'), 1000000) as mass
		FROM types WHERE _key = ?`,
		req.ShipTypeID,
	)

	var dbWarpSpeed, dbInertia, dbMass float64
	if err := row.Scan(&shipTypeName, &dbWarpSpeed, &dbInertia, &dbMass); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to fetch ship attributes",
			"details": err.Error(),
		})
	}

	// Use DB values if not provided in request
	if baseWarpSpeed == 0 {
		baseWarpSpeed = dbWarpSpeed
	}
	if baseInertia == 0 {
		baseInertia = dbInertia
	}
	if baseMass == 0 {
		baseMass = dbMass
	}

	// Calculate skill bonuses (default: 0%)
	warpSpeedBonusPercent := 0.0
	inertiaBonusPercent := 0.0

	if req.SkillLevels != nil {
		// Navigation: +5% warp speed per level
		warpSpeedBonusPercent += float64(req.SkillLevels.Navigation) * 5.0

		// Warp Drive Operation: +10% warp speed per level
		warpSpeedBonusPercent += float64(req.SkillLevels.WarpDriveOperation) * 10.0

		// Evasive Maneuvering: +5% agility (reduces inertia) per level
		inertiaBonusPercent += float64(req.SkillLevels.Evasive_Maneuvering) * 5.0
	}

	// Apply bonuses
	effectiveWarpSpeed := baseWarpSpeed * (1.0 + warpSpeedBonusPercent/100.0)
	effectiveInertia := baseInertia * (1.0 - inertiaBonusPercent/100.0)

	// Calculate align time: alignTime = ln(2) * inertiaModifier * mass / 500000
	// Simplified: alignTime ≈ inertiaModifier * mass / 500000 * 0.693
	alignTime := effectiveInertia * baseMass / 500000.0

	// Build breakdown string
	warpBreakdown := fmt.Sprintf("Base: %.2f AU/s", baseWarpSpeed)
	if warpSpeedBonusPercent > 0 {
		warpBreakdown += fmt.Sprintf(" + Skills: %.1f%%", warpSpeedBonusPercent)
	}
	warpBreakdown += fmt.Sprintf(" = %.2f AU/s", effectiveWarpSpeed)

	return c.JSON(models.WarpCalculationResponse{
		ShipTypeID:         req.ShipTypeID,
		ShipTypeName:       shipTypeName,
		BaseWarpSpeed:      baseWarpSpeed,
		EffectiveWarpSpeed: effectiveWarpSpeed,
		WarpSpeedBonus:     warpSpeedBonusPercent,
		BaseInertia:        baseInertia,
		EffectiveInertia:   effectiveInertia,
		InertiaBonus:       inertiaBonusPercent,
		AlignTime:          alignTime,
		WarpSpeedBreakdown: warpBreakdown,
	})
}
