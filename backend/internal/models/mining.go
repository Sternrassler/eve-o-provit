package models

// OreRankingRequest is the input for the ore-ranking endpoint. RegionID <= 0 means
// "use the character's current region". AllowLowSec controls whether low-sec ores
// are included in the ranking (true = willing to operate in low-sec too).
type OreRankingRequest struct {
	RegionID    int  `json:"region_id"`
	AllowLowSec bool `json:"allow_low_sec"` // true = willing to operate in low-sec too
}

// SellLocation describes where a market order sits. NPC stations resolve to a name +
// system; citadels (not in the SDE) carry only IsStructure (UI shows "Player-Structure",
// the region is already known since the ranking is region-scoped).
type SellLocation struct {
	StationName string `json:"station_name,omitempty"`
	SystemName  string `json:"system_name,omitempty"`
	IsStructure bool   `json:"is_structure"`
	LocationID  int64  `json:"location_id,omitempty"` // station/structure id — in-game waypoint target
}

// RefineMaterial is one reprocessing output mineral with where to sell it.
// EffectiveQty is the amount you actually get per portionSize batch (qty × net yield).
type RefineMaterial struct {
	MaterialTypeID int64        `json:"material_type_id"`
	MaterialName   string       `json:"material_name"`
	EffectiveQty   int64        `json:"effective_qty"`
	BuyPrice       float64      `json:"buy_price"`
	Sell           SellLocation `json:"sell"`
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
	// Where to act:
	BestStationName   string           `json:"best_station_name,omitempty"`   // reprocess station (NPC type name)
	BestStationSystem string           `json:"best_station_system,omitempty"` // reprocess station system
	RawSell           *SellLocation    `json:"raw_sell,omitempty"`            // where to sell the raw ore
	Materials         []RefineMaterial `json:"materials,omitempty"`           // per-mineral sell breakdown
	// Yield accuracy (hull role/skill bonus + best-case ore crystal):
	HullYieldMultiplier float64 `json:"hull_yield_multiplier,omitempty"` // hull-wide (same on every row); 1.0 = none
	CrystalMultiplier   float64 `json:"crystal_multiplier,omitempty"`    // per ore; 1.0 = no crystals used
	IsEstimate          bool    `json:"is_estimate,omitempty"`           // hull bonus / crystal not fully resolved
	EstimateReason      string  `json:"estimate_reason,omitempty"`       // short reason, only when IsEstimate
	// Haul-downtime cycle (effective ISK/h, greedy one cycle from current system):
	EffectiveISKPerHour float64 `json:"effective_isk_per_hour,omitempty"` // sort key; 0 when not resolvable
	LoadVolumeM3        float64 `json:"load_volume_m3,omitempty"`         // ore-hold m³ filled per load
	MarketLoads         float64 `json:"market_loads,omitempty"`           // full ore-hold loads the Best path's best reachable order(s) can absorb (VolumeRemain cap); 0<x<1 means the best price won't even take one load
	FillMinutes         float64 `json:"fill_minutes,omitempty"`           // time to fill the hold
	CycleMinutes        float64 `json:"cycle_minutes,omitempty"`          // fill + legs + stops
	RouteJumps          int     `json:"route_jumps,omitempty"`            // jumps over the cycle's legs
	SellSystemName      string  `json:"sell_system_name,omitempty"`       // chosen sell hub (refine) / ore-sell system (raw)
}

// OreRankingResponse is the ore-ranking result for a region + current system context.
// NoMiningSetup is true when the active ship has no mining modules (per-m³ values are
// still populated; isk/h is 0). Rows is always non-nil (never JSON null).
type OreRankingResponse struct {
	RegionID           int          `json:"region_id"`
	OriginSystemID     int64        `json:"origin_system_id,omitempty"`    // character's current system (ranking origin)
	OriginSystemName   string       `json:"origin_system_name,omitempty"`  // resolved name of the current system
	OriginStationName  string       `json:"origin_station_name,omitempty"` // docked station name, if any
	SystemSecurity     float64      `json:"system_security,omitempty"`     // current system, displayed sec
	Quarter            string       `json:"quarter,omitempty"`             // amarr|caldari|gallente|minmatar|""
	NoMiningSetup      bool         `json:"no_mining_setup"`
	NotAvailableReason string       `json:"not_available_reason,omitempty"` // set when no ores apply here
	Rows               []OreRankRow `json:"rows"`
}
