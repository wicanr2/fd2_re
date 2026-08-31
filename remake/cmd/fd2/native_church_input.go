package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// nativeChurchMenuInput is the single typed consumer input for the verified
// 0x2D7BD selection loop and 0x3072F four-service dispatch.
type nativeChurchMenuInput struct {
	delta  int
	enter  bool
	escape bool
}

type nativeChurchStatusInput struct {
	delta  int
	enter  bool
	escape bool
}

type nativeChurchTransferInput struct {
	delta  int
	enter  bool
	escape bool
}

type nativeChurchReviveInput struct {
	delta  int
	enter  bool
	escape bool
}

type nativeChurchClassInput struct {
	delta  int
	enter  bool
	escape bool
}

func (g *Game) handleNativeChurchMenuInput(input nativeChurchMenuInput) bool {
	if g.churchMode != "menu" || g.nativeChurchUIBlocksInput() {
		return false
	}
	if input.delta != 0 {
		g.churchSel = campaign.AdvanceNativeChurchServiceSelection(g.churchSel, input.delta)
		g.resetNativeChurchUIPulse()
		return true
	}
	if input.escape {
		if !g.beginNativeChurchMenuClosing(g.leaveChurch) {
			g.leaveChurch()
		}
		return true
	}
	if !input.enter {
		return true
	}

	selected := g.churchSel
	openService := func() {
		switch selected {
		case 0: // 0x2FFA5 caller-owned roster → 0x17AED(actor)
			g.churchMode = "status_roster"
			g.churchIDs = g.churchRosterIDs()
			g.churchSel = 0
			g.churchRosterStart = 0
			g.beginNativeChurchRosterOpening()
		case 1: // 0x2F8EA raw source→destination inventory transfer
			g.churchMode = "transfer_source"
			g.churchIDs = g.churchTransferSourceIDs()
			g.churchSel = 0
			g.churchRosterStart = 0
			g.nativeChurchTextIndex = 512
			g.beginNativeChurchRosterOpening()
		case 2, 3: // 0x30DC3 revive / 0x31385 class-change
			g.churchMode = map[int]string{2: "revive", 3: "class"}[selected]
			g.churchIDs = g.churchCandidates(g.churchMode)
			g.churchSel = 0
			g.churchVerticalStart = 0
			if g.churchMode == "class" {
				g.beginNativeClassListOpening()
			} else if len(g.churchIDs) == 0 {
				g.openNativeChurchReviveEmpty()
			} else {
				g.nativeChurchTextIndex = 589
				g.beginNativeChurchReviveListOpening()
			}
		}
	}
	if !g.beginNativeChurchMenuClosing(openService) {
		openService()
	}
	return true
}

func (g *Game) handleNativeChurchStatusInput(input nativeChurchStatusInput) bool {
	if g.nativeChurchUIBlocksInput() || g.nativeClassUIBlocksInput() {
		return false
	}
	switch g.churchMode {
	case "status_roster":
		listLen := len(g.churchIDs)
		if input.delta != 0 {
			g.churchSel = campaign.AdvanceNativeTwoColumnSelection(
				g.churchSel, listLen, input.delta,
			)
		}
		g.churchRosterStart, _ = campaign.NativeTwoColumnWindow(
			listLen, g.churchSel, g.churchRosterStart,
		)
		if input.escape {
			if !g.beginNativeChurchRosterClosing(g.returnToNativeChurchMenu) {
				g.returnToNativeChurchMenu()
			}
			return true
		}
		if !input.enter {
			return true
		}
		if listLen == 0 || g.churchSel < 0 || g.churchSel >= listLen {
			return true
		}
		id := g.churchIDs[g.churchSel]
		openStatus := func() {
			if !g.beginNativeChurchStatus(id) {
				g.msg = "角色缺少原版 status/command panel provenance"
				g.returnToNativeStatusRoster()
			}
		}
		if !g.beginNativeChurchRosterClosing(openStatus) {
			openStatus()
		}
		return true
	case "status_view", "status_commands":
		if !input.enter && !input.escape {
			return true
		}
		if g.churchMode == "status_view" && len(g.churchCommandPanel) != 0 {
			if !g.beginNativeChurchStatusCommandTransition() {
				g.closeNativeChurchStatus(g.churchStatusPanel)
			}
			return true
		}
		panel := g.churchStatusPanel
		if g.churchMode == "status_commands" {
			panel = g.churchCommandPanel
		}
		g.closeNativeChurchStatus(panel)
		return true
	default:
		return false
	}
}

func (g *Game) handleNativeChurchTransferInput(input nativeChurchTransferInput) bool {
	if g.nativeChurchUIBlocksInput() || g.nativeClassUIBlocksInput() {
		return false
	}
	if g.churchMode == "transfer_full" {
		if input.enter || input.escape {
			if !g.beginNativeChurchTransferFullClosing(g.returnToNativeTransferSource) {
				g.returnToNativeTransferSource()
			}
		}
		return true
	}
	if g.churchMode != "transfer_source" && g.churchMode != "transfer_item" && g.churchMode != "transfer_dest" {
		return false
	}
	listLen := len(g.churchIDs)
	if g.churchMode == "transfer_item" {
		listLen = len(g.churchTransferItems)
	}
	if input.delta != 0 {
		g.churchSel = campaign.AdvanceNativeTwoColumnSelection(g.churchSel, listLen, input.delta)
	}
	if g.churchMode == "transfer_item" {
		g.churchItemStart, _ = campaign.NativeTwoColumnWindow(listLen, g.churchSel, g.churchItemStart)
	} else {
		g.churchRosterStart, _ = campaign.NativeTwoColumnWindow(listLen, g.churchSel, g.churchRosterStart)
	}
	if input.escape {
		switch g.churchMode {
		case "transfer_source":
			if !g.beginNativeChurchRosterClosing(g.returnToNativeChurchMenu) {
				g.returnToNativeChurchMenu()
			}
		case "transfer_item":
			if !g.beginNativeChurchTransferItemClosing(g.returnToNativeTransferSource) {
				g.returnToNativeTransferSource()
			}
		case "transfer_dest":
			if !g.beginNativeChurchRosterClosing(g.returnToNativeTransferSource) {
				g.returnToNativeTransferSource()
			}
		}
		return true
	}
	if !input.enter || listLen == 0 || g.churchSel < 0 || g.churchSel >= listLen {
		return true
	}
	switch g.churchMode {
	case "transfer_source":
		sourceID := g.churchIDs[g.churchSel]
		items := g.churchTransferItemSlots(sourceID)
		if len(items) == 0 {
			if message, ok := g.localeMessage("church.transfer.empty"); ok {
				g.msg = message
			}
			return true
		}
		openItems := func() {
			g.churchTransferSource, g.churchTransferItems = sourceID, items
			g.churchMode, g.churchSel, g.churchItemStart = "transfer_item", 0, 0
			g.beginNativeChurchTransferItemOpening()
		}
		if !g.beginNativeChurchRosterClosing(openItems) {
			openItems()
		}
	case "transfer_item":
		itemSlot := g.churchTransferItems[g.churchSel]
		openDestinations := func() {
			g.churchTransferItem = itemSlot
			g.churchIDs = g.churchTransferDestinationIDs(g.churchTransferSource)
			g.churchMode, g.churchSel, g.churchRosterStart = "transfer_dest", 0, 0
			g.nativeChurchTextIndex = 510
			g.beginNativeChurchRosterOpening()
		}
		if !g.beginNativeChurchTransferItemClosing(openDestinations) {
			openDestinations()
		}
	case "transfer_dest":
		destinationID := g.churchIDs[g.churchSel]
		apply := func() { g.applyNativeChurchTransfer(destinationID) }
		if !g.beginNativeChurchRosterClosing(apply) {
			apply()
		}
	}
	return true
}

func (g *Game) applyNativeChurchTransfer(destinationID int) {
	source := g.partyRoster[g.churchTransferSource]
	if g.churchTransferItem < 0 || g.churchTransferItem >= len(source.Inventory) {
		g.msg = "來源角色物品索引已失效"
		g.returnToNativeTransferSource()
		return
	}
	itemID := source.Inventory[g.churchTransferItem]
	successMessage, ok := g.localeMessage("church.transfer.success", itemID)
	if !ok {
		g.returnToNativeTransferSource()
		return
	}
	destination := g.partyRoster[destinationID]
	count, err := battle.NativeInventoryAvailableCount(destination.NativeInventoryFlags)
	if err != nil {
		g.msg = fmt.Sprintf("目的角色缺少原版 8-byte 物品欄旗標：%v", err)
		g.returnToNativeTransferSource()
		return
	}
	if count == 8 {
		g.churchTransferDest, g.churchMode = destinationID, "transfer_full"
		if !g.beginNativeChurchTransferFullOpening() {
			g.msg = "無法還原原版物品欄已滿提示"
			g.returnToNativeTransferSource()
		}
		return
	}
	if destinationID == g.churchTransferSource {
		if err := battle.TransferNativeInventoryItem(&source, g.churchTransferItem, &source); err != nil {
			g.msg = err.Error()
		} else {
			campaign.RecomputeEquipment(&source, g.shopItemStats)
			g.partyRoster[g.churchTransferSource] = source
			g.msg = successMessage
		}
	} else if err := battle.TransferNativeInventoryItem(&source, g.churchTransferItem, &destination); err != nil {
		g.msg = err.Error()
	} else {
		campaign.RecomputeEquipment(&source, g.shopItemStats)
		g.partyRoster[g.churchTransferSource], g.partyRoster[destinationID] = source, destination
		g.msg = successMessage
	}
	g.returnToNativeTransferSource()
}

func (g *Game) handleNativeChurchReviveInput(input nativeChurchReviveInput) bool {
	if g.nativeChurchUIBlocksInput() || g.nativeClassUIBlocksInput() {
		return false
	}
	switch g.churchMode {
	case "revive":
		if input.escape {
			if !g.beginNativeChurchReviveListClosing(g.returnToNativeChurchMenu) {
				g.returnToNativeChurchMenu()
			}
			return true
		}
		if input.delta < 0 && g.churchSel > 0 {
			g.churchSel--
		} else if input.delta > 0 && g.churchSel+1 < len(g.churchIDs) {
			g.churchSel++
		}
		g.churchVerticalStart, _ = campaign.NativeThreeRowWindow(
			len(g.churchIDs), g.churchSel, g.churchVerticalStart,
		)
		if !input.enter || len(g.churchIDs) == 0 || g.churchSel >= len(g.churchIDs) {
			return true
		}
		id := g.churchIDs[g.churchSel]
		fee, ok := g.nativeReviveFeeForUnit(g.partyRoster[id])
		if !ok {
			g.msg = "缺少原版復活費率資料"
			return true
		}
		openConfirmation := func() {
			g.churchReviveID, g.churchReviveFee = id, fee
			g.churchMode, g.churchSel = "revive_confirm", 0
			g.beginNativeChurchReviveConfirmationOpening()
		}
		if !g.beginNativeChurchReviveListClosing(openConfirmation) {
			openConfirmation()
		}
		return true
	case "revive_confirm":
		if input.escape {
			if !g.beginNativeChurchReviveConfirmationClosing(g.returnToNativeReviveList) {
				g.returnToNativeReviveList()
			}
			return true
		}
		if input.delta != 0 {
			g.churchSel = campaign.AdvanceNativeClassConfirmation(g.churchSel, input.delta)
		}
		if !input.enter {
			return true
		}
		if g.churchSel != 0 {
			if !g.beginNativeChurchReviveConfirmationClosing(g.returnToNativeReviveList) {
				g.returnToNativeReviveList()
			}
			return true
		}
		apply := func() {
			if g.gold < g.churchReviveFee {
				g.churchMode = "revive_insufficient"
				return
			}
			if !g.reviveChurchUnit(g.churchReviveID) {
				g.returnToNativeReviveList()
				return
			}
			success := campaign.PlanNativeChurchReviveSuccess()
			g.playBGMCount(fmt.Sprintf("FDMUS_%03d", success.StartMusicTrack), success.MusicLoopCount)
			after := func() {
				g.playBGMCount(fmt.Sprintf("FDMUS_%03d", success.ReturnMusicTrack), success.MusicLoopCount)
				g.returnToNativeReviveList()
			}
			if !g.beginNativeChurchReviveSuccess(after) {
				after()
			}
		}
		if !g.beginNativeChurchReviveChoiceClosing(apply) {
			apply()
		}
		return true
	case "revive_empty", "revive_insufficient":
		if input.enter || input.escape {
			after := g.returnToNativeReviveList
			if g.churchMode == "revive_empty" {
				after = g.returnToNativeChurchMenu
			}
			if !g.beginNativeChurchReviveMessageClosing(after) {
				after()
			}
		}
		return true
	default:
		return false
	}
}

func (g *Game) handleNativeChurchClassInput(input nativeChurchClassInput) bool {
	if g.nativeChurchUIBlocksInput() || g.nativeClassUIBlocksInput() {
		return false
	}
	switch g.churchMode {
	case "class":
		if input.escape {
			if !g.beginNativeClassListClosing(g.returnToNativeChurchMenu) {
				g.returnToNativeChurchMenu()
			}
			return true
		}
		if input.delta < 0 && g.churchSel > 0 {
			g.churchSel--
		} else if input.delta > 0 && g.churchSel+1 < len(g.churchIDs) {
			g.churchSel++
		}
		g.churchVerticalStart, _ = campaign.NativeThreeRowWindow(
			len(g.churchIDs), g.churchSel, g.churchVerticalStart,
		)
		if !input.enter || len(g.churchIDs) == 0 || g.churchSel >= len(g.churchIDs) {
			return true
		}
		id := g.churchIDs[g.churchSel]
		unit := g.partyRoster[id]
		target, ok := campaign.NativeClassChangeTarget(&unit, g.classChangeTable)
		if !ok {
			g.msg = "缺少原版轉職目標資料"
			return true
		}
		openConfirmation := func() {
			g.churchClassID = id
			g.churchBranches = []campaign.ClassChangeBranch{target}
			g.churchMode, g.churchSel = "class_confirm", 0
			g.beginNativeClassConfirmationOpening()
		}
		if !g.beginNativeClassListClosing(openConfirmation) {
			openConfirmation()
		}
		return true
	case "class_confirm":
		if input.escape {
			if !g.beginNativeClassConfirmationClosing(g.returnToNativeClassList) {
				g.returnToNativeClassList()
			}
			return true
		}
		if input.delta != 0 {
			g.churchSel = campaign.AdvanceNativeClassConfirmation(g.churchSel, input.delta)
		}
		if !input.enter {
			return true
		}
		if g.churchSel != 0 {
			if !g.beginNativeClassConfirmationClosing(g.returnToNativeClassList) {
				g.returnToNativeClassList()
			}
			return true
		}
		apply := func() {
			if g.applyChurchClassChange(0) {
				g.beginNativeClassListOpening()
				return
			}
			g.returnToNativeClassList()
		}
		if !g.beginNativeClassConfirmationClosing(apply) {
			apply()
		}
		return true
	default:
		return false
	}
}
