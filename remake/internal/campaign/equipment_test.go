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
