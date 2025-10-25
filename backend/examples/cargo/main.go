// Example: EVE Cargo Calculatorpackage cargo

// This example demonstrates cargo capacity calculations with and without skills
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Command line flags
	var (
		dbPath            = flag.String("db", "../../data/sqlite/eve-sde.db", "Path to SQLite database")
		shipTypeID        = flag.Int64("ship", 648, "Ship type ID (default: Badger)")
		itemTypeID        = flag.Int64("item", 34, "Item type ID (default: Tritanium)")
		racialHaulerLevel = flag.Int("racial-hauler", -1, "Racial Hauler skill level (0-5, -1 for none)")
		freighterLevel    = flag.Int("freighter", -1, "Freighter skill level (0-5, -1 for none)")
		cargoMultiplier   = flag.Float64("cargo-mult", -1, "Custom cargo multiplier (e.g. 1.5 for +50%)")
		showShipInfo      = flag.Bool("ship-info", false, "Show detailed ship capacity information")
		initViews         = flag.Bool("init-views", false, "Initialize cargo views and exit")
	)
	flag.Parse()

	// Open database
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Check if database exists
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		log.Fatalf("Database not found: %s\nPlease ensure the eve-sde database is available.", *dbPath)
	}

	// Initialize views if requested
	if *initViews {
		log.Println("Initializing cargo views...")
		if err := initializeCargoViews(db); err != nil {
			log.Fatalf("Failed to initialize views: %v", err)
		}
		log.Println("✓ Cargo views initialized successfully")
		return
	}

	// Check if views exist
	var viewExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='view' AND name='v_item_volumes'").Scan(&viewExists)
	if err != nil || viewExists == 0 {
		log.Println("⚠ Cargo views not found. Initializing...")
		if err := initializeCargoViews(db); err != nil {
			log.Fatalf("Failed to initialize views: %v", err)
		}
		log.Println("✓ Cargo views initialized")
	}

	// Build skill modifiers from flags
	var skills *cargo.SkillModifiers
	if *racialHaulerLevel >= 0 || *freighterLevel >= 0 || *cargoMultiplier > 0 {
		skills = &cargo.SkillModifiers{}

		if *racialHaulerLevel >= 0 && *racialHaulerLevel <= 5 {
			skills.RacialHaulerLevel = racialHaulerLevel
		}
		if *freighterLevel >= 0 && *freighterLevel <= 5 {
			skills.FreighterLevel = freighterLevel
		}
		if *cargoMultiplier > 0 {
			skills.CargoMultiplier = cargoMultiplier
		}
	}

	// Show ship info if requested
	if *showShipInfo {
		showShipCapacities(db, *shipTypeID, skills)
		return
	}

	// Calculate cargo fit
	fmt.Printf("\n=== EVE Cargo Calculator ===\n\n")

	result, err := cargo.CalculateCargoFit(db, *shipTypeID, *itemTypeID, skills)
	if err != nil {
		log.Fatalf("Failed to calculate cargo fit: %v", err)
	}

	// Display results
	fmt.Printf("Ship: %s (Type ID: %d)\n", result.ShipName, result.ShipTypeID)
	fmt.Printf("Base Cargo Capacity: %s m³\n", formatNumber(result.BaseCapacity))

	if result.SkillsApplied {
		fmt.Printf("Skill Bonus: +%.1f%%\n", result.SkillBonus)
		if skills.RacialHaulerLevel != nil {
			fmt.Printf("  - Racial Hauler: Level %d\n", *skills.RacialHaulerLevel)
		}
		if skills.FreighterLevel != nil {
			fmt.Printf("  - Freighter: Level %d\n", *skills.FreighterLevel)
		}
		if skills.CargoMultiplier != nil {
			fmt.Printf("  - Custom Multiplier: %.2fx\n", *skills.CargoMultiplier)
		}
		fmt.Printf("Effective Capacity: %s m³\n", formatNumber(result.EffectiveCapacity))
	} else {
		fmt.Printf("Effective Capacity: %s m³ (no skills applied)\n", formatNumber(result.EffectiveCapacity))
	}

	fmt.Printf("\nItem: %s (Type ID: %d)\n", result.ItemName, result.ItemTypeID)
	fmt.Printf("Volume per unit: %.4f m³\n", result.ItemVolume)

	fmt.Printf("\n=== Cargo Fit Results ===\n")
	fmt.Printf("Max Quantity: %s units\n", formatNumber(float64(result.MaxQuantity)))
	fmt.Printf("Total Volume: %s m³\n", formatNumber(result.TotalVolume))
	fmt.Printf("Remaining Space: %s m³\n", formatNumber(result.RemainingSpace))
	fmt.Printf("Utilization: %.2f%%\n", result.UtilizationPct)

	// Show example comparison
	if !result.SkillsApplied {
		fmt.Printf("\n💡 Tip: Use --racial-hauler 5 to see the effect of skills!\n")
		fmt.Printf("   Example: go run examples/cargo/main.go --ship %d --item %d --racial-hauler 5\n",
			*shipTypeID, *itemTypeID)
	}
}

// showShipCapacities displays detailed ship capacity information
func showShipCapacities(db *sql.DB, shipTypeID int64, skills *cargo.SkillModifiers) {
	ship, err := cargo.GetShipCapacities(db, shipTypeID, skills)
	if err != nil {
		log.Fatalf("Failed to get ship capacities: %v", err)
	}

	fmt.Printf("\n=== Ship Capacity Details ===\n\n")
	fmt.Printf("Ship: %s (Type ID: %d)\n\n", ship.ShipName, ship.ShipTypeID)

	fmt.Printf("Base Capacities:\n")
	fmt.Printf("  Cargo Hold: %s m³\n", formatNumber(ship.BaseCargoHold))
	fmt.Printf("  Total:      %s m³\n", formatNumber(ship.BaseTotalCapacity))

	if ship.SkillsApplied {
		fmt.Printf("\nEffective Capacities (with skills):\n")
		fmt.Printf("  Cargo Hold: %s m³ (+%.1f%%)\n",
			formatNumber(ship.EffectiveCargoHold),
			((ship.EffectiveCargoHold/ship.BaseCargoHold)-1)*100)
		fmt.Printf("  Total:      %s m³ (+%.1f%%)\n",
			formatNumber(ship.EffectiveTotalCapacity),
			ship.SkillBonus)
	}
}

// initializeCargoViews creates the required SQL views for cargo calculations
func initializeCargoViews(db *sql.DB) error {
	views := []string{
		// v_item_volumes: Item volume and price information
		`CREATE VIEW IF NOT EXISTS v_item_volumes AS
		SELECT
			t._key AS type_id,
			json_extract(t.name, '$.en') AS type_name,
			CAST(t.volume AS REAL) AS volume,
			COALESCE(CAST(t.basePrice AS REAL), 0) AS base_price,
			CASE
				WHEN CAST(t.volume AS REAL) > 0
				THEN COALESCE(CAST(t.basePrice AS REAL), 0) / CAST(t.volume AS REAL)
				ELSE 0
			END AS isk_per_m3
		FROM invTypes t
		WHERE t.published = 1
		  AND CAST(t.volume AS REAL) > 0`,

		// v_ship_cargo_capacities: Ship cargo hold capacities
		`CREATE VIEW IF NOT EXISTS v_ship_cargo_capacities AS
		SELECT
			t._key AS type_id,
			json_extract(t.name, '$.en') AS type_name,
			COALESCE(CAST(t.capacity AS REAL), 0) AS cargo_capacity
		FROM invTypes t
		WHERE t.published = 1
		  AND json_extract(t.name, '$.en') IS NOT NULL
		  AND (
			  t.groupID IN (
				  SELECT _key FROM invGroups
				  WHERE categoryID = 6  -- Ships category
			  )
			  OR t._key IN (SELECT typeID FROM invMetaTypes WHERE metaGroupID = 2)  -- Tech II
		  )`,
	}

	for _, view := range views {
		if _, err := db.Exec(view); err != nil {
			return fmt.Errorf("failed to create view: %w", err)
		}
	}
	return nil
}

// formatNumber formats a number with thousand separators
func formatNumber(n float64) string {
	if n == 0 {
		return "0"
	}

	// Handle integers
	if n == float64(int64(n)) {
		s := fmt.Sprintf("%d", int64(n))
		return addCommas(s)
	}

	// Handle floats
	s := fmt.Sprintf("%.2f", n)
	parts := []rune(s)

	// Find decimal point
	dotIdx := -1
	for i, r := range parts {
		if r == '.' {
			dotIdx = i
			break
		}
	}

	if dotIdx > 0 {
		intPart := string(parts[:dotIdx])
		decPart := string(parts[dotIdx:])
		return addCommas(intPart) + decPart
	}

	return addCommas(s)
}

// addCommas adds thousand separators to a string
func addCommas(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}

	result := ""
	for i, r := range s {
		if i > 0 && (n-i)%3 == 0 {
			result += ","
		}
		result += string(r)
	}
	return result
}
