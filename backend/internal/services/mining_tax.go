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

// BestStation returns the station with the lowest reprocessing tax, or nil if none.
func BestStation(s []StationStanding) *StationStanding {
	var best *StationStanding
	bestTax := math.Inf(1)
	for i := range s {
		if tax := ReprocessTax(s[i].BaseTake, s[i].Standing); tax < bestTax {
			bestTax, best = tax, &s[i]
		}
	}
	return best
}
