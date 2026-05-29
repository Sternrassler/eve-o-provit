package models

// AssetItem is one aggregated owned stack: a type at a location with a summed quantity.
type AssetItem struct {
	TypeID       int    `json:"type_id"`
	Name         string `json:"name"`
	Quantity     int    `json:"quantity"`
	LocationID   int64  `json:"location_id"`
	LocationName string `json:"location_name"`
	SystemID     int    `json:"system_id"`
	RegionID     int    `json:"region_id"`
	Marketable   bool   `json:"marketable"`
}

// AssetsResponse is the GET /trading/assets payload.
type AssetsResponse struct {
	Assets []AssetItem `json:"assets"`
	Count  int         `json:"count"`
}

// SellOptionsRequest is the POST /trading/assets/sell-options body.
type SellOptionsRequest struct {
	TypeID      int   `json:"type_id"`
	LocationID  int64 `json:"location_id"`
	Quantity    int   `json:"quantity"`
	AvoidLowSec bool  `json:"avoid_low_sec"`
}

// SellOption is one ranked place to sell the item (taker / sell into a buy order).
type SellOption struct {
	Scope         string  `json:"scope"` // "hub" | "current_region"
	RegionID      int     `json:"region_id"`
	RegionName    string  `json:"region_name"`
	StationID     int64   `json:"station_id"`
	StationName   string  `json:"station_name"`
	SystemName    string  `json:"system_name"`
	BuyPrice      float64 `json:"buy_price"`
	UnitNet       float64 `json:"unit_net"`
	TotalNet      float64 `json:"total_net"`
	Jumps         int     `json:"jumps"`
	TravelTimeMin float64 `json:"travel_time_min"`
	SecurityRisk  string  `json:"security_risk"` // "safe" | "caution" | "danger"
	HasData       bool    `json:"has_data"`
}

// SellOptionsResponse is the ranked result for one selected item.
type SellOptionsResponse struct {
	TypeID         int           `json:"type_id"`
	Name           string        `json:"name"`
	Quantity       int           `json:"quantity"`
	OriginSystemID int           `json:"origin_system_id"`
	Best           *SellOption   `json:"best"`
	Options        []SellOption  `json:"options"`
	SkillsApplied  SkillsApplied `json:"skills_applied"`
}
