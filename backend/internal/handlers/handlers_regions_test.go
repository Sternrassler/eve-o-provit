package handlers

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

// MockRegionQuerier implements database.RegionQuerier for testing
type MockRegionQuerier struct {
	GetAllRegionsFunc func(ctx context.Context) ([]database.RegionData, error)
}

func (m *MockRegionQuerier) GetAllRegions(ctx context.Context) ([]database.RegionData, error) {
	if m.GetAllRegionsFunc != nil {
		return m.GetAllRegionsFunc(ctx)
	}
	return nil, errors.New("GetAllRegionsFunc not implemented")
}

// TestGetRegions_Success_Unit tests successful regions retrieval
func TestGetRegions_Success_Unit(t *testing.T) {
	// Arrange
	mockRegionQuerier := &MockRegionQuerier{
		GetAllRegionsFunc: func(ctx context.Context) ([]database.RegionData, error) {
			return []database.RegionData{
				{ID: 10000002, Name: "The Forge"},
				{ID: 10000043, Name: "Domain"},
				{ID: 10000032, Name: "Sinq Laison"},
			}, nil
		},
	}

	handler := &Handler{
		regionQuerier: mockRegionQuerier,
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.RegionsResponse
	parseJSON(resp.Body, &result)

	assert.Equal(t, 3, result.Count)
	assert.Len(t, result.Regions, 3)
	assert.Equal(t, int64(10000002), result.Regions[0].ID)
	assert.Equal(t, "The Forge", result.Regions[0].Name)
	assert.Equal(t, int64(10000043), result.Regions[1].ID)
	assert.Equal(t, "Domain", result.Regions[1].Name)
}

// TestGetRegions_EmptyResult_Unit tests handler with no regions
func TestGetRegions_EmptyResult_Unit(t *testing.T) {
	// Arrange
	mockRegionQuerier := &MockRegionQuerier{
		GetAllRegionsFunc: func(ctx context.Context) ([]database.RegionData, error) {
			return []database.RegionData{}, nil
		},
	}

	handler := &Handler{
		regionQuerier: mockRegionQuerier,
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.RegionsResponse
	parseJSON(resp.Body, &result)

	assert.Equal(t, 0, result.Count)
	assert.Empty(t, result.Regions)
}

// TestGetRegions_QueryError_Unit tests database query failure
func TestGetRegions_QueryError_Unit(t *testing.T) {
	// Arrange
	mockRegionQuerier := &MockRegionQuerier{
		GetAllRegionsFunc: func(ctx context.Context) ([]database.RegionData, error) {
			return nil, errors.New("database connection lost")
		},
	}

	handler := &Handler{
		regionQuerier: mockRegionQuerier,
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	assert.Contains(t, body, "Failed to fetch regions")
	assert.Contains(t, body, "database connection lost")
}

// TestGetRegions_NilQuerier_Unit tests handler with nil region querier
func TestGetRegions_NilQuerier_Unit(t *testing.T) {
	// Arrange
	handler := &Handler{
		regionQuerier: nil, // Simulate missing dependency
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

// TestGetRegions_SingleRegion_Unit tests single region result
func TestGetRegions_SingleRegion_Unit(t *testing.T) {
	// Arrange
	mockRegionQuerier := &MockRegionQuerier{
		GetAllRegionsFunc: func(ctx context.Context) ([]database.RegionData, error) {
			return []database.RegionData{
				{ID: 10000002, Name: "The Forge"},
			}, nil
		},
	}

	handler := &Handler{
		regionQuerier: mockRegionQuerier,
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.RegionsResponse
	parseJSON(resp.Body, &result)

	assert.Equal(t, 1, result.Count)
	assert.Len(t, result.Regions, 1)
	assert.Equal(t, int64(10000002), result.Regions[0].ID)
	assert.Equal(t, "The Forge", result.Regions[0].Name)
}

// TestGetRegions_LargeDataset_Unit tests handling of many regions
func TestGetRegions_LargeDataset_Unit(t *testing.T) {
	// Arrange
	largeDataset := make([]database.RegionData, 100)
	for i := 0; i < 100; i++ {
		largeDataset[i] = database.RegionData{
			ID:   int64(10000000 + i),
			Name: "Region " + string(rune(i)),
		}
	}

	mockRegionQuerier := &MockRegionQuerier{
		GetAllRegionsFunc: func(ctx context.Context) ([]database.RegionData, error) {
			return largeDataset, nil
		},
	}

	handler := &Handler{
		regionQuerier: mockRegionQuerier,
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.RegionsResponse
	parseJSON(resp.Body, &result)

	assert.Equal(t, 100, result.Count)
	assert.Len(t, result.Regions, 100)
}

// TestGetRegions_SpecialCharactersInNames_Unit tests region names with special characters
func TestGetRegions_SpecialCharactersInNames_Unit(t *testing.T) {
	// Arrange
	mockRegionQuerier := &MockRegionQuerier{
		GetAllRegionsFunc: func(ctx context.Context) ([]database.RegionData, error) {
			return []database.RegionData{
				{ID: 10000001, Name: "Derelik"},
				{ID: 10000002, Name: "The Forge"},
				{ID: 10000003, Name: "Vale of the Silent"},
			}, nil
		},
	}

	handler := &Handler{
		regionQuerier: mockRegionQuerier,
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var result models.RegionsResponse
	parseJSON(resp.Body, &result)

	assert.Equal(t, 3, result.Count)
	// Verify special characters are preserved
	assert.Contains(t, result.Regions[2].Name, "of the")
}

// TestGetRegions_ContextCancellation_Unit tests behavior when context is cancelled
func TestGetRegions_ContextCancellation_Unit(t *testing.T) {
	// Arrange
	mockRegionQuerier := &MockRegionQuerier{
		GetAllRegionsFunc: func(ctx context.Context) ([]database.RegionData, error) {
			return nil, context.Canceled
		},
	}

	handler := &Handler{
		regionQuerier: mockRegionQuerier,
	}

	app := fiber.New()
	app.Get("/regions", handler.GetRegions)

	// Act
	req := httptest.NewRequest("GET", "/regions", nil)
	resp, _ := app.Test(req)

	// Assert
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)

	bodyBytes, _ := io.ReadAll(resp.Body)
	body := string(bodyBytes)
	assert.Contains(t, body, "Failed to fetch regions")
}
