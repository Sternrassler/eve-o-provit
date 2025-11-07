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
	// Fee rate constants
	const (
		baseFeeRate         = 0.03   // Base 3%
		brokerSkillRate     = 0.003  // -0.3% per level
		maxBrokerReduction  = 0.015  // Max -1.5% at level V
		factionStandingRate = 0.0003 // -0.03% per 1.0 standing
		maxFactionReduction = 0.003  // Max -0.3% at 10.0 standing
		corpStandingRate    = 0.0002 // -0.02% per 1.0 standing
		maxCorpReduction    = 0.002  // Max -0.2% at 10.0 standing
		minFeeRate          = 0.01   // Min 1%
		minFeeISK           = 100.0  // Min 100 ISK
	)

	// Broker Relations: -0.3% per level (max -1.5% at level V)
	brokerSkillReduction := brokerSkillRate * float64(brokerRelationsLevel)
	if brokerSkillReduction > maxBrokerReduction {
		brokerSkillReduction = maxBrokerReduction
	}

	// Advanced Broker Relations: -0.3% per level (max -1.5% at level V)
	advBrokerSkillReduction := brokerSkillRate * float64(advancedBrokerRelationsLevel)
	if advBrokerSkillReduction > maxBrokerReduction {
		advBrokerSkillReduction = maxBrokerReduction
	}

	// Faction Standing: -0.03% per 1.0 standing (max -0.3% at 10.0 standing)
	// Only positive standings reduce fees (negative ignored)
	factionReduction := 0.0
	if factionStanding > 0 {
		factionReduction = factionStandingRate * factionStanding
		if factionReduction > maxFactionReduction {
			factionReduction = maxFactionReduction
		}
	}

	// Corp Standing: -0.02% per 1.0 standing (max -0.2% at 10.0 standing)
	// Only positive standings reduce fees (negative ignored)
	corpReduction := 0.0
	if corpStanding > 0 {
		corpReduction = corpStandingRate * corpStanding
		if corpReduction > maxCorpReduction {
			corpReduction = maxCorpReduction
		}
	}

	// Calculate effective fee rate
	feeRate := baseFeeRate - brokerSkillReduction - advBrokerSkillReduction - factionReduction - corpReduction

	// Enforce minimum 1% fee
	if feeRate < minFeeRate {
		feeRate = minFeeRate
	}

	// Calculate fee
	fee := orderValue * feeRate

	// Enforce minimum 100 ISK
	if fee < minFeeISK {
		return minFeeISK
	}

	return fee
}
