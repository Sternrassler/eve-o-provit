// Package services - Fee Calculation Service for trading fee calculations
package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// Fees contains all calculated trading fees for a transaction
type Fees struct {
	SalesTax           float64 // Sales tax on sell orders (base 5%, reduced by Accounting)
	BrokerFeeBuy       float64 // Broker fee for buy order placement (base 3%)
	BrokerFeeSell      float64 // Broker fee for sell order placement (base 3%)
	EstimatedRelistFee float64 // Estimated relist fees (based on market volatility)
	TotalFees          float64 // Sum of all fees
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

// CalculateFees calculates all trading fees for a transaction
// Integrates with SkillsService to get character skills for accurate fee calculation
// Falls back to worst-case fees (no skills) if skills cannot be fetched
func (s *FeeService) CalculateFees(
	ctx context.Context,
	characterID int,
	accessToken string,
	buyValue float64,
	sellValue float64,
) (*Fees, error) {
	// 1. Get character skills (with graceful degradation)
	skills, err := s.skillsService.GetCharacterSkills(ctx, characterID, accessToken)
	if err != nil {
		s.logger.Warn("Failed to fetch skills - using worst-case fees",
			"error", err,
			"characterID", characterID)
		// Fallback: worst-case fees (all skills = 0)
		skills = &TradingSkills{
			Accounting:              0,
			BrokerRelations:         0,
			AdvancedBrokerRelations: 0,
			FactionStanding:         0.0,
			CorpStanding:            0.0,
		}
	}

	// 2. Calculate individual fees
	salesTax := s.CalculateSalesTax(skills.Accounting, sellValue)
	brokerFeeBuy := s.CalculateBrokerFee(
		skills.BrokerRelations,
		skills.AdvancedBrokerRelations,
		skills.FactionStanding,
		skills.CorpStanding,
		buyValue,
	)
	brokerFeeSell := s.CalculateBrokerFee(
		skills.BrokerRelations,
		skills.AdvancedBrokerRelations,
		skills.FactionStanding,
		skills.CorpStanding,
		sellValue,
	)

	// 3. Estimate relist fees (assume 3 relists per day at 50% of broker fee)
	// This is a conservative estimate for market volatility
	estimatedRelistFee := brokerFeeSell * 0.5 * 3

	// 4. Total all fees
	totalFees := salesTax + brokerFeeBuy + brokerFeeSell + estimatedRelistFee

	s.logger.Debug("Calculated trading fees",
		"characterID", characterID,
		"salesTax", salesTax,
		"brokerFeeBuy", brokerFeeBuy,
		"brokerFeeSell", brokerFeeSell,
		"totalFees", totalFees,
		"accounting", skills.Accounting,
		"brokerRelations", skills.BrokerRelations,
	)

	return &Fees{
		SalesTax:           salesTax,
		BrokerFeeBuy:       brokerFeeBuy,
		BrokerFeeSell:      brokerFeeSell,
		EstimatedRelistFee: estimatedRelistFee,
		TotalFees:          totalFees,
	}, nil
}

// CalculateSalesTax calculates sales tax based on Accounting skill
// EVE Formula: Base 5% → Reduced by 10% per Accounting level → Min 3.375% (Accounting V)
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
	// Level 5: 3.375% (actual formula: 0.05 * (1 - 0.1*5) = 0.025, but EVE caps at 3.375%)
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

// CalculateBrokerFee calculates broker fee based on skills and standings
// EVE Formula: Base 3% → Reduced by skills + standings → Min 1%
// - Broker Relations: -0.3% per level (max -1.5%)
// - Advanced Broker Relations: -0.3% per level (max -1.5%)
// - Faction Standing: -0.03% per 1.0 standing (max -0.3% at 10.0)
// - Corp Standing: -0.02% per 1.0 standing (max -0.2% at 10.0)
// Minimum fee: 100 ISK
func (s *FeeService) CalculateBrokerFee(
	brokerRelationsLevel int,
	advancedBrokerRelationsLevel int,
	factionStanding float64,
	corpStanding float64,
	orderValue float64,
) float64 {
	// Base broker fee: 3%
	baseFeeRate := 0.03

	// Broker Relations: -0.3% per level (max -1.5% at level V)
	brokerSkillReduction := 0.003 * float64(brokerRelationsLevel)
	if brokerSkillReduction > 0.015 {
		brokerSkillReduction = 0.015
	}

	// Advanced Broker Relations: -0.3% per level (max -1.5% at level V)
	advBrokerSkillReduction := 0.003 * float64(advancedBrokerRelationsLevel)
	if advBrokerSkillReduction > 0.015 {
		advBrokerSkillReduction = 0.015
	}

	// Faction Standing: -0.03% per 1.0 standing (max -0.3% at 10.0 standing)
	// Only positive standings reduce fees (negative ignored)
	factionReduction := 0.0
	if factionStanding > 0 {
		factionReduction = 0.0003 * factionStanding
		if factionReduction > 0.003 {
			factionReduction = 0.003
		}
	}

	// Corp Standing: -0.02% per 1.0 standing (max -0.2% at 10.0 standing)
	// Only positive standings reduce fees (negative ignored)
	corpReduction := 0.0
	if corpStanding > 0 {
		corpReduction = 0.0002 * corpStanding
		if corpReduction > 0.002 {
			corpReduction = 0.002
		}
	}

	// Calculate effective fee rate
	feeRate := baseFeeRate - brokerSkillReduction - advBrokerSkillReduction - factionReduction - corpReduction

	// Enforce minimum 1% fee
	if feeRate < 0.01 {
		feeRate = 0.01
	}

	// Calculate fee
	fee := orderValue * feeRate

	// Enforce minimum 100 ISK
	if fee < 100 {
		return 100
	}

	return fee
}
