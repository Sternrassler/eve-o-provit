// Package handlers - Trading endpoints
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// TradingHandler handles trading-related HTTP requests
type TradingHandler struct {
	calculator                services.RouteCalculatorServicer // Interface for testability
	sdeQuerier                database.SDEQuerier              // For type info lookups
	shipService               services.ShipServicer            // For ship capacity queries
	systemService             services.SystemServicer          // For system/region/station info
	characterHelper           *services.CharacterHelper
	tradingService            *services.TradingService
	inventorySellOrchestrator services.InventorySellOrchestrator // New: Orchestrator for business logic
}

// NewTradingHandler creates a new trading handler instance
func NewTradingHandler(
	calculator *services.RouteCalculator,
	sdeQuerier database.SDEQuerier,
	shipService services.ShipServicer,
	systemService services.SystemServicer,
	charHelper *services.CharacterHelper,
	tradingService *services.TradingService,
) *TradingHandler {
	// Create orchestrator (Phase 2 refactoring)
	navigationService := services.NewNavigationService(sdeQuerier)
	orchestrator := services.NewInventorySellOrchestrator(charHelper, navigationService, tradingService)

	return &TradingHandler{
		calculator:                calculator,
		sdeQuerier:                sdeQuerier,
		shipService:               shipService,
		systemService:             systemService,
		characterHelper:           charHelper,
		tradingService:            tradingService,
		inventorySellOrchestrator: orchestrator,
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
	systemInfo, err := h.systemService.GetSystemInfo(ctx, esiLoc.SolarSystemID)
	if err == nil {
		location.SolarSystemName = systemInfo.SystemName
		location.RegionID = systemInfo.RegionID
		location.RegionName = systemInfo.RegionName
	}

	if esiLoc.StationID != nil {
		stationName, err := h.systemService.GetStationName(ctx, *esiLoc.StationID)
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
	typeInfo, err := h.sdeQuerier.GetTypeInfo(ctx, int(esiShip.ShipTypeID))
	if err == nil {
		ship.ShipTypeName = typeInfo.Name
	}

	capacities, err := h.shipService.GetShipCapacities(ctx, esiShip.ShipTypeID)
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
		typeInfo, err := h.sdeQuerier.GetTypeInfo(ctx, int(asset.TypeID))
		if err != nil {
			continue
		}

		// Check if it's a ship (categoryID = 6)
		if typeInfo.CategoryID == nil || *typeInfo.CategoryID != 6 {
			continue
		}

		// Get cargo capacity
		capacities, err := h.shipService.GetShipCapacities(ctx, asset.TypeID)
		if err != nil {
			// Skip if we can't get ship capacities (probably not a ship)
			continue
		}

		locationName, _ := h.systemService.GetStationName(ctx, asset.LocationID)

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
	items, err := h.sdeQuerier.SearchItems(c.Context(), query, limit)
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
	// 1. Parse request
	var req models.InventorySellRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// 2. Validate request
	if err := req.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// 3. Extract auth context
	characterID := c.Locals("character_id").(int)
	accessToken := c.Locals("access_token").(string)

	// 4. Delegate to orchestrator (single call)
	routes, err := h.inventorySellOrchestrator.CalculateSellRoutes(
		c.Context(), req, characterID, accessToken,
	)
	if err != nil {
		// Handle business errors with appropriate status codes
		if be, ok := services.IsBusinessError(err); ok {
			return c.Status(be.Status).JSON(fiber.Map{
				"error": be.Message,
			})
		}
		// Handle internal errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to calculate sell routes",
			"details": err.Error(),
		})
	}

	// 5. Return response
	return c.JSON(fiber.Map{
		"routes": routes,
		"count":  len(routes),
	})
}
