package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// MockESIClient mocks the ESI client for testing
type MockESIClient struct {
	getCharacterSkillsFunc func(ctx context.Context, characterID int, accessToken string) (*CharacterSkillsResponse, error)
}

func (m *MockESIClient) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*CharacterSkillsResponse, error) {
	if m.getCharacterSkillsFunc != nil {
		return m.getCharacterSkillsFunc(ctx, characterID, accessToken)
	}
	return nil, errors.New("not mocked")
}

// TestSkillsService_GetCharacterSkills_CacheHit tests cache hit scenario
func TestSkillsService_GetCharacterSkills_CacheHit(t *testing.T) {
	// Setup miniredis
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	// Pre-populate cache
	ctx := context.Background()
	cachedSkills := &TradingSkills{
		Accounting:      4,
		BrokerRelations: 5,
		Navigation:      3,
	}
	cacheKey := "character_skills:12345"
	cachedData, _ := json.Marshal(cachedSkills)
	require.NoError(t, redisClient.Set(ctx, cacheKey, cachedData, 0).Err())

	// Mock ESI client (should NOT be called)
	mockESI := &MockESIClient{
		getCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*CharacterSkillsResponse, error) {
			t.Fatal("ESI should not be called on cache hit")
			return nil, nil
		},
	}

	// Create service
	service := NewSkillsService(mockESI, redisClient, logger.NewNoop())

	// Execute
	result, err := service.GetCharacterSkills(ctx, 12345, "test-token")

	// Verify
	require.NoError(t, err)
	assert.Equal(t, 4, result.Accounting)
	assert.Equal(t, 5, result.BrokerRelations)
	assert.Equal(t, 3, result.Navigation)
}

// TestSkillsService_GetCharacterSkills_CacheMiss tests ESI fetch and caching
func TestSkillsService_GetCharacterSkills_CacheMiss(t *testing.T) {
	// Setup miniredis
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	ctx := context.Background()

	// Mock ESI response
	mockESI := &MockESIClient{
		getCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*CharacterSkillsResponse, error) {
			return &CharacterSkillsResponse{
				Skills: []Skill{
					{SkillID: 16622, ActiveSkillLevel: 4}, // Accounting IV
					{SkillID: 3446, ActiveSkillLevel: 5},  // Broker Relations V
					{SkillID: 3449, ActiveSkillLevel: 3},  // Navigation III
				},
			}, nil
		},
	}

	// Create service
	service := NewSkillsService(mockESI, redisClient, logger.NewNoop())

	// Execute
	result, err := service.GetCharacterSkills(ctx, 12345, "test-token")

	// Verify result
	require.NoError(t, err)
	assert.Equal(t, 4, result.Accounting)
	assert.Equal(t, 5, result.BrokerRelations)
	assert.Equal(t, 3, result.Navigation)

	// Verify cached
	cacheKey := "character_skills:12345"
	cachedData, err := redisClient.Get(ctx, cacheKey).Bytes()
	require.NoError(t, err)
	
	var cachedSkills TradingSkills
	require.NoError(t, json.Unmarshal(cachedData, &cachedSkills))
	assert.Equal(t, 4, cachedSkills.Accounting)
	assert.Equal(t, 5, cachedSkills.BrokerRelations)
}

// TestSkillsService_GetCharacterSkills_ESIError tests graceful fallback
func TestSkillsService_GetCharacterSkills_ESIError(t *testing.T) {
	// Setup miniredis
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	ctx := context.Background()

	// Mock ESI error
	mockESI := &MockESIClient{
		getCharacterSkillsFunc: func(ctx context.Context, characterID int, accessToken string) (*CharacterSkillsResponse, error) {
			return nil, errors.New("ESI timeout")
		},
	}

	// Create service
	service := NewSkillsService(mockESI, redisClient, logger.NewNoop())

	// Execute
	result, err := service.GetCharacterSkills(ctx, 12345, "test-token")

	// Verify graceful degradation
	require.NoError(t, err, "Should not return error on ESI failure")
	assert.Equal(t, 0, result.Accounting, "Default skills should be 0")
	assert.Equal(t, 0, result.BrokerRelations)
	assert.Equal(t, 0, result.Navigation)
}

// TestSkillsService_ExtractTradingSkills tests skill extraction logic
func TestSkillsService_ExtractTradingSkills(t *testing.T) {
	tests := []struct {
		name     string
		esiSkill Skill
		validate func(*testing.T, *TradingSkills)
	}{
		{
			name:     "Accounting skill",
			esiSkill: Skill{SkillID: 16622, ActiveSkillLevel: 5},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 5, skills.Accounting)
			},
		},
		{
			name:     "Broker Relations skill",
			esiSkill: Skill{SkillID: 3446, ActiveSkillLevel: 4},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 4, skills.BrokerRelations)
			},
		},
		{
			name:     "Advanced Broker Relations skill",
			esiSkill: Skill{SkillID: 3447, ActiveSkillLevel: 3},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 3, skills.AdvancedBrokerRelations)
			},
		},
		{
			name:     "Navigation skill",
			esiSkill: Skill{SkillID: 3449, ActiveSkillLevel: 5},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 5, skills.Navigation)
			},
		},
		{
			name:     "Evasive Maneuvering skill",
			esiSkill: Skill{SkillID: 3452, ActiveSkillLevel: 4},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 4, skills.EvasiveManeuvering)
			},
		},
		{
			name:     "Gallente Industrial skill",
			esiSkill: Skill{SkillID: 3348, ActiveSkillLevel: 5},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 5, skills.GallenteIndustrial)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			s := miniredis.RunT(t)
			defer s.Close()

			redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
			defer redisClient.Close()

			esiResponse := &CharacterSkillsResponse{
				Skills: []Skill{tt.esiSkill},
			}

			service := &SkillsService{
				esiClient:   &MockESIClient{},
				redisClient: redisClient,
				logger:      logger.NewNoop(),
			}

			// Execute
			result := service.extractTradingSkills(esiResponse)

			// Verify
			tt.validate(t, result)
		})
	}
}

// TestSkillsService_MultipleSkills tests extraction of multiple skills
func TestSkillsService_MultipleSkills(t *testing.T) {
	// Setup
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	// Mock ESI response with multiple skills
	esiResponse := &CharacterSkillsResponse{
		Skills: []Skill{
			{SkillID: 16622, ActiveSkillLevel: 5}, // Accounting V
			{SkillID: 3446, ActiveSkillLevel: 5},  // Broker Relations V
			{SkillID: 3447, ActiveSkillLevel: 4},  // Advanced Broker Relations IV
			{SkillID: 3327, ActiveSkillLevel: 5},  // Spaceship Command V
			{SkillID: 3449, ActiveSkillLevel: 5},  // Navigation V
			{SkillID: 3452, ActiveSkillLevel: 4},  // Evasive Maneuvering IV
			{SkillID: 3348, ActiveSkillLevel: 5},  // Gallente Industrial V
		},
	}

	service := &SkillsService{
		esiClient:   &MockESIClient{},
		redisClient: redisClient,
		logger:      logger.NewNoop(),
	}

	// Execute
	result := service.extractTradingSkills(esiResponse)

	// Verify all extracted correctly
	assert.Equal(t, 5, result.Accounting)
	assert.Equal(t, 5, result.BrokerRelations)
	assert.Equal(t, 4, result.AdvancedBrokerRelations)
	assert.Equal(t, 5, result.SpaceshipCommand)
	assert.Equal(t, 5, result.Navigation)
	assert.Equal(t, 4, result.EvasiveManeuvering)
	assert.Equal(t, 5, result.GallenteIndustrial)
}

// TestSkillsService_UnknownSkills tests that unknown skill IDs are ignored
func TestSkillsService_UnknownSkills(t *testing.T) {
	// Setup
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	// Mock ESI response with unknown skill IDs
	esiResponse := &CharacterSkillsResponse{
		Skills: []Skill{
			{SkillID: 99999, ActiveSkillLevel: 5}, // Unknown skill
			{SkillID: 16622, ActiveSkillLevel: 4}, // Accounting IV
			{SkillID: 88888, ActiveSkillLevel: 3}, // Unknown skill
		},
	}

	service := &SkillsService{
		esiClient:   &MockESIClient{},
		redisClient: redisClient,
		logger:      logger.NewNoop(),
	}

	// Execute
	result := service.extractTradingSkills(esiResponse)

	// Verify only known skill extracted
	assert.Equal(t, 4, result.Accounting)
	assert.Equal(t, 0, result.BrokerRelations, "Unknown skills should not affect other skills")
}

// TestSkillsService_GetDefaultSkills tests default skills
func TestSkillsService_GetDefaultSkills(t *testing.T) {
	// Setup
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer redisClient.Close()

	service := &SkillsService{
		esiClient:   &MockESIClient{},
		redisClient: redisClient,
		logger:      logger.NewNoop(),
	}

	// Execute
	result := service.getDefaultSkills()

	// Verify all skills are 0 (worst-case)
	assert.Equal(t, 0, result.Accounting)
	assert.Equal(t, 0, result.BrokerRelations)
	assert.Equal(t, 0, result.AdvancedBrokerRelations)
	assert.Equal(t, 0.0, result.FactionStanding)
	assert.Equal(t, 0, result.SpaceshipCommand)
	assert.Equal(t, 0, result.CargoOptimization)
	assert.Equal(t, 0, result.Navigation)
	assert.Equal(t, 0, result.EvasiveManeuvering)
	assert.Equal(t, 0, result.GallenteIndustrial)
	assert.Equal(t, 0, result.CaldarIndustrial)
	assert.Equal(t, 0, result.AmarrIndustrial)
	assert.Equal(t, 0, result.MinmatarIndustrial)
}
