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

// TestFeeService_CalculateBrokerFee tests broker fee calculation with various skill combinations
func TestFeeService_CalculateBrokerFee(t *testing.T) {
	mockSkills := &MockSkillsService{}
	testLogger := logger.NewNoop()
	service := NewFeeService(mockSkills, testLogger)

	tests := []struct {
		name           string
		brokerLvl      int
		advBrokerLvl   int
		standing       float64
		orderValue     float64
		expectedFee    float64
		expectedFeePct float64
	}{
		{
			name:           "No skills, no standing",
			brokerLvl:      0,
			advBrokerLvl:   0,
			standing:       0.0,
			orderValue:     1000000,
			expectedFee:    30000, // 3%
			expectedFeePct: 0.03,
		},
		{
			name:           "Broker Relations V only",
			brokerLvl:      5,
			advBrokerLvl:   0,
			standing:       0.0,
			orderValue:     1000000,
			expectedFee:    15000, // 1.5% (3% - 1.5%)
			expectedFeePct: 0.015,
		},
		{
			name:           "Advanced Broker Relations V only",
			brokerLvl:      0,
			advBrokerLvl:   5,
			standing:       0.0,
			orderValue:     1000000,
			expectedFee:    15000, // 1.5% (3% - 1.5%)
			expectedFeePct: 0.015,
		},
		{
			name:           "Both Broker skills V (max skills)",
			brokerLvl:      5,
			advBrokerLvl:   5,
			standing:       0.0,
			orderValue:     1000000,
			expectedFee:    10000, // 1% (min enforced: 3% - 1.5% - 1.5% = 0%, but min 1%)
			expectedFeePct: 0.01,
		},
		{
			name:           "Both Broker V + 10.0 standing (all bonuses)",
			brokerLvl:      5,
			advBrokerLvl:   5,
			standing:       10.0,
			orderValue:     1000000,
			expectedFee:    10000, // 1% (min enforced even with standing bonus)
			expectedFeePct: 0.01,
		},
		{
			name:           "Partial skills (III + II)",
			brokerLvl:      3,
			advBrokerLvl:   2,
			standing:       0.0,
			orderValue:     1000000,
			expectedFee:    15000, // 1.5% (3% - 0.9% - 0.6% = 1.5%)
			expectedFeePct: 0.015,
		},
		{
			name:           "Standing bonus only (5.0 standing)",
			brokerLvl:      0,
			advBrokerLvl:   0,
			standing:       5.0,
			orderValue:     1000000,
			expectedFee:    28500, // 2.85% (3% - 0.15% standing bonus)
			expectedFeePct: 0.0285,
		},
		{
			name:           "Minimum fee enforcement (100 ISK)",
			brokerLvl:      5,
			advBrokerLvl:   5,
			standing:       10.0,
			orderValue:     5000, // Small order
			expectedFee:    100,  // Min 100 ISK enforced
			expectedFeePct: 0.02, // Would be 1%, but min kicks in
		},
		{
			name:           "Large order (100M ISK)",
			brokerLvl:      5,
			advBrokerLvl:   5,
			standing:       0.0,
			orderValue:     100000000,
			expectedFee:    1000000, // 1% = 1M ISK
			expectedFeePct: 0.01,
		},
		{
			name:           "Negative standing (ignored)",
			brokerLvl:      0,
			advBrokerLvl:   0,
			standing:       -5.0, // Negative standings don't increase fees
			orderValue:     1000000,
			expectedFee:    30000, // 3% (standing ignored)
			expectedFeePct: 0.03,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := service.(*FeeService).CalculateBrokerFee(
				tt.brokerLvl,
				tt.advBrokerLvl,
				tt.standing,
				tt.orderValue,
			)

			// Allow 0.01 ISK tolerance
			if !floatEquals(fee, tt.expectedFee, 0.01) {
				t.Errorf("Expected fee %.2f ISK, got %.2f ISK", tt.expectedFee, fee)
			}

			// Verify percentage (except when minimum kicks in)
			if tt.orderValue >= 10000 { // Min fee doesn't apply
				calculatedPct := fee / tt.orderValue
				if !floatEquals(calculatedPct, tt.expectedFeePct, 0.0001) {
					t.Errorf("Expected fee rate %.4f%%, got %.4f%%",
						tt.expectedFeePct*100, calculatedPct*100)
				}
			}
		})
	}
}

// TestFeeService_CalculateFees tests complete fee calculation with SkillsService integration
func TestFeeService_CalculateFees(t *testing.T) {
	testLogger := logger.NewNoop()

	tests := []struct {
		name               string
		mockSkills         *TradingSkills
		mockError          error
		buyValue           float64
		sellValue          float64
		expectedSalesTax   float64
		expectedBrokerBuy  float64
		expectedBrokerSell float64
	}{
		{
			name: "No skills (worst-case fees)",
			mockSkills: &TradingSkills{
				Accounting:              0,
				BrokerRelations:         0,
				AdvancedBrokerRelations: 0,
				FactionStanding:         0.0,
			},
			buyValue:           1000000,
			sellValue:          1200000,
			expectedSalesTax:   60000, // 5% of 1.2M
			expectedBrokerBuy:  30000, // 3% of 1M
			expectedBrokerSell: 36000, // 3% of 1.2M
		},
		{
			name: "Max trading skills",
			mockSkills: &TradingSkills{
				Accounting:              5,
				BrokerRelations:         5,
				AdvancedBrokerRelations: 5,
				FactionStanding:         10.0,
			},
			buyValue:           1000000,
			sellValue:          1200000,
			expectedSalesTax:   30000, // 2.5% of 1.2M
			expectedBrokerBuy:  10000, // 1% of 1M (min enforced)
			expectedBrokerSell: 12000, // 1% of 1.2M (min enforced)
		},
		{
			name: "Partial skills (realistic scenario)",
			mockSkills: &TradingSkills{
				Accounting:              4,
				BrokerRelations:         3,
				AdvancedBrokerRelations: 2,
				FactionStanding:         5.0,
			},
			buyValue:           5000000,
			sellValue:          6000000,
			expectedSalesTax:   180000, // 3% of 6M (5% - 40% = 3%)
			expectedBrokerBuy:  67500,  // 1.35% of 5M (3% - 0.9% - 0.6% - 0.15% = 1.35%)
			expectedBrokerSell: 81000,  // 1.35% of 6M
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSkillsService := &MockSkillsService{
				GetCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockSkills, nil
				},
			}

			service := NewFeeService(mockSkillsService, testLogger)
			ctx := context.Background()

			fees, err := service.CalculateFees(ctx, 123456, "token", tt.buyValue, tt.sellValue)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify individual fees
			if !floatEquals(fees.SalesTax, tt.expectedSalesTax, 0.01) {
				t.Errorf("Sales Tax: expected %.2f, got %.2f", tt.expectedSalesTax, fees.SalesTax)
			}
			if !floatEquals(fees.BrokerFeeBuy, tt.expectedBrokerBuy, 0.01) {
				t.Errorf("Broker Fee Buy: expected %.2f, got %.2f", tt.expectedBrokerBuy, fees.BrokerFeeBuy)
			}
			if !floatEquals(fees.BrokerFeeSell, tt.expectedBrokerSell, 0.01) {
				t.Errorf("Broker Fee Sell: expected %.2f, got %.2f", tt.expectedBrokerSell, fees.BrokerFeeSell)
			}

			// Verify relist fee (should be brokerFeeSell * 0.5 * 3)
			expectedRelistFee := tt.expectedBrokerSell * 0.5 * 3
			if !floatEquals(fees.EstimatedRelistFee, expectedRelistFee, 0.01) {
				t.Errorf("Relist Fee: expected %.2f, got %.2f", expectedRelistFee, fees.EstimatedRelistFee)
			}

			// Verify total
			expectedTotal := tt.expectedSalesTax + tt.expectedBrokerBuy + tt.expectedBrokerSell + expectedRelistFee
			if !floatEquals(fees.TotalFees, expectedTotal, 0.01) {
				t.Errorf("Total Fees: expected %.2f, got %.2f", expectedTotal, fees.TotalFees)
			}
		})
	}
}

// TestFeeService_CalculateFees_SkillsFallback tests graceful degradation when skills unavailable
func TestFeeService_CalculateFees_SkillsFallback(t *testing.T) {
	testLogger := logger.NewNoop()

	// Mock SkillsService that always fails
	mockSkillsService := &MockSkillsService{
		GetCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
			return nil, context.DeadlineExceeded // Simulate timeout
		},
	}

	service := NewFeeService(mockSkillsService, testLogger)
	ctx := context.Background()

	fees, err := service.CalculateFees(ctx, 123456, "token", 1000000, 1200000)
	if err != nil {
		t.Fatalf("Should not error even when skills unavailable: %v", err)
	}

	// Should use worst-case fees (all skills = 0)
	expectedSalesTax := 60000.0   // 5% of 1.2M
	expectedBrokerBuy := 30000.0  // 3% of 1M
	expectedBrokerSell := 36000.0 // 3% of 1.2M

	if !floatEquals(fees.SalesTax, expectedSalesTax, 0.01) {
		t.Errorf("Sales Tax: expected %.2f (worst-case), got %.2f", expectedSalesTax, fees.SalesTax)
	}
	if !floatEquals(fees.BrokerFeeBuy, expectedBrokerBuy, 0.01) {
		t.Errorf("Broker Fee Buy: expected %.2f (worst-case), got %.2f", expectedBrokerBuy, fees.BrokerFeeBuy)
	}
	if !floatEquals(fees.BrokerFeeSell, expectedBrokerSell, 0.01) {
		t.Errorf("Broker Fee Sell: expected %.2f (worst-case), got %.2f", expectedBrokerSell, fees.BrokerFeeSell)
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
