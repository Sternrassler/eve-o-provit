package main

import (
	"log"
	"os"

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
		AllowOrigins: getEnv("CORS_ORIGINS", "http://localhost:3000"),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "eve-o-provit-api",
		})
	})

	// API Routes (TODO)
	api := app.Group("/api/v1")

	// Trading endpoints
	trading := api.Group("/trading")
	trading.Get("/profit-margins", handleProfitMargins)

	// Manufacturing endpoints
	manufacturing := api.Group("/manufacturing")
	manufacturing.Get("/blueprints", handleBlueprints)

	// Start server
	port := getEnv("PORT", "8080")
	log.Printf("Starting EVE-O-Provit API on port %s", port)
	log.Fatal(app.Listen(":" + port))
}

// Placeholder handlers (TODO: implement)
func handleProfitMargins(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Profit margins endpoint - TODO",
	})
}

func handleBlueprints(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Blueprints endpoint - TODO",
	})
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
