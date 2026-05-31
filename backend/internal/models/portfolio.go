package models

// PortfolioRequest is the body for POST /api/v1/trading/portfolio/optimize.
type PortfolioRequest struct {
	RegionID        int      `json:"region_id"`
	ShipTypeID      int      `json:"ship_type_id"`
	CargoCapacity   float64  `json:"cargo_capacity,omitempty"` // Optional: selected ship instance's effective cargo (m³). When >0, overrides the per-type fitting recompute.
	Capital         float64  `json:"capital"`                  // ISK budget
	TimeBudgetMin   float64  `json:"time_budget_min"`          // available trading minutes/day
	LiquidityCapPct float64  `json:"liquidity_cap_pct"`        // 0..100, max share of daily volume per item
	MaxItemPct      float64  `json:"max_item_pct"`             // 0..100, max share of capital per item
	SecZones        []string `json:"sec_zones"`                // "high","low","null"
} // @name PortfolioRequest

// PortfolioItem is one allocated position in the suggested portfolio.
//
// ISKPerHour rates the position by time-value: `daily_profit / (time_used/60)`
// where `time_used = trips_per_day * trip_minutes`. Items are pre-sorted by
// this metric desc (= the actually most efficient line on screen).
type PortfolioItem struct {
	TypeID      int     `json:"type_id"`
	Name        string  `json:"name"`
	CapitalUsed float64 `json:"capital_used"`
	Units       int     `json:"units"`
	TripsPerDay float64 `json:"trips_per_day"`
	DailyProfit float64 `json:"daily_profit"`
	ROIPercent  float64 `json:"roi_percent"`  // daily_profit / capital_used * 100
	ISKPerHour  float64 `json:"isk_per_hour"` // net daily profit per hour of haul time
	// Where to execute the haul: buy at the buy station, sell at the sell station.
	BuySystemName   string `json:"buy_system_name"`
	BuyStationName  string `json:"buy_station_name"`
	BuyStationID    int64  `json:"buy_station_id"`
	SellSystemName  string `json:"sell_system_name"`
	SellStationName string `json:"sell_station_name"`
	SellStationID   int64  `json:"sell_station_id"`
} // @name PortfolioItem

// PortfolioResult is the response for POST /api/v1/trading/portfolio/optimize.
type PortfolioResult struct {
	Items                []PortfolioItem `json:"items"`
	TotalCapitalUsed     float64         `json:"total_capital_used"`
	TotalDailyProfit     float64         `json:"total_daily_profit"`
	TimeUsedMin          float64         `json:"time_used_min"`
	DiversificationScore int             `json:"diversification_score"` // 0..100
	SkillsApplied        SkillsApplied   `json:"skills_applied"`
} // @name PortfolioResult
