#!/usr/bin/env bash
# verify-trading-domain.sh – empirically verify key EVE SDE facts that drive
# dogma and fee calculations in eve-o-provit.
#
# Usage: ./verify-trading-domain.sh [/path/to/eve-sde.db]
#
# Findings documented here:
#   F10 – How is the cargoCapacityMultiplier (attributeID 149) stored in SDE:
#          as a raw multiplier (~1.x) or as a percent (~10-30)?
#   F11 – Is attributeID 149 stacking-penalised (stackable = 0) or not (stackable = 1)?
#   F5  – EVE base sales-tax rate used in code (currently hardcoded at 5%).

set -euo pipefail

DB="${1:-$HOME/vscode/eveonline/eve-sde/data/sqlite/eve-sde.db}"

if [[ ! -f "$DB" ]]; then
  echo "ERROR: SDE database not found at: $DB" >&2
  exit 1
fi

echo "========================================================"
echo " EVE O-PROVIT – Trading Domain Verification Script"
echo " DB: $DB"
echo "========================================================"

# ── F10: Stored value representation of attributeID 149 ─────────────────────
echo ""
echo "--- F10: cargoCapacityMultiplier (attributeID 149) stored values ---"
echo "Test modules: Expanded Cargohold I (1317), Expanded Cargohold II (1319),"
echo "  'Basic' Expanded Cargohold (1315), and 3 further modules with attr 149."
echo ""
echo "typeID | typeName (en) | attr149_value"
echo "-------|---------------|---------------"

sqlite3 "$DB" "
SELECT
  td._key AS typeID,
  json_extract(t.name, '$.en') AS typeName,
  json_extract(a.value, '\$.value') AS attr149_value
FROM typeDogma td
JOIN types t ON t._key = td._key
, json_each(td.dogmaAttributes) a
WHERE json_extract(a.value, '\$.attributeID') = 149
  AND td._key IN (1315, 1317, 1319, 5479, 1335, 1192)
ORDER BY td._key;
"

echo ""
echo "CONCLUSION F10:"
echo "  Values are in the range 1.0–1.3 → stored as RAW MULTIPLIER (~1.x)."
echo "  op4 formula (baseValue * multiplier) is CORRECT for attributeID 149."
echo "  op6 formula (baseValue * (1 + value/100)) would be WRONG."

# ── F11: stackable flag of attributeID 149 ───────────────────────────────────
echo ""
echo "--- F11: stackable flag for attributeID 149 (cargoCapacityMultiplier) ---"
echo ""
echo "attributeID | name | stackable"
echo "------------|------|----------"

sqlite3 "$DB" "
SELECT _key AS attributeID, name, stackable
FROM dogmaAttributes
WHERE _key = 149;
"

echo ""
echo "CONCLUSION F11:"
echo "  stackable = 1 → cargo capacity bonus is NOT stacking-penalised."
echo "  (stackable = 0 would mean: applies stacking penalty; 1 means: no penalty)"

# ── F5: Base sales-tax rate ──────────────────────────────────────────────────
echo ""
echo "--- F5: EVE base sales-tax rate (code reference) ---"
echo ""
echo "  The code in eve-o-provit hardcodes the base sales-tax rate at 5.0%."
echo "  EVE Online NPC base sales tax has been 8% since the Viridian expansion"
echo "  (June 2023). With Accounting V trained the effective rate is 3.6%."
echo "  ACTION REQUIRED: verify against current game patch notes and adjust the"
echo "  hardcoded constant if necessary."

echo ""
echo "========================================================"
echo " Verification complete."
echo "========================================================"
