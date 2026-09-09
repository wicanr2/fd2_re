package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"testing"
)

func TestNativeDialogueFocusOwnsInputBeforeOpening(t *testing.T) {
	g := &Game{
		m:    &MapData{W: 31, H: 64, TileW: 24, TileH: 24},
		camX: 72, camY: 480, curX: 8, curY: 21,
		hasStoryNativeMapView: true,
		storyNativeMapView:    battle.NativeMapViewState{CameraX: 3, CameraY: 20, CursorX: 8, CursorY: 21, VisibleCursorX: 5, VisibleCursorY: 1},
		storyActors:           []battle.Unit{{X: 8, Y: 16, HasNativeMapPresentation: true, NativeMapPresentation: battle.NativeMapPresentationState{X: 8, Y: 16}}},
		dialog:                []battle.DialogLine{{NativeDialogue: &battle.NativeDialogueLayout{Control: "FFED", Operand: 0, HasMotionTargetY: true, MotionTargetY: 2}}},
	}
	if err := g.startNativeDialogueFrames(); err != nil {
		t.Fatal(err)
	}
	if g.focusJob == nil || len(g.dialog) != 0 || len(g.nativeDialogueOpening) != 0 {
		t.Fatal("聚焦前已公開新框或失去阻擋擁有者")
	}
	g.stepFocusUnit()
	if g.curY != 20 || g.camY != 456 || g.storyNativeMapView.CameraY != 19 || len(g.dialog) != 0 {
		t.Fatalf("未依原版安全帶同步視野: cursor=%d camera=%v view=%+v", g.curY, g.camY, g.storyNativeMapView)
	}
}

func TestNativeDialogueFocusAfterInterpolatedPan(t *testing.T) {
	g := &Game{
		m:    &MapData{W: 18, H: 51, TileW: 24, TileH: 24},
		camX: 72, camY: 96, curX: 8, curY: 8,
		hasStoryNativeMapView: true,
		storyNativeMapView:    battle.NativeMapViewState{CameraX: 3, CameraY: 4, CursorX: 8, CursorY: 8, VisibleCursorX: 5, VisibleCursorY: 4},
	}
	continued := false
	g.camPan = &camPanJob{fromX: 72, fromY: 96, toY: 1032, frames: 2, then: func() { continued = true }}
	g.stepCamPan()
	if continued || g.storyNativeMapView.CameraY != 4 {
		t.Fatal("插值中途發布非整格原生視圖")
	}
	g.stepCamPan()
	view := g.storyNativeMapView
	if !continued || g.loadErr != "" || view.CameraX != 0 || view.CameraY != 43 || view.CursorX != 5 || view.CursorY != 47 || view.VisibleCursorX != 5 || view.VisibleCursorY != 4 {
		t.Fatalf("鏡頭終點未同步: %+v error=%s", view, g.loadErr)
	}
	_, _, focused, err := g.nativeFocusEndpoint(10, 47)
	if err != nil || focused.CursorX != 10 || focused.VisibleCursorX != 10 || focused.VisibleCursorY != 4 {
		t.Fatalf("終點後聚焦失敗: %+v %v", focused, err)
	}
}

func TestNativeDialogueSpeakingCountsGlyphsAcrossPages(t *testing.T) {
	// 前三列各一字，第四列先捲動十幀，再發布一個字形。
	layout := &campaign.NativeDialogueLayout{SourceDAT: "FDTXT_001", Control: "FFEE", Pages: [][]string{{"甲", "乙", "丙", "丁"}}}
	steps, err := campaign.NativeStoryDialogueGlyphSteps(layout, 0)
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{dialog: []battle.DialogLine{{NativeDialogue: &battle.NativeDialogueLayout{}}}, nativeDialogueProgress: -1,
		nativeDialogueProgressive: [][][]byte{make([][]byte, len(steps))}, nativeDialogueGlyphSteps: [][]bool{steps}}
	for i := 0; i < 4; i++ {
		g.stepNativeStoryDialogueProgress()
	}
	if g.nativeDialogueSpeakingHalf != 1 || g.nativeDialogueSpeakingCycle != 1 {
		t.Fatal("前三字未保留兩字一次的原版計數")
	}
	for i := 0; i < 10; i++ {
		g.stepNativeStoryDialogueProgress()
	}
	if g.nativeDialogueSpeakingHalf != 1 || g.nativeDialogueSpeakingCycle != 1 {
		t.Fatal("捲動幀被誤算成普通字形")
	}
	g.stepNativeStoryDialogueProgress()
	if g.nativeDialogueSpeakingHalf != 0 || g.nativeDialogueSpeakingCycle != 2 || g.nativeDialogueSpeakingFrame != 2 {
		t.Fatal("第四字未切到來源 frame 2")
	}
	g.nativeDialogueClosingLive = true
	g.finishNativeStoryDialogueClosing()
	if g.nativeDialogueSpeakingCycle != 2 {
		t.Fatal("收框錯誤重設跨句計數器")
	}
}
