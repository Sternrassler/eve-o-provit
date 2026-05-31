// Package reprocessing provides SDE ore catalog and reprocessing output queries.
package reprocessing

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
		ORDER BY t._key`)
	if err != nil {
		return nil, fmt.Errorf("list ores: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Ore
	for rows.Next() {
		var o Ore
		if err := rows.Scan(&o.TypeID, &o.Name, &o.GroupID, &o.PortionSize, &o.VolumeM3); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
