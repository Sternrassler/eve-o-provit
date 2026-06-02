package mining

import (
	"database/sql"
	"fmt"
	"math"
)

// Ore group ids (SDE "Asteroid" category).
const (
	grpVeldspar    = 462
	grpScordite    = 460
	grpPlagioclase = 458
	grpPyroxeres   = 459
	grpOmber       = 469
	grpKernite     = 457
	grpJaspet      = 456
	grpHemorphite  = 455
	grpHedbergite  = 454
)

// SystemQuarterAndSec returns a system's empire quarter ("amarr"/"caldari"/
// "gallente"/"minmatar", or "" if not an empire region) and its raw security
// status, from the SDE.
func SystemQuarterAndSec(db *sql.DB, systemID int64) (quarter string, sec float64, err error) {
	var regionID int64
	err = db.QueryRow(
		`SELECT securityStatus, regionID FROM mapSolarSystems WHERE _key = ?`, systemID,
	).Scan(&sec, &regionID)
	if err != nil {
		return "", 0, fmt.Errorf("system %d: %w", systemID, err)
	}
	var factionID sql.NullInt64
	if err = db.QueryRow(`SELECT factionID FROM mapRegions WHERE _key = ?`, regionID).Scan(&factionID); err != nil {
		return "", sec, fmt.Errorf("region %d: %w", regionID, err)
	}
	return factionToQuarter(factionID.Int64), sec, nil
}

func factionToQuarter(factionID int64) string {
	switch factionID {
	case 500003, 500007, 500008: // Amarr, Ammatar, Khanid
		return "amarr"
	case 500001: // Caldari
		return "caldari"
	case 500004: // Gallente
		return "gallente"
	case 500002: // Minmatar
		return "minmatar"
	default:
		return ""
	}
}

// displaySec rounds the raw security status to EVE's displayed 1-decimal value
// (half away from zero); a positive value that rounds to 0.0 shows as 0.1.
func displaySec(sec float64) float64 {
	r := math.Round(sec*10) / 10
	if sec > 0 && r <= 0 {
		return 0.1
	}
	return r
}

// AvailableOreGroups returns the ore groups that spawn in a system, per EVE
// University's documented high-sec/low-sec rules. Empire quarter + displayed
// security decide the set. allowLow gates low-sec systems: in a low-sec system
// with allowLow=false the result is empty (the player won't operate there).
// Null-sec (sec<=0) is out of scope (empty); see issue #161.
//
// High-sec is empire-specific (source: EVE-Uni "Ore" distribution table,
// wiki.eveuniversity.org/Ore, cross-checked against in-game belts): Veldspar +
// Scordite everywhere; Pyroxeres ONLY in Amarr+Caldari high-sec (0.9–0.5);
// Plagioclase in Gallente+Minmatar (0.9–0.5) and Caldari (≤0.7). An unknown
// quarter ("") yields only the quarter-independent Veldspar+Scordite.
func AvailableOreGroups(quarter string, sec float64, allowLow bool) map[int64]bool {
	out := map[int64]bool{}
	d := displaySec(sec)
	switch {
	case d >= 0.5: // high-sec
		out[grpVeldspar] = true
		out[grpScordite] = true
		switch quarter {
		case "amarr":
			if d <= 0.9 {
				out[grpPyroxeres] = true
			}
		case "caldari":
			if d <= 0.9 {
				out[grpPyroxeres] = true
			}
			if d <= 0.7 {
				out[grpPlagioclase] = true
			}
		case "gallente", "minmatar":
			if d <= 0.9 {
				out[grpPlagioclase] = true
			}
		}
	case d > 0 && allowLow: // low-sec, player willing
		switch quarter {
		case "amarr":
			out[grpPyroxeres], out[grpKernite], out[grpJaspet] = true, true, true
			if d <= 0.2 {
				out[grpHemorphite] = true
			}
		case "caldari":
			out[grpKernite], out[grpPyroxeres] = true, true
			if d <= 0.2 {
				out[grpHedbergite] = true
			}
		case "gallente":
			out[grpOmber], out[grpJaspet] = true, true
			if d <= 0.2 {
				out[grpHemorphite] = true
			}
		case "minmatar":
			out[grpKernite], out[grpOmber] = true, true
			if d <= 0.2 {
				out[grpHedbergite] = true
			}
		}
	}
	return out
}
