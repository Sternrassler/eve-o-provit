// Package services provides ship-related operations for EVE Online data
package services

import (
	"context"
	"database/sql"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
)

// ShipService provides ship-related operations using SDE database
type ShipService struct {
	sdeDB *sql.DB
}

// NewShipService creates a new ship service
func NewShipService(sdeDB *sql.DB) *ShipService {
	return &ShipService{
		sdeDB: sdeDB,
	}
}

// GetShipCapacities retrieves cargo capacity for a ship type
func (s *ShipService) GetShipCapacities(ctx context.Context, shipTypeID int64) (*ShipCapacities, error) {
	// Call the cargo package function (no skills applied)
	capacities, err := cargo.GetShipCapacities(s.sdeDB, shipTypeID, nil)
	if err != nil {
		return nil, err
	}

	// Convert to our service model
	return &ShipCapacities{
		ShipTypeID:             capacities.ShipTypeID,
		ShipName:               capacities.ShipName,
		BaseCargoHold:          capacities.BaseCargoHold,
		EffectiveCargoHold:     capacities.EffectiveCargoHold,
		BaseTotalCapacity:      capacities.BaseTotalCapacity,
		EffectiveTotalCapacity: capacities.EffectiveTotalCapacity,
		SkillBonus:             capacities.SkillBonus,
		SkillsApplied:          capacities.SkillsApplied,
	}, nil
}
