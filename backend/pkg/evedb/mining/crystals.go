package mining

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// miningCrystalGroupID is the SDE charge group for ore Mining Crystals.
const miningCrystalGroupID = 482

// chargeGroupAttrIDs are the dogma attributes that declare which charge groups a
// module accepts (chargeGroup1..4). A mining module is crystal-capable iff one of
// these equals miningCrystalGroupID.
var chargeGroupAttrIDs = []int64{604, 609, 610, 611}

// CrystalCapable reports whether any of the given module type ids can load ore
// Mining Crystals (i.e. a Modulated Strip Miner / Deep Core laser). Strip Miner I
// and basic Miners cannot — for those crystals genuinely do not apply.
func CrystalCapable(db *sql.DB, moduleTypeIDs []int64) (bool, error) {
	for _, tid := range moduleTypeIDs {
		attrs, err := typeAttrs(db, tid)
		if err != nil {
			return false, err
		}
		for _, aid := range chargeGroupAttrIDs {
			if int64(attrs[aid]) == miningCrystalGroupID {
				return true, nil
			}
		}
	}
	return false, nil
}

// OreCrystalMultiplierT2 returns the best-case (T2) ore crystal yield multiplier
// for the ore group, read live from the SDE. The crystal is found by name
// ("<oreGroupName> Mining Crystal II"); its yield is attribute 782. found=false
// when the group has no matching crystal (e.g. Mercoxit under this naming) — the
// caller must then mark the row as an estimate, never substitute 1.0.
func OreCrystalMultiplierT2(db *sql.DB, oreGroupID int64) (mult float64, found bool, err error) {
	var groupName string
	err = db.QueryRow(
		`SELECT json_extract(name,'$.en') FROM groups WHERE _key = ?`, oreGroupID,
	).Scan(&groupName)
	if err == sql.ErrNoRows || groupName == "" {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("ore group %d name: %w", oreGroupID, err)
	}

	crystalName := groupName + " Mining Crystal II"
	var crystalTypeID int64
	err = db.QueryRow(
		`SELECT _key FROM types WHERE json_extract(name,'$.en') = ?`, crystalName,
	).Scan(&crystalTypeID)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("crystal %q: %w", crystalName, err)
	}

	attrs, err := typeAttrs(db, crystalTypeID)
	if err != nil {
		return 0, false, err
	}
	const attrAsteroidYieldMultiplier = 782
	v, ok := attrs[attrAsteroidYieldMultiplier]
	if !ok || v <= 0 {
		return 0, false, nil
	}
	return v, true, nil
}

// typeAttrs returns the typeDogma dogmaAttributes of a type as attributeID→value.
// Empty map (not an error) when the type has no dogma row.
func typeAttrs(db *sql.DB, typeID int64) (map[int64]float64, error) {
	var attrsJSON sql.NullString
	err := db.QueryRow(`SELECT dogmaAttributes FROM typeDogma WHERE _key = ?`, typeID).Scan(&attrsJSON)
	if err == sql.ErrNoRows {
		return map[int64]float64{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("type attrs %d: %w", typeID, err)
	}
	if !attrsJSON.Valid || attrsJSON.String == "" {
		return map[int64]float64{}, nil
	}
	var list []struct {
		AttributeID int64   `json:"attributeID"`
		Value       float64 `json:"value"`
	}
	if err := json.Unmarshal([]byte(attrsJSON.String), &list); err != nil {
		return nil, fmt.Errorf("parse attrs %d: %w", typeID, err)
	}
	m := make(map[int64]float64, len(list))
	for _, a := range list {
		m[a.AttributeID] = a.Value
	}
	return m, nil
}
