package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// nativeShopRecipientInput is the immutable per-update input consumed by the
// production recipient states. Game.Update builds it from Ebiten; tests feed
// the same consumer without changing nativeShopMode directly.
type nativeShopRecipientInput struct {
	enter, escape         bool
	left, right, up, down bool
}

func (g *Game) handleNativeShopRecipientInput(
	input nativeShopRecipientInput,
) bool {
	switch g.nativeShopMode {
	case "recipient_consumable", "recipient_equipment":
		count := len(g.shopRecipients)
		if g.nativeShopMode == "recipient_consumable" {
			delta := 0
			switch {
			case input.left:
				delta = -1
			case input.right:
				delta = 1
			case input.up:
				delta = -2
			case input.down:
				delta = 2
			}
			if delta != 0 {
				g.shopRecipientSel = campaign.AdvanceNativeTwoColumnSelection(
					g.shopRecipientSel, count, delta,
				)
				g.nativeShopRecipientStart, _ = campaign.NativeTwoColumnWindow(
					count, g.shopRecipientSel, g.nativeShopRecipientStart,
				)
			}
		} else {
			nextSelection, nextStart, ok := advanceNativeShopEquipmentRecipient(
				count,
				g.shopRecipientSel,
				g.nativeShopRecipientStart,
				input.up,
				input.down,
			)
			if !ok {
				g.msg = "原版購買 recipient 游標狀態無效"
				g.returnToNativeShopPurchaseList()
				return true
			}
			g.shopRecipientSel = nextSelection
			g.nativeShopRecipientStart = nextStart
		}
		if input.escape {
			if !g.beginNativeShopRecipientClosing(
				g.returnToNativeShopPurchaseList,
			) {
				g.returnToNativeShopPurchaseList()
			}
			return true
		}
		if input.enter && count != 0 {
			unit := g.partyRoster[g.shopRecipients[g.shopRecipientSel]]
			if nativeShopInventoryFull(unit) {
				openFull := func() {
					g.nativeShopMode = "recipient_full"
					if !g.beginNativeShopRecipientFullOpening() {
						g.msg = "原版購買滿欄訊息無法還原"
						g.returnToNativeShopPurchaseList()
					}
				}
				if !g.beginNativeShopRecipientClosing(openFull) {
					openFull()
				}
				return true
			}
			beginTransaction := func() {
				if !g.stageNativeShopPurchase() {
					g.nativeShopHasPendingUnit = false
					g.nativeShopPendingUnit = battle.Unit{}
					g.msg = "原版購買交易缺少 raw 資料"
					g.returnToNativeShopPurchaseList()
				}
			}
			if !g.beginNativeShopRecipientClosing(beginTransaction) {
				beginTransaction()
			}
			return true
		}
		return true
	case "recipient_full":
		if input.enter || input.escape {
			if !g.beginNativeShopRecipientFullClosing(
				g.returnToNativeShopPurchaseList,
			) {
				g.returnToNativeShopPurchaseList()
			}
			return true
		}
		return true
	case "no_recipient":
		if input.enter || input.escape {
			if !g.beginNativeShopNoEligibleRecipientClosing(
				g.returnToNativeShopPurchaseList,
			) {
				g.returnToNativeShopPurchaseList()
			}
			return true
		}
		return true
	}
	return false
}
