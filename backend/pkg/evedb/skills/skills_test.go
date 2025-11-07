package skills

import (
	"math"
	"testing"
)

func TestIsTradingSkill(t *testing.T) {
	tests := []struct {
		name   string
		typeID int
		want   bool
	}{
		{"Accounting", TypeIDAccounting, true},
		{"Broker Relations", TypeIDBrokerRelations, true},
		{"Advanced Broker Relations", TypeIDAdvancedBrokerRelations, true},
		{"Trade", TypeIDTrade, true},
		{"Retail", TypeIDRetail, true},
		{"Wholesale", TypeIDWholesale, true},
		{"Tycoon", TypeIDTycoon, true},
		{"Not a trading skill", 9999, false},
		{"Navigation skill", TypeIDNavigation, false},
		{"Cargo skill", TypeIDAmarrIndustrial, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTradingSkill(tt.typeID)
			if got != tt.want {
				t.Errorf("IsTradingSkill(%d) = %v, want %v", tt.typeID, got, tt.want)
			}
		})
	}
}

func TestIsNavigationSkill(t *testing.T) {
	tests := []struct {
		name   string
		typeID int
		want   bool
	}{
		{"Navigation", TypeIDNavigation, true},
		{"Warp Drive Operation", TypeIDWarpDriveOperation, true},
		{"Evasive Maneuvering", TypeIDEvasiveManeuvering, true},
		{"Spaceship Command", TypeIDSpaceshipCommand, true},
		{"Not a navigation skill", 9999, false},
		{"Trading skill", TypeIDAccounting, false},
		{"Cargo skill", TypeIDCaldariIndustrial, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNavigationSkill(tt.typeID)
			if got != tt.want {
				t.Errorf("IsNavigationSkill(%d) = %v, want %v", tt.typeID, got, tt.want)
			}
		})
	}
}

func TestIsCargoSkill(t *testing.T) {
	tests := []struct {
		name   string
		typeID int
		want   bool
	}{
		{"Amarr Industrial", TypeIDAmarrIndustrial, true},
		{"Caldari Industrial", TypeIDCaldariIndustrial, true},
		{"Gallente Industrial", TypeIDGallenteIndustrial, true},
		{"Minmatar Industrial", TypeIDMinmatarIndustrial, true},
		{"Transport Ships", TypeIDTransportShips, true},
		{"Amarr Freighter", TypeIDAmarrFreighter, true},
		{"Caldari Freighter", TypeIDCaldariFreighter, true},
		{"Gallente Freighter", TypeIDGallenteFreighter, true},
		{"Minmatar Freighter", TypeIDMinmatarFreighter, true},
		{"Not a cargo skill", 9999, false},
		{"Trading skill", TypeIDBrokerRelations, false},
		{"Navigation skill", TypeIDWarpDriveOperation, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCargoSkill(tt.typeID)
			if got != tt.want {
				t.Errorf("IsCargoSkill(%d) = %v, want %v", tt.typeID, got, tt.want)
			}
		})
	}
}

func TestGetCargoBonus(t *testing.T) {
	tests := []struct {
		name   string
		typeID int
		level  int
		want   float64
	}{
		{"Amarr Industrial Level 0", TypeIDAmarrIndustrial, 0, 0.0},
		{"Amarr Industrial Level 1", TypeIDAmarrIndustrial, 1, 0.05},
		{"Amarr Industrial Level 3", TypeIDAmarrIndustrial, 3, 0.15},
		{"Amarr Industrial Level 5", TypeIDAmarrIndustrial, 5, 0.25},
		{"Caldari Industrial Level 5", TypeIDCaldariIndustrial, 5, 0.25},
		{"Transport Ships Level 4", TypeIDTransportShips, 4, 0.20},
		{"Amarr Freighter Level 5", TypeIDAmarrFreighter, 5, 0.25},
		{"Not a cargo skill", 9999, 5, 0.0},
		{"Trading skill", TypeIDAccounting, 5, 0.0},
		{"Navigation skill", TypeIDNavigation, 5, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCargoBonus(tt.typeID, tt.level)
			if math.Abs(got-tt.want) > 0.0001 {
				t.Errorf("GetCargoBonus(%d, %d) = %v, want %v", tt.typeID, tt.level, got, tt.want)
			}
		})
	}
}

func TestConstantValues(t *testing.T) {
	// Trading Skills
	if TypeIDAccounting != 16622 {
		t.Errorf("TypeIDAccounting = %d, want 16622", TypeIDAccounting)
	}
	if TypeIDBrokerRelations != 3446 {
		t.Errorf("TypeIDBrokerRelations = %d, want 3446", TypeIDBrokerRelations)
	}
	if TypeIDAdvancedBrokerRelations != 33467 {
		t.Errorf("TypeIDAdvancedBrokerRelations = %d, want 33467", TypeIDAdvancedBrokerRelations)
	}
	if TypeIDTrade != 3443 {
		t.Errorf("TypeIDTrade = %d, want 3443", TypeIDTrade)
	}
	if TypeIDRetail != 3444 {
		t.Errorf("TypeIDRetail = %d, want 3444", TypeIDRetail)
	}
	if TypeIDWholesale != 16596 {
		t.Errorf("TypeIDWholesale = %d, want 16596", TypeIDWholesale)
	}
	if TypeIDTycoon != 18580 {
		t.Errorf("TypeIDTycoon = %d, want 18580", TypeIDTycoon)
	}

	// Navigation Skills
	if TypeIDNavigation != 3449 {
		t.Errorf("TypeIDNavigation = %d, want 3449", TypeIDNavigation)
	}
	if TypeIDWarpDriveOperation != 3453 {
		t.Errorf("TypeIDWarpDriveOperation = %d, want 3453", TypeIDWarpDriveOperation)
	}
	if TypeIDEvasiveManeuvering != 3452 {
		t.Errorf("TypeIDEvasiveManeuvering = %d, want 3452", TypeIDEvasiveManeuvering)
	}
	if TypeIDSpaceshipCommand != 3327 {
		t.Errorf("TypeIDSpaceshipCommand = %d, want 3327", TypeIDSpaceshipCommand)
	}

	// Cargo Skills
	if TypeIDAmarrIndustrial != 3273 {
		t.Errorf("TypeIDAmarrIndustrial = %d, want 3273", TypeIDAmarrIndustrial)
	}
	if TypeIDCaldariIndustrial != 3274 {
		t.Errorf("TypeIDCaldariIndustrial = %d, want 3274", TypeIDCaldariIndustrial)
	}
	if TypeIDGallenteIndustrial != 3275 {
		t.Errorf("TypeIDGallenteIndustrial = %d, want 3275", TypeIDGallenteIndustrial)
	}
	if TypeIDMinmatarIndustrial != 3276 {
		t.Errorf("TypeIDMinmatarIndustrial = %d, want 3276", TypeIDMinmatarIndustrial)
	}
	if TypeIDTransportShips != 19719 {
		t.Errorf("TypeIDTransportShips = %d, want 19719", TypeIDTransportShips)
	}
	if TypeIDAmarrFreighter != 20524 {
		t.Errorf("TypeIDAmarrFreighter = %d, want 20524", TypeIDAmarrFreighter)
	}
	if TypeIDCaldariFreighter != 20525 {
		t.Errorf("TypeIDCaldariFreighter = %d, want 20525", TypeIDCaldariFreighter)
	}
	if TypeIDGallenteFreighter != 20526 {
		t.Errorf("TypeIDGallenteFreighter = %d, want 20526", TypeIDGallenteFreighter)
	}
	if TypeIDMinmatarFreighter != 20527 {
		t.Errorf("TypeIDMinmatarFreighter = %d, want 20527", TypeIDMinmatarFreighter)
	}
}
