// Package esi - Interface definitions for ESI client
package esi

import (
	"context"
	"time"
)

// PriceHistoryEntry holds a single day's market history data from ESI.
// This type lives in pkg/esi to decouple the ESI client from the database layer.
// The service layer maps this to database.PriceHistory.
type PriceHistoryEntry struct {
	Date       time.Time
	Highest    *float64
	Lowest     *float64
	Average    *float64
	Volume     *int64
	OrderCount *int
}

// AutopilotWaypointSetter defines the interface for setting autopilot waypoints via ESI
type AutopilotWaypointSetter interface {
	// SetAutopilotWaypoint sets a waypoint in the EVE client via ESI
	// Returns error with specific messages: "unauthorized", "not_found", or generic error
	SetAutopilotWaypoint(ctx context.Context, accessToken string, destinationID int64, clearOther, addToBeginning bool) error
}

// CharacterLocationGetter defines the interface for getting character location via ESI
type CharacterLocationGetter interface {
	// GetCharacterLocation fetches character location from ESI
	GetCharacterLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocationResponse, error)
}

// CharacterLocationResponse represents the ESI location response
type CharacterLocationResponse struct {
	SolarSystemID int64  `json:"solar_system_id"`
	StationID     *int64 `json:"station_id,omitempty"`
	StructureID   *int64 `json:"structure_id,omitempty"`
}
