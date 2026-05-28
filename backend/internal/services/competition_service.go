package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// CompetitionService derives the "order update frequency" competition indicator for
// Multi-Hub Comparison (#43). It serves the live snapshot-churn metric when available
// and falls back to a daily-order-count baseline (Ansatz A+C).
type CompetitionService struct {
	repo       *database.CompetitionRepository
	marketRepo *database.MarketRepository
	logger     *logger.Logger
}

// NewCompetitionService creates a new competition service.
func NewCompetitionService(repo *database.CompetitionRepository, marketRepo *database.MarketRepository, logger *logger.Logger) *CompetitionService {
	return &CompetitionService{repo: repo, marketRepo: marketRepo, logger: logger}
}

// Register lazily marks a (type, region) pair for collector tracking.
func (s *CompetitionService) Register(ctx context.Context, typeID, regionID int) {
	if err := s.repo.RegisterTracked(ctx, typeID, regionID); err != nil {
		s.logger.Warn("competition: register tracked failed", "error", err, "type", typeID, "region", regionID)
	}
}

// GetCompetition returns the competition indicator for (type, region). Live churn wins;
// otherwise a baseline derived from the latest daily order_count (spread over 24h).
func (s *CompetitionService) GetCompetition(ctx context.Context, typeID, regionID int) models.CompetitionInfo {
	if m, ok, err := s.repo.GetMetric(ctx, typeID, regionID); err == nil && ok && m.Source == "live" {
		return models.CompetitionInfo{ChangesPerHour: m.ChangesPerHour, Source: "live"}
	}
	return models.CompetitionInfo{ChangesPerHour: s.baseline(ctx, typeID, regionID), Source: "baseline"}
}

// baseline maps the most recent daily order_count to an hourly figure.
func (s *CompetitionService) baseline(ctx context.Context, typeID, regionID int) float64 {
	hist, err := s.marketRepo.GetVolumeHistory(ctx, typeID, regionID, 1)
	if err != nil || len(hist) == 0 || hist[0].OrderCount == nil {
		return 0
	}
	return float64(*hist[0].OrderCount) / 24.0
}

// ComputeChurn counts order changes between two consecutive snapshots and expresses
// them per hour. A change is an order added, removed, or repriced. Pure function.
func ComputeChurn(prev, curr map[int64]float64, hours float64) float64 {
	if hours <= 0 {
		return 0
	}
	changes := 0
	for id, price := range curr {
		old, existed := prev[id]
		if !existed {
			changes++ // added
		} else if old != price {
			changes++ // repriced
		}
	}
	for id := range prev {
		if _, stillThere := curr[id]; !stillThere {
			changes++ // removed
		}
	}
	return float64(changes) / hours
}
