package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) setupNativeShopTransfer() bool {
	if len(g.partyJoinOrder) == 0 {
		return false
	}
	for _, id := range g.partyJoinOrder {
		unit, ok := g.partyRoster[id]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey ||
			battle.ValidateNativeInventoryProjection(&unit) != nil {
			return false
		}
	}
	g.nativeShopTransferSource = -1
	g.nativeShopTransferItem = -1
	g.nativeShopTransferItems = nil
	g.nativeShopTransferDest = -1
	g.nativeShopTransferIDs = nil
	g.nativeShopTransferSel = 0
	g.nativeShopTransferTop = 0
	return g.returnToNativeShopTransferLoop()
}

func (g *Game) returnToNativeShopTransferLoop() bool {
	g.nativeShopMode = "transfer_intro"
	g.nativeShopTransferSource = -1
	g.nativeShopTransferItem = -1
	g.nativeShopTransferItems = nil
	g.nativeShopTransferDest = -1
	g.nativeShopTransferIDs = nil
	g.nativeShopTransferSel = 0
	g.nativeShopTransferTop = 0
	return g.beginNativeShopTransferMessageOpening()
}

// beginNativeShopTransferDestinationCancel 是目的角色名冊 Escape 的正式擁有者。
// 它只安排原版五幀名冊收合；來源提示與六幀展開必須等收合的 after 才發布。
// 任一合成步驟失敗時不直接改模式，讓呼叫端依失敗即關閉政策停止流程。
func (g *Game) beginNativeShopTransferDestinationCancel() bool {
	if g.nativeShopMode != "transfer_dest" || g.nativeShopUIJob != nil {
		return false
	}
	after := func() {
		if !g.returnToNativeShopTransferLoop() {
			g.nativeShopMode = ""
			g.msg = "原版商店 transfer source prompt 無法還原"
		}
	}
	return g.beginNativeShopTransferRosterClosing(after)
}

func (g *Game) composeNativeShopTransferMessage() ([]byte, bool) {
	stable, stableOK := g.composeNativeShopStable()
	_, portrait, portraitID, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	switch g.nativeShopMode {
	case "transfer_intro":
		if g.localeID != "" && g.localeID != "zh-Hant" {
			frame, err := g.composeLocalizedNativeShopPlainMessage(
				stable, portrait, portraitID, "shop.transfer.source_prompt",
			)
			return frame, err == nil
		}
		frame, err := campaign.ComposeNativeShopTransferMessage(
			stable, g.nativeClassUI.dialogue, portrait, portraitID,
			g.nativeClassUI.strings, g.nativeClassUI.font,
			campaign.NativeShopTransferSourcePrompt, -1,
		)
		return frame, err == nil
	case "transfer_dest_prompt":
		if g.localeID != "" && g.localeID != "zh-Hant" {
			frame, err := g.composeLocalizedNativeShopPlainMessage(
				stable, portrait, portraitID, "shop.transfer.destination_prompt",
			)
			return frame, err == nil
		}
		frame, err := campaign.ComposeNativeShopTransferMessage(
			stable, g.nativeClassUI.dialogue, portrait, portraitID,
			g.nativeClassUI.strings, g.nativeClassUI.font,
			campaign.NativeShopTransferDestinationPrompt, -1,
		)
		return frame, err == nil
	case "transfer_empty":
		unit, ok := g.partyRoster[g.nativeShopTransferSource]
		if !ok || !unit.HasNativeIdentity {
			return nil, false
		}
		if g.localeID != "" && g.localeID != "zh-Hant" {
			name, err := g.localeEntities.CharacterName(unit.NativeIdentity)
			if err != nil {
				return nil, false
			}
			frame, err := g.composeLocalizedNativeShopPlainMessage(
				stable, portrait, portraitID, "shop.transfer.empty_source", name,
			)
			return frame, err == nil
		}
		frame, err := campaign.ComposeNativeShopTransferMessage(
			stable, g.nativeClassUI.dialogue, portrait, portraitID,
			g.nativeClassUI.strings, g.nativeClassUI.font,
			campaign.NativeShopTransferEmptySource,
			unit.NativeIdentity+1,
		)
		return frame, err == nil
	case "transfer_full":
		unit, ok := g.partyRoster[g.nativeShopTransferDest]
		if !ok || !unit.HasNativeIdentity {
			return nil, false
		}
		if g.localeID != "" && g.localeID != "zh-Hant" {
			name, err := g.localeEntities.CharacterName(unit.NativeIdentity)
			if err != nil {
				return nil, false
			}
			frame, err := g.composeLocalizedNativeShopPlainMessage(
				stable, portrait, portraitID, "shop.recipient.full", name,
			)
			return frame, err == nil
		}
		frame, err := campaign.ComposeNativeShopPurchaseRecipientFull(
			stable, g.nativeClassUI.dialogue, portrait, portraitID,
			g.nativeClassUI.strings, g.nativeClassUI.font,
			g.nativeShopVariant, unit.NativeIdentity+1,
		)
		return frame, err == nil
	}
	return nil, false
}

func (g *Game) beginNativeShopTransferMessageOpening() bool {
	final, ok := g.composeNativeShopTransferMessage()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListOpeningFrames(stable, final)
	if err != nil || len(frames) != 6 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeShopTransferMessageClosing(after func()) bool {
	final, ok := g.composeNativeShopTransferMessage()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(stable, final)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		frames: frames, restore: stable, after: after,
	}
	return true
}

func (g *Game) openNativeShopTransferSourceRoster() {
	g.nativeShopMode = "transfer_source"
	g.nativeShopTransferIDs = append(
		g.nativeShopTransferIDs[:0], g.partyJoinOrder...,
	)
	g.nativeShopTransferSel = 0
	g.nativeShopTransferTop = 0
	g.beginNativeShopTransferRosterOpening()
}

func (g *Game) openNativeShopTransferDestinationRoster() {
	// 已證實：0x2f8ea 再以完整隊伍呼叫 0x2e6b8，沒有排除來源角色；
	// 選自己會依原始背包順序執行 remove→append，並以未裝備狀態放到尾端。
	g.nativeShopMode = "transfer_dest"
	g.nativeShopTransferIDs = append(
		g.nativeShopTransferIDs[:0], g.partyJoinOrder...,
	)
	g.nativeShopTransferSel = 0
	g.nativeShopTransferTop = 0
	g.beginNativeShopTransferRosterOpening()
}

func (g *Game) composeNativeShopTransferRoster() ([]byte, bool) {
	if g.nativeShopMode != "transfer_source" &&
		g.nativeShopMode != "transfer_dest" {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(g.nativeShopTransferIDs), g.nativeShopTransferSel,
		g.nativeShopTransferTop,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopTransferTop = start
	rows := make([]campaign.NativeRosterRow, 0, visible)
	identities := make([]int, 0, visible)
	for i := 0; i < visible; i++ {
		unit, ok := g.partyRoster[g.nativeShopTransferIDs[start+i]]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey {
			return nil, false
		}
		sprite, err := g.nativeClassUI.units.SpriteFor(
			unit.MapSelectorKey, 0, 0,
		)
		if err != nil {
			return nil, false
		}
		rows = append(rows, campaign.NativeRosterRow{
			Sprite: sprite, NameTextIndex: unit.NativeIdentity + 1,
		})
		identities = append(identities, unit.NativeIdentity)
	}
	stable, stableOK := g.composeNativeShopStable()
	assets, _, _, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	var frame []byte
	var err error
	if g.localeID != "" && g.localeID != "zh-Hant" {
		frame, err = g.composeLocalizedNativeRoster(
			stable, assets.Panel, rows, identities, g.nativeShopTransferSel-start,
		)
	} else {
		frame, err = campaign.ComposeNativeRosterFrame(
			stable, assets.Panel, rows, g.nativeShopTransferSel-start,
			g.nativeClassUI.strings, g.nativeClassUI.font,
		)
	}
	return frame, err == nil
}

func (g *Game) beginNativeShopTransferRosterOpening() bool {
	final, ok := g.composeNativeShopTransferRoster()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListOpeningFrames(stable, final)
	if err != nil || len(frames) != 6 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeShopTransferRosterClosing(after func()) bool {
	final, ok := g.composeNativeShopTransferRoster()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(stable, final)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		frames: frames, restore: stable, after: after,
	}
	return true
}

func nativeShopTransferItemSlots(unit battle.Unit) ([]int, bool) {
	if err := battle.ValidateNativeInventoryProjection(&unit); err != nil {
		return nil, false
	}
	slots := make([]int, 0, len(unit.Inventory))
	for compact := range unit.Inventory {
		eligible, err := battle.NativeInventoryCompactEligible(
			unit.NativeInventoryFlags, unit.InventorySlots, compact,
		)
		if err != nil {
			return nil, false
		}
		if eligible {
			slots = append(slots, compact)
		}
	}
	return slots, true
}

func (g *Game) openNativeShopTransferItems() bool {
	unit, ok := g.partyRoster[g.nativeShopTransferSource]
	if !ok {
		return false
	}
	items, ok := nativeShopTransferItemSlots(unit)
	if !ok || len(items) == 0 {
		return false
	}
	g.nativeShopTransferItems = items
	g.nativeShopTransferSel = 0
	g.nativeShopTransferTop = 0
	g.nativeShopMode = "transfer_items"
	return g.beginNativeShopTransferItemsOpening()
}

func (g *Game) composeNativeShopTransferItems() ([]byte, bool) {
	if g.nativeShopMode != "transfer_items" ||
		len(g.nativeShopTransferItems) == 0 {
		return nil, false
	}
	unit, ok := g.partyRoster[g.nativeShopTransferSource]
	if !ok {
		return nil, false
	}
	itemIDs := make([]int, len(g.nativeShopTransferItems))
	for index, compact := range g.nativeShopTransferItems {
		if compact < 0 || compact >= len(unit.Inventory) {
			return nil, false
		}
		itemIDs[index] = unit.Inventory[compact]
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(itemIDs), g.nativeShopTransferSel, g.nativeShopTransferTop,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopTransferTop = start
	stable, stableOK := g.composeNativeShopStable()
	assets, _, _, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	var frame []byte
	var err error
	if g.localeID != "" && g.localeID != "zh-Hant" {
		frame, err = g.composeLocalizedNativeShopItemIDs(
			stable, assets, itemIDs, start, g.nativeShopTransferSel,
			battle.NativeFacilityThreeQuarterPrice,
		)
	} else {
		frame, err = campaign.ComposeNativeShopItemListFrame(
			stable, assets, g.nativeShopUI.itemAssets,
			itemIDs, start, g.nativeShopTransferSel,
			g.nativeShopUI.effectRows,
			battle.NativeFacilityThreeQuarterPrice,
		)
	}
	return frame, err == nil
}

func (g *Game) beginNativeShopTransferItemsOpening() bool {
	final, ok := g.composeNativeShopTransferItems()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListOpeningFrames(stable, final)
	if err != nil || len(frames) != 6 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeShopTransferItemsClosing(after func()) bool {
	final, ok := g.composeNativeShopTransferItems()
	if !ok {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassListClosingFrames(stable, final)
	if err != nil || len(frames) != 5 {
		return false
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		frames: frames, restore: stable, after: after,
	}
	return true
}

func (g *Game) applyNativeShopTransfer(destinationID int) bool {
	sourceID := g.nativeShopTransferSource
	source, sourceOK := g.partyRoster[sourceID]
	destination, destinationOK := g.partyRoster[destinationID]
	if !sourceOK || !destinationOK ||
		g.nativeShopTransferItem < 0 ||
		g.nativeShopTransferItem >= len(source.Inventory) {
		return false
	}
	if sourceID == destinationID {
		unit := cloneNativeShopUnit(source)
		if err := battle.TransferNativeInventoryItem(
			&unit, g.nativeShopTransferItem, &unit,
		); err != nil {
			return false
		}
		campaign.RecomputeEquipment(&unit, g.shopItemStats)
		g.partyRoster[sourceID] = unit
		return true
	}
	source = cloneNativeShopUnit(source)
	destination = cloneNativeShopUnit(destination)
	if err := battle.TransferNativeInventoryItem(
		&source, g.nativeShopTransferItem, &destination,
	); err != nil {
		return false
	}
	campaign.RecomputeEquipment(&source, g.shopItemStats)
	g.partyRoster[sourceID] = source
	g.partyRoster[destinationID] = destination
	return true
}
