package main

import (
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func (g *Game) setupNativeShopSellRoster() bool {
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
	g.nativeShopMode = "sell_roster"
	g.shopSellUnitSel = 0
	g.nativeShopSellRosterTop = 0
	return true
}

func (g *Game) returnToNativeShopSellRoster() {
	if len(g.partyJoinOrder) == 0 {
		g.nativeShopMode = ""
		return
	}
	if g.shopSellUnitSel < 0 {
		g.shopSellUnitSel = 0
	}
	if g.shopSellUnitSel >= len(g.partyJoinOrder) {
		g.shopSellUnitSel = len(g.partyJoinOrder) - 1
	}
	g.nativeShopMode = "sell_roster"
	g.nativeShopSellItemIDs = nil
	g.nativeShopSellRosterTop, _ = campaign.NativeTwoColumnWindow(
		len(g.partyJoinOrder), g.shopSellUnitSel,
		g.nativeShopSellRosterTop,
	)
	g.beginNativeShopSellRosterOpening()
}

func (g *Game) composeNativeShopSellRoster() ([]byte, bool) {
	if g.nativeShopMode != "sell_roster" ||
		g.shopSellUnitSel < 0 ||
		g.shopSellUnitSel >= len(g.partyJoinOrder) {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(g.partyJoinOrder), g.shopSellUnitSel,
		g.nativeShopSellRosterTop,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopSellRosterTop = start
	rows := make([]campaign.NativeRosterRow, 0, visible)
	identities := make([]int, 0, visible)
	for i := 0; i < visible; i++ {
		unit, ok := g.partyRoster[g.partyJoinOrder[start+i]]
		if !ok || !unit.HasNativeIdentity || !unit.HasMapSelectorKey {
			return nil, false
		}
		sprite, err := g.nativeClassUI.units.SpriteFor(
			unit.MapSelectorKey, 0, g.nativeShopSellRosterCycle,
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
			stable, assets.Panel, rows, identities, g.shopSellUnitSel-start,
		)
	} else {
		frame, err = campaign.ComposeNativeRosterFrame(
			stable, assets.Panel, rows, g.shopSellUnitSel-start,
			g.nativeClassUI.strings, g.nativeClassUI.font,
		)
	}
	return frame, err == nil
}

func (g *Game) beginNativeShopSellRosterOpening() bool {
	final, ok := g.composeNativeShopSellRoster()
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

func (g *Game) beginNativeShopSellRosterClosing(after func()) bool {
	final, ok := g.composeNativeShopSellRoster()
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

func (g *Game) nativeShopSellUnit() (int, battle.Unit, bool) {
	if g.shopSellUnitSel < 0 ||
		g.shopSellUnitSel >= len(g.partyJoinOrder) {
		return 0, battle.Unit{}, false
	}
	id := g.partyJoinOrder[g.shopSellUnitSel]
	unit, ok := g.partyRoster[id]
	return id, unit, ok
}

func nativeShopActiveItemIDs(unit battle.Unit) ([]int, bool) {
	if len(unit.InventorySlots) != 8 ||
		len(unit.NativeInventoryFlags) != 8 {
		return nil, false
	}
	items := make([]int, 0, 8)
	for slot, flag := range unit.NativeInventoryFlags {
		if flag&0x80 == 0 {
			items = append(items, unit.InventorySlots[slot])
		}
	}
	return items, true
}

func (g *Game) setupNativeShopSellItems() bool {
	_, unit, ok := g.nativeShopSellUnit()
	if !ok {
		return false
	}
	items, ok := nativeShopActiveItemIDs(unit)
	if !ok || len(items) == 0 {
		return false
	}
	g.nativeShopSellItemIDs = items
	g.shopSellSlotSel = 0
	g.nativeShopSellItemTop = 0
	g.nativeShopMode = "sell_items"
	return true
}

func (g *Game) composeNativeShopSellItems() ([]byte, bool) {
	if g.nativeShopMode != "sell_items" ||
		len(g.nativeShopSellItemIDs) == 0 ||
		g.shopSellSlotSel < 0 ||
		g.shopSellSlotSel >= len(g.nativeShopSellItemIDs) {
		return nil, false
	}
	start, visible := campaign.NativeTwoColumnWindow(
		len(g.nativeShopSellItemIDs), g.shopSellSlotSel,
		g.nativeShopSellItemTop,
	)
	if visible == 0 {
		return nil, false
	}
	g.nativeShopSellItemTop = start
	stable, stableOK := g.composeNativeShopStable()
	assets, _, _, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	frame, err := campaign.ComposeNativeShopItemListFrame(
		stable, assets, g.nativeShopUI.itemAssets,
		g.nativeShopSellItemIDs, start, g.shopSellSlotSel,
		g.nativeShopUI.effectRows,
		battle.NativeFacilityThreeQuarterPrice,
	)
	return frame, err == nil
}

func (g *Game) beginNativeShopSellItemsOpening() bool {
	final, ok := g.composeNativeShopSellItems()
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

func (g *Game) beginNativeShopSellItemsClosing(after func()) bool {
	final, ok := g.composeNativeShopSellItems()
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

func (g *Game) composeNativeShopSellEmpty() ([]byte, bool) {
	if g.nativeShopMode != "sell_empty" {
		return nil, false
	}
	_, unit, unitOK := g.nativeShopSellUnit()
	stable, stableOK := g.composeNativeShopStable()
	_, portrait, portraitID, stateOK := g.nativeShopState()
	if !unitOK || !stableOK || !stateOK ||
		unit.BattleFig < 0 || unit.BattleFig > 0xff {
		return nil, false
	}
	frame, err := campaign.ComposeNativeShopSellEmpty(
		stable, g.nativeClassUI.dialogue, portrait, portraitID,
		g.nativeClassUI.strings, g.nativeClassUI.font,
		g.nativeShopVariant, unit.BattleFig+1,
	)
	return frame, err == nil
}

func (g *Game) beginNativeShopSellEmptyOpening() bool {
	final, ok := g.composeNativeShopSellEmpty()
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

func (g *Game) beginNativeShopSellEmptyClosing(after func()) bool {
	final, ok := g.composeNativeShopSellEmpty()
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

func (g *Game) nativeShopSellSelection() (
	itemID, fullPrice, salePrice int,
	ok bool,
) {
	if g.shopSellSlotSel < 0 ||
		g.shopSellSlotSel >= len(g.nativeShopSellItemIDs) {
		return 0, 0, 0, false
	}
	itemID = g.nativeShopSellItemIDs[g.shopSellSlotSel]
	fullPrice, err := battle.NativeFacilityItemListPrice(
		g.nativeShopUI.effectRows, itemID,
		battle.NativeFacilityFullPrice,
	)
	if err != nil {
		return 0, 0, 0, false
	}
	salePrice, err = battle.NativeFacilityItemListPrice(
		g.nativeShopUI.effectRows, itemID,
		battle.NativeFacilityThreeQuarterPrice,
	)
	return itemID, fullPrice, salePrice, err == nil
}

func (g *Game) composeNativeShopSellQuestionBase() ([]byte, bool) {
	itemID, _, salePrice, selectionOK := g.nativeShopSellSelection()
	stable, stableOK := g.composeNativeShopStable()
	_, portrait, portraitID, stateOK := g.nativeShopState()
	if !selectionOK || !stableOK || !stateOK {
		return nil, false
	}
	frame, err := campaign.ComposeNativeShopSellQuestionBase(
		stable, g.nativeClassUI.dialogue, portrait, portraitID,
		g.nativeClassUI.strings, g.nativeClassUI.font,
		g.nativeShopVariant, itemID, salePrice,
	)
	return frame, err == nil
}

func (g *Game) composeNativeShopSellConfirmation() ([]byte, bool) {
	if g.nativeShopMode != "sell_confirm" ||
		g.nativeShopSellConfirmSel < 0 ||
		g.nativeShopSellConfirmSel > 1 {
		return nil, false
	}
	question, ok := g.composeNativeShopSellQuestionBase()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativeConfirmationChoices(
		question, g.nativeClassUI.choices,
		g.nativeShopSellConfirmSel, g.nativeShopUIPulse/2,
	)
	return frame, err == nil
}

func (g *Game) beginNativeShopSellConfirmationOpening() bool {
	question, ok := g.composeNativeShopSellQuestionBase()
	if !ok {
		return false
	}
	frames, err := campaign.NativeClassConfirmationOpeningFrames(
		question, g.nativeClassUI.choices,
	)
	if err != nil || len(frames) != 4 {
		return false
	}
	g.resetNativeShopUIPulse()
	g.nativeShopUIJob = &nativeClassUIJob{frames: frames}
	return true
}

func (g *Game) beginNativeShopSellConfirmationClosing(after func()) bool {
	question, ok := g.composeNativeShopSellQuestionBase()
	if !ok {
		return false
	}
	choiceFrames, err := campaign.NativeClassConfirmationClosingFrames(
		question, g.nativeClassUI.choices,
	)
	if err != nil || len(choiceFrames) != 4 {
		return false
	}
	stable, ok := g.composeNativeShopStable()
	if !ok {
		return false
	}
	postChoiceClose := choiceFrames[len(choiceFrames)-1]
	dialogueFrames, err := campaign.NativeClassListClosingFrames(
		stable, postChoiceClose,
	)
	if err != nil || len(dialogueFrames) != 5 {
		return false
	}
	frames := append(choiceFrames, dialogueFrames...)
	g.nativeShopUIJob = &nativeClassUIJob{
		frames: frames, restore: stable, after: after,
	}
	return true
}

func (g *Game) beginNativeShopSellSuccess() bool {
	unitID, unit, unitOK := g.nativeShopSellUnit()
	_, fullPrice, _, selectionOK := g.nativeShopSellSelection()
	timeline, timelineOK := g.nativeShopSuccessTimeline()
	assets, _, _, stateOK := g.nativeShopState()
	if !unitOK || !selectionOK || !timelineOK || !stateOK || len(timeline) == 0 {
		return false
	}
	staged := cloneNativeShopUnit(unit)
	nextGold, err := campaign.SellNativeSlot(
		g.gold, &staged, g.shopSellSlotSel, fullPrice,
		g.shopItemStats,
	)
	if err != nil {
		return false
	}
	creditFrames, creditedGold, err := campaign.ComposeNativeGoldCreditFrames(
		timeline[len(timeline)-1].frame, assets.GoldRollStrip,
		g.gold, nextGold-g.gold,
	)
	if err != nil || creditedGold != nextGold {
		return false
	}
	creditTimeline := make([]nativeClassUITimelineStep, len(creditFrames))
	for i, frame := range creditFrames {
		creditTimeline[i] = nativeClassUITimelineStep{
			frame: frame, palette: g.nativeClassUI.palette,
			duration: campaign.NativeGoldRollDelayMilliseconds * time.Millisecond,
		}
	}
	g.nativeShopPendingUnit = staged
	g.nativeShopPendingGold = nextGold
	g.nativeShopHasPendingUnit = true
	g.nativeShopMode = "sell_success"
	finish := func() {
		g.partyRoster[unitID] = cloneNativeShopUnit(
			g.nativeShopPendingUnit,
		)
		g.nativeShopHasPendingUnit = false
		g.nativeShopPendingUnit = battle.Unit{}
		g.nativeShopSellItemIDs = nil
		g.returnToNativeShopSellRoster()
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		timeline: timeline,
		after: func() {
			// 0x2d3ff commits the balance before its first upward 6x9
			// digit window. Raw removal/recompute follow the visible roll.
			g.gold = g.nativeShopPendingGold
			if len(creditTimeline) == 0 {
				finish()
				return
			}
			g.nativeShopUIJob = &nativeClassUIJob{
				timeline: creditTimeline,
				after:    finish,
			}
		},
	}
	return true
}
