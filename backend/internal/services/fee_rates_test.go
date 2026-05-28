package services

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestSalesTaxRate(t *testing.T) {
	cases := map[int]float64{
		0: 0.05,
		1: 0.045,
		5: 0.025,
		6: 0.025, // clamped at -50%
	}
	for level, want := range cases {
		if got := SalesTaxRate(level); !almostEqual(got, want) {
			t.Errorf("SalesTaxRate(%d) = %v, want %v", level, got, want)
		}
	}
}

func TestBrokerFeeRate(t *testing.T) {
	// Base 3%, no skills/standings.
	if got := BrokerFeeRate(0, 0, 0, 0); !almostEqual(got, 0.03) {
		t.Errorf("base broker fee = %v, want 0.03", got)
	}
	// Broker Relations V: 3% - 5*0.3% = 1.5%.
	if got := BrokerFeeRate(5, 0, 0, 0); !almostEqual(got, 0.015) {
		t.Errorf("broker V = %v, want 0.015", got)
	}
	// Advanced Broker Relations lowers the floor; with high standings rate cannot go below floor.
	got := BrokerFeeRate(5, 5, 0, 0)
	if got < 0 || got > 0.03 {
		t.Errorf("broker fee out of range: %v", got)
	}
	// Standings reduce the rate further but never below 0.
	if r := BrokerFeeRate(5, 5, 10, 10); r < 0 {
		t.Errorf("broker fee went negative: %v", r)
	}
}
