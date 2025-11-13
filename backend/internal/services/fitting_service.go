// Package services - Fitting Service for ship fitting detection and bonus calculations
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	esiclient "github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
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
	CargoBonus          float64 `json:"cargo_bonus_m3"`        // Total effective capacity in m³ (base + skills + modules)
	WarpSpeedMultiplier float64 `json:"warp_speed_multiplier"` // 1.0 = no change (MULTIPLICATIVE)
	InertiaModifier     float64 `json:"inertia_modifier"`      // 1.0 = no change (MULTIPLICATIVE)
	AlignTime           float64 `json:"align_time_seconds"`    // Calculated align time in seconds (NEW: Issue #79)

	// Deterministic Breakdown (Issue #77)
	BaseCargo      float64 `json:"base_cargo_m3"`      // Base cargo from SDE (Attr 38)
	SkillsBonusM3  float64 `json:"skills_bonus_m3"`    // Cargo bonus from skills (absolute m³)
	SkillsBonusPct float64 `json:"skills_bonus_pct"`   // Skill bonus as percentage
	ModulesBonusM3 float64 `json:"modules_bonus_m3"`   // Cargo bonus from modules (absolute m³)
	EffectiveCargo float64 `json:"effective_cargo_m3"` // Final effective capacity

	// Ship Base Attributes (for display when no modules fitted)
	BaseWarpSpeed   float64 `json:"base_warp_speed"`     // Base warp speed in AU/s (e.g., 3.0)
	BaseInertia     float64 `json:"base_inertia"`        // Base inertia modifier (e.g., 1.0)
	WarpSpeedAUS    float64 `json:"warp_speed_au_s"`     // Final warp speed in AU/s (with skills + modules)
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
	esiClient     *esiclient.Client
	sdeDB         *sql.DB
	redisClient   *redis.Client
	skillsService SkillsServicer
	logger        *logger.Logger
}

// NewFittingService creates a new Fitting Service instance
func NewFittingService(
	esiClient *esiclient.Client,
	sdeDB *sql.DB,
	redisClient *redis.Client,
	skillsService SkillsServicer,
	logger *logger.Logger,
) *FittingService {
	return &FittingService{
		esiClient:     esiClient,
		sdeDB:         sdeDB,
		redisClient:   redisClient,
		skillsService: skillsService,
		logger:        logger,
	}
}

// GetShipFitting fetches ship fitting from ESI with caching
// Returns empty fitting (no bonuses) if ESI fails - ensures graceful degradation
func (s *FittingService) GetShipFitting(
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

// InvalidateFittingCache removes fitting data from Redis cache
func (s *FittingService) InvalidateFittingCache(ctx context.Context, characterID int, shipTypeID int) {
	cacheKey := fmt.Sprintf("fitting:%d:%d", characterID, shipTypeID)
	if err := s.redisClient.Del(ctx, cacheKey).Err(); err != nil {
		s.logger.Warn("Failed to invalidate fitting cache", "error", err, "cacheKey", cacheKey)
	} else {
		s.logger.Debug("Fitting cache invalidated", "characterID", characterID, "shipTypeID", shipTypeID)
	}
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

	// 4. Fetch character skills
	skills, err := s.skillsService.GetCharacterSkills(ctx, characterID, accessToken)
	if err != nil {
		s.logger.Warn("Failed to fetch character skills, using default", "error", err)
		skills = nil // Will use graceful degradation in deterministic calculation
	}

	// 5. Convert to cargo.CharacterSkills format (array-based)
	var charSkills *cargo.CharacterSkills
	if skills != nil {
		// Map TradingSkills to ESI CharacterSkills format
		skillsList := []struct {
			SkillID           int64 `json:"skill_id"`
			ActiveSkillLevel  int   `json:"active_skill_level"`
			TrainedSkillLevel int   `json:"trained_skill_level"`
		}{}

		// Add Spaceship Command if present
		if skills.SpaceshipCommand > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3327, ActiveSkillLevel: skills.SpaceshipCommand, TrainedSkillLevel: skills.SpaceshipCommand})
		}

		// Add Racial Industrial Skills
		if skills.GallenteIndustrial > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3348, ActiveSkillLevel: skills.GallenteIndustrial, TrainedSkillLevel: skills.GallenteIndustrial})
		}
		if skills.CaldariIndustrial > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3346, ActiveSkillLevel: skills.CaldariIndustrial, TrainedSkillLevel: skills.CaldariIndustrial})
		}
		if skills.AmarrIndustrial > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3347, ActiveSkillLevel: skills.AmarrIndustrial, TrainedSkillLevel: skills.AmarrIndustrial})
		}
		if skills.MinmatarIndustrial > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3349, ActiveSkillLevel: skills.MinmatarIndustrial, TrainedSkillLevel: skills.MinmatarIndustrial})
		}

		// Add Racial Hauler Skills (Issue #77 - deterministic)
		if skills.GallenteHauler > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3340, ActiveSkillLevel: skills.GallenteHauler, TrainedSkillLevel: skills.GallenteHauler})
		}
		if skills.CaldariHauler > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3341, ActiveSkillLevel: skills.CaldariHauler, TrainedSkillLevel: skills.CaldariHauler})
		}
		if skills.AmarrHauler > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3342, ActiveSkillLevel: skills.AmarrHauler, TrainedSkillLevel: skills.AmarrHauler})
		}
		if skills.MinmatarHauler > 0 {
			skillsList = append(skillsList, struct {
				SkillID           int64 `json:"skill_id"`
				ActiveSkillLevel  int   `json:"active_skill_level"`
				TrainedSkillLevel int   `json:"trained_skill_level"`
			}{SkillID: 3343, ActiveSkillLevel: skills.MinmatarHauler, TrainedSkillLevel: skills.MinmatarHauler})
		}

		charSkills = &cargo.CharacterSkills{
			Skills: skillsList,
		}
	}

	// 6. Convert fitted modules to cargo.FittedItem format
	fittedItems := make([]cargo.FittedItem, 0, len(fittedModules))
	for _, mod := range fittedModules {
		fittedItems = append(fittedItems, cargo.FittedItem{
			TypeID: int64(mod.TypeID),
			Slot:   mod.Slot,
		})
	}

	// 7. Calculate deterministic cargo capacity
	capacities, err := cargo.GetShipCapacitiesDeterministic(
		ctx,
		s.sdeDB,
		int64(shipTypeID),
		charSkills,
		fittedItems,
	)
	if err != nil {
		s.logger.Error("Deterministic capacity calculation failed", "error", err)
		// Fallback to basic calculation without bonuses
		return &FittingData{
			ShipTypeID:    shipTypeID,
			FittedModules: fittedModules,
			Bonuses: FittingBonuses{
				CargoBonus:          0,
				WarpSpeedMultiplier: 1,
				InertiaModifier:     1,
				BaseCargo:           0,
				SkillsBonusM3:       0,
				SkillsBonusPct:      0,
				ModulesBonusM3:      0,
				EffectiveCargo:      0,
			},
		}, nil
	}

	// 8. Convert to FittingBonuses format with deterministic breakdown
	// CargoBonus = EffectiveCargoHold (total effective capacity in m³)
	// Frontend displays this as "Cargo Bonus" but it's actually total effective capacity
	cargoBonus := capacities.EffectiveCargoHold

	// Calculate breakdown from AppliedBonuses
	var skillsBonusM3 float64
	var skillsBonusPct float64
	var modulesBonusM3 float64

	baseCargo := capacities.BaseCargoHold
	for _, bonus := range capacities.AppliedBonuses {
		switch bonus.Source {
		case "Skill":
			// Skills are percentage bonuses
			skillsBonusPct += bonus.Value
			skillsBonusM3 = baseCargo * (skillsBonusPct / 100.0)
		case "Module", "Rig":
			// Modules/Rigs are multiplicative - calculate absolute bonus
			modulesBonusM3 += bonus.Value
		}
	}

	effectiveCargo := capacities.EffectiveCargoHold

	// 9. Get ship base attributes (warp speed, inertia) from SDE
	baseWarpSpeedMultiplier, _, baseInertia, err := s.getShipBaseAttributes(ctx, int64(shipTypeID))
	if err != nil {
		s.logger.Warn("Failed to get ship base attributes", "error", err)
		// Use fallback defaults
		baseWarpSpeedMultiplier = 3.0
		baseInertia = 1.0
	}

	// Base warp speed is 1 AU/s × multiplier from SDE (e.g., 3.0 for cruisers)
	baseWarpSpeed := 1.0 * baseWarpSpeedMultiplier

	// 10. Calculate deterministic Warp Speed (Issue #78 - with skills + modules + stacking penalties)
	var effectiveWarpSpeed float64
	warpSpeedResult, err := navigation.GetShipWarpSpeedDeterministic(
		ctx,
		s.sdeDB,
		int64(shipTypeID),
		charSkills,
		fittedItems,
	)
	if err != nil {
		s.logger.Warn("Failed to calculate warp speed deterministically, using fallback", "error", err)
		effectiveWarpSpeed = baseWarpSpeed // Fallback to base speed
	} else {
		effectiveWarpSpeed = warpSpeedResult.EffectiveWarpSpeed
	}

	// 11. Calculate deterministic Inertia + Align Time (Issue #79 - with skills + modules + stacking penalties)
	var effectiveInertia, alignTime float64
	inertiaResult, err := navigation.GetShipInertiaDeterministic(
		ctx,
		s.sdeDB,
		int64(shipTypeID),
		charSkills,
		fittedItems,
	)
	if err != nil {
		s.logger.Warn("Failed to calculate inertia deterministically, using fallback", "error", err)
		effectiveInertia = baseInertia // Fallback to base inertia
		alignTime = 0                  // Unknown align time
	} else {
		effectiveInertia = inertiaResult.EffectiveInertia
		alignTime = inertiaResult.AlignTime
	}

	return &FittingData{
		ShipTypeID:    shipTypeID,
		FittedModules: fittedModules,
		Bonuses: FittingBonuses{
			CargoBonus:          cargoBonus,
			WarpSpeedMultiplier: effectiveWarpSpeed / baseWarpSpeed, // Multiplier for legacy compatibility
			InertiaModifier:     effectiveInertia,                   // Absolute inertia value
			AlignTime:           alignTime,                          // Calculated align time in seconds
			// Deterministic breakdown
			BaseCargo:      baseCargo,
			SkillsBonusM3:  skillsBonusM3,
			SkillsBonusPct: skillsBonusPct,
			ModulesBonusM3: modulesBonusM3,
			EffectiveCargo: effectiveCargo,
			// Ship attributes
			BaseWarpSpeed: baseWarpSpeed,
			BaseInertia:   baseInertia,
			WarpSpeedAUS:  effectiveWarpSpeed, // Final warp speed in AU/s (for route calculation)
		},
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
	// Query SDE for dogma attributes (stored as JSON in typeDogma table)
	// Attributes we care about: 38 (Cargo Bonus), 20 (Warp Speed), 70 (Inertia), 4 (Volume)
	query := `SELECT dogmaAttributes FROM typeDogma WHERE _key = ?`

	var dogmaJSON string
	err := s.sdeDB.QueryRowContext(ctx, query, typeID).Scan(&dogmaJSON)
	if err != nil {
		return nil, "", fmt.Errorf("SDE query failed: %w", err)
	}

	// Parse JSON array of dogma attributes
	var attributes []struct {
		AttributeID int     `json:"attributeID"`
		Value       float64 `json:"value"`
	}
	if err := json.Unmarshal([]byte(dogmaJSON), &attributes); err != nil {
		return nil, "", fmt.Errorf("JSON parse failed: %w", err)
	}

	// Extract relevant attributes
	// 38: capacity (not a bonus, but module cargo capacity)
	// 20: warpSpeedMultiplier
	// 70: inertiaModifier
	// 4: volume
	// 614: cargoCapacityBonus (%-based cargo bonus for rigs)
	dogmaAttribs := make(map[int]float64)
	relevantAttribs := map[int]bool{38: true, 20: true, 70: true, 4: true, 614: true}

	for _, attr := range attributes {
		if relevantAttribs[attr.AttributeID] {
			dogmaAttribs[attr.AttributeID] = attr.Value
		}
	}

	// Get type name from types table (name is JSON with all languages)
	var nameJSON string
	nameQuery := `SELECT name FROM types WHERE _key = ?`
	if err := s.sdeDB.QueryRowContext(ctx, nameQuery, typeID).Scan(&nameJSON); err != nil {
		return dogmaAttribs, fmt.Sprintf("Unknown (Type %d)", typeID), nil
	}

	// Parse name JSON and extract English name
	var names map[string]string
	if err := json.Unmarshal([]byte(nameJSON), &names); err != nil {
		return dogmaAttribs, fmt.Sprintf("Unknown (Type %d)", typeID), nil
	}

	// Prefer English, fallback to first available
	typeName := names["en"]
	if typeName == "" {
		for _, name := range names {
			typeName = name
			break
		}
	}
	if typeName == "" {
		typeName = fmt.Sprintf("Unknown (Type %d)", typeID)
	}

	return dogmaAttribs, typeName, nil
}

// getShipBaseAttributes retrieves base warp speed, mass, and inertia from SDE
// Returns: warpSpeed (AU/s), mass (kg), inertiaModifier, error
func (s *FittingService) getShipBaseAttributes(ctx context.Context, shipTypeID int64) (float64, float64, float64, error) {
	// Query typeDogma for ship base attributes
	// Attribute 600: warpSpeedMultiplier (base warp speed, e.g., 1.0 for most ships)
	// Attribute 4: mass (kg)
	// Attribute 70: inertiaModifier (base inertia)
	query := `SELECT dogmaAttributes FROM typeDogma WHERE _key = ?`

	var dogmaJSON string
	err := s.sdeDB.QueryRowContext(ctx, query, shipTypeID).Scan(&dogmaJSON)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("SDE query failed: %w", err)
	}

	// Parse JSON array
	var attributes []struct {
		AttributeID int     `json:"attributeID"`
		Value       float64 `json:"value"`
	}
	if err := json.Unmarshal([]byte(dogmaJSON), &attributes); err != nil {
		return 0, 0, 0, fmt.Errorf("JSON parse failed: %w", err)
	}

	// Extract attributes
	var warpSpeed, mass, inertia float64
	for _, attr := range attributes {
		switch attr.AttributeID {
		case 600: // warpSpeedMultiplier (base)
			warpSpeed = attr.Value
		case 4: // mass
			mass = attr.Value
		case 70: // inertiaModifier
			inertia = attr.Value
		}
	}

	// Defaults if not found
	if warpSpeed == 0 {
		warpSpeed = 3.0 // Default cruiser/hauler warp speed (1 AU/s base × 3 multiplier)
	}
	if inertia == 0 {
		inertia = 1.0
	}

	return warpSpeed, mass, inertia, nil
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
			BaseCargo:           0.0,
			SkillsBonusM3:       0.0,
			SkillsBonusPct:      0.0,
			ModulesBonusM3:      0.0,
			EffectiveCargo:      0.0,
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
