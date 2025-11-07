// Package services - Skills Service for centralized character skills management
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	esiclient "github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// esiSkillsResponse represents ESI /v4/characters/{id}/skills/ response
type esiSkillsResponse struct {
	Skills        []esiSkill `json:"skills"`
	TotalSP       int64      `json:"total_sp"`
	UnallocatedSP int        `json:"unallocated_sp,omitempty"`
}

// esiSkill represents a single skill from ESI
type esiSkill struct {
	SkillID            int   `json:"skill_id"`
	ActiveSkillLevel   int   `json:"active_skill_level"`
	TrainedSkillLevel  int   `json:"trained_skill_level"`
	SkillPointsInSkill int64 `json:"skillpoints_in_skill"`
}

// TradingSkills contains all trading-relevant character skills
// All skill levels are 0-5 (0 = untrained, 5 = max level)
type TradingSkills struct {
	// Trading Skills
	Accounting              int     // Sales Tax reduction (-10% per level, max -50%)
	BrokerRelations         int     // Broker Fee reduction (-0.3% per level, max -1.5%)
	AdvancedBrokerRelations int     // Additional Broker Fee reduction (-0.3% per level, max -1.5%)
	FactionStanding         float64 // Station/Corp standing (0.0-10.0, affects broker fees)

	// Cargo Skills
	SpaceshipCommand  int // +5% cargo capacity per level (max +25%)
	CargoOptimization int // Ship-specific cargo bonus (+5% per level, max +25%)

	// Navigation Skills
	Navigation         int // Warp speed increase (+5% per level, max +25%)
	EvasiveManeuvering int // Align time reduction (-5% per level, max -25%)

	// Ship-specific Industrial Skills (each +5% cargo per level)
	GallenteIndustrial int // Iteron, Nereus, etc.
	CaldariIndustrial  int // Badger, Crane, etc.
	AmarrIndustrial    int // Bestower, Sigil, etc.
	MinmatarIndustrial int // Wreathe, Hoarder, etc.
}

// SkillsService provides character skills fetching with caching
type SkillsService struct {
	esiClient   *esiclient.Client
	redisClient *redis.Client
	logger      *logger.Logger
}

// NewSkillsService creates a new Skills Service instance
func NewSkillsService(
	esiClient *esiclient.Client,
	redisClient *redis.Client,
	logger *logger.Logger,
) SkillsServicer {
	return &SkillsService{
		esiClient:   esiClient,
		redisClient: redisClient,
		logger:      logger,
	}
}

// GetCharacterSkills fetches character skills from ESI with caching
// Returns default skills (all = 0) if ESI fails - ensures graceful degradation
func (s *SkillsService) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*TradingSkills, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("character_skills:%d", characterID)
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		s.logger.Debug("Skills cache hit", "characterID", characterID)
		var skills TradingSkills
		if err := json.Unmarshal(cachedData, &skills); err == nil {
			return &skills, nil
		}
		s.logger.Warn("Failed to unmarshal cached skills", "error", err)
	}

	// 2. Cache miss - fetch from ESI
	s.logger.Debug("Skills cache miss - fetching from ESI", "characterID", characterID)
	esiSkills, err := s.fetchSkillsFromESI(ctx, characterID, accessToken)
	if err != nil {
		s.logger.Error("ESI skills fetch failed - using defaults", "error", err, "characterID", characterID)
		// Graceful degradation: return default skills (worst-case fees/cargo)
		return s.getDefaultSkills(), nil
	}

	// 3. Extract trading skills
	skills := s.extractTradingSkills(esiSkills)

	// 4. Cache the result (5min TTL)
	if skillsData, err := json.Marshal(skills); err == nil {
		if err := s.redisClient.Set(ctx, cacheKey, skillsData, 5*time.Minute).Err(); err != nil {
			s.logger.Warn("Failed to cache skills", "error", err)
		}
	}

	s.logger.Info("Skills fetched from ESI and cached",
		"characterID", characterID,
		"accounting", skills.Accounting,
		"brokerRelations", skills.BrokerRelations,
	)

	return skills, nil
}

// fetchSkillsFromESI fetches character skills from ESI API
// Follows the pattern from trading.go (direct HTTP request with Authorization header)
func (s *SkillsService) fetchSkillsFromESI(ctx context.Context, characterID int, accessToken string) (*esiSkillsResponse, error) {
	endpoint := fmt.Sprintf("/v4/characters/%d/skills/", characterID)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", "https://esi.evetech.net"+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Execute request through ESI client (handles rate limiting, caching, retries)
	resp, err := s.esiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("esi request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("unauthorized: status %d", resp.StatusCode)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var skillsResp esiSkillsResponse
	if err := json.NewDecoder(resp.Body).Decode(&skillsResp); err != nil {
		return nil, fmt.Errorf("parse skills response: %w", err)
	}

	return &skillsResp, nil
}

// extractTradingSkills extracts relevant trading skills from ESI skill list
func (s *SkillsService) extractTradingSkills(esiSkills *esiSkillsResponse) *TradingSkills {
	skills := &TradingSkills{
		// Default faction standing (neutral)
		FactionStanding: 0.0,
	}

	for _, skill := range esiSkills.Skills {
		switch skill.SkillID {
		// Trading Skills
		case 16622: // Accounting
			skills.Accounting = skill.ActiveSkillLevel
		case 3446: // Broker Relations
			skills.BrokerRelations = skill.ActiveSkillLevel
		case 3447: // Advanced Broker Relations (formerly Visibility)
			skills.AdvancedBrokerRelations = skill.ActiveSkillLevel

		// Cargo Skills
		case 3327: // Spaceship Command
			skills.SpaceshipCommand = skill.ActiveSkillLevel
		// Note: Generic cargo optimization skill ID needs verification
		// Different ship classes have different cargo skills

		// Navigation Skills
		case 3449: // Navigation
			skills.Navigation = skill.ActiveSkillLevel
		case 3452: // Evasive Maneuvering
			skills.EvasiveManeuvering = skill.ActiveSkillLevel

		// Racial Industrial Skills
		case 3348: // Gallente Industrial
			skills.GallenteIndustrial = skill.ActiveSkillLevel
		case 3346: // Caldari Industrial
			skills.CaldariIndustrial = skill.ActiveSkillLevel
		case 3347: // Amarr Industrial
			skills.AmarrIndustrial = skill.ActiveSkillLevel
		case 3349: // Minmatar Industrial
			skills.MinmatarIndustrial = skill.ActiveSkillLevel
		}
	}

	// TODO: Fetch faction standing from ESI /characters/{character_id}/standings/
	// This is a separate endpoint and requires additional API call
	// For now, default to 0.0 (neutral standing)

	return skills
}

// getDefaultSkills returns default skills (all = 0) for fallback scenarios
// Used when ESI is unavailable or character skills cannot be fetched
// This ensures worst-case calculations (highest fees, lowest cargo)
func (s *SkillsService) getDefaultSkills() *TradingSkills {
	return &TradingSkills{
		// All skills = 0 (untrained)
		Accounting:              0,
		BrokerRelations:         0,
		AdvancedBrokerRelations: 0,
		FactionStanding:         0.0,
		SpaceshipCommand:        0,
		CargoOptimization:       0,
		Navigation:              0,
		EvasiveManeuvering:      0,
		GallenteIndustrial:      0,
		CaldariIndustrial:       0,
		AmarrIndustrial:         0,
		MinmatarIndustrial:      0,
	}
}
