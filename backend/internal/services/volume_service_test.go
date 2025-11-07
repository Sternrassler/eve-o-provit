// Package services - Unit tests for VolumeService
package services

import (
	"context"
	"testing"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMarketRepository implements MarketRepositoryInterface for testing
type MockMarketRepository struct {
	mock.Mock
}

func (m *MockMarketRepository) GetVolumeHistory(ctx context.Context, typeID, regionID, days int) ([]database.PriceHistory, error) {
	args := m.Called(ctx, typeID, regionID, days)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.PriceHistory), args.Error(1)
}

func (m *MockMarketRepository) UpsertPriceHistory(ctx context.Context, history []database.PriceHistory) error {
	args := m.Called(ctx, history)
	return args.Error(0)
}

// MockESIClient implements ESIMarketHistoryFetcher for testing
type MockESIClient struct {
	mock.Mock
}

func (m *MockESIClient) FetchMarketHistory(ctx context.Context, regionID, typeID int) ([]database.PriceHistory, error) {
	args := m.Called(ctx, regionID, typeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.PriceHistory), args.Error(1)
}

func TestCalculateLiquidationTime(t *testing.T) {
	vs := NewVolumeService(nil, nil)

	tests := []struct {
		name         string
		quantity     int
		dailyVolume  float64
		expectedDays float64
	}{
		{
			name:         "High volume item",
			quantity:     1000,
			dailyVolume:  500,
			expectedDays: 20.0, // 1000 / (500 * 0.10) = 20 days
		},
		{
			name:         "Low volume item",
			quantity:     100,
			dailyVolume:  10,
			expectedDays: 100.0, // 100 / (10 * 0.10) = 100 days
		},
		{
			name:         "Zero volume - illiquid",
			quantity:     100,
			dailyVolume:  0,
			expectedDays: 999.0, // Illiquid market
		},
		{
			name:         "Tiny volume - effectively illiquid",
			quantity:     1000,
			dailyVolume:  0.5,
			expectedDays: 20000.0, // 1000 / (0.5 * 0.10) = 20000 days
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			days := vs.CalculateLiquidationTime(tt.quantity, tt.dailyVolume)
			assert.Equal(t, tt.expectedDays, days, "Liquidation time should match expected")
		})
	}
}

func TestGetVolumeMetrics_NoData(t *testing.T) {
	mockRepo := new(MockMarketRepository)
	mockESI := new(MockESIClient)
	vs := NewVolumeService(mockRepo, mockESI)

	ctx := context.Background()
	typeID := 34
	regionID := 10000002

	// Mock: No historical data
	mockRepo.On("GetVolumeHistory", ctx, typeID, regionID, 30).Return([]database.PriceHistory{}, nil)

	metrics, err := vs.GetVolumeMetrics(ctx, typeID, regionID)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, typeID, metrics.TypeID)
	assert.Equal(t, regionID, metrics.RegionID)
	assert.Equal(t, 0.0, metrics.DailyVolumeAvg)
	assert.Equal(t, 0.0, metrics.DailyISKTurnover)
	assert.Equal(t, 0, metrics.LiquidityScore)
	assert.Equal(t, 0, metrics.DataDays)

	mockRepo.AssertExpectations(t)
}

func TestGetVolumeMetrics_WithData(t *testing.T) {
	mockRepo := new(MockMarketRepository)
	mockESI := new(MockESIClient)
	vs := NewVolumeService(mockRepo, mockESI)

	ctx := context.Background()
	typeID := 34
	regionID := 10000002

	// Mock: 5 days of historical data
	volume1 := int64(1000)
	volume2 := int64(1200)
	volume3 := int64(800)
	volume4 := int64(1100)
	volume5 := int64(900)
	avgPrice := 5.0

	history := []database.PriceHistory{
		{TypeID: typeID, RegionID: regionID, Date: time.Now().AddDate(0, 0, -1), Volume: &volume1, Average: &avgPrice},
		{TypeID: typeID, RegionID: regionID, Date: time.Now().AddDate(0, 0, -2), Volume: &volume2, Average: &avgPrice},
		{TypeID: typeID, RegionID: regionID, Date: time.Now().AddDate(0, 0, -3), Volume: &volume3, Average: &avgPrice},
		{TypeID: typeID, RegionID: regionID, Date: time.Now().AddDate(0, 0, -4), Volume: &volume4, Average: &avgPrice},
		{TypeID: typeID, RegionID: regionID, Date: time.Now().AddDate(0, 0, -5), Volume: &volume5, Average: &avgPrice},
	}

	mockRepo.On("GetVolumeHistory", ctx, typeID, regionID, 30).Return(history, nil)

	metrics, err := vs.GetVolumeMetrics(ctx, typeID, regionID)

	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.Equal(t, typeID, metrics.TypeID)
	assert.Equal(t, regionID, metrics.RegionID)

	// Average volume: (1000 + 1200 + 800 + 1100 + 900) / 5 = 1000
	assert.InDelta(t, 1000.0, metrics.DailyVolumeAvg, 0.01)

	// Average ISK turnover: 1000 * 5.0 = 5000 ISK/day
	assert.InDelta(t, 5000.0, metrics.DailyISKTurnover, 0.01)

	// Liquidity score should be > 0 with volume data
	assert.Greater(t, metrics.LiquidityScore, 0)
	assert.LessOrEqual(t, metrics.LiquidityScore, 100)

	// Data days should match valid entries
	assert.Equal(t, 5, metrics.DataDays)

	mockRepo.AssertExpectations(t)
}

func TestFetchAndStoreMarketHistory(t *testing.T) {
	mockRepo := new(MockMarketRepository)
	mockESI := new(MockESIClient)
	vs := NewVolumeService(mockRepo, mockESI)

	ctx := context.Background()
	typeID := 34
	regionID := 10000002

	// Mock ESI response
	volume := int64(1000)
	avgPrice := 5.0
	esiHistory := []database.PriceHistory{
		{TypeID: typeID, RegionID: regionID, Date: time.Now(), Volume: &volume, Average: &avgPrice},
	}

	mockESI.On("FetchMarketHistory", ctx, regionID, typeID).Return(esiHistory, nil)
	mockRepo.On("UpsertPriceHistory", ctx, esiHistory).Return(nil)

	err := vs.FetchAndStoreMarketHistory(ctx, typeID, regionID)

	assert.NoError(t, err)
	mockESI.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestFetchAndStoreMarketHistory_CacheHit(t *testing.T) {
	mockRepo := new(MockMarketRepository)
	mockESI := new(MockESIClient)
	vs := NewVolumeService(mockRepo, mockESI)

	ctx := context.Background()
	typeID := 34
	regionID := 10000002

	// Mock: ESI returns empty (cache hit)
	mockESI.On("FetchMarketHistory", ctx, regionID, typeID).Return([]database.PriceHistory{}, nil)
	// UpsertPriceHistory should NOT be called

	err := vs.FetchAndStoreMarketHistory(ctx, typeID, regionID)

	assert.NoError(t, err)
	mockESI.AssertExpectations(t)
	mockRepo.AssertNotCalled(t, "UpsertPriceHistory")
}

func TestCalculateVolatility(t *testing.T) {
	vs := NewVolumeService(nil, nil)

	tests := []struct {
		name             string
		volumes          []int64
		expectedRange    string // "low", "medium", "high"
		expectedMaxValue float64
	}{
		{
			name:             "Stable volume",
			volumes:          []int64{1000, 1010, 990, 1005, 995},
			expectedRange:    "low",
			expectedMaxValue: 0.1, // CV should be very low
		},
		{
			name:             "Moderate volatility",
			volumes:          []int64{1000, 1200, 800, 1100, 900},
			expectedRange:    "medium",
			expectedMaxValue: 0.5,
		},
		{
			name:             "High volatility",
			volumes:          []int64{1000, 3000, 500, 2500, 100},
			expectedRange:    "high",
			expectedMaxValue: 1.0,
		},
		{
			name:             "Insufficient data",
			volumes:          []int64{1000},
			expectedRange:    "high",
			expectedMaxValue: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := make([]database.PriceHistory, len(tt.volumes))
			for i, vol := range tt.volumes {
				v := vol
				history[i] = database.PriceHistory{Volume: &v}
			}

			volatility := vs.calculateVolatility(history)

			assert.GreaterOrEqual(t, volatility, 0.0)
			assert.LessOrEqual(t, volatility, 1.0)
			assert.LessOrEqual(t, volatility, tt.expectedMaxValue)
		})
	}
}

func TestCalculateLiquidityScore(t *testing.T) {
	vs := NewVolumeService(nil, nil)

	tests := []struct {
		name           string
		dailyVolumeAvg float64
		volumes        []int64
		expectedMin    int
		expectedMax    int
	}{
		{
			name:           "High volume, low volatility",
			dailyVolumeAvg: 1000,
			volumes:        []int64{1000, 1010, 990, 1005, 995},
			expectedMin:    70, // Should have high score
			expectedMax:    100,
		},
		{
			name:           "Low volume",
			dailyVolumeAvg: 10,
			volumes:        []int64{10, 10, 10, 10, 10},
			expectedMin:    0,
			expectedMax:    60, // Lower score due to low volume
		},
		{
			name:           "Zero volume",
			dailyVolumeAvg: 0,
			volumes:        []int64{},
			expectedMin:    0,
			expectedMax:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history := make([]database.PriceHistory, len(tt.volumes))
			for i, vol := range tt.volumes {
				v := vol
				history[i] = database.PriceHistory{Volume: &v}
			}

			score := vs.calculateLiquidityScore(tt.dailyVolumeAvg, history)

			assert.GreaterOrEqual(t, score, tt.expectedMin)
			assert.LessOrEqual(t, score, tt.expectedMax)
			assert.GreaterOrEqual(t, score, 0)
			assert.LessOrEqual(t, score, 100)
		})
	}
}
