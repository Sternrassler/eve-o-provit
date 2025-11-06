package handlers

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// ItemSearchRow matches the inline struct from SDEQuerier interface
type ItemSearchRow struct {
	TypeID    int
	Name      string
	GroupName string
}

// MockSDESearcher for SearchItems tests
type MockSDESearcher struct {
	SearchItemsFunc func(ctx context.Context, query string, limit int) ([]ItemSearchRow, error)
}

func (m *MockSDESearcher) GetTypeInfo(ctx context.Context, typeID int) (*database.TypeInfo, error) {
	return nil, nil
}

func (m *MockSDESearcher) SearchTypes(ctx context.Context, searchTerm string, limit int) ([]database.TypeInfo, error) {
	return nil, nil
}

func (m *MockSDESearcher) SearchItems(ctx context.Context, query string, limit int) ([]struct {
	TypeID    int
	Name      string
	GroupName string
}, error) {
	if m.SearchItemsFunc != nil {
		rows, err := m.SearchItemsFunc(ctx, query, limit)
		// Convert to inline struct
		result := make([]struct {
			TypeID    int
			Name      string
			GroupName string
		}, len(rows))
		for i, row := range rows {
			result[i].TypeID = row.TypeID
			result[i].Name = row.Name
			result[i].GroupName = row.GroupName
		}
		return result, err
	}
	return nil, nil
}

func (m *MockSDESearcher) GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error) {
	return 0, nil
}

func (m *MockSDESearcher) GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error) {
	return 0, nil
}

func (m *MockSDESearcher) GetSystemName(ctx context.Context, systemID int64) (string, error) {
	return "", nil
}

func (m *MockSDESearcher) GetStationName(ctx context.Context, stationID int64) (string, error) {
	return "", nil
}

func (m *MockSDESearcher) GetRegionName(ctx context.Context, regionID int) (string, error) {
	return "", nil
}

func (m *MockSDESearcher) GetSystemSecurityStatus(ctx context.Context, systemID int64) (float64, error) {
	return 0, nil
}

func TestSearchItems_Success_Unit(t *testing.T) {
	// Mock SDE querier
	mockSDE := &MockSDESearcher{
		SearchItemsFunc: func(ctx context.Context, query string, limit int) ([]ItemSearchRow, error) {
			return []ItemSearchRow{
				{TypeID: 34, Name: "Tritanium", GroupName: "Mineral"},
				{TypeID: 35, Name: "Pyerite", GroupName: "Mineral"},
			}, nil
		},
	}

	// Setup handler
	tradingHandler := &TradingHandler{
		sdeQuerier:    mockSDE,
		shipService:   &MockShipService{},
		systemService: &MockSystemService{},
	}

	// Create request
	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search?q=itan", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verify response
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "Tritanium")
	assert.Contains(t, string(body), "Pyerite")
	assert.Contains(t, string(body), `"count":2`)
}

func TestSearchItems_QueryTooShort_Unit(t *testing.T) {
	tradingHandler := &TradingHandler{}

	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search?q=ab", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "must be at least 3 characters")
}

func TestSearchItems_MissingQuery(t *testing.T) {
	tradingHandler := &TradingHandler{}

	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 400, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "must be at least 3 characters")
}

func TestSearchItems_WithCustomLimit(t *testing.T) {
	mockSDE := &MockSDESearcher{
		SearchItemsFunc: func(ctx context.Context, query string, limit int) ([]ItemSearchRow, error) {
			// Verify limit is passed correctly
			assert.Equal(t, 50, limit)
			return []ItemSearchRow{
				{TypeID: 34, Name: "Tritanium", GroupName: "Mineral"},
			}, nil
		},
	}

	// mockSDE is used in tradingHandler
	tradingHandler := &TradingHandler{
		sdeQuerier:    mockSDE,
		shipService:   &MockShipService{},
		systemService: &MockSystemService{},
	}

	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search?q=trit&limit=50", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchItems_LimitExceedsMax(t *testing.T) {
	mockSDE := &MockSDESearcher{
		SearchItemsFunc: func(ctx context.Context, query string, limit int) ([]ItemSearchRow, error) {
			// Verify limit is capped at 100
			assert.Equal(t, 20, limit) // Should use default when > 100
			return []ItemSearchRow{}, nil
		},
	}

	// mockSDE is used in tradingHandler
	tradingHandler := &TradingHandler{
		sdeQuerier:    mockSDE,
		shipService:   &MockShipService{},
		systemService: &MockSystemService{},
	}

	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search?q=trit&limit=200", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchItems_InvalidLimit(t *testing.T) {
	mockSDE := &MockSDESearcher{
		SearchItemsFunc: func(ctx context.Context, query string, limit int) ([]ItemSearchRow, error) {
			// Should use default limit (20) when invalid
			assert.Equal(t, 20, limit)
			return []ItemSearchRow{}, nil
		},
	}

	// mockSDE is used in tradingHandler
	tradingHandler := &TradingHandler{
		sdeQuerier:    mockSDE,
		shipService:   &MockShipService{},
		systemService: &MockSystemService{},
	}

	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search?q=trit&limit=invalid", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)
}

func TestSearchItems_SDEError_Unit(t *testing.T) {
	mockSDE := &MockSDESearcher{
		SearchItemsFunc: func(ctx context.Context, query string, limit int) ([]ItemSearchRow, error) {
			return nil, assert.AnError
		},
	}

	// mockSDE is used in tradingHandler
	tradingHandler := &TradingHandler{
		sdeQuerier:    mockSDE,
		shipService:   &MockShipService{},
		systemService: &MockSystemService{},
	}

	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search?q=trit", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 500, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "failed to search items")
}

func TestSearchItems_EmptyResults(t *testing.T) {
	mockSDE := &MockSDESearcher{
		SearchItemsFunc: func(ctx context.Context, query string, limit int) ([]ItemSearchRow, error) {
			return []ItemSearchRow{}, nil
		},
	}

	// mockSDE is used in tradingHandler
	tradingHandler := &TradingHandler{
		sdeQuerier:    mockSDE,
		shipService:   &MockShipService{},
		systemService: &MockSystemService{},
	}

	app := fiber.New()
	app.Get("/search", tradingHandler.SearchItems)

	req := httptest.NewRequest("GET", "/search?q=nonexistent", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"count":0`)
}
