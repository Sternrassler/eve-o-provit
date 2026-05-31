// Package mining computes deterministic ore-mining yield (m³/hour) from a ship's
// fitted mining modules + the character's mining skills, using the SDE dogma.
package mining

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// MiningSkills holds the yield-relevant mining skill levels.
type MiningSkills struct {
	Mining       int // +5%/level
	Astrogeology int // +5%/level
}

const (
	attrMiningAmount = 77 // m³ per cycle
	attrDuration     = 73 // cycle time, milliseconds
)

// MiningRateM3PerHour sums m³/hour over the given mining module type ids. Modules
// without a miningAmount attribute are skipped (not mining lasers). crystalMul scales
// the per-cycle yield for an ore-specific crystal (1.0 = none). Skills Mining +
// Astrogeology give +5%/level each.
func MiningRateM3PerHour(db *sql.DB, moduleTypeIDs []int64, skills MiningSkills, crystalMul float64) (float64, error) {
	skillMul := (1 + 0.05*float64(skills.Mining)) * (1 + 0.05*float64(skills.Astrogeology))
	var total float64
	for _, tid := range moduleTypeIDs {
		amount, durationMs, ok, err := moduleMiningAttrs(db, tid)
		if err != nil {
			return 0, err
		}
		if !ok || durationMs <= 0 {
			continue
		}
		perCycle := amount * skillMul * crystalMul
		total += perCycle / (durationMs / 1000.0) * 3600.0
	}
	return total, nil
}

func moduleMiningAttrs(db *sql.DB, typeID int64) (amount, durationMs float64, ok bool, err error) {
	var attrsJSON sql.NullString
	err = db.QueryRow(`SELECT dogmaAttributes FROM typeDogma WHERE _key = ?`, typeID).Scan(&attrsJSON)
	if err == sql.ErrNoRows {
		return 0, 0, false, nil
	}
	if err != nil {
		return 0, 0, false, fmt.Errorf("mining attrs %d: %w", typeID, err)
	}
	if !attrsJSON.Valid || attrsJSON.String == "" {
		return 0, 0, false, nil
	}
	var attrs []struct {
		AttributeID int64   `json:"attributeID"`
		Value       float64 `json:"value"`
	}
	if err := json.Unmarshal([]byte(attrsJSON.String), &attrs); err != nil {
		return 0, 0, false, fmt.Errorf("parse dogma %d: %w", typeID, err)
	}
	m := make(map[int64]float64, len(attrs))
	for _, a := range attrs {
		m[a.AttributeID] = a.Value
	}
	amount, hasAmount := m[attrMiningAmount]
	return amount, m[attrDuration], hasAmount, nil
}
