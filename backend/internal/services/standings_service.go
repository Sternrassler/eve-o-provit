// Package services - Standings + mining/reprocessing skill fetching (ore-ranking, #mining).
package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/Sternrassler/eve-o-provit/backend/internal/services/esiconfig"
)

// Reprocessing / mining skill type ids (SDE).
const (
	skillReprocessing           = 3385 // +3%/level reprocessing yield
	skillReprocessingEfficiency = 3389 // +2%/level reprocessing yield
	skillMining                 = 3386 // +5%/level mining amount
	skillAstrogeology           = 3410 // +5%/level mining amount
)

// MiningReprocessingSkills holds the ore-ranking-relevant skill levels for a character.
// OreProcessing maps an ore GROUP id → that group's "<Group> Processing" skill level
// (e.g. Veldspar Processing). Missing/untrained groups are absent (treated as level 0).
type MiningReprocessingSkills struct {
	Reprocessing           int
	ReprocessingEfficiency int
	Mining                 int
	Astrogeology           int
	Accounting             int           // sales tax (reused via SalesTaxRate)
	OreProcessing          map[int64]int // ore groupID → processing skill level
}

// parseStandings parses an ESI /characters/{id}/standings/ body into a map of
// from_id → standing. Pure helper (unit-tested with a fixture, no HTTP).
func parseStandings(body []byte) (map[int64]float64, error) {
	var rows []struct {
		FromType string  `json:"from_type"`
		FromID   int64   `json:"from_id"`
		Standing float64 `json:"standing"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("decode standings: %w", err)
	}
	out := make(map[int64]float64, len(rows))
	for _, r := range rows {
		out[r.FromID] = r.Standing
	}
	return out, nil
}

// GetCharacterStandings fetches the character's standings (ESI /characters/{id}/standings/)
// and returns a map of from_id → standing. 401/403 (no standings) yields an empty map, not
// an error, so a character without standings still ranks ores (at base reprocessing tax).
func (s *SkillsService) GetCharacterStandings(ctx context.Context, characterID int, accessToken string) (map[int64]float64, error) {
	endpoint := fmt.Sprintf("/v2/characters/%d/standings/", characterID)
	req, err := http.NewRequestWithContext(ctx, "GET", esiconfig.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create standings request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.esiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("standings request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return map[int64]float64{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESI standings returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read standings body: %w", err)
	}
	return parseStandings(body)
}

// GetMiningReprocessingSkills fetches the character's mining/reprocessing skill levels,
// including the per-ore-group "<Group> Processing" levels (mapped via SDE name lookup).
// On an ESI failure it returns all-zero skills + the error so the caller can fail loud or
// degrade. The ore-group processing levels are populated only for groups whose processing
// skill the character has trained (others default to 0).
func (s *SkillsService) GetMiningReprocessingSkills(ctx context.Context, sdeDB *sql.DB, characterID int, accessToken string) (*MiningReprocessingSkills, error) {
	esiSkills, err := s.fetchSkillsFromESI(ctx, characterID, accessToken)
	if err != nil {
		return &MiningReprocessingSkills{OreProcessing: map[int64]int{}}, fmt.Errorf("mining skills unavailable: %w", err)
	}

	out := &MiningReprocessingSkills{OreProcessing: map[int64]int{}}
	// skillType id → level, for resolving ore-group processing skills below.
	levelByType := make(map[int64]int, len(esiSkills.Skills))
	for _, sk := range esiSkills.Skills {
		levelByType[int64(sk.SkillID)] = sk.ActiveSkillLevel
		switch sk.SkillID {
		case skillReprocessing:
			out.Reprocessing = sk.ActiveSkillLevel
		case skillReprocessingEfficiency:
			out.ReprocessingEfficiency = sk.ActiveSkillLevel
		case skillMining:
			out.Mining = sk.ActiveSkillLevel
		case skillAstrogeology:
			out.Astrogeology = sk.ActiveSkillLevel
		case 16622: // Accounting (sales tax)
			out.Accounting = sk.ActiveSkillLevel
		}
	}

	// Resolve per-ore-group processing skill levels: groupID → "<Group> Processing"
	// skill type id (SDE, cached) → level from the character's skills.
	for groupID, skillTypeID := range s.oreProcessingSkillTypes(ctx, sdeDB) {
		if lvl, ok := levelByType[skillTypeID]; ok {
			out.OreProcessing[groupID] = lvl
		}
	}

	return out, nil
}

// oreProcessingSkillCache caches the groupID → "<Group> Processing" skill type id map.
var (
	oreProcessingSkillCache   map[int64]int64
	oreProcessingSkillCacheMu sync.Mutex
)

// oreProcessingSkillTypes returns groupID → "<Group> Processing" skill type id for all
// asteroid ore groups, resolved by SDE name lookup and cached process-wide. A group whose
// processing skill cannot be resolved is omitted (its level stays 0). On any SDE error it
// returns whatever was resolved (best-effort: ore-specific bonus simply not applied).
func (s *SkillsService) oreProcessingSkillTypes(ctx context.Context, sdeDB *sql.DB) map[int64]int64 {
	oreProcessingSkillCacheMu.Lock()
	defer oreProcessingSkillCacheMu.Unlock()
	if oreProcessingSkillCache != nil {
		return oreProcessingSkillCache
	}

	out := map[int64]int64{}
	if sdeDB == nil {
		return out
	}

	// For each asteroid ore group, the matching skill is "<GroupName> Processing"
	// (e.g. Veldspar Processing). A few groups use "<GroupName> Ore Processing"
	// (e.g. Mercoxit Ore Processing) — match both forms.
	rows, err := sdeDB.QueryContext(ctx, `
		SELECT g._key, st._key
		FROM groups g
		JOIN categories c ON g.categoryID = c._key
		JOIN types st ON json_extract(st.name,'$.en') IN (
			json_extract(g.name,'$.en') || ' Processing',
			json_extract(g.name,'$.en') || ' Ore Processing'
		)
		WHERE json_extract(c.name,'$.en') = 'Asteroid'`)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("ore processing skill lookup failed", "error", err)
		}
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var groupID, skillTypeID int64
		if err := rows.Scan(&groupID, &skillTypeID); err != nil {
			continue
		}
		out[groupID] = skillTypeID
	}
	oreProcessingSkillCache = out
	return out
}
