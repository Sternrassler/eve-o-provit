// Package main is the entry point for EVE-O-Provit API
//
// @title EVE-O-Provit API
// @version 0.1.0
// @description REST API for EVE Online trading, manufacturing and profit optimization
// @description
// @description Features:
// @description - Intra-region trading route calculation
// @description - Character skills and ship fitting integration
// @description - Deterministic cargo/warp speed calculations
// @description - Market data caching with staleness tracking
// @description - EVE SSO authentication
//
// @contact.name EVE-O-Provit Team
// @contact.url https://github.com/Sternrassler/eve-o-provit
// @contact.email support@example.com
//
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
//
// @host localhost:8080
// @BasePath /
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description EVE SSO Bearer Token. Format: "Bearer {token}"
//
// @tag.name Health
// @tag.description Health check and version endpoints
//
// @tag.name SDE
// @tag.description Static Data Export (regions, types, etc.)
//
// @tag.name Market
// @tag.description Market orders and data staleness
//
// @tag.name Character
// @tag.description Character information, skills, location
//
// @tag.name Trading
// @tag.description Trading route calculation and item search
//
// @tag.name Fitting
// @tag.description Ship fitting with deterministic bonus calculations
//
// @tag.name Calculations
// @tag.description Deterministic ship bonus calculations (cargo, warp, inertia)
//
// @tag.name ESI
// @tag.description Direct ESI proxy endpoints (UI operations)
package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/handlers"
	_ "github.com/Sternrassler/eve-o-provit/backend/internal/models" // For OpenAPI
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
	applogger "github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/redis/go-redis/v9"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "github.com/Sternrassler/eve-o-provit/backend/docs" // Import generated docs
)

func main() {
	ctx := context.Background()

	// Initialize Redis
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Redis connection established")
	}

	// Initialize Database
	dbConfig := database.Config{
		PostgresURL: getEnv("DATABASE_URL", "postgresql://eveprovit:dev@localhost:5432/eveprovit?sslmode=disable"),
		SDEPath:     getEnv("SDE_PATH", "data/sde/eve-sde.db"),
	}

	db, err := database.New(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to databases: %v", err)
	}
	defer db.Close()

	log.Println("Database connections established")

	// Initialize repositories
	sdeRepo := database.NewSDERepository(db.SDE)
	marketRepo := database.NewMarketRepository(db.Postgres)

	// Initialize ESI Client
	esiConfig := esi.Config{
		UserAgent:      getEnv("ESI_USER_AGENT", "eve-o-provit/0.1.0 (your-email@example.com)"),
		RateLimit:      getEnvInt("ESI_RATE_LIMIT", 10),
		ErrorThreshold: getEnvInt("ESI_ERROR_THRESHOLD", 15),
		MaxRetries:     getEnvInt("ESI_MAX_RETRIES", 3),
	}

	esiClient, err := esi.NewClient(redisClient, esiConfig, marketRepo)
	if err != nil {
		log.Fatalf("Failed to create ESI client: %v", err)
	}
	defer esiClient.Close()

	log.Println("ESI client initialized")

	// Initialize application logger
	appLogger := applogger.New()

	characterHelper := services.NewCharacterHelper(redisClient)

	// Skills Service (Phase 0 - Issue #54)
	skillsService := services.NewSkillsService(esiClient.GetRawClient(), redisClient, appLogger)

	// Fitting Service (Phase 3 - Issue #76 - Ship Fitting Integration)
	fittingService := services.NewFittingService(esiClient.GetRawClient(), db.SDE, redisClient, skillsService, appLogger)

	// Cargo Service (Phase 0 - Issue #56 - Cargo Skills Integration + Phase 3 Fitting)
	cargoService := services.NewCargoService(skillsService, fittingService)

	// Fee Service (Phase 0 - Issue #55)
	feeService := services.NewFeeService(skillsService, appLogger)

	// Route Service Configuration
	routeConfig := services.Config{
		CalculationTimeout:      time.Duration(getEnvInt("ROUTE_CALCULATION_TIMEOUT", 120)) * time.Second,
		MarketFetchTimeout:      time.Duration(getEnvInt("ROUTE_MARKET_FETCH_TIMEOUT", 60)) * time.Second,
		RouteCalculationTimeout: time.Duration(getEnvInt("ROUTE_ROUTE_CALC_TIMEOUT", 90)) * time.Second,
	}

	// Route Service with cargo + fee integration
	routeService := services.NewRouteService(esiClient, db.SDE, sdeRepo, marketRepo, redisClient, cargoService, skillsService, feeService, routeConfig)

	// Ship Service (Phase 0 - Issue #57 - Remove Raw DB Access)
	shipService := services.NewShipService(db.SDE)

	// System Service (Phase 0 - Issue #57 - Remove Raw DB Access)
	systemService := services.NewSystemService(sdeRepo)

	// Initialize handlers
	h := handlers.New(db, sdeRepo, marketRepo, esiClient)
	tradingHandler := handlers.NewTradingHandler(routeService, sdeRepo, shipService, systemService, characterHelper, cargoService)
	characterHandler := handlers.NewCharacterHandler(skillsService)
	fittingHandler := handlers.NewFittingHandler(fittingService)
	calculationHandler := handlers.NewCalculationHandler(db.SDE, fittingService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName: "EVE-O-Provit API v0.1.0",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     getEnv("CORS_ORIGINS", "http://localhost:9000"),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Swagger UI (public, no auth)
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// API Routes
	api := app.Group("/api/v1")

	// Public health endpoints
	api.Get("/health", h.Health)
	api.Get("/version", h.Version)

	// Public SDE endpoints
	api.Get("/types/:id", h.GetType)
	api.Get("/sde/regions", h.GetRegions)

	// Public market endpoints
	api.Get("/market/staleness/:region", h.GetMarketDataStaleness)
	api.Get("/market/:region/:type", h.GetMarketOrders)

	// Trading routes (authentication required)
	api.Post("/trading/routes/calculate", evesso.AuthMiddleware, tradingHandler.CalculateRoutes)

	// Item search endpoint (public)
	api.Get("/items/search", tradingHandler.SearchItems)

	// Calculation endpoints (public - deterministic calculations)
	api.Post("/calculations/cargo", calculationHandler.CalculateCargo)
	api.Post("/calculations/warp", calculationHandler.CalculateWarp)

	// Protected routes (require Bearer token)
	protected := api.Group("", evesso.AuthMiddleware)

	// Character info endpoint
	protected.Get("/character", handleCharacterInfo)

	// Character location & ship endpoints (used by frontend for auto-selection)
	protected.Get("/character/location", tradingHandler.GetCharacterLocation)
	protected.Get("/character/ship", tradingHandler.GetCharacterShip)
	protected.Get("/character/ships", tradingHandler.GetCharacterShips)

	// Character context endpoints
	// Character skills endpoint (Issue #54)
	protected.Get("/characters/:characterId/skills", characterHandler.GetCharacterSkills)

	// Character fitting endpoint (Issue #76 - Phase 3)
	protected.Get("/characters/:characterId/fitting/:shipTypeId", fittingHandler.GetCharacterFitting)

	// ESI UI endpoints (require esi-ui.write_waypoint.v1 scope)
	esiUI := protected.Group("/esi/ui")
	esiUI.Post("/autopilot/waypoint", tradingHandler.SetAutopilotWaypoint)

	// Trading endpoints
	trading := protected.Group("/trading")
	trading.Get("/profit-margins", handleProfitMargins)

	// Manufacturing endpoints
	manufacturing := protected.Group("/manufacturing")
	manufacturing.Get("/blueprints", handleBlueprints)

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Starting EVE-O-Provit API on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// handleCharacterInfo handles GET /api/v1/character
//
// @Summary Get character info
// @Description Get authenticated character information
// @Tags Character
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "Character info with character_id, character_name, scopes, portrait_url"
// @Failure 401 {object} models.ErrorResponse
// @Router /api/v1/character [get]
func handleCharacterInfo(c *fiber.Ctx) error {
	characterID := c.Locals("character_id").(int)
	characterName := c.Locals("character_name").(string)
	scopes := c.Locals("scopes").(string)

	return c.JSON(fiber.Map{
		"character_id":   characterID,
		"character_name": characterName,
		"scopes":         strings.Split(scopes, " "),
		"portrait_url":   evesso.GetPortraitURL(characterID, 128),
	})
}

func handleProfitMargins(c *fiber.Ctx) error {
	characterName := c.Locals("character_name").(string)

	return c.JSON(fiber.Map{
		"message":    "Profit margins endpoint - TODO",
		"authorized": true,
		"character":  characterName,
	})
}

func handleBlueprints(c *fiber.Ctx) error {
	characterName := c.Locals("character_name").(string)

	return c.JSON(fiber.Map{
		"message":    "Blueprints endpoint - TODO",
		"authorized": true,
		"character":  characterName,
	})
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
