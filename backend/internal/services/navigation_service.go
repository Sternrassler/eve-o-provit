// Package services - Navigation service for system/location resolution
package services

import (
	"context"

	"github.com/Sternrassler/eve-o-provit/backend/internal/database"
)

// NavigationService implements NavigationServicer using SDEQuerier
type NavigationService struct {
	sdeQuerier     database.SDEQuerier
	fittingService FittingServicer
}

// NewNavigationService creates a new navigation service
func NewNavigationService(sdeQuerier database.SDEQuerier, fittingService FittingServicer) *NavigationService {
	return &NavigationService{
		sdeQuerier:     sdeQuerier,
		fittingService: fittingService,
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

// GetEffectiveWarpSpeed calculates effective warp speed including fitting bonuses
// Formula: baseWarpSpeed × (1 + skillBonuses) × fittingMultiplier
// Skills: Navigation, Evasive Maneuvering, Spaceship Command (implementation pending)
// Fitting: Hyperspatial modules, Low-Friction nozzles (multiplicative %)
func (s *NavigationService) GetEffectiveWarpSpeed(
	ctx context.Context,
	characterID int,
	shipTypeID int,
	baseWarpSpeed float64,
	accessToken string,
) (float64, error) {
	// TODO: Add skill bonuses when SkillsService supports navigation skills
	// For now: base warp speed only (no skill bonuses)
	warpSpeedWithSkills := baseWarpSpeed
	
	// Get fitting bonuses (nil check for optional dependency)
	if s.fittingService == nil {
		// No fitting service available, return base warp speed
		return warpSpeedWithSkills, nil
	}
	
	fitting, err := s.fittingService.GetCharacterFitting(ctx, characterID, shipTypeID, accessToken)
	if err != nil {
		// Fitting data unavailable (not an error - ship might not be fitted)
		// Return base warp speed only
		return warpSpeedWithSkills, nil
	}
	
	// Apply fitting multiplier (e.g., 1.20 = +20% warp speed)
	effectiveWarpSpeed := warpSpeedWithSkills * fitting.Bonuses.WarpSpeedMultiplier
	
	return effectiveWarpSpeed, nil
}

// GetEffectiveInertia calculates effective inertia modifier including fitting bonuses
// Formula: baseInertia × (1 + skillBonuses) × fittingModifier
// Skills: Evasive Maneuvering, Spaceship Command (implementation pending)
// Fitting: Inertial Stabilizers, Nanofiber modules (multiplicative %)
// Lower values = better agility (faster align time)
func (s *NavigationService) GetEffectiveInertia(
	ctx context.Context,
	characterID int,
	shipTypeID int,
	baseInertia float64,
	accessToken string,
) (float64, error) {
	// TODO: Add skill bonuses when SkillsService supports navigation skills
	// For now: base inertia only (no skill bonuses)
	inertiaWithSkills := baseInertia
	
	// Get fitting bonuses (nil check for optional dependency)
	if s.fittingService == nil {
		// No fitting service available, return base inertia
		return inertiaWithSkills, nil
	}
	
	fitting, err := s.fittingService.GetCharacterFitting(ctx, characterID, shipTypeID, accessToken)
	if err != nil {
		// Fitting data unavailable (not an error - ship might not be fitted)
		// Return base inertia only
		return inertiaWithSkills, nil
	}
	
	// Apply fitting modifier (e.g., 0.87 = -13% inertia = better agility)
	effectiveInertia := inertiaWithSkills * fitting.Bonuses.InertiaModifier
	
	return effectiveInertia, nil
}
