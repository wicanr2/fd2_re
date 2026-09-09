package main

import (
	"errors"
	"fmt"

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
	// 地形尺寸屬於目前 LOADCH；素材中的 cell +3 只作來源證據。
	// 原版 0x4DBFC 在載入後逐格寫入 0xff，與 battle.LoadScenario 相同。
	// 複製 serialized 值會讓整個故事背景誤走 0x11EEE 的亮色 LUT 分支。
	candidate.W, candidate.H = g.m.W, g.m.H
	candidate.NativeTileBlitModes = make([]byte, len(g.m.NativeTileBlitModes))
	for i := range candidate.NativeTileBlitModes {
		candidate.NativeTileBlitModes[i] = 0xff
	}
	candidate.Units = nil
	candidate.Roster = nil
	candidate.NativeMapSelectorCache = nil
	candidate.NativeMapSelectorError = nil
	candidate.HasNativeMapCycleState = false
	candidate.HasNativeTerrainPhaseState = false
	candidate.HasNativeMapBinaryTimingState = false
	actors := make([]*battle.Unit, len(g.storyActors))
	for index := range g.storyActors {
		// 快取重建不是再次建構角色；constructor 會把 pose/motion 清零。
		// 私有複本只用來取得 slot，不覆蓋已完成的 ACT／scroll_step。
		clone := g.storyActors[index]
		actors[index] = &clone
	}
	if err := candidate.AppendNativeMapSelectorBatch(actors); err != nil {
		return err
	}
	for index, actor := range actors {
		original := &g.storyActors[index]
		if !original.HasNativeMapPresentation &&
			!actor.SetMapPlacement(original.X, original.Y, original.Dir) {
			return errors.New("native story map: spawned actor placement is invalid")
		}
	}
	if err := candidate.MaterializeNativeMapViewState(g.storyNativeMapView); err != nil {
		return fmt.Errorf("native story map state view %+v: %w", g.storyNativeMapView, err)
	}
	if !candidate.MaterializeNativeMapRangeMode(0) {
		return errors.New("native story map: opening selector 0 is unavailable")
	}
	// 場景視窗不畫戰鬥 HUD；anchor 仍保留資料映像的合法初值，避免把
	// 隱藏狀態誤解成不存在的版面資料。
	if !candidate.MaterializeNativeMapHUDState(0, 0, 1) {
		return errors.New("native story map: hidden HUD state is invalid")
	}
	for index, actor := range actors {
		if !g.storyActors[index].HasNativeMapPresentation {
			g.storyActors[index].NativeMapPresentation = actor.NativeMapPresentation
			g.storyActors[index].HasNativeMapPresentation = true
		}
		g.storyActors[index].MapSelectorSlot = actor.MapSelectorSlot
		g.storyActors[index].HasMapSelectorSlot = actor.HasMapSelectorSlot
		candidate.Units[index] = &g.storyActors[index]
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
		return fmt.Errorf("native story map frame view %+v: %w", g.storyNativeMapView, err)
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
