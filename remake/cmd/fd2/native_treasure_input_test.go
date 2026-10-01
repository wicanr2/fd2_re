package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"testing"
)

func TestNativeTreasurePromptCommitsOnceAndRejectsChangedSource(t *testing.T) {
	u := nativeNeutralTestUnit(7, 13)
	u.X, u.Y, u.OnField = 7, 13, true
	u.Inventory = nil
	u.InventorySlots = []int{255, 255, 255, 255, 255, 255, 255, 255}
	u.NativeInventoryFlags = []int{128, 128, 128, 128, 128, 128, 128, 128}
	r := battle.Treasure{Slot: 0, Kind: "item", Value: 192}
	g := &Game{st: &battle.State{Treasures: map[battle.Cell]battle.Treasure{{X: 7, Y: 13}: r}, OpenedTreasure: map[int]bool{}}, sel: u, nativeMovePlan: &battle.NativePlayerMovement{}}
	p := &nativeTreasurePrompt{actor: u, reward: r, x: 7, y: 13}
	if len(u.Inventory) != 0 || g.st.OpenedTreasure[0] {
		t.Fatal("prompt preparation changed inventory")
	}
	if !g.commitNativeTreasurePrompt(p) || len(u.Inventory) != 1 || u.Inventory[0] != 192 || !g.st.OpenedTreasure[0] {
		t.Fatal("accepted transaction failed")
	}
	if g.commitNativeTreasurePrompt(p) || len(u.Inventory) != 1 {
		t.Fatal("repeated acknowledgement duplicated item")
	}
}

func TestNativeTreasureNoFinishesActionWithoutOpeningChest(t *testing.T) {
	u := nativeNeutralTestUnit(7, 13)
	r := battle.Treasure{Slot: 0, Kind: "item", Value: 192}
	g := &Game{st: &battle.State{OpenedTreasure: map[int]bool{}, HasNativeMapViewState: true}, sel: u, nativeMovePlan: &battle.NativePlayerMovement{}}
	g.finishNativeTreasurePrompt(&nativeTreasurePrompt{actor: u, reward: r, x: 7, y: 13})
	if g.st.OpenedTreasure[0] || !u.Acted || g.sel != nil || u.NativeRecordByte5&0x80 == 0 {
		t.Fatal("NO changed chest or left action active")
	}
}

func TestNativeTreasureGoldWaitsForClosing(t *testing.T) {
	u := nativeNeutralTestUnit(7, 13)
	u.X, u.Y, u.OnField = 7, 13, true
	r := battle.Treasure{Slot: 0, Kind: "gold", Value: 3500}
	g := &Game{gold: 10, st: &battle.State{Treasures: map[battle.Cell]battle.Treasure{{X: 7, Y: 13}: r}}, sel: u}
	p := &nativeTreasurePrompt{actor: u, reward: r, x: 7, y: 13}
	if !g.commitNativeTreasurePrompt(p) || g.gold != 10 || g.st.NativeEventState[0] != 0 {
		t.Fatal("gold or event state committed before closing")
	}
	if g.commitNativeTreasurePrompt(p) {
		t.Fatal("gold YES acknowledged twice")
	}
	g.finishNativeTreasurePrompt(p)
	if g.loadErr != "" || g.gold != 3510 || g.st.NativeEventState[0] != 1 {
		t.Fatalf("closing failed: error=%s gold=%d event=%d", g.loadErr, g.gold, g.st.NativeEventState[0])
	}
}
