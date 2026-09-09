package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestBattleWalkPreservesNativeSevenTickRecordLifecycle(t *testing.T) {
	u := &battle.Unit{X: 2, Y: 3}
	if err := u.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m: &MapData{TileW: 24, TileH: 24},
		walk: &walkAnim{
			u:    u,
			path: []battle.Cell{{X: 2, Y: 3}, {X: 3, Y: 3}},
		},
	}
	for tick := 1; tick <= 6; tick++ {
		g.stepBattleWalk()
		raw := u.NativeMapPresentation
		if raw.X != 2 || raw.Y != 3 || raw.Pose != 3 || raw.Motion != byte(tick) {
			t.Fatalf("tick %d raw=%+v", tick, raw)
		}
		if u.X != 2 || u.Y != 3 || u.OffX <= 0 {
			t.Fatalf("tick %d normalized=(%d,%d off=%v)", tick, u.X, u.Y, u.OffX)
		}
	}
	g.stepBattleWalk()
	if g.walk != nil || !g.moved || !g.ring {
		t.Fatalf("seventh tick did not finish player walk: walk=%v moved=%v ring=%v", g.walk, g.moved, g.ring)
	}
	// 抵達時 raw +3 回到靜止值 0，不是最後一格的行走方向。tick 1..6 的
	// 姿態與動作契約不變。dosgolem 原版收據以 10 萬指令粒度量到 pose 2→0、
	// motion 1..6→0，見 docs/data/ui-traces/fd2-map-walk-pose-20260909.json；
	// 先前這裡釘的 Pose: 3 沒有原版依據，屬重製端實作的自我一致。
	if u.NativeMapPresentation != (battle.NativeMapPresentationState{X: 3, Y: 3, Pose: nativeMapRestPose}) ||
		u.X != 3 || u.Y != 3 || u.OffX != 0 || u.OffY != 0 {
		t.Fatalf("seventh tick raw=%+v normalized=(%d,%d,%v,%v)",
			u.NativeMapPresentation, u.X, u.Y, u.OffX, u.OffY)
	}
}

func TestBattleWalkAppliesMap25Selector0OnlyAfterLeftStepCommit(t *testing.T) {
	st, err := battle.Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	trigger := st.Units[0]
	trigger.NativeRecordByte6 = 1
	trigger.HasNativeRecordByte6 = true
	trigger.SetMapPlacement(11, 36, 1)
	if err := trigger.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	for index := 39; index <= 44; index++ {
		st.Units[index].NativeRecordByte34 = 0xA7
		st.Units[index].HasNativeRecordByte34 = true
	}
	g := &Game{
		m:  &MapData{TileW: 24, TileH: 24},
		st: st,
		walk: &walkAnim{
			u:    trigger,
			path: []battle.Cell{{X: 11, Y: 36}, {X: 10, Y: 36}},
		},
	}
	for tick := 1; tick <= 6; tick++ {
		g.stepBattleWalk()
		for index := 39; index <= 44; index++ {
			if got := st.Units[index].NativeRecordByte34; got != 0xA7 {
				t.Fatalf("tick %d unit%d byte34=%#x, selector0 ran before commit", tick, index, got)
			}
		}
	}
	g.stepBattleWalk()
	for index := 39; index <= 44; index++ {
		if got := st.Units[index].NativeRecordByte34; got != 0xA0 {
			t.Fatalf("unit%d byte34=%#x, want 0xa0 after left-step selector0", index, got)
		}
	}
}

func TestBattleWalkDoesNotGeneralizeSelector0ToRightStep(t *testing.T) {
	st, err := battle.Load("../../assets/maps/map25/map25_units.json")
	if err != nil {
		t.Fatal(err)
	}
	trigger := st.Units[0]
	trigger.NativeRecordByte6 = 1
	trigger.HasNativeRecordByte6 = true
	trigger.SetMapPlacement(9, 36, 3)
	if err := trigger.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	for index := 39; index <= 44; index++ {
		st.Units[index].NativeRecordByte34 = 0xA7
		st.Units[index].HasNativeRecordByte34 = true
	}
	g := &Game{
		m:  &MapData{TileW: 24, TileH: 24},
		st: st,
		walk: &walkAnim{
			u:    trigger,
			path: []battle.Cell{{X: 9, Y: 36}, {X: 10, Y: 36}},
		},
	}
	for tick := 1; tick <= 7; tick++ {
		g.stepBattleWalk()
	}
	for index := 39; index <= 44; index++ {
		if got := st.Units[index].NativeRecordByte34; got != 0xA7 {
			t.Fatalf("unit%d byte34=%#x, right step incorrectly ran selector0", index, got)
		}
	}
}

func TestBattleWalkActivatesMap26Event63OnlyAfterLeftStepCommit(t *testing.T) {
	st, err := battle.Load("../../assets/maps/map26/map26_units.json")
	if err != nil {
		t.Fatal(err)
	}
	x, y := -1, -1
	for cell, slot := range st.NativeFieldEventSlots {
		if slot < 0 || slot >= len(st.NativeFieldEvents) {
			continue
		}
		event := st.NativeFieldEvents[slot]
		if event.EventID == 62 && event.Selector == 0 && cell%st.W > 0 {
			x, y = cell%st.W, cell/st.W
			break
		}
	}
	if x < 0 {
		t.Fatal("map26 has no event62 selector0 cell reachable by a left step")
	}
	trigger := st.Units[0]
	trigger.SetMapPlacement(x+1, y, 1)
	if err := trigger.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	st.NativeRoundCounter = 8
	g := &Game{
		m:  &MapData{TileW: 24, TileH: 24},
		st: st,
		walk: &walkAnim{
			u:    trigger,
			path: []battle.Cell{{X: x + 1, Y: y}, {X: x, Y: y}},
		},
	}
	for tick := 1; tick <= 6; tick++ {
		g.stepBattleWalk()
		if st.NativeTurnEventControls[0].Turn != 0xff || st.NativeEventState[17] != 0 {
			t.Fatalf("tick %d activated event63 before selector0 commit", tick)
		}
	}
	g.stepBattleWalk()
	if g.loadErr != "" || g.walk != nil ||
		st.NativeTurnEventControls[0] != (battle.NativeTurnEventControl{Turn: 9, EventID: 63, RawCamp: 0}) ||
		st.NativeEventState[17] != 1 {
		t.Fatalf("event62 player path err=%q walk=%v row=%#v state17=%d", g.loadErr, g.walk, st.NativeTurnEventControls[0], st.NativeEventState[17])
	}
}

func TestOriginalActingUpdatesMaterializedRawPresentation(t *testing.T) {
	slot := 0
	u := &battle.Unit{X: 4, Y: 5}
	if err := u.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		m:  &MapData{TileW: 24, TileH: 24},
		st: &battle.State{Units: []*battle.Unit{u}},
	}
	j := &actPoseJob{acting: []campaign.ActingFrame{{
		Beats: 1,
		Units: []campaign.ActingUnit{{Slot: &slot, Pose: 2}},
	}}}
	g.beginActingFrame(j)
	for tick := 1; tick <= 6; tick++ {
		g.stepOriginalActing(j)
		raw := u.NativeMapPresentation
		if raw.X != 4 || raw.Y != 5 || raw.Pose != 2 || raw.Motion != byte(tick) {
			t.Fatalf("acting tick %d raw=%+v", tick, raw)
		}
	}
	g.stepOriginalActing(j)
	if u.NativeMapPresentation != (battle.NativeMapPresentationState{X: 4, Y: 4, Pose: 2}) ||
		u.X != 4 || u.Y != 4 {
		t.Fatalf("acting completion raw=%+v normalized=(%d,%d)",
			u.NativeMapPresentation, u.X, u.Y)
	}
}

func TestGameCursorUsesMaterializedNativeMapView(t *testing.T) {
	st := &battle.State{W: 30, H: 30}
	if err := st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 4, CameraY: 5, CursorX: 15, CursorY: 11,
		VisibleCursorX: 11, VisibleCursorY: 6,
	}); err != nil {
		t.Fatal(err)
	}
	if !st.MaterializeNativeMapHUDState(1, 1, 0xf2) {
		t.Fatal("HUD state rejected")
	}
	g := &Game{m: &MapData{W: 30, H: 30, TileW: 24, TileH: 24}, st: st}
	g.moveMapCursor(1, 0)
	if g.curX != 16 || g.curY != 11 || g.camX != 5*24 || g.camY != 5*24 {
		t.Fatalf("game view cursor=(%d,%d) camera=(%v,%v)",
			g.curX, g.curY, g.camX, g.camY)
	}
	if st.NativeMapHUDState.AnchorX != 1 {
		t.Fatalf("HUD anchor did not follow visible cursor: %+v", st.NativeMapHUDState)
	}
}

func TestScreenshotCursorUsesNativeViewAndHUDStateMachine(t *testing.T) {
	st := &battle.State{W: 30, H: 30}
	if err := st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
		VisibleCursorX: 7, VisibleCursorY: 4,
	}); err != nil {
		t.Fatal(err)
	}
	if !st.MaterializeNativeMapHUDState(1, 1, 1) {
		t.Fatal("HUD state rejected")
	}
	g := &Game{m: &MapData{W: 30, H: 30, TileW: 24, TileH: 24}, st: st}
	g.syncNativeMapView()
	if !g.positionScreenshotCursor(8, 15) {
		t.Fatal("screenshot cursor rejected")
	}
	view := st.NativeMapViewState
	if view.CameraX != 1 || view.CameraY != 13 ||
		view.CursorX != 8 || view.CursorY != 15 ||
		view.VisibleCursorX != 7 || view.VisibleCursorY != 2 ||
		g.curX != 8 || g.curY != 15 {
		t.Fatalf("screenshot view=%+v game=(%d,%d)", view, g.curX, g.curY)
	}
	if st.NativeMapHUDState.AnchorX != 1 {
		t.Fatalf("screenshot HUD anchor=%+v", st.NativeMapHUDState)
	}
}

func TestGameMaterializesEditableNativeMapRuntime(t *testing.T) {
	rangeMode := 1
	g := &Game{
		m:  &MapData{W: 24, H: 24, TileW: 24, TileH: 24},
		st: &battle.State{W: 24, H: 24},
	}
	n := &campaign.Node{
		NativeMapView: &campaign.NativeMapViewConfig{
			CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
			VisibleCursorX: 7, VisibleCursorY: 4,
			RangeMode: &rangeMode,
		},
		NativeMapHUD: &campaign.NativeMapHUDConfig{
			DisplayGateA: 1, DisplayGateB: 1, AnchorX: 1,
		},
	}
	if !g.materializeNativeMapRuntime(n) {
		t.Fatal(g.loadErr)
	}
	if !g.st.HasNativeMapViewState || !g.st.HasNativeMapHUDState ||
		!g.st.HasNativeMapRangeModeState || g.st.NativeMapRangeMode != 1 ||
		g.curX != 8 || g.curY != 17 || g.camX != 24 || g.camY != 13*24 {
		t.Fatalf("materialized game=%+v view=%+v HUD=%+v",
			g, g.st.NativeMapViewState, g.st.NativeMapHUDState)
	}
}

func TestGameMaterializesSourcedViewWithoutInventingHUD(t *testing.T) {
	rangeMode := 0
	g := &Game{
		m:  &MapData{W: 31, H: 57, TileW: 24, TileH: 24},
		st: &battle.State{W: 31, H: 57},
	}
	n := &campaign.Node{NativeMapView: &campaign.NativeMapViewConfig{
		CameraX: 9, CameraY: 49, CursorX: 14, CursorY: 54,
		VisibleCursorX: 5, VisibleCursorY: 5, RangeMode: &rangeMode,
	}}
	if !g.materializeNativeMapRuntime(n) {
		t.Fatal(g.loadErr)
	}
	if !g.st.HasNativeMapViewState || g.st.HasNativeMapHUDState ||
		!g.st.HasNativeMapRangeModeState || g.st.NativeMapRangeMode != 0 ||
		g.curX != 14 || g.curY != 54 || g.camX != 9*24 || g.camY != 49*24 {
		t.Fatalf("materialized view-only game=%+v view=%+v HUD=%+v",
			g, g.st.NativeMapViewState, g.st.NativeMapHUDState)
	}
}

func TestBattleCh27MaterializesVerifiedViewWithPersistentHUD(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", "assets/scenarios/campaign_full.json")
	t.Setenv("FD2_CAMP_NODE", "battle_ch27")
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.st == nil || !g.st.HasNativeMapViewState ||
		!g.st.HasNativeMapRangeModeState || g.st.NativeMapRangeMode != 0 ||
		!g.st.HasNativeMapHUDState {
		t.Fatalf("chapter27 native state=%#v", g.st)
	}
	if hud := g.st.NativeMapHUDState; hud.DisplayGateA != 1 ||
		hud.DisplayGateB != 1 || hud.AnchorX != 1 {
		t.Fatalf("chapter27 inherited HUD=%+v", hud)
	}
	view := g.st.NativeMapViewState
	if view.CameraX != 9 || view.CameraY != 49 ||
		view.CursorX != 14 || view.CursorY != 54 ||
		view.VisibleCursorX != 5 || view.VisibleCursorY != 5 ||
		g.curX != 14 || g.curY != 54 ||
		g.camX != 9*24 || g.camY != 49*24 {
		t.Fatalf("chapter27 view=%+v camera=(%v,%v) cursor=(%d,%d)",
			view, g.camX, g.camY, g.curX, g.curY)
	}
}

func TestGameMaterializesInheritedHUDWithoutOverwritingSavedGateOrAnchor(t *testing.T) {
	rangeMode := 0
	g := &Game{
		m:  &MapData{W: 31, H: 57, TileW: 24, TileH: 24},
		st: &battle.State{W: 31, H: 57},
		nativeMapHUDPersistent: battle.NativeMapHUDPersistentState{
			DisplayGateA: 0, AnchorX: 0xf2,
			HasDisplayGateA: true, HasAnchorX: true,
		},
	}
	n := &campaign.Node{
		NativeMapView: &campaign.NativeMapViewConfig{
			CameraX: 9, CameraY: 49, CursorX: 14, CursorY: 54,
			VisibleCursorX: 5, VisibleCursorY: 5, RangeMode: &rangeMode,
		},
		NativeMapHUDInherited: &campaign.NativeMapHUDInheritedConfig{DisplayGateB: 1},
	}
	if !g.materializeNativeMapRuntime(n) {
		t.Fatal(g.loadErr)
	}
	if hud := g.st.NativeMapHUDState; !g.st.HasNativeMapHUDState ||
		hud.DisplayGateA != 0 || hud.DisplayGateB != 1 || hud.AnchorX != 0xf2 {
		t.Fatalf("inherited runtime HUD=%+v valid=%v", hud, g.st.HasNativeMapHUDState)
	}
}

func TestGameRejectsInvalidEditableNativeMapRuntime(t *testing.T) {
	rangeMode := 1
	g := &Game{
		m:  &MapData{W: 24, H: 24, TileW: 24, TileH: 24},
		st: &battle.State{W: 24, H: 24},
	}
	n := &campaign.Node{
		NativeMapView: &campaign.NativeMapViewConfig{
			// 可見游標落在 13×8 視窗外。原版沒有 visible == cursor - camera
			// 的恆等式，執行期也不夾這個全域；但節點常數是進場就要畫出來的
			// 靜止視圖，游標框與指令環會立刻消費它，所以這條入口要求它在
			// 視窗內。
			CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
			VisibleCursorX: 13, VisibleCursorY: 4,
			RangeMode: &rangeMode,
		},
		NativeMapHUD: &campaign.NativeMapHUDConfig{
			DisplayGateA: 1, DisplayGateB: 1, AnchorX: 1,
		},
	}
	if g.materializeNativeMapRuntime(n) || g.loadErr == "" ||
		g.st.HasNativeMapViewState || g.st.HasNativeMapHUDState {
		t.Fatalf("invalid native runtime was not rejected: err=%q state=%+v", g.loadErr, g.st)
	}
}

func TestGameRejectsUnmaterializedDynamicCampaignSelector(t *testing.T) {
	rangeMode := 11
	g := &Game{
		m:  &MapData{W: 24, H: 24, TileW: 24, TileH: 24},
		st: &battle.State{W: 24, H: 24},
	}
	n := &campaign.Node{
		NativeMapView: &campaign.NativeMapViewConfig{
			CameraX: 1, CameraY: 13, CursorX: 8, CursorY: 17,
			VisibleCursorX: 7, VisibleCursorY: 4,
			RangeMode: &rangeMode,
		},
		NativeMapHUD: &campaign.NativeMapHUDConfig{
			DisplayGateA: 1, DisplayGateB: 1, AnchorX: 1,
		},
	}
	if g.materializeNativeMapRuntime(n) || g.loadErr == "" ||
		g.st.HasNativeMapRangeModeState {
		t.Fatalf("runtime-owned selector was accepted as campaign bootstrap: err=%q state=%+v", g.loadErr, g.st)
	}
}
