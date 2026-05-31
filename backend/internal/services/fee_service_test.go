package services

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// MockSkillsService for testing FeeService
type MockSkillsService struct {
	GetCharacterSkillsFunc func(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error)
}

func (m *MockSkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
	if m.GetCharacterSkillsFunc != nil {
		return m.GetCharacterSkillsFunc(ctx, characterID, accessToken)
	}
	// Default: no skills
	return &TradingSkills{
		Accounting:              0,
		BrokerRelations:         0,
		AdvancedBrokerRelations: 0,
		FactionStanding:         0.0,
		CorpStanding:            0.0,
	}, nil
}

// TestFeeService_CalculateSalesTax tests sales tax calculation with various Accounting skill levels
func TestFeeService_CalculateSalesTax(t *testing.T) {
	mockSkills := &MockSkillsService{}
	testLogger := logger.NewNoop() // Use noop logger for tests
	service := NewFeeService(mockSkills, testLogger)

	tests := []struct {
		name           string
		accountingLvl  int
		orderValue     float64
		expectedTax    float64
		expectedTaxPct float64
	}{
		{
			name:           "No Accounting skill (Level 0)",
			accountingLvl:  0,
			orderValue:     1000000, // 1M ISK
			expectedTax:    50000,   // 5%
			expectedTaxPct: 0.05,
		},
		{
			name:           "Accounting I",
			accountingLvl:  1,
			orderValue:     1000000,
			expectedTax:    45000, // 4.5%
			expectedTaxPct: 0.045,
		},
		{
			name:           "Accounting II",
			accountingLvl:  2,
			orderValue:     1000000,
			expectedTax:    40000, // 4%
			expectedTaxPct: 0.04,
		},
		{
			name:           "Accounting III",
			accountingLvl:  3,
			orderValue:     1000000,
			expectedTax:    35000, // 3.5%
			expectedTaxPct: 0.035,
		},
		{
			name:           "Accounting IV",
			accountingLvl:  4,
			orderValue:     1000000,
			expectedTax:    30000, // 3%
			expectedTaxPct: 0.03,
		},
		{
			name:           "Accounting V (max)",
			accountingLvl:  5,
			orderValue:     1000000,
			expectedTax:    25000, // 2.5%
			expectedTaxPct: 0.025,
		},
		{
			name:           "Minimum fee enforcement (100 ISK)",
			accountingLvl:  5,
			orderValue:     1000, // Very small order
			expectedTax:    100,  // Min 100 ISK enforced
			expectedTaxPct: 0.10, // Would be 2.5%, but min kicks in
		},
		{
			name:           "Large order (100M ISK)",
			accountingLvl:  5,
			orderValue:     100000000,
			expectedTax:    2500000, // 2.5% = 2.5M ISK
			expectedTaxPct: 0.025,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tax := service.(*FeeService).CalculateSalesTax(tt.accountingLvl, tt.orderValue)

			// Allow 0.01 ISK tolerance for floating point
			if !floatEquals(tax, tt.expectedTax, 0.01) {
				t.Errorf("Expected tax %.2f ISK, got %.2f ISK", tt.expectedTax, tax)
			}

			// Verify percentage (except when minimum kicks in)
			if tt.orderValue >= 4000 { // Min fee doesn't apply above 4000 ISK
				calculatedPct := tax / tt.orderValue
				if !floatEquals(calculatedPct, tt.expectedTaxPct, 0.0001) {
					t.Errorf("Expected tax rate %.4f%%, got %.4f%%",
						tt.expectedTaxPct*100, calculatedPct*100)
				}
			}
		})
	}
}

// floatEquals checks if two floats are equal within a tolerance
func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
