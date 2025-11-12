// Package skills provides EVE Online skill type ID constants and helper functions
// for trading, navigation, cargo, and social skills.
package skills

// Trading Skills
const (
	TypeIDAccounting              = 16622
	TypeIDBrokerRelations         = 3446
	TypeIDAdvancedBrokerRelations = 33467
	TypeIDTrade                   = 3443
	TypeIDRetail                  = 3444
	TypeIDWholesale               = 16596
	TypeIDTycoon                  = 18580
)

// Navigation Skills
const (
	TypeIDNavigation         = 3449 // Navigation skill: +5% warp speed per level
	TypeIDWarpDriveOperation = 3453
	TypeIDEvasiveManeuvering = 3452
	TypeIDSpaceshipCommand   = 3327
)

// Social Skills (Standings)
const (
	TypeIDConnections         = 3359
	TypeIDDiplomacy           = 3357
	TypeIDCriminalConnections = 3361
)

// Cargo Skills (for profit calculations)
const (
	TypeIDAmarrIndustrial    = 3273
	TypeIDCaldariIndustrial  = 3274
	TypeIDGallenteIndustrial = 3275
	TypeIDMinmatarIndustrial = 3276
	TypeIDTransportShips     = 19719

	// Freighters
	TypeIDAmarrFreighter    = 20524
	TypeIDCaldariFreighter  = 20525
	TypeIDGallenteFreighter = 20526
	TypeIDMinmatarFreighter = 20527
)

// IsTradingSkill returns true if the given type ID is a trading skill
func IsTradingSkill(typeID int) bool {
	switch typeID {
	case TypeIDAccounting,
		TypeIDBrokerRelations,
		TypeIDAdvancedBrokerRelations,
		TypeIDTrade,
		TypeIDRetail,
		TypeIDWholesale,
		TypeIDTycoon:
		return true
	}
	return false
}

// IsNavigationSkill returns true if the given type ID is a navigation skill
func IsNavigationSkill(typeID int) bool {
	switch typeID {
	case TypeIDNavigation,
		TypeIDWarpDriveOperation,
		TypeIDEvasiveManeuvering,
		TypeIDSpaceshipCommand:
		return true
	}
	return false
}

// IsCargoSkill returns true if the given type ID is a cargo skill
func IsCargoSkill(typeID int) bool {
	switch typeID {
	case TypeIDAmarrIndustrial,
		TypeIDCaldariIndustrial,
		TypeIDGallenteIndustrial,
		TypeIDMinmatarIndustrial,
		TypeIDTransportShips,
		TypeIDAmarrFreighter,
		TypeIDCaldariFreighter,
		TypeIDGallenteFreighter,
		TypeIDMinmatarFreighter:
		return true
	}
	return false
}

// IsSocialSkill returns true if the given type ID is a social skill
func IsSocialSkill(typeID int) bool {
	switch typeID {
	case TypeIDConnections,
		TypeIDDiplomacy,
		TypeIDCriminalConnections:
		return true
	}
	return false
}

// GetCargoBonus calculates the cargo capacity bonus for a given skill and level.
// Returns 0.0 if the typeID is not a cargo skill.
// For cargo skills, returns 5% per level (0.05 * level).
func GetCargoBonus(typeID int, level int) float64 {
	if !IsCargoSkill(typeID) {
		return 0.0
	}
	return 0.05 * float64(level) // +5% per level
}
