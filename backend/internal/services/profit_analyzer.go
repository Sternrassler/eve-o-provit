// Package services - Profit analysis for trading
package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/models"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
)

const (
	// MinCargoUtilization is the minimum cargo usage required (10%)
	MinCargoUtilization = 0.10
)

// ProfitAnalyzer handles profit calculation and item pair analysis
type ProfitAnalyzer struct {
	sdeDB      *sql.DB
	sdeQuerier database.SDEQuerier
}

// NewProfitAnalyzer creates a new profit analyzer
func NewProfitAnalyzer(sdeDB *sql.DB, sdeQuerier database.SDEQuerier) *ProfitAnalyzer {
	return &ProfitAnalyzer{
		sdeDB:      sdeDB,
		sdeQuerier: sdeQuerier,
	}
}

// FindProfitableItems analyzes market orders to find profitable trading opportunities
func (pa *ProfitAnalyzer) FindProfitableItems(ctx context.Context, orders []database.MarketOrder, cargoCapacity float64, systemIDResolver func(int64) int64) ([]models.ItemPair, error) {
	// Group orders by type_id
	ordersByType := make(map[int][]database.MarketOrder)
	for _, order := range orders {
		ordersByType[order.TypeID] = append(ordersByType[order.TypeID], order)
	}

	var profitableItems []models.ItemPair

	// Analyze each type
	for typeID, typeOrders := range ordersByType {
		// Find lowest sell price and highest buy price
		var lowestSell, highestBuy *database.MarketOrder

		for i := range typeOrders {
			order := &typeOrders[i]
			if order.IsBuyOrder {
				if highestBuy == nil || order.Price > highestBuy.Price {
					highestBuy = order
				}
			} else {
				if lowestSell == nil || order.Price < lowestSell.Price {
					lowestSell = order
				}
			}
		}

		// Skip if we don't have both buy and sell orders
		if lowestSell == nil || highestBuy == nil {
			continue
		}

		// Calculate spread (sell to buy orders at highestBuy.Price, buy from sell orders at lowestSell.Price)
		spread := ((highestBuy.Price - lowestSell.Price) / lowestSell.Price) * 100

		// Skip if spread is too low or negative
		if spread < MinSpreadPercent {
			continue
		}

		// Get item info
		itemInfo, err := pa.sdeQuerier.GetTypeInfo(ctx, typeID)
		if err != nil {
			log.Printf("Skipped typeID %d - GetTypeInfo failed: %v", typeID, err)
			continue
		}

		// Get item volume
		itemVol, err := cargo.GetItemVolume(pa.sdeDB, int64(typeID))
		if err != nil {
			log.Printf("Skipped typeID %d (%s) - GetItemVolume failed: %v", typeID, itemInfo.Name, err)
			continue
		}

		// In-memory volume filter: Skip items that are too large
		// Minimum threshold: item must fill at least 10% of cargo
		minQuantity := 1
		if itemVol.Volume > 0 {
			minQuantity = int(cargoCapacity * MinCargoUtilization / itemVol.Volume)
			if minQuantity < 1 {
				minQuantity = 1
			}
		}

		// Skip if item won't fit enough in cargo (reduces candidates by ~80%)
		if itemVol.Volume*float64(minQuantity) > cargoCapacity {
			continue
		}

		// Calculate available volume - limited by BOTH buy and sell side
		// We can only trade the minimum of what we can buy AND what we can sell
		buyAvailable := lowestSell.VolumeRemain  // How much we can buy
		sellAvailable := highestBuy.VolumeRemain // How much we can sell (demand)

		// Take the minimum - we're bottlenecked by the smaller side
		availableQuantity := buyAvailable
		if sellAvailable < buyAvailable {
			availableQuantity = sellAvailable
		}

		availableVolumeM3 := float64(availableQuantity) * itemVol.Volume

		// Resolve system IDs if resolver provided
		buySystemID := lowestSell.LocationID  // Default fallback
		sellSystemID := highestBuy.LocationID // Default fallback
		if systemIDResolver != nil {
			buySystemID = systemIDResolver(lowestSell.LocationID)
			sellSystemID = systemIDResolver(highestBuy.LocationID)
		}

		profitableItems = append(profitableItems, models.ItemPair{
			TypeID:            typeID,
			ItemName:          itemInfo.Name,
			ItemVolume:        itemVol.Volume,
			BuyStationID:      lowestSell.LocationID, // Buy from sell orders
			BuySystemID:       buySystemID,
			BuyPrice:          lowestSell.Price,
			SellStationID:     highestBuy.LocationID, // Sell to buy orders
			SellSystemID:      sellSystemID,
			SellPrice:         highestBuy.Price,
			SpreadPercent:     spread,
			AvailableVolumeM3: availableVolumeM3,
			AvailableQuantity: availableQuantity,
		})
	}

	return profitableItems, nil
}

// CalculateProfitPerTour calculates profit for a single round trip
func (pa *ProfitAnalyzer) CalculateProfitPerTour(buyPrice, sellPrice float64, quantity int) float64 {
	return (sellPrice - buyPrice) * float64(quantity)
}

// CalculateQuantityPerTour calculates how many units fit in one cargo load
func (pa *ProfitAnalyzer) CalculateQuantityPerTour(itemVolume, cargoCapacity float64) (int, error) {
	if itemVolume <= 0 {
		return 0, fmt.Errorf("invalid item volume: %f", itemVolume)
	}

	quantity := int(cargoCapacity / itemVolume)
	if quantity <= 0 {
		return 0, fmt.Errorf("item too large for cargo")
	}

	return quantity, nil
}

// CalculateNumberOfTours determines how many trips are needed/possible
func (pa *ProfitAnalyzer) CalculateNumberOfTours(availableQuantity, quantityPerTour int) int {
	if quantityPerTour <= 0 {
		return 0
	}

	tours := availableQuantity / quantityPerTour
	if tours > 10 {
		tours = 10 // Max 10 tours
	}
	if tours < 1 {
		tours = 1 // At least 1 tour
	}

	return tours
}
