// Package cargo provides EVE Online cargo and hauling calculation functionality
// Migrated from eve-sde project: github.com/Sternrassler/eve-sde/pkg/evedb/cargo
package cargo

import (
	"database/sql"
	"fmt"
)

// SkillModifiers contains optional skill levels for capacity calculations
type SkillModifiers struct {
	RacialHaulerLevel *int     `json:"racial_hauler_level,omitempty"` // +5% per level
	FreighterLevel    *int     `json:"freighter_level,omitempty"`     // +5% per level
	MiningBargeLevel  *int     `json:"mining_barge_level,omitempty"`  // Ore hold
	CargoMultiplier   *float64 `json:"cargo_multiplier,omitempty"`
}

// ItemVolume contains volume and pricing information for an item
type ItemVolume struct {
	TypeID         int64   `json:"type_id"`
	ItemName       string  `json:"item_name"`
	Volume         float64 `json:"volume"`
	Capacity       float64 `json:"capacity"`
	PackagedVolume float64 `json:"packaged_volume"`
	BasePrice      float64 `json:"base_price"`
	CategoryID     int64   `json:"category_id"`
	CategoryName   string  `json:"category_name"`
	MarketGroupID  *int64  `json:"market_group_id,omitempty"`
	IskPerM3       float64 `json:"isk_per_m3"`
}

// ShipCapacities contains all cargo holds of a ship
type ShipCapacities struct {
	ShipTypeID             int64   `json:"ship_type_id"`
	ShipName               string  `json:"ship_name"`
	BaseCargoHold          float64 `json:"base_cargo_hold"`
	EffectiveCargoHold     float64 `json:"effective_cargo_hold"`
	BaseTotalCapacity      float64 `json:"base_total_capacity"`
	EffectiveTotalCapacity float64 `json:"effective_total_capacity"`
	SkillBonus             float64 `json:"skill_bonus"`
	SkillsApplied          bool    `json:"skills_applied"`
}

// CargoFitResult describes how many items fit in a ship
type CargoFitResult struct {
	ShipTypeID        int64   `json:"ship_type_id"`
	ShipName          string  `json:"ship_name"`
	ItemTypeID        int64   `json:"item_type_id"`
	ItemName          string  `json:"item_name"`
	ItemVolume        float64 `json:"item_volume"`
	BaseCapacity      float64 `json:"base_capacity"`
	EffectiveCapacity float64 `json:"effective_capacity"`
	SkillBonus        float64 `json:"skill_bonus"`
	SkillsApplied     bool    `json:"skills_applied"`
	MaxQuantity       int     `json:"max_quantity"`
	TotalVolume       float64 `json:"total_volume"`
	RemainingSpace    float64 `json:"remaining_space"`
	UtilizationPct    float64 `json:"utilization_pct"`
}

// GetItemVolume retrieves volume information for an item
func GetItemVolume(db *sql.DB, itemTypeID int64) (*ItemVolume, error) {
	// Query directly from types table in SDE
	// Note: SDE doesn't have packagedVolume - using volume for all items
	query := `
		SELECT 
			_key,
			json_extract(name, '$.en'),
			COALESCE(volume, 0),
			COALESCE(capacity, 0),
			COALESCE(volume, 0) as packaged_volume,
			COALESCE(basePrice, 0),
			groupID,
			'' as category_name,
			marketGroupID,
			0.0 as isk_per_m3
		FROM types
		WHERE _key = ?
	`

	var item ItemVolume
	var marketGroupID sql.NullInt64

	err := db.QueryRow(query, itemTypeID).Scan(
		&item.TypeID,
		&item.ItemName,
		&item.Volume,
		&item.Capacity,
		&item.PackagedVolume,
		&item.BasePrice,
		&item.CategoryID,
		&item.CategoryName,
		&marketGroupID,
		&item.IskPerM3,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("item with type ID %d not found", itemTypeID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query item volume: %w", err)
	}

	if marketGroupID.Valid {
		item.MarketGroupID = &marketGroupID.Int64
	}

	return &item, nil
}

// GetShipCapacities retrieves cargo holds for a ship
func GetShipCapacities(db *sql.DB, shipTypeID int64, skills *SkillModifiers) (*ShipCapacities, error) {
	query := `
		SELECT 
			ship_type_id,
			ship_name,
			COALESCE(base_cargo_capacity, 0) as base_cargo
		FROM v_ship_cargo_capacities
		WHERE ship_type_id = ?
	`

	var ship ShipCapacities
	err := db.QueryRow(query, shipTypeID).Scan(
		&ship.ShipTypeID,
		&ship.ShipName,
		&ship.BaseCargoHold,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("ship with type ID %d not found", shipTypeID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query ship capacities: %w", err)
	}

	ship.BaseTotalCapacity = ship.BaseCargoHold

	// Apply skill modifiers
	if skills != nil {
		ship.SkillsApplied = true
		ship.EffectiveCargoHold = ApplySkillModifiers(ship.BaseCargoHold, skills)
		ship.EffectiveTotalCapacity = ship.EffectiveCargoHold

		if ship.BaseTotalCapacity > 0 {
			ship.SkillBonus = ((ship.EffectiveTotalCapacity / ship.BaseTotalCapacity) - 1.0) * 100.0
		}
	} else {
		ship.EffectiveCargoHold = ship.BaseCargoHold
		ship.EffectiveTotalCapacity = ship.BaseTotalCapacity
	}

	return &ship, nil
}

// CalculateCargoFit calculates how many items fit in a ship
func CalculateCargoFit(db *sql.DB, shipTypeID, itemTypeID int64, skills *SkillModifiers) (*CargoFitResult, error) {
	// Get ship capacities
	ship, err := GetShipCapacities(db, shipTypeID, skills)
	if err != nil {
		return nil, err
	}

	// Get item volume
	item, err := GetItemVolume(db, itemTypeID)
	if err != nil {
		return nil, err
	}

	// Use packaged volume if available (for ships being transported)
	itemVol := item.Volume
	if item.PackagedVolume > 0 {
		itemVol = item.PackagedVolume
	}

	if itemVol <= 0 {
		return nil, fmt.Errorf("item %s has zero or negative volume", item.ItemName)
	}

	// Calculate fit
	result := &CargoFitResult{
		ShipTypeID:        ship.ShipTypeID,
		ShipName:          ship.ShipName,
		ItemTypeID:        item.TypeID,
		ItemName:          item.ItemName,
		ItemVolume:        itemVol,
		BaseCapacity:      ship.BaseTotalCapacity,
		EffectiveCapacity: ship.EffectiveTotalCapacity,
		SkillBonus:        ship.SkillBonus,
		SkillsApplied:     ship.SkillsApplied,
	}

	// Calculate max quantity
	result.MaxQuantity = int(result.EffectiveCapacity / itemVol)
	result.TotalVolume = float64(result.MaxQuantity) * itemVol
	result.RemainingSpace = result.EffectiveCapacity - result.TotalVolume

	// Calculate utilization percentage
	if result.EffectiveCapacity > 0 {
		result.UtilizationPct = (result.TotalVolume / result.EffectiveCapacity) * 100.0
	}

	return result, nil
}

// ApplySkillModifiers calculates effective capacity based on skills
func ApplySkillModifiers(baseCapacity float64, skills *SkillModifiers) float64 {
	if skills == nil {
		return baseCapacity
	}

	effective := baseCapacity

	// Racial Hauler Skill (5% per level)
	if skills.RacialHaulerLevel != nil {
		bonus := float64(*skills.RacialHaulerLevel) * 0.05
		effective *= (1.0 + bonus)
	}

	// Freighter Skill (5% per level)
	if skills.FreighterLevel != nil {
		bonus := float64(*skills.FreighterLevel) * 0.05
		effective *= (1.0 + bonus)
	}

	// Custom multiplier
	if skills.CargoMultiplier != nil {
		effective *= *skills.CargoMultiplier
	}

	return effective
}
