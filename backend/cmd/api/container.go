package main

import (
	"context"
	"log"
	"time"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/handlers"
	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/esi"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evesso"
	applogger "github.com/Sternrassler/eve-o-provit/backend/pkg/logger"
	"github.com/redis/go-redis/v9"
)

// AppContainer holds all initialized application dependencies.
type AppContainer struct {
	Redis     *redis.Client
	DB        *database.DB
	SDERepo   *database.SDERepository
	MarketRepo *database.MarketRepository
	ESIClient *esi.Client
	AppLogger *applogger.Logger

	// Handlers
	AuthHandler        *evesso.AuthHandler
	Handlers           *handlers.Handler
	TradingHandler     *handlers.TradingHandler
	CharacterHandler   *handlers.CharacterHandler
	FittingHandler     *handlers.FittingHandler
	CalculationHandler *handlers.CalculationHandler
}

// NewContainer initializes all application dependencies and returns a ready-to-use AppContainer.
func NewContainer(ctx context.Context) (*AppContainer, error) {
	c := &AppContainer{}

	// EVE SSO Config (public client — PKCE flow, no client_secret needed)
	eveClientID := getEnv("EVE_CLIENT_ID", "")
	if eveClientID == "" {
		log.Fatal("EVE_CLIENT_ID environment variable is required")
	}

	// Redis
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")
	redisOpts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	c.Redis = redis.NewClient(redisOpts)

	if err := c.Redis.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("Redis connection established")
	}

	// Database
	databaseURL := getEnv("DATABASE_URL", "")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	dbConfig := database.Config{
		PostgresURL: databaseURL,
		SDEPath:     getEnv("SDE_PATH", "data/sde/eve-sde.db"),
	}
	c.DB, err = database.New(ctx, dbConfig)
	if err != nil {
		c.Redis.Close()
		return nil, err
	}
	log.Println("Database connections established")

	// Repositories
	c.SDERepo = database.NewSDERepository(c.DB.SDE)
	c.MarketRepo = database.NewMarketRepository(c.DB.Postgres)

	// ESI Client
	esiConfig := esi.Config{
		UserAgent:      getEnv("ESI_USER_AGENT", "eve-o-provit/0.1.0 (your-email@example.com)"),
		RateLimit:      getEnvInt("ESI_RATE_LIMIT", 10),
		ErrorThreshold: getEnvInt("ESI_ERROR_THRESHOLD", 15),
		MaxRetries:     getEnvInt("ESI_MAX_RETRIES", 3),
	}
	c.ESIClient, err = esi.NewClient(c.Redis, esiConfig)
	if err != nil {
		c.DB.Close()
		c.Redis.Close()
		return nil, err
	}
	log.Println("ESI client initialized")

	// Application logger
	c.AppLogger = applogger.New()

	// Services
	characterHelper := services.NewCharacterHelper(c.Redis)
	skillsService := services.NewSkillsService(c.ESIClient.GetRawClient(), c.Redis, c.AppLogger)
	fittingService := services.NewFittingService(c.ESIClient.GetRawClient(), c.DB.SDE, c.Redis, skillsService, c.AppLogger)
	cargoService := services.NewCargoService(skillsService, fittingService)
	feeService := services.NewFeeService(skillsService, c.AppLogger)

	routeConfig := services.Config{
		CalculationTimeout:      time.Duration(getEnvInt("ROUTE_CALCULATION_TIMEOUT", 120)) * time.Second,
		MarketFetchTimeout:      time.Duration(getEnvInt("ROUTE_MARKET_FETCH_TIMEOUT", 60)) * time.Second,
		RouteCalculationTimeout: time.Duration(getEnvInt("ROUTE_ROUTE_CALC_TIMEOUT", 90)) * time.Second,
	}
	routeService := services.NewRouteService(c.ESIClient, c.DB.SDE, c.SDERepo, c.MarketRepo, c.Redis, cargoService, fittingService, skillsService, feeService, routeConfig)
	shipService := services.NewShipService(c.DB.SDE)
	systemService := services.NewSystemService(c.SDERepo)

	// Auth handler
	eveCallbackURL := getEnv("EVE_CALLBACK_URL", "http://localhost:9000/callback")
	c.AuthHandler = evesso.NewAuthHandler(eveClientID, eveCallbackURL)

	// Request handlers
	c.Handlers = handlers.New(c.DB, c.SDERepo, c.MarketRepo, c.ESIClient)
	c.TradingHandler = handlers.NewTradingHandler(routeService, c.SDERepo, shipService, systemService, characterHelper, cargoService)
	c.CharacterHandler = handlers.NewCharacterHandler(skillsService)
	c.FittingHandler = handlers.NewFittingHandler(fittingService)
	c.CalculationHandler = handlers.NewCalculationHandler(c.DB.SDE, fittingService)

	return c, nil
}

// Close releases all resources held by the container.
func (c *AppContainer) Close() {
	if c.ESIClient != nil {
		c.ESIClient.Close()
	}
	if c.DB != nil {
		c.DB.Close()
	}
	if c.Redis != nil {
		c.Redis.Close()
	}
}
