// Package models provides data structures for trading operations
package models

import "time"

// TradingRoute represents a profitable trading route
type TradingRoute struct {
	ItemTypeID        int     `json:"item_type_id"`
	ItemName          string  `json:"item_name"`
	BuySystemID       int64   `json:"buy_system_id"`
	BuySystemName     string  `json:"buy_system_name"`
	BuyStationID      int64   `json:"buy_station_id"`
	BuyStationName    string  `json:"buy_station_name"`
	BuyPrice          float64 `json:"buy_price"`
	SellSystemID      int64   `json:"sell_system_id"`
	SellSystemName    string  `json:"sell_system_name"`
	SellStationID     int64   `json:"sell_station_id"`
	SellStationName   string  `json:"sell_station_name"`
	SellPrice         float64 `json:"sell_price"`
	Quantity          int     `json:"quantity"`
	ProfitPerUnit     float64 `json:"profit_per_unit"`
	TotalProfit       float64 `json:"total_profit"`
	SpreadPercent     float64 `json:"spread_percent"`
	TravelTimeSeconds float64 `json:"travel_time_seconds"`
	RoundTripSeconds  float64 `json:"round_trip_seconds"`
	ISKPerHour        float64 `json:"isk_per_hour"`
	Jumps             int     `json:"jumps"`
	ItemVolume        float64 `json:"item_volume"`
}

// RouteCalculationRequest represents the request to calculate trading routes
type RouteCalculationRequest struct {
	RegionID      int     `json:"region_id"`
	ShipTypeID    int     `json:"ship_type_id"`
	CargoCapacity float64 `json:"cargo_capacity,omitempty"`
}

// RouteCalculationResponse represents the response with calculated routes
type RouteCalculationResponse struct {
	RegionID          int            `json:"region_id"`
	RegionName        string         `json:"region_name"`
	ShipTypeID        int            `json:"ship_type_id"`
	ShipName          string         `json:"ship_name"`
	CargoCapacity     float64        `json:"cargo_capacity"`
	CalculationTimeMS int64          `json:"calculation_time_ms"`
	Routes            []TradingRoute `json:"routes"`
	Warning           string         `json:"warning,omitempty"`
}

// ItemPair represents a profitable buy/sell opportunity for an item
type ItemPair struct {
	TypeID        int     `json:"type_id"`
	ItemName      string  `json:"item_name"`
	ItemVolume    float64 `json:"item_volume"`
	BuyStationID  int64   `json:"buy_station_id"`
	BuySystemID   int64   `json:"buy_system_id"`
	BuyPrice      float64 `json:"buy_price"`
	SellStationID int64   `json:"sell_station_id"`
	SellSystemID  int64   `json:"sell_system_id"`
	SellPrice     float64 `json:"sell_price"`
	SpreadPercent float64 `json:"spread_percent"`
}

// CharacterLocation represents character location information
type CharacterLocation struct {
	SolarSystemID   int64   `json:"solar_system_id"`
	SolarSystemName string  `json:"solar_system_name"`
	RegionID        int64   `json:"region_id"`
	RegionName      string  `json:"region_name"`
	StationID       *int64  `json:"station_id,omitempty"`
	StationName     *string `json:"station_name,omitempty"`
	StructureID     *int64  `json:"structure_id,omitempty"`
}

// CharacterShip represents character ship information
type CharacterShip struct {
	ShipTypeID    int64   `json:"ship_type_id"`
	ShipName      string  `json:"ship_name"`
	ShipItemID    int64   `json:"ship_item_id"`
	ShipTypeName  string  `json:"ship_type_name"`
	CargoCapacity float64 `json:"cargo_capacity"`
}

// CharacterAssetShip represents a ship in character assets
type CharacterAssetShip struct {
	ItemID        int64   `json:"item_id"`
	TypeID        int64   `json:"type_id"`
	TypeName      string  `json:"type_name"`
	LocationID    int64   `json:"location_id"`
	LocationName  string  `json:"location_name"`
	LocationFlag  string  `json:"location_flag"`
	CargoCapacity float64 `json:"cargo_capacity"`
	IsSingleton   bool    `json:"is_singleton"`
}

// CharacterShipsResponse represents the response for character ships
type CharacterShipsResponse struct {
	Ships []CharacterAssetShip `json:"ships"`
	Count int                  `json:"count"`
}

// CachedData represents cached market data
type CachedData struct {
	Data      interface{}
	ExpiresAt time.Time
}
