// Package database - SDE repository wrapper
package database

import (
	"context"
	"database/sql"
	"fmt"
)

// TypeInfo represents basic type information from SDE
type TypeInfo struct {
	TypeID       int     `json:"type_id"`
	Name         string  `json:"name"`
	Volume       float64 `json:"volume"`
	Capacity     float64 `json:"capacity"`
	BasePrice    float64 `json:"base_price"`
	MarketGroup  *int    `json:"market_group_id,omitempty"`
	CategoryID   *int    `json:"category_id,omitempty"`
	CategoryName *string `json:"category_name,omitempty"`
}

// SDERepository provides read-only access to SDE data
type SDERepository struct {
	db *sql.DB
}

// NewSDERepository creates a new SDE repository
func NewSDERepository(db *sql.DB) *SDERepository {
	return &SDERepository{db: db}
}

// GetTypeInfo retrieves type information by ID
func (r *SDERepository) GetTypeInfo(ctx context.Context, typeID int) (*TypeInfo, error) {
	query := `
		SELECT 
			t._key as type_id,
			COALESCE(json_extract(t.name, '$.en'), json_extract(t.name, '$.de'), 'Unknown') as name,
			COALESCE(t.volume, 0) as volume,
			COALESCE(t.capacity, 0) as capacity,
			COALESCE(t.basePrice, 0) as base_price,
			t.marketGroupID,
			g.categoryID,
			COALESCE(json_extract(c.name, '$.en'), json_extract(c.name, '$.de')) as category_name
		FROM types t
		LEFT JOIN groups g ON t.groupID = g._key
		LEFT JOIN categories c ON g.categoryID = c._key
		WHERE t._key = ?
	`

	var info TypeInfo
	err := r.db.QueryRowContext(ctx, query, typeID).Scan(
		&info.TypeID,
		&info.Name,
		&info.Volume,
		&info.Capacity,
		&info.BasePrice,
		&info.MarketGroup,
		&info.CategoryID,
		&info.CategoryName,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("type %d not found", typeID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query type info: %w", err)
	}

	return &info, nil
}

// SearchTypes searches for types by name
func (r *SDERepository) SearchTypes(ctx context.Context, searchTerm string, limit int) ([]TypeInfo, error) {
	query := `
		SELECT 
			t._key as type_id,
			COALESCE(json_extract(t.name, '$.en'), json_extract(t.name, '$.de'), 'Unknown') as name,
			COALESCE(t.volume, 0) as volume,
			COALESCE(t.capacity, 0) as capacity,
			COALESCE(t.basePrice, 0) as base_price,
			t.marketGroupID,
			g.categoryID,
			COALESCE(json_extract(c.name, '$.en'), json_extract(c.name, '$.de')) as category_name
		FROM types t
		LEFT JOIN groups g ON t.groupID = g._key
		LEFT JOIN categories c ON g.categoryID = c._key
		WHERE t.published = 1
		AND (
			json_extract(t.name, '$.en') LIKE '%' || ? || '%'
			OR json_extract(t.name, '$.de') LIKE '%' || ? || '%'
		)
		ORDER BY t.name
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search types: %w", err)
	}
	defer rows.Close()

	var types []TypeInfo
	for rows.Next() {
		var info TypeInfo
		err := rows.Scan(
			&info.TypeID,
			&info.Name,
			&info.Volume,
			&info.Capacity,
			&info.BasePrice,
			&info.MarketGroup,
			&info.CategoryID,
			&info.CategoryName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan type info: %w", err)
		}
		types = append(types, info)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return types, nil
}

// GetSystemIDForLocation retrieves the solar system ID for a given location ID (station/structure)
func (r *SDERepository) GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error) {
	// Try npcStations table first (for NPC stations)
	// Note: Table uses _key for station ID (consistent with SDE schema)
	query := `SELECT solarSystemID FROM npcStations WHERE _key = ?`
	var systemID int64
	err := r.db.QueryRowContext(ctx, query, locationID).Scan(&systemID)
	if err == nil {
		return systemID, nil
	}

	// If not found in npcStations, try mapDenormalize (for structures/citadels)
	if err == sql.ErrNoRows {
		query = `SELECT solarSystemID FROM mapDenormalize WHERE itemID = ? LIMIT 1`
		err = r.db.QueryRowContext(ctx, query, locationID).Scan(&systemID)
		if err == nil {
			return systemID, nil
		}

		// If still not found, check if it's already a system ID
		if locationID >= 30000000 && locationID < 40000000 {
			return locationID, nil
		}

		return 0, fmt.Errorf("location %d not found in SDE", locationID)
	}

	return 0, fmt.Errorf("failed to query system ID for location %d: %w", locationID, err)
}

// GetSystemName retrieves the solar system name by ID
func (r *SDERepository) GetSystemName(ctx context.Context, systemID int64) (string, error) {
	query := `
		SELECT COALESCE(
			json_extract(name, '$.en'),
			json_extract(name, '$.de'),
			'Unknown'
		)
		FROM mapSolarSystems
		WHERE _key = ?
	`
	var name string
	err := r.db.QueryRowContext(ctx, query, systemID).Scan(&name)
	if err == sql.ErrNoRows {
		return fmt.Sprintf("System-%d", systemID), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query system name: %w", err)
	}
	return name, nil
}

// GetStationName retrieves the station name by ID
func (r *SDERepository) GetStationName(ctx context.Context, stationID int64) (string, error) {
	// NPC stations store their name in the types table via typeID
	query := `
		SELECT COALESCE(
			json_extract(t.name, '$.en'),
			json_extract(t.name, '$.de'),
			'Unknown'
		)
		FROM npcStations s
		JOIN types t ON s.typeID = t._key
		WHERE s._key = ?
	`
	var name string
	err := r.db.QueryRowContext(ctx, query, stationID).Scan(&name)
	if err == sql.ErrNoRows {
		// Station not found - could be a structure/citadel
		return fmt.Sprintf("Station-%d", stationID), nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query station name: %w", err)
	}
	return name, nil
}

// GetRegionIDForSystem retrieves the region ID for a given solar system ID
func (r *SDERepository) GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error) {
	query := `
		SELECT c.regionID
		FROM mapSolarSystems s
		JOIN mapConstellations c ON s.constellationID = c._key
		WHERE s._key = ?
	`
	var regionID int
	err := r.db.QueryRowContext(ctx, query, systemID).Scan(&regionID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("system %d not found in SDE", systemID)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query region ID for system %d: %w", systemID, err)
	}
	return regionID, nil
}

// SearchItems searches for published items by name with group information
func (r *SDERepository) SearchItems(ctx context.Context, searchTerm string, limit int) ([]struct {
	TypeID    int
	Name      string
	GroupName string
}, error) {
	query := `
		SELECT 
			t._key as type_id,
			COALESCE(json_extract(t.name, '$.en'), json_extract(t.name, '$.de'), 'Unknown') as name,
			COALESCE(json_extract(g.name, '$.en'), json_extract(g.name, '$.de'), 'Unknown') as group_name
		FROM types t
		JOIN groups g ON t.groupID = g._key
		WHERE t.published = 1
		AND (
			json_extract(t.name, '$.en') LIKE '%' || ? || '%'
			OR json_extract(t.name, '$.de') LIKE '%' || ? || '%'
		)
		ORDER BY json_extract(t.name, '$.en') ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, searchTerm, searchTerm, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search items: %w", err)
	}
	defer rows.Close()

	var results []struct {
		TypeID    int
		Name      string
		GroupName string
	}

	for rows.Next() {
		var item struct {
			TypeID    int
			Name      string
			GroupName string
		}
		err := rows.Scan(&item.TypeID, &item.Name, &item.GroupName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return results, nil
}
