package main

import (
	"context"
	"fmt"
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
	Redis      *redis.Client
	DB         *database.DB
	SDERepo    *database.SDERepository
	MarketRepo *database.MarketRepository
	ESIClient  *esi.Client
	AppLogger  *applogger.Logger

	// Auth
	TokenValidator *evesso.TokenValidator

	// Handlers
	AuthHandler        *evesso.AuthHandler
	Handlers           *handlers.Handler
	TradingHandler     *handlers.TradingHandler
	CharacterHandler   *handlers.CharacterHandler
	FittingHandler     *handlers.FittingHandler
	CalculationHandler *handlers.CalculationHandler
	MultiHubHandler    *handlers.MultiHubHandler
	PortfolioHandler   *handlers.PortfolioHandler
	HaulingHandler     *handlers.HaulingHandler

	// Background workers
	CompetitionCollector *services.CompetitionCollector
}

// NewContainer initializes all application dependencies and returns a ready-to-use AppContainer.
func NewContainer(ctx context.Context) (*AppContainer, error) {
	c := &AppContainer{}
	var err error

	// EVE SSO Config (public client — PKCE flow, no client_secret needed)
	eveClientID := getEnv("EVE_CLIENT_ID", "")
	if eveClientID == "" {
		log.Fatal("EVE_CLIENT_ID environment variable is required")
	}

	c.TokenValidator, err = evesso.NewTokenValidator(ctx, eveClientID)
	if err != nil {
		return nil, fmt.Errorf("init token validator: %w", err)
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
	c.AuthHandler = evesso.NewAuthHandler(eveClientID, eveCallbackURL, c.TokenValidator)

	// Mobile EVE application (optional — set EVE_MOBILE_CLIENT_ID to enable)
	mobileClientID := getEnv("EVE_MOBILE_CLIENT_ID", "")
	if mobileClientID != "" {
		mobileRedirectURI := getEnv("EVE_MOBILE_REDIRECT_URI", "eveauth-eveoprovit://callback")
		c.AuthHandler.WithMobile(mobileClientID, mobileRedirectURI)
		c.TokenValidator.AddAcceptedClientID(mobileClientID)
		log.Printf("Mobile auth enabled (client_id=%s, redirect=%s)", mobileClientID, mobileRedirectURI)
	}

	// Request handlers
	c.Handlers = handlers.New(c.DB, c.SDERepo, c.MarketRepo, c.ESIClient)
	c.TradingHandler = handlers.NewTradingHandler(routeService, c.SDERepo, shipService, systemService, characterHelper, cargoService)
	c.CharacterHandler = handlers.NewCharacterHandler(skillsService)
	c.FittingHandler = handlers.NewFittingHandler(fittingService)
	c.CalculationHandler = handlers.NewCalculationHandler(c.DB.SDE, fittingService)

	// Multi-Hub Comparison (#43): competition tracking + hub comparison service.
	competitionRepo := database.NewCompetitionRepository(c.DB.Postgres)
	competitionService := services.NewCompetitionService(competitionRepo, c.MarketRepo, c.AppLogger)
	c.CompetitionCollector = services.NewCompetitionCollector(competitionRepo, c.ESIClient, c.AppLogger)
	volumeService := services.NewVolumeService(c.MarketRepo, c.ESIClient)
	multiHubService := services.NewMultiHubComparisonService(skillsService, c.ESIClient, volumeService, competitionService, c.SDERepo, c.AppLogger)
	c.MultiHubHandler = handlers.NewMultiHubHandler(multiHubService)

	// ROI Calculator / capital-allocation optimizer (#44) — reuses the route engine.
	portfolioService := services.NewPortfolioService(routeService, skillsService, c.AppLogger)
	c.PortfolioHandler = handlers.NewPortfolioHandler(portfolioService)

	// Neighborhood hauling routes (#45) — reuses order fetch + navigation + cargo + skills.
	haulingRouteFinder := services.NewRouteFinder(c.ESIClient, c.MarketRepo, c.SDERepo, c.DB.SDE, c.Redis)
	haulingService := services.NewHaulingService(c.SDERepo, haulingRouteFinder, fittingService, skillsService, characterHelper, c.DB.SDE, c.AppLogger)
	c.HaulingHandler = handlers.NewHaulingHandler(haulingService)

	// Start the competition collector in the background (lazy-tracked pairs).
	go c.CompetitionCollector.Start(ctx)

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
