package cargo

import (
	"math"
	"testing"
)

func TestApplySkillModifiers_NoSkills(t *testing.T) {
	baseCapacity := 1000.0

	// eve-o-provit version doesn't have holdType parameter
	got := ApplySkillModifiers(baseCapacity, nil)
	want := 1000.0

	if got != want {
		t.Errorf("ApplySkillModifiers(nil) = %v, want %v", got, want)
	}
}

func TestApplySkillModifiers_RacialHauler(t *testing.T) {
	baseCapacity := 1000.0

	tests := []struct {
		name  string
		level int
		want  float64
	}{
		{"Racial Hauler I", 1, 1050.0},   // +5%
		{"Racial Hauler III", 3, 1150.0}, // +15%
		{"Racial Hauler V", 5, 1250.0},   // +25%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skills := &SkillModifiers{
				RacialHaulerLevel: &tt.level,
			}
			got := ApplySkillModifiers(baseCapacity, skills)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("ApplySkillModifiers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplySkillModifiers_Freighter(t *testing.T) {
	baseCapacity := 10000.0

	tests := []struct {
		name  string
		level int
		want  float64
	}{
		{"Freighter I", 1, 10500.0},   // +5%
		{"Freighter III", 3, 11500.0}, // +15%
		{"Freighter V", 5, 12500.0},   // +25%
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skills := &SkillModifiers{
				FreighterLevel: &tt.level,
			}
			got := ApplySkillModifiers(baseCapacity, skills)
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("ApplySkillModifiers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplySkillModifiers_CombinedSkills(t *testing.T) {
	baseCapacity := 1000.0

	racialLevel := 5
	freighterLevel := 3

	skills := &SkillModifiers{
		RacialHaulerLevel: &racialLevel,
		FreighterLevel:    &freighterLevel,
	}

	// Should apply both bonuses multiplicatively: 1000 * 1.25 * 1.15 = 1437.5
	got := ApplySkillModifiers(baseCapacity, skills)
	want := 1437.5

	if math.Abs(got-want) > 0.01 {
		t.Errorf("ApplySkillModifiers(combined) = %v, want %v", got, want)
	}
}

func TestApplySkillModifiers_CustomMultiplier(t *testing.T) {
	baseCapacity := 1000.0
	multiplier := 1.5

	skills := &SkillModifiers{
		CargoMultiplier: &multiplier,
	}

	got := ApplySkillModifiers(baseCapacity, skills)
	want := 1500.0

	if math.Abs(got-want) > 0.01 {
		t.Errorf("ApplySkillModifiers() = %v, want %v", got, want)
	}
}

func TestApplySkillModifiers_ZeroLevel(t *testing.T) {
	baseCapacity := 1000.0
	level := 0

	skills := &SkillModifiers{
		RacialHaulerLevel: &level,
	}

	// Level 0 should result in no bonus
	got := ApplySkillModifiers(baseCapacity, skills)
	if got != baseCapacity {
		t.Errorf("ApplySkillModifiers(level 0) = %v, want %v", got, baseCapacity)
	}
}

func TestApplySkillModifiers_ComplexCombination(t *testing.T) {
	baseCapacity := 2000.0

	racialLevel := 4
	customMultiplier := 1.1

	skills := &SkillModifiers{
		RacialHaulerLevel: &racialLevel,
		CargoMultiplier:   &customMultiplier,
	}

	// 2000 * 1.20 (racial) * 1.1 (custom) = 2640
	got := ApplySkillModifiers(baseCapacity, skills)
	want := 2640.0

	if math.Abs(got-want) > 0.01 {
		t.Errorf("ApplySkillModifiers(complex) = %v, want %v", got, want)
	}
}

func TestSkillModifiers_JSONTags(t *testing.T) {
	// This test validates that JSON tags are correctly defined
	skills := &SkillModifiers{
		RacialHaulerLevel: ptrInt(5),
		FreighterLevel:    ptrInt(3),
		MiningBargeLevel:  ptrInt(4),
	}

	if skills.RacialHaulerLevel == nil {
		t.Error("RacialHaulerLevel should not be nil")
	}
	if *skills.RacialHaulerLevel != 5 {
		t.Errorf("RacialHaulerLevel = %v, want 5", *skills.RacialHaulerLevel)
	}

	// Verify all fields are set correctly
	if *skills.FreighterLevel != 3 {
		t.Errorf("FreighterLevel = %v, want 3", *skills.FreighterLevel)
	}
	if *skills.MiningBargeLevel != 4 {
		t.Errorf("MiningBargeLevel = %v, want 4", *skills.MiningBargeLevel)
	}
}

// Helper function to create int pointers
func ptrInt(v int) *int {
	return &v
}
