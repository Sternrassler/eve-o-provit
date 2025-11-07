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

// GetRawClient returns the underlying ESI client for direct access
// Used by pagination.BatchFetcher to implement PageFetcher interface
func (c *Client) GetRawClient() *esiclient.Client {
	return c.esi
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

// FetchMarketOrders fetches ALL market order pages for a region and stores them in the database
// This implementation ensures complete data by fetching all pages sequentially
// Future enhancement: Use MarketOrderFetcher for parallel pagination with worker pools
func (c *Client) FetchMarketOrders(ctx context.Context, regionID int) error {
	fetchedAt := time.Now()
	allDBOrders := make([]database.MarketOrder, 0, 10000) // Pre-allocate for ~10k orders

	// Fetch pages sequentially until X-Pages header indicates we've reached the end
	page := 1
	for {
		endpoint := fmt.Sprintf("/v1/markets/%d/orders/?page=%d", regionID, page)

		resp, err := c.esi.Get(ctx, endpoint)
		if err != nil {
			return fmt.Errorf("ESI request failed for page %d: %w", page, err)
		}

		// Handle Not Modified (cache hit) - treat as end of pagination
		if resp.StatusCode == 304 {
			resp.Body.Close()
			break
		}

		// Check for errors
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("unexpected ESI status %d for page %d: %s", resp.StatusCode, page, string(body))
		}

		// Parse X-Pages header to determine total pages
		totalPages := 1
		if xPages := resp.Header.Get("X-Pages"); xPages != "" {
			if _, err := fmt.Sscanf(xPages, "%d", &totalPages); err != nil {
				resp.Body.Close()
				return fmt.Errorf("invalid X-Pages header '%s': %w", xPages, err)
			}
		}

		// Parse response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read response body for page %d: %w", page, err)
		}

		var esiOrders []ESIMarketOrder
		if err := json.Unmarshal(body, &esiOrders); err != nil {
			return fmt.Errorf("failed to parse ESI response for page %d: %w", page, err)
		}

		// Convert ESI orders to database models
		for _, esiOrder := range esiOrders {
			var minVolume *int
			if esiOrder.MinVolume > 0 {
				minVolume = &esiOrder.MinVolume
			}

			allDBOrders = append(allDBOrders, database.MarketOrder{
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

		// Check if we've fetched all pages
		if page >= totalPages {
			break
		}

		page++
	}

	// Store all orders in database (single batch operation)
	if err := c.repo.UpsertMarketOrders(ctx, allDBOrders); err != nil {
		return fmt.Errorf("failed to store %d market orders: %w", len(allDBOrders), err)
	}

	return nil
}

// FetchMarketOrdersPage fetches a single page of market orders from ESI
// This is an INTERNAL method used by MarketOrderFetcher for parallel pagination
// DO NOT call this directly - use FetchMarketOrders or MarketOrderFetcher.FetchAllPages instead
// Returns the orders, total page count (from X-Pages header), and any error
func (c *Client) FetchMarketOrdersPage(ctx context.Context, regionID, page int) ([]ESIMarketOrder, int, error) {
	endpoint := fmt.Sprintf("/v1/markets/%d/orders/?page=%d", regionID, page)

	resp, err := c.esi.Get(ctx, endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("ESI request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle Not Modified (cache hit)
	if resp.StatusCode == 304 {
		return nil, 0, fmt.Errorf("304 Not Modified - use cached data")
	}

	// Check for errors
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("unexpected ESI status %d: %s", resp.StatusCode, string(body))
	}

	// Parse X-Pages header to get total page count
	totalPages := 1
	if xPages := resp.Header.Get("X-Pages"); xPages != "" {
		if _, err := fmt.Sscanf(xPages, "%d", &totalPages); err != nil {
			return nil, 0, fmt.Errorf("invalid X-Pages header '%s': %w", xPages, err)
		}
	}

	// Parse response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var esiOrders []ESIMarketOrder
	if err := json.Unmarshal(body, &esiOrders); err != nil {
		return nil, 0, fmt.Errorf("failed to parse ESI response: %w", err)
	}

	return esiOrders, totalPages, nil
}

// GetMarketOrders retrieves market orders from database (cached)
func (c *Client) GetMarketOrders(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
	return c.repo.GetMarketOrders(ctx, regionID, typeID)
}

// ESIMarketHistory represents a single day's market history from ESI
type ESIMarketHistory struct {
	Average    float64 `json:"average"`
	Date       string  `json:"date"` // Format: "2015-05-01"
	Highest    float64 `json:"highest"`
	Lowest     float64 `json:"lowest"`
	OrderCount int64   `json:"order_count"`
	Volume     int64   `json:"volume"`
}

// FetchMarketHistory fetches market history from ESI for a specific type and region
// ESI Endpoint: GET /v1/markets/{region_id}/history/?type_id={type_id}
// Returns up to 13 months of historical data
func (c *Client) FetchMarketHistory(ctx context.Context, regionID, typeID int) ([]database.PriceHistory, error) {
	endpoint := fmt.Sprintf("/v1/markets/%d/history/?type_id=%d", regionID, typeID)

	resp, err := c.esi.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("ESI request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle Not Modified (cache hit) - return empty slice
	if resp.StatusCode == 304 {
		return []database.PriceHistory{}, nil
	}

	// Check for errors
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected ESI status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var esiHistory []ESIMarketHistory
	if err := json.Unmarshal(body, &esiHistory); err != nil {
		return nil, fmt.Errorf("failed to parse ESI response: %w", err)
	}

	// Convert to database models
	dbHistory := make([]database.PriceHistory, 0, len(esiHistory))
	for _, h := range esiHistory {
		// Parse date string
		date, err := time.Parse("2006-01-02", h.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date %s: %w", h.Date, err)
		}

		// Convert orderCount from int64 to int
		orderCount := int(h.OrderCount)

		dbHistory = append(dbHistory, database.PriceHistory{
			TypeID:     typeID,
			RegionID:   regionID,
			Date:       date,
			Highest:    &h.Highest,
			Lowest:     &h.Lowest,
			Average:    &h.Average,
			Volume:     &h.Volume,
			OrderCount: &orderCount,
		})
	}

	return dbHistory, nil
}
