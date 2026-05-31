package services

import "math"

// ReprocessTax = max(0, baseTake − 0.0075·standing). baseTake is the station's
// reprocessingStationsTake (0.05 for NPC); standing is the player's standing with
// the station's owner corp. Reaches 0 at standing ≈ 6.67 (for baseTake 0.05).
func ReprocessTax(baseTake, standing float64) float64 {
	return math.Max(0, baseTake-0.0075*standing)
}

// StationStanding pairs a reprocessing station with the player's standing to its owner.
type StationStanding struct {
	StationID int64
	BaseRate  float64 // reprocessingEfficiency (0.50)
	BaseTake  float64 // reprocessingStationsTake (0.05)
	Standing  float64
}

// BestStation returns the station that yields the most refined material, i.e. the one
// maximizing baseRate × (1 − tax), or nil if none. (Picking lowest tax alone is wrong when
// stations in the same region differ in reprocessingEfficiency.)
func BestStation(s []StationStanding) *StationStanding {
	var best *StationStanding
	bestScore := math.Inf(-1)
	for i := range s {
		score := s[i].BaseRate * (1 - ReprocessTax(s[i].BaseTake, s[i].Standing))
		if score > bestScore {
			bestScore, best = score, &s[i]
		}
	}
	return best
}
