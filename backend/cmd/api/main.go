// Package main is the entry point for EVE-O-Provit API
//
// @title EVE-O-Provit API
// @version 0.28.0
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

	_ "github.com/Sternrassler/eve-o-provit/backend/internal/models" // For OpenAPI
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	_ "github.com/Sternrassler/eve-o-provit/backend/docs" // Import generated docs
)

func main() {
	ctx := context.Background()

	c, err := NewContainer(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}
	defer c.Close()

	app := setupApp(c)

	port := getEnv("PORT", "8080")
	log.Printf("Starting EVE-O-Provit API on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

func setupApp(c *AppContainer) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: "EVE-O-Provit API v0.1.0",
	})

	// CORS — AllowCredentials:true requires explicit origins (never wildcard).
	// Setting CORS_ORIGINS=* with credentials enabled is a security misconfiguration.
	corsOrigins := getEnv("CORS_ORIGINS", "http://localhost:9000")
	for _, origin := range strings.Split(corsOrigins, ",") {
		if strings.TrimSpace(origin) == "*" {
			log.Fatal("CORS_ORIGINS must not contain '*' when AllowCredentials is true — set explicit origins")
		}
	}

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, Cookie",
		AllowMethods:     "GET, POST, OPTIONS",
		AllowCredentials: true,
	}))

	// Prometheus metrics (internal — not rate limited)
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	// Swagger UI (public, no auth)
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Rate limiting for expensive/auth endpoints
	routeCalcLimiter := limiter.New(limiter.Config{
		Max:        getEnvInt("RATE_LIMIT_CALCULATE", 10),
		Expiration: 60 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})
	authLimiter := limiter.New(limiter.Config{
		Max:        getEnvInt("RATE_LIMIT_AUTH", 20),
		Expiration: 60 * time.Second,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})

	// Auth routes (rate limited)
	auth := app.Group("/auth", authLimiter)
	auth.Post("/callback", c.AuthHandler.HandleCallback)
	auth.Get("/session", c.AuthHandler.HandleSession)
	auth.Post("/refresh", c.AuthHandler.HandleRefresh)
	auth.Post("/logout", c.AuthHandler.HandleLogout)
	auth.Post("/mobile/callback", c.AuthHandler.HandleMobileCallback)
	auth.Post("/mobile/refresh", c.AuthHandler.HandleMobileRefresh)

	// API routes
	api := app.Group("/api/v1")

	api.Get("/health", c.Handlers.Health)
	api.Get("/version", c.Handlers.Version)
	api.Get("/types/:id", c.Handlers.GetType)
	api.Get("/sde/regions", c.Handlers.GetRegions)
	api.Get("/market/staleness/:region", c.Handlers.GetMarketDataStaleness)
	api.Get("/market/:region/:type", c.Handlers.GetMarketOrders)

	api.Post("/trading/routes/calculate", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.TradingHandler.CalculateRoutes)
	api.Post("/trading/hubs/compare", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.MultiHubHandler.CompareHubs)
	api.Post("/trading/portfolio/optimize", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.PortfolioHandler.Optimize)
	api.Post("/trading/hauling/routes", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.HaulingHandler.FindRoutes)
	api.Get("/trading/assets", evesso.NewAuthMiddleware(c.TokenValidator), c.AssetsHandler.ListAssets)
	api.Post("/trading/assets/sell-options", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.AssetsHandler.SellOptions)
	api.Post("/mining/ore-ranking", routeCalcLimiter, evesso.NewAuthMiddleware(c.TokenValidator), c.MiningHandler.OreRanking)
	api.Get("/items/search", c.TradingHandler.SearchItems)

	// Protected routes
	protected := api.Group("", evesso.NewAuthMiddleware(c.TokenValidator))

	protected.Get("/character", handleCharacterInfo)
	protected.Get("/character/location", c.TradingHandler.GetCharacterLocation)
	protected.Get("/character/ship", c.TradingHandler.GetCharacterShip)
	protected.Get("/character/ships", c.TradingHandler.GetCharacterShips)
	protected.Get("/character/wallet", c.TradingHandler.GetCharacterWallet)

	protected.Get("/characters/:characterId/skills", c.CharacterHandler.GetCharacterSkills)
	protected.Get("/characters/:characterId/fitting/:shipTypeId", c.FittingHandler.GetCharacterFitting)

	esiUI := protected.Group("/esi/ui")
	esiUI.Post("/autopilot/waypoint", c.TradingHandler.SetAutopilotWaypoint)
	esiUI.Post("/openwindow/marketdetails", c.TradingHandler.OpenMarketDetails)

	trading := protected.Group("/trading")
	trading.Get("/profit-margins", handleProfitMargins)

	manufacturing := protected.Group("/manufacturing")
	manufacturing.Get("/blueprints", handleBlueprints)

	return app
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
	characterID, ok := c.Locals("character_id").(int)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	characterName, ok := c.Locals("character_name").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	scopes, ok := c.Locals("scopes").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	return c.JSON(fiber.Map{
		"character_id":   characterID,
		"character_name": characterName,
		"scopes":         strings.Split(scopes, " "),
		"portrait_url":   evesso.GetPortraitURL(characterID, 128),
	})
}

func handleProfitMargins(c *fiber.Ctx) error {
	characterName, ok := c.Locals("character_name").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(fiber.Map{
		"message":    "Profit margins endpoint - TODO",
		"authorized": true,
		"character":  characterName,
	})
}

func handleBlueprints(c *fiber.Ctx) error {
	characterName, ok := c.Locals("character_name").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
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
