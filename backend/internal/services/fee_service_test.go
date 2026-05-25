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

// TestFeeService_CalculateBrokerFee tests broker fee calculation with various skill combinations
func TestFeeService_CalculateBrokerFee(t *testing.T) {
	mockSkills := &MockSkillsService{}
	testLogger := logger.NewNoop()
	service := NewFeeService(mockSkills, testLogger)

	tests := []struct {
		name            string
		brokerLvl       int
		advBrokerLvl    int
		factionStanding float64
		corpStanding    float64
		orderValue      float64
		expectedFee     float64
		expectedFeePct  float64
	}{
		{
			name:            "No skills, no standing",
			brokerLvl:       0,
			advBrokerLvl:    0,
			factionStanding: 0.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     30000, // 3%
			expectedFeePct:  0.03,
		},
		{
			name:            "Broker Relations V only",
			brokerLvl:       5,
			advBrokerLvl:    0,
			factionStanding: 0.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     15000, // 1.5% (3% - 1.5%)
			expectedFeePct:  0.015,
		},
		{
			name:            "Advanced Broker Relations V only",
			brokerLvl:       0,
			advBrokerLvl:    5,
			factionStanding: 0.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     15000, // 1.5% (3% - 1.5%)
			expectedFeePct:  0.015,
		},
		{
			name:            "Both Broker skills V (max skills)",
			brokerLvl:       5,
			advBrokerLvl:    5,
			factionStanding: 0.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     10000, // 1% (min enforced: 3% - 1.5% - 1.5% = 0%, but min 1%)
			expectedFeePct:  0.01,
		},
		{
			name:            "Both Broker V + max faction standing (10.0)",
			brokerLvl:       5,
			advBrokerLvl:    5,
			factionStanding: 10.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     10000, // 1% (min enforced: 3% - 3% - 0.3% = -0.3%, but min 1%)
			expectedFeePct:  0.01,
		},
		{
			name:            "Both Broker V + max corp standing (10.0)",
			brokerLvl:       5,
			advBrokerLvl:    5,
			factionStanding: 0.0,
			corpStanding:    10.0,
			orderValue:      1000000,
			expectedFee:     10000, // 1% (min enforced: 3% - 3% - 0.2% = -0.2%, but min 1%)
			expectedFeePct:  0.01,
		},
		{
			name:            "Both Broker V + max faction + max corp (all bonuses)",
			brokerLvl:       5,
			advBrokerLvl:    5,
			factionStanding: 10.0,
			corpStanding:    10.0,
			orderValue:      1000000,
			expectedFee:     10000, // 1% (min enforced: 3% - 3% - 0.3% - 0.2% = -0.5%, but min 1%)
			expectedFeePct:  0.01,
		},
		{
			name:            "Partial skills (III + II)",
			brokerLvl:       3,
			advBrokerLvl:    2,
			factionStanding: 0.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     15000, // 1.5% (3% - 0.9% - 0.6% = 1.5%)
			expectedFeePct:  0.015,
		},
		{
			name:            "Faction standing bonus only (5.0)",
			brokerLvl:       0,
			advBrokerLvl:    0,
			factionStanding: 5.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     28500, // 2.85% (3% - 0.15%)
			expectedFeePct:  0.0285,
		},
		{
			name:            "Corp standing bonus only (5.0)",
			brokerLvl:       0,
			advBrokerLvl:    0,
			factionStanding: 0.0,
			corpStanding:    5.0,
			orderValue:      1000000,
			expectedFee:     29000, // 2.9% (3% - 0.1%)
			expectedFeePct:  0.029,
		},
		{
			name:            "Both standings (faction 5.0, corp 5.0)",
			brokerLvl:       0,
			advBrokerLvl:    0,
			factionStanding: 5.0,
			corpStanding:    5.0,
			orderValue:      1000000,
			expectedFee:     27500, // 2.75% (3% - 0.15% - 0.1%)
			expectedFeePct:  0.0275,
		},
		{
			name:            "Minimum fee enforcement (100 ISK)",
			brokerLvl:       5,
			advBrokerLvl:    5,
			factionStanding: 10.0,
			corpStanding:    10.0,
			orderValue:      5000, // Small order
			expectedFee:     100,  // Min 100 ISK enforced
			expectedFeePct:  0.02, // Would be 1%, but min kicks in
		},
		{
			name:            "Large order (100M ISK)",
			brokerLvl:       5,
			advBrokerLvl:    5,
			factionStanding: 0.0,
			corpStanding:    0.0,
			orderValue:      100000000,
			expectedFee:     1000000, // 1% = 1M ISK
			expectedFeePct:  0.01,
		},
		{
			name:            "Negative faction standing (ignored)",
			brokerLvl:       0,
			advBrokerLvl:    0,
			factionStanding: -5.0,
			corpStanding:    0.0,
			orderValue:      1000000,
			expectedFee:     30000, // 3% (standing ignored)
			expectedFeePct:  0.03,
		},
		{
			name:            "Negative corp standing (ignored)",
			brokerLvl:       0,
			advBrokerLvl:    0,
			factionStanding: 0.0,
			corpStanding:    -5.0,
			orderValue:      1000000,
			expectedFee:     30000, // 3% (standing ignored)
			expectedFeePct:  0.03,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := service.(*FeeService).CalculateBrokerFee(
				tt.brokerLvl,
				tt.advBrokerLvl,
				tt.factionStanding,
				tt.corpStanding,
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

// TestFeeService_CalculateFees_WithSkills tests CalculateFees (Modell A) with real skills via mock.
// Broker-Fees sind stets 0; SalesTax hängt vom Accounting-Skill ab.
func TestFeeService_CalculateFees_WithSkills(t *testing.T) {
	log := logger.NewNoop()

	tests := []struct {
		name             string
		mockSkills       *TradingSkills
		sellValue        float64
		expectedSalesTax float64
	}{
		{
			name: "No skills (worst-case tax)",
			mockSkills: &TradingSkills{
				Accounting: 0,
			},
			sellValue:        1_200_000,
			expectedSalesTax: 60_000, // 5% of 1.2M
		},
		{
			name: "Max Accounting (level 5)",
			mockSkills: &TradingSkills{
				Accounting: 5,
			},
			sellValue:        1_200_000,
			expectedSalesTax: 30_000, // 2.5% of 1.2M
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSkillsService := &MockSkillsService{
				GetCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
					return tt.mockSkills, nil
				},
			}

			service := NewFeeService(mockSkillsService, log)
			fees, err := service.CalculateFees(context.Background(), 123456, "token", 1_000_000, tt.sellValue)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !floatEquals(fees.SalesTax, tt.expectedSalesTax, 0.01) {
				t.Errorf("SalesTax: expected %.2f, got %.2f", tt.expectedSalesTax, fees.SalesTax)
			}
			// Modell A: Broker-Fees und Relist immer 0
			if fees.BrokerFeeBuy != 0 || fees.BrokerFeeSell != 0 || fees.EstimatedRelistFee != 0 {
				t.Errorf("Broker/Relist Fees müssen 0 sein (Modell A)")
			}
			if !floatEquals(fees.TotalFees, fees.SalesTax, 0.01) {
				t.Errorf("TotalFees (%.2f) muss == SalesTax (%.2f) sein", fees.TotalFees, fees.SalesTax)
			}
		})
	}
}

// TestFeeService_CalculateFees_SkillsFallback tests graceful degradation when skills unavailable.
// Modell A: auch bei Skill-Fehler kein Fehler, Sales-Tax auf worst-case (Accounting 0).
func TestFeeService_CalculateFees_SkillsFallback(t *testing.T) {
	log := logger.NewNoop()

	mockSkillsService := &MockSkillsService{
		GetCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
			return nil, context.DeadlineExceeded // Simulate timeout
		},
	}

	service := NewFeeService(mockSkillsService, log)
	fees, err := service.CalculateFees(context.Background(), 123456, "token", 1_000_000, 1_200_000)
	if err != nil {
		t.Fatalf("Should not error even when skills unavailable: %v", err)
	}

	expectedSalesTax := 60_000.0 // 5% of 1.2M (Accounting 0 fallback)
	if !floatEquals(fees.SalesTax, expectedSalesTax, 0.01) {
		t.Errorf("SalesTax: expected %.2f (worst-case), got %.2f", expectedSalesTax, fees.SalesTax)
	}
	if fees.BrokerFeeBuy != 0 || fees.BrokerFeeSell != 0 {
		t.Errorf("Broker-Fees müssen auch im Fallback 0 sein (Modell A)")
	}
}

// TestFeeService_CalculateFees_ModelA prüft Modell A: nur Sales-Tax, keine Broker-Fees.
func TestFeeService_CalculateFees_ModelA(t *testing.T) {
	svc := NewFeeService(nil, testLogger())
	buyValue := 10_000_000.0
	sellValue := 12_000_000.0

	fees, err := svc.CalculateFees(context.Background(), 0, "", buyValue, sellValue)
	if err != nil {
		t.Fatalf("CalculateFees: %v", err)
	}
	if fees.BrokerFeeBuy != 0 || fees.BrokerFeeSell != 0 {
		t.Errorf("Broker-Fees müssen 0 sein (Modell A), got buy=%.2f sell=%.2f", fees.BrokerFeeBuy, fees.BrokerFeeSell)
	}
	if fees.EstimatedRelistFee != 0 {
		t.Errorf("Relist-Fee muss 0 sein (Modell A), got %.2f", fees.EstimatedRelistFee)
	}
	wantTax := 600_000.0 // 5% von 12M (Accounting 0)
	if !floatEquals(fees.SalesTax, wantTax, 0.01) {
		t.Errorf("SalesTax = %.2f, want %.2f", fees.SalesTax, wantTax)
	}
	if !floatEquals(fees.TotalFees, fees.SalesTax, 0.01) {
		t.Errorf("TotalFees (%.2f) muss == SalesTax (%.2f) sein", fees.TotalFees, fees.SalesTax)
	}
}

// testLogger returns a noop logger for use in tests.
func testLogger() *logger.Logger {
	return logger.NewNoop()
}

// floatEquals checks if two floats are equal within a tolerance
func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
