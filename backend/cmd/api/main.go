package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/handlers"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/redis/go-redis/v9"
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
		SDEPath:     getEnv("SDE_PATH", "../eve-sde/data/sqlite/sde.sqlite"),
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

	// Initialize handlers
	h := handlers.New(db, sdeRepo, marketRepo, esiClient)

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

	// Public endpoints
	app.Get("/health", h.Health)
	app.Get("/version", h.Version)

	// API Routes
	api := app.Group("/api/v1")

	// Public SDE endpoints
	api.Get("/types/:id", h.GetType)

	// Public market endpoints
	api.Get("/market/:region/:type", h.GetMarketOrders)

	// Protected routes (require Bearer token)
	protected := api.Group("", evesso.AuthMiddleware)

	// Character info endpoint
	protected.Get("/character", handleCharacterInfo)

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

// Placeholder handlers
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
