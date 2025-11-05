// Package services - Navigation service for system/location resolution
package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
)

// NavigationService implements NavigationServicer using SDEQuerier
type NavigationService struct {
	sdeQuerier database.SDEQuerier
}

// NewNavigationService creates a new navigation service
func NewNavigationService(sdeQuerier database.SDEQuerier) *NavigationService {
	return &NavigationService{
		sdeQuerier: sdeQuerier,
	}
}

// GetSystemIDForLocation delegates to SDEQuerier
func (s *NavigationService) GetSystemIDForLocation(ctx context.Context, locationID int64) (int64, error) {
	return s.sdeQuerier.GetSystemIDForLocation(ctx, locationID)
}

// GetRegionIDForSystem delegates to SDEQuerier
func (s *NavigationService) GetRegionIDForSystem(ctx context.Context, systemID int64) (int, error) {
	return s.sdeQuerier.GetRegionIDForSystem(ctx, systemID)
}
