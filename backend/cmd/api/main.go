package main

import (
	"log"
	"os"
	"strings"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
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

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "eve-o-provit-api",
		})
	})

	// API Routes
	api := app.Group("/api/v1")

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
