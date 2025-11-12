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

// esiStanding represents a single standing entry from ESI /v2/characters/{id}/standings/
type esiStanding struct {
	FromID   int     `json:"from_id"`
	FromType string  `json:"from_type"` // "faction", "npc_corp", "agent"
	Standing float64 `json:"standing"`  // -10.0 to +10.0
}

// TradingSkills contains all trading-relevant character skills
// All skill levels are 0-5 (0 = untrained, 5 = max level)
type TradingSkills struct {
	// Trading Skills
	Accounting              int     // Sales Tax reduction (-10% per level, max -50%)
	BrokerRelations         int     // Broker Fee reduction (-0.3% per level, max -1.5%)
	AdvancedBrokerRelations int     // Additional Broker Fee reduction (-0.3% per level, max -1.5%)
	FactionStanding         float64 // Faction standing (-10.0 to +10.0, affects broker fees: -0.03% per 1.0)
	CorpStanding            float64 // Corp standing (-10.0 to +10.0, affects broker fees: -0.02% per 1.0)

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

	// Racial Hauler Skills (deterministic, from SDE attribute 182)
	// Each grants +5% cargo capacity per level for respective race
	GallenteHauler int // Type ID 3340 - Gallente T1 haulers (Nereus, Iteron Mark V)
	CaldariHauler  int // Type ID 3341 - Caldari T1 haulers (Badger)
	AmarrHauler    int // Type ID 3342 - Amarr T1 haulers (Bestower, Sigil)
	MinmatarHauler int // Type ID 3343 - Minmatar T1 haulers (Wreathe, Hoarder)
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

	// 2. Cache miss - fetch from ESI (skills + standings in parallel for efficiency)
	s.logger.Debug("Skills cache miss - fetching from ESI", "characterID", characterID)
	esiSkills, err := s.fetchSkillsFromESI(ctx, characterID, accessToken)
	if err != nil {
		s.logger.Error("ESI skills fetch failed - using defaults", "error", err, "characterID", characterID)
		// Graceful degradation: return default skills (worst-case fees/cargo)
		return s.getDefaultSkills(), nil
	}

	// 3. Fetch standings from ESI (separate endpoint, best-effort)
	factionStanding, corpStanding := s.fetchStandingsFromESI(ctx, characterID, accessToken)

	// 4. Extract trading skills
	skills := s.extractTradingSkills(esiSkills)
	skills.FactionStanding = factionStanding
	skills.CorpStanding = corpStanding

	// 5. Cache the result (5min TTL)
	if skillsData, err := json.Marshal(skills); err == nil {
		if err := s.redisClient.Set(ctx, cacheKey, skillsData, 5*time.Minute).Err(); err != nil {
			s.logger.Warn("Failed to cache skills", "error", err)
		}
	}

	s.logger.Info("Skills fetched from ESI and cached",
		"characterID", characterID,
		"accounting", skills.Accounting,
		"brokerRelations", skills.BrokerRelations,
		"factionStanding", skills.FactionStanding,
		"corpStanding", skills.CorpStanding,
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

// fetchStandingsFromESI fetches character standings from ESI API
// Returns (factionStanding, corpStanding) - uses max standing per category
// Gracefully degrades to (0.0, 0.0) on error (no impact on fee calculation)
func (s *SkillsService) fetchStandingsFromESI(ctx context.Context, characterID int, accessToken string) (float64, float64) {
	endpoint := fmt.Sprintf("/v2/characters/%d/standings/", characterID)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", "https://esi.evetech.net"+endpoint, nil)
	if err != nil {
		s.logger.Warn("Failed to create standings request", "error", err)
		return 0.0, 0.0
	}

	// Add authorization header
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Execute request through ESI client
	resp, err := s.esiClient.Do(req)
	if err != nil {
		s.logger.Warn("ESI standings request failed - using neutral standings", "error", err)
		return 0.0, 0.0
	}
	defer resp.Body.Close()

	// Handle HTTP errors (401/403 = no standings, treat as neutral)
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		s.logger.Debug("Standings unauthorized - using neutral", "status", resp.StatusCode)
		return 0.0, 0.0
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("ESI standings returned error", "status", resp.StatusCode)
		return 0.0, 0.0
	}

	// Parse JSON response
	var standings []esiStanding
	if err := json.NewDecoder(resp.Body).Decode(&standings); err != nil {
		s.logger.Warn("Failed to parse standings response", "error", err)
		return 0.0, 0.0
	}

	// Extract highest standings per category
	return s.extractHighestStandings(standings)
}

// extractHighestStandings finds the highest standing per category (faction, npc_corp)
// EVE Broker Fee formula uses highest faction and corp standings
// Note: Agent standings are ignored - not relevant for broker fees
func (s *SkillsService) extractHighestStandings(standings []esiStanding) (float64, float64) {
	var maxFactionStanding float64 = 0.0
	var maxCorpStanding float64 = 0.0
	hasFaction := false
	hasCorp := false

	for _, standing := range standings {
		switch standing.FromType {
		case "faction":
			if !hasFaction || standing.Standing > maxFactionStanding {
				maxFactionStanding = standing.Standing
				hasFaction = true
			}
		case "npc_corp":
			if !hasCorp || standing.Standing > maxCorpStanding {
				maxCorpStanding = standing.Standing
				hasCorp = true
			}
		}
	}

	s.logger.Debug("Extracted standings",
		"faction", maxFactionStanding,
		"corp", maxCorpStanding,
	)

	return maxFactionStanding, maxCorpStanding
}

// extractTradingSkills extracts relevant trading skills from ESI skill list
func (s *SkillsService) extractTradingSkills(esiSkills *esiSkillsResponse) *TradingSkills {
	skills := &TradingSkills{
		// Standings are fetched separately and assigned by caller
		FactionStanding: 0.0,
		CorpStanding:    0.0,
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

		// Racial Hauler Skills (Issue #77 - deterministic cargo calculation)
		case 3340: // Gallente Hauler
			skills.GallenteHauler = skill.ActiveSkillLevel
		case 3341: // Caldari Hauler
			skills.CaldariHauler = skill.ActiveSkillLevel
		case 3342: // Amarr Hauler
			skills.AmarrHauler = skill.ActiveSkillLevel
		case 3343: // Minmatar Hauler
			skills.MinmatarHauler = skill.ActiveSkillLevel
		}
	}

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
		CorpStanding:            0.0,
		SpaceshipCommand:        0,
		CargoOptimization:       0,
		Navigation:              0,
		EvasiveManeuvering:      0,
		GallenteIndustrial:      0,
		CaldariIndustrial:       0,
		AmarrIndustrial:         0,
		MinmatarIndustrial:      0,
		GallenteHauler:          0,
		CaldariHauler:           0,
		AmarrHauler:             0,
		MinmatarHauler:          0,
	}
}
