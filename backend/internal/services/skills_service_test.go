package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	esiclient "github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
)

// mockESIServer creates a test HTTP server that mimics ESI API responses
type mockESIServer struct {
	server     *httptest.Server
	skillsResp *esiSkillsResponse
	statusCode int
}

func newMockESIServer(skillsResp *esiSkillsResponse, statusCode int) *mockESIServer {
	mock := &mockESIServer{
		skillsResp: skillsResp,
		statusCode: statusCode,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return configured status code
		if mock.statusCode != http.StatusOK {
			w.WriteHeader(mock.statusCode)
			w.Write([]byte(`{"error": "test error"}`))
			return
		}

		// Return skills response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mock.skillsResp)
	}))

	return mock
}

func (m *mockESIServer) Close() {
	if m.server != nil {
		m.server.Close()
	}
}

// createTestESIClient creates an ESI client connected to a mock HTTP server
func createTestESIClient(t *testing.T, mockServer *mockESIServer, redisClient *redis.Client) *esiclient.Client {
	cfg := esiclient.DefaultConfig(redisClient, "eve-o-provit-test/1.0")
	cfg.MaxRetries = 0        // No retries in tests
	cfg.RespectExpires = true // ESI requirement - MUST be true

	client, err := esiclient.New(cfg)
	require.NoError(t, err)

	// Replace HTTP client with one that redirects to mock server
	mockHTTPClient := &http.Client{
		Transport: &mockTransport{
			mockServer: mockServer,
		},
	}
	client.SetHTTPClient(mockHTTPClient)

	return client
}

// mockTransport redirects ESI requests to mock server
type mockTransport struct {
	mockServer *mockESIServer
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect to mock server
	req.URL.Scheme = "http"
	req.URL.Host = t.mockServer.server.URL[7:] // Remove "http://"
	return http.DefaultTransport.RoundTrip(req)
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
		FactionStanding: 0.0,
		CorpStanding:    0.0,
	}
	cacheKey := "character_skills:12345"
	cachedData, _ := json.Marshal(cachedSkills)
	require.NoError(t, redisClient.Set(ctx, cacheKey, cachedData, 0).Err())

	// Mock ESI server (should NOT be called due to cache hit)
	mockServer := newMockESIServer(nil, http.StatusOK)
	defer mockServer.Close()

	esiClient := createTestESIClient(t, mockServer, redisClient)
	defer esiClient.Close()

	// Create service
	service := NewSkillsService(esiClient, redisClient, logger.NewNoop())

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
	mockSkills := &esiSkillsResponse{
		Skills: []esiSkill{
			{SkillID: 16622, ActiveSkillLevel: 4}, // Accounting IV
			{SkillID: 3446, ActiveSkillLevel: 5},  // Broker Relations V
			{SkillID: 3449, ActiveSkillLevel: 3},  // Navigation III
		},
	}
	mockServer := newMockESIServer(mockSkills, http.StatusOK)
	defer mockServer.Close()

	esiClient := createTestESIClient(t, mockServer, redisClient)
	defer esiClient.Close()

	// Create service
	service := NewSkillsService(esiClient, redisClient, logger.NewNoop())

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

	// Mock ESI error (500 Internal Server Error)
	mockServer := newMockESIServer(nil, http.StatusInternalServerError)
	defer mockServer.Close()

	esiClient := createTestESIClient(t, mockServer, redisClient)
	defer esiClient.Close()

	// Create service
	service := NewSkillsService(esiClient, redisClient, logger.NewNoop())

	// Execute
	result, err := service.GetCharacterSkills(ctx, 12345, "test-token")

	// Verify graceful degradation
	require.NoError(t, err, "Should not return error on ESI failure")
	assert.Equal(t, 0, result.Accounting, "Default skills should be 0")
	assert.Equal(t, 0, result.BrokerRelations)
	assert.Equal(t, 0, result.Navigation)
	assert.Equal(t, 0.0, result.FactionStanding)
	assert.Equal(t, 0.0, result.CorpStanding)
}

// TestSkillsService_ExtractTradingSkills tests skill extraction logic
func TestSkillsService_ExtractTradingSkills(t *testing.T) {
	tests := []struct {
		name     string
		esiSkill esiSkill
		validate func(*testing.T, *TradingSkills)
	}{
		{
			name:     "Accounting skill",
			esiSkill: esiSkill{SkillID: 16622, ActiveSkillLevel: 5},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 5, skills.Accounting)
			},
		},
		{
			name:     "Broker Relations skill",
			esiSkill: esiSkill{SkillID: 3446, ActiveSkillLevel: 4},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 4, skills.BrokerRelations)
			},
		},
		{
			name:     "Advanced Broker Relations skill",
			esiSkill: esiSkill{SkillID: 3447, ActiveSkillLevel: 3},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 3, skills.AdvancedBrokerRelations)
			},
		},
		{
			name:     "Navigation skill",
			esiSkill: esiSkill{SkillID: 3449, ActiveSkillLevel: 5},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 5, skills.Navigation)
			},
		},
		{
			name:     "Evasive Maneuvering skill",
			esiSkill: esiSkill{SkillID: 3452, ActiveSkillLevel: 4},
			validate: func(t *testing.T, skills *TradingSkills) {
				assert.Equal(t, 4, skills.EvasiveManeuvering)
			},
		},
		{
			name:     "Gallente Industrial skill",
			esiSkill: esiSkill{SkillID: 3348, ActiveSkillLevel: 5},
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

			esiResponse := &esiSkillsResponse{
				Skills: []esiSkill{tt.esiSkill},
			}

			service := &SkillsService{
				esiClient:   nil, // Not needed for extraction test
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
	esiResponse := &esiSkillsResponse{
		Skills: []esiSkill{
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
		esiClient:   nil, // Not needed for extraction test
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
	esiResponse := &esiSkillsResponse{
		Skills: []esiSkill{
			{SkillID: 99999, ActiveSkillLevel: 5}, // Unknown skill
			{SkillID: 16622, ActiveSkillLevel: 4}, // Accounting IV
			{SkillID: 88888, ActiveSkillLevel: 3}, // Unknown skill
		},
	}

	service := &SkillsService{
		esiClient:   nil, // Not needed for extraction test
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
		esiClient:   nil, // Not needed for default skills
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
	assert.Equal(t, 0.0, result.CorpStanding)
	assert.Equal(t, 0, result.SpaceshipCommand)
	assert.Equal(t, 0, result.CargoOptimization)
	assert.Equal(t, 0, result.Navigation)
	assert.Equal(t, 0, result.EvasiveManeuvering)
	assert.Equal(t, 0, result.GallenteIndustrial)
	assert.Equal(t, 0, result.CaldariIndustrial)
	assert.Equal(t, 0, result.AmarrIndustrial)
	assert.Equal(t, 0, result.MinmatarIndustrial)
}

// TestSkillsService_ExtractHighestStandings tests standing extraction logic
func TestSkillsService_ExtractHighestStandings(t *testing.T) {
	tests := []struct {
		name                    string
		standings               []esiStanding
		expectedFactionStanding float64
		expectedCorpStanding    float64
	}{
		{
			name: "Multiple faction and corp standings - takes max",
			standings: []esiStanding{
				{FromID: 500001, FromType: "faction", Standing: 5.5},
				{FromID: 500002, FromType: "faction", Standing: 3.2},
				{FromID: 1000035, FromType: "npc_corp", Standing: 7.1},
				{FromID: 1000081, FromType: "npc_corp", Standing: 4.8},
			},
			expectedFactionStanding: 5.5,
			expectedCorpStanding:    7.1,
		},
		{
			name: "Only faction standings",
			standings: []esiStanding{
				{FromID: 500001, FromType: "faction", Standing: 8.0},
				{FromID: 500002, FromType: "faction", Standing: 2.5},
			},
			expectedFactionStanding: 8.0,
			expectedCorpStanding:    0.0,
		},
		{
			name: "Only corp standings",
			standings: []esiStanding{
				{FromID: 1000035, FromType: "npc_corp", Standing: 6.3},
				{FromID: 1000081, FromType: "npc_corp", Standing: 9.1},
			},
			expectedFactionStanding: 0.0,
			expectedCorpStanding:    9.1,
		},
		{
			name: "Negative standings (still take max)",
			standings: []esiStanding{
				{FromID: 500001, FromType: "faction", Standing: -5.0},
				{FromID: 500002, FromType: "faction", Standing: -2.0},
				{FromID: 1000035, FromType: "npc_corp", Standing: -3.5},
				{FromID: 1000081, FromType: "npc_corp", Standing: -1.2},
			},
			expectedFactionStanding: -2.0,
			expectedCorpStanding:    -1.2,
		},
		{
			name: "Agent standings ignored",
			standings: []esiStanding{
				{FromID: 3008416, FromType: "agent", Standing: 10.0},
				{FromID: 500001, FromType: "faction", Standing: 3.0},
			},
			expectedFactionStanding: 3.0,
			expectedCorpStanding:    0.0,
		},
		{
			name:                    "Empty standings",
			standings:               []esiStanding{},
			expectedFactionStanding: 0.0,
			expectedCorpStanding:    0.0,
		},
		{
			name: "Max positive standings (10.0)",
			standings: []esiStanding{
				{FromID: 500001, FromType: "faction", Standing: 10.0},
				{FromID: 1000035, FromType: "npc_corp", Standing: 10.0},
			},
			expectedFactionStanding: 10.0,
			expectedCorpStanding:    10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			s := miniredis.RunT(t)
			defer s.Close()

			redisClient := redis.NewClient(&redis.Options{Addr: s.Addr()})
			defer redisClient.Close()

			service := &SkillsService{
				esiClient:   nil,
				redisClient: redisClient,
				logger:      logger.NewNoop(),
			}

			// Execute
			factionStanding, corpStanding := service.extractHighestStandings(tt.standings)

			// Verify
			assert.Equal(t, tt.expectedFactionStanding, factionStanding,
				"Faction standing mismatch")
			assert.Equal(t, tt.expectedCorpStanding, corpStanding,
				"Corp standing mismatch")
		})
	}
}
