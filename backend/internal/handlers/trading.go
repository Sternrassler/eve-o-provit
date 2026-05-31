// Package handlers - Trading endpoints
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	_ "github.com/Sternrassler/eve-o-provit/backend/internal/models" // For OpenAPI
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/gofiber/fiber/v2"
)

// TradingHandler handles trading-related HTTP requests
type TradingHandler struct {
	calculator      services.RouteCalculatorServicer // Interface for testability
	sdeQuerier      database.SDEQuerier              // For type info lookups
	shipService     services.ShipServicer            // For ship capacity queries
	systemService   services.SystemServicer          // For system/region/station info
	characterHelper *services.CharacterHelper
	cargoService    services.CargoServicer   // For effective cargo capacity calculation
	fittingService  services.FittingServicer // For effective cargo per ship (dropdown enrichment)
}

// NewTradingHandler creates a new trading handler instance
func NewTradingHandler(
	calculator services.RouteCalculatorServicer,
	sdeQuerier database.SDEQuerier,
	shipService services.ShipServicer,
	systemService services.SystemServicer,
	charHelper *services.CharacterHelper,
	cargoService services.CargoServicer,
	fittingService services.FittingServicer,
) *TradingHandler {
	return &TradingHandler{
		calculator:      calculator,
		sdeQuerier:      sdeQuerier,
		shipService:     shipService,
		systemService:   systemService,
		characterHelper: charHelper,
		cargoService:    cargoService,
		fittingService:  fittingService,
	}
}

// CalculateRoutes handles POST /api/v1/trading/routes/calculate
// Supports optional authentication for skill-aware cargo calculations
// Supports optional volume filtering for liquidity-based route selection
//
// @Summary Calculate trading routes
// @Description Calculate optimal intra-region trading routes with profit analysis
// @Description Uses character skills and ship fitting for accurate cargo capacity
// @Description Supports deterministic navigation parameters (warp_speed, align_time) from frontend fitting calculation
// @Description Supports volume filtering for liquidity-based selection
// @Tags Trading
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.RouteCalculationRequest true "Route calculation request"
// @Success 200 {object} models.RouteCalculationResponse "Successfully calculated routes"
// @Success 206 {object} models.RouteCalculationResponse "Partial results (timeout)"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/trading/routes/calculate [post]
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

	// Create context with optional character info for skill-aware calculations
	ctx := c.UserContext()

	// Extract required character authentication (set by AuthMiddleware)
	characterID := c.Locals("character_id")
	accessToken := c.Locals("access_token")

	if characterID == nil || accessToken == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required for trading operations",
		})
	}

	// Add character context for skill-aware cargo calculations
	ctx = context.WithValue(ctx, services.CtxKeyCharacterID, characterID)
	ctx = context.WithValue(ctx, services.CtxKeyAccessToken, accessToken)

	// Extract deterministic navigation parameters from request
	var warpSpeed, alignTime *float64
	if req.WarpSpeed > 0 {
		warpSpeed = &req.WarpSpeed
	}
	if req.AlignTime > 0 {
		alignTime = &req.AlignTime
	}

	// Calculate routes (with or without volume filtering)
	var result *models.RouteCalculationResponse
	var err error

	// Use CalculateWithFilters if volume metrics requested or filters applied
	if req.IncludeVolumeMetrics || req.MinDailyVolume > 0 || req.MaxLiquidationDays > 0 {
		result, err = h.calculator.CalculateWithFilters(ctx, &req)
	} else {
		result, err = h.calculator.Calculate(ctx, req.RegionID, req.ShipTypeID, req.CargoCapacity, warpSpeed, alignTime)
	}

	if err != nil {
		log.Printf("ERROR: CalculateRoutes failed for regionID=%d: %v path=%s", req.RegionID, err, c.Path())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to calculate routes",
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
//
// @Summary Get character location
// @Description Get character's current location (solar system)
// @Tags Character
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "Location data with solar_system_id, station_id, structure_id"
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/character/location [get]
func (h *TradingHandler) GetCharacterLocation(c *fiber.Ctx) error {
	characterID, ok := c.Locals("character_id").(int)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}
	accessToken, ok := c.Locals("access_token").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}

	// Call ESI
	location, err := h.fetchESICharacterLocation(c.Context(), characterID, accessToken)
	if err != nil {
		if err.Error() == "unauthorized" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		log.Printf("ERROR: GetCharacterLocation failed for characterID=%d: %v", characterID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch character location",
		})
	}

	return c.JSON(location)
}

// GetCharacterShip handles GET /api/v1/character/ship
//
// @Summary Get current ship
// @Description Get character's current active ship
// @Tags Character
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "Ship data with ship_item_id, ship_name, ship_type_id, ship_type_name"
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/character/ship [get]
func (h *TradingHandler) GetCharacterShip(c *fiber.Ctx) error {
	characterID, ok := c.Locals("character_id").(int)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}
	accessToken, ok := c.Locals("access_token").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}

	// Call ESI
	ship, err := h.fetchESICharacterShip(c.Context(), characterID, accessToken)
	if err != nil {
		if err.Error() == "unauthorized" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		log.Printf("ERROR: GetCharacterShip failed for characterID=%d: %v", characterID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch character ship",
		})
	}

	return c.JSON(ship)
}

// GetCharacterShips handles GET /api/v1/character/ships
//
// @Summary Get character ships
// @Description Get list of all character's ships in current hangar
// @Tags Character
// @Security BearerAuth
// @Produce json
// @Success 200 {array} map[string]interface{} "Array of ships with ship_item_id, ship_name, ship_type_id, ship_type_name"
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/character/ships [get]
func (h *TradingHandler) GetCharacterShips(c *fiber.Ctx) error {
	characterID, ok := c.Locals("character_id").(int)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}
	accessToken, ok := c.Locals("access_token").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}

	// Call ESI
	ships, err := h.fetchESICharacterShips(c.Context(), characterID, accessToken)
	if err != nil {
		if err.Error() == "unauthorized" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Not authenticated",
			})
		}
		log.Printf("ERROR: GetCharacterShips failed for characterID=%d: %v", characterID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch character ships",
		})
	}

	return c.JSON(ships)
}

// GetCharacterWallet handles GET /api/v1/character/wallet
//
// @Summary Get character wallet balance
// @Description Get the authenticated character's wallet balance in ISK (used to prefill ROI capital)
// @Tags Character
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "Object with balance (float, ISK)"
// @Failure 401 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/character/wallet [get]
func (h *TradingHandler) GetCharacterWallet(c *fiber.Ctx) error {
	characterID, ok := c.Locals("character_id").(int)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}
	accessToken, ok := c.Locals("access_token").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}

	balance, err := h.characterHelper.GetWalletBalance(c.Context(), characterID, accessToken)
	if err != nil {
		log.Printf("ERROR: GetCharacterWallet failed for characterID=%d: %v", characterID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch wallet balance",
		})
	}

	return c.JSON(fiber.Map{"balance": balance})
}

// OpenMarketDetails handles POST /api/v1/esi/ui/openwindow/marketdetails
// Opens the EVE client's market-details window for a given type via ESI UI API.
//
// @Summary Open market details window in EVE client
// @Description Opens the market window for a given type_id in the EVE client.
// @Description Requires scope: esi-ui.open_window.v1
// @Tags ESI UI
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{type_id=int} true "Type ID to open"
// @Success 204 "Market details window opened"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/esi/ui/openwindow/marketdetails [post]
func (h *TradingHandler) OpenMarketDetails(c *fiber.Ctx) error {
	accessToken, ok := c.Locals("access_token").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}

	var req struct {
		TypeID int `json:"type_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.TypeID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid type_id"})
	}

	if err := h.openESIMarketDetails(c.Context(), accessToken, req.TypeID); err != nil {
		switch err.Error() {
		case "unauthorized":
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Not authenticated or missing scope esi-ui.open_window.v1"})
		case "not_found":
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "EVE client not running"})
		default:
			log.Printf("ERROR: OpenMarketDetails failed for typeID=%d: %v", req.TypeID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to open market details"})
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// openESIMarketDetails calls ESI /ui/openwindow/marketdetails for the given type.
func (h *TradingHandler) openESIMarketDetails(ctx context.Context, accessToken string, typeID int) error {
	url := fmt.Sprintf("https://esi.evetech.net/latest/ui/openwindow/marketdetails/?type_id=%d", typeID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		return nil
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("unauthorized")
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("not_found")
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("ESI returned status %d: %s", resp.StatusCode, string(body))
}

// SetAutopilotWaypoint handles POST /api/v1/esi/ui/autopilot/waypoint
// Sets a waypoint in the EVE client's autopilot via ESI UI API
//
// @Summary Set autopilot waypoint
// @Description Set autopilot waypoint in EVE client via ESI UI API
// @Description Requires scope: esi-ui.write_waypoint.v1
// @Tags ESI UI
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body object{destination_id=int64,clear_other_waypoints=bool,add_to_beginning=bool} true "Waypoint request"
// @Success 204 "Waypoint set successfully"
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/esi/ui/autopilot/waypoint [post]
func (h *TradingHandler) SetAutopilotWaypoint(c *fiber.Ctx) error {
	// Extract auth context
	accessToken, ok := c.Locals("access_token").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "authentication required"})
	}

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
			log.Printf("ERROR: SetAutopilotWaypoint failed for destinationID=%d: %v", req.DestinationID, err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to set waypoint",
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
		CharacterID:   int64(characterID),
		SolarSystemID: esiLoc.SolarSystemID,
		StationID:     esiLoc.StationID,
		StructureID:   esiLoc.StructureID,
	}

	// Get system and region names from SDE.
	// Fail-loud (issue #147 MED): log lookup errors instead of silently swallowing
	// them; the names are secondary display data so we still return the location.
	systemInfo, err := h.systemService.GetSystemInfo(ctx, esiLoc.SolarSystemID)
	if err != nil {
		log.Printf("WARN: system info lookup failed for systemID=%d: %v", esiLoc.SolarSystemID, err)
	} else {
		location.SolarSystemName = systemInfo.SystemName
		location.RegionID = systemInfo.RegionID
		location.RegionName = systemInfo.RegionName
	}

	if esiLoc.StationID != nil {
		stationName, err := h.systemService.GetStationName(ctx, *esiLoc.StationID)
		if err != nil {
			log.Printf("WARN: station name lookup failed for stationID=%d: %v", *esiLoc.StationID, err)
		} else {
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

	// Get ship type name and cargo capacity.
	// Fail-loud (issue #147 A5): the cargo-capacity lookup is critical — returning
	// CargoCapacity=0 would make the optimizer divide by zero volume and produce
	// garbage routes that look real. Propagate the error so the endpoint fails
	// clearly instead. (The type-name lookup is non-critical display data.)
	typeInfo, err := h.sdeQuerier.GetTypeInfo(ctx, int(esiShip.ShipTypeID))
	if err != nil {
		log.Printf("WARN: ship type name lookup failed for shipTypeID=%d: %v", esiShip.ShipTypeID, err)
	} else {
		ship.ShipTypeName = typeInfo.Name
	}

	capacities, err := h.shipService.GetShipCapacities(ctx, esiShip.ShipTypeID)
	if err != nil {
		return nil, fmt.Errorf("critical: cargo capacity lookup failed for shipTypeID=%d: %w", esiShip.ShipTypeID, err)
	}
	ship.CargoCapacity = capacities.BaseCargoHold

	// Effective cargo (hull + skills + fitted modules) for the dropdown label.
	// Fail-loud (issue #147 A3): a fitting FETCH ERROR is logged and surfaced via
	// EffectiveCargoUnavailable so the client renders a visible degraded state
	// instead of silently falling back to the base hull as if it were effective.
	// "no fitting / no cargo bonus" (err==nil) is NOT an error and leaves the flag
	// unset with EffectiveCargoCapacity==0.
	if h.fittingService != nil {
		fit, ferr := h.fittingService.GetShipFitting(ctx, characterID, int(esiShip.ShipTypeID), accessToken)
		applyEffectiveCargoToShip(ship, fit, ferr)
		if ferr != nil {
			log.Printf("WARN: effective-cargo enrichment failed for characterID=%d shipTypeID=%d: %v (marking effective_cargo_unavailable)", characterID, esiShip.ShipTypeID, ferr)
		}
	}

	return ship, nil
}

// applyEffectiveCargoToShip applies a fitting-fetch result to a CharacterShip per
// the issue #147 A3 contract:
//   - err != nil               → EffectiveCargoUnavailable=true (explicit "unknown")
//   - err == nil, fit/bonus ≤ 0 → leave EffectiveCargoCapacity=0, flag unset
//   - err == nil, bonus > 0     → set EffectiveCargoCapacity
func applyEffectiveCargoToShip(ship *models.CharacterShip, fit *services.FittingData, err error) {
	if err != nil {
		ship.EffectiveCargoUnavailable = true
		return
	}
	if fit != nil && fit.Bonuses.EffectiveCargo > 0 {
		ship.EffectiveCargoCapacity = fit.Bonuses.EffectiveCargo
	}
}

// applyEffectiveCargoToAssetShip is the CharacterAssetShip variant of
// applyEffectiveCargoToShip (same issue #147 A3 contract).
func applyEffectiveCargoToAssetShip(ship *models.CharacterAssetShip, fit *services.FittingData, err error) {
	if err != nil {
		ship.EffectiveCargoUnavailable = true
		return
	}
	if fit != nil && fit.Bonuses.EffectiveCargo > 0 {
		ship.EffectiveCargoCapacity = fit.Bonuses.EffectiveCargo
	}
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

		// Use base cargo capacity for ship list
		// (effective capacity with fitting is shown when ship is selected)
		locationName, lnErr := h.systemService.GetStationName(ctx, asset.LocationID)
		if lnErr != nil {
			// Fail-loud (issue #147 MED): log instead of silently discarding the
			// error; location name is secondary display data so we still proceed.
			log.Printf("WARN: station name lookup failed for locationID=%d: %v", asset.LocationID, lnErr)
		}

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

	// Enrich each singleton ship with its effective cargo (hull + skills +
	// fitted modules) — the same value the optimizer uses. The dropdown then
	// shows the number that matches EVE in-game instead of the bare hull base.
	// Fan out with capped concurrency; fittings are cached, so subsequent loads
	// are cheap. Best-effort: leave EffectiveCargoCapacity == 0 on failure so
	// the client falls back to the base label.
	h.enrichShipsWithEffectiveCargo(ctx, ships, characterID, accessToken)

	return &models.CharacterShipsResponse{
		Ships: ships,
		Count: len(ships),
	}, nil
}

// enrichShipsWithEffectiveCargo fills EffectiveCargoCapacity for each singleton
// ship in-place. Caps concurrency at 4 to avoid hammering ESI; non-singletons
// (packaged) and ships with no resolvable fitting keep EffectiveCargoCapacity=0.
func (h *TradingHandler) enrichShipsWithEffectiveCargo(ctx context.Context, ships []models.CharacterAssetShip, characterID int, accessToken string) {
	if h.fittingService == nil {
		return
	}
	const maxConcurrent = 4
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for i := range ships {
		if !ships[i].IsSingleton {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			fit, err := h.fittingService.GetShipFitting(ctx, characterID, int(ships[idx].TypeID), accessToken)
			applyEffectiveCargoToAssetShip(&ships[idx], fit, err)
			if err != nil {
				// Fail-loud (issue #147 A3): log the fitting fetch error; the
				// EffectiveCargoUnavailable flag set above signals it to the client.
				log.Printf("WARN: effective-cargo enrichment failed for characterID=%d shipTypeID=%d itemID=%d: %v (marking effective_cargo_unavailable)", characterID, ships[idx].TypeID, ships[idx].ItemID, err)
			}
		}(i)
	}
	wg.Wait()
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
//
// @Summary Search EVE items
// @Description Search for EVE Online items by name (fuzzy matching)
// @Tags Trading
// @Produce json
// @Param q query string true "Search query (min 3 characters)" minlength(3)
// @Param limit query int false "Maximum results (default 20, max 100)" minimum(1) maximum(100) default(20)
// @Success 200 {array} models.ItemSearchResult
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/items/search [get]
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
		log.Printf("ERROR: SearchItems failed for query=%q: %v", query, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to search items",
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
