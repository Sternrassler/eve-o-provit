package models

// OreRankingRequest is the input for the ore-ranking endpoint. RegionID <= 0 means
// "use the character's current region". SecBand filters the ore set to a security band
// ("high" | "low" | "null"); empty/unknown band yields no ores.
type OreRankingRequest struct {
	RegionID int    `json:"region_id"`
	SecBand  string `json:"sec_band"`
}

// OreRankRow is one ore's raw-vs-refine ranking row. Per-m³ and isk/h figures are
// internal (used for sorting); only the user-facing fields are serialized.
type OreRankRow struct {
	OreTypeID        int64   `json:"ore_type_id"`
	OreName          string  `json:"ore_name"`
	MiningM3PerHour  float64 `json:"mining_m3_per_hour"`
	RawISKPerHour    float64 `json:"raw_isk_per_hour"`
	RefineISKPerHour float64 `json:"refine_isk_per_hour"`
	RawNetPerM3      float64 `json:"raw_net_per_m3"`
	RefineNetPerM3   float64 `json:"refine_net_per_m3"`
	Best             string  `json:"best"`
	DeltaISKPerHour  float64 `json:"delta_isk_per_hour"`
	BestStationID    int64   `json:"best_station_id,omitempty"`
	BestStationTax   float64 `json:"best_station_tax"`
}

// OreRankingResponse is the ore-ranking result for a region + security band.
// NoMiningSetup is true when the active ship has no mining modules (per-m³ values are
// still populated; isk/h is 0). Rows is always non-nil (never JSON null).
type OreRankingResponse struct {
	RegionID      int          `json:"region_id"`
	SecBand       string       `json:"sec_band"`
	NoMiningSetup bool         `json:"no_mining_setup"`
	Rows          []OreRankRow `json:"rows"`
}
