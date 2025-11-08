package services

import (
	"context"
	"errors"
	"testing"
)

// TestGetEffectiveWarpSpeed_NoFitting tests warp speed with no fitting data
func TestGetEffectiveWarpSpeed_NoFitting(t *testing.T) {
	mockFitting := &mockFittingService{
		err: errors.New("no fitting data"),
	}
	
	service := NewNavigationService(nil, mockFitting)
	
	ctx := context.Background()
	warpSpeed, err := service.GetEffectiveWarpSpeed(ctx, 12345, 20183, 3.0, "token")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Base 3.0 AU/s (no fitting bonuses)
	expected := 3.0
	if warpSpeed != expected {
		t.Errorf("Expected %.2f AU/s, got %.2f AU/s", expected, warpSpeed)
	}
}

// TestGetEffectiveWarpSpeed_WithFitting tests warp speed with fitting bonuses
func TestGetEffectiveWarpSpeed_WithFitting(t *testing.T) {
	mockFitting := &mockFittingService{
		fitting: &FittingData{
			Bonuses: FittingBonuses{
				WarpSpeedMultiplier: 1.488, // 3x Hyperspatial Velocity Optimizer I
			},
		},
	}
	
	service := NewNavigationService(nil, mockFitting)
	
	ctx := context.Background()
	warpSpeed, err := service.GetEffectiveWarpSpeed(ctx, 12345, 20183, 3.0, "token")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Base 3.0 × 1.488 = 4.464 AU/s
	expected := 4.464
	if warpSpeed != expected {
		t.Errorf("Expected %.3f AU/s, got %.3f AU/s", expected, warpSpeed)
	}
}

// TestGetEffectiveWarpSpeed_NilFittingService tests graceful degradation
func TestGetEffectiveWarpSpeed_NilFittingService(t *testing.T) {
	service := NewNavigationService(nil, nil)
	
	ctx := context.Background()
	warpSpeed, err := service.GetEffectiveWarpSpeed(ctx, 12345, 20183, 4.5, "token")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Base 4.5 AU/s (no fitting service)
	expected := 4.5
	if warpSpeed != expected {
		t.Errorf("Expected %.2f AU/s, got %.2f AU/s", expected, warpSpeed)
	}
}

// TestGetEffectiveInertia_NoFitting tests inertia with no fitting data
func TestGetEffectiveInertia_NoFitting(t *testing.T) {
	mockFitting := &mockFittingService{
		err: errors.New("no fitting data"),
	}
	
	service := NewNavigationService(nil, mockFitting)
	
	ctx := context.Background()
	inertia, err := service.GetEffectiveInertia(ctx, 12345, 20183, 0.5, "token")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Base 0.5 (no fitting bonuses)
	expected := 0.5
	if inertia != expected {
		t.Errorf("Expected %.3f, got %.3f", expected, inertia)
	}
}

// TestGetEffectiveInertia_WithFitting tests inertia with fitting bonuses
func TestGetEffectiveInertia_WithFitting(t *testing.T) {
	mockFitting := &mockFittingService{
		fitting: &FittingData{
			Bonuses: FittingBonuses{
				InertiaModifier: 0.7566, // 2x Inertial Stabilizers II
			},
		},
	}
	
	service := NewNavigationService(nil, mockFitting)
	
	ctx := context.Background()
	inertia, err := service.GetEffectiveInertia(ctx, 12345, 20183, 0.5, "token")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Base 0.5 × 0.7566 = 0.3783 (better agility)
	expected := 0.3783
	if inertia != expected {
		t.Errorf("Expected %.4f, got %.4f", expected, inertia)
	}
}

// TestGetEffectiveInertia_NilFittingService tests graceful degradation
func TestGetEffectiveInertia_NilFittingService(t *testing.T) {
	service := NewNavigationService(nil, nil)
	
	ctx := context.Background()
	inertia, err := service.GetEffectiveInertia(ctx, 12345, 20183, 1.2, "token")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Base 1.2 (no fitting service)
	expected := 1.2
	if inertia != expected {
		t.Errorf("Expected %.2f, got %.2f", expected, inertia)
	}
}

// TestGetEffectiveInertia_OverdriveInjectorPenalty tests cargo expansion penalty
func TestGetEffectiveInertia_OverdriveInjectorPenalty(t *testing.T) {
	mockFitting := &mockFittingService{
		fitting: &FittingData{
			Bonuses: FittingBonuses{
				CargoBonus:      -500.0, // Penalty (not used in this test)
				InertiaModifier: 1.10,   // +10% inertia = worse agility
			},
		},
	}
	
	service := NewNavigationService(nil, mockFitting)
	
	ctx := context.Background()
	inertia, err := service.GetEffectiveInertia(ctx, 12345, 20183, 0.8, "token")
	
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	
	// Base 0.8 × 1.10 = 0.88 (worse agility due to cargo expansion)
	expected := 0.88
	delta := 0.001 // Allow small floating point error
	if inertia < expected-delta || inertia > expected+delta {
		t.Errorf("Expected %.2f (±%.3f), got %.2f", expected, delta, inertia)
	}
}
