// Package services - Fitting Service for ship fitting detection and bonus calculations
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"time"

	esiclient "github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// esiAsset represents ESI /v5/characters/{id}/assets/ response item
type esiAsset struct {
	ItemID       int64  `json:"item_id"`
	TypeID       int    `json:"type_id"`
	LocationID   int64  `json:"location_id"`
	LocationFlag string `json:"location_flag"`
	LocationType string `json:"location_type"`
	IsSingleton  bool   `json:"is_singleton"`
	Quantity     int    `json:"quantity"`
}

// FittedModule represents a single fitted module with dogma attributes
type FittedModule struct {
	TypeID       int             `json:"type_id"`
	TypeName     string          `json:"type_name"`
	Slot         string          `json:"slot"` // HiSlot0-7, MedSlot0-7, LoSlot0-7, RigSlot0-2
	DogmaAttribs map[int]float64 `json:"dogma_attributes"`
}

// FittingBonuses contains aggregated bonuses from all fitted modules
type FittingBonuses struct {
	CargoBonus          float64 `json:"cargo_bonus"`           // m³ (ADDITIVE)
	WarpSpeedMultiplier float64 `json:"warp_speed_multiplier"` // 1.0 = no change (MULTIPLICATIVE)
	InertiaModifier     float64 `json:"inertia_modifier"`      // 1.0 = no change (MULTIPLICATIVE)
}

// FittingData contains all fitting information for a ship
type FittingData struct {
	ShipTypeID     int            `json:"ship_type_id"`
	FittedModules  []FittedModule `json:"fitted_modules"`
	Bonuses        FittingBonuses `json:"bonuses"`
	Cached         bool           `json:"cached"`
	CacheExpiresAt time.Time      `json:"cache_expires_at,omitempty"`
}

// FittingService provides ship fitting detection and bonus calculations
type FittingService struct {
	esiClient   *esiclient.Client
	sdeDB       *sql.DB
	redisClient *redis.Client
	logger      *logger.Logger
}

// NewFittingService creates a new Fitting Service instance
func NewFittingService(
	esiClient *esiclient.Client,
	sdeDB *sql.DB,
	redisClient *redis.Client,
	logger *logger.Logger,
) *FittingService {
	return &FittingService{
		esiClient:   esiClient,
		sdeDB:       sdeDB,
		redisClient: redisClient,
		logger:      logger,
	}
}

// GetCharacterFitting fetches ship fitting from ESI with caching
// Returns empty fitting (no bonuses) if ESI fails - ensures graceful degradation
func (s *FittingService) GetCharacterFitting(
	ctx context.Context,
	characterID int,
	shipTypeID int,
	accessToken string,
) (*FittingData, error) {
	// 1. Check Redis cache first
	cacheKey := fmt.Sprintf("fitting:%d:%d", characterID, shipTypeID)
	cachedData, err := s.redisClient.Get(ctx, cacheKey).Bytes()
	if err == nil {
		s.logger.Debug("Fitting cache hit", "characterID", characterID, "shipTypeID", shipTypeID)
		var fitting FittingData
		if err := json.Unmarshal(cachedData, &fitting); err == nil {
			fitting.Cached = true
			return &fitting, nil
		}
		s.logger.Warn("Failed to unmarshal cached fitting", "error", err)
	}

	// 2. Cache miss - fetch from ESI
	s.logger.Debug("Fitting cache miss - fetching from ESI", "characterID", characterID, "shipTypeID", shipTypeID)

	fitting, err := s.fetchFittingFromESI(ctx, characterID, shipTypeID, accessToken)
	if err != nil {
		// Graceful degradation: Return empty fitting on error
		s.logger.Error("Failed to fetch fitting from ESI", "error", err, "characterID", characterID, "shipTypeID", shipTypeID)
		return s.getDefaultFitting(shipTypeID), nil
	}

	// 3. Cache the result (5 minutes TTL, same as SkillsService)
	cacheData, err := json.Marshal(fitting)
	if err == nil {
		expiration := 5 * time.Minute
		if err := s.redisClient.Set(ctx, cacheKey, cacheData, expiration).Err(); err != nil {
			s.logger.Warn("Failed to cache fitting", "error", err)
		}
		fitting.CacheExpiresAt = time.Now().Add(expiration)
	}

	fitting.Cached = false
	return fitting, nil
}

// fetchFittingFromESI fetches assets from ESI and filters for fitted modules
func (s *FittingService) fetchFittingFromESI(
	ctx context.Context,
	characterID int,
	shipTypeID int,
	accessToken string,
) (*FittingData, error) {
	// 1. Fetch character assets from ESI
	assets, err := s.fetchESIAssets(ctx, characterID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ESI assets: %w", err)
	}

	// 2. Find the ship instance by type_id
	var shipItemID int64
	for _, asset := range assets {
		if asset.TypeID == shipTypeID && asset.IsSingleton {
			shipItemID = asset.ItemID
			break
		}
	}

	if shipItemID == 0 {
		// Ship not found in assets
		return s.getDefaultFitting(shipTypeID), nil
	}

	// 3. Filter fitted modules (modules where location_id == ship_item_id)
	fittedModules := []FittedModule{}
	for _, asset := range assets {
		if asset.LocationID == shipItemID && isFittedSlot(asset.LocationFlag) {
			// Fetch dogma attributes for this module
			dogmaAttribs, typeName, err := s.fetchDogmaAttributes(ctx, asset.TypeID)
			if err != nil {
				s.logger.Warn("Failed to fetch dogma attributes", "typeID", asset.TypeID, "error", err)
				continue
			}

			fittedModules = append(fittedModules, FittedModule{
				TypeID:       asset.TypeID,
				TypeName:     typeName,
				Slot:         asset.LocationFlag,
				DogmaAttribs: dogmaAttribs,
			})
		}
	}

	// 4. Calculate bonuses
	bonuses := s.calculateBonuses(fittedModules)

	return &FittingData{
		ShipTypeID:    shipTypeID,
		FittedModules: fittedModules,
		Bonuses:       bonuses,
	}, nil
}

// fetchESIAssets fetches character assets from ESI /v5/characters/{id}/assets/
func (s *FittingService) fetchESIAssets(
	ctx context.Context,
	characterID int,
	accessToken string,
) ([]esiAsset, error) {
	endpoint := fmt.Sprintf("/latest/characters/%d/assets/", characterID)

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

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI returned status %d: %s", resp.StatusCode, string(body))
	}

	var assets []esiAsset
	if err := json.NewDecoder(resp.Body).Decode(&assets); err != nil {
		return nil, fmt.Errorf("failed to decode ESI response: %w", err)
	}

	return assets, nil
}

// fetchDogmaAttributes queries SDE for dogma attributes of a module
// Returns: map[attributeID]value, typeName, error
func (s *FittingService) fetchDogmaAttributes(ctx context.Context, typeID int) (map[int]float64, string, error) {
	// Query SDE for dogma attributes
	// Attributes: 38 (Cargo Bonus), 20 (Warp Speed), 70 (Inertia), 4 (Volume)
	query := `
		SELECT 
			td.attribute_id,
			td.value,
			t.type_name
		FROM type_dogma td
		JOIN types t ON t.type_id = td.type_id
		WHERE td.type_id = ?
			AND td.attribute_id IN (38, 20, 70, 4)
	`

	rows, err := s.sdeDB.QueryContext(ctx, query, typeID)
	if err != nil {
		return nil, "", fmt.Errorf("SDE query failed: %w", err)
	}
	defer rows.Close()

	dogmaAttribs := make(map[int]float64)
	var typeName string

	for rows.Next() {
		var attributeID int
		var value float64
		var name string
		if err := rows.Scan(&attributeID, &value, &name); err != nil {
			return nil, "", fmt.Errorf("row scan failed: %w", err)
		}
		dogmaAttribs[attributeID] = value
		typeName = name
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration failed: %w", err)
	}

	// If no type name found, fallback query
	if typeName == "" {
		nameQuery := `SELECT type_name FROM types WHERE type_id = ?`
		if err := s.sdeDB.QueryRowContext(ctx, nameQuery, typeID).Scan(&typeName); err != nil {
			typeName = fmt.Sprintf("Unknown (Type %d)", typeID)
		}
	}

	return dogmaAttribs, typeName, nil
}

// calculateBonuses aggregates bonuses from all fitted modules
// Applies EVE Online stacking penalties: S(u) = e^(-(u/2.67)^2)
// where u is 0-based position after sorting by bonus strength (descending)
func (s *FittingService) calculateBonuses(modules []FittedModule) FittingBonuses {
	// Group modules by attribute ID
	cargoMods := []float64{}
	warpMods := []float64{}
	inertiaMods := []float64{}

	for _, mod := range modules {
		// Cargo bonus (Attribute 38)
		if cargoBonus, ok := mod.DogmaAttribs[38]; ok {
			cargoMods = append(cargoMods, cargoBonus)
		}
		// Warp speed multiplier (Attribute 20)
		if warpBonus, ok := mod.DogmaAttribs[20]; ok {
			warpMods = append(warpMods, warpBonus)
		}
		// Inertia modifier (Attribute 70)
		if inertiaBonus, ok := mod.DogmaAttribs[70]; ok {
			inertiaMods = append(inertiaMods, inertiaBonus)
		}
	}

	// Apply stacking penalties and aggregate
	bonuses := FittingBonuses{
		CargoBonus:          applyStackingPenalty(cargoMods),
		WarpSpeedMultiplier: 1 + applyStackingPenalty(warpMods),
		InertiaModifier:     1 + applyStackingPenalty(inertiaMods),
	}

	return bonuses
}

// applyStackingPenalty applies EVE Online stacking penalty formula
// Formula: S(u) = e^(-(u/2.67)^2) where u = position (0-based)
// 1st module: 100% effectiveness (S(0) = 1.0)
// 2nd module: ~86.9% effectiveness (S(1) ≈ 0.869)
// 3rd module: ~57.1% effectiveness (S(2) ≈ 0.571)
// 4th module: ~28.3% effectiveness (S(3) ≈ 0.283)
func applyStackingPenalty(bonuses []float64) float64 {
	if len(bonuses) == 0 {
		return 0.0
	}

	// Sort bonuses descending (strongest first)
	sort.Sort(sort.Reverse(sort.Float64Slice(bonuses)))

	result := 0.0
	for i, bonus := range bonuses {
		// EVE Online formula: S(u) = e^(-(u/2.67)^2)
		u := float64(i)
		penalty := math.Exp(-math.Pow(u/2.67, 2))
		result += bonus * penalty
	}

	return result
}

// getDefaultFitting returns empty fitting with no bonuses (graceful degradation)
func (s *FittingService) getDefaultFitting(shipTypeID int) *FittingData {
	return &FittingData{
		ShipTypeID:    shipTypeID,
		FittedModules: []FittedModule{},
		Bonuses: FittingBonuses{
			CargoBonus:          0.0,
			WarpSpeedMultiplier: 1.0,
			InertiaModifier:     1.0,
		},
	}
}

// isFittedSlot checks if a location_flag represents a fitted module slot
func isFittedSlot(locationFlag string) bool {
	fittedSlots := map[string]bool{
		// High slots
		"HiSlot0": true, "HiSlot1": true, "HiSlot2": true, "HiSlot3": true,
		"HiSlot4": true, "HiSlot5": true, "HiSlot6": true, "HiSlot7": true,
		// Med slots
		"MedSlot0": true, "MedSlot1": true, "MedSlot2": true, "MedSlot3": true,
		"MedSlot4": true, "MedSlot5": true, "MedSlot6": true, "MedSlot7": true,
		// Low slots
		"LoSlot0": true, "LoSlot1": true, "LoSlot2": true, "LoSlot3": true,
		"LoSlot4": true, "LoSlot5": true, "LoSlot6": true, "LoSlot7": true,
		// Rig slots
		"RigSlot0": true, "RigSlot1": true, "RigSlot2": true,
	}
	return fittedSlots[locationFlag]
}
