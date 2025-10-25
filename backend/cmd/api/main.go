package main

import (
	"log"
	"os"
	"strings"
	"time"

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
		AllowOrigins:     getEnv("CORS_ORIGINS", "http://localhost:3000"),
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
	}))

	// Initialize EVE SSO
	authHandler := initializeAuth()

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "eve-o-provit-api",
		})
	})

	// API Routes
	api := app.Group("/api/v1")

	// Auth endpoints
	auth := api.Group("/auth")
	auth.Get("/login", authHandler.HandleLogin)
	auth.Get("/callback", authHandler.HandleCallback)
	auth.Post("/logout", authHandler.HandleLogout)
	auth.Get("/verify", authHandler.HandleVerify)
	auth.Post("/refresh", authHandler.HandleRefresh)
	auth.Get("/character", authHandler.HandleCharacter)

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

// initializeAuth sets up EVE SSO authentication
func initializeAuth() *evesso.Handler {
	// Get configuration from environment
	clientID := getEnv("EVE_CLIENT_ID", "")
	clientSecret := getEnv("EVE_CLIENT_SECRET", "")
	callbackURL := getEnv("EVE_CALLBACK_URL", "http://localhost:8082/api/v1/auth/callback")
	jwtSecret := getEnv("JWT_SECRET", "")

	// Warn if using defaults (for development)
	if clientID == "" {
		log.Println("WARNING: EVE_CLIENT_ID not set, using empty value")
	}
	if clientSecret == "" {
		log.Println("WARNING: EVE_CLIENT_SECRET not set, using empty value")
	}
	if jwtSecret == "" {
		log.Println("WARNING: JWT_SECRET not set, using default (insecure for production)")
		jwtSecret = "default-jwt-secret-change-in-production"
	}

	// Parse scopes from environment
	scopesStr := getEnv("EVE_SCOPES", "publicData")
	scopes := parseScopes(scopesStr)

	// Parse session duration
	durationStr := getEnv("SESSION_DURATION", "24h")
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Printf("WARNING: Invalid SESSION_DURATION '%s', using default 24h", durationStr)
		duration = 24 * time.Hour
	}

	// Create EVE SSO client
	ssoClient := evesso.NewClient(&evesso.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CallbackURL:  callbackURL,
		Scopes:       scopes,
	})

	// Create session manager
	sessionManager := evesso.NewSessionManager(jwtSecret, duration)

	// Create and return handler
	return evesso.NewHandler(ssoClient, sessionManager)
}

// parseScopes splits a comma or space-separated string of scopes
func parseScopes(scopesStr string) []string {
	if scopesStr == "" {
		return []string{}
	}

	// Support both comma and space separation
	var scopes []string
	if strings.Contains(scopesStr, ",") {
		scopes = strings.Split(scopesStr, ",")
	} else {
		scopes = strings.Split(scopesStr, " ")
	}

	// Trim whitespace from each scope
	for i := range scopes {
		scopes[i] = strings.TrimSpace(scopes[i])
	}

	return scopes
}
