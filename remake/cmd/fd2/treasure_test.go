package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestFinishSelectedWaitRestsThenClaimsTreasure(t *testing.T) {
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	u := &battle.Unit{Camp: battle.Own, HP: 40, MaxHP: 100, OnField: true, X: 18, Y: 37}
	g := &Game{
		st: &battle.State{
			Treasures:      map[battle.Cell]battle.Treasure{{X: 18, Y: 37}: {Slot: 0, Kind: "item", Value: 0xd2}},
			OpenedTreasure: map[int]bool{},
		},
		sel: u, selOrigX: 18, selOrigY: 37, moved: true, ring: true, localeCatalog: catalog,
	}
	g.finishSelectedWait()
	if u.HP != 60 || len(u.Inventory) != 1 || u.Inventory[0] != 0xd2 || !u.Acted || !g.st.OpenedTreasure[0] {
		t.Fatalf("rest/claim = hp%d inventory=%#v acted=%v opened=%v", u.HP, u.Inventory, u.Acted, g.st.OpenedTreasure)
	}
	if g.msg != "取得物品 D2h" || g.sel != nil || g.ring {
		t.Fatalf("wait UI state msg=%q sel=%#v ring=%v", g.msg, g.sel, g.ring)
	}
}

func TestFinishSelectedWaitAfterMoveDoesNotHealAndGoldUsesCampaignBank(t *testing.T) {
	u := &battle.Unit{Camp: battle.Own, HP: 40, MaxHP: 100, OnField: true, X: 3, Y: 4}
	g := &Game{
		st: &battle.State{
			Treasures:      map[battle.Cell]battle.Treasure{{X: 3, Y: 4}: {Slot: 7, Kind: "gold", Value: 3000, Hidden: true}},
			OpenedTreasure: map[int]bool{},
		},
		sel: u, selOrigX: 1, selOrigY: 1, moved: true, gold: 500,
	}
	g.finishSelectedWait()
	if u.HP != 40 || g.gold != 3500 || !g.st.OpenedTreasure[7] {
		t.Fatalf("moved wait = hp%d gold%d opened=%v", u.HP, g.gold, g.st.OpenedTreasure)
	}
}

func TestFinishSelectedWaitExecutesEditableNativeTreasureEvent(t *testing.T) {
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	u := &battle.Unit{Camp: battle.Own, HP: 10, MaxHP: 10, OnField: true, X: 2, Y: 3}
	g := &Game{
		st: &battle.State{
			Treasures: map[battle.Cell]battle.Treasure{
				{X: 2, Y: 3}: {Slot: 0, Kind: "event", NativeType: 2, Value: 58},
			},
			OpenedTreasure: map[int]bool{},
			NativeTreasureEventRules: map[int]battle.NativeTreasureEventRule{
				58: {EventID: 58, ItemBySlot: []int{0x1D}, OpenSlots: []int{0}},
			},
		},
		sel: u, selOrigX: 2, selOrigY: 3, localeCatalog: catalog,
	}
	g.finishSelectedWait()
	if len(u.Inventory) != 1 || u.Inventory[0] != 0x1D ||
		!g.st.OpenedTreasure[0] || g.msg != "取得物品 1Dh" {
		t.Fatalf("event58 wait inventory=%v opened=%v msg=%q", u.Inventory, g.st.OpenedTreasure, g.msg)
	}
}

func TestFinishSelectedWaitHonorsNativeRecoveryGatesAndFloor(t *testing.T) {
	for _, rawIndex := range []int{3, 4} {
		u := &battle.Unit{Camp: battle.Own, HP: 40, MaxHP: 100, OnField: true, X: 2, Y: 3}
		u.NativeTransient[rawIndex] = 1
		g := &Game{st: &battle.State{}, sel: u, selOrigX: 2, selOrigY: 3}
		g.finishSelectedWait()
		if u.HP != 40 {
			t.Fatalf("raw +%#x gate healed to %d", 0x22+rawIndex, u.HP)
		}
	}

	u := &battle.Unit{Camp: battle.Own, HP: 1, MaxHP: 4, OnField: true, X: 2, Y: 3}
	g := &Game{st: &battle.State{}, sel: u, selOrigX: 2, selOrigY: 3}
	g.finishSelectedWait()
	if u.HP != 1 {
		t.Fatalf("floor(maxHP/5) invented minimum heal: HP=%d", u.HP)
	}
}

func TestSpecialDeathRewardPersistsThroughPostBattleSync(t *testing.T) {
	killer := &battle.Unit{Camp: battle.Own, Fig: 0, HP: 50, MaxHP: 50, OnField: true, Inventory: []int{1}}
	dead := &battle.Unit{Camp: battle.Enemy, Fig: 102, HP: 0, MaxHP: 100, OnField: true,
		DeathEffect: &battle.DeathEffect{Type: 2, Value: 39},
		DeathReward: &battle.DeathEffect{Type: 0, Value: 0xd3}}
	g := &Game{
		st:           &battle.State{Units: []*battle.Unit{killer, dead}},
		partyMembers: map[int]bool{0: true},
	}
	g.awardDeathReward(dead, killer)
	g.awardDeathReward(dead, killer) // death transition is once-only
	if len(killer.Inventory) != 2 || killer.Inventory[1] != 0xd3 {
		t.Fatalf("D3 reward = %#v", killer.Inventory)
	}
	if err := g.syncPartyFromBattle(); err != nil {
		t.Fatal(err)
	}
	if got := g.partyRoster[0].Inventory; len(got) != 2 || got[1] != 0xd3 {
		t.Fatalf("postbattle sync lost D3: %#v", got)
	}
}
