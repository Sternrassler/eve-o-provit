package services

import (
	"context"
	"testing"

	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/cargo"
	"github.com/Sternrassler/eve-o-provit/backend/pkg/evedb/testutil"
)

// Reinforced Bulkheads I (typeID 1333) has dogma attr 149 (cargoCapacityMultiplier) = 0.9,
// which reduces cargo capacity by 10%. Confirmed in SDE typeDogma table.
const reinforcedBulkheadTypeID = 1333

func TestEffectiveCargoForShipItem_PerInstance(t *testing.T) {
	db := testutil.OpenTestDB(t)
	s := &FittingService{sdeDB: db}
	ctx := context.Background()

	skills := &cargo.CharacterSkills{Skills: []struct {
		SkillID           int64 `json:"skill_id"`
		ActiveSkillLevel  int   `json:"active_skill_level"`
		TrainedSkillLevel int   `json:"trained_skill_level"`
	}{{SkillID: 3340, ActiveSkillLevel: 1, TrainedSkillLevel: 1}}}

	emptyAssets := []esiAsset{{ItemID: 111, TypeID: 657, IsSingleton: true}}
	empty, unavailEmpty := s.EffectiveCargoForShipItem(ctx, 657, 111, emptyAssets, skills)
	if unavailEmpty {
		t.Fatalf("empty ship should not be unavailable")
	}
	if empty < 6089 || empty > 6091 {
		t.Errorf("empty Iteron effective: want ~6090, got %.1f", empty)
	}

	bulkAssets := []esiAsset{
		{ItemID: 222, TypeID: 657, IsSingleton: true},
		{ItemID: 999, TypeID: reinforcedBulkheadTypeID, LocationID: 222, LocationFlag: "LoSlot0"},
	}
	bulk, _ := s.EffectiveCargoForShipItem(ctx, 657, 222, bulkAssets, skills)
	if bulk >= empty {
		t.Errorf("bulkhead ship cargo should be below the empty ship: empty=%.1f bulk=%.1f", empty, bulk)
	}
}
