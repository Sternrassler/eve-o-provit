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

// TestActiveShipEffectiveCargo_IsInstanceScoped guards the dropdown follow-up bug:
// the active/flown ship (the ROI default prefill) must compute effective cargo for
// THIS exact instance by ship_item_id, not per type / first-instance. Two same-type
// ships differing only by their fitting must yield different effective cargo through
// the same item-scoped seam that EffectiveCargoForActiveShip delegates to.
func TestActiveShipEffectiveCargo_IsInstanceScoped(t *testing.T) {
	db := testutil.OpenTestDB(t)
	s := &FittingService{sdeDB: db}
	ctx := context.Background()

	skills := &cargo.CharacterSkills{Skills: []struct {
		SkillID           int64 `json:"skill_id"`
		ActiveSkillLevel  int   `json:"active_skill_level"`
		TrainedSkillLevel int   `json:"trained_skill_level"`
	}{{SkillID: 3340, ActiveSkillLevel: 1, TrainedSkillLevel: 1}}}

	// Two same-type (Iteron, 657) instances in the shared asset list: 333 is empty,
	// 444 carries a Reinforced Bulkhead reducing its cargo.
	assets := []esiAsset{
		{ItemID: 333, TypeID: 657, IsSingleton: true},
		{ItemID: 444, TypeID: 657, IsSingleton: true},
		{ItemID: 555, TypeID: reinforcedBulkheadTypeID, LocationID: 444, LocationFlag: "LoSlot0"},
	}

	effEmpty, unavailEmpty := s.EffectiveCargoForShipItem(ctx, 657, 333, assets, skills)
	effBulk, unavailBulk := s.EffectiveCargoForShipItem(ctx, 657, 444, assets, skills)

	if unavailEmpty || unavailBulk {
		t.Fatalf("both instances should resolve: empty=%v bulk=%v", unavailEmpty, unavailBulk)
	}
	if effBulk >= effEmpty {
		t.Errorf("active ship must be instance-scoped by item_id: empty=%.1f must exceed bulk=%.1f", effEmpty, effBulk)
	}
}
