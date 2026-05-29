package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// resolveTradingRates fetches the character's skills and derives the sales-tax
// and broker-fee rates plus the SkillsApplied payload. On a skills fetch error
// it degrades gracefully to level-0 rates and reports Applied=false. Shared by
// every trading feature so the skill-adjusted fee model stays consistent.
func resolveTradingRates(ctx context.Context, skills SkillsServicer, characterID int, accessToken string) (salesTaxRate, brokerFeeRate float64, applied models.SkillsApplied) {
	s, err := skills.GetCharacterSkills(ctx, characterID, accessToken)
	if err != nil || s == nil {
		s = &TradingSkills{}
	}
	salesTaxRate = SalesTaxRate(s.Accounting)
	brokerFeeRate = BrokerFeeRate(s.BrokerRelations, s.AdvancedBrokerRelations, s.FactionStanding, s.CorpStanding)
	applied = models.SkillsApplied{
		Applied:         err == nil,
		Accounting:      s.Accounting,
		BrokerRelations: s.BrokerRelations,
		SalesTaxRate:    salesTaxRate,
		BrokerFeeRate:   brokerFeeRate,
	}
	return
}
