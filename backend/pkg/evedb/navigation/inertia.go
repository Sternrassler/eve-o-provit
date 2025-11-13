// Package navigation provides EVE Online navigation and route planning functionality
package navigation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/dogma"
)

// ShipInertia contains inertia and align time information with applied bonuses
type ShipInertia struct {
	ShipTypeID       int64          `json:"ship_type_id"`
	ShipName         string         `json:"ship_name"`
	BaseInertia      float64        `json:"base_inertia"`      // Base Inertia Modifier (SDE Attribut 70)
	EffectiveInertia float64        `json:"effective_inertia"` // Final Inertia (mit Bonuses)
	ShipMass         float64        `json:"ship_mass"`         // kg (SDE Attribut 4)
	AlignTime        float64        `json:"align_time"`        // Sekunden (berechnet)
	AppliedBonuses   []AppliedBonus `json:"applied_bonuses"`
}

// GetShipInertiaDeterministic calculates inertia and align time deterministically from SDE + ESI data
// Implements the 7-step workflow from Issue #79
// Uses cargo.CharacterSkills and cargo.FittedItem for consistency with cargo calculations
func GetShipInertiaDeterministic(
	ctx context.Context,
	db *sql.DB,
	shipTypeID int64,
	characterSkills *cargo.CharacterSkills,
	fittedItems []cargo.FittedItem,
) (*ShipInertia, error) {

	// Step 1-2: Get base inertia and ship mass from SDE
	baseInertia, shipMass, shipName, err := getBaseInertiaAndMass(db, shipTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base inertia and mass: %w", err)
	}

	result := &ShipInertia{
		ShipTypeID:       shipTypeID,
		ShipName:         shipName,
		BaseInertia:      baseInertia,
		EffectiveInertia: baseInertia,
		ShipMass:         shipMass,
		AppliedBonuses:   make([]AppliedBonus, 0),
	}

	// Step 3-5: Apply Evasive Maneuvering skill bonus
	// Evasive Maneuvering Skill (3452): -5% inertia per level (lower is better!)
	if characterSkills != nil {
		evasiveLevel := getCharacterSkillLevel(characterSkills, 3452) // Evasive Maneuvering Skill ID
		if evasiveLevel > 0 {
			// Evasive Maneuvering provides -5% inertia per level
			// Lower inertia = faster align = better
			skillBonus := 5.0 * float64(evasiveLevel)
			result.EffectiveInertia *= (1.0 - (skillBonus / 100.0))

			result.AppliedBonuses = append(result.AppliedBonuses, AppliedBonus{
				Source:    "Skill",
				Name:      "Evasive Maneuvering",
				Value:     -skillBonus, // Negative because it's a reduction
				Operation: 6,           // PostPercent
				Count:     evasiveLevel,
			})
		}
	}

	// Step 6-7: Apply module/rig bonuses with stacking penalties
	if len(fittedItems) > 0 {
		itemGroups := groupItemsByType(fittedItems)

		for typeID, items := range itemGroups {
			// Get dogma effects for this module/rig type
			moduleEffect, err := dogma.GetModuleEffects(db, typeID)
			if err != nil {
				// Skip modules without dogma effects
				continue
			}

			// Find inertia modifiers (Attribut 70)
			inertiaMods := findInertiaModifiers(moduleEffect)
			if len(inertiaMods) == 0 {
				continue
			}

			count := len(items)

			// Apply each modifier with stacking penalties
			for _, mod := range inertiaMods {
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
				result.EffectiveInertia = dogma.ApplyModifierWithStacking(
					db,
					result.EffectiveInertia,
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

	// Calculate final align time using the formula: ln(2) × inertia × mass / 500000
	result.AlignTime = CalculateAlignTime(result.EffectiveInertia, result.ShipMass)

	return result, nil
}

// getBaseInertiaAndMass retrieves base inertia modifier and ship mass from SDE
// Attribut 70: inertiaModifier (from dogmaAttributes JSON)
// mass: direct column in types table
func getBaseInertiaAndMass(db *sql.DB, shipTypeID int64) (float64, float64, string, error) {
	// Query for ship name, mass (direct column), and dogma attributes
	query := `
		SELECT 
			json_extract(t.name, '$.en'),
			t.mass,
			td.dogmaAttributes
		FROM types t
		LEFT JOIN typeDogma td ON t._key = td._key
		WHERE t._key = ?
	`

	var shipName string
	var mass float64
	var dogmaJSON sql.NullString

	err := db.QueryRow(query, shipTypeID).Scan(&shipName, &mass, &dogmaJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, "", fmt.Errorf("ship type %d not found", shipTypeID)
		}
		return 0, 0, "", fmt.Errorf("query failed: %w", err)
	}

	// Validate mass
	if mass <= 0 {
		return 0, 0, "", fmt.Errorf("ship type %d has invalid mass: %.0f", shipTypeID, mass)
	}

	// Parse dogma attributes JSON to find inertia
	if !dogmaJSON.Valid || dogmaJSON.String == "" {
		return 0, 0, "", fmt.Errorf("ship type %d has no dogma attributes", shipTypeID)
	}

	var attributes []struct {
		AttributeID int64   `json:"attributeID"`
		Value       float64 `json:"value"`
	}

	if err := json.Unmarshal([]byte(dogmaJSON.String), &attributes); err != nil {
		return 0, 0, "", fmt.Errorf("failed to parse dogma attributes: %w", err)
	}

	// Find inertia modifier attribute (70)
	var inertia float64
	var hasInertia bool

	for _, attr := range attributes {
		if attr.AttributeID == 70 { // inertiaModifier
			inertia = attr.Value
			hasInertia = true
			break
		}
	}

	if !hasInertia {
		return 0, 0, "", fmt.Errorf("ship type %d has no inertia modifier attribute (70)", shipTypeID)
	}

	return inertia, mass, shipName, nil
}

// CalculateAlignTime calculates align time using EVE's formula
// Formula: align_time = ln(2) × inertia_modifier × mass / 500000
// Source: EVE University Wiki
// This is the canonical implementation - all other code should use this function
func CalculateAlignTime(inertia float64, mass float64) float64 {
	return math.Log(2) * inertia * mass / 500000.0
}

// findInertiaModifiers finds dogma modifiers that affect inertia (Attribut 70)
func findInertiaModifiers(moduleEffect *dogma.ModuleEffect) []dogma.ModifierInfo {
	modifiers := make([]dogma.ModifierInfo, 0)

	for _, eff := range moduleEffect.Effects {
		for _, mod := range eff.ModifierInfo {
			// Check if this modifier affects inertiaModifier (Attribut 70)
			if mod.ModifiedAttributeID == 70 {
				modifiers = append(modifiers, mod)
			}
		}
	}

	return modifiers
}
