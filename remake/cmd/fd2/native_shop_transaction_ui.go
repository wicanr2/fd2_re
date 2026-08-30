package main

import (
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func cloneNativeShopUnit(source battle.Unit) battle.Unit {
	out := source
	out.Inventory = append([]int(nil), source.Inventory...)
	out.Equipped = append([]bool(nil), source.Equipped...)
	out.InventorySlots = append([]int(nil), source.InventorySlots...)
	out.NativeInventoryFlags = append(
		[]int(nil), source.NativeInventoryFlags...,
	)
	out.Spells = append([]int(nil), source.Spells...)
	return out
}

func (g *Game) composeNativeShopEquipQuestion() ([]byte, bool) {
	if !g.nativeShopHasPendingUnit {
		return nil, false
	}
	good, goodOK := g.nativeShopSelectedGood()
	_, portrait, portraitID, stateOK := g.nativeShopState()
	stable, stableOK := g.composeNativeShopStable()
	if !goodOK || !stateOK || !stableOK {
		return nil, false
	}
	if g.localeID != "" && g.localeID != "zh-Hant" {
		frame, err := g.composeLocalizedNativeShopPlainMessage(
			stable, portrait, portraitID, "shop.purchase.equip_question",
		)
		return frame, err == nil
	}
	frame, err := campaign.ComposeNativeShopPurchaseMessage(
		stable, g.nativeClassUI.dialogue, portrait, portraitID,
		g.nativeClassUI.strings, g.nativeClassUI.font,
		campaign.NativeShopPurchaseEquipQuestion,
		g.nativeShopVariant, good.ID, good.Price,
	)
	return frame, err == nil
}

func (g *Game) composeNativeShopEquipConfirmation() ([]byte, bool) {
	if g.nativeShopMode != "equip_confirm" ||
		g.nativeShopEquipSel < 0 || g.nativeShopEquipSel > 1 {
		return nil, false
	}
	question, ok := g.composeNativeShopEquipQuestion()
	if !ok {
		return nil, false
	}
	frame, err := campaign.ComposeNativeConfirmationChoices(
		question, g.nativeClassUI.choices,
		g.nativeShopEquipSel, g.nativeShopUIPulse/2,
	)
	return frame, err == nil
}

func (g *Game) beginNativeShopEquipConfirmationOpening() bool {
	question, ok := g.composeNativeShopEquipQuestion()
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

func (g *Game) beginNativeShopEquipConfirmationClosing(after func()) bool {
	question, ok := g.composeNativeShopEquipQuestion()
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

func (g *Game) stageNativeShopPurchase() bool {
	good, goodOK := g.nativeShopSelectedGood()
	if !goodOK || g.shopRecipientSel < 0 ||
		g.shopRecipientSel >= len(g.shopRecipients) {
		return false
	}
	recipientID := g.shopRecipients[g.shopRecipientSel]
	source, ok := g.partyRoster[recipientID]
	if !ok {
		return false
	}
	staged := cloneNativeShopUnit(source)
	slot, err := campaign.ReserveGood(g.gold, &staged, good)
	if err != nil {
		return false
	}
	g.shopPending = good
	g.shopEquipUnit = recipientID
	g.shopEquipSlot = slot
	g.nativeShopPendingUnit = staged
	g.nativeShopHasPendingUnit = true
	itemType, ok := g.shopItemTypes[good.ID]
	if !ok {
		g.nativeShopHasPendingUnit = false
		return false
	}
	if itemType < 0x20 {
		g.nativeShopMode = "equip_confirm"
		g.nativeShopEquipSel = 0
		if !g.beginNativeShopEquipConfirmationOpening() {
			g.nativeShopHasPendingUnit = false
			return false
		}
		return true
	}
	return g.beginNativeShopPurchaseSuccess()
}

func (g *Game) beginNativeShopPurchaseSuccess() bool {
	if !g.nativeShopHasPendingUnit {
		return false
	}
	timeline, ok := g.nativeShopSuccessTimeline()
	if !ok {
		return false
	}
	good, goodOK := g.nativeShopSelectedGood()
	assets, _, _, stateOK := g.nativeShopState()
	if !goodOK || !stateOK || len(timeline) == 0 {
		return false
	}
	debitFrames, nextGold, err := campaign.ComposeNativeGoldDebitFrames(
		timeline[len(timeline)-1].frame, assets.GoldRollStrip,
		g.gold, good.Price,
	)
	if err != nil {
		return false
	}
	debitTimeline := make([]nativeClassUITimelineStep, len(debitFrames))
	for i, frame := range debitFrames {
		debitTimeline[i] = nativeClassUITimelineStep{
			frame: frame, palette: g.nativeClassUI.palette,
			duration: campaign.NativeGoldRollDelayMilliseconds * time.Millisecond,
		}
	}
	recipientID := g.shopEquipUnit
	staged := cloneNativeShopUnit(g.nativeShopPendingUnit)
	g.partyRoster[recipientID] = staged
	g.nativeShopMode = "success"
	finish := func() {
		g.nativeShopHasPendingUnit = false
		g.nativeShopPendingUnit = battle.Unit{}
		g.returnToNativeShopPurchaseList()
	}
	g.nativeShopUIJob = &nativeClassUIJob{
		timeline: timeline,
		after: func() {
			// 0x2d516 subtracts the balance before its first 6x9 digit
			// window, after 0x2f4c6's success presentation has completed.
			g.gold = nextGold
			if len(debitTimeline) == 0 {
				finish()
				return
			}
			g.nativeShopUIJob = &nativeClassUIJob{
				timeline: debitTimeline,
				after:    finish,
			}
		},
	}
	return true
}

func (g *Game) nativeShopSuccessTimeline() (
	[]nativeClassUITimelineStep,
	bool,
) {
	stable, stableOK := g.composeNativeShopBare()
	assets, portrait, portraitID, stateOK := g.nativeShopState()
	if !stableOK || !stateOK {
		return nil, false
	}
	animation, final, err := campaign.ComposeNativeShopPurchaseSuccessFrames(
		stable, assets, portrait, portraitID, g.nativeShopVariant,
	)
	if err != nil || len(animation) == 0 || len(final) != 320*200 {
		return nil, false
	}
	plan, err := campaign.PlanNativeShopPurchaseSuccess(g.nativeShopVariant)
	if err != nil {
		return nil, false
	}
	timeline := make([]nativeClassUITimelineStep, 0, len(animation)+2)
	if plan.PreDelayBIOSTicks != 0 {
		timeline = append(timeline, nativeClassUITimelineStep{
			frame: stable, palette: g.nativeClassUI.palette,
			duration: time.Duration(plan.PreDelayBIOSTicks) * nativeBIOSTickPeriod,
		})
	}
	for i, frame := range animation {
		delay := plan.PerFrameDelayBIOSTicks
		if i == len(animation)-1 {
			delay += plan.PostDelayBIOSTicks
		}
		timeline = append(timeline, nativeClassUITimelineStep{
			frame: frame, palette: g.nativeClassUI.palette,
			duration: time.Duration(delay) * nativeBIOSTickPeriod,
		})
	}
	timeline = append(timeline, nativeClassUITimelineStep{
		frame: final, palette: g.nativeClassUI.palette,
	})

	return timeline, true
}
