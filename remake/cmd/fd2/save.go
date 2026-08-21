// save.go — 存檔/讀檔(自有 JSON 格式,doc19;非破解原版 FD2.SAV)。
// 語意:存「campaign 節點邊界」進度(目前節點/旗標/金幣/道具),戰鬥中存檔=回到該戰鬥節點重開。
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// savePath/saveSlotPath 存檔位置:$XDG_DATA_HOME/fd2_re/fd2_save_N.json；四槽
// selector 對應 native 0x30550 的 UI contract，但 JSON 不是原版 FD2.SAV ABI。
func savePath() string { return saveSlotPath(0) }

func saveSlotPath(slot int) string {
	if slot < 0 || slot > 3 {
		slot = 0
	}
	if slot == 0 {
		// Preserve the pre-selector single-save file as slot 1.
		return userDataPath("fd2_save.json")
	}
	return userDataPath(fmt.Sprintf("fd2_save_%d.json", slot))
}

// writeSaveFile replaces the save in one rename. Campaign progress is only
// persisted at node boundaries, so a truncated JSON file must never turn a
// valid town/preparation save into an unreadable slot after a process stop.
func writeSaveFile(path string, raw []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".fd2-save-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

type saveData struct {
	Node           string              `json:"node"`
	Flags          map[string]bool     `json:"flags"`
	Gold           int                 `json:"gold"`
	Items          []string            `json:"items"`
	PartyMembers   map[int]bool        `json:"party_members,omitempty"`
	PartyJoinOrder []int               `json:"party_join_order,omitempty"`
	PartyDeploy    map[int]bool        `json:"party_deploy,omitempty"`
	PartyRoster    map[int]battle.Unit `json:"party_roster,omitempty"`
	Chapter        int                 `json:"chapter,omitempty"`
	NativeHUDGateA *int                `json:"native_hud_gate_a,omitempty"`
}

func (g *Game) saveGame() { g.saveGameToSlot(0) }

func (g *Game) saveGameToSlot(slot int) {
	if g.camp == nil {
		g.msg = "存檔:僅 campaign 模式支援(FD2_CAMPAIGN=1)"
		return
	}
	if n := g.camp.Node(); n != nil && n.Type == "cutscene" && ((n.HandlerBinding != "" && g.st != nil) || strings.HasPrefix(g.camp.NodeID(), "postbattle_")) {
		// Post-battle handlers intentionally retain the completed canonical battle
		// array for slot predicates, rewards, SPAWN, ACT and sync_party. The save
		// format is node-boundary-only and does not serialize that transient array;
		// saving this node would reload into a guaranteed fail-closed context. This
		// applies equally to an unbound node: it must not create a fake save that
		// appears to have crossed the persistence boundary.
		g.msg = "戰後演出進行中，請在下一個節點存檔"
		return
	}
	g.captureNativeMapHUDPersistence()
	d := saveData{
		Node: g.camp.Cur, Flags: g.camp.Flags, Gold: g.gold, Items: g.items,
		PartyMembers: g.partyMembers, PartyJoinOrder: g.partyJoinOrder,
		PartyDeploy: g.partyDeploy,
		PartyRoster: g.partyRoster, Chapter: g.handlerChapter,
	}
	if g.nativeMapHUDPersistent.HasDisplayGateA {
		gateA := int(g.nativeMapHUDPersistent.DisplayGateA)
		d.NativeHUDGateA = &gateA
	}
	raw, err := json.MarshalIndent(d, "", " ")
	if err != nil {
		return
	}
	if writeSaveFile(saveSlotPath(slot), raw) == nil {
		g.msg = fmt.Sprintf("已存檔(槽位%d：%s)", slot+1, g.camp.Cur)
	}
}

func (g *Game) loadGame() { g.loadGameFromSlot(0) }

func (g *Game) loadGameFromSlot(slot int) {
	if g.camp == nil {
		return
	}
	raw, err := os.ReadFile(saveSlotPath(slot))
	if err != nil {
		g.msg = "無存檔"
		return
	}
	var d saveData
	if json.Unmarshal(raw, &d) != nil {
		return
	}
	if _, ok := g.camp.C.Nodes[d.Node]; !ok {
		g.msg = "存檔節點不存在:" + d.Node
		return
	}
	if d.NativeHUDGateA != nil && (*d.NativeHUDGateA < 0 || *d.NativeHUDGateA > 0xff) {
		g.msg = "存檔 native HUD gate A 超出原始 byte 範圍"
		return
	}
	g.captureNativeMapHUDPersistence()
	g.camp.Cur = d.Node
	g.camp.Flags = d.Flags
	g.gold, g.items = d.Gold, d.Items
	g.partyMembers, g.partyJoinOrder = d.PartyMembers, d.PartyJoinOrder
	g.partyDeploy = d.PartyDeploy
	g.partyRoster, g.handlerChapter = d.PartyRoster, d.Chapter
	if d.NativeHUDGateA != nil {
		g.restoreNativeMapHUDGateA(byte(*d.NativeHUDGateA))
	}
	// 節點邊界存檔不保留舊戰場陣列；先清除它也可避免 enterNode 以讀檔前的
	// gate A 覆蓋剛還原的存檔值。教會選單、候選與 indexed 工作同屬未序列化
	// 暫態，必須在進入存檔節點前一併丟棄。
	g.dialog, g.st, g.sel = nil, nil, nil
	g.nativeChapterRestore = nil
	g.clearChurchTransientStateForLoad()
	g.clearShopTransientStateForLoad()
	g.enterNode()
	g.msg = fmt.Sprintf("已讀檔(槽位%d：%s)", slot+1, d.Node)
}

func (g *Game) clearShopTransientStateForLoad() {
	g.nativeShopUIJob = nil
	g.resetNativeShopUIPulse()
	g.nativeShopVariant = 0
	g.nativeShopMode = ""
	g.nativeShopServiceSel = 0
	g.nativeShopItemStart = 0
	g.nativeShopConfirmSel = 0
	g.nativeShopRecipientStart = 0
	g.nativeShopRecipientCycle = 0
	g.nativeShopEquipSel = 0
	g.nativeShopPendingUnit = battle.Unit{}
	g.nativeShopHasPendingUnit = false
	g.nativeShopPendingGold = 0
	g.nativeShopSellRosterTop = 0
	g.nativeShopSellItemTop = 0
	g.nativeShopSellConfirmSel = 0
	g.nativeShopSellItemIDs = nil
	g.nativeShopEquipRosterTop = 0
	g.nativeShopEquipUnitSel = 0
	g.nativeShopTransferSource = -1
	g.nativeShopTransferItem = -1
	g.nativeShopTransferItems = nil
	g.nativeShopTransferDest = -1
	g.nativeShopTransferIDs = nil
	g.nativeShopTransferSel = 0
	g.nativeShopTransferTop = 0

	g.shopSel = 0
	g.shopRecipientSel = 0
	g.shopRecipients = nil
	g.shopPicking = false
	g.shopPending = campaign.Good{}
	g.shopEquipPrompt = false
	g.shopEquipUnit = 0
	g.shopEquipSlot = 0
	g.shopMode = ""
	g.shopSellPicking = false
	g.shopSellUnitSel = 0
	g.shopSellSlotSel = 0
	g.clearNativeItemPanel()
}

func (g *Game) clearChurchTransientStateForLoad() {
	g.nativeClassUIJob = nil
	g.nativeChurchUIJob = nil
	g.resetNativeClassUIPulse()
	g.resetNativeChurchUIPulse()
	g.nativeChurchTextIndex = 0
	g.churchSel = 0
	g.churchMode = ""
	g.churchIDs = nil
	g.churchRosterStart = 0
	g.churchVerticalStart = 0
	g.churchStatusID = -1
	g.churchStatusPanel = nil
	g.churchCommandPanel = nil
	g.churchItemStart = 0
	g.churchTransferSource = -1
	g.churchTransferItem = -1
	g.churchTransferItems = nil
	g.churchTransferDest = -1
	g.churchReviveID = -1
	g.churchReviveFee = 0
	g.churchClassID = -1
	g.churchBranches = nil
}

func (g *Game) restoreNativeMapHUDGateA(gateA byte) {
	if !g.nativeMapHUDPersistent.HasAnchorX {
		g.nativeMapHUDPersistent = battle.InitialNativeMapHUDPersistentState()
	}
	g.nativeMapHUDPersistent.RestoreSavedGateA(gateA)
}
