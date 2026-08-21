package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestSaveDataRoundTripsPersistentParty(t *testing.T) {
	gateA := 0
	want := saveData{
		Node: "story_ch02", Flags: map[string]bool{"won_ch01": true}, Gold: 321,
		PartyMembers: map[int]bool{0: true, 9: true}, PartyJoinOrder: []int{0, 9},
		PartyDeploy: map[int]bool{0: true, 9: false},
		PartyRoster: map[int]battle.Unit{
			9: {Fig: 9, Name: "悠妮", Lv: 4, HP: 23, MaxHP: 37, MP: 18, MaxMP: 24, Exp: 67.5, Spells: []int{0, 4, 13}, Inventory: []int{0xc6, 0x64}},
		},
		Chapter:        1,
		NativeHUDGateA: &gateA,
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got saveData
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	yuni, ok := got.PartyRoster[9]
	if !ok || got.Node != want.Node || got.Chapter != 1 || got.NativeHUDGateA == nil || *got.NativeHUDGateA != 0 || !got.PartyMembers[0] || len(got.PartyJoinOrder) != 2 || !got.PartyDeploy[0] || got.PartyDeploy[9] {
		t.Fatalf("campaign progress did not round-trip: %#v", got)
	}
	if yuni.Fig != 9 || yuni.Lv != 4 || yuni.HP != 23 || yuni.MaxHP != 37 || yuni.MP != 18 || yuni.MaxMP != 24 || yuni.Exp != 67.5 || len(yuni.Spells) != 3 || len(yuni.Inventory) != 2 || yuni.Inventory[0] != 0xc6 || yuni.Inventory[1] != 0x64 {
		t.Fatalf("party roster did not round-trip: %#v", yuni)
	}
}

func TestSaveSlotsAreBoundedAndDistinct(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("FD2_MUTE", "1")
	if saveSlotPath(-1) != saveSlotPath(0) || saveSlotPath(4) != saveSlotPath(0) {
		t.Fatal("invalid save slot did not fail closed to slot 0")
	}
	if saveSlotPath(0) == saveSlotPath(1) || filepath.Base(saveSlotPath(3)) != "fd2_save_3.json" {
		t.Fatalf("save slot paths are not distinct/bounded: %q %q", saveSlotPath(0), saveSlotPath(3))
	}
}

func TestSaveRejectsPostBattleHandlerWithoutSerializableRuntimeContext(t *testing.T) {
	c := &campaign.Campaign{Start: "post", Nodes: map[string]*campaign.Node{
		"post":   {Type: "cutscene", HandlerBinding: "assets/cutscenes/bindings/ch01_post.json", Next: "choice"},
		"choice": {Type: "choice"},
	}}
	g := &Game{camp: campaign.NewRunner(c), st: &battle.State{Units: []*battle.Unit{{Fig: 0}}}}
	g.saveGame()
	if g.msg != "戰後演出進行中，請在下一個節點存檔" {
		t.Fatalf("unsafe postbattle save was not rejected: %q", g.msg)
	}
}

func TestSaveRejectsUnboundPostbattleBoundary(t *testing.T) {
	c := &campaign.Campaign{Start: "postbattle_ch04_persist", Nodes: map[string]*campaign.Node{
		"postbattle_ch04_persist": {Type: "cutscene", Next: "town_ch05"},
		"town_ch05":               {Type: "town"},
	}}
	g := &Game{camp: campaign.NewRunner(c)}
	g.saveGame()
	if g.msg != "戰後演出進行中，請在下一個節點存檔" {
		t.Fatalf("unbound postbattle save was not rejected: %q", g.msg)
	}
}

func TestWriteSaveFileReplacesCompleteJSONAtomically(t *testing.T) {
	path := t.TempDir() + "/fd2_save.json"
	if err := os.WriteFile(path, []byte(`{"node":"old"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"node":"town_ch04","gold":99}`)
	if err := writeSaveFile(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("save contents = %q, want %q", got, want)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".fd2-save-") {
			t.Fatalf("temporary save file leaked: %s", entry.Name())
		}
	}
}

func TestCampaignSaveLoadRestoresTownBoundaryAndParty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	c := &campaign.Campaign{Start: "town_ch02", Flags: map[string]bool{}, Nodes: map[string]*campaign.Node{
		"town_ch02":   {Type: "town"},
		"church_ch02": {Type: "church", Next: "town_ch02"},
	}}
	u := battle.Unit{Name: "悠妮", Portrait: 0x34, ClassID: 21, Lv: 20, HP: 44, MaxHP: 44, MV: 7, Inventory: []int{0x64}}
	g := &Game{
		camp: cRunner(c), gold: 279, items: []string{"sky-key"}, handlerChapter: 2,
		partyMembers: map[int]bool{0: true}, partyJoinOrder: []int{0},
		partyDeploy: map[int]bool{0: true}, partyRoster: map[int]battle.Unit{0: u},
		nativeMapHUDPersistent: battle.NativeMapHUDPersistentState{
			DisplayGateA: 0, AnchorX: 0xf2,
			HasDisplayGateA: true, HasAnchorX: true,
		},
	}
	g.saveGameToSlot(2)
	if g.msg != "已存檔(槽位3：town_ch02)" {
		t.Fatalf("save boundary message=%q", g.msg)
	}
	g.camp.Cur, g.gold, g.items = "church_ch02", 1, nil
	g.partyMembers, g.partyJoinOrder = nil, nil
	g.partyDeploy, g.partyRoster = nil, nil
	g.nativeMapHUDPersistent = battle.InitialNativeMapHUDPersistentState()
	g.shopMode = "sell"
	g.shopPicking = true
	g.shopEquipPrompt = true
	g.shopRecipients = []int{9}
	g.nativeShopUIJob = &nativeClassUIJob{frames: [][]byte{{1}}}
	g.nativeShopMode = "transfer_full"
	g.nativeShopVariant = 5
	g.nativeShopHasPendingUnit = true
	g.nativeShopSellItemIDs = []int{0x64}
	g.nativeShopTransferSource = 9
	g.nativeShopTransferItems = []int{0}
	g.nativeShopTransferIDs = []int{9}
	g.nativeItemPanelBase = []byte{1}
	g.nativeItemPanelRecord = []byte{1}
	g.nativeItemEffectRows = []byte{1}
	g.itemAnimStep = 11
	g.itemClosing = true
	g.loadGameFromSlot(2)
	if g.camp.NodeID() != "town_ch02" || g.gold != 279 || len(g.items) != 1 || g.items[0] != "sky-key" {
		t.Fatalf("campaign boundary did not restore: node=%q gold=%d items=%#v", g.camp.NodeID(), g.gold, g.items)
	}
	got, ok := g.partyRoster[0]
	if !ok || got.Portrait != 0x34 || got.ClassID != 21 || got.MV != 7 || got.HP != 44 || len(g.partyJoinOrder) != 1 || !g.partyDeploy[0] {
		t.Fatalf("persistent party did not restore: roster=%#v join=%#v deploy=%#v", g.partyRoster, g.partyJoinOrder, g.partyDeploy)
	}
	if !g.nativeMapHUDPersistent.HasDisplayGateA || g.nativeMapHUDPersistent.DisplayGateA != 0 ||
		!g.nativeMapHUDPersistent.HasAnchorX || g.nativeMapHUDPersistent.AnchorX != 1 {
		t.Fatalf("save HUD persistence=%+v", g.nativeMapHUDPersistent)
	}
	if g.st != nil || g.sel != nil || g.shopMode != "" || g.churchMode != "" ||
		g.shopPicking || g.shopEquipPrompt || len(g.shopRecipients) != 0 ||
		g.nativeShopUIJob != nil || g.nativeShopMode != "" || g.nativeShopVariant != 0 ||
		g.nativeShopHasPendingUnit || len(g.nativeShopSellItemIDs) != 0 ||
		g.nativeShopTransferSource != -1 || len(g.nativeShopTransferItems) != 0 ||
		len(g.nativeShopTransferIDs) != 0 || len(g.nativeItemPanelBase) != 0 ||
		len(g.nativeItemPanelRecord) != 0 || len(g.nativeItemEffectRows) != 0 ||
		g.itemAnimStep != 0 || g.itemClosing {
		t.Fatalf("town boundary retained transient scene state: st=%v sel=%v shop=%q nativeShop=%q church=%q", g.st, g.sel, g.shopMode, g.nativeShopMode, g.churchMode)
	}
}

func TestCampaignLoadRejectsOutOfRangeNativeHUDGateWithoutMutation(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	c := &campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{
		"town": {Type: "town"},
	}}
	invalid := 256
	raw, err := json.Marshal(saveData{Node: "town", Gold: 99, NativeHUDGateA: &invalid})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(saveSlotPath(0)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(saveSlotPath(0), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		camp: campaign.NewRunner(c), gold: 7,
		nativeMapHUDPersistent: battle.InitialNativeMapHUDPersistentState(),
	}
	g.loadGameFromSlot(0)
	if g.gold != 7 || g.nativeMapHUDPersistent.DisplayGateA != 1 ||
		g.msg != "存檔 native HUD gate A 超出原始 byte 範圍" {
		t.Fatalf("invalid HUD save mutated game: gold=%d HUD=%+v msg=%q",
			g.gold, g.nativeMapHUDPersistent, g.msg)
	}
}

func cRunner(c *campaign.Campaign) *campaign.Runner { return campaign.NewRunner(c) }
