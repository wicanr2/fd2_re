package main

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func setNativeTransferInventory(unit battle.Unit, items ...int) battle.Unit {
	unit.Inventory = append([]int(nil), items...)
	unit.Equipped = make([]bool, len(items))
	unit.InventorySlots = make([]int, 8)
	unit.NativeInventoryFlags = make([]int, 8)
	for slot := 0; slot < 8; slot++ {
		unit.InventorySlots[slot] = 0xff
		unit.NativeInventoryFlags[slot] = 0x80
	}
	for slot, item := range items {
		unit.InventorySlots[slot] = item
		unit.NativeInventoryFlags[slot] = 0
	}
	return unit
}

func enterNativeShopTransferProductionPath(
	t *testing.T, g *Game, screen *ebiten.Image,
) {
	t.Helper()
	drainNativeShopProductionJobs(t, g, screen)
	for step := 0; step < 3; step++ {
		if !g.handleNativeShopInputState(nativeShopTransferInput{delta: 1}) {
			t.Fatalf("service3 Right%d was not consumed", step+1)
		}
	}
	if g.nativeShopServiceSel != 3 {
		t.Fatalf("service selection=%d, want 3", g.nativeShopServiceSel)
	}
	if !g.handleNativeShopInputState(nativeShopTransferInput{enter: true}) {
		t.Fatal("service3 Enter was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_intro" {
		t.Fatalf("service3 mode=%q, want transfer_intro", g.nativeShopMode)
	}
	if !g.handleNativeShopInputState(nativeShopTransferInput{enter: true}) {
		t.Fatal("transfer intro Enter was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_source" {
		t.Fatalf("transfer intro returned mode=%q, want transfer_source", g.nativeShopMode)
	}
}

func enterNativeShopTransferDestination(
	t *testing.T, g *Game, screen *ebiten.Image,
) {
	t.Helper()
	if !g.handleNativeShopInputState(nativeShopTransferInput{enter: true}) {
		t.Fatal("transfer source Enter was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_items" {
		t.Fatalf("source selection mode=%q, want transfer_items", g.nativeShopMode)
	}
	if !g.handleNativeShopInputState(nativeShopTransferInput{enter: true}) {
		t.Fatal("transfer item Enter was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_dest_prompt" {
		t.Fatalf("item selection mode=%q, want transfer_dest_prompt", g.nativeShopMode)
	}
	if !g.handleNativeShopInputState(nativeShopTransferInput{enter: true}) {
		t.Fatal("destination prompt Enter was not consumed")
	}
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_dest" {
		t.Fatalf("destination prompt mode=%q, want transfer_dest", g.nativeShopMode)
	}
}

func TestNativeShopTransferProductionInputEmptyAndReturn(t *testing.T) {
	g, screen := newNativeShopRecipientPathGame(t)
	before := cloneNativeShopUnit(g.partyRoster[0])
	enterNativeShopTransferProductionPath(t, g, screen)
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_empty" ||
		!reflect.DeepEqual(g.partyRoster[0], before) || g.gold != 1234 {
		t.Fatalf("empty feedback mode=%q gold=%d unit=%#v", g.nativeShopMode, g.gold, g.partyRoster[0])
	}
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_intro" ||
		!reflect.DeepEqual(g.partyRoster[0], before) || g.gold != 1234 {
		t.Fatal("empty feedback return changed roster or gold")
	}
}

func TestNativeShopTransferProductionInputFullIsAtomic(t *testing.T) {
	g, screen := newNativeShopRecipientPathGame(t)
	g.partyRoster[0] = setNativeTransferInventory(g.partyRoster[0], 0)
	g.partyRoster[1] = setNativeTransferInventory(g.partyRoster[1], 0, 0, 0, 0, 0, 0, 0, 0)
	wantSource := cloneNativeShopUnit(g.partyRoster[0])
	wantDestination := cloneNativeShopUnit(g.partyRoster[1])
	enterNativeShopTransferProductionPath(t, g, screen)
	enterNativeShopTransferDestination(t, g, screen)
	g.handleNativeShopInputState(nativeShopTransferInput{delta: 1})
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_full" || g.gold != 1234 ||
		!reflect.DeepEqual(g.partyRoster[0], wantSource) ||
		!reflect.DeepEqual(g.partyRoster[1], wantDestination) {
		t.Fatalf("full feedback leaked transaction: mode=%q gold=%d", g.nativeShopMode, g.gold)
	}
	g.handleNativeShopInputState(nativeShopTransferInput{escape: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_intro" {
		t.Fatalf("full feedback return mode=%q, want transfer_intro", g.nativeShopMode)
	}
}

func TestNativeShopTransferProductionInputDestinationCancelThenSelf(t *testing.T) {
	g, screen := newNativeShopRecipientPathGame(t)
	g.shopItemStats[1] = campaign.ItemStats{Type: 0}
	g.partyRoster[0] = setNativeTransferInventory(g.partyRoster[0], 0, 1)
	want := cloneNativeShopUnit(g.partyRoster[0])
	enterNativeShopTransferProductionPath(t, g, screen)
	enterNativeShopTransferDestination(t, g, screen)
	g.handleNativeShopInputState(nativeShopTransferInput{escape: true})
	drainNativeShopProductionJobs(t, g, screen)
	if g.nativeShopMode != "transfer_intro" || g.gold != 1234 ||
		!reflect.DeepEqual(g.partyRoster[0], want) {
		t.Fatal("destination cancel did not restore source loop atomically")
	}

	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	enterNativeShopTransferDestination(t, g, screen)
	g.handleNativeShopInputState(nativeShopTransferInput{enter: true})
	drainNativeShopProductionJobs(t, g, screen)
	got := g.partyRoster[0]
	if g.nativeShopMode != "transfer_intro" || g.gold != 1234 ||
		!reflect.DeepEqual(got.Inventory, []int{1, 0}) ||
		!reflect.DeepEqual(got.Equipped, []bool{false, false}) ||
		!reflect.DeepEqual(got.NativeInventoryFlags, []int{0, 0, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}) {
		t.Fatalf("self transfer mode=%q gold=%d unit=%#v", g.nativeShopMode, g.gold, got)
	}
}
