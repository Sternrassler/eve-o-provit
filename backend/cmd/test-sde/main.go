package main

import (
	"fmt"
	"log"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb"
)

func main() {
	// Open SDE database
	db, err := evedb.Open("../data/sde/eve-sde.db")
	if err != nil {
		log.Fatalf("Failed to open SDE database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Successfully connected to SDE database")
	fmt.Printf("   Database path: %s\n", db.Path())

	// Test ping
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database ping successful")

	// Query a simple stat
	var tableCount int
	query := "SELECT COUNT(*) FROM sqlite_master WHERE type='table'"
	err = db.Conn().QueryRow(query).Scan(&tableCount)
	if err != nil {
		log.Fatalf("Failed to query tables: %v", err)
	}

	fmt.Printf("✅ Database contains %d tables\n", tableCount)

	// Query view count
	var viewCount int
	query = "SELECT COUNT(*) FROM sqlite_master WHERE type='view'"
	err = db.Conn().QueryRow(query).Scan(&viewCount)
	if err != nil {
		log.Fatalf("Failed to query views: %v", err)
	}

	fmt.Printf("✅ Database contains %d views\n", viewCount)
}
