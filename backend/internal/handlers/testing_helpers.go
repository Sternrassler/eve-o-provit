// Package handlers - Test helper functions
package handlers

import (
	"context"
	"encoding/json"
	"io"

	"github.com/Sternrassler/eve-o-provit/backend/internal/services"
)

// parseJSON is a test helper to parse JSON response body
func parseJSON(body io.Reader, dest interface{}) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// MockShipService is a mock implementation of ShipServicer for testing
type MockShipService struct{}

func (m *MockShipService) GetShipCapacities(ctx context.Context, shipTypeID int64) (*services.ShipCapacities, error) {
	return &services.ShipCapacities{
		ShipTypeID:    shipTypeID,
		BaseCargoHold: 1000.0,
	}, nil
}

// MockSystemService is a mock implementation of SystemServicer for testing
type MockSystemService struct{}

func (m *MockSystemService) GetSystemInfo(ctx context.Context, systemID int64) (*services.SystemInfo, error) {
	return &services.SystemInfo{
		SystemName: "Test System",
		RegionID:   10000002,
		RegionName: "Test Region",
	}, nil
}

func (m *MockSystemService) GetStationName(ctx context.Context, stationID int64) (string, error) {
	return "Test Station", nil
}

// newTestTradingHandler creates a TradingHandler for testing with minimal dependencies
func newTestTradingHandler() *TradingHandler {
	return &TradingHandler{
		sdeQuerier:    &MockSDEQuerier{},
		shipService:   &MockShipService{},
		systemService: &MockSystemService{},
	}
}
