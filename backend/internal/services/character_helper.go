// Package services - Character Helper for Skills and Location
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// CharacterHelper provides character-related ESI operations
type CharacterHelper struct {
	redisClient *redis.Client
}

// NewCharacterHelper creates a new character helper
func NewCharacterHelper(redisClient *redis.Client) *CharacterHelper {
	return &CharacterHelper{
		redisClient: redisClient,
	}
}

// CharacterSkills represents character skills from ESI
type CharacterSkills struct {
	TotalSP int              `json:"total_sp"`
	Skills  []CharacterSkill `json:"skills"`
}

// CharacterSkill represents a single skill
type CharacterSkill struct {
	SkillID            int   `json:"skill_id"`
	ActiveSkillLevel   int   `json:"active_skill_level"`
	TrainedSkillLevel  int   `json:"trained_skill_level"`
	SkillpointsInSkill int64 `json:"skillpoints_in_skill"`
}

// CharacterLocation represents character location from ESI
type CharacterLocation struct {
	SolarSystemID int64  `json:"solar_system_id"`
	StationID     *int64 `json:"station_id,omitempty"`
	StructureID   *int64 `json:"structure_id,omitempty"`
}

// GetCharacterSkills fetches character skills with Redis caching (TTL: 1h)
func (h *CharacterHelper) GetCharacterSkills(ctx context.Context, characterID int, accessToken string) (*CharacterSkills, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("character_skills:%d", characterID)
	cached, err := h.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var skills CharacterSkills
		if err := json.Unmarshal([]byte(cached), &skills); err == nil {
			return &skills, nil
		}
	}

	// Fetch from ESI
	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/skills/", characterID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI skills API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var skills CharacterSkills
	if err := json.Unmarshal(body, &skills); err != nil {
		return nil, err
	}

	// Cache for 1 hour (skills change rarely)
	h.redisClient.Set(ctx, cacheKey, string(body), 1*time.Hour)

	return &skills, nil
}

// GetCharacterLocation fetches character location with Redis caching (TTL: 5min)
func (h *CharacterHelper) GetCharacterLocation(ctx context.Context, characterID int, accessToken string) (*CharacterLocation, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("character_location:%d", characterID)
	cached, err := h.redisClient.Get(ctx, cacheKey).Result()
	if err == nil {
		var location CharacterLocation
		if err := json.Unmarshal([]byte(cached), &location); err == nil {
			return &location, nil
		}
	}

	// Fetch from ESI
	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/location/", characterID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI location API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var location CharacterLocation
	if err := json.Unmarshal(body, &location); err != nil {
		return nil, err
	}

	// Cache for 5 minutes (location can change)
	h.redisClient.Set(ctx, cacheKey, string(body), 5*time.Minute)

	return &location, nil
}

// GetActiveShipTypeID returns the type ID of the ship the character is currently
// flying (ESI /characters/{id}/ship/). Used to make travel-time estimates use
// the active ship's speed when no ship is explicitly selected (sell-from-assets).
func (h *CharacterHelper) GetActiveShipTypeID(ctx context.Context, characterID int, accessToken string) (int, error) {
	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/ship/", characterID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("ESI ship API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ship struct {
		ShipTypeID int `json:"ship_type_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ship); err != nil {
		return 0, err
	}
	return ship.ShipTypeID, nil
}

// GetWalletBalance returns the character's wallet balance in ISK
// (ESI /characters/{id}/wallet/). Cached briefly so the ROI capital prefill
// doesn't hit ESI on every screen open. Never returns 0 on failure — the
// caller must distinguish "broke" from "lookup failed".
func (h *CharacterHelper) GetWalletBalance(ctx context.Context, characterID int, accessToken string) (float64, error) {
	cacheKey := fmt.Sprintf("character_wallet:%d", characterID)

	if cached, err := h.redisClient.Get(ctx, cacheKey).Result(); err == nil {
		if balance, perr := strconv.ParseFloat(cached, 64); perr == nil {
			return balance, nil
		}
	}

	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/wallet/", characterID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("ESI wallet API returned status %d: %s", resp.StatusCode, string(body))
	}

	var balance float64
	if err := json.NewDecoder(resp.Body).Decode(&balance); err != nil {
		return 0, err
	}

	// Balance changes often; the prefill tolerates ~1 min of staleness.
	h.redisClient.Set(ctx, cacheKey, strconv.FormatFloat(balance, 'f', -1, 64), 60*time.Second)

	return balance, nil
}

// CalculateTaxRate calculates broker fee + sales tax based on character skills
func (h *CharacterHelper) CalculateTaxRate(ctx context.Context, characterID int, accessToken string) (float64, error) {
	skills, err := h.GetCharacterSkills(ctx, characterID, accessToken)
	if err != nil {
		// Fail-loud (issue #147 B4): do not fabricate a 5.5% rate that looks real.
		// Return the error so the caller decides how to surface "tax rate unknown".
		return 0, fmt.Errorf("tax rate unavailable (skills fetch failed): %w", err)
	}

	// Find skill levels
	accountingLevel := 0
	brokerRelationsLevel := 0

	for _, skill := range skills.Skills {
		if skill.SkillID == 16622 { // Accounting
			accountingLevel = skill.ActiveSkillLevel
		}
		if skill.SkillID == 3446 { // Broker Relations
			brokerRelationsLevel = skill.ActiveSkillLevel
		}
	}

	// Calculate tax rate
	// Base Broker Fee: 3.0%
	// Base Sales Tax: 2.5%
	baseBrokerFee := 0.03
	baseSalesTax := 0.025

	// Broker Fee reduction: 0.3% per level (0.03 * level)
	brokerFee := baseBrokerFee * (1 - 0.03*float64(brokerRelationsLevel))

	// Sales Tax reduction: 11% per level (0.11 * level)
	salesTax := baseSalesTax * (1 - 0.11*float64(accountingLevel))

	return brokerFee + salesTax, nil
}
