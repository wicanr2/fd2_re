package main

import (
	"errors"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// materializeNativeStoryMapState 建立 LOADCH 場景專用的索引地圖載體。
// 場景沿用原始 terrain/unit constructor 與 raw view，但明確不顯示戰鬥 HUD；
// 它不會把 cutscene 單位升格成可操作的 battle.State。
func (g *Game) materializeNativeStoryMapState(source *battle.State) error {
	if g == nil || source == nil || g.m == nil || !g.hasStoryNativeMapView {
		return errors.New("native story map: source, field, or raw view is unavailable")
	}
	if g.m.W <= 0 || g.m.H <= 0 || len(g.m.Tiles) != g.m.W*g.m.H ||
		len(g.m.NativeTileBlitModes) != len(g.m.Tiles) {
		return errors.New("native story map: LOADCH terrain renderer inputs are incomplete")
	}
	candidate := *source
	// 劇情地圖的 terrain owner 是目前 LOADCH 的可編輯地圖，不是建立
	// roster fixture 時所沿用的上一個 battle.State。逐格複製可避免後續
	// overlay 修改回寫 MapData，也確保 buildNativeMapFrameInput 只收到同一張圖。
	candidate.W, candidate.H = g.m.W, g.m.H
	candidate.NativeTileBlitModes = append([]byte(nil), g.m.NativeTileBlitModes...)
	candidate.Units = nil
	candidate.Roster = nil
	candidate.NativeMapSelectorCache = nil
	candidate.NativeMapSelectorError = nil
	candidate.HasNativeMapCycleState = false
	candidate.HasNativeTerrainPhaseState = false
	candidate.HasNativeMapBinaryTimingState = false
	actors := make([]*battle.Unit, len(g.storyActors))
	for index := range g.storyActors {
		actors[index] = &g.storyActors[index]
	}
	if err := candidate.AppendNativeMapSelectorBatch(actors); err != nil {
		return err
	}
	if err := candidate.MaterializeNativeMapViewState(g.storyNativeMapView); err != nil {
		return err
	}
	if !candidate.MaterializeNativeMapRangeMode(0) {
		return errors.New("native story map: opening selector 0 is unavailable")
	}
	// 場景視窗不畫戰鬥 HUD；anchor 仍保留資料映像的合法初值，避免把
	// 隱藏狀態誤解成不存在的版面資料。
	if !candidate.MaterializeNativeMapHUDState(0, 0, 1) {
		return errors.New("native story map: hidden HUD state is invalid")
	}
	g.storyNativeMapState = &candidate
	return nil
}

// appendNativeStoryMapActors 在 SPAWN 後延續同一 first-seen selector cache。
// storyActors append 可能搬移底層陣列，所以成功後一併重建全部 unit pointers。
func (g *Game) appendNativeStoryMapActors(start int) error {
	if g == nil || g.storyNativeMapState == nil || start < 0 || start > len(g.storyActors) {
		return errors.New("native story map: append boundary is unavailable")
	}
	added := make([]*battle.Unit, 0, len(g.storyActors)-start)
	for index := start; index < len(g.storyActors); index++ {
		added = append(added, &g.storyActors[index])
	}
	if err := battle.MaterializeNativeMapSelectorSlots(added, g.storyNativeMapState.NativeMapSelectorCache); err != nil {
		return err
	}
	g.storyNativeMapState.Units = g.storyNativeMapState.Units[:0]
	for index := range g.storyActors {
		g.storyNativeMapState.Units = append(g.storyNativeMapState.Units, &g.storyActors[index])
	}
	return nil
}

func (g *Game) composeNativeStoryMapFrame() error {
	if g == nil || g.st != nil || g.storyNativeMapState == nil || !g.hasStoryNativeMapView {
		return errors.New("native story map: caller-owned scene state is unavailable")
	}
	if err := g.storyNativeMapState.MaterializeNativeMapViewState(g.storyNativeMapView); err != nil {
		return err
	}
	// map32 前兩筆特殊劇情角色不在目前閉合的 constructor table 範圍，
	// 因而沒有 0x129EC gate 的 raw race/class。99% 玩家可見模式只在本次
	// 背景合成的私有 clone 採保守前景重畫；不可污染 storyActors、戰鬥或存檔。
	frameState := *g.storyNativeMapState
	frameState.Units = make([]*battle.Unit, len(g.storyNativeMapState.Units))
	for index, unit := range g.storyNativeMapState.Units {
		if unit == nil {
			return errors.New("native story map: roster contains a nil unit")
		}
		clone := *unit
		if !clone.HasNativeRecordRace || !clone.HasNativeRecordClass {
			if !clone.HasNativeMapPresentation || !clone.HasNativeRecordByte5 || !clone.HasBattleFig {
				return errors.New("native story map: approximate foreground gate lacks required provenance")
			}
			if !clone.HasNativeRecordRace {
				clone.NativeRecordRace, clone.HasNativeRecordRace = 0, true
			}
			if !clone.HasNativeRecordClass {
				clone.NativeRecordClass, clone.HasNativeRecordClass = 0, true
			}
		}
		frameState.Units[index] = &clone
	}
	previous := g.st
	g.st = &frameState
	err := g.composeNativeMapFrame()
	// composeNativeMapFrame 會更新時序全域值，但私有角色clone只供畫面使用；
	// 只發布時序，不以近似角色取代正式場景角色。
	g.storyNativeMapState.NativeMapCycleState = g.st.NativeMapCycleState
	g.storyNativeMapState.HasNativeMapCycleState = g.st.HasNativeMapCycleState
	g.storyNativeMapState.NativeTerrainPhaseState = g.st.NativeTerrainPhaseState
	g.storyNativeMapState.HasNativeTerrainPhaseState = g.st.HasNativeTerrainPhaseState
	g.storyNativeMapState.NativeTerrainFlipState = g.st.NativeTerrainFlipState
	g.storyNativeMapState.NativeUnitPixelShiftState = g.st.NativeUnitPixelShiftState
	g.storyNativeMapState.HasNativeMapBinaryTimingState = g.st.HasNativeMapBinaryTimingState
	g.st = previous
	return err
}
