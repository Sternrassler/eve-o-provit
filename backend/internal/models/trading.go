// Package models provides data structures for trading operations
package models

import "time"

// TradingRoute represents a profitable trading route
type TradingRoute struct {
	ItemTypeID             int     `json:"item_type_id"`
	ItemName               string  `json:"item_name"`
	BuySystemID            int64   `json:"buy_system_id"`
	BuySystemName          string  `json:"buy_system_name"`
	BuyStationID           int64   `json:"buy_station_id"`
	BuyStationName         string  `json:"buy_station_name"`
	BuyPrice               float64 `json:"buy_price"`
	SellSystemID           int64   `json:"sell_system_id"`
	SellSystemName         string  `json:"sell_system_name"`
	SellStationID          int64   `json:"sell_station_id"`
	SellStationName        string  `json:"sell_station_name"`
	SellPrice              float64 `json:"sell_price"`
	BuySecurityStatus      float64 `json:"buy_security_status"`
	SellSecurityStatus     float64 `json:"sell_security_status"`
	MinRouteSecurityStatus float64 `json:"min_route_security_status"` // Minimum security of all systems on route
	Quantity               int     `json:"quantity"`
	ProfitPerUnit          float64 `json:"profit_per_unit"`
	TotalProfit            float64 `json:"total_profit"`
	SpreadPercent          float64 `json:"spread_percent"`
	TravelTimeSeconds      float64 `json:"travel_time_seconds"`
	RoundTripSeconds       float64 `json:"round_trip_seconds"`
	ISKPerHour             float64 `json:"isk_per_hour"`
	Jumps                  int     `json:"jumps"`
	ItemVolume             float64 `json:"item_volume"`
	// Multi-tour fields
	NumberOfTours    int     `json:"number_of_tours"`
	ProfitPerTour    float64 `json:"profit_per_tour"`
	TotalTimeMinutes float64 `json:"total_time_minutes"`
	// Navigation Skills fields
	BaseTravelTimeSeconds    float64 `json:"base_travel_time_seconds"`    // Travel time without navigation skills
	SkilledTravelTimeSeconds float64 `json:"skilled_travel_time_seconds"` // Travel time with navigation skills applied
	BaseISKPerHour           float64 `json:"base_isk_per_hour"`           // ISK/h without navigation skills
	TimeImprovementPercent   float64 `json:"time_improvement_percent"`    // Percentage improvement from skills
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
	TypeID            int     `json:"type_id"`
	ItemName          string  `json:"item_name"`
	ItemVolume        float64 `json:"item_volume"`
	BuyStationID      int64   `json:"buy_station_id"`
	BuySystemID       int64   `json:"buy_system_id"`
	BuyPrice          float64 `json:"buy_price"`
	SellStationID     int64   `json:"sell_station_id"`
	SellSystemID      int64   `json:"sell_system_id"`
	SellPrice         float64 `json:"sell_price"`
	SpreadPercent     float64 `json:"spread_percent"`
	AvailableVolumeM3 float64 `json:"available_volume_m3"` // Total m³ available from sell orders
	AvailableQuantity int     `json:"available_quantity"`  // Total items available
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

// Region represents an EVE Online region
type Region struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// RegionsResponse represents the response for regions list
type RegionsResponse struct {
	Regions []Region `json:"regions"`
	Count   int      `json:"count"`
}

// ItemSearchResult represents a search result for items
type ItemSearchResult struct {
	TypeID    int    `json:"type_id"`
	Name      string `json:"name"`
	GroupName string `json:"group_name"`
}

// InventorySellRequest represents a request to find profitable sell routes for inventory
type InventorySellRequest struct {
	TypeID           int     `json:"type_id"`
	Quantity         int     `json:"quantity"`
	BuyPricePerUnit  float64 `json:"buy_price_per_unit"`
	RegionID         int     `json:"region_id"`
	MinProfitPerUnit float64 `json:"min_profit_per_unit"`
	SecurityFilter   string  `json:"security_filter"` // "all", "highsec", "highlow"
}

// Validate validates the inventory sell request parameters
func (r *InventorySellRequest) Validate() error {
	if r.TypeID <= 0 {
		return &ValidationError{Field: "type_id", Message: "Invalid type_id"}
	}
	if r.Quantity <= 0 {
		return &ValidationError{Field: "quantity", Message: "Invalid quantity"}
	}
	if r.BuyPricePerUnit <= 0 {
		return &ValidationError{Field: "buy_price_per_unit", Message: "Invalid buy_price_per_unit"}
	}
	if r.RegionID <= 0 {
		return &ValidationError{Field: "region_id", Message: "Invalid region_id"}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ShipType represents a ship with its navigation characteristics
type ShipType struct {
	TypeID        int     // EVE Online Type ID
	Name          string  // Ship name
	BaseWarpSpeed float64 // Base warp speed in AU/s (typically 3.0 for haulers)
	BaseAlignTime float64 // Base align time in seconds
	CargoCapacity float64 // Cargo capacity in m³
}

// CommonHaulers defines commonly used hauler ships with their base stats
// Warp speeds are in AU/s, align times in seconds
var CommonHaulers = map[int]ShipType{
	648: { // Iteron Mark V (Gallente)
		TypeID:        648,
		Name:          "Iteron Mark V",
		BaseWarpSpeed: 3.0,
		BaseAlignTime: 8.5,
		CargoCapacity: 63000,
	},
	649: { // Badger (Caldari)
		TypeID:        649,
		Name:          "Badger",
		BaseWarpSpeed: 3.0,
		BaseAlignTime: 7.2,
		CargoCapacity: 50000,
	},
	650: { // Nereus (Gallente)
		TypeID:        650,
		Name:          "Nereus",
		BaseWarpSpeed: 3.0,
		BaseAlignTime: 6.8,
		CargoCapacity: 45000,
	},
	651: { // Wreathe (Minmatar)
		TypeID:        651,
		Name:          "Wreathe",
		BaseWarpSpeed: 3.0,
		BaseAlignTime: 8.0,
		CargoCapacity: 48000,
	},
	// Default hauler (used as fallback)
	0: {
		TypeID:        0,
		Name:          "Generic Hauler",
		BaseWarpSpeed: 3.0,
		BaseAlignTime: 8.0,
		CargoCapacity: 50000,
	},
}

// GetShipType returns ship type by ID, or default hauler if not found
func GetShipType(typeID int) ShipType {
	if ship, ok := CommonHaulers[typeID]; ok {
		return ship
	}
	return CommonHaulers[0] // Return default hauler
}

// InventorySellRoute represents a profitable sell opportunity for inventory
type InventorySellRoute struct {
	SellStationID          int64   `json:"sell_station_id"`
	SellStationName        string  `json:"sell_station_name"`
	SellSystemID           int64   `json:"sell_system_id"`
	SellSystemName         string  `json:"sell_system_name"`
	SellSecurityStatus     float64 `json:"sell_security_status"`
	BuyOrderPrice          float64 `json:"buy_order_price"`
	TaxRate                float64 `json:"tax_rate"`
	NetPricePerUnit        float64 `json:"net_price_per_unit"`
	ProfitPerUnit          float64 `json:"profit_per_unit"`
	AvailableQuantity      int     `json:"available_quantity"`
	TotalProfit            float64 `json:"total_profit"`
	RouteJumps             int     `json:"route_jumps"`
	RouteSystemIDs         []int64 `json:"route_system_ids"`
	MinRouteSecurityStatus float64 `json:"min_route_security_status"`
}
