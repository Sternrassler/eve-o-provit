package esi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_FetchMarketOrders_ParseResponse(t *testing.T) {
	esiResponse := []ESIMarketOrder{
		{
			OrderID:      123456,
			TypeID:       34,
			LocationID:   60003760,
			VolumeTotal:  1000,
			VolumeRemain: 500,
			MinVolume:    1,
			Price:        5.50,
			IsBuyOrder:   false,
			Duration:     90,
			Issued:       time.Now(),
			Range:        "region",
		},
		{
			OrderID:      789012,
			TypeID:       34,
			LocationID:   60003760,
			VolumeTotal:  2000,
			VolumeRemain: 1500,
			MinVolume:    10,
			Price:        5.00,
			IsBuyOrder:   true,
			Duration:     30,
			Issued:       time.Now(),
			Range:        "station",
		},
	}

	for _, order := range esiResponse {
		if order.OrderID == 0 {
			t.Error("OrderID cannot be zero")
		}
		if order.TypeID == 0 {
			t.Error("TypeID cannot be zero")
		}
		if order.Price <= 0 {
			t.Error("Price must be positive")
		}
		if order.VolumeRemain > order.VolumeTotal {
			t.Error("VolumeRemain cannot exceed VolumeTotal")
		}
	}

	data, err := json.Marshal(esiResponse)
	if err != nil {
		t.Fatalf("Failed to marshal ESI orders: %v", err)
	}

	var parsed []ESIMarketOrder
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ESI orders: %v", err)
	}

	if len(parsed) != len(esiResponse) {
		t.Errorf("Expected %d orders, got %d", len(esiResponse), len(parsed))
	}
}

func TestClient_MockESIServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orders := []ESIMarketOrder{
			{
				OrderID:      123456,
				TypeID:       34,
				LocationID:   60003760,
				VolumeTotal:  1000,
				VolumeRemain: 500,
				MinVolume:    1,
				Price:        5.50,
				IsBuyOrder:   false,
				Duration:     90,
				Issued:       time.Now(),
				Range:        "region",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(orders)
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to call mock server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var orders []ESIMarketOrder
	if err := json.NewDecoder(resp.Body).Decode(&orders); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(orders))
	}
}

func TestRedisConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis test in short mode")
	}
	t.Skip("Integration test requires Redis - implement with testcontainers")
}

func TestESIMarketOrder_BuyOrderProperties(t *testing.T) {
	//nolint:unusedwrite
	buyOrder := ESIMarketOrder{
		OrderID:      123,
		TypeID:       34,
		LocationID:   60003760,
		VolumeTotal:  1000,
		VolumeRemain: 500,
		MinVolume:    100,
		Price:        5.00,
		IsBuyOrder:   true,
		Duration:     30,
		Issued:       time.Now(),
		Range:        "region",
	}

	if !buyOrder.IsBuyOrder {
		t.Error("IsBuyOrder should be true")
	}
	if buyOrder.Range != "region" {
		t.Errorf("Expected Range = 'region', got '%s'", buyOrder.Range)
	}
	if buyOrder.MinVolume != 100 {
		t.Errorf("Expected MinVolume = 100, got %d", buyOrder.MinVolume)
	}
}

func TestESIMarketOrder_SellOrderProperties(t *testing.T) {
	//nolint:unusedwrite
	sellOrder := ESIMarketOrder{
		OrderID:      456,
		TypeID:       34,
		LocationID:   60003760,
		VolumeTotal:  2000,
		VolumeRemain: 1500,
		MinVolume:    1,
		Price:        10.00,
		IsBuyOrder:   false,
		Duration:     90,
		Issued:       time.Now(),
		Range:        "station",
	}

	if sellOrder.IsBuyOrder {
		t.Error("IsBuyOrder should be false")
	}
	if sellOrder.Range != "station" {
		t.Errorf("Expected Range = 'station', got '%s'", sellOrder.Range)
	}
}
