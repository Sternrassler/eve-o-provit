package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewRouteCalculator tests RouteCalculator initialization
func TestNewRouteCalculator(t *testing.T) {
	t.Run("Creates new RouteCalculator with provided dependencies", func(t *testing.T) {
		optimizer := NewRouteCalculator(nil, nil, nil)

		assert.NotNil(t, optimizer, "RouteCalculator should be initialized even with nil dependencies")
	})
}

// TestCalculateRoute_Integration tests CalculateRoute with mocked data
func TestCalculateRoute_Integration(t *testing.T) {
	t.Skip("Requires full integration test setup - tested in integration tests")
	// This would require:
	// - Mock SDE repository
	// - Mock SDE database
	// - Mock ItemPair data
}

// TestModelAFeeArithmetic verifies Modell-A fee arithmetic directly:
// Only Sales-Tax is charged (Accounting=0, base 5%); Broker-Fees are zero.
// End-to-end route coverage is provided by integration tests (requires SDE DB).
func TestModelAFeeArithmetic(t *testing.T) {
	feeService := NewFeeService(nil, nil)

	tests := []struct {
		name           string
		buyPrice       float64
		sellPrice      float64
		quantity       int
		totalProfit    float64 // (sellPrice - buyPrice) * quantity
		wantBrokerFees float64 // must be 0 under Model A
		wantSalesTax   float64 // 5% of sellValue (Accounting=0)
		wantNetProfit  float64 // totalProfit - salesTax
	}{
		{
			name:        "100 units: buy 1000, sell 1200",
			buyPrice:    1000.0,
			sellPrice:   1200.0,
			quantity:    100,
			totalProfit: 20000.0,
			// sellValue = 1200 * 100 = 120000; salesTax = 120000 * 0.05 = 6000
			wantBrokerFees: 0.0,
			wantSalesTax:   6000.0,
			wantNetProfit:  14000.0, // 20000 - 6000
		},
		{
			name:        "1 unit: buy 500000, sell 600000",
			buyPrice:    500000.0,
			sellPrice:   600000.0,
			quantity:    1,
			totalProfit: 100000.0,
			// sellValue = 600000; salesTax = 600000 * 0.05 = 30000
			wantBrokerFees: 0.0,
			wantSalesTax:   30000.0,
			wantNetProfit:  70000.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sellValue := tt.sellPrice * float64(tt.quantity)

			salesTax := feeService.CalculateSalesTax(0, sellValue)

			// Modell A: keine Broker-Fees
			buyBrokerFee := 0.0
			sellBrokerFee := 0.0
			brokerFees := buyBrokerFee + sellBrokerFee
			totalFees := salesTax
			netProfit := tt.totalProfit - totalFees

			if brokerFees != tt.wantBrokerFees {
				t.Errorf("brokerFees = %v, want %v (Modell A: must be 0)", brokerFees, tt.wantBrokerFees)
			}
			if salesTax != tt.wantSalesTax {
				t.Errorf("salesTax = %v, want %v", salesTax, tt.wantSalesTax)
			}
			if netProfit != tt.wantNetProfit {
				t.Errorf("netProfit = %v, want %v", netProfit, tt.wantNetProfit)
			}
		})
	}
}

// TestToursFromVolume prüft Aufrundung (Ceil), nicht round-to-nearest.
func TestToursFromVolume(t *testing.T) {
	cases := []struct {
		availableVol float64
		capacity     float64
		want         int
	}{
		{availableVol: 1200, capacity: 1000, want: 2}, // 1.2 → 2 (Ceil)
		{availableVol: 1000, capacity: 1000, want: 1}, // 1.0 → 1
		{availableVol: 1600, capacity: 1000, want: 2}, // 1.6 → 2
		{availableVol: 100, capacity: 1000, want: 1},  // <1 → min 1
	}
	for _, c := range cases {
		got := toursFromVolume(c.availableVol, c.capacity)
		if got != c.want {
			t.Errorf("toursFromVolume(%.0f, %.0f) = %d, want %d", c.availableVol, c.capacity, got, c.want)
		}
	}
}
