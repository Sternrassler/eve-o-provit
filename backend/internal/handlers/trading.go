// Package handlers - Trading endpoints
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/navigation"
	"github.com/gofiber/fiber/v2"
)

// TradingHandler handles trading-related HTTP requests
type TradingHandler struct {
	calculator      *services.RouteCalculator
	handler         *Handler
	characterHelper *services.CharacterHelper
}

// NewTradingHandler creates a new trading handler instance
func NewTradingHandler(calculator *services.RouteCalculator, baseHandler *Handler, charHelper *services.CharacterHelper) *TradingHandler {
	return &TradingHandler{
		calculator:      calculator,
		handler:         baseHandler,
		characterHelper: charHelper,
	}
}

// CalculateRoutes handles POST /api/v1/trading/routes/calculate
func (h *TradingHandler) CalculateRoutes(c *fiber.Ctx) error {
	var req models.RouteCalculationRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.RegionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid region_id",
		})
	}
	if req.ShipTypeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ship_type_id",
		})
	}

	// Calculate routes
	result, err := h.calculator.Calculate(c.Context(), req.RegionID, req.ShipTypeID, req.CargoCapacity)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to calculate routes",
			"details": err.Error(),
		})
	}

	// Check if we have a timeout warning (partial results)
	if result.Warning != "" {
		c.Set("Warning", `199 - "`+result.Warning+`"`)
		return c.Status(fiber.StatusPartialContent).JSON(result)
	}

	return c.JSON(result)
}

// GetCharacterLocation handles GET /api/v1/character/location
func (h *TradingHandler) GetCharacterLocation(c *fiber.Ctx) error {
	characterID := c.Locals("character_id").(int)
	accessToken := c.Locals("access_token").(string)

	// Call ESI
	location, err := h.fetchESICharacterLocation(c.Context(), characterID, accessToken)
	if err != nil {
		if err.Error() == "unauthorized" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch character location",
			"details": err.Error(),
		})
	}

	return c.JSON(location)
}

// GetCharacterShip handles GET /api/v1/character/ship
func (h *TradingHandler) GetCharacterShip(c *fiber.Ctx) error {
	characterID := c.Locals("character_id").(int)
	accessToken := c.Locals("access_token").(string)

	// Call ESI
	ship, err := h.fetchESICharacterShip(c.Context(), characterID, accessToken)
	if err != nil {
		if err.Error() == "unauthorized" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch character ship",
			"details": err.Error(),
		})
	}

	return c.JSON(ship)
}

// GetCharacterShips handles GET /api/v1/character/ships
func (h *TradingHandler) GetCharacterShips(c *fiber.Ctx) error {
	characterID := c.Locals("character_id").(int)
	accessToken := c.Locals("access_token").(string)

	// Call ESI
	ships, err := h.fetchESICharacterShips(c.Context(), characterID, accessToken)
	if err != nil {
		if err.Error() == "unauthorized" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch character ships",
			"details": err.Error(),
		})
	}

	return c.JSON(ships)
}

// SetAutopilotWaypoint handles POST /api/v1/esi/ui/autopilot/waypoint
// Sets a waypoint in the EVE client's autopilot via ESI UI API
func (h *TradingHandler) SetAutopilotWaypoint(c *fiber.Ctx) error {
	// Extract auth context
	accessToken := c.Locals("access_token").(string)

	// Parse request body
	var req struct {
		DestinationID  int64 `json:"destination_id"`
		ClearOther     bool  `json:"clear_other_waypoints"`
		AddToBeginning bool  `json:"add_to_beginning"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate destination_id
	if req.DestinationID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid destination_id",
		})
	}

	// Call ESI UI Autopilot Waypoint endpoint
	err := h.setESIAutopilotWaypoint(c.Context(), accessToken, req.DestinationID, req.ClearOther, req.AddToBeginning)
	if err != nil {
		switch err.Error() {
		case "unauthorized":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated or missing scope esi-ui.write_waypoint.v1",
			})
		case "not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "EVE client not running or destination not found",
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "Failed to set waypoint",
				"details": err.Error(),
			})
		}
	}

	// Success (ESI returns 204 No Content)
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ESI helper functions

type esiLocationResponse struct {
	SolarSystemID int64  `json:"solar_system_id"`
	StationID     *int64 `json:"station_id,omitempty"`
	StructureID   *int64 `json:"structure_id,omitempty"`
}

func (h *TradingHandler) fetchESICharacterLocation(ctx context.Context, characterID int, accessToken string) (*models.CharacterLocation, error) {
	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/location/", characterID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("unauthorized")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI returned status %d: %s", resp.StatusCode, string(body))
	}

	var esiLoc esiLocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&esiLoc); err != nil {
		return nil, err
	}

	// Enrich with SDE data
	location := &models.CharacterLocation{
		SolarSystemID: esiLoc.SolarSystemID,
		StationID:     esiLoc.StationID,
		StructureID:   esiLoc.StructureID,
	}

	// Get system and region names from SDE
	systemInfo, err := h.getSystemInfo(ctx, esiLoc.SolarSystemID)
	if err == nil {
		location.SolarSystemName = systemInfo.SystemName
		location.RegionID = systemInfo.RegionID
		location.RegionName = systemInfo.RegionName
	}

	if esiLoc.StationID != nil {
		stationName, err := h.getStationName(ctx, *esiLoc.StationID)
		if err == nil {
			location.StationName = &stationName
		}
	}

	return location, nil
}

type esiShipResponse struct {
	ShipTypeID int64  `json:"ship_type_id"`
	ShipName   string `json:"ship_name"`
	ShipItemID int64  `json:"ship_item_id"`
}

func (h *TradingHandler) fetchESICharacterShip(ctx context.Context, characterID int, accessToken string) (*models.CharacterShip, error) {
	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/ship/", characterID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("unauthorized")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI returned status %d: %s", resp.StatusCode, string(body))
	}

	var esiShip esiShipResponse
	if err := json.NewDecoder(resp.Body).Decode(&esiShip); err != nil {
		return nil, err
	}

	// Enrich with SDE data
	ship := &models.CharacterShip{
		ShipTypeID: esiShip.ShipTypeID,
		ShipName:   esiShip.ShipName,
		ShipItemID: esiShip.ShipItemID,
	}

	// Get ship type name and cargo capacity
	typeInfo, err := h.handler.sdeRepo.GetTypeInfo(ctx, int(esiShip.ShipTypeID))
	if err == nil {
		ship.ShipTypeName = typeInfo.Name
	}

	capacities, err := cargo.GetShipCapacities(h.handler.db.SDE, esiShip.ShipTypeID, nil)
	if err == nil {
		ship.CargoCapacity = capacities.BaseCargoHold
	}

	return ship, nil
}

type esiAssetResponse struct {
	ItemID       int64  `json:"item_id"`
	TypeID       int64  `json:"type_id"`
	LocationID   int64  `json:"location_id"`
	LocationFlag string `json:"location_flag"`
	IsSingleton  bool   `json:"is_singleton"`
}

func (h *TradingHandler) fetchESICharacterShips(ctx context.Context, characterID int, accessToken string) (*models.CharacterShipsResponse, error) {
	url := fmt.Sprintf("https://esi.evetech.net/latest/characters/%d/assets/", characterID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("unauthorized")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ESI returned status %d: %s", resp.StatusCode, string(body))
	}

	var esiAssets []esiAssetResponse
	if err := json.NewDecoder(resp.Body).Decode(&esiAssets); err != nil {
		return nil, err
	}

	// Filter for ships in hangars (categoryID = 6)
	var ships []models.CharacterAssetShip
	for _, asset := range esiAssets {
		if asset.LocationFlag != "Hangar" {
			continue
		}

		// Get type info to check category
		typeInfo, err := h.handler.sdeRepo.GetTypeInfo(ctx, int(asset.TypeID))
		if err != nil {
			continue
		}

		// Check if it's a ship (categoryID = 6)
		if typeInfo.CategoryID == nil || *typeInfo.CategoryID != 6 {
			continue
		}

		// Get cargo capacity
		capacities, err := cargo.GetShipCapacities(h.handler.db.SDE, asset.TypeID, nil)
		if err != nil {
			// Skip if we can't get ship capacities (probably not a ship)
			continue
		}

		locationName, _ := h.getStationName(ctx, asset.LocationID)

		ships = append(ships, models.CharacterAssetShip{
			ItemID:        asset.ItemID,
			TypeID:        asset.TypeID,
			TypeName:      typeInfo.Name,
			LocationID:    asset.LocationID,
			LocationName:  locationName,
			LocationFlag:  asset.LocationFlag,
			CargoCapacity: capacities.BaseCargoHold,
			IsSingleton:   asset.IsSingleton,
		})
	}

	return &models.CharacterShipsResponse{
		Ships: ships,
		Count: len(ships),
	}, nil
}

// SDE helper functions

type systemInfo struct {
	SystemName string
	RegionID   int64
	RegionName string
}

func (h *TradingHandler) getSystemInfo(ctx context.Context, systemID int64) (*systemInfo, error) {
	query := `
		SELECT s.name, s.regionID, r.name
		FROM mapSolarSystems s
		JOIN mapRegions r ON s.regionID = r._key
		WHERE s._key = ?
	`

	var info systemInfo
	var systemNameJSON, regionNameJSON string
	err := h.handler.db.SDE.QueryRowContext(ctx, query, systemID).Scan(
		&systemNameJSON,
		&info.RegionID,
		&regionNameJSON,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON names and extract English version
	var systemNames map[string]string
	if err := json.Unmarshal([]byte(systemNameJSON), &systemNames); err == nil {
		if enName, ok := systemNames["en"]; ok {
			info.SystemName = enName
		}
	}

	var regionNames map[string]string
	if err := json.Unmarshal([]byte(regionNameJSON), &regionNames); err == nil {
		if enName, ok := regionNames["en"]; ok {
			info.RegionName = enName
		}
	}

	return &info, nil
}

func (h *TradingHandler) getStationName(ctx context.Context, stationID int64) (string, error) {
	// Try staStations first (old SDE format)
	query := `SELECT stationName FROM staStations WHERE stationID = ?`

	var name string
	err := h.handler.db.SDE.QueryRowContext(ctx, query, stationID).Scan(&name)
	if err == nil {
		return name, nil
	}

	// Try mapDenormalize as fallback
	query = `SELECT itemName FROM mapDenormalize WHERE itemID = ?`
	err = h.handler.db.SDE.QueryRowContext(ctx, query, stationID).Scan(&name)
	if err == nil {
		return name, nil
	}

	// For NPC stations, fetch name from ESI Universe Names API
	type esiNameResponse struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		Category string `json:"category"`
	}

	url := "https://esi.evetech.net/latest/universe/names/"
	payload := fmt.Sprintf("[%d]", stationID)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(payload))
	if err != nil {
		return strconv.FormatInt(stationID, 10), nil
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return strconv.FormatInt(stationID, 10), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		var esiNames []esiNameResponse
		if err := json.NewDecoder(resp.Body).Decode(&esiNames); err == nil && len(esiNames) > 0 {
			return esiNames[0].Name, nil
		}
	}

	// Final fallback: just return ID as string
	return strconv.FormatInt(stationID, 10), nil
}

// setESIAutopilotWaypoint sets a waypoint in the EVE client via ESI UI API
func (h *TradingHandler) setESIAutopilotWaypoint(ctx context.Context, accessToken string, destinationID int64, clearOther, addToBeginning bool) error {
	url := "https://esi.evetech.net/latest/ui/autopilot/waypoint/"

	// Build query parameters
	params := fmt.Sprintf("?destination_id=%d&clear_other_waypoints=%t&add_to_beginning=%t",
		destinationID, clearOther, addToBeginning)

	req, err := http.NewRequestWithContext(ctx, "POST", url+params, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// ESI returns 204 No Content on success
	if resp.StatusCode == 204 {
		return nil
	}

	// 403: Missing scope esi-ui.write_waypoint.v1
	if resp.StatusCode == 403 || resp.StatusCode == 401 {
		return fmt.Errorf("unauthorized")
	}

	// 404: EVE client not running or destination not found
	if resp.StatusCode == 404 {
		return fmt.Errorf("not_found")
	}

	// Other errors
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("ESI returned status %d: %s", resp.StatusCode, string(body))
}

// SearchItems handles GET /api/v1/items/search
func (h *TradingHandler) SearchItems(c *fiber.Ctx) error {
	query := c.Query("q")
	if len(query) < 3 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "query parameter 'q' must be at least 3 characters",
		})
	}

	// Parse limit (default 20, max 100)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			if parsedLimit > 0 && parsedLimit <= 100 {
				limit = parsedLimit
			}
		}
	}

	// Search items via SDE repository
	items, err := h.handler.sdeRepo.SearchItems(c.Context(), query, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed to search items",
			"details": err.Error(),
		})
	}

	// Convert to response model
	var results []models.ItemSearchResult
	for _, item := range items {
		results = append(results, models.ItemSearchResult{
			TypeID:    item.TypeID,
			Name:      item.Name,
			GroupName: item.GroupName,
		})
	}

	return c.JSON(fiber.Map{
		"items": results,
		"count": len(results),
	})
}

// CalculateInventorySellRoutes handles POST /api/v1/trading/inventory-sell
func (h *TradingHandler) CalculateInventorySellRoutes(c *fiber.Ctx) error {
	characterID := c.Locals("character_id").(int)
	accessToken := c.Locals("access_token").(string)

	var req models.InventorySellRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate request
	if req.TypeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid type_id",
		})
	}
	if req.Quantity <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid quantity",
		})
	}
	if req.BuyPricePerUnit <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid buy_price_per_unit",
		})
	}
	if req.RegionID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid region_id",
		})
	}

	// Get character location (current station)
	location, err := h.characterHelper.GetCharacterLocation(c.Context(), characterID, accessToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch character location",
			"details": err.Error(),
		})
	}

	// Determine starting station (fallback to system if in space)
	var startStationID int64
	if location.StationID != nil {
		startStationID = *location.StationID
	} else {
		// Character in space - must be docked
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Character must be docked at a station to calculate sell routes",
		})
	}

	// Get start system ID
	startSystemID, err := h.handler.sdeRepo.GetSystemIDForLocation(c.Context(), startStationID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to determine starting system",
			"details": err.Error(),
		})
	}

	// Calculate tax rate based on character skills
	taxRate, err := h.characterHelper.CalculateTaxRate(c.Context(), characterID, accessToken)
	if err != nil {
		// Use fallback tax rate
		taxRate = 0.055
	}

	// Fetch all buy orders for the item in the region
	orders, err := h.handler.esiClient.GetMarketOrders(c.Context(), req.RegionID, req.TypeID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to fetch market orders",
			"details": err.Error(),
		})
	}

	log.Printf("[DEBUG] Raw ESI orders count: %d for type_id=%d region_id=%d", len(orders), req.TypeID, req.RegionID)
	if len(orders) > 0 {
		log.Printf("[DEBUG] First order sample: Price=%.2f, IsBuy=%v, Volume=%d", orders[0].Price, orders[0].IsBuyOrder, orders[0].VolumeRemain)
	}

	// Filter for buy orders only
	var buyOrders []struct {
		Price        float64
		VolumeRemain int
		LocationID   int64
	}
	for _, order := range orders {
		if order.IsBuyOrder {
			buyOrders = append(buyOrders, struct {
				Price        float64
				VolumeRemain int
				LocationID   int64
			}{
				Price:        order.Price,
				VolumeRemain: order.VolumeRemain,
				LocationID:   order.LocationID,
			})
		}
	}

	log.Printf("[DEBUG] InventorySell: Found %d buy orders for type_id=%d in region_id=%d", len(buyOrders), req.TypeID, req.RegionID)

	// Calculate routes for each buy order
	var routes []models.InventorySellRoute
	skipped := map[string]int{}
	for _, order := range buyOrders {
		// Calculate net price after tax
		netPrice := order.Price * (1 - taxRate)
		profitPerUnit := netPrice - req.BuyPricePerUnit

		log.Printf("[DEBUG] Order: price=%.2f ISK, taxRate=%.4f, netPrice=%.2f ISK, buyPrice=%.2f ISK, profit=%.2f ISK, minProfit=%.2f ISK",
			order.Price, taxRate, netPrice, req.BuyPricePerUnit, profitPerUnit, req.MinProfitPerUnit)

		// Filter by minimum profit
		if profitPerUnit < req.MinProfitPerUnit {
			skipped["profit_too_low"]++
			log.Printf("[DEBUG] Skipped: profit %.2f < min %.2f", profitPerUnit, req.MinProfitPerUnit)
			continue
		}

		// Calculate available quantity
		availableQuantity := req.Quantity
		if order.VolumeRemain < availableQuantity {
			availableQuantity = order.VolumeRemain
		}

		// Get station/system information
		systemID, err := h.handler.sdeRepo.GetSystemIDForLocation(c.Context(), order.LocationID)
		if err != nil {
			skipped["invalid_location"]++
			continue // Skip invalid locations
		}

		// Calculate route navigation
		travelResult, err := navigation.ShortestPath(h.handler.db.SDE, startSystemID, systemID, false)
		if err != nil {
			skipped["navigation_failed"]++
			continue // Skip if route calculation fails
		}

		// Get min security status of route
		minRouteSecurity := h.getMinRouteSecurityStatus(c.Context(), travelResult.Route)

		// Apply security filter
		if req.SecurityFilter == "highsec" && minRouteSecurity < 0.5 {
			skipped["security_highsec"]++
			continue
		}
		if req.SecurityFilter == "highlow" && minRouteSecurity <= 0.0 {
			skipped["security_highlow"]++
			continue
		}

		systemName, _ := h.handler.sdeRepo.GetSystemName(c.Context(), systemID)
		stationName, _ := h.handler.sdeRepo.GetStationName(c.Context(), order.LocationID)

		route := models.InventorySellRoute{
			SellStationID:          order.LocationID,
			SellStationName:        stationName,
			SellSystemID:           systemID,
			SellSystemName:         systemName,
			SellSecurityStatus:     h.getSystemSecurityStatus(c.Context(), systemID),
			BuyOrderPrice:          order.Price,
			TaxRate:                taxRate,
			NetPricePerUnit:        netPrice,
			ProfitPerUnit:          profitPerUnit,
			AvailableQuantity:      availableQuantity,
			TotalProfit:            profitPerUnit * float64(availableQuantity),
			RouteJumps:             len(travelResult.Route) - 1,
			RouteSystemIDs:         travelResult.Route,
			MinRouteSecurityStatus: minRouteSecurity,
		}

		routes = append(routes, route)
	}

	// Sort by profit per unit (descending)
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].ProfitPerUnit > routes[j].ProfitPerUnit
	})

	log.Printf("[DEBUG] InventorySell: Generated %d routes. Skipped: %+v", len(routes), skipped)

	return c.JSON(fiber.Map{
		"routes": routes,
		"count":  len(routes),
	})
}

// getMinRouteSecurityStatus finds the minimum security status across all systems in a route
func (h *TradingHandler) getMinRouteSecurityStatus(ctx context.Context, route []int64) float64 {
	if len(route) == 0 {
		return 1.0
	}

	minSec := 1.0
	for _, systemID := range route {
		sec := h.getSystemSecurityStatus(ctx, systemID)
		if sec < minSec {
			minSec = sec
		}
	}
	return minSec
}

// getSystemSecurityStatus retrieves security status for a system
func (h *TradingHandler) getSystemSecurityStatus(ctx context.Context, systemID int64) float64 {
	query := `SELECT security FROM mapSolarSystems WHERE _key = ?`
	var security float64
	if err := h.handler.db.SDE.QueryRowContext(ctx, query, systemID).Scan(&security); err != nil {
		return 1.0 // Default to high-sec on error
	}
	return security
}
