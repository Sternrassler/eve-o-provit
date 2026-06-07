// Package esi provides EVE Swagger Interface (ESI) API client functionality
// Wrapper around github.com/Sternrassler/eve-esi-client
package esi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	esiclient "github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/redis/go-redis/v9"

	"github.com/Sternrassler/eve-o-provit/backend/internal/metrics"
)

// Config holds ESI client configuration
type Config struct {
	UserAgent      string
	RateLimit      int
	ErrorThreshold int
	MaxRetries     int
}

// Client wraps the ESI client with application-specific logic.
// It handles only HTTP communication with the ESI API — persistence is the
// responsibility of the service layer.
type Client struct {
	esi *esiclient.Client
}

// NewClient creates a new ESI client
func NewClient(redisClient *redis.Client, cfg Config) (*Client, error) {
	esiCfg := esiclient.DefaultConfig(redisClient, cfg.UserAgent)
	esiCfg.RateLimit = cfg.RateLimit
	esiCfg.ErrorThreshold = cfg.ErrorThreshold
	esiCfg.MaxRetries = cfg.MaxRetries
	esiCfg.RespectExpires = true // ESI Compliance (MUST)

	esiClient, err := esiclient.New(esiCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create ESI client: %w", err)
	}

	// Instrument the outbound HTTP transport with Prometheus metrics
	// (eveoprovit_esi_*). The http.Client mirrors the eve-esi-client default
	// (30s timeout) — the library only exposes a setter, not its default client.
	esiClient.SetHTTPClient(&http.Client{
		Timeout:   30 * time.Second,
		Transport: metrics.NewESITransport(nil),
	})

	return &Client{esi: esiClient}, nil
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

// FetchMarketOrdersPage fetches a single page of market orders from ESI.
// Use pagination.BatchFetcher (via GetRawClient) for full parallel pagination.
// Returns the orders, total page count (from X-Pages header), and any error.
func (c *Client) FetchMarketOrdersPage(ctx context.Context, regionID, page int) ([]ESIMarketOrder, int, error) {
	endpoint := fmt.Sprintf("/v1/markets/%d/orders/?page=%d", regionID, page)

	resp, err := c.esi.Get(ctx, endpoint)
	if err != nil {
		return nil, 0, fmt.Errorf("ESI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 304 {
		return nil, 0, fmt.Errorf("304 Not Modified - use cached data")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, 0, fmt.Errorf("unexpected ESI status %d: %s", resp.StatusCode, string(body))
	}

	totalPages := 1
	if xPages := resp.Header.Get("X-Pages"); xPages != "" {
		if _, err := fmt.Sscanf(xPages, "%d", &totalPages); err != nil {
			return nil, 0, fmt.Errorf("invalid X-Pages header '%s': %w", xPages, err)
		}
	}

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

// FetchMarketOrdersForType fetches all market orders (buy + sell) for a single type
// in a region via the lightweight per-type ESI endpoint. Much cheaper than a full
// region fetch — used by Multi-Hub Comparison (#43) and the competition collector.
// Follows X-Pages pagination (capped to avoid runaway loops on pathological items).
func (c *Client) FetchMarketOrdersForType(ctx context.Context, regionID, typeID int) ([]ESIMarketOrder, error) {
	const maxPages = 20
	var all []ESIMarketOrder
	totalPages := 1
	for page := 1; page <= totalPages && page <= maxPages; page++ {
		endpoint := fmt.Sprintf("/v1/markets/%d/orders/?order_type=all&type_id=%d&page=%d", regionID, typeID, page)
		resp, err := c.esi.Get(ctx, endpoint)
		if err != nil {
			return nil, fmt.Errorf("ESI request failed: %w", err)
		}
		if resp.StatusCode == 304 {
			resp.Body.Close()
			break
		}
		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("unexpected ESI status %d: %s", resp.StatusCode, string(body))
		}
		if page == 1 {
			if xPages := resp.Header.Get("X-Pages"); xPages != "" {
				_, _ = fmt.Sscanf(xPages, "%d", &totalPages)
			}
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		var orders []ESIMarketOrder
		if err := json.Unmarshal(body, &orders); err != nil {
			return nil, fmt.Errorf("failed to parse ESI response: %w", err)
		}
		all = append(all, orders...)
	}
	return all, nil
}

// ESIMarketHistory represents a single day's market history from ESI API
type ESIMarketHistory struct {
	Average    float64 `json:"average"`
	Date       string  `json:"date"` // Format: "2015-05-01"
	Highest    float64 `json:"highest"`
	Lowest     float64 `json:"lowest"`
	OrderCount int64   `json:"order_count"`
	Volume     int64   `json:"volume"`
}

// FetchMarketHistory fetches market history from ESI for a specific type and region.
// Returns up to 13 months of historical data as PriceHistoryEntry values.
// The service layer is responsible for mapping entries to database.PriceHistory.
func (c *Client) FetchMarketHistory(ctx context.Context, regionID, typeID int) ([]PriceHistoryEntry, error) {
	endpoint := fmt.Sprintf("/v1/markets/%d/history/?type_id=%d", regionID, typeID)

	resp, err := c.esi.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("ESI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 304 {
		return []PriceHistoryEntry{}, nil
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected ESI status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var esiHistory []ESIMarketHistory
	if err := json.Unmarshal(body, &esiHistory); err != nil {
		return nil, fmt.Errorf("failed to parse ESI response: %w", err)
	}

	entries := make([]PriceHistoryEntry, 0, len(esiHistory))
	for _, h := range esiHistory {
		date, err := time.Parse("2006-01-02", h.Date)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date %s: %w", h.Date, err)
		}
		orderCount := int(h.OrderCount)
		entries = append(entries, PriceHistoryEntry{
			Date:       date,
			Highest:    &h.Highest,
			Lowest:     &h.Lowest,
			Average:    &h.Average,
			Volume:     &h.Volume,
			OrderCount: &orderCount,
		})
	}

	return entries, nil
}
