// Package esi provides EVE Swagger Interface (ESI) API client functionality
// Wrapper around github.com/Sternrassler/eve-esi-client
package esi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	esiclient "github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/redis/go-redis/v9"
)

// Config holds ESI client configuration
type Config struct {
	UserAgent      string
	RateLimit      int
	ErrorThreshold int
	MaxRetries     int
}

// Client wraps the ESI client with application-specific logic
type Client struct {
	esi  *esiclient.Client
	repo *database.MarketRepository
}

// NewClient creates a new ESI client
func NewClient(redisClient *redis.Client, cfg Config, repo *database.MarketRepository) (*Client, error) {
	// Create ESI client config
	esiCfg := esiclient.DefaultConfig(redisClient, cfg.UserAgent)
	esiCfg.RateLimit = cfg.RateLimit
	esiCfg.ErrorThreshold = cfg.ErrorThreshold
	esiCfg.MaxRetries = cfg.MaxRetries
	esiCfg.RespectExpires = true // ESI Compliance (MUST)

	// Create ESI client
	esiClient, err := esiclient.New(esiCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ESI client: %w", err)
	}

	return &Client{
		esi:  esiClient,
		repo: repo,
	}, nil
}

// Close closes the ESI client
func (c *Client) Close() error {
	return c.esi.Close()
}

// ESIMarketOrder represents a market order from ESI API
type ESIMarketOrder struct {
	OrderID      int64     `json:"order_id"`
	TypeID       int       `json:"type_id"`
	LocationID   int64     `json:"location_id"`
	VolumeTotal  int       `json:"volume_total"`
	VolumeRemain int       `json:"volume_remain"`
	MinVolume    int       `json:"min_volume"`
	Price        float64   `json:"price"`
	IsBuyOrder   bool      `json:"is_buy_order"`
	Duration     int       `json:"duration"`
	Issued       time.Time `json:"issued"`
	Range        string    `json:"range"`
}

// FetchMarketOrders fetches market orders for a region and stores them in the database
func (c *Client) FetchMarketOrders(ctx context.Context, regionID int) error {
	endpoint := fmt.Sprintf("/v1/markets/%d/orders/", regionID)

	resp, err := c.esi.Get(ctx, endpoint)
	if err != nil {
		return fmt.Errorf("ESI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 304 {
		// Not Modified - Cache is still valid
		return nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected ESI status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var esiOrders []ESIMarketOrder
	if err := json.Unmarshal(body, &esiOrders); err != nil {
		return fmt.Errorf("failed to parse ESI response: %w", err)
	}

	// Convert to database models
	fetchedAt := time.Now()
	dbOrders := make([]database.MarketOrder, 0, len(esiOrders))
	for _, esiOrder := range esiOrders {
		var minVolume *int
		if esiOrder.MinVolume > 0 {
			minVolume = &esiOrder.MinVolume
		}

		dbOrders = append(dbOrders, database.MarketOrder{
			OrderID:      esiOrder.OrderID,
			TypeID:       esiOrder.TypeID,
			RegionID:     regionID,
			LocationID:   esiOrder.LocationID,
			IsBuyOrder:   esiOrder.IsBuyOrder,
			Price:        esiOrder.Price,
			VolumeTotal:  esiOrder.VolumeTotal,
			VolumeRemain: esiOrder.VolumeRemain,
			MinVolume:    minVolume,
			Issued:       esiOrder.Issued,
			Duration:     esiOrder.Duration,
			FetchedAt:    fetchedAt,
		})
	}

	// Store in database
	if err := c.repo.UpsertMarketOrders(ctx, dbOrders); err != nil {
		return fmt.Errorf("failed to store market orders: %w", err)
	}

	return nil
}

// GetMarketOrders retrieves market orders from database (cached)
func (c *Client) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
	return c.repo.GetMarketOrders(ctx, regionID, typeID)
}
