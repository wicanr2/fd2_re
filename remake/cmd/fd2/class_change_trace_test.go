package main

import (
	"math/rand"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestCampaignTownChurchClassChangeReturnTrace(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	c := &campaign.Campaign{
		Start: "town_ch02",
		Flags: map[string]bool{},
		Nodes: map[string]*campaign.Node{
			"town_ch02": {Type: "town", Options: []campaign.Option{
				{Label: "武器店", To: "shop"},
				{Label: "教會", To: "church_ch02"},
			}},
			"shop":        {Type: "shop", Next: "town_ch02"},
			"church_ch02": {Type: "church", Next: "town_ch02"},
		},
	}
	u := battle.Unit{
		Name: "悠妮", Portrait: 9, BattleFig: 9, HasBattleFig: true,
		ClassID: 5, NativeRecordClass: 5, HasNativeRecordClass: true,
		NativeIdentity: 9, HasNativeIdentity: true,
		MapSelectorKey: 9, HasMapSelectorKey: true,
		Lv: 20, Exp: 31, HP: 22, MaxHP: 30, MP: 7, MaxMP: 10,
		AP: 20, DP: 18, DX: 12, MV: 5,
		Inventory: []int{0x5a, 0x64}, Equipped: []bool{false, true},
		InventorySlots:       []int{0x5a, 0x64, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0, 0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
	}
	optionalTarget, specialTarget := 0x3b, 0x34
	g := &Game{
		camp:           campaign.NewRunner(c),
		partyRoster:    map[int]battle.Unit{9: u},
		partyMembers:   map[int]bool{9: true},
		partyJoinOrder: []int{9},
		classChangeTable: campaign.ClassChangeTable{
			Current: map[int]campaign.ClassChangeCurrent{9: {
				Portrait: 9, DefaultTarget: 0x29, ItemID: 0x58,
				OptionalTarget: &optionalTarget, SpecialItem: 0x5a, SpecialTarget: &specialTarget,
			}},
			Targets: map[int]campaign.ClassChangeTarget{
				0x29: {Portrait: 0x29, ClassID: 13},
				0x3b: {Portrait: 0x3b, ClassID: 22},
				0x34: {Portrait: 0x34, ClassID: 21, MobilityIncrement: 2},
			},
		},
		classChangeGrowth: map[int]campaign.ClassChangeGrowth{0x34: {AP: [2]int{10, 11}, DP: [2]int{20, 21}, DX: [2]int{30, 31}, HP: [2]int{40, 41}, MP: [2]int{50, 51}}},
		shopItemStats: map[int]campaign.ItemStats{
			0x64: {AP: 3, DP: 2, HIT: 1, EV: 2, MV: 1},
		},
		rng: rand.New(rand.NewSource(1)),
	}
	g.stepCampaignMenu(campaign.MenuDown)
	selected, confirm := g.stepCampaignMenu(campaign.MenuConfirm)
	if selected != 1 || !confirm || g.camp.Advance("opt1") != "church_ch02" {
		t.Fatalf("town→church trace=(%d,%v,%q)", selected, confirm, g.camp.NodeID())
	}
	target, ok := campaign.NativeClassChangeTarget(&u, g.classChangeTable)
	if !ok || target.Branch != "special" || target.Portrait != 0x34 {
		t.Fatalf("native target priority=%+v ok=%v", target, ok)
	}
	g.churchMode, g.churchClassID = "class_confirm", 9
	g.churchBranches = []campaign.ClassChangeBranch{target}
	if !g.applyChurchClassChange(0) {
		t.Fatalf("class change failed: msg=%q", g.msg)
	}
	changed := g.partyRoster[9]
	if changed.Portrait != 0x34 || changed.ClassID != 21 || changed.MV != 8 || changed.Exp != 0 ||
		changed.AP != 33 || changed.DP != 40 || changed.DX != 42 || changed.HP != 70 || changed.MP != 60 ||
		len(changed.Inventory) != 1 || changed.Inventory[0] != 0x64 || len(changed.Equipped) != 1 || !changed.Equipped[0] {
		t.Fatalf("class mutation=%#v", changed)
	}
	g.leaveChurch()
	if got := g.camp.NodeID(); got != "town_ch02" {
		t.Fatalf("church class return node=%q, want town_ch02", got)
	}
	g.partyDeploy = map[int]bool{9: true}
	g.gold, g.items, g.handlerChapter = 279, []string{"sky-key"}, 2
	g.saveGameToSlot(2)
	if g.msg != "已存檔(槽位3：town_ch02)" {
		t.Fatalf("class-change save boundary=%q", g.msg)
	}

	g.camp.Cur = "church_ch02"
	g.partyMembers, g.partyJoinOrder, g.partyDeploy, g.partyRoster = nil, nil, nil, nil
	g.gold, g.items, g.handlerChapter = 0, nil, 0
	g.churchMode = "class_confirm"
	g.churchSel = 3
	g.churchClassID = 9
	g.churchBranches = []campaign.ClassChangeBranch{target}
	g.nativeChurchTextIndex = 590
	g.nativeClassUIJob = &nativeClassUIJob{}
	g.nativeChurchUIJob = &nativeChurchUIJob{}
	g.st = &battle.State{}
	g.loadGameFromSlot(2)

	if g.camp.NodeID() != "town_ch02" || g.gold != 279 || len(g.items) != 1 ||
		g.items[0] != "sky-key" || g.handlerChapter != 2 {
		t.Fatalf("class-change campaign restore: node=%q gold=%d items=%#v chapter=%d",
			g.camp.NodeID(), g.gold, g.items, g.handlerChapter)
	}
	restored, ok := g.partyRoster[9]
	if !ok || !g.partyMembers[9] || len(g.partyJoinOrder) != 1 || g.partyJoinOrder[0] != 9 || !g.partyDeploy[9] {
		t.Fatalf("class-change party topology: roster=%#v members=%#v order=%#v deploy=%#v",
			g.partyRoster, g.partyMembers, g.partyJoinOrder, g.partyDeploy)
	}
	if restored.Portrait != 0x34 || restored.ClassID != 21 || restored.NativeRecordClass != 21 ||
		!restored.HasNativeRecordClass || restored.BattleFig != 0x34 || restored.MapSelectorKey != 0x34 ||
		!restored.HasMapSelectorKey || restored.MV != 8 || restored.Exp != 0 ||
		restored.AP != 33 || restored.DP != 40 || restored.DX != 42 || restored.HP != 70 || restored.MP != 60 ||
		len(restored.Inventory) != 1 || restored.Inventory[0] != 0x64 ||
		len(restored.Equipped) != 1 || !restored.Equipped[0] ||
		len(restored.InventorySlots) != 8 || restored.InventorySlots[0] != 0xff || restored.InventorySlots[1] != 0x64 ||
		len(restored.NativeInventoryFlags) != 8 || restored.NativeInventoryFlags[0] != 0x80 || restored.NativeInventoryFlags[1] != 0x40 ||
		!restored.EquipmentBaseSet || restored.BaseAP != 30 || restored.BaseDP != 38 || restored.BaseMV != 7 {
		t.Fatalf("class-change persisted record=%#v", restored)
	}
	if g.st != nil || g.churchMode != "" || g.churchSel != 0 || len(g.churchBranches) != 0 ||
		g.churchClassID != -1 || g.nativeChurchTextIndex != 0 ||
		g.nativeClassUIJob != nil || g.nativeChurchUIJob != nil {
		t.Fatalf("class-change transient state leaked: st=%v mode=%q sel=%d class=%d branches=%#v text=%d class_job=%v church_job=%v",
			g.st, g.churchMode, g.churchSel, g.churchClassID, g.churchBranches,
			g.nativeChurchTextIndex, g.nativeClassUIJob, g.nativeChurchUIJob)
	}
}
