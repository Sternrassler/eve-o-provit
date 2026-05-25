// Package navigation provides EVE Online navigation and route planning functionalitypackage navigation

// Migrated from eve-sde project: github.com/Sternrassler/eve-sde/pkg/evedb/navigation
package navigation

import (
	"database/sql"
	"fmt"
	"math"
	"sync"
)

var (
	graphCacheMu    sync.RWMutex
	graphCache      = map[bool]map[int64][]edge{} // key: avoidLowSec
	graphLoadsTotal int
)

// resetGraphCache leert den Graph-Cache (für Tests).
func resetGraphCache() {
	graphCacheMu.Lock()
	defer graphCacheMu.Unlock()
	graphCache = map[bool]map[int64][]edge{}
}

// graphLoadCount gibt zurück, wie oft der Graph aus der DB geladen wurde (für Tests).
func graphLoadCount() int {
	graphCacheMu.RLock()
	defer graphCacheMu.RUnlock()
	return graphLoadsTotal
}

// getCachedGraph lädt den Graphen einmalig pro avoidLowSec-Variante und cached ihn.
func getCachedGraph(db *sql.DB, avoidLowSec bool) (map[int64][]edge, error) {
	graphCacheMu.RLock()
	if g, ok := graphCache[avoidLowSec]; ok {
		graphCacheMu.RUnlock()
		return g, nil
	}
	graphCacheMu.RUnlock()

	graphCacheMu.Lock()
	defer graphCacheMu.Unlock()
	if g, ok := graphCache[avoidLowSec]; ok { // double-check
		return g, nil
	}
	g, err := loadGraph(db, avoidLowSec)
	if err != nil {
		return nil, err
	}
	graphCache[avoidLowSec] = g
	graphLoadsTotal++
	return g, nil
}

// bfs findet den kürzesten Pfad (in Jumps) per Breitensuche.
// Alle Stargate-Kanten haben Gewicht 1, daher liefert BFS das Optimum.
func bfs(graph map[int64][]edge, start, goal int64) ([]int64, bool) {
	if _, exists := graph[start]; !exists {
		return nil, false
	}
	if start == goal {
		return []int64{start}, true
	}
	visited := map[int64]bool{start: true}
	prev := map[int64]int64{}
	queue := []int64{start}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, e := range graph[current] {
			if visited[e.toSystemID] {
				continue
			}
			visited[e.toSystemID] = true
			prev[e.toSystemID] = current
			if e.toSystemID == goal {
				return reconstructPath(prev, start, goal), true
			}
			queue = append(queue, e.toSystemID)
		}
	}
	return nil, false
}

// NavigationParams contains optional parameters for route calculation
type NavigationParams struct {
	WarpSpeed       *float64 `json:"warp_speed,omitempty"`        // AU/s (default: 3.0)
	AlignTime       *float64 `json:"align_time,omitempty"`        // seconds (default: 6.0)
	ShipMass        *float64 `json:"ship_mass,omitempty"`         // kg (for exact align calc)
	InertiaModifier *float64 `json:"inertia_modifier,omitempty"`  // ship agility
	AvgWarpDistance *float64 `json:"avg_warp_distance,omitempty"` // AU per system (default: 15)
	AvoidLowSec     bool     `json:"avoid_lowsec"`                // route via high-sec only
}

// RouteResult contains calculated route information
type RouteResult struct {
	TotalSeconds      float64                `json:"total_seconds"`
	TotalMinutes      float64                `json:"total_minutes"`
	Jumps             int                    `json:"jumps"`
	AvgSecondsPerJump float64                `json:"avg_seconds_per_jump"`
	Route             []int64                `json:"route"`
	ParametersUsed    map[string]interface{} `json:"parameters_used"`
}

// PathResult contains just the path information
type PathResult struct {
	FromSystemID int64   `json:"from_system_id"`
	ToSystemID   int64   `json:"to_system_id"`
	Jumps        int     `json:"jumps"`
	Route        []int64 `json:"route"`
}

// Default navigation parameters
const (
	DefaultWarpSpeed       = 3.0  // AU/s (Cruiser average)
	DefaultAlignTime       = 6.0  // seconds (medium ships)
	DefaultGateJumpDelay   = 10.0 // seconds (gate jump animation + loading)
	DefaultAvgWarpDistance = 15.0 // AU (statistical assumption)
	WarpCorrectionFactor   = 1.4  // Simplified warp time correction
)

// edge represents a stargate connection
type edge struct {
	toSystemID int64
}

// CalculateWarpTime calculates warp time using CCP's 3-phase formula
// Reference: https://wiki.eveuniversity.org/Warp_time_calculation
func CalculateWarpTime(distanceAU, warpSpeedAU float64) float64 {
	const AU = 149597870700.0 // meters in 1 AU

	k := warpSpeedAU
	j := math.Min(k/3.0, 2.0)

	// Phase 1: Acceleration (always 1 AU)
	tAccel := 25.7312 / k

	// Phase 2: Deceleration
	dDecel := (k * AU) / j
	tDecel := (math.Log(k*AU) - math.Log(100)) / j // 100 m/s dropout speed

	// Phase 3: Cruise
	totalDistanceMeters := distanceAU * AU
	dCruise := math.Max(0, totalDistanceMeters-AU-dDecel)
	tCruise := dCruise / (k * AU)

	return tAccel + tCruise + tDecel
}

// CalculateSimplifiedWarpTime uses simplified formula for quick estimation
func CalculateSimplifiedWarpTime(distanceAU, warpSpeedAU float64) float64 {
	return (distanceAU / warpSpeedAU) * WarpCorrectionFactor
}

// getEffectiveParams returns effective parameters with defaults applied
func getEffectiveParams(params *NavigationParams) (warpSpeed, alignTime, avgWarpDist float64, source string) {
	source = "default"
	warpSpeed = DefaultWarpSpeed
	alignTime = DefaultAlignTime
	avgWarpDist = DefaultAvgWarpDistance

	if params == nil {
		return
	}

	if params.WarpSpeed != nil {
		warpSpeed = *params.WarpSpeed
		source = "provided"
	}

	if params.AlignTime != nil {
		alignTime = *params.AlignTime
		source = "provided"
	} else if params.ShipMass != nil && params.InertiaModifier != nil {
		// Use canonical align time calculation
		alignTime = CalculateAlignTime(*params.InertiaModifier, *params.ShipMass)
		source = "calculated"
	}

	if params.AvgWarpDistance != nil {
		avgWarpDist = *params.AvgWarpDistance
	}

	return
}

// ShortestPath finds the shortest path between two systems using Dijkstra's algorithm
func ShortestPath(db *sql.DB, fromSystemID, toSystemID int64, avoidLowSec bool) (*PathResult, error) {
	// Load the graph from cache (or DB on first call)
	graph, err := getCachedGraph(db, avoidLowSec)
	if err != nil {
		return nil, fmt.Errorf("failed to load graph: %w", err)
	}

	// Run BFS (optimal for unit-weight graphs)
	path, found := bfs(graph, fromSystemID, toSystemID)
	if !found {
		return nil, fmt.Errorf("no path found between systems %d and %d", fromSystemID, toSystemID)
	}

	result := &PathResult{
		FromSystemID: fromSystemID,
		ToSystemID:   toSystemID,
		Jumps:        len(path) - 1, // jumps = number of systems - 1
		Route:        path,
	}

	return result, nil
}

// loadGraph loads the stargate graph from the database
func loadGraph(db *sql.DB, avoidLowSec bool) (map[int64][]edge, error) {
	var query string
	if avoidLowSec {
		query = `
			SELECT DISTINCT g.from_system_id, g.to_system_id
			FROM v_stargate_graph g
			JOIN mapSolarSystems sf ON g.from_system_id = sf._key
			JOIN mapSolarSystems st ON g.to_system_id = st._key
			WHERE sf.securityStatus >= 0.45 AND st.securityStatus >= 0.45
		`
	} else {
		query = `
			SELECT from_system_id, to_system_id
			FROM v_stargate_graph
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query graph: %w", err)
	}
	defer rows.Close()

	graph := make(map[int64][]edge)
	for rows.Next() {
		var from, to int64
		if err := rows.Scan(&from, &to); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}
		graph[from] = append(graph[from], edge{toSystemID: to})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating graph: %w", err)
	}

	return graph, nil
}

// reconstructPath builds the path from start to goal using the prev map
func reconstructPath(prev map[int64]int64, start, goal int64) []int64 {
	path := []int64{goal}
	current := goal

	for current != start {
		current = prev[current]
		path = append([]int64{current}, path...)
	}

	return path
}

// CalculateTravelTime calculates total travel time for a route with optional ship parameters
// Set useExactFormula=true to use the exact 3-phase CCP warp formula, false for simplified linear approximation
func CalculateTravelTime(db *sql.DB, fromSystemID, toSystemID int64, params *NavigationParams, useExactFormula bool) (*RouteResult, error) {
	// Get effective parameters
	warpSpeed, alignTime, avgWarpDist, source := getEffectiveParams(params)

	// Determine if we should avoid low-sec
	avoidLowSec := false
	if params != nil {
		avoidLowSec = params.AvoidLowSec
	}

	// Find the shortest path
	path, err := ShortestPath(db, fromSystemID, toSystemID, avoidLowSec)
	if err != nil {
		return nil, err
	}

	// Calculate time per jump using selected formula
	var warpTime float64
	var formulaUsed string
	if useExactFormula {
		warpTime = CalculateWarpTime(avgWarpDist, warpSpeed)
		formulaUsed = "exact_3phase"
	} else {
		warpTime = CalculateSimplifiedWarpTime(avgWarpDist, warpSpeed)
		formulaUsed = "simplified_linear"
	}
	timePerJump := alignTime + warpTime + DefaultGateJumpDelay

	// Calculate total time
	totalSeconds := float64(path.Jumps) * timePerJump

	result := &RouteResult{
		TotalSeconds:      totalSeconds,
		TotalMinutes:      totalSeconds / 60.0,
		Jumps:             path.Jumps,
		AvgSecondsPerJump: timePerJump,
		Route:             path.Route,
		ParametersUsed: map[string]interface{}{
			"warp_speed": warpSpeed,
			"align_time": alignTime,
			"source":     source,
			"formula":    formulaUsed,
		},
	}

	return result, nil
}
