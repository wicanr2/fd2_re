package campaign

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestEquipItemRecomputesFromBaseAndReplacesEXECategory(t *testing.T) {
	u := &battle.Unit{AP: 16, DP: 12, HIT: 97, EV: 2, MV: 4, BaseAP: 6, BaseDP: 4, BaseHIT: 2, BaseEV: 2, BaseMV: 4, EquipmentBaseSet: true, Inventory: []int{0, 1, 132}}
	stats := map[int]ItemStats{
		0:   {Type: 1, AP: 10, HIT: 95},
		1:   {Type: 1, AP: 20, HIT: 95},
		132: {Type: 22, DP: 8},
	}
	if err := EquipItem(u, 0, stats); err != nil || u.AP != 16 || u.HIT != 97 {
		t.Fatalf("initial equip err=%v AP=%d HIT=%d", err, u.AP, u.HIT)
	}
	if err := EquipItem(u, 1, stats); err != nil || u.AP != 26 || u.HIT != 97 || u.Equipped[0] || !u.Equipped[1] {
		t.Fatalf("replacement err=%v AP=%d HIT=%d flags=%v", err, u.AP, u.HIT, u.Equipped)
	}
}

func TestInitializeEquipmentBaseSubtractsAuthoredStartingGearOnce(t *testing.T) {
	u := &battle.Unit{AP: 16, DP: 12, HIT: 97, EV: 2, Inventory: []int{0, 132}, Equipped: []bool{true, true}}
	stats := map[int]ItemStats{0: {Type: 1, AP: 10, HIT: 95}, 132: {Type: 22, DP: 8}}
	InitializeEquipmentBase(u, stats)
	if !u.EquipmentBaseSet || u.BaseAP != 6 || u.BaseDP != 4 || u.BaseHIT != 2 || u.AP != 16 || u.DP != 12 || u.HIT != 97 {
		t.Fatalf("base conversion = base(%d,%d,%d) effective(%d,%d,%d)", u.BaseAP, u.BaseDP, u.BaseHIT, u.AP, u.DP, u.HIT)
	}
	InitializeEquipmentBase(u, stats)
	if u.BaseAP != 6 || u.AP != 16 {
		t.Fatalf("base conversion repeated: base=%d AP=%d", u.BaseAP, u.AP)
	}
}

func TestRecomputeAfterClassChangeUsesDXForHitAndEVBase(t *testing.T) {
	u := &battle.Unit{AP: 10, DP: 8, DX: 12, HIT: 99, EV: 77, MV: 4, Inventory: []int{1}, Equipped: []bool{true}}
	stats := map[int]ItemStats{1: {AP: 3, DP: 2, HIT: 5, EV: 7}}
	RecomputeAfterClassChange(u, stats)
	if u.BaseHIT != 12 || u.BaseEV != 12 || u.HIT != 17 || u.EV != 19 {
		t.Fatalf("class base synthesis hit/ev base=%d/%d effective=%d/%d", u.BaseHIT, u.BaseEV, u.HIT, u.EV)
	}
	if u.AP != 13 || u.DP != 10 {
		t.Fatalf("class base AP/DP effective=%d/%d", u.AP, u.DP)
	}
}

func TestRecomputeAfterClassChangeDoesNotDoubleCountExistingEquipment(t *testing.T) {
	u := &battle.Unit{
		AP: 18, DP: 12, DX: 14, MV: 5,
		Inventory: []int{1}, Equipped: []bool{true},
		EquipmentBaseSet: true, BaseAP: 15, BaseDP: 10, BaseMV: 5,
	}
	stats := map[int]ItemStats{1: {Type: 1, AP: 3, DP: 2}}
	// AP/DP already include the old weapon (15+3, 10+2); after a class
	// change the raw growth has made them 18/12. Recompute must retain those
	// totals, not produce 21/14 by adding the weapon twice.
	RecomputeAfterClassChange(u, stats)
	if u.BaseAP != 15 || u.BaseDP != 10 || u.AP != 18 || u.DP != 12 {
		t.Fatalf("existing equipment double-counted: base=%d/%d effective=%d/%d", u.BaseAP, u.BaseDP, u.AP, u.DP)
	}
}
