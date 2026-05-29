package models

// HaulingRequest is the body for POST /api/v1/trading/hauling/routes.
type HaulingRequest struct {
	OriginRegionID int     `json:"origin_region_id"` // 0 = use character's current region
	ShipTypeID     int     `json:"ship_type_id"`
	Capital        float64 `json:"capital"`
	AvoidLowSec    bool    `json:"avoid_low_sec"`
	MaxRoutes      int     `json:"max_routes"` // 0 = default 15
} // @name HaulingRequest

// HaulingItem is one item in a route's optimal cargo load.
type HaulingItem struct {
	TypeID      int     `json:"type_id"`
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	BuyPrice    float64 `json:"buy_price"`
	SellPrice   float64 `json:"sell_price"`
	UnitVolume  float64 `json:"unit_volume"`
	Profit      float64 `json:"profit"` // net profit for this position
	ProfitPerM3 float64 `json:"profit_per_m3"`
} // @name HaulingItem

// HaulingRoute is one buy-station → sell-station haul with its optimal load.
type HaulingRoute struct {
	BuySystemName   string        `json:"buy_system_name"`
	BuyStationName  string        `json:"buy_station_name"`
	BuyStationID    int64         `json:"buy_station_id"`
	SellSystemName  string        `json:"sell_system_name"`
	SellStationName string        `json:"sell_station_name"`
	SellStationID   int64         `json:"sell_station_id"`
	Jumps           int           `json:"jumps"`
	TravelTimeMin   float64       `json:"travel_time_min"`
	SecurityRisk    string        `json:"security_risk"` // "safe" | "caution" | "danger"
	TotalProfit     float64       `json:"total_profit"`
	TotalCapital    float64       `json:"total_capital"`
	TotalVolume     float64       `json:"total_volume"`
	ISKPerHour      float64       `json:"isk_per_hour"`
	Items           []HaulingItem `json:"items"`
} // @name HaulingRoute

// HaulingResponse is the response for POST /api/v1/trading/hauling/routes.
type HaulingResponse struct {
	OriginRegionID int            `json:"origin_region_id"`
	RegionsScanned []int          `json:"regions_scanned"`
	Routes         []HaulingRoute `json:"routes"`
	SkillsApplied  SkillsApplied  `json:"skills_applied"`
} // @name HaulingResponse
