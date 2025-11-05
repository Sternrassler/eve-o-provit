// Package handlers - GetMarketOrders handler unit tests with mocks
package handlers

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
)

func TestGetMarketOrders_Success_WithMockService(t *testing.T) {
	// Setup
	app := fiber.New()

	mockMarketService := &MockMarketService{
		FetchAndStoreMarketOrdersFunc: func(ctx context.Context, regionID int) (int, error) {
			assert.Equal(t, 10000002, regionID)
			return 150, nil // 150 orders stored
		},
		GetMarketOrdersFunc: func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
			assert.Equal(t, 10000002, regionID)
			assert.Equal(t, 34, typeID)
			minVol := 1
			return []database.MarketOrder{
				{
					OrderID:      123456,
					TypeID:       34,
					RegionID:     10000002,
					LocationID:   60003760,
					VolumeRemain: 100,
					VolumeTotal:  100,
					Price:        5.50,
					IsBuyOrder:   false,
					Duration:     90,
					Issued:       time.Now(),
					MinVolume:    &minVol,
					FetchedAt:    time.Now(),
				},
				{
					OrderID:      123457,
					TypeID:       34,
					RegionID:     10000002,
					LocationID:   60003760,
					VolumeRemain: 50,
					VolumeTotal:  50,
					Price:        5.45,
					IsBuyOrder:   true,
					Duration:     90,
					Issued:       time.Now(),
					MinVolume:    &minVol,
					FetchedAt:    time.Now(),
				},
			}, nil
		},
	}

	h := &Handler{
		marketService: mockMarketService,
		esiClient:     nil, // Not used in this test
	}

	app.Get("/markets/:region/orders/:type", h.GetMarketOrders)

	// Execute
	req := httptest.NewRequest("GET", "/markets/10000002/orders/34", nil)
	resp, err := app.Test(req, -1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Verify response body contains expected data
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	assert.Contains(t, bodyStr, `"order_id":123456`)
	assert.Contains(t, bodyStr, `"price":5.5`)
	assert.Contains(t, bodyStr, `"is_buy_order":false`)
	assert.Contains(t, bodyStr, `"order_id":123457`)
}

func TestGetMarketOrders_MarketServiceError(t *testing.T) {
	// Setup
	app := fiber.New()

	mockMarketService := &MockMarketService{
		FetchAndStoreMarketOrdersFunc: func(ctx context.Context, regionID int) (int, error) {
			return 0, errors.New("ESI API rate limit exceeded")
		},
		GetMarketOrdersFunc: func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
			return nil, nil // Not reached due to FetchAndStore error
		},
	}

	h := &Handler{
		marketService: mockMarketService,
		esiClient:     nil,
	}

	app.Get("/markets/:region/orders/:type", h.GetMarketOrders)

	// Execute - with refresh=true to trigger FetchAndStoreMarketOrders
	req := httptest.NewRequest("GET", "/markets/10000002/orders/34?refresh=true", nil)
	resp, err := app.Test(req, -1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	assert.Contains(t, bodyStr, "Failed to fetch and store market data")
}

func TestGetMarketOrders_ESIClientError(t *testing.T) {
	// Setup
	app := fiber.New()

	mockMarketService := &MockMarketService{
		FetchAndStoreMarketOrdersFunc: func(ctx context.Context, regionID int) (int, error) {
			return 100, nil // Fetch succeeded
		},
		GetMarketOrdersFunc: func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
			return nil, errors.New("database connection error")
		},
	}

	h := &Handler{
		marketService: mockMarketService,
		esiClient:     nil,
	}

	app.Get("/markets/:region/orders/:type", h.GetMarketOrders)

	// Execute
	req := httptest.NewRequest("GET", "/markets/10000002/orders/34", nil)
	resp, err := app.Test(req, -1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 500, resp.StatusCode)

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	assert.Contains(t, bodyStr, "Failed to get market orders")
}

func TestGetMarketOrders_EmptyResult(t *testing.T) {
	// Setup
	app := fiber.New()

	mockMarketService := &MockMarketService{
		FetchAndStoreMarketOrdersFunc: func(ctx context.Context, regionID int) (int, error) {
			return 0, nil // No new orders
		},
		GetMarketOrdersFunc: func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
			return []database.MarketOrder{}, nil // Empty result
		},
	}

	h := &Handler{
		marketService: mockMarketService,
		esiClient:     nil,
	}

	app.Get("/markets/:region/orders/:type", h.GetMarketOrders)

	// Execute
	req := httptest.NewRequest("GET", "/markets/10000002/orders/999999", nil)
	resp, err := app.Test(req, -1)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	assert.Equal(t, "[]", bodyStr) // Empty JSON array
}

func TestGetMarketOrders_StatusCodes(t *testing.T) {
	tests := []struct {
		name               string
		url                string
		setupMock          func() *MockMarketService
		expectedStatusCode int
		expectedBodyPart   string
	}{
		{
			name: "200 OK - Valid request",
			url:  "/markets/10000002/orders/34",
			setupMock: func() *MockMarketService {
				return &MockMarketService{
					FetchAndStoreMarketOrdersFunc: func(ctx context.Context, regionID int) (int, error) {
						return 10, nil
					},
					GetMarketOrdersFunc: func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
						return []database.MarketOrder{{OrderID: 123}}, nil
					},
				}
			},
			expectedStatusCode: 200,
			expectedBodyPart:   `"order_id":123`,
		},
		{
			name: "400 Bad Request - Invalid region ID",
			url:  "/markets/invalid/orders/34",
			setupMock: func() *MockMarketService {
				return &MockMarketService{}
			},
				expectedStatusCode: 400,
				expectedBodyPart:   "invalid region ID",
		},
		{
			name: "400 Bad Request - Invalid type ID",
			url:  "/markets/10000002/orders/invalid",
			setupMock: func() *MockMarketService {
				return &MockMarketService{}
			},
				expectedStatusCode: 400,
				expectedBodyPart:   "invalid type ID",
		},
		{
			name: "500 Internal Server Error - Service error",
			url:  "/markets/10000002/orders/34?refresh=true",
			setupMock: func() *MockMarketService {
				return &MockMarketService{
					FetchAndStoreMarketOrdersFunc: func(ctx context.Context, regionID int) (int, error) {
						return 0, errors.New("internal error")
					},
					GetMarketOrdersFunc: func(ctx context.Context, regionID, typeID int) ([]database.MarketOrder, error) {
						return nil, nil // Not reached due to FetchAndStore error
					},
				}
			},
			expectedStatusCode: 500,
			expectedBodyPart:   "Failed to fetch and store market data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := fiber.New()
			h := &Handler{
				marketService: tt.setupMock(),
				esiClient:     nil,
			}
			app.Get("/markets/:region/orders/:type", h.GetMarketOrders)

			req := httptest.NewRequest("GET", tt.url, nil)
			resp, err := app.Test(req, -1)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)

			if tt.expectedBodyPart != "" {
				body := make([]byte, 2048)
				n, _ := resp.Body.Read(body)
				bodyStr := string(body[:n])
				assert.Contains(t, bodyStr, tt.expectedBodyPart)
			}
		})
	}
}
