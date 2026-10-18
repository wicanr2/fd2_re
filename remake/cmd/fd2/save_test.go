package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func TestStartNativeSlotPreservesContainerAndRejectsMissingProvenance(t *testing.T) {
	plain := bytes.Repeat([]byte{0x5a}, fdsave.FileSize)
	stored, err := fdsave.Encode(plain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "FD2.SAV")
	if err := os.WriteFile(path, stored, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_SAVE", path)
	g := &Game{handlerChapter: 1, gold: 1000}
	if err := g.saveNativeChapterSlot(2); err == nil || g.nativeChapterSlotBaseline != nil {
		t.Fatal("沒有 START／LOAD 來源卻發布保存基底")
	}
	g.nativeChapterSlotStarted = true
	if err := os.WriteFile(path, stored[:5], 0o644); err != nil {
		t.Fatal(err)
	}
	if err := g.saveNativeChapterSlot(2); err == nil || g.nativeChapterSlotBaseline != nil || len(g.nativeChapterSlotPlain) != 0 {
		t.Fatal("壞容器沒有零發布拒收")
	}
	if raw, _ := os.ReadFile(path); !bytes.Equal(raw, stored[:5]) {
		t.Fatal("拒收改動了容器")
	}
	if err := os.WriteFile(path, stored, 0o644); err != nil {
		t.Fatal(err)
	}
	join, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	items, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	u, err := join.MaterializePersistentUnit(0, battle.Unit{Fig: 0}, items)
	if err != nil {
		t.Fatal(err)
	}
	g.partyRoster, g.partyJoinOrder = map[int]battle.Unit{0: u}, []int{0}
	if err := g.saveNativeChapterSlot(2); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := fdsave.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	start, end, _ := fdsave.SlotBounds(2)
	if !bytes.Equal(result[:start], plain[:start]) || !bytes.Equal(result[end:fdsave.ChecksumOff], plain[end:fdsave.ChecksumOff]) || !bytes.Equal(result[start+fdsave.RosterSize+10:end], plain[start+fdsave.RosterSize+10:end]) {
		t.Fatal("START 保存改動了其他槽、目前戰況或 metadata 尾段")
	}
	snapshot, err := fdsave.InspectChapterSlot(result, 2)
	if err != nil || snapshot.Verified.Chapter != 1 || snapshot.Verified.RosterCount != 1 || snapshot.Verified.Currency != 1000 || snapshot.Records[0].Raw[8] != 0 || g.nativeChapterSlotBaseline == nil || g.nativeChapterSlotStarted {
		t.Fatalf("START 保存發布不完整：%+v err=%v", snapshot.Verified, err)
	}
	if err := g.saveNativeChapterSlot(2); err != nil {
		t.Fatal(err)
	}
	if again, _ := os.ReadFile(path); !bytes.Equal(again, raw) {
		t.Fatal("相同狀態重存不一致")
	}
}

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
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{camp: campaign.NewRunner(c), st: &battle.State{Units: []*battle.Unit{{Fig: 0}}}, localeCatalog: catalog}
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
	catalog, err := loadOfficialLocale("zh-Hant")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{camp: campaign.NewRunner(c), localeCatalog: catalog}
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
	attachOfficialLocale(t, g)
	options := g.currentNativeSystemOptions()
	options.Raw53AF9, options.Raw51E61, options.Raw51E62 = 1, 0, 0
	g.nativeSystemOptions = &options
	g.saveGameToSlot(2)
	if g.msg != "已存檔（槽位 3：town_ch02）" {
		t.Fatalf("save boundary message=%q", g.msg)
	}
	g.camp.Cur, g.gold, g.items = "church_ch02", 1, nil
	g.partyMembers, g.partyJoinOrder = nil, nil
	g.partyDeploy, g.partyRoster = nil, nil
	g.nativeMapHUDPersistent = battle.InitialNativeMapHUDPersistentState()
	g.nativeSystemOptions = nil
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
	g.nativePlayerStatus = &nativePlayerStatusState{phase: "status", steady: []byte{1}}
	callbackRan := false
	g.nativeLevelUpDialogue = &nativeLevelUpDialogueState{then: func() { callbackRan = true }}
	g.pendingNativeLevelUps = []pendingNativeLevelUp{{unit: &u, exp: 1}}
	g.nativeMovePlan = &battle.NativePlayerMovement{}
	g.loadGameFromSlot(2)
	if g.nativeLevelUpDialogue != nil || len(g.pendingNativeLevelUps) != 0 || g.nativeMovePlan != nil || callbackRan {
		t.Fatal("LOAD retained discarded battle owners or ran their continuation")
	}
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
	gotOptions := g.currentNativeSystemOptions()
	if gotOptions.Raw53AF9 != 1 || gotOptions.Raw51AAB != 0 ||
		gotOptions.Raw51E61 != 0 || gotOptions.Raw51E62 != 0 {
		t.Fatalf("save system options=%+v", gotOptions)
	}
	if g.st != nil || g.sel != nil || g.nativePlayerStatus != nil || g.shopMode != "" || g.churchMode != "" ||
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
		g.msg != "存檔 native HUD gate A 超出原始布林範圍" {
		t.Fatalf("invalid HUD save mutated game: gold=%d HUD=%+v msg=%q",
			g.gold, g.nativeMapHUDPersistent, g.msg)
	}
}

func TestCampaignLoadRejectsOutOfRangeNativeSystemOptionWithoutMutation(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	c := &campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{
		"town": {Type: "town"},
	}}
	invalid := 2
	raw, err := json.Marshal(saveData{Node: "town", Gold: 99, NativeRaw53AF9: &invalid})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(saveSlotPath(0)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(saveSlotPath(0), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	status := &nativePlayerStatusState{phase: "status", steady: []byte{1}}
	levelUp := &nativeLevelUpDialogueState{}
	movement := &battle.NativePlayerMovement{}
	g := &Game{camp: campaign.NewRunner(c), gold: 7, nativePlayerStatus: status,
		nativeLevelUpDialogue: levelUp, nativeMovePlan: movement}
	g.loadGameFromSlot(0)
	if g.nativeLevelUpDialogue != levelUp || g.nativeMovePlan != movement {
		t.Fatal("rejected LOAD discarded current battle owners")
	}
	if g.gold != 7 || g.nativeSystemOptions != nil || g.nativePlayerStatus != status ||
		g.msg != "存檔 native 系統設定超出原始布林範圍" {
		t.Fatalf("invalid system option mutated game: gold=%d options=%v msg=%q", g.gold, g.nativeSystemOptions, g.msg)
	}
}

func TestCampaignLoadRejectsBrokenPersistentPartyTopologyWithoutMutation(t *testing.T) {
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })
	c := &campaign.Campaign{Start: "town", Nodes: map[string]*campaign.Node{
		"town": {Type: "town"}, "preparation": {Type: "preparation"},
	}}
	broken := saveData{
		Node: "preparation", Gold: 999,
		PartyMembers: map[int]bool{0: true}, PartyJoinOrder: []int{0, 0},
		PartyDeploy: map[int]bool{0: true}, PartyRoster: map[int]battle.Unit{0: {Name: "索爾"}},
	}
	raw, err := json.Marshal(broken)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(saveSlotPath(0)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(saveSlotPath(0), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	wantRoster := map[int]battle.Unit{9: {Name: "悠妮", HP: 17}}
	g := &Game{
		camp: campaign.NewRunner(c), gold: 7,
		partyMembers: map[int]bool{9: true}, partyJoinOrder: []int{9},
		partyDeploy: map[int]bool{9: true}, partyRoster: clonePartyRoster(wantRoster),
	}
	g.loadGameFromSlot(0)
	if g.camp.NodeID() != "town" || g.gold != 7 ||
		!reflect.DeepEqual(g.partyJoinOrder, []int{9}) ||
		!reflect.DeepEqual(g.partyRoster, wantRoster) ||
		g.msg != "存檔持續隊伍的 JOIN 順序重複角色 0" {
		t.Fatalf("broken party save mutated state: node=%q gold=%d order=%v roster=%#v msg=%q",
			g.camp.NodeID(), g.gold, g.partyJoinOrder, g.partyRoster, g.msg)
	}
}

func cRunner(c *campaign.Campaign) *campaign.Runner { return campaign.NewRunner(c) }

func TestTitleJSONLoadAcceptsSavedZeroChapterAndKeepsBytes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("FD2_NATIVE_SAVE", "")
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	graph := &campaign.Campaign{Start: "saved_boundary", Nodes: map[string]*campaign.Node{
		"saved_boundary": {Type: "story"}, "other": {Type: "story"},
	}}
	writer := &Game{camp: campaign.NewRunner(graph), gold: 279}
	if err := writer.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(saveSlotPath(0))
	if err != nil || bytes.Contains(before, []byte(`"chapter"`)) {
		t.Fatalf("zero chapter save changed format: %s, %v", before, err)
	}
	cold := &Game{camp: campaign.NewRunner(graph), titlePhase: "loadslots", gold: 7, handlerChapter: 9}
	attachOfficialLocale(t, cold)
	cold.camp.Cur = "other"
	if !cold.confirmTitleLoadSlot(0) || cold.titlePhase != "" || cold.camp.NodeID() != "saved_boundary" || cold.gold != 279 || cold.handlerChapter != 0 {
		t.Fatalf("saved zero chapter refused: title=%q node=%q gold=%d chapter=%d error=%q", cold.titlePhase, cold.camp.NodeID(), cold.gold, cold.handlerChapter, cold.loadErr)
	}
	after, err := os.ReadFile(saveSlotPath(0))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("LOAD rewrote the input save", err)
	}
}

func TestTitleJSONLoadRejectsBeforeLeavingSelector(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Setenv("FD2_NATIVE_SAVE", "")
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	graph := &campaign.Campaign{Start: "safe", Nodes: map[string]*campaign.Node{"safe": {Type: "town"}}}
	for name, data := range map[string]saveData{
		"unknown node":      {Node: "missing", Chapter: 1, Gold: 999},
		"negative chapter":  {Node: "safe", Chapter: -1, Gold: 999},
		"oversized chapter": {Node: "safe", Chapter: 256, Gold: 999},
	} {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(saveSlotPath(0)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(saveSlotPath(0), raw, 0o644); err != nil {
				t.Fatal(err)
			}
			g := &Game{camp: campaign.NewRunner(graph), titlePhase: "loadslots", gold: 7, handlerChapter: 9}
			if g.confirmTitleLoadSlot(0) || g.titlePhase != "loadslots" || g.camp.NodeID() != "safe" || g.gold != 7 || g.handlerChapter != 9 {
				t.Fatalf("invalid JSON escaped selector or mutated state: title=%q node=%q gold=%d chapter=%d", g.titlePhase, g.camp.NodeID(), g.gold, g.handlerChapter)
			}
		})
	}
}
