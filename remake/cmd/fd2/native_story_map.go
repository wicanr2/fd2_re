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
	candidate := *source
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
	previous := g.st
	g.st = g.storyNativeMapState
	err := g.composeNativeMapFrame()
	g.storyNativeMapState = g.st
	g.st = previous
	return err
}
