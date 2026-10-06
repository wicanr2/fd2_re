package main

import (
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestNativeMessageCloseRedrawsHUDAfterRestoreDraw(t *testing.T) {
	for _, owner := range []string{"end_cancel", "reward"} {
		for _, gateB := range []byte{0, 1} {
			t.Run(owner+string(rune('0'+gateB)), func(t *testing.T) {
				g := newNativeHUDRedrawGame(t, gateB)
				source := make([]byte, 320*200)
				g.nativeSystemEndTurnUI = &nativeSystemEndTurnUIState{source: source, dialogue: source}
				continued := 0
				if owner == "reward" {
					g.nativeSystemEndTurnUI.rewardMessage = &nativeDeathRewardMessageState{
						awaitAck: true, final: source, kind: 1, value: 7,
						then: func() {
							continued++
							if g.st.NativeMapHUDState.AnchorX != map[byte]int{0: 1, 1: 242}[gateB] {
								t.Fatal("續行早於重繪")
							}
						},
					}
					g.acknowledgeNativeDeathRewardMessage()
				} else {
					g.nativeSystemEndTurnDelay = 1
					g.stepNativeSystemEndTurn()
				}
				job := g.nativeClassUIJob
				if job == nil || len(job.frames) != 5 || len(job.restore) != len(source) {
					t.Fatal("關框 owner 未建立")
				}
				for frame := 0; frame < 5; frame++ {
					g.stepNativeClassUILifecycle(time.Time{})
					if job.frame != frame {
						t.Fatal("未 Draw 卻前進")
					}
					job.drawn = true
					g.stepNativeClassUILifecycle(time.Time{})
					if g.st.NativeMapHUDState.AnchorX != 1 || continued != 0 || g.gold != 0 {
						t.Fatal("restore 前提前評估或續行")
					}
				}
				job.drawn = true
				g.stepNativeClassUILifecycle(time.Time{})
				want := map[byte]int{0: 1, 1: 242}[gateB]
				if g.nativeClassUIJob != nil || g.nativeSystemEndTurnUI != nil || g.st.NativeMapHUDState.AnchorX != want {
					t.Fatalf("關框結果 anchor=%d want=%d", g.st.NativeMapHUDState.AnchorX, want)
				}
				g.stepNativeClassUILifecycle(time.Time{})
				if owner == "reward" && (continued != 1 || g.gold != 7) {
					t.Fatalf("重複續行或金額錯誤：%d %d", continued, g.gold)
				}
			})
		}
	}
}

func TestNativeSharedCloseDoesNotRedrawBattleHUD(t *testing.T) {
	g := newNativeHUDRedrawGame(t, 1)
	g.nativeClassUIJob = &nativeClassUIJob{restore: make([]byte, 320*200), drawn: true}
	g.stepNativeClassUILifecycle(time.Time{})
	if g.st.NativeMapHUDState.AnchorX != 1 {
		t.Fatal("無原版重繪契約的共用關框改了 anchor")
	}
}

func TestChapter24MapEntryKeepsBothNormalSelectionViews(t *testing.T) {
	mode := 1
	node := &campaign.Node{NativeMapView: &campaign.NativeMapViewConfig{
		CameraX: 9, CameraY: 13, CursorX: 20, CursorY: 19,
		VisibleCursorX: 11, VisibleCursorY: 6, RangeMode: &mode,
	}, NativeMapHUDInherited: &campaign.NativeMapHUDInheritedConfig{DisplayGateB: 1}}
	for _, cameraX := range []int{9, 10} {
		view := battle.NativeMapViewState{CameraX: cameraX, CameraY: 13,
			CursorX: 20, CursorY: 19, VisibleCursorX: 20 - cameraX, VisibleCursorY: 6}
		g := &Game{st: &battle.State{W: 39, H: 30}, m: &MapData{TileW: 24, TileH: 24},
			camp:                    campaign.NewRunner(&campaign.Campaign{Start: "battle_ch24"}),
			handlerInheritedMapView: view, hasHandlerInheritedMapView: true}
		if !g.materializeNativeMapRuntime(node) || g.st.NativeMapViewState != view {
			t.Fatalf("正常選人視圖遭常數覆蓋：%+v err=%s", g.st.NativeMapViewState, g.loadErr)
		}
	}
	// 不完整原生載體不能部分發布到battle.State。
	g := &Game{st: &battle.State{W: 39, H: 30},
		camp:                    campaign.NewRunner(&campaign.Campaign{Start: "battle_ch24"}),
		handlerInheritedMapView: battle.NativeMapViewState{CameraX: -1}, hasHandlerInheritedMapView: true}
	if g.materializeNativeMapRuntime(node) || g.st.HasNativeMapViewState || g.st.HasNativeMapHUDState {
		t.Fatal("缺欄或非法交接視圖被部分發布")
	}
}

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
