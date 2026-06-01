package services

import (
	"context"
	"strings"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// locSDE is the SDE subset the sell-location resolver needs.
type locSDE interface {
	GetStationName(ctx context.Context, stationID int64) (string, error)
	GetSystemName(ctx context.Context, systemID int64) (string, error)
	GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error)
}

// citadelIDFloor is the lower bound of Upwell-structure (citadel) item ids; below it are
// NPC station ids. Citadels are not in the SDE, so they resolve to IsStructure only.
const citadelIDFloor = int64(1_000_000_000_000)

// locResolver maps market-order location ids to SellLocation, memoizing within a request
// (many ores' best buys sit at the same regional hub).
type locResolver struct {
	sde   locSDE
	cache map[int64]models.SellLocation
}

func newLocResolver(sde locSDE) *locResolver {
	return &locResolver{sde: sde, cache: map[int64]models.SellLocation{}}
}

// resolve maps a locationID to a SellLocation. NPC stations → station-type name + system;
// citadels / unresolvable → IsStructure (fail-loud: never a fabricated NPC name).
func (r *locResolver) resolve(ctx context.Context, locationID int64) models.SellLocation {
	if loc, ok := r.cache[locationID]; ok {
		return loc
	}
	var loc models.SellLocation
	switch {
	case locationID >= citadelIDFloor:
		loc = models.SellLocation{IsStructure: true}
	default:
		name, err := r.sde.GetStationName(ctx, locationID)
		if err != nil || strings.HasPrefix(name, "Station-") {
			// GetStationName returns "Station-<id>" for non-NPC ids — treat as a structure.
			loc = models.SellLocation{IsStructure: true}
		} else {
			loc.StationName = name
			if sysID, e := r.sde.GetSystemIDForLocation(ctx, locationID); e == nil {
				if sysName, e2 := r.sde.GetSystemName(ctx, sysID); e2 == nil {
					loc.SystemName = sysName
				}
			}
		}
	}
	loc.LocationID = locationID
	r.cache[locationID] = loc
	return loc
}
