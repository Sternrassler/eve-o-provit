// Package services - Volume metrics and liquidity analysis
package services

import (
	"context"
	"fmt"
	"math"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// Constants for volume and liquidity calculations
const (
	// IlliquidMarketDays represents the liquidation time for markets with zero volume
	IlliquidMarketDays = 999.0

	// DefaultMarketSharePercent is the assumed market share a trader can capture (10%)
	DefaultMarketSharePercent = 0.10

	// Liquidity score calculation constants
	liquidityScoreVolumeMax     = 50.0  // Maximum points from volume component
	liquidityScoreVolatilityMax = 50.0  // Maximum points from volatility component
	liquidityScoreVolumeScale   = 5.0   // Scaling factor for volume score (100 items/day = 10 points)
	liquidityScoreVolumeDivisor = 100.0 // Divisor for volume normalization
)

// VolumeServicer defines the interface for volume metrics calculations
type VolumeServicer interface {
	GetVolumeMetrics(ctx context.Context, typeID, regionID int) (*models.VolumeMetrics, error)
	CalculateLiquidationTime(quantity int, dailyVolume float64) float64
	FetchAndStoreMarketHistory(ctx context.Context, typeID, regionID int) error
}

// VolumeService handles volume metrics and liquidity calculations
type VolumeService struct {
	marketRepo MarketRepositoryInterface
	esiClient  ESIMarketHistoryFetcher
}

// ESIMarketHistoryFetcher interface for fetching market history from ESI
type ESIMarketHistoryFetcher interface {
	FetchMarketHistory(ctx context.Context, regionID, typeID int) ([]database.PriceHistory, error)
}

// MarketRepositoryInterface extends database operations needed for volume analysis
type MarketRepositoryInterface interface {
	GetVolumeHistory(ctx context.Context, typeID, regionID, days int) ([]database.PriceHistory, error)
	UpsertPriceHistory(ctx context.Context, history []database.PriceHistory) error
}

// NewVolumeService creates a new volume service
func NewVolumeService(marketRepo MarketRepositoryInterface, esiClient ESIMarketHistoryFetcher) *VolumeService {
	return &VolumeService{
		marketRepo: marketRepo,
		esiClient:  esiClient,
	}
}

// FetchAndStoreMarketHistory fetches market history from ESI and stores it in the database
func (vs *VolumeService) FetchAndStoreMarketHistory(ctx context.Context, typeID, regionID int) error {
	history, err := vs.esiClient.FetchMarketHistory(ctx, regionID, typeID)
	if err != nil {
		return fmt.Errorf("failed to fetch market history: %w", err)
	}

	if len(history) == 0 {
		return nil // No new data (cache hit)
	}

	if err := vs.marketRepo.UpsertPriceHistory(ctx, history); err != nil {
		return fmt.Errorf("failed to store market history: %w", err)
	}

	return nil
}

// GetVolumeMetrics calculates volume metrics for a type in a region
// Uses 30-day historical data to compute averages and liquidity scores
func (vs *VolumeService) GetVolumeMetrics(ctx context.Context, typeID, regionID int) (*models.VolumeMetrics, error) {
	// Fetch last 30 days of volume history
	const lookbackDays = 30
	history, err := vs.marketRepo.GetVolumeHistory(ctx, typeID, regionID, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume history: %w", err)
	}

	if len(history) == 0 {
		// No historical data available - return zero metrics
		return &models.VolumeMetrics{
			TypeID:           typeID,
			RegionID:         regionID,
			DailyVolumeAvg:   0,
			DailyISKTurnover: 0,
			LiquidityScore:   0,
			DataDays:         0,
		}, nil
	}

	// Calculate average daily volume
	totalVolume := int64(0)
	totalISK := 0.0
	validDays := 0

	for _, h := range history {
		if h.Volume != nil && *h.Volume > 0 {
			totalVolume += *h.Volume
			validDays++

			// Calculate ISK turnover if average price available
			if h.Average != nil && *h.Average > 0 {
				totalISK += float64(*h.Volume) * *h.Average
			}
		}
	}

	// Compute averages
	dailyVolumeAvg := 0.0
	dailyISKTurnover := 0.0
	if validDays > 0 {
		dailyVolumeAvg = float64(totalVolume) / float64(validDays)
		dailyISKTurnover = totalISK / float64(validDays)
	}

	// Calculate liquidity score (0-100)
	liquidityScore := vs.calculateLiquidityScore(dailyVolumeAvg, history)

	return &models.VolumeMetrics{
		TypeID:           typeID,
		RegionID:         regionID,
		DailyVolumeAvg:   dailyVolumeAvg,
		DailyISKTurnover: dailyISKTurnover,
		LiquidityScore:   liquidityScore,
		DataDays:         validDays,
	}, nil
}

// CalculateLiquidationTime estimates the number of days to sell inventory
// Assumes trader can capture DefaultMarketSharePercent (10%) of daily market volume
func (vs *VolumeService) CalculateLiquidationTime(quantity int, dailyVolume float64) float64 {
	if dailyVolume <= 0 {
		return IlliquidMarketDays // Illiquid market
	}

	// Assume trader can capture configured percentage of daily volume
	yourDailyVolume := dailyVolume * DefaultMarketSharePercent

	if yourDailyVolume <= 0 {
		return IlliquidMarketDays
	}

	days := float64(quantity) / yourDailyVolume
	return days
}

// calculateLiquidityScore computes a 0-100 liquidity score
// Higher volume and lower volatility = higher score
func (vs *VolumeService) calculateLiquidityScore(dailyVolumeAvg float64, history []database.PriceHistory) int {
	if dailyVolumeAvg <= 0 || len(history) < 2 {
		return 0
	}

	// Volume score component (0-50 points)
	// Higher volume = better liquidity
	// 100 items/day = 10 points, 1000 items/day = 50 points (capped)
	volumeScore := math.Min(liquidityScoreVolumeMax, (dailyVolumeAvg/liquidityScoreVolumeDivisor)*liquidityScoreVolumeScale)

	// Volatility score component (0-50 points)
	// Lower volatility = more stable market = better liquidity
	volatility := vs.calculateVolatility(history)
	volatilityScore := math.Max(0, liquidityScoreVolatilityMax*(1-volatility))

	totalScore := int(math.Round(volumeScore + volatilityScore))
	if totalScore > 100 {
		totalScore = 100
	}
	if totalScore < 0 {
		totalScore = 0
	}

	return totalScore
}

// calculateVolatility computes coefficient of variation for volume data
// Returns value between 0 (no volatility) and 1+ (high volatility)
func (vs *VolumeService) calculateVolatility(history []database.PriceHistory) float64 {
	if len(history) < 2 {
		return 1.0 // Assume high volatility if insufficient data
	}

	// Extract volumes
	volumes := make([]float64, 0, len(history))
	for _, h := range history {
		if h.Volume != nil && *h.Volume > 0 {
			volumes = append(volumes, float64(*h.Volume))
		}
	}

	if len(volumes) < 2 {
		return 1.0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range volumes {
		sum += v
	}
	mean := sum / float64(len(volumes))

	if mean == 0 {
		return 1.0
	}

	// Calculate standard deviation
	variance := 0.0
	for _, v := range volumes {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(volumes))
	stdDev := math.Sqrt(variance)

	// Coefficient of variation (CV = stdDev / mean)
	// Normalize to 0-1 range (CV > 1 is clamped to 1)
	cv := stdDev / mean
	if cv > 1.0 {
		cv = 1.0
	}

	return cv
}
