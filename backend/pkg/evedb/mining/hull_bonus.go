package mining

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// recognisedHullYieldEffects maps a ship-bonus effect (by dogmaEffects.name)
// that modifies ore-mining yield (attr 77) to the skill whose level scales it.
// The modifier's own skillTypeID is a constant (3386) in the SDE and must be ignored.
// A hull whose attr-77 ship bonus is NOT in this set is reported unresolved, so the
// caller marks the row as an estimate instead of computing a partial (wrong) value.
var recognisedHullYieldEffects = map[string]int64{
	"miningBargeBonusOreMiningYield":   17940, // Mining Barge
	"exhumersBonusOreMiningYield":      22551, // Exhumers
	"miningFrigateBonusOreMiningYield": 32918, // Mining Frigate
}

const attrMiningAmountModified = 77

// HullMiningYieldMultiplier returns the multiplicative ore-mining-yield bonus the
// hull grants (role + per-skill-level), read from SDE dogma. resolved=false when the
// hull has an attr-77 ship modifier whose effect is not recognised — the caller must
// then mark the row as an estimate, never silently use 1.0. A hull with no attr-77
// ship bonus returns (1.0, true): genuinely no bonus, not a fallback.
func HullMiningYieldMultiplier(db *sql.DB, hullTypeID int64, skillLevels map[int64]int) (mult float64, resolved bool, err error) {
	hullAttrs, err := typeAttrs(db, hullTypeID)
	if err != nil {
		return 0, false, err
	}
	effectIDs, err := typeEffectIDs(db, hullTypeID)
	if err != nil {
		return 0, false, err
	}

	mult = 1.0
	resolved = true
	for _, eid := range effectIDs {
		effectName, mods, err := effectYieldModifiers(db, eid)
		if err != nil {
			return 0, false, err
		}
		if len(mods) == 0 {
			continue // this effect doesn't touch attr-77 on the ship
		}
		skillID, ok := recognisedHullYieldEffects[effectName]
		if !ok {
			resolved = false // unrecognised mining bonus on this hull
			continue
		}
		for _, m := range mods {
			value := hullAttrs[m.ModifyingAttributeID] // per-level % on this hull
			level := skillLevels[skillID]
			mult *= 1.0 + (value/100.0)*float64(level)
		}
	}
	return mult, resolved, nil
}

// typeEffectIDs returns the dogmaEffects ids referenced by a type's typeDogma.
func typeEffectIDs(db *sql.DB, typeID int64) ([]int64, error) {
	var effectsJSON sql.NullString
	err := db.QueryRow(`SELECT dogmaEffects FROM typeDogma WHERE _key = ?`, typeID).Scan(&effectsJSON)
	if err == sql.ErrNoRows || !effectsJSON.Valid || effectsJSON.String == "" {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("type effects %d: %w", typeID, err)
	}
	var refs []struct {
		EffectID int64 `json:"effectID"`
	}
	if err := json.Unmarshal([]byte(effectsJSON.String), &refs); err != nil {
		return nil, fmt.Errorf("parse effects %d: %w", typeID, err)
	}
	out := make([]int64, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.EffectID)
	}
	return out, nil
}

type yieldModifier struct {
	ModifyingAttributeID int64 `json:"modifyingAttributeID"`
}

// effectYieldModifiers returns an effect's internal name and the subset of its
// modifierInfo that modifies ore-mining yield (attr 77) on the ship. The internal
// name lives in dogmaEffects.name (e.g. "miningBargeBonusOreMiningYield");
// displayName is the localized UI label and is empty for these hidden bonus effects.
func effectYieldModifiers(db *sql.DB, effectID int64) (effectName string, mods []yieldModifier, err error) {
	var name sql.NullString
	var modifierJSON sql.NullString
	err = db.QueryRow(
		`SELECT name, modifierInfo FROM dogmaEffects WHERE _key = ?`, effectID,
	).Scan(&name, &modifierJSON)
	if err == sql.ErrNoRows {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("effect %d: %w", effectID, err)
	}
	if !modifierJSON.Valid || modifierJSON.String == "" {
		return name.String, nil, nil
	}
	var all []struct {
		Domain               string `json:"domain"`
		ModifiedAttributeID  int64  `json:"modifiedAttributeID"`
		ModifyingAttributeID int64  `json:"modifyingAttributeID"`
	}
	if err := json.Unmarshal([]byte(modifierJSON.String), &all); err != nil {
		return "", nil, fmt.Errorf("parse modifierInfo %d: %w", effectID, err)
	}
	for _, m := range all {
		if m.ModifiedAttributeID == attrMiningAmountModified && m.Domain == "shipID" {
			mods = append(mods, yieldModifier{ModifyingAttributeID: m.ModifyingAttributeID})
		}
	}
	return name.String, mods, nil
}
