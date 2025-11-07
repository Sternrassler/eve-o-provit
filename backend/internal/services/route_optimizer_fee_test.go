package services

import (
	"testing"
)

// TestFeeCalculation_WorstCase tests fee calculation with worst-case skills (all = 0)
func TestFeeCalculation_WorstCase(t *testing.T) {
	tests := []struct {
		name              string
		buyValue          float64
		sellValue         float64
		wantBuyBrokerFee  float64
		wantSellBrokerFee float64
		wantSalesTax      float64
		wantTotalFees     float64
	}{
		{
			name:              "100k ISK trade",
			buyValue:          100000.0,
			sellValue:         100000.0,
			wantBuyBrokerFee:  3000.0,  // 3% of 100k
			wantSellBrokerFee: 3000.0,  // 3% of 100k
			wantSalesTax:      5000.0,  // 5% of 100k
			wantTotalFees:     11000.0, // 3k + 3k + 5k
		},
		{
			name:              "1M ISK trade",
			buyValue:          1000000.0,
			sellValue:         1000000.0,
			wantBuyBrokerFee:  30000.0,  // 3% of 1M
			wantSellBrokerFee: 30000.0,  // 3% of 1M
			wantSalesTax:      50000.0,  // 5% of 1M
			wantTotalFees:     110000.0, // 30k + 30k + 50k
		},
		{
			name:              "10M ISK trade",
			buyValue:          10000000.0,
			sellValue:         10000000.0,
			wantBuyBrokerFee:  300000.0,  // 3% of 10M
			wantSellBrokerFee: 300000.0,  // 3% of 10M
			wantSalesTax:      500000.0,  // 5% of 10M
			wantTotalFees:     1100000.0, // 300k + 300k + 500k = 1.1M (11% total)
		},
		{
			name:              "Small trade (minimum fees apply)",
			buyValue:          1000.0,
			sellValue:         1000.0,
			wantBuyBrokerFee:  100.0, // Minimum 100 ISK
			wantSellBrokerFee: 100.0, // Minimum 100 ISK
			wantSalesTax:      100.0, // Minimum 100 ISK
			wantTotalFees:     300.0, // 100 + 100 + 100
		},
		{
			name:              "Different buy/sell values",
			buyValue:          500000.0,
			sellValue:         750000.0,
			wantBuyBrokerFee:  15000.0, // 3% of 500k
			wantSellBrokerFee: 22500.0, // 3% of 750k
			wantSalesTax:      37500.0, // 5% of 750k
			wantTotalFees:     75000.0, // 15k + 22.5k + 37.5k
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal FeeService for testing
			feeService := &FeeService{}

			// Calculate fees with worst-case skills
			buyBrokerFee := feeService.CalculateBrokerFee(0, 0, 0, 0, tt.buyValue)
			sellBrokerFee := feeService.CalculateBrokerFee(0, 0, 0, 0, tt.sellValue)
			salesTax := feeService.CalculateSalesTax(0, tt.sellValue)
			totalFees := buyBrokerFee + sellBrokerFee + salesTax

			// Verify each fee component
			if buyBrokerFee != tt.wantBuyBrokerFee {
				t.Errorf("buyBrokerFee = %.2f, want %.2f", buyBrokerFee, tt.wantBuyBrokerFee)
			}
			if sellBrokerFee != tt.wantSellBrokerFee {
				t.Errorf("sellBrokerFee = %.2f, want %.2f", sellBrokerFee, tt.wantSellBrokerFee)
			}
			if salesTax != tt.wantSalesTax {
				t.Errorf("salesTax = %.2f, want %.2f", salesTax, tt.wantSalesTax)
			}
			if totalFees != tt.wantTotalFees {
				t.Errorf("totalFees = %.2f, want %.2f", totalFees, tt.wantTotalFees)
			}
		})
	}
}

// TestNetProfit_WithFees tests that net profit correctly subtracts fees
func TestNetProfit_WithFees(t *testing.T) {
	tests := []struct {
		name          string
		quantity      int
		buyPrice      float64
		sellPrice     float64
		wantGross     float64
		wantNetProfit float64
	}{
		{
			name:          "100 units @ 100 ISK profit/unit",
			quantity:      100,
			buyPrice:      1000.0,
			sellPrice:     1100.0,
			wantGross:     10000.0, // 100 * (1100 - 1000)
			wantNetProfit: -1800.0, // 10k gross - 11.8k fees
		},
		{
			name:          "1000 units @ 1000 ISK profit/unit",
			quantity:      1000,
			buyPrice:      10000.0,
			sellPrice:     11000.0,
			wantGross:     1000000.0,  // 1000 * 1000
			wantNetProfit: -180000.0,  // 1M gross - 1.18M fees
		},
		{
			name:          "High margin trade - 100% markup",
			quantity:      100,
			buyPrice:      10000.0,
			sellPrice:     20000.0,
			wantGross:     1000000.0, // 100 * 10000
			wantNetProfit: 810000.0,  // 1M - 190k fees
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate gross profit
			grossProfit := tt.wantGross

			// Calculate order values
			buyValue := tt.buyPrice * float64(tt.quantity)
			sellValue := tt.sellPrice * float64(tt.quantity)

			// Calculate fees (worst-case: 3% + 3% + 5% = 11% total)
			feeService := &FeeService{}
			buyBrokerFee := feeService.CalculateBrokerFee(0, 0, 0, 0, buyValue)
			sellBrokerFee := feeService.CalculateBrokerFee(0, 0, 0, 0, sellValue)
			salesTax := feeService.CalculateSalesTax(0, sellValue)
			totalFees := buyBrokerFee + sellBrokerFee + salesTax

			// Calculate net profit
			netProfit := grossProfit - totalFees

			// Verify (allow small floating point error)
			if netProfit < tt.wantNetProfit-0.01 || netProfit > tt.wantNetProfit+0.01 {
				t.Errorf("netProfit = %.2f, want %.2f (gross=%.2f, fees=%.2f)",
					netProfit, tt.wantNetProfit, grossProfit, totalFees)
			}
		})
	}
}

// TestMinimumFees_Enforcement tests that minimum fees (100 ISK) are enforced
func TestMinimumFees_Enforcement(t *testing.T) {
	feeService := &FeeService{}

	tests := []struct {
		name    string
		value   float64
		wantFee float64
		feeType string
	}{
		{
			name:    "Broker fee - below minimum",
			value:   1000.0, // 3% = 30 ISK, but min is 100
			wantFee: 100.0,
			feeType: "broker",
		},
		{
			name:    "Sales tax - below minimum",
			value:   1000.0, // 5% = 50 ISK, but min is 100
			wantFee: 100.0,
			feeType: "sales",
		},
		{
			name:    "Broker fee - above minimum",
			value:   10000.0, // 3% = 300 ISK
			wantFee: 300.0,
			feeType: "broker",
		},
		{
			name:    "Sales tax - above minimum",
			value:   10000.0, // 5% = 500 ISK
			wantFee: 500.0,
			feeType: "sales",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fee float64
			if tt.feeType == "broker" {
				fee = feeService.CalculateBrokerFee(0, 0, 0, 0, tt.value)
			} else {
				fee = feeService.CalculateSalesTax(0, tt.value)
			}

			if fee != tt.wantFee {
				t.Errorf("%s fee = %.2f, want %.2f", tt.feeType, fee, tt.wantFee)
			}
		})
	}
}
