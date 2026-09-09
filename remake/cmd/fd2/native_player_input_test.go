package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func nativeNeutralTestUnit(x, y byte) *battle.Unit {
	u := nativeItemPanelTestUnit()
	u.HasBattleFig, u.HasNativeRecordByte5 = true, true
	u.NativeRecordByte6 = 2
	u.HasNativeMapPresentation = true
	u.NativeMapPresentation.X, u.NativeMapPresentation.Y = x, y
	u.HasNativeRecordByte34, u.HasNativeRecordByte35, u.HasNativeRecordByte36 = true, true, true
	u.HasNativeRecordWord42, u.HasNativeRecordWord46 = true, true
	u.NativeRecordWord42, u.NativeRecordWord46 = 100, 40
	return u
}

func TestNativePlayerCycleRawOrderSkipsEachGateAndWraps(t *testing.T) {
	units := []*battle.Unit{nativeNeutralTestUnit(7, 14), nativeNeutralTestUnit(8, 15), nativeNeutralTestUnit(9, 16), nativeNeutralTestUnit(10, 17), nativeNeutralTestUnit(11, 18)}
	units[1].NativeRecordByte5 = 1
	units[2].NativeRecordByte5 = 4
	units[3].NativeRecordByte5 = 0x80
	units[4].NativeRecordByte6 = 0
	g := &Game{st: &battle.State{W: 30, H: 30, Units: units}, nativeNextPlayerIndex: 1}
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CursorX: 1, CursorY: 1, VisibleCursorX: 1, VisibleCursorY: 1}); err != nil {
		t.Fatal(err)
	}
	g.cycleNativePlayerUnit()
	if g.loadErr != "" || g.nativePlayerFocus == nil || *g.nativePlayerFocus != (battle.Cell{7, 14}) || g.nativeNextPlayerIndex != 1 {
		t.Fatalf("cycle failed: %s target=%v next=%d", g.loadErr, g.nativePlayerFocus, g.nativeNextPlayerIndex)
	}
	g.nativePlayerFocus = nil
	units[0].NativeRecordByte5 = 0x80
	g.cycleNativePlayerUnit()
	if g.nativePlayerFocus != nil || g.nativeNextPlayerIndex != 1 {
		t.Fatal("no candidate changed state")
	}
	units[0].NativeRecordByte5 = 0
	g.sel = units[0]
	g.cycleNativePlayerUnit()
	if g.nativePlayerFocus != nil {
		t.Fatal("selection cancellation was bypassed")
	}
}

func TestNativePlayerFocusUsesRawBattleViewAndXFirstSafeBand(t *testing.T) {
	g := &Game{st: &battle.State{W: 30, H: 30}, m: &MapData{W: 30, H: 30, TileW: 24, TileH: 24}, hasStoryNativeMapView: true}
	g.storyNativeMapView = battle.NativeMapViewState{CursorX: 25}
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraY: 13, CursorX: 7, CursorY: 14, VisibleCursorX: 7, VisibleCursorY: 1}); err != nil {
		t.Fatal(err)
	}
	u := nativeNeutralTestUnit(15, 20)
	if !g.beginNativePlayerFocus(u) {
		t.Fatal(g.loadErr)
	}
	g.stepNativePlayerFocus()
	if g.st.NativeMapViewState.CursorX != 8 || g.st.NativeMapViewState.CursorY != 14 {
		t.Fatal("focus did not move X first")
	}
	for n := 0; n < 50 && g.nativePlayerFocus != nil; n++ {
		g.stepNativePlayerFocus()
	}
	v := g.st.NativeMapViewState
	if g.nativePlayerFocus != nil || v.CursorX != 15 || v.CursorY != 20 || v.CameraX != 4 || v.CameraY != 14 || g.storyNativeMapView.CursorX != 25 {
		t.Fatalf("focus state %+v err=%s", v, g.loadErr)
	}
}

func TestNativePlayerStatusGatesAndFailedResourcesDoNotSelect(t *testing.T) {
	g := &Game{}
	u := nativeNeutralTestUnit(7, 14)
	if handled, err := g.inspectNativePlayerUnit(u, false); handled || err != nil {
		t.Fatalf("normal unit was intercepted: %v %v", handled, err)
	}
	u.BattleFig = 0x79
	if handled, err := g.inspectNativePlayerUnit(u, true); !handled || err != nil || g.nativePlayerStatus != nil {
		t.Fatal("special record did not reject")
	}
	u.BattleFig = 0
	u.NativeRecordByte5 = 0x80
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	if handled, err := g.inspectNativePlayerUnit(u, false); !handled || err == nil || g.nativePlayerStatus != nil || g.sel != nil {
		t.Fatal("missing status resources did not fail closed")
	}
}

func TestNativePlayerStatusTwoPagesWaitForDrawAndRestore(t *testing.T) {
	s := &nativePlayerStatusState{source: make([]byte, 64000), status: make([]byte, 64000), commands: make([]byte, 64000), phase: "status"}
	g := &Game{nativePlayerStatus: s}
	g.advanceNativePlayerStatus()
	if len(s.frames) != 14 {
		t.Fatal("command transition missing")
	}
	g.stepNativePlayerStatus()
	if s.frame != 0 {
		t.Fatal("undrawn frame advanced")
	}
	for n := 0; n < 14; n++ {
		s.drawn = true
		g.stepNativePlayerStatus()
	}
	if s.phase != "commands" || len(s.frames) != 0 {
		t.Fatal("command page not steady")
	}
	g.advanceNativePlayerStatus()
	for n := 0; n < 12; n++ {
		s.drawn = true
		g.stepNativePlayerStatus()
	}
	if g.nativePlayerStatus != nil {
		t.Fatal("status did not close")
	}
}

func TestNativePlayerWaitPublishesRawCompletion(t *testing.T) {
	u := nativeNeutralTestUnit(7, 14)
	g := &Game{st: &battle.State{HasNativeMapViewState: true}, sel: u, nativeMovePlan: &battle.NativePlayerMovement{}}
	g.finishSuccessfulUnitAction(u, nil)
	if !u.Acted || u.NativeRecordByte5&0x80 == 0 {
		t.Fatal("wait did not publish both completion consumers")
	}
}

func TestNativePlayerUnmovedWaitPublishesRawCompletionWithoutPreview(t *testing.T) {
	u := nativeNeutralTestUnit(7, 13)
	g := &Game{st: &battle.State{HasNativeMapViewState: true}, sel: u}
	g.finishSuccessfulUnitAction(u, nil)
	if !u.Acted || u.NativeRecordByte5&0x80 == 0 {
		t.Fatal("unmoved wait lost raw completion after movement preview closed")
	}
}
