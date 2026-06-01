package mining

import (
	"database/sql"
	"fmt"
)

const attrMiningHoldCapacity = 1556

// OreHoldCapacity returns the m³ a ship fills while mining: the dedicated ore
// hold (attr 1556) if present, otherwise the regular cargo (types.capacity).
// found=false when neither is > 0 (the caller marks the row as an estimate).
func OreHoldCapacity(db *sql.DB, shipTypeID int64) (capM3 float64, found bool, err error) {
	attrs, err := typeAttrs(db, shipTypeID)
	if err != nil {
		return 0, false, err
	}
	if v, ok := attrs[attrMiningHoldCapacity]; ok && v > 0 {
		return v, true, nil
	}
	var capacity sql.NullFloat64
	err = db.QueryRow(`SELECT capacity FROM types WHERE _key = ?`, shipTypeID).Scan(&capacity)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("ship capacity %d: %w", shipTypeID, err)
	}
	if capacity.Valid && capacity.Float64 > 0 {
		return capacity.Float64, true, nil
	}
	return 0, false, nil
}

// EffectiveISKPerHour folds the mining cycle into an hourly rate:
//
//	t_fill  = oreHoldM3 / m3h            (hours to fill the hold)
//	cycle   = t_fill + (travelSecs + stopSecs)/3600
//	eff     = (oreHoldM3 * netPerM3) / cycle
//
// travelSecs is the sum of the one-way leg seconds for the path (no return trip);
// stopSecs is the total dock/action overhead. Returns 0,0,0 when the ship cannot
// fill the hold (m3h<=0 or oreHoldM3<=0).
func EffectiveISKPerHour(oreHoldM3, m3h, netPerM3, travelSecs, stopSecs float64) (effISKPerHour, cycleMinutes, fillMinutes float64) {
	if m3h <= 0 || oreHoldM3 <= 0 {
		return 0, 0, 0
	}
	fillHours := oreHoldM3 / m3h
	cycleHours := fillHours + (travelSecs+stopSecs)/3600.0
	if cycleHours <= 0 {
		return 0, 0, 0
	}
	iskLoad := oreHoldM3 * netPerM3
	return iskLoad / cycleHours, cycleHours * 60.0, fillHours * 60.0
}
