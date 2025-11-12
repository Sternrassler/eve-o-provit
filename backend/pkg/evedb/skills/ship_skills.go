// Package skills provides SDE-based ship skill queries for deterministic cargo calculations
package skills

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// ShipSkillRequirement represents a required skill for a ship with cargo bonus info
type ShipSkillRequirement struct {
	SkillTypeID   int64   `json:"skill_type_id"`   // Type ID of required skill (e.g., 3340 = Gallente Hauler)
	MinimumLevel  int     `json:"minimum_level"`   // Minimum level to fly ship (e.g., 1)
	BonusPerLevel float64 `json:"bonus_per_level"` // Bonus per skill level (e.g., 5.0 = 5%)
	AttributeID   int64   `json:"attribute_id"`    // Dogma attribute ID for bonus (e.g., 496)
}

// ShipCargoSkills contains all cargo-relevant skills for a ship
type ShipCargoSkills struct {
	ShipTypeID   int64                  `json:"ship_type_id"`
	ShipName     string                 `json:"ship_name"`
	BaseCapacity float64                `json:"base_capacity"` // Attribute 38
	Skills       []ShipSkillRequirement `json:"skills"`
}

// ShipNavigationSkills contains all navigation-relevant skills for a ship
type ShipNavigationSkills struct {
	ShipTypeID int64                     `json:"ship_type_id"`
	ShipName   string                    `json:"ship_name"`
	Skills     []ShipSkillRequirementNav `json:"skills"`
}

// ShipSkillRequirementNav represents a required skill with navigation bonus info
type ShipSkillRequirementNav struct {
	SkillTypeID   int64   `json:"skill_type_id"`   // Type ID of required skill (e.g., 3456 = Navigation)
	SkillName     string  `json:"skill_name"`      // Skill name (e.g., "Navigation")
	MinimumLevel  int     `json:"minimum_level"`   // Minimum level to fly ship (e.g., 1)
	BonusPerLevel float64 `json:"bonus_per_level"` // Bonus per skill level (e.g., 5.0 = 5%)
	AttributeID   int64   `json:"attribute_id"`    // Dogma attribute ID for bonus (e.g., 1281)
}

// DogmaAttribute represents a single dogma attribute entry
type DogmaAttribute struct {
	AttributeID int64   `json:"attributeID"`
	Value       float64 `json:"value"`
}

// GetShipCargoSkills retrieves cargo-relevant skills from SDE for a ship
// Returns base capacity + required skills + bonuses per level
func GetShipCargoSkills(db *sql.DB, shipTypeID int64) (*ShipCargoSkills, error) {
	// Query types + typeDogma for ship attributes
	query := `
		SELECT 
			t._key,
			COALESCE(json_extract(t.name, '$.en'), 'Unknown Ship'),
			COALESCE(t.capacity, 0),
			td.dogmaAttributes
		FROM types t
		LEFT JOIN typeDogma td ON t._key = td._key
		WHERE t._key = ?
	`

	var typeID int64
	var shipName string
	var baseCapacity float64
	var dogmaAttribsJSON sql.NullString

	err := db.QueryRow(query, shipTypeID).Scan(&typeID, &shipName, &baseCapacity, &dogmaAttribsJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ship type ID %d not found in SDE", shipTypeID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ship dogma: %w", err)
	}

	result := &ShipCargoSkills{
		ShipTypeID:   typeID,
		ShipName:     shipName,
		BaseCapacity: baseCapacity, // From types.capacity column
		Skills:       make([]ShipSkillRequirement, 0),
	}

	// Parse dogma attributes JSON
	if !dogmaAttribsJSON.Valid || dogmaAttribsJSON.String == "" {
		return result, nil // Ship has no dogma attributes
	}

	var attributes []DogmaAttribute
	if err := json.Unmarshal([]byte(dogmaAttribsJSON.String), &attributes); err != nil {
		return nil, fmt.Errorf("failed to parse dogma attributes JSON: %w", err)
	}

	// Build attribute map for quick lookup
	attrMap := make(map[int64]float64)
	for _, attr := range attributes {
		attrMap[attr.AttributeID] = attr.Value
	}

	// Extract required skills and bonuses
	// Attributes: 182-184 = requiredSkill1-3, 277-279 = requiredSkill1Level-3Level
	// Attributes: 496, 813 = shipBonusGI, shipBonusGI2 (cargo bonuses)
	skillPairs := []struct {
		skillAttr int64
		levelAttr int64
		bonusAttr int64
	}{
		{182, 277, 496}, // requiredSkill1, requiredSkill1Level, shipBonusGI
		{183, 278, 496}, // requiredSkill2, requiredSkill2Level, shipBonusGI
		{184, 279, 496}, // requiredSkill3, requiredSkill3Level, shipBonusGI
	}

	for _, pair := range skillPairs {
		skillTypeID, hasSkill := attrMap[pair.skillAttr]
		minLevel, hasLevel := attrMap[pair.levelAttr]
		bonus, hasBonus := attrMap[pair.bonusAttr]

		if hasSkill && hasLevel {
			req := ShipSkillRequirement{
				SkillTypeID:  int64(skillTypeID),
				MinimumLevel: int(minLevel),
				AttributeID:  pair.bonusAttr,
			}

			// Bonus is optional (not all required skills provide cargo bonus)
			if hasBonus {
				req.BonusPerLevel = bonus
			}

			result.Skills = append(result.Skills, req)
		}
	}

	return result, nil
}

// GetBaseCapacity retrieves only the base cargo capacity for a ship
// Convenience function for simple queries
func GetBaseCapacity(db *sql.DB, shipTypeID int64) (float64, error) {
	query := `
		SELECT COALESCE(t.capacity, 0)
		FROM types t
		WHERE t._key = ?
	`

	var capacity float64
	err := db.QueryRow(query, shipTypeID).Scan(&capacity)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("ship type ID %d not found in SDE", shipTypeID)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query ship capacity: %w", err)
	}

	return capacity, nil
}

// GetShipNavigationSkills retrieves navigation-relevant skills from SDE for a ship
// Returns required Navigation skills + bonuses per level (e.g., Navigation skill +5% warp speed/level)
func GetShipNavigationSkills(db *sql.DB, shipTypeID int64) (*ShipNavigationSkills, error) {
	// Query types + typeDogma for ship attributes
	query := `
		SELECT 
			t._key,
			COALESCE(json_extract(t.name, '$.en'), 'Unknown Ship'),
			td.dogmaAttributes
		FROM types t
		LEFT JOIN typeDogma td ON t._key = td._key
		WHERE t._key = ?
	`

	var typeID int64
	var shipName string
	var dogmaAttribsJSON sql.NullString

	err := db.QueryRow(query, shipTypeID).Scan(&typeID, &shipName, &dogmaAttribsJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ship type ID %d not found in SDE", shipTypeID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ship dogma: %w", err)
	}

	result := &ShipNavigationSkills{
		ShipTypeID: typeID,
		ShipName:   shipName,
		Skills:     make([]ShipSkillRequirementNav, 0),
	}

	// Parse dogma attributes JSON
	if !dogmaAttribsJSON.Valid || dogmaAttribsJSON.String == "" {
		return result, nil // Ship has no dogma attributes
	}

	var attributes []DogmaAttribute
	if err := json.Unmarshal([]byte(dogmaAttribsJSON.String), &attributes); err != nil {
		return nil, fmt.Errorf("failed to parse dogma attributes JSON: %w", err)
	}

	// Build attribute map for quick lookup
	attrMap := make(map[int64]float64)
	for _, attr := range attributes {
		attrMap[attr.AttributeID] = attr.Value
	}

	// Extract Navigation skill (if present)
	// Attributes: 182 = requiredSkill1, 277 = requiredSkill1Level, 1281 = Navigation Bonus
	// Navigation Skill ID: 3456 (TypeIDNavigation from skills.go)
	skillPairs := []struct {
		skillAttr int64
		levelAttr int64
		bonusAttr int64
	}{
		{182, 277, 1281}, // requiredSkill1, requiredSkill1Level, Navigation bonus
		{183, 278, 1281}, // requiredSkill2, requiredSkill2Level, Navigation bonus
		{184, 279, 1281}, // requiredSkill3, requiredSkill3Level, Navigation bonus
	}

	for _, pair := range skillPairs {
		skillTypeID, hasSkill := attrMap[pair.skillAttr]
		minLevel, hasLevel := attrMap[pair.levelAttr]
		bonus, hasBonus := attrMap[pair.bonusAttr]

		if hasSkill && hasLevel {
			// Check if this is a navigation skill (3456 = Navigation)
			if int64(skillTypeID) == TypeIDNavigation {
				req := ShipSkillRequirementNav{
					SkillTypeID:  int64(skillTypeID),
					SkillName:    "Navigation",
					MinimumLevel: int(minLevel),
					AttributeID:  pair.bonusAttr,
				}

				// Bonus is optional
				if hasBonus {
					req.BonusPerLevel = bonus
				} else {
					// Navigation skill default: +5% per level
					req.BonusPerLevel = 5.0
				}

				result.Skills = append(result.Skills, req)
			}
		}
	}

	return result, nil
}
