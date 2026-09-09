package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"testing"
)

func TestNativePlayerMovementCancelRestoresRawCursorBeforeSync(t *testing.T) {
	st := &battle.State{W: 30, H: 30}
	err := st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 12, CursorX: 7, CursorY: 13, VisibleCursorX: 7, VisibleCursorY: 1})
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{st: st, m: &MapData{W: 30, H: 30, TileW: 24, TileH: 24}, selOrigX: 7, selOrigY: 14}
	g.restorePlayerMovementCursor()
	g.syncNativeMapView()
	if g.curX != 7 || g.curY != 14 || st.NativeMapViewState.CursorY != 14 {
		t.Fatalf("cancel lost its destination after sync: cursor=(%d,%d) raw=%+v", g.curX, g.curY, st.NativeMapViewState)
	}
}

func TestNativePlayerMovementAdmitsOnlyPreparedNativePreview(t *testing.T) {
	g := &Game{sel: &battle.Unit{}}
	if g.nativeMapFrameAdmission(false, true) {
		t.Fatal("unprepared selection was admitted")
	}
	g.nativeMovePlan = &battle.NativePlayerMovement{}
	if !g.nativeMapFrameAdmission(false, true) {
		t.Fatal("prepared movement lost native viewport")
	}
}

func TestNativePlayerMovementFinishesBeforeSelectionIsReleased(t *testing.T) {
	u := &battle.Unit{}
	g := &Game{sel: u, nativeMovePlan: &battle.NativePlayerMovement{}}
	g.finishSuccessfulUnitAction(u, func() {
		if g.nativeMovePlan != nil || !u.Acted {
			t.Fatal("successful action retained movement preview state")
		}
		g.sel = nil
	})
}
