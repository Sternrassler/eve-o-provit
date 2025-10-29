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
