// Package navigation provides EVE Online navigation and route planning functionality
package navigation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/dogma"
)

// ShipWarpSpeed contains warp speed information with applied bonuses
type ShipWarpSpeed struct {
	ShipTypeID         int64          `json:"ship_type_id"`
	ShipName           string         `json:"ship_name"`
	BaseWarpSpeed      float64        `json:"base_warp_speed"`      // AU/s (SDE Attribut 20)
	EffectiveWarpSpeed float64        `json:"effective_warp_speed"` // AU/s (final mit allen Bonuses)
	AppliedBonuses     []AppliedBonus `json:"applied_bonuses"`
}

// AppliedBonus represents a single bonus applied to warp speed (aligned with cargo.AppliedBonus)
type AppliedBonus struct {
	Source    string  `json:"source"`    // "Skill", "Module", "Rig"
	Name      string  `json:"name"`      // Skill/Module name
	Value     float64 `json:"value"`     // Bonus value (% or absolute)
	Operation int     `json:"operation"` // Dogma operation code
	Count     int     `json:"count"`     // Skill level or module quantity
}

// GetShipWarpSpeedDeterministic calculates warp speed deterministically from SDE + ESI data
// Implements the 7-step workflow from Issue #78
// Uses cargo.CharacterSkills and cargo.FittedItem for consistency with cargo calculations
func GetShipWarpSpeedDeterministic(
	ctx context.Context,
	db *sql.DB,
	shipTypeID int64,
	characterSkills *cargo.CharacterSkills,
	fittedItems []cargo.FittedItem,
) (*ShipWarpSpeed, error) {

	// Step 1: Get base warp speed from SDE (Attribut 20: warpSpeedMultiplier)
	baseWarpSpeed, shipName, err := getBaseWarpSpeed(db, shipTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base warp speed: %w", err)
	}

	result := &ShipWarpSpeed{
		ShipTypeID:         shipTypeID,
		ShipName:           shipName,
		BaseWarpSpeed:      baseWarpSpeed,
		EffectiveWarpSpeed: baseWarpSpeed,
		AppliedBonuses:     make([]AppliedBonus, 0),
	}

	// Step 2-4: Apply Navigation skill bonus (passive bonus for all ships)
	// Navigation Skill (3456): +5% warp speed per level
	if characterSkills != nil {
		navLevel := getCharacterSkillLevel(characterSkills, 3456) // Navigation Skill ID
		if navLevel > 0 {
			// Navigation skill provides +5% warp speed per level
			skillBonus := 5.0 * float64(navLevel)
			result.EffectiveWarpSpeed *= (1.0 + (skillBonus / 100.0))

			result.AppliedBonuses = append(result.AppliedBonuses, AppliedBonus{
				Source:    "Skill",
				Name:      "Navigation",
				Value:     skillBonus,
				Operation: 6, // PostPercent
				Count:     navLevel,
			})
		}
	} // Step 5-7: Apply module/rig bonuses with stacking penalties
	if len(fittedItems) > 0 {
		itemGroups := groupItemsByType(fittedItems)

		for typeID, items := range itemGroups {
			// Get dogma effects for this module/rig type
			moduleEffect, err := dogma.GetModuleEffects(db, typeID)
			if err != nil {
				// Skip modules without dogma effects
				continue
			}

			// Find warp speed modifiers (Attribut 20)
			warpMods := findWarpSpeedModifiers(moduleEffect)
			if len(warpMods) == 0 {
				continue
			}

			count := len(items)

			// Apply each modifier with stacking penalties
			for _, mod := range warpMods {
				modValue, exists := moduleEffect.Attributes[mod.ModifyingAttributeID]
				if !exists {
					continue
				}

				// Check if attribute is stackable
				isStackable, err := dogma.IsAttributeStackable(db, mod.ModifyingAttributeID)
				if err != nil {
					// Default to stackable on error
					isStackable = true
				}

				// Apply modifier with stacking penalties
				result.EffectiveWarpSpeed = dogma.ApplyModifierWithStacking(
					db,
					result.EffectiveWarpSpeed,
					mod,
					modValue,
					count,
					isStackable,
				)

				// Determine source type (Module vs Rig)
				source := "Module"
				if len(items) > 0 && len(items[0].Slot) >= 3 && items[0].Slot[:3] == "Rig" {
					source = "Rig"
				}

				result.AppliedBonuses = append(result.AppliedBonuses, AppliedBonus{
					Source:    source,
					Name:      moduleEffect.TypeName,
					Value:     modValue,
					Operation: mod.Operation,
					Count:     count,
				})
			}
		}
	}

	return result, nil
}

// getBaseWarpSpeed retrieves base warp speed from SDE (Attribut 20 or 600: warpSpeedMultiplier)
func getBaseWarpSpeed(db *sql.DB, shipTypeID int64) (float64, string, error) {
	// Query for ship name and dogma attributes
	query := `
		SELECT 
			json_extract(t.name, '$.en'),
			td.dogmaAttributes
		FROM types t
		LEFT JOIN typeDogma td ON t._key = td._key
		WHERE t._key = ?
	`

	var shipName string
	var dogmaJSON sql.NullString

	err := db.QueryRow(query, shipTypeID).Scan(&shipName, &dogmaJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("ship type %d not found", shipTypeID)
		}
		return 0, "", fmt.Errorf("query failed: %w", err)
	}

	// Parse dogma attributes JSON to find warp speed (attribute 600 or 20)
	if !dogmaJSON.Valid || dogmaJSON.String == "" {
		return 0, "", fmt.Errorf("ship type %d has no dogma attributes", shipTypeID)
	}

	var attributes []struct {
		AttributeID int64   `json:"attributeID"`
		Value       float64 `json:"value"`
	}

	if err := json.Unmarshal([]byte(dogmaJSON.String), &attributes); err != nil {
		return 0, "", fmt.Errorf("failed to parse dogma attributes: %w", err)
	}

	// Find warp speed attribute (600 = warpSpeedMultiplier)
	for _, attr := range attributes {
		if attr.AttributeID == 600 {
			return attr.Value, shipName, nil
		}
	}

	return 0, "", fmt.Errorf("ship type %d has no warp speed attribute (600)", shipTypeID)
}

// getCharacterSkillLevel retrieves character's skill level from ESI data
func getCharacterSkillLevel(charSkills *cargo.CharacterSkills, skillTypeID int64) int {
	if charSkills == nil {
		return 0
	}
	for _, skill := range charSkills.Skills {
		if skill.SkillID == skillTypeID {
			return skill.TrainedSkillLevel
		}
	}
	return 0
}

// groupItemsByType groups fitted items by TypeID
func groupItemsByType(items []cargo.FittedItem) map[int64][]cargo.FittedItem {
	groups := make(map[int64][]cargo.FittedItem)
	for _, item := range items {
		groups[item.TypeID] = append(groups[item.TypeID], item)
	}
	return groups
}

// findWarpSpeedModifiers finds dogma modifiers that affect warp speed (Attribut 600)
func findWarpSpeedModifiers(moduleEffect *dogma.ModuleEffect) []dogma.ModifierInfo {
	modifiers := make([]dogma.ModifierInfo, 0)

	for _, eff := range moduleEffect.Effects {
		for _, mod := range eff.ModifierInfo {
			// Check if this modifier affects warpSpeedMultiplier (Attribut 600)
			if mod.ModifiedAttributeID == 600 {
				modifiers = append(modifiers, mod)
			}
		}
	}

	return modifiers
}
