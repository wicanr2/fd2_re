package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// newNativeHUDRedrawGame 建一個只有視圖與 HUD 狀態的戰場：可見游標 (2,6) 落在 0x1AD2A 的
// 翻右區（Y>5、X<3），anchor 仍在左側 1。
func newNativeHUDRedrawGame(t *testing.T, gateB byte) *Game {
	t.Helper()
	g := &Game{st: &battle.State{W: 30, H: 30}, m: &MapData{W: 30, H: 30, TileW: 24, TileH: 24}}
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 5, CameraY: 10, CursorX: 7, CursorY: 16, VisibleCursorX: 2, VisibleCursorY: 6,
	}); err != nil {
		t.Fatal(err)
	}
	if !g.st.MaterializeNativeMapRangeMode(0) || !g.st.MaterializeNativeMapHUDState(1, gateB, 1) {
		t.Fatal("HUD 狀態建不起來")
	}
	g.syncNativeMapView()
	return g
}

func TestNativeFocusRedrawsHUDBeforeFirstStep(t *testing.T) {
	// 0x12CEA 開頭 0x12D01 無條件 0x11CAC(0)：目標就是目前格也要評估一次 anchor。
	g := newNativeHUDRedrawGame(t, 1)
	g.aiFocusCursor(7, 16)
	if got := g.st.NativeMapHUDState.AnchorX; got != 0xf2 {
		t.Fatalf("聚焦開頭沒有重繪評估 anchor：%#x", got)
	}
}

func TestNativeFocusSkipsHUDWhenGateBClosed(t *testing.T) {
	// 0x1ACF3 閘 B 為 0 直接返回，0x1AD2A 不跑（敵方回合、0x1A30B 內的回合開頭聚焦）。
	g := newNativeHUDRedrawGame(t, 0)
	g.aiFocusCursor(7, 16)
	if got := g.st.NativeMapHUDState.AnchorX; got != 1 {
		t.Fatalf("閘 B 關著卻評估了 anchor：%#x", got)
	}
}

func TestFocusUnitJobOnBattleViewUsesRedrawEntry(t *testing.T) {
	// 0x13FD4 原地回復的 0x12D7B 走 focusUnitJob：同一支 0x12CEA，開頭重繪一次。
	g := newNativeHUDRedrawGame(t, 1)
	g.focusJob = &focusUnitJob{targetX: 7, targetY: 16, nativeView: true, then: func() {}}
	for step := 0; g.focusJob != nil && step < 10; step++ {
		g.stepFocusUnit()
	}
	if g.focusJob != nil || g.loadErr != "" {
		t.Fatalf("聚焦沒有結束：%s", g.loadErr)
	}
	if got := g.st.NativeMapHUDState.AnchorX; got != 0xf2 {
		t.Fatalf("focusUnitJob 沒有經重繪入口評估 anchor：%#x", got)
	}
}

func TestFocusUnitJobStepWithoutRedrawKeepsAnchor(t *testing.T) {
	// 往右走一格只動可見游標、[0x51A83]==0：0x11BFA 不重繪，anchor 停在開頭評估的結果。
	g := newNativeHUDRedrawGame(t, 1)
	g.st.NativeMapHUDState.AnchorX = 1
	g.st.NativeMapViewState.VisibleCursorY = 4
	g.st.NativeMapViewState.CursorY = 14
	g.syncNativeMapView()
	g.focusJob = &focusUnitJob{targetX: 7, targetY: 16, nativeView: true, then: func() {}}
	for step := 0; g.focusJob != nil && step < 10; step++ {
		g.stepFocusUnit()
	}
	v := g.st.NativeMapViewState
	if v.CursorY != 16 || v.VisibleCursorY != 6 {
		t.Fatalf("聚焦終點 %+v", v)
	}
	if got := g.st.NativeMapHUDState.AnchorX; got != 1 {
		t.Fatalf("沒重繪的步進評估了 anchor：%#x", got)
	}
}
