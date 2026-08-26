package main

import (
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func enterNativeShopEquipPanel(t *testing.T, g *Game, selectUnit int) {
	t.Helper()
	for g.nativeShopServiceSel != 2 {
		g.handleNativeShopInputState(nativeShopTransferInput{delta: 1})
	}
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	if g.nativeShopMode != "menu" || g.nativeShopUIJob == nil {
		t.Fatalf("service2 mode=%q job=%#v", g.nativeShopMode, g.nativeShopUIJob)
	}
	g.nativeShopEquipUnitSel = selectUnit
}

func settleNativeShopEquipPanelOpening(g *Game) {
	for g.nativeShopEquipPanelBlocksInput() {
		g.stepNativeShopEquipPanelLifecycle()
	}
}

func TestNativeShopEquipTypedInputSuccessAndReturn(t *testing.T) {
	g, screen := newNativeShopRecipientPathGame(t)
	g.partyRoster[0] = setNativeTransferInventory(g.partyRoster[0], 0, 1)
	g.shopItemTypes[0], g.shopItemTypes[1] = 0, 0
	g.shopEquipTypes[1] = []int{0}
	g.shopItemStats = map[int]campaign.ItemStats{0: {Type: 0, AP: 1}, 1: {Type: 0, AP: 3}}
	drainNativeShopProductionJobs(t, g, screen)
	enterNativeShopEquipPanel(t, g, 0)
	drainNativeShopProductionJobs(t, g, screen)
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "equip_panel" {
		t.Fatalf("panel mode=%q", g.nativeShopMode)
	}
	settleNativeShopEquipPanelOpening(g)
	g.handleNativeShopInputState(nativeShopTransferInput{delta: 2})
	if g.itemSel != 1 {
		t.Fatalf("item selection=%d", g.itemSel)
	}
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	unit := g.partyRoster[0]
	if !reflect.DeepEqual(unit.Equipped, []bool{false, true}) || unit.AP != 32 {
		t.Fatalf("equip result=%#v", unit)
	}
	g.handleNativeShopInputState(nativeShopTransferInput{escape: true})
	for g.nativeShopMode == "equip_panel" {
		g.stepNativeShopEquipPanelLifecycle()
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "equip_roster" || g.nativeShopEquipUnitSel != 0 {
		t.Fatalf("panel return mode=%q unit=%d", g.nativeShopMode, g.nativeShopEquipUnitSel)
	}
	g.handleNativeShopInputState(nativeShopTransferInput{escape: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "menu" || g.nativeShopServiceSel != 2 {
		t.Fatalf("service return mode=%q selection=%d", g.nativeShopMode, g.nativeShopServiceSel)
	}
}

func TestNativeShopEquipTypedInputIncompatibleAndEmptyAreAtomic(t *testing.T) {
	g, screen := newNativeShopRecipientPathGame(t)
	g.partyRoster[0] = setNativeTransferInventory(g.partyRoster[0], 1)
	g.shopItemTypes[1] = 9
	g.shopEquipTypes[1] = []int{0}
	want := cloneNativeShopUnit(g.partyRoster[0])
	drainNativeShopProductionJobs(t, g, screen)
	enterNativeShopEquipPanel(t, g, 0)
	drainNativeShopProductionJobs(t, g, screen)
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	settleNativeShopEquipPanelOpening(g)
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	if !reflect.DeepEqual(g.partyRoster[0], want) || g.nativeShopMode != "equip_panel" {
		t.Fatal("incompatible item changed unit or closed panel")
	}
	g.handleNativeShopInputState(nativeShopTransferInput{escape: true})
	for g.nativeShopMode == "equip_panel" {
		g.stepNativeShopEquipPanelLifecycle()
	}
	drainNativeShopProductionJobs(t, g, screen)

	g.nativeShopEquipUnitSel = 1
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	settleNativeShopEquipPanelOpening(g)
	if g.nativeShopMode != "equip_panel" || len(g.partyRoster[1].Inventory) != 0 {
		t.Fatal("empty inventory panel was unavailable")
	}
	g.handleNativeShopInputState(nativeShopTransferInput{escape: true})
	for g.nativeShopMode == "equip_panel" {
		g.stepNativeShopEquipPanelLifecycle()
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "equip_roster" {
		t.Fatalf("empty return mode=%q", g.nativeShopMode)
	}
}
