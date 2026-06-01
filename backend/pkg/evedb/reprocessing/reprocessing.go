// Package reprocessing provides SDE ore catalog and reprocessing output queries.
package reprocessing

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Material represents a single output material from reprocessing an ore.
type Material struct {
	MaterialTypeID int64 `json:"materialTypeID"`
	Quantity       int64 `json:"quantity"`
}

// Ore represents an asteroid ore type with its reprocessing output.
type Ore struct {
	TypeID      int64
	Name        string
	GroupID     int64
	PortionSize int64
	VolumeM3    float64
	Materials   []Material
}

// GetOre returns a single ore type by its typeID, including reprocessing materials.
func GetOre(db *sql.DB, oreTypeID int64) (*Ore, error) {
	var o Ore
	var matsJSON sql.NullString
	err := db.QueryRow(`
		SELECT t._key, COALESCE(json_extract(t.name,'$.en'),'?'), t.groupID,
		       COALESCE(t.portionSize,1), COALESCE(t.volume,0), tm.materials
		FROM types t LEFT JOIN typeMaterials tm ON tm._key = t._key
		WHERE t._key = ?`, oreTypeID).
		Scan(&o.TypeID, &o.Name, &o.GroupID, &o.PortionSize, &o.VolumeM3, &matsJSON)
	if err != nil {
		return nil, fmt.Errorf("get ore %d: %w", oreTypeID, err)
	}
	if matsJSON.Valid && matsJSON.String != "" {
		if err := json.Unmarshal([]byte(matsJSON.String), &o.Materials); err != nil {
			return nil, fmt.Errorf("parse materials for %d: %w", oreTypeID, err)
		}
	}
	return &o, nil
}

// ListOres returns all published asteroid ore types from the SDE.
func ListOres(db *sql.DB) ([]Ore, error) {
	rows, err := db.Query(`
		SELECT t._key, COALESCE(json_extract(t.name,'$.en'),'?'), t.groupID,
		       COALESCE(t.portionSize,1), COALESCE(t.volume,0)
		FROM types t
		JOIN groups g ON t.groupID = g._key
		JOIN categories c ON g.categoryID = c._key
		WHERE json_extract(c.name,'$.en') = 'Asteroid' AND t.published = 1
		  -- Exclude compressed forms ("Compressed X", "Batch Compressed X"): they are
		  -- not mined (raw ore is compressed afterwards) and their tiny volume skews ISK/m³.
		  AND COALESCE(json_extract(t.name,'$.en'),'') NOT LIKE '%Compressed%'
		ORDER BY t._key`)
	if err != nil {
		return nil, fmt.Errorf("list ores: %w", err)
	}
	defer func() { _ = rows.Close() }()
	disp, err := oreDisplayNames(db)
	if err != nil {
		return nil, err
	}
	var out []Ore
	for rows.Next() {
		var o Ore
		if err := rows.Scan(&o.TypeID, &o.Name, &o.GroupID, &o.PortionSize, &o.VolumeM3); err != nil {
			return nil, err
		}
		if name, ok := disp[o.TypeID]; ok {
			o.Name = name // real in-game name (Concentrated Veldspar, Vivid Hemorphite, …)
		} else if strings.Contains(o.Name, "-Grade") {
			continue // un-belt-able variant (IV-Grade/0-Grade): not mined, drop it
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// oreDisplayNames maps a raw ore variant typeID to its real in-game name, derived
// from CCP's compression blueprints: a "Compressed <Descriptive> Blueprint" lists
// the raw ore variant as its manufacturing material, so <Descriptive> is that
// ore's display name. Base ores map to their own name (identity). Variants without
// such a blueprint (e.g. IV-Grade / 0-Grade) are absent from the map.
func oreDisplayNames(db *sql.DB) (map[int64]string, error) {
	rows, err := db.Query(`
		SELECT json_extract(b.activities,'$.manufacturing.materials[0].typeID'),
		       replace(replace(json_extract(t.name,'$.en'),'Compressed ',''),' Blueprint','')
		FROM blueprints b
		JOIN types t ON t._key = b.blueprintTypeID
		WHERE json_extract(t.name,'$.en') LIKE 'Compressed %Blueprint'`)
	if err != nil {
		return nil, fmt.Errorf("ore display names: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]string{}
	for rows.Next() {
		var oreID sql.NullInt64
		var name string
		if err := rows.Scan(&oreID, &name); err != nil {
			return nil, err
		}
		if oreID.Valid && name != "" {
			out[oreID.Int64] = name
		}
	}
	return out, rows.Err()
}

// ReprocessingSkills holds the character's reprocessing-relevant skill levels.
type ReprocessingSkills struct {
	Reprocessing           int // +3%/level
	ReprocessingEfficiency int // +2%/level
	OreProcessing          int // ore-specific "<Ore> Processing", +2%/level
}

// NetYield = baseRate × (1+0.03·R) × (1+0.02·RE) × (1+0.02·OP).
// baseRate is the station's reprocessingEfficiency (0.50 for NPC stations).
func NetYield(baseRate float64, s ReprocessingSkills) float64 {
	return baseRate *
		(1 + 0.03*float64(s.Reprocessing)) *
		(1 + 0.02*float64(s.ReprocessingEfficiency)) *
		(1 + 0.02*float64(s.OreProcessing))
}
