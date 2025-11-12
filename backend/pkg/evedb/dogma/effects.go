// Package dogma provides SDE-based dogma effect parsing for deterministic attribute calculations
package dogma

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
)

// ModifierInfo represents a single modifier from dogma effects
type ModifierInfo struct {
	Domain               string `json:"domain"`               // "shipID", "charID", "targetID"
	Func                 string `json:"func"`                 // "ItemModifier", "LocationGroupModifier"
	ModifiedAttributeID  int64  `json:"modifiedAttributeID"`  // 38 = capacity
	ModifyingAttributeID int64  `json:"modifyingAttributeID"` // 149 = cargoCapacityMultiplier
	Operation            int    `json:"operation"`            // 4/6 = PostPercent
}

// DogmaEffect represents a complete dogma effect with modifiers
type DogmaEffect struct {
	EffectID     int64          `json:"effect_id"`
	EffectName   string         `json:"effect_name"`
	ModifierInfo []ModifierInfo `json:"modifier_info"`
}

// ModuleEffect represents all effects and attributes for a module/rig
type ModuleEffect struct {
	TypeID      int64             `json:"type_id"`
	TypeName    string            `json:"type_name"`
	Attributes  map[int64]float64 `json:"attributes"`   // AttributeID → Value
	Effects     []DogmaEffect     `json:"effects"`      // All dogma effects
	IsStackable bool              `json:"is_stackable"` // From dogmaAttributes
}

// GetModuleEffects retrieves complete dogma effects for a module/rig from SDE
func GetModuleEffects(db *sql.DB, moduleTypeID int64) (*ModuleEffect, error) {
	// Query types + typeDogma for module
	query := `
		SELECT 
			t._key,
			COALESCE(json_extract(t.name, '$.en'), 'Unknown Module'),
			td.dogmaAttributes,
			td.dogmaEffects
		FROM types t
		LEFT JOIN typeDogma td ON t._key = td._key
		WHERE t._key = ?
	`

	var typeID int64
	var typeName string
	var dogmaAttribsJSON, dogmaEffectsJSON sql.NullString

	err := db.QueryRow(query, moduleTypeID).Scan(&typeID, &typeName, &dogmaAttribsJSON, &dogmaEffectsJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("module type ID %d not found in SDE", moduleTypeID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query module dogma: %w", err)
	}

	result := &ModuleEffect{
		TypeID:     typeID,
		TypeName:   typeName,
		Attributes: make(map[int64]float64),
		Effects:    make([]DogmaEffect, 0),
	}

	// Parse dogmaAttributes JSON
	if dogmaAttribsJSON.Valid && dogmaAttribsJSON.String != "" {
		var attributes []struct {
			AttributeID int64   `json:"attributeID"`
			Value       float64 `json:"value"`
		}
		if err := json.Unmarshal([]byte(dogmaAttribsJSON.String), &attributes); err != nil {
			return nil, fmt.Errorf("failed to parse dogma attributes JSON: %w", err)
		}

		for _, attr := range attributes {
			result.Attributes[attr.AttributeID] = attr.Value
		}
	}

	// Parse dogmaEffects JSON
	if dogmaEffectsJSON.Valid && dogmaEffectsJSON.String != "" {
		var effectRefs []struct {
			EffectID  int64 `json:"effectID"`
			IsDefault bool  `json:"isDefault"`
		}
		if err := json.Unmarshal([]byte(dogmaEffectsJSON.String), &effectRefs); err != nil {
			return nil, fmt.Errorf("failed to parse dogma effects JSON: %w", err)
		}

		// Query dogmaEffects table for each effect
		for _, ref := range effectRefs {
			effect, err := getEffectDetails(db, ref.EffectID)
			if err != nil {
				// Skip effects that can't be loaded (non-critical)
				continue
			}
			result.Effects = append(result.Effects, *effect)
		}
	}

	// Check stackable flag for relevant attributes
	result.IsStackable = checkStackable(db, result.Attributes)

	return result, nil
}

// getEffectDetails retrieves modifier info for a dogma effect
func getEffectDetails(db *sql.DB, effectID int64) (*DogmaEffect, error) {
	query := `
		SELECT 
			_key,
			COALESCE(name, 'Unknown Effect'),
			modifierInfo
		FROM dogmaEffects
		WHERE _key = ?
	`

	var id int64
	var name string
	var modifierInfoJSON sql.NullString

	err := db.QueryRow(query, effectID).Scan(&id, &name, &modifierInfoJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to query dogma effect %d: %w", effectID, err)
	}

	effect := &DogmaEffect{
		EffectID:     id,
		EffectName:   name,
		ModifierInfo: make([]ModifierInfo, 0),
	}

	if modifierInfoJSON.Valid && modifierInfoJSON.String != "" {
		if err := json.Unmarshal([]byte(modifierInfoJSON.String), &effect.ModifierInfo); err != nil {
			return nil, fmt.Errorf("failed to parse modifier info JSON: %w", err)
		}
	}

	return effect, nil
}

// checkStackable checks if any attribute in the map is stackable
func checkStackable(db *sql.DB, attributes map[int64]float64) bool {
	if len(attributes) == 0 {
		return true // Default to stackable
	}

	// Query dogmaAttributes for stackable flag
	// Check common cargo attributes: 149 (cargoCapacityMultiplier), 614 (cargoCapacityBonus)
	cargoAttrs := []int64{149, 614}
	for _, attrID := range cargoAttrs {
		if _, exists := attributes[attrID]; exists {
			stackable, err := isAttributeStackable(db, attrID)
			if err == nil {
				return stackable
			}
		}
	}

	return true // Default to stackable if check fails
}

// isAttributeStackable checks if a dogma attribute is stackable
func isAttributeStackable(db *sql.DB, attributeID int64) (bool, error) {
	query := `
		SELECT COALESCE(stackable, 1)
		FROM dogmaAttributes
		WHERE _key = ?
	`

	var stackable int
	err := db.QueryRow(query, attributeID).Scan(&stackable)
	if err != nil {
		return true, err
	}

	return stackable == 1, nil
}

// ApplyModifier calculates the effect of a modifier on a base value
// Implements EVE's dogma operation codes
func ApplyModifier(baseValue float64, modifier ModifierInfo, modifierValue float64, count int) float64 {
	switch modifier.Operation {
	case 0: // PreAssignment - direct value override
		return modifierValue

	case 1: // PreMul - pre-multiplicative
		return baseValue * modifierValue

	case 2: // PreDiv - pre-division
		if modifierValue != 0 {
			return baseValue / modifierValue
		}
		return baseValue

	case 3: // ModAdd - modulo add
		return baseValue + modifierValue

	case 4: // PostPercent - post-multiplicative (val/100 + 1)
		// Used by modules (e.g., Expanded Cargohold I)
		// Example: 1.175 → +17.5% → multiply by 1.175
		multiplier := modifierValue
		return baseValue * math.Pow(multiplier, float64(count))

	case 6: // PostPercent - post-multiplicative (1 + val/100)
		// Used by rigs (e.g., Medium Cargohold Optimization I)
		// Example: 15.0 → +15% → multiply by 1.15
		multiplier := 1.0 + (modifierValue / 100.0)
		return baseValue * math.Pow(multiplier, float64(count))

	case 5: // PostDiv - post-division
		if modifierValue != 0 {
			return baseValue / math.Pow(modifierValue, float64(count))
		}
		return baseValue

	case 7: // PostAssignment - direct override (highest priority)
		return modifierValue

	default:
		// Unknown operation - return unchanged
		return baseValue
	}
}

// ApplyModifierWithStacking applies a modifier with EVE's deterministic stacking penalties
// Stacking penalty formula: effectiveness = e^(-(n-1)² / 2.67²)
// where n is the module number (1st, 2nd, 3rd, etc.)
func ApplyModifierWithStacking(db *sql.DB, baseValue float64, modifier ModifierInfo, modifierValue float64, count int, isStackable bool) float64 {
	// If not stackable or only 1 module, use regular application
	if !isStackable || count <= 1 {
		return ApplyModifier(baseValue, modifier, modifierValue, count)
	}

	// Apply stacking penalties (EVE formula)
	result := baseValue

	switch modifier.Operation {
	case 4: // PostMul (modules: value is multiplier)
		for i := 0; i < count; i++ {
			penalty := calculateStackingPenalty(i)
			effectiveMultiplier := 1.0 + ((modifierValue - 1.0) * penalty)
			result *= effectiveMultiplier
		}
		return result

	case 6: // PostPercent (rigs: value is percentage)
		for i := 0; i < count; i++ {
			penalty := calculateStackingPenalty(i)
			bonusPercent := modifierValue / 100.0
			effectiveBonus := bonusPercent * penalty
			result *= (1.0 + effectiveBonus)
		}
		return result

	default:
		// For other operations, fall back to regular application
		return ApplyModifier(baseValue, modifier, modifierValue, count)
	}
}

// calculateStackingPenalty calculates EVE's stacking penalty for the nth module (0-indexed)
// Formula: e^(-(n)² / 2.67²)
// Results: 1st=100%, 2nd=86.9%, 3rd=57.1%, 4th=28.3%, 5th=10.6%, 6th=3.0%
func calculateStackingPenalty(n int) float64 {
	if n == 0 {
		return 1.0 // First module: 100% effectiveness
	}

	// EVE's stacking formula
	exponent := -math.Pow(float64(n), 2) / math.Pow(2.67, 2)
	return math.Exp(exponent)
}

// IsAttributeStackable checks if a dogma attribute has stacking penalties
func IsAttributeStackable(db *sql.DB, attributeID int64) (bool, error) {
	query := `
		SELECT COALESCE(stackable, 1)
		FROM dogmaAttributes
		WHERE _key = ?
	`

	var stackable int
	err := db.QueryRow(query, attributeID).Scan(&stackable)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil // Default to stackable if not found
		}
		return true, err
	}

	return stackable == 1, nil
}

// FindCargoModifiers extracts cargo-relevant modifiers from module effects
// Returns modifiers that affect capacity (Attribute 38)
func FindCargoModifiers(effect *ModuleEffect) []ModifierInfo {
	modifiers := make([]ModifierInfo, 0)

	for _, eff := range effect.Effects {
		for _, mod := range eff.ModifierInfo {
			// Filter for capacity modifiers (modifiedAttributeID == 38)
			if mod.ModifiedAttributeID == 38 {
				modifiers = append(modifiers, mod)
			}
		}
	}

	return modifiers
}

// CalculateCargoBonus calculates total cargo bonus from multiple modules/rigs
// Handles stacking and operation codes deterministically
func CalculateCargoBonus(baseCapacity float64, modules map[int64][]ModuleEffect) float64 {
	capacity := baseCapacity

	// Process each module type separately (grouped by TypeID)
	for _, moduleGroup := range modules {
		if len(moduleGroup) == 0 {
			continue
		}

		// Get first module as reference (all same type)
		refModule := moduleGroup[0]
		count := len(moduleGroup)

		// Find cargo modifiers
		cargoMods := FindCargoModifiers(&refModule)
		if len(cargoMods) == 0 {
			continue
		}

		// Apply each modifier
		for _, mod := range cargoMods {
			// Get modifier value from attributes
			modValue, exists := refModule.Attributes[mod.ModifyingAttributeID]
			if !exists {
				continue
			}

			// Apply modifier with count (handles stacking)
			capacity = ApplyModifier(capacity, mod, modValue, count)
		}
	}

	return capacity
}
