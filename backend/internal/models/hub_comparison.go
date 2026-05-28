package models

// HubComparisonRequest is the body for POST /api/v1/trading/hubs/compare.
type HubComparisonRequest struct {
	TypeID int `json:"type_id" example:"34"`
} // @name HubComparisonRequest

// CompetitionInfo describes the order-update-frequency competition indicator for
// one (item, hub). Source is "live" when derived from snapshot churn, or
// "baseline" when derived from daily order_count until live data exists.
type CompetitionInfo struct {
	ChangesPerHour float64 `json:"changes_per_hour" example:"42.5"`
	Source         string  `json:"source" example:"baseline"` // "live" | "baseline"
} // @name CompetitionInfo

// HubRow is one hub's row in the comparison table for the requested item.
type HubRow struct {
	RegionID         int     `json:"region_id" example:"10000002"`
	RegionName       string  `json:"region_name" example:"The Forge"`
	HubName          string  `json:"hub_name" example:"Jita"`
	SystemID         int     `json:"system_id" example:"30000142"`
	Tier             string  `json:"tier" example:"primary"` // "primary" | "secondary"
	HasData          bool    `json:"has_data" example:"true"`
	BuyPrice         float64 `json:"buy_price" example:"5.50"`
	SellPrice        float64 `json:"sell_price" example:"6.10"`
	SpreadPercent    float64 `json:"spread_percent" example:"10.9"`
	NetMarginPercent float64 `json:"net_margin_percent" example:"3.4"`
	NetProfitPerUnit float64 `json:"net_profit_per_unit" example:"0.18"`
	DailyVolume      float64 `json:"daily_volume" example:"125000"`
	LiquidityScore   int     `json:"liquidity_score" example:"72"`

	Competition CompetitionInfo `json:"competition"`
} // @name HubRow

// SkillsApplied summarizes which character skills were factored into the margins.
type SkillsApplied struct {
	Applied         bool    `json:"applied" example:"true"`
	Accounting      int     `json:"accounting" example:"5"`
	BrokerRelations int     `json:"broker_relations" example:"5"`
	SalesTaxRate    float64 `json:"sales_tax_rate" example:"0.025"`
	BrokerFeeRate   float64 `json:"broker_fee_rate" example:"0.015"`
} // @name SkillsApplied

// HubComparisonResult is the response for POST /api/v1/trading/hubs/compare.
// Hubs are sorted by net margin descending; BestHubRegionID points at the top hub
// that has data.
type HubComparisonResult struct {
	TypeID          int           `json:"type_id" example:"34"`
	ItemName        string        `json:"item_name" example:"Tritanium"`
	Hubs            []HubRow      `json:"hubs"`
	BestHubRegionID int           `json:"best_hub_region_id" example:"10000032"`
	SkillsApplied   SkillsApplied `json:"skills_applied"`
} // @name HubComparisonResult
