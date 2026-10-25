package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func unitStageRelocationGame(t *testing.T, top bool, set *modernStoryPortraitSet) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g, _, _ := player9StageGame(t, false, true)
	a, b, e := g.st.Units[0], g.st.Units[1], g.st.Units[4]
	a.Camp, a.OnField, a.NativeIdentity, a.HasNativeIdentity = battle.Own, true, 24, true
	a.NativeRecordByte5, a.HasNativeRecordByte5 = 0, true
	a.MP, a.MaxMP = 30, 40
	a.Inventory = []int{101}
	a.Equipped = []bool{false}
	a.InventorySlots = []int{101, 255, 255, 255, 255, 255, 255, 255}
	a.NativeInventoryFlags = []int{0, 128, 128, 128, 128, 128, 128, 128}
	b.Camp, b.OnField, b.NativeIdentity, b.HasNativeIdentity = battle.Own, true, 1, true
	b.NativeRecordClass, b.HasNativeRecordClass = 1, true
	b.NativeRecordByte5, b.HasNativeRecordByte5 = 0, true
	b.InventorySlots = []int{255, 255, 255, 255, 255, 255, 255, 255}
	b.NativeInventoryFlags = []int{128, 128, 128, 128, 128, 128, 128, 128}
	y, destY := 19, 14
	if top {
		y, destY = 14, 19
	}
	a.SetMapPlacement(26, y, 3)
	b.SetMapPlacement(27, y, 1)
	e.SetMapPlacement(20, 20, 1)
	for _, u := range []*battle.Unit{a, b, e} {
		u.OnField = true
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			t.Fatal(err)
		}
	}
	g.st.Units = []*battle.Unit{a, b, e}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for i := range book {
		book[i].ID = i
	}
	book[23].MPCost = 20
	g.nativeCommandBook, g.st.NativeCommandBook = book, book
	g.sel, g.moved, g.itemOpen = a, true, true
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	g.st.NativeTerrainMoveCodes = make([]byte, g.st.W*g.st.H)
	for i := range g.st.NativeTerrainMoveCodes {
		g.st.NativeTerrainMoveCodes[i] = 2
	}
	g.st.NativeTileBlitModes[b.Y*g.st.W+b.X] = 0
	g.m.NativeTileBlitModes[b.Y*g.st.W+b.X] = 0
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 16, CameraY: 14, CursorX: 27, CursorY: y, VisibleCursorX: 11, VisibleCursorY: y - 14}); err != nil {
		t.Fatal(err)
	}
	g.camX, g.camY, g.curX, g.curY = 16*24, 14*24, 27, y
	g.modernStoryPortraits = set
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if targeting, err := g.beginNativeTargetItem(0, 101); err != nil || !targeting {
		t.Fatal("stage target input", targeting, err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if applied, err := g.applyNativeTargetItem(b); err != nil || applied || !g.nativeItemRelocating {
		t.Fatal("stage target confirm", applied, err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if applied, err := g.applyNativeRelocationDestination(27, destY); err != nil || !applied {
		t.Fatal("stage destination confirm", applied, err)
	}
	return g, a, b
}
