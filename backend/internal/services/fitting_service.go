// Package services - Fitting Service for ship fitting detection and bonus calculations
package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"sort"
	"time"

	esiclient "github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services/esiconfig"
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
	BaseWarpSpeed float64 `json:"base_warp_speed"` // Base warp speed in AU/s (e.g., 3.0)
	BaseInertia   float64 `json:"base_inertia"`    // Base inertia modifier (e.g., 1.0)
	WarpSpeedAUS  float64 `json:"warp_speed_au_s"` // Final warp speed in AU/s (with skills + modules)
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
		// Fail-loud (issue #147 B3): return the error too so callers can tell
		// "ESI down" apart from the legitimate "ship has no modules fitted" case
		// (which fetchFittingFromESI still returns as default fitting + nil). We
		// still return the default fitting payload so a caller may degrade if it
		// chooses, but the non-nil error is the explicit signal.
		s.logger.Error("Failed to fetch fitting from ESI", "error", err, "characterID", characterID, "shipTypeID", shipTypeID)
		return s.getDefaultFitting(shipTypeID), fmt.Errorf("fitting unavailable: %w", err)
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

	// 4. + 5. Fetch character skills and convert to cargo.CharacterSkills format
	charSkills := s.characterCargoSkills(ctx, characterID, accessToken)

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
		s.logger.Warn("Deterministic capacity calculation failed, using fallback", "error", err)
		// Continue with fallback - try to get at least base values from SDE
	}

	// 8. Get ship base attributes (warp speed, inertia, cargo) from SDE
	baseWarpSpeedMultiplier, _, baseInertia, err := s.getShipBaseAttributes(ctx, int64(shipTypeID))
	if err != nil {
		s.logger.Warn("Failed to get ship base attributes", "error", err)
		// Use fallback defaults
		baseWarpSpeedMultiplier = 3.0
		baseInertia = 1.0
	}

	// Determine cargo values (either from deterministic calculation or fallback)
	var baseCargo, effectiveCargo, cargoBonus float64
	var skillsBonusM3, skillsBonusPct, modulesBonusM3 float64

	if capacities != nil {
		// Deterministic calculation succeeded
		cargoBonus = capacities.EffectiveCargoHold
		baseCargo = capacities.BaseCargoHold
		effectiveCargo = capacities.EffectiveCargoHold

		// Calculate breakdown from AppliedBonuses
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
	} else {
		// Fallback: Try to get base cargo from SDE view
		var baseCapacity float64
		err = s.sdeDB.QueryRowContext(ctx, `
			SELECT COALESCE(base_cargo_capacity, 0)
			FROM v_ship_cargo_capacities
			WHERE ship_type_id = ?
		`, int64(shipTypeID)).Scan(&baseCapacity)
		if err != nil {
			s.logger.Warn("Failed to get base cargo capacity from SDE", "error", err)
			baseCapacity = 0
		}
		baseCargo = baseCapacity
		effectiveCargo = baseCapacity
		cargoBonus = baseCapacity
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

// characterCargoSkills fetches the character's skills via SkillsServicer and converts
// the cargo-relevant ones (Spaceship Command, racial Industrial + Hauler skills) into
// the array-based cargo.CharacterSkills format used by the deterministic calculators.
// Returns nil on a skills-fetch failure (the calculators degrade gracefully on nil).
func (s *FittingService) characterCargoSkills(ctx context.Context, characterID int, accessToken string) *cargo.CharacterSkills {
	skills, err := s.skillsService.GetCharacterSkills(ctx, characterID, accessToken)
	if err != nil {
		s.logger.Warn("Failed to fetch character skills, using default", "error", err)
		return nil
	}
	if skills == nil {
		return nil
	}

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

	return &cargo.CharacterSkills{Skills: skillsList}
}

// EnrichShipsEffectiveCargo fills each singleton ship's EffectiveCargoCapacity (from its
// OWN fitting by item_id), EffectiveCargoUnavailable (true on a calc error → fail-loud),
// and Name (custom ESI name, falling back to TypeName). Best-effort: an assets-fetch error
// leaves the ships unenriched rather than failing the whole list.
func (s *FittingService) EnrichShipsEffectiveCargo(ctx context.Context, characterID int, ships []models.CharacterAssetShip, accessToken string) {
	assets, err := s.fetchESIAssets(ctx, characterID, accessToken)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("enrich ships: assets fetch failed", "characterID", characterID, "error", err)
		}
		return
	}
	charSkills := s.characterCargoSkills(ctx, characterID, accessToken)

	itemIDs := make([]int64, 0, len(ships))
	for i := range ships {
		itemIDs = append(itemIDs, ships[i].ItemID)
	}
	names := s.FetchAssetNames(ctx, characterID, itemIDs, accessToken)

	for i := range ships {
		if n := names[ships[i].ItemID]; n != "" {
			ships[i].Name = n
		} else {
			ships[i].Name = ships[i].TypeName
		}
		if !ships[i].IsSingleton {
			continue
		}
		eff, unavail := s.EffectiveCargoForShipItem(ctx, int(ships[i].TypeID), ships[i].ItemID, assets, charSkills)
		if eff > 0 {
			ships[i].EffectiveCargoCapacity = eff
		}
		ships[i].EffectiveCargoUnavailable = unavail
	}
}

// EffectiveCargoForActiveShip computes the effective cargo for a single ship instance by
// item_id (used for the flown/active ship). Returns (effective, unavailable). On an
// assets/calc error returns (0, true) so the caller fails loud (falls back to base + flag).
func (s *FittingService) EffectiveCargoForActiveShip(ctx context.Context, characterID, shipTypeID int, shipItemID int64, accessToken string) (float64, bool) {
	assets, err := s.fetchESIAssets(ctx, characterID, accessToken)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("active ship effective cargo: assets fetch failed", "error", err)
		}
		return 0, true
	}
	charSkills := s.characterCargoSkills(ctx, characterID, accessToken)
	return s.EffectiveCargoForShipItem(ctx, shipTypeID, shipItemID, assets, charSkills)
}

// ActiveShipFittedModuleTypeIDs returns the TYPE IDs of the modules fitted to the
// character's current/active ship (ESI /characters/{id}/ship/). It resolves the active
// ship's item id, then collects every fitted module's TypeID from the asset list (via
// the same fitted-slot filter used for effective-cargo). Returns an empty slice (no error)
// when the active ship has no fitted modules; an error only on an ESI assets/ship failure.
func (s *FittingService) ActiveShipFittedModuleTypeIDs(ctx context.Context, characterID int, accessToken string) ([]int64, error) {
	shipItemID, err := s.fetchActiveShipItemID(ctx, characterID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("active ship lookup failed: %w", err)
	}
	assets, err := s.fetchESIAssets(ctx, characterID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("active ship modules: assets fetch failed: %w", err)
	}
	items := s.fittedItemsForShip(ctx, assets, shipItemID)
	out := make([]int64, 0, len(items))
	for _, it := range items {
		out = append(out, it.TypeID)
	}
	return out, nil
}

// fetchActiveShipItemID resolves the item id of the ship the character is currently
// flying (ESI /characters/{id}/ship/ → ship_item_id).
func (s *FittingService) fetchActiveShipItemID(ctx context.Context, characterID int, accessToken string) (int64, error) {
	endpoint := fmt.Sprintf("/latest/characters/%d/ship/", characterID)
	req, err := http.NewRequestWithContext(ctx, "GET", esiconfig.BaseURL+endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("create ship request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.esiClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("esi ship request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("ESI ship returned status %d: %s", resp.StatusCode, string(body))
	}
	var ship struct {
		ShipItemID int64 `json:"ship_item_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ship); err != nil {
		return 0, fmt.Errorf("decode ship response: %w", err)
	}
	return ship.ShipItemID, nil
}

// fetchESIAssets fetches character assets from ESI /v5/characters/{id}/assets/
func (s *FittingService) fetchESIAssets(
	ctx context.Context,
	characterID int,
	accessToken string,
) ([]esiAsset, error) {
	endpoint := fmt.Sprintf("/latest/characters/%d/assets/", characterID)

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", esiconfig.BaseURL+endpoint, nil)
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
	// No player fitting available (ship not owned / ESI failure). Fall back to
	// the hull's BASE cargo from the SDE so callers still get a ship-specific
	// capacity — returning 0 here would make every such ship look identical
	// (and break route/ROI calc, which divides cargo by item volume).
	ctx := context.Background()
	baseCargo := 0.0
	if caps, err := cargo.GetShipCapacities(s.sdeDB, int64(shipTypeID), nil); err == nil && caps != nil {
		baseCargo = caps.BaseCargoHold
	}
	// Also derive the hull's base warp speed + align time (no skills/modules) so
	// route/ROI travel time is ship-specific even without a player fitting.
	warpAUS, alignTime := 0.0, 0.0
	if ws, err := navigation.GetShipWarpSpeedDeterministic(ctx, s.sdeDB, int64(shipTypeID), nil, nil); err == nil && ws != nil {
		warpAUS = ws.EffectiveWarpSpeed
	}
	if ir, err := navigation.GetShipInertiaDeterministic(ctx, s.sdeDB, int64(shipTypeID), nil, nil); err == nil && ir != nil {
		alignTime = ir.AlignTime
	}
	return &FittingData{
		ShipTypeID:    shipTypeID,
		FittedModules: []FittedModule{},
		Bonuses: FittingBonuses{
			CargoBonus:          baseCargo,
			WarpSpeedMultiplier: 1.0,
			InertiaModifier:     1.0,
			AlignTime:           alignTime,
			BaseCargo:           baseCargo,
			SkillsBonusM3:       0.0,
			SkillsBonusPct:      0.0,
			ModulesBonusM3:      0.0,
			EffectiveCargo:      baseCargo,
			WarpSpeedAUS:        warpAUS,
		},
	}
}

// fittedItemsForShip collects cargo.FittedItem entries for all modules fitted to a specific
// ship item. It uses only the asset list (no ESI dogma lookups), making it suitable for
// lightweight per-instance effective-cargo calculations.
func (s *FittingService) fittedItemsForShip(ctx context.Context, assets []esiAsset, shipItemID int64) []cargo.FittedItem {
	items := make([]cargo.FittedItem, 0)
	for _, asset := range assets {
		if asset.LocationID == shipItemID && isFittedSlot(asset.LocationFlag) {
			items = append(items, cargo.FittedItem{TypeID: int64(asset.TypeID), Slot: asset.LocationFlag})
		}
	}
	return items
}

// EffectiveCargoForShipItem computes the effective cargo hold for a specific ship instance
// (identified by shipItemID) using the supplied asset list and character skills.
// Returns (effectiveCargo, unavailable): unavailable is true when the SDE lookup fails.
func (s *FittingService) EffectiveCargoForShipItem(
	ctx context.Context, shipTypeID int, shipItemID int64, assets []esiAsset, charSkills *cargo.CharacterSkills,
) (float64, bool) {
	items := s.fittedItemsForShip(ctx, assets, shipItemID)
	caps, err := cargo.GetShipCapacitiesDeterministic(ctx, s.sdeDB, int64(shipTypeID), charSkills, items)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("effective cargo per-item calc failed", "shipTypeID", shipTypeID, "itemID", shipItemID, "error", err)
		}
		return 0, true
	}
	return caps.EffectiveCargoHold, false
}

// parseAssetNames parses the JSON body from ESI /assets/names/ into a map
// of item_id → name. Entries with empty name or the sentinel "None" are
// dropped so callers can fall back to the type name.
func parseAssetNames(body []byte) (map[int64]string, error) {
	var rows []struct {
		ItemID int64  `json:"item_id"`
		Name   string `json:"name"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode asset names: %w", err)
	}
	out := make(map[int64]string, len(rows))
	for _, r := range rows {
		if r.Name == "" || r.Name == "None" {
			continue
		}
		out[r.ItemID] = r.Name
	}
	return out, nil
}

// assetNamesMaxIDs is ESI's per-request id cap for /assets/names/ (max 1000 ids).
const assetNamesMaxIDs = 1000

// assetNamesCacheTTL is the Redis TTL for cached asset-name results (1 h, matching the assets cache window).
const assetNamesCacheTTL = time.Hour

// assetNamesCacheKey returns a stable Redis key for the given characterID and item-id set.
// The key is independent of input order: the ids are sorted before hashing.
func assetNamesCacheKey(characterID int, itemIDs []int64) string {
	sorted := make([]int64, len(itemIDs))
	copy(sorted, itemIDs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	h := fnv.New32a()
	for _, id := range sorted {
		// Write each id as 8 bytes (big-endian equivalent via repeated shifts).
		b := [8]byte{
			byte(id >> 56), byte(id >> 48), byte(id >> 40), byte(id >> 32),
			byte(id >> 24), byte(id >> 16), byte(id >> 8), byte(id),
		}
		_, _ = h.Write(b[:])
	}
	return fmt.Sprintf("assetnames:%d:%x", characterID, h.Sum32())
}

// chunkInt64 splits ids into consecutive slices of at most size elements.
// A size <= 0 yields a single chunk with all ids.
func chunkInt64(ids []int64, size int) [][]int64 {
	if size <= 0 || len(ids) <= size {
		return [][]int64{ids}
	}
	chunks := make([][]int64, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		chunks = append(chunks, ids[start:end])
	}
	return chunks
}

// FetchAssetNames resolves custom names for the given item_ids (best-effort: on any
// error returns whatever partial names were resolved so labels fall back to the type
// name — never blocks the list).
// Results are cached in Redis (1 h TTL) keyed by characterID + sorted item-id set,
// but ONLY when every ESI batch succeeded. A transient ESI error must NOT poison the
// cache for the full TTL.
// On a Redis error the function falls through to ESI transparently.
func (s *FittingService) FetchAssetNames(ctx context.Context, characterID int, itemIDs []int64, accessToken string) map[int64]string {
	if len(itemIDs) == 0 {
		return map[int64]string{}
	}

	cacheKey := assetNamesCacheKey(characterID, itemIDs)

	// 1. Try Redis cache.
	if cachedBytes, err := s.redisClient.Get(ctx, cacheKey).Bytes(); err == nil {
		var names map[int64]string
		if err := json.Unmarshal(cachedBytes, &names); err == nil {
			return names
		}
		if s.logger != nil {
			s.logger.Warn("asset names: failed to unmarshal cached entry", "key", cacheKey)
		}
	}

	// 2. Cache miss (or Redis error) — fetch from ESI.
	names, ok := s.fetchAssetNamesFromESI(ctx, characterID, itemIDs, accessToken)

	// 3. Store to Redis only when all batches succeeded (fail-loud: don't cache ESI errors).
	if ok {
		if data, err := json.Marshal(names); err == nil {
			if err := s.redisClient.Set(ctx, cacheKey, data, assetNamesCacheTTL).Err(); err != nil {
				if s.logger != nil {
					s.logger.Warn("asset names: failed to cache in Redis", "key", cacheKey, "error", err)
				}
			}
		}
	}

	return names
}

// fetchAssetNamesFromESI resolves custom names via ESI /assets/names/ (chunk-and-merge).
// Returns (names, ok): ok is true only when every batch fetch succeeded (no ESI/build/parse
// error). A genuinely-empty success (all ships unnamed) returns (empty map, true).
// On any batch failure the partial names collected so far are returned with ok=false so
// the caller can decide not to cache the result.
func (s *FittingService) fetchAssetNamesFromESI(ctx context.Context, characterID int, itemIDs []int64, accessToken string) (map[int64]string, bool) {
	out := make(map[int64]string, len(itemIDs))
	allOK := true
	for _, batch := range chunkInt64(itemIDs, assetNamesMaxIDs) {
		names, batchOK := s.fetchAssetNamesBatch(ctx, characterID, batch, accessToken)
		if !batchOK {
			allOK = false
		}
		for id, name := range names {
			out[id] = name
		}
	}
	return out, allOK
}

// fetchAssetNamesBatch POSTs a single batch of ≤1000 ids to ESI /assets/names/.
// Returns (names, ok): ok is false whenever a request-build, network, HTTP-status, or
// parse error occurs so the caller can propagate a "partial result" signal without
// caching the incomplete response.
func (s *FittingService) fetchAssetNamesBatch(ctx context.Context, characterID int, itemIDs []int64, accessToken string) (map[int64]string, bool) {
	bodyBytes, _ := json.Marshal(itemIDs)
	endpoint := fmt.Sprintf("/latest/characters/%d/assets/names/", characterID)
	req, err := http.NewRequestWithContext(ctx, "POST", esiconfig.BaseURL+endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("asset names: build request failed", "error", err)
		}
		return map[int64]string{}, false
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.esiClient.Do(req)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("asset names: esi request failed", "error", err)
		}
		return map[int64]string{}, false
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		if s.logger != nil {
			s.logger.Warn("asset names: non-200", "status", resp.StatusCode)
		}
		return map[int64]string{}, false
	}
	raw, _ := io.ReadAll(resp.Body)
	names, err := parseAssetNames(raw)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("asset names: parse failed", "error", err)
		}
		return map[int64]string{}, false
	}
	return names, true
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
