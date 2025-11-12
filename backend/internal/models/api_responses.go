// Package models defines API request/response models for OpenAPI documentation
package models

import "time"

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status" example:"ok"`
	Timestamp string `json:"timestamp" example:"2025-11-12T10:00:00Z"`
} // @name HealthResponse

// VersionResponse represents the version information response
type VersionResponse struct {
	Version   string `json:"version" example:"0.1.0"`
	BuildTime string `json:"build_time,omitempty" example:"2025-11-12T10:00:00Z"`
	GitCommit string `json:"git_commit,omitempty" example:"abc123def"`
} // @name VersionResponse

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error" example:"Invalid request"`
	Message string `json:"message,omitempty" example:"Detailed error message"`
	Code    int    `json:"code,omitempty" example:"400"`
} // @name ErrorResponse

// RegionResponse represents an EVE Online region
type RegionResponse struct {
	RegionID   int64  `json:"region_id" example:"10000002"`
	RegionName string `json:"region_name" example:"The Forge"`
} // @name RegionResponse

// TypeResponse represents an EVE Online item type
type TypeResponse struct {
	TypeID      int64   `json:"type_id" example:"34"`
	TypeName    string  `json:"type_name" example:"Tritanium"`
	GroupID     int64   `json:"group_id" example:"18"`
	CategoryID  int64   `json:"category_id" example:"4"`
	Volume      float64 `json:"volume" example:"0.01"`
	PackagedVol float64 `json:"packaged_volume,omitempty" example:"0.01"`
} // @name TypeResponse

// MarketOrderResponse represents a market order
type MarketOrderResponse struct {
	OrderID      int64     `json:"order_id" example:"123456789"`
	TypeID       int64     `json:"type_id" example:"34"`
	LocationID   int64     `json:"location_id" example:"60003760"`
	SystemID     int64     `json:"system_id" example:"30000142"`
	IsBuyOrder   bool      `json:"is_buy_order" example:"false"`
	Price        float64   `json:"price" example:"5.50"`
	VolumeRemain int64     `json:"volume_remain" example:"100000"`
	VolumeTotal  int64     `json:"volume_total" example:"100000"`
	MinVolume    int64     `json:"min_volume" example:"1"`
	Duration     int       `json:"duration" example:"90"`
	Issued       time.Time `json:"issued" example:"2025-11-12T10:00:00Z"`
	Range        string    `json:"range" example:"region"`
} // @name MarketOrderResponse

// MarketDataStalenessResponse represents market data age information
type MarketDataStalenessResponse struct {
	RegionID       int64     `json:"region_id" example:"10000002"`
	RegionName     string    `json:"region_name" example:"The Forge"`
	LastUpdate     time.Time `json:"last_update" example:"2025-11-12T10:00:00Z"`
	AgeMinutes     int       `json:"age_minutes" example:"15"`
	Status         string    `json:"status" example:"fresh"` // fresh, stale, very_stale
	RefreshAllowed bool      `json:"refresh_allowed" example:"true"`
} // @name MarketDataStalenessResponse

// CharacterInfoResponse represents authenticated character information
type CharacterInfoResponse struct {
	CharacterID   int      `json:"character_id" example:"12345678"`
	CharacterName string   `json:"character_name" example:"John Doe"`
	Scopes        []string `json:"scopes" example:"esi-skills.read_skills.v1,esi-location.read_location.v1"`
	PortraitURL   string   `json:"portrait_url" example:"https://images.evetech.net/characters/12345678/portrait?size=128"`
} // @name CharacterInfoResponse

// CharacterSkillsResponse represents character skills from ESI
type CharacterSkillsResponse struct {
	CharacterID int64            `json:"character_id" example:"12345678"`
	TotalSP     int64            `json:"total_sp" example:"50000000"`
	UnallocSP   int64            `json:"unallocated_sp,omitempty" example:"0"`
	Skills      []CharacterSkill `json:"skills"`
	CachedUntil time.Time        `json:"cached_until,omitempty" example:"2025-11-12T11:00:00Z"`
} // @name CharacterSkillsResponse

// CharacterSkill represents a single trained skill
type CharacterSkill struct {
	SkillID            int64 `json:"skill_id" example:"3340"`
	ActiveSkillLevel   int   `json:"active_skill_level" example:"5"`
	TrainedSkillLevel  int   `json:"trained_skill_level" example:"5"`
	SkillPointsInSkill int64 `json:"skillpoints_in_skill" example:"256000"`
} // @name CharacterSkill

// CharacterLocationResponse represents character location
type CharacterLocationResponse struct {
	SolarSystemID int64  `json:"solar_system_id" example:"30000142"`
	StationID     int64  `json:"station_id,omitempty" example:"60003760"`
	StructureID   int64  `json:"structure_id,omitempty" example:"1000000000001"`
	SystemName    string `json:"system_name,omitempty" example:"Jita"`
} // @name CharacterLocationResponse

// CharacterShipResponse represents character's current ship
type CharacterShipResponse struct {
	ShipTypeID   int64  `json:"ship_type_id" example:"650"`
	ShipItemID   int64  `json:"ship_item_id" example:"1000000000002"`
	ShipName     string `json:"ship_name" example:"Nereus"`
	ShipTypeName string `json:"ship_type_name,omitempty" example:"Nereus"`
} // @name CharacterShipResponse

// ShipAsset represents a ship in character's assets
type ShipAsset struct {
	ItemID       int64  `json:"item_id" example:"1000000000002"`
	TypeID       int64  `json:"type_id" example:"650"`
	LocationID   int64  `json:"location_id" example:"60003760"`
	LocationFlag string `json:"location_flag" example:"Hangar"`
	TypeName     string `json:"type_name,omitempty" example:"Nereus"`
} // @name ShipAsset

// CharacterFittingResponse represents a character's ship fitting with bonuses
type CharacterFittingResponse struct {
	CharacterID     int64     `json:"character_id" example:"12345678"`
	ShipTypeID      int64     `json:"ship_type_id" example:"650"`
	ShipName        string    `json:"ship_name" example:"Nereus"`
	EffectiveCargo  float64   `json:"effective_cargo_m3" example:"9656.9"`
	BaseCargoHold   float64   `json:"base_cargo_hold_m3" example:"2700.0"`
	BaseFuelBay     float64   `json:"base_fuel_bay_m3,omitempty" example:"0.0"`
	WarpSpeed       float64   `json:"warp_speed_au_s" example:"6.87"`
	BaseWarpSpeed   float64   `json:"base_warp_speed_au_s" example:"3.0"`
	InertiaModifier float64   `json:"inertia_modifier,omitempty" example:"1.0"`
	AppliedSkills   []string  `json:"applied_skills,omitempty" example:"Gallente Hauler V,Navigation V"`
	FittedModules   int       `json:"fitted_modules,omitempty" example:"5"`
	CachedUntil     time.Time `json:"cached_until,omitempty" example:"2025-11-12T11:00:00Z"`
} // @name CharacterFittingResponse

// ItemSearchResult represents a single item search result
type ItemSearchResult struct {
	TypeID    int    `json:"type_id" example:"34"`
	Name      string `json:"name" example:"Tritanium"`
	GroupName string `json:"group_name" example:"Mineral"`
} // @name ItemSearchResult

// TradingRouteRequest represents a request to calculate trading routes
type TradingRouteRequest struct {
	RegionID      int64   `json:"region_id" example:"10000002" validate:"required"`
	TypeIDs       []int64 `json:"type_ids" example:"34,35,36" validate:"required,min=1"`
	MaxInvestment float64 `json:"max_investment" example:"1000000000"`
	CargoCapacity float64 `json:"cargo_capacity,omitempty" example:"9656.9"`
	MaxJumps      int     `json:"max_jumps,omitempty" example:"5"`
	CharacterID   int64   `json:"character_id,omitempty" example:"12345678"`
	ShipTypeID    int64   `json:"ship_type_id,omitempty" example:"650"`
} // @name TradingRouteRequest

// TradingRouteResponse represents a calculated trading route
type TradingRouteResponse struct {
	TypeID        int64   `json:"type_id" example:"34"`
	TypeName      string  `json:"type_name" example:"Tritanium"`
	BuyLocation   string  `json:"buy_location" example:"Jita IV - Moon 4 - Caldari Navy Assembly Plant"`
	SellLocation  string  `json:"sell_location" example:"Amarr VIII (Oris) - Emperor Family Academy"`
	BuyPrice      float64 `json:"buy_price" example:"5.50"`
	SellPrice     float64 `json:"sell_price" example:"6.00"`
	Quantity      int64   `json:"quantity" example:"100000"`
	Investment    float64 `json:"investment" example:"550000.00"`
	Revenue       float64 `json:"revenue" example:"600000.00"`
	Profit        float64 `json:"profit" example:"50000.00"`
	ROI           float64 `json:"roi_percent" example:"9.09"`
	ProfitPerJump float64 `json:"profit_per_jump,omitempty" example:"10000.00"`
	Jumps         int     `json:"jumps,omitempty" example:"5"`
} // @name TradingRouteResponse
