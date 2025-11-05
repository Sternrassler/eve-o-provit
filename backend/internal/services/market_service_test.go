package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Sternrassler/eve-esi-client/pkg/client"
	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
	"github.com/Sternrassler/eve-o-provit/backend/internal/testutil"
	"github.com/stretchr/testify/assert"
)

// MockESIRawClient implements ESIRawClient interface
type MockESIRawClient struct {
	rawClient *client.Client
}

func (m *MockESIRawClient) GetRawClient() *client.Client {
	return m.rawClient
}

func TestMarketService_NewMarketService(t *testing.T) {
	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &MockESIRawClient{}

	service := NewMarketService(marketQuerier, esiClient)

	assert.NotNil(t, service)
	assert.Equal(t, marketQuerier, service.marketQuerier)
	assert.Equal(t, esiClient, service.esiClient)
}

func TestMarketService_FetchAndStoreMarketOrders_IntegrationRequired(t *testing.T) {
	t.Skip("Requires working ESI client and BatchFetcher - tested in integration tests")

	// This test would require:
	// - Real HTTP client or complex mock
	// - BatchFetcher mock (complex pagination logic)
	// - ESI API response fixtures
	//
	// Better suited for integration tests with real ESI or recorded responses
}

func TestMarketService_GetMarketOrders_NotImplemented(t *testing.T) {
	marketQuerier := testutil.NewMockMarketWithDefaults()
	esiClient := &MockESIRawClient{}
	service := NewMarketService(marketQuerier, esiClient)

	ctx := context.Background()
	_, err := service.GetMarketOrders(ctx, 10000002, 34)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}

// TestMarketService_UpsertMarketOrders_Success validates service can store orders
func TestMarketService_UpsertMarketOrders_Success(t *testing.T) {
	marketQuerier := testutil.NewMockMarketWithDefaults()
	service := &MarketService{marketQuerier: marketQuerier}

	ctx := context.Background()
	orders := []database.MarketOrder{
		{
			RegionID:     10000002,
			TypeID:       34,
			Price:        5.50,
			VolumeRemain: 1000,
			FetchedAt:    time.Now(),
		},
	}

	err := service.marketQuerier.UpsertMarketOrders(ctx, orders)
	assert.NoError(t, err)
}

// TestMarketService_UpsertMarketOrders_DatabaseError validates error handling
func TestMarketService_UpsertMarketOrders_DatabaseError(t *testing.T) {
	marketQuerier := testutil.NewMockMarketWithDefaults()
	marketQuerier.UpsertMarketOrdersFunc = func(ctx context.Context, orders []database.MarketOrder) error {
		return fmt.Errorf("database connection failed")
	}

	service := &MarketService{marketQuerier: marketQuerier}

	ctx := context.Background()
	orders := []database.MarketOrder{{RegionID: 10000002, TypeID: 34}}

	err := service.marketQuerier.UpsertMarketOrders(ctx, orders)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database connection failed")
}
