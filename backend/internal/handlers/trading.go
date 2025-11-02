// Package handlers - Trading endpoints
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/gofiber/fiber/v2"
)

// TradingHandler handles trading-related HTTP requests
type TradingHandler struct {
	calculator *services.RouteCalculator
	handler    *Handler
}

// NewTradingHandler creates a new trading handler instance
func NewTradingHandler(calculator *services.RouteCalculator, baseHandler *Handler) *TradingHandler {
	return &TradingHandler{
		calculator: calculator,
		handler:    baseHandler,
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
		SELECT s.solarSystemName, s.regionID, r.regionName
		FROM mapSolarSystems s
		JOIN mapRegions r ON s.regionID = r.regionID
		WHERE s.solarSystemID = ?
	`

	var info systemInfo
	err := h.handler.db.SDE.QueryRowContext(ctx, query, systemID).Scan(
		&info.SystemName,
		&info.RegionID,
		&info.RegionName,
	)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

func (h *TradingHandler) getStationName(ctx context.Context, stationID int64) (string, error) {
	query := `SELECT stationName FROM staStations WHERE stationID = ?`

	var name string
	err := h.handler.db.SDE.QueryRowContext(ctx, query, stationID).Scan(&name)
	if err != nil {
		// Try denormalize table as fallback
		query = `SELECT itemName FROM mapDenormalize WHERE itemID = ?`
		err = h.handler.db.SDE.QueryRowContext(ctx, query, stationID).Scan(&name)
		if err != nil {
			return strconv.FormatInt(stationID, 10), nil
		}
	}

	return name, nil
}
