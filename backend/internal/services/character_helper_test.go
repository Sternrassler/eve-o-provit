package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCharacterHelper tests character helper initialization
func TestNewCharacterHelper(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	assert.NotNil(t, helper)
	assert.NotNil(t, helper.redisClient)
}

// TestCalculateTaxRate_NoSkills tests tax calculation without skills
func TestCalculateTaxRate_NoSkills(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Simulate ESI failure (no access token) - should return fallback rate
	taxRate, err := helper.CalculateTaxRate(ctx, 12345, "")
	require.NoError(t, err)
	assert.Equal(t, 0.055, taxRate, "Fallback tax rate should be 5.5%")
}

// TestCalculateTaxRate_MaxSkills tests tax calculation with maxed skills
func TestCalculateTaxRate_MaxSkills(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Cache skills data with max levels (Accounting V, Broker Relations V)
	skills := CharacterSkills{
		TotalSP: 5000000,
		Skills: []CharacterSkill{
			{SkillID: 16622, ActiveSkillLevel: 5, TrainedSkillLevel: 5, SkillpointsInSkill: 256000}, // Accounting V
			{SkillID: 3446, ActiveSkillLevel: 5, TrainedSkillLevel: 5, SkillpointsInSkill: 256000},  // Broker Relations V
		},
	}

	skillsJSON, _ := json.Marshal(skills)
	cacheKey := "character_skills:12345"
	s.Set(cacheKey, string(skillsJSON))
	s.SetTTL(cacheKey, 1*time.Hour)

	taxRate, err := helper.CalculateTaxRate(ctx, 12345, "test-token")
	require.NoError(t, err)

	// Calculate expected rate
	// Broker Fee: 3% * (1 - 0.03*5) = 3% * 0.85 = 2.55%
	// Sales Tax: 2.5% * (1 - 0.11*5) = 2.5% * 0.45 = 1.125%
	// Total: 2.55% + 1.125% = 3.675%
	expectedRate := 0.03675
	assert.InDelta(t, expectedRate, taxRate, 0.0001, "Tax rate with max skills should be ~3.675%")
}

// TestCalculateTaxRate_PartialSkills tests tax calculation with partial skills
func TestCalculateTaxRate_PartialSkills(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Cache skills data with level 3 skills
	skills := CharacterSkills{
		TotalSP: 2000000,
		Skills: []CharacterSkill{
			{SkillID: 16622, ActiveSkillLevel: 3, TrainedSkillLevel: 3, SkillpointsInSkill: 40000}, // Accounting III
			{SkillID: 3446, ActiveSkillLevel: 3, TrainedSkillLevel: 3, SkillpointsInSkill: 40000},  // Broker Relations III
		},
	}

	skillsJSON, _ := json.Marshal(skills)
	cacheKey := "character_skills:12345"
	s.Set(cacheKey, string(skillsJSON))
	s.SetTTL(cacheKey, 1*time.Hour)

	taxRate, err := helper.CalculateTaxRate(ctx, 12345, "test-token")
	require.NoError(t, err)

	// Calculate expected rate
	// Broker Fee: 3% * (1 - 0.03*3) = 3% * 0.91 = 2.73%
	// Sales Tax: 2.5% * (1 - 0.11*3) = 2.5% * 0.67 = 1.675%
	// Total: 2.73% + 1.675% = 4.405%
	expectedRate := 0.04405
	assert.InDelta(t, expectedRate, taxRate, 0.0001, "Tax rate with level 3 skills should be ~4.405%")
}

// TestCalculateTaxRate_AccountingOnly tests tax with only Accounting trained
func TestCalculateTaxRate_AccountingOnly(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Cache skills data with only Accounting
	skills := CharacterSkills{
		TotalSP: 256000,
		Skills: []CharacterSkill{
			{SkillID: 16622, ActiveSkillLevel: 5, TrainedSkillLevel: 5, SkillpointsInSkill: 256000}, // Accounting V only
		},
	}

	skillsJSON, _ := json.Marshal(skills)
	cacheKey := "character_skills:12345"
	s.Set(cacheKey, string(skillsJSON))
	s.SetTTL(cacheKey, 1*time.Hour)

	taxRate, err := helper.CalculateTaxRate(ctx, 12345, "test-token")
	require.NoError(t, err)

	// Calculate expected rate
	// Broker Fee: 3% (no reduction)
	// Sales Tax: 2.5% * (1 - 0.11*5) = 2.5% * 0.45 = 1.125%
	// Total: 3% + 1.125% = 4.125%
	expectedRate := 0.04125
	assert.InDelta(t, expectedRate, taxRate, 0.0001, "Tax rate with Accounting V only should be ~4.125%")
}

// TestCalculateTaxRate_BrokerRelationsOnly tests tax with only Broker Relations
func TestCalculateTaxRate_BrokerRelationsOnly(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Cache skills data with only Broker Relations
	skills := CharacterSkills{
		TotalSP: 256000,
		Skills: []CharacterSkill{
			{SkillID: 3446, ActiveSkillLevel: 5, TrainedSkillLevel: 5, SkillpointsInSkill: 256000}, // Broker Relations V only
		},
	}

	skillsJSON, _ := json.Marshal(skills)
	cacheKey := "character_skills:12345"
	s.Set(cacheKey, string(skillsJSON))
	s.SetTTL(cacheKey, 1*time.Hour)

	taxRate, err := helper.CalculateTaxRate(ctx, 12345, "test-token")
	require.NoError(t, err)

	// Calculate expected rate
	// Broker Fee: 3% * (1 - 0.03*5) = 3% * 0.85 = 2.55%
	// Sales Tax: 2.5% (no reduction)
	// Total: 2.55% + 2.5% = 5.05%
	expectedRate := 0.0505
	assert.InDelta(t, expectedRate, taxRate, 0.0001, "Tax rate with Broker Relations V only should be ~5.05%")
}

// TestGetCharacterSkills_CacheHit tests skills retrieval with cache hit
func TestGetCharacterSkills_CacheHit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Pre-populate cache
	skills := CharacterSkills{
		TotalSP: 5000000,
		Skills: []CharacterSkill{
			{SkillID: 16622, ActiveSkillLevel: 5, TrainedSkillLevel: 5, SkillpointsInSkill: 256000},
		},
	}

	skillsJSON, _ := json.Marshal(skills)
	cacheKey := "character_skills:12345"
	s.Set(cacheKey, string(skillsJSON))
	s.SetTTL(cacheKey, 1*time.Hour)

	// This should hit the cache (no ESI call needed)
	result, err := helper.GetCharacterSkills(ctx, 12345, "fake-token")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 5000000, result.TotalSP)
	assert.Len(t, result.Skills, 1)
	assert.Equal(t, 16622, result.Skills[0].SkillID)
}

// TestGetCharacterLocation_CacheHit tests location retrieval with cache hit
func TestGetCharacterLocation_CacheHit(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Pre-populate cache
	stationID := int64(60003760)
	location := CharacterLocation{
		SolarSystemID: 30000142,
		StationID:     &stationID,
	}

	locationJSON, _ := json.Marshal(location)
	cacheKey := "character_location:12345"
	s.Set(cacheKey, string(locationJSON))
	s.SetTTL(cacheKey, 5*time.Minute)

	// This should hit the cache
	result, err := helper.GetCharacterLocation(ctx, 12345, "fake-token")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(30000142), result.SolarSystemID)
	assert.NotNil(t, result.StationID)
	assert.Equal(t, int64(60003760), *result.StationID)
}

// TestGetCharacterLocation_InSpace tests location when character is in space
func TestGetCharacterLocation_InSpace(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// Character in space (no station/structure)
	location := CharacterLocation{
		SolarSystemID: 30000142,
		StationID:     nil,
		StructureID:   nil,
	}

	locationJSON, _ := json.Marshal(location)
	cacheKey := "character_location:12345"
	s.Set(cacheKey, string(locationJSON))
	s.SetTTL(cacheKey, 5*time.Minute)

	result, err := helper.GetCharacterLocation(ctx, 12345, "fake-token")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, int64(30000142), result.SolarSystemID)
	assert.Nil(t, result.StationID)
	assert.Nil(t, result.StructureID)
}

// TestCharacterSkill_Marshaling tests JSON marshaling of character skills
func TestCharacterSkill_Marshaling(t *testing.T) {
	skill := CharacterSkill{
		SkillID:            16622,
		ActiveSkillLevel:   5,
		TrainedSkillLevel:  5,
		SkillpointsInSkill: 256000,
	}

	jsonData, err := json.Marshal(skill)
	require.NoError(t, err)

	var unmarshaled CharacterSkill
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, skill.SkillID, unmarshaled.SkillID)
	assert.Equal(t, skill.ActiveSkillLevel, unmarshaled.ActiveSkillLevel)
	assert.Equal(t, skill.TrainedSkillLevel, unmarshaled.TrainedSkillLevel)
	assert.Equal(t, skill.SkillpointsInSkill, unmarshaled.SkillpointsInSkill)
}

// TestCharacterLocation_Marshaling tests JSON marshaling of character location
func TestCharacterLocation_Marshaling(t *testing.T) {
	stationID := int64(60003760)
	location := CharacterLocation{
		SolarSystemID: 30000142,
		StationID:     &stationID,
		StructureID:   nil,
	}

	jsonData, err := json.Marshal(location)
	require.NoError(t, err)

	var unmarshaled CharacterLocation
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, location.SolarSystemID, unmarshaled.SolarSystemID)
	assert.NotNil(t, unmarshaled.StationID)
	assert.Equal(t, *location.StationID, *unmarshaled.StationID)
	assert.Nil(t, unmarshaled.StructureID)
}

// TestCalculateTaxRate_SkillFormulas tests skill reduction formulas
func TestCalculateTaxRate_SkillFormulas(t *testing.T) {
	tests := []struct {
		name                 string
		accountingLevel      int
		brokerRelationsLevel int
		expectedBrokerFee    float64
		expectedSalesTax     float64
		expectedTotal        float64
	}{
		{
			name:                 "Level 0 (untrained)",
			accountingLevel:      0,
			brokerRelationsLevel: 0,
			expectedBrokerFee:    0.03,     // 3% * (1 - 0) = 3%
			expectedSalesTax:     0.025,    // 2.5% * (1 - 0) = 2.5%
			expectedTotal:        0.055,    // 5.5%
		},
		{
			name:                 "Level 1",
			accountingLevel:      1,
			brokerRelationsLevel: 1,
			expectedBrokerFee:    0.0291,   // 3% * (1 - 0.03) = 2.91%
			expectedSalesTax:     0.02225,  // 2.5% * (1 - 0.11) = 2.225%
			expectedTotal:        0.05135,  // 5.135%
		},
		{
			name:                 "Level 2",
			accountingLevel:      2,
			brokerRelationsLevel: 2,
			expectedBrokerFee:    0.0282,   // 3% * (1 - 0.06) = 2.82%
			expectedSalesTax:     0.0195,   // 2.5% * (1 - 0.22) = 1.95%
			expectedTotal:        0.0477,   // 4.77%
		},
		{
			name:                 "Level 4",
			accountingLevel:      4,
			brokerRelationsLevel: 4,
			expectedBrokerFee:    0.0264,   // 3% * (1 - 0.12) = 2.64%
			expectedSalesTax:     0.014,    // 2.5% * (1 - 0.44) = 1.4%
			expectedTotal:        0.0404,   // 4.04%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := miniredis.RunT(t)
			defer s.Close()

			redisClient := redis.NewClient(&redis.Options{
				Addr: s.Addr(),
			})
			defer redisClient.Close()

			helper := NewCharacterHelper(redisClient)
			ctx := context.Background()

			// Cache skills data
			skills := CharacterSkills{
				TotalSP: 1000000,
				Skills:  []CharacterSkill{},
			}

			if tt.accountingLevel > 0 {
				skills.Skills = append(skills.Skills, CharacterSkill{
					SkillID:           16622,
					ActiveSkillLevel:  tt.accountingLevel,
					TrainedSkillLevel: tt.accountingLevel,
				})
			}

			if tt.brokerRelationsLevel > 0 {
				skills.Skills = append(skills.Skills, CharacterSkill{
					SkillID:           3446,
					ActiveSkillLevel:  tt.brokerRelationsLevel,
					TrainedSkillLevel: tt.brokerRelationsLevel,
				})
			}

			skillsJSON, _ := json.Marshal(skills)
			cacheKey := "character_skills:12345"
			s.Set(cacheKey, string(skillsJSON))
			s.SetTTL(cacheKey, 1*time.Hour)

			taxRate, err := helper.CalculateTaxRate(ctx, 12345, "test-token")
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedTotal, taxRate, 0.0001, "Total tax rate mismatch")
		})
	}
}

// TestGetCharacterSkills_EmptyCache tests skills fetch with cache miss
func TestGetCharacterSkills_EmptyCache(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// This will fail because we're not mocking the ESI HTTP call
	// But we can test that it attempts to make the call
	_, err := helper.GetCharacterSkills(ctx, 12345, "fake-token")
	assert.Error(t, err, "Should error when ESI is not available")
}

// TestGetCharacterLocation_EmptyCache tests location fetch with cache miss
func TestGetCharacterLocation_EmptyCache(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})
	defer redisClient.Close()

	helper := NewCharacterHelper(redisClient)
	ctx := context.Background()

	// This will fail because we're not mocking the ESI HTTP call
	_, err := helper.GetCharacterLocation(ctx, 12345, "fake-token")
	assert.Error(t, err, "Should error when ESI is not available")
}
