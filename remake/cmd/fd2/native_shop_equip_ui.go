package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) handleNativeShopEquipInput(input nativeShopTransferInput) bool {
	switch g.nativeShopMode {
	case "equip_roster":
		count := len(g.partyJoinOrder)
		if input.delta != 0 {
			g.nativeShopEquipUnitSel = campaign.AdvanceNativeTwoColumnSelection(
				g.nativeShopEquipUnitSel, count, input.delta,
			)
			g.nativeShopEquipRosterTop, _ = campaign.NativeTwoColumnWindow(
				count, g.nativeShopEquipUnitSel, g.nativeShopEquipRosterTop,
			)
		}
		if input.escape {
			openMenu := func() {
				g.nativeShopMode = "menu"
				g.nativeShopServiceSel = 2
				g.beginNativeShopServiceOpening()
			}
			if !g.beginNativeShopEquipRosterClosing(openMenu) {
				openMenu()
			}
			return true
		}
		if input.enter && count != 0 {
			openPanel := func() {
				if !g.openNativeShopEquipPanel() {
					g.nativeShopMode = ""
					g.msg = "原版商店 equip item panel 無法還原"
				}
			}
			if !g.beginNativeShopEquipRosterClosing(openPanel) {
				openPanel()
			}
		}
		return true
	case "equip_panel":
		if g.nativeShopEquipPanelBlocksInput() {
			return true
		}
		if input.escape {
			g.beginNativeShopEquipPanelClose()
			return true
		}
		_, unit, ok := g.nativeShopEquipUnit()
		if !ok {
			return true
		}
		rawSlots := nativeItemRawSlots(&unit)
		if len(rawSlots) != 0 && input.delta != 0 {
			scanCode := map[int]int{-2: 72, 2: 80, -1: 75, 1: 77}[input.delta]
			selected, _, err := battle.AdvanceNativeItemSelector(
				g.itemSel, len(rawSlots), scanCode, false, 0,
			)
			if err == nil && selected != g.itemSel {
				g.itemSel = selected
				g.refreshNativeItemPanelMode(&unit, true)
			}
		}
		if input.enter && !g.applyNativeShopEquipSelection() {
			g.msg = "原版商店 equip transaction 缺少 raw 對映"
		}
		return true
	default:
		return false
	}
}

func (g *Game) setupNativeShopEquipRoster() bool {
	if len(g.partyJoinOrder) == 0 {
		return false
	}
	for _, id := range g.partyJoinOrder {
		unit, ok := g.partyRoster[id]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey ||
			len(unit.InventorySlots) != 8 ||
			len(unit.NativeInventoryFlags) != 8 {
			return false
		}
	}
	g.nativeShopMode = "equip_roster"
	g.nativeShopEquipUnitSel = 0
	g.nativeShopEquipRosterTop = 0
	return true
}

func (g *Game) returnToNativeShopEquipRoster() {
	g.clearNativeItemPanel()
	if len(g.partyJoinOrder) == 0 {
		g.nativeShopMode = ""
		return
	}
	if g.nativeShopEquipUnitSel < 0 {
		g.nativeShopEquipUnitSel = 0
	}
	if g.nativeShopEquipUnitSel >= len(g.partyJoinOrder) {
		g.nativeShopEquipUnitSel = len(g.partyJoinOrder) - 1
	}
	g.nativeShopMode = "equip_roster"
	g.nativeShopEquipRosterTop, _ = campaign.NativeTwoColumnWindow(
		len(g.partyJoinOrder), g.nativeShopEquipUnitSel,
		g.nativeShopEquipRosterTop,
	)
	g.beginNativeShopEquipRosterOpening()
}

func (g *Game) nativeShopEquipUnit() (int, battle.Unit, bool) {
	if g.nativeShopEquipUnitSel < 0 ||
		g.nativeShopEquipUnitSel >= len(g.partyJoinOrder) {
		return 0, battle.Unit{}, false
	}
	id := g.partyJoinOrder[g.nativeShopEquipUnitSel]
	unit, ok := g.partyRoster[id]
	return id, unit, ok
}

func (g *Game) composeNativeShopEquipRoster() ([]byte, bool) {
	if g.nativeShopMode != "equip_roster" {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(g.partyJoinOrder), g.nativeShopEquipUnitSel,
		g.nativeShopEquipRosterTop,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopEquipRosterTop = start
	rows := make([]campaign.NativeRosterRow, 0, visible)
	for i := 0; i < visible; i++ {
		unit, ok := g.partyRoster[g.partyJoinOrder[start+i]]
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
	}
	stable, stableOK := g.composeNativeShopStable()
	assets, _, _, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	frame, err := campaign.ComposeNativeRosterFrame(
		stable, assets.Panel, rows, g.nativeShopEquipUnitSel-start,
		g.nativeClassUI.strings, g.nativeClassUI.font,
	)
	return frame, err == nil
}

func (g *Game) beginNativeShopEquipRosterOpening() bool {
	final, ok := g.composeNativeShopEquipRoster()
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

func (g *Game) beginNativeShopEquipRosterClosing(after func()) bool {
	final, ok := g.composeNativeShopEquipRoster()
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

func (g *Game) openNativeShopEquipPanel() bool {
	_, unit, ok := g.nativeShopEquipUnit()
	if !ok {
		return false
	}
	g.itemSel = 0
	if !g.prepareNativeItemPanelMode(&unit, true) {
		return false
	}
	g.nativeShopMode = "equip_panel"
	g.itemAnimStep = 0
	g.itemClosing = false
	return true
}

func (g *Game) beginNativeShopEquipPanelClose() {
	if g.nativeItemPanel == nil {
		g.returnToNativeShopEquipRoster()
		return
	}
	g.itemAnimStep = 0
	g.itemClosing = true
}

func (g *Game) stepNativeShopEquipPanelLifecycle() {
	if g.nativeShopMode != "equip_panel" || g.nativeItemPanel == nil {
		return
	}
	if g.itemAnimStep < 11 {
		g.itemAnimStep++
		return
	}
	if g.itemClosing {
		g.returnToNativeShopEquipRoster()
	}
}

func (g *Game) nativeShopEquipPanelBlocksInput() bool {
	return g.nativeShopMode == "equip_panel" &&
		(g.itemClosing || g.itemAnimStep < 11)
}

func (g *Game) applyNativeShopEquipSelection() bool {
	unitID, unit, ok := g.nativeShopEquipUnit()
	if !ok {
		return false
	}
	rawSlots := nativeItemRawSlots(&unit)
	if len(rawSlots) == 0 {
		g.beginNativeShopEquipPanelClose()
		return true
	}
	if g.itemSel < 0 || g.itemSel >= len(rawSlots) {
		return false
	}
	rawSlot := rawSlots[g.itemSel]
	itemID := unit.InventorySlots[rawSlot]
	itemType, typeOK := g.shopItemTypes[itemID]
	if !typeOK || !campaign.CanEquip(
		unit.ClassID, itemType, g.shopEquipTypes,
	) {
		// 0x1c068 jumps directly back to 0x1c022: no feedback owner.
		return true
	}
	if err := campaign.EquipNativeCompactSlot(
		&unit, g.itemSel, g.shopItemStats,
	); err != nil {
		return false
	}

	// 0x1c084..0x1c0cc 會在裝備成功後原地重畫面板。先在候選 unit 上完成
	// 所有 renderer 前置；若重建失敗，既有 panel 與 persistent roster 都不得改變。
	oldPanel := g.nativeItemPanel
	oldBase := g.nativeItemPanelBase
	oldRecord := g.nativeItemPanelRecord
	oldAssets := g.nativeItemPanelAssets
	oldEffectRows := g.nativeItemEffectRows
	oldSelection := g.itemSel
	if !g.rebuildNativeItemPanelContents(&unit, true) {
		g.nativeItemPanel = oldPanel
		g.nativeItemPanelBase = oldBase
		g.nativeItemPanelRecord = oldRecord
		g.nativeItemPanelAssets = oldAssets
		g.nativeItemEffectRows = oldEffectRows
		g.itemSel = oldSelection
		return false
	}
	g.partyRoster[unitID] = cloneNativeShopUnit(unit)
	return true
}
