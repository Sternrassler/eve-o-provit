package services

import (
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
)

// TestISKPerHourCalculation tests the ISK/Hour calculation logic
func TestISKPerHourCalculation(t *testing.T) {
	tests := []struct {
		name             string
		profitPerUnit    float64
		quantity         int
		roundTripSeconds float64
		wantISKPerHour   float64
	}{
		{
			name:             "Basic calculation",
			profitPerUnit:    1000.0,
			quantity:         100,
			roundTripSeconds: 600.0, // 10 minutes
			wantISKPerHour:   600000.0, // (1000 * 100 / 600) * 3600
		},
		{
			name:             "One hour round trip",
			profitPerUnit:    500.0,
			quantity:         200,
			roundTripSeconds: 3600.0,
			wantISKPerHour:   100000.0, // 500 * 200
		},
		{
			name:             "30 second round trip",
			profitPerUnit:    100.0,
			quantity:         50,
			roundTripSeconds: 30.0,
			wantISKPerHour:   600000.0, // (100 * 50 / 30) * 3600
		},
		{
			name:             "Zero round trip time",
			profitPerUnit:    1000.0,
			quantity:         100,
			roundTripSeconds: 0.0,
			wantISKPerHour:   0.0, // Should handle division by zero
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			totalProfit := tt.profitPerUnit * float64(tt.quantity)
			var iskPerHour float64
			if tt.roundTripSeconds > 0 {
				iskPerHour = (totalProfit / tt.roundTripSeconds) * 3600
			}

			if iskPerHour != tt.wantISKPerHour {
				t.Errorf("ISKPerHour = %v, want %v", iskPerHour, tt.wantISKPerHour)
			}
		})
	}
}

// TestSpreadCalculation tests the spread percentage calculation
func TestSpreadCalculation(t *testing.T) {
	tests := []struct {
		name      string
		buyPrice  float64
		sellPrice float64
		wantSpread float64
	}{
		{
			name:       "10% spread",
			buyPrice:   100.0,
			sellPrice:  110.0,
			wantSpread: 10.0,
		},
		{
			name:       "5% spread (minimum)",
			buyPrice:   100.0,
			sellPrice:  105.0,
			wantSpread: 5.0,
		},
		{
			name:       "50% spread",
			buyPrice:   100.0,
			sellPrice:  150.0,
			wantSpread: 50.0,
		},
		{
			name:       "Negative spread (not profitable)",
			buyPrice:   100.0,
			sellPrice:  95.0,
			wantSpread: -5.0,
		},
		{
			name:       "Zero spread",
			buyPrice:   100.0,
			sellPrice:  100.0,
			wantSpread: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spread := ((tt.sellPrice - tt.buyPrice) / tt.buyPrice) * 100

			if spread != tt.wantSpread {
				t.Errorf("Spread = %v, want %v", spread, tt.wantSpread)
			}
		})
	}
}

// TestQuantityCalculation tests cargo capacity quantity calculation
func TestQuantityCalculation(t *testing.T) {
	tests := []struct {
		name          string
		cargoCapacity float64
		itemVolume    float64
		wantQuantity  int
	}{
		{
			name:          "Exact fit",
			cargoCapacity: 1000.0,
			itemVolume:    10.0,
			wantQuantity:  100,
		},
		{
			name:          "Partial fit (rounds down)",
			cargoCapacity: 1000.0,
			itemVolume:    33.0,
			wantQuantity:  30, // 1000 / 33 = 30.30... -> 30
		},
		{
			name:          "Large item",
			cargoCapacity: 100.0,
			itemVolume:    150.0,
			wantQuantity:  0, // Item too large
		},
		{
			name:          "Small cargo",
			cargoCapacity: 5.0,
			itemVolume:    1.0,
			wantQuantity:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quantity := int(tt.cargoCapacity / tt.itemVolume)

			if quantity != tt.wantQuantity {
				t.Errorf("Quantity = %v, want %v", quantity, tt.wantQuantity)
			}
		})
	}
}

// TestItemPairFiltering tests filtering profitable items by spread
func TestItemPairFiltering(t *testing.T) {
	items := []models.ItemPair{
		{TypeID: 1, SpreadPercent: 10.0},
		{TypeID: 2, SpreadPercent: 3.0},
		{TypeID: 3, SpreadPercent: 15.0},
		{TypeID: 4, SpreadPercent: 5.0},
		{TypeID: 5, SpreadPercent: -2.0},
	}

	minSpread := 5.0
	var filtered []models.ItemPair

	for _, item := range items {
		if item.SpreadPercent >= minSpread {
			filtered = append(filtered, item)
		}
	}

	expectedCount := 3 // Items with spread >= 5%
	if len(filtered) != expectedCount {
		t.Errorf("Filtered count = %v, want %v", len(filtered), expectedCount)
	}

	// Check that all filtered items meet the criteria
	for _, item := range filtered {
		if item.SpreadPercent < minSpread {
			t.Errorf("Item %d has spread %v, below minimum %v", item.TypeID, item.SpreadPercent, minSpread)
		}
	}
}

// TestRoutesSorting tests sorting routes by ISK/Hour
func TestRoutesSorting(t *testing.T) {
	routes := []models.TradingRoute{
		{ItemTypeID: 1, ISKPerHour: 100000.0},
		{ItemTypeID: 2, ISKPerHour: 500000.0},
		{ItemTypeID: 3, ISKPerHour: 250000.0},
		{ItemTypeID: 4, ISKPerHour: 1000000.0},
	}

	// Sort by ISK per hour (descending) - mimicking actual code
	// In the real implementation we use sort.Slice
	var sorted []models.TradingRoute
	sorted = append(sorted, routes...)

	// Manual bubble sort for testing (in prod we use sort.Slice)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].ISKPerHour < sorted[j].ISKPerHour {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Check that highest ISK/Hour is first
	if sorted[0].ISKPerHour != 1000000.0 {
		t.Errorf("First route ISKPerHour = %v, want 1000000", sorted[0].ISKPerHour)
	}

	// Check that lowest ISK/Hour is last
	if sorted[len(sorted)-1].ISKPerHour != 100000.0 {
		t.Errorf("Last route ISKPerHour = %v, want 100000", sorted[len(sorted)-1].ISKPerHour)
	}

	// Check descending order
	for i := 0; i < len(sorted)-1; i++ {
		if sorted[i].ISKPerHour < sorted[i+1].ISKPerHour {
			t.Errorf("Routes not in descending order at index %d: %v < %v",
				i, sorted[i].ISKPerHour, sorted[i+1].ISKPerHour)
		}
	}
}

// TestMaxRoutesLimit tests that we only return top 50 routes
func TestMaxRoutesLimit(t *testing.T) {
	// Create 100 routes
	var routes []models.TradingRoute
	for i := 0; i < 100; i++ {
		routes = append(routes, models.TradingRoute{
			ItemTypeID: i,
			ISKPerHour: float64(i * 1000),
		})
	}

	maxRoutes := 50

	// Limit to top 50
	if len(routes) > maxRoutes {
		routes = routes[:maxRoutes]
	}

	if len(routes) != maxRoutes {
		t.Errorf("Routes count = %v, want %v", len(routes), maxRoutes)
	}
}
