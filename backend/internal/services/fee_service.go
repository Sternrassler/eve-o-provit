// Package services - Fee Calculation Service for trading fee calculations
package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// Fees contains all calculated trading fees for a transaction.
// Trading-Modell A (Sofort-Arbitrage): nur SalesTax ist > 0; Broker/Relist sind stets 0.
type Fees struct {
	SalesTax           float64 // Sales tax on sell orders (base 5%, reduced by Accounting)
	BrokerFeeBuy       float64 // Modell A: 0 (keine Order-Platzierung)
	BrokerFeeSell      float64 // Modell A: 0 (keine Order-Platzierung)
	EstimatedRelistFee float64 // Modell A: 0 (kein Relisting)
	TotalFees          float64 // Sum of all fees (== SalesTax unter Modell A)
}

// FeeService provides trading fee calculations with skill integration
type FeeService struct {
	skillsService SkillsServicer
	logger        *logger.Logger
}

// NewFeeService creates a new Fee Service instance
func NewFeeService(
	skillsService SkillsServicer,
	logger *logger.Logger,
) FeeServicer {
	return &FeeService{
		skillsService: skillsService,
		logger:        logger,
	}
}

// CalculateFees calculates all trading fees for a transaction.
// Trading-Modell A (Sofort-Arbitrage): Kauf/Verkauf gegen bestehende Orders.
// Es fallen KEINE Broker-Fees an (keine Order-Platzierung), nur Sales-Tax beim Verkauf.
func (s *FeeService) CalculateFees(
	ctx context.Context,
	characterID int,
	accessToken string,
	buyValue float64,
	sellValue float64,
) (*Fees, error) {
	// Trading-Modell A (Sofort-Arbitrage): Kauf/Verkauf gegen bestehende Orders.
	// Es fallen KEINE Broker-Fees an (keine Order-Platzierung), nur Sales-Tax beim Verkauf.
	accounting := 0 // worst-case, falls Skills nicht ladbar
	if s.skillsService != nil {
		skills, err := s.skillsService.GetCharacterSkills(ctx, characterID, accessToken)
		if err != nil {
			s.logger.Warn("Failed to fetch skills - using worst-case sales tax",
				"error", err, "characterID", characterID)
		} else {
			accounting = skills.Accounting
		}
	}

	salesTax := s.CalculateSalesTax(accounting, sellValue)

	return &Fees{
		SalesTax:           salesTax,
		BrokerFeeBuy:       0, // Modell A: keine Broker-Fee
		BrokerFeeSell:      0, // Modell A: keine Broker-Fee
		EstimatedRelistFee: 0, // Modell A: kein Relisting
		TotalFees:          salesTax,
	}, nil
}

// CalculateSalesTax calculates sales tax based on Accounting skill
// EVE Formula: Base 5% → Reduced by 10% per Accounting level → 2.5% at Accounting V
// Minimum fee: 100 ISK
func (s *FeeService) CalculateSalesTax(accountingLevel int, orderValue float64) float64 {
	// Base tax rate: 5%
	baseTaxRate := 0.05

	// Accounting skill: -10% per level (max -50% at level V)
	// Level 0: 5.00%
	// Level 1: 4.50%
	// Level 2: 4.00%
	// Level 3: 3.50%
	// Level 4: 3.25%
	// Level 5: 2.5% (0.05 * (1 - 0.1*5) = 0.025)
	skillReduction := 0.10 * float64(accountingLevel)
	if skillReduction > 0.50 {
		skillReduction = 0.50
	}

	taxRate := baseTaxRate * (1 - skillReduction)

	// Calculate tax
	tax := orderValue * taxRate

	// Enforce minimum 100 ISK
	if tax < 100 {
		return 100
	}

	return tax
}
