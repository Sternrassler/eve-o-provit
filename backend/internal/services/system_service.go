// Package services - System Service for system/region/station information
package services

import (
	"context"
	"fmt"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
)

// SystemService provides system-related operations using SDE repository
type SystemService struct {
	sdeQuerier database.SDEQuerier
}

// NewSystemService creates a new system service
func NewSystemService(sdeQuerier database.SDEQuerier) *SystemService {
	return &SystemService{
		sdeQuerier: sdeQuerier,
	}
}

// GetSystemInfo retrieves combined system and region information
func (s *SystemService) GetSystemInfo(ctx context.Context, systemID int64) (*SystemInfo, error) {
	// Get system name
	systemName, err := s.sdeQuerier.GetSystemName(ctx, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get system name: %w", err)
	}

	// Get region ID
	regionID, err := s.sdeQuerier.GetRegionIDForSystem(ctx, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get region ID: %w", err)
	}

	// Get region name
	regionName, err := s.sdeQuerier.GetRegionName(ctx, regionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get region name: %w", err)
	}

	return &SystemInfo{
		SystemName: systemName,
		RegionID:   int64(regionID),
		RegionName: regionName,
	}, nil
}

// GetStationName retrieves station name by ID
func (s *SystemService) GetStationName(ctx context.Context, stationID int64) (string, error) {
	return s.sdeQuerier.GetStationName(ctx, stationID)
}
