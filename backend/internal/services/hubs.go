package services

// Hub is one of the major EVE trade hubs used by Multi-Hub Comparison (Issue #43).
// IDs verified against the SDE (v_trade_hubs / mapSolarSystems.regionID) 2026-05-28.
type Hub struct {
	Name       string // English hub name, e.g. "Jita"
	SystemID   int    // solar system ID
	RegionID   int    // region ID (ESI market orders are keyed by region)
	RegionName string
	Tier       string // "primary" | "secondary"
}

// MajorHubs is the fixed registry of the five major trade hubs.
// Jita is the primary hub (the one #43 helps traders avoid); the rest are secondary.
var MajorHubs = []Hub{
	{Name: "Jita", SystemID: 30000142, RegionID: 10000002, RegionName: "The Forge", Tier: "primary"},
	{Name: "Amarr", SystemID: 30002187, RegionID: 10000043, RegionName: "Domain", Tier: "secondary"},
	{Name: "Dodixie", SystemID: 30002659, RegionID: 10000032, RegionName: "Sinq Laison", Tier: "secondary"},
	{Name: "Rens", SystemID: 30002510, RegionID: 10000030, RegionName: "Heimatar", Tier: "secondary"},
	{Name: "Hek", SystemID: 30002053, RegionID: 10000042, RegionName: "Metropolis", Tier: "secondary"},
}
