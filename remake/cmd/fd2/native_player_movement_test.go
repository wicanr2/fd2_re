package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"reflect"
	"testing"
)

func TestNativePhysicalAttackOwnCellConfirmPreservesTargetState(t *testing.T) {
	actor := &battle.Unit{Camp: battle.Own, OnField: true, X: 0, Y: 0,
		HP: 10, MaxHP: 20, HasNativeRecordByte5: true}
	enemy := &battle.Unit{Camp: battle.Enemy, OnField: true, X: 1, Y: 0,
		HP: 12, MaxHP: 12, HasNativeRecordByte5: true}
	st := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, enemy},
		HasNativeMapViewState: true, NativeMapRangeMode: 1,
		NativeTileBlitModes: []byte{0xff, 0}}
	g := &Game{st: st, m: &MapData{W: 2, H: 1}, sel: actor, moved: true,
		nativeRNGState: 17791}
	beforeActor, beforeEnemy := *actor, *enemy
	beforeField := append([]byte(nil), st.NativeTileBlitModes...)
	g.confirm()
	if g.sel != actor || !g.moved || g.ring || g.atk != nil || g.nativeRNGState != 17791 ||
		!reflect.DeepEqual(*actor, beforeActor) || !reflect.DeepEqual(*enemy, beforeEnemy) ||
		!reflect.DeepEqual(st.NativeTileBlitModes, beforeField) {
		t.Fatalf("rejected own-cell confirmation changed target state: actor=%+v selected=%v", actor, g.sel)
	}
	// 原生閘門在合法敵方格通過，自己格即使被誤標非FF仍不能當敵方。
	g.curX = 1
	if !g.nativePhysicalAttackConfirmationAllowed() {
		t.Fatal("active enemy inside marked field was rejected")
	}
	g.curX, st.NativeTileBlitModes[0] = 0, 0
	if g.nativePhysicalAttackConfirmationAllowed() {
		t.Fatal("own unit was accepted as physical enemy target")
	}
	st.NativeTileBlitModes = nil
	g.confirm()
	if actor.Acted || actor.HP != 10 || g.sel != actor {
		t.Fatal("missing native field submitted wait")
	}
}

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
