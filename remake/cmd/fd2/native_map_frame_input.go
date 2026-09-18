package main

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativeMapFrameRuntime is the still-caller-owned raw global subset needed by
// one steady 0x11cac frame. It deliberately has no conversion from the
// remake's 640x400 pixel camera or normalized selection/highlight state.
type nativeMapFrameRuntime struct {
	HUD             indexedmap.NativeMapHUDInput
	ChapterAuxPhase int
}

// buildNativeMapFrameInput joins the all-or-nothing original asset bundle,
// exact exported FDFIELD cells, battle-local raw roster and explicit runtime
// globals. Any incomplete provenance rejects the entire frame input.
func buildNativeMapFrameInput(
	assets *nativeMapAssets,
	field *MapData,
	state *battle.State,
	runtime nativeMapFrameRuntime,
) (indexedmap.NativeFrameInput, error) {
	if !nativeMapAssetsAvailable(assets) || field == nil || state == nil {
		return indexedmap.NativeFrameInput{}, errors.New("native map frame: incomplete assets, field, or battle state")
	}
	if !state.HasNativeMapHUDState {
		return indexedmap.NativeFrameInput{}, errors.New("native map frame: HUD runtime state is not materialized")
	}
	if field.W <= 0 || field.H <= 0 || len(field.Tiles) != field.W*field.H ||
		!bytes.Equal(field.NativeTerrainControl, assets.Controls) {
		return indexedmap.NativeFrameInput{}, errors.New("native map frame: editable field does not match native FDSHAP controls")
	}
	if !state.HasNativeMapViewState || !state.HasNativeMapRangeModeState ||
		state.NativeMapRangeMode < 0 || state.NativeMapRangeMode > 5 {
		return indexedmap.NativeFrameInput{}, errors.New("native map frame: raw runtime globals are outside verified bounds")
	}
	view := state.NativeMapViewState
	// 撿走寶箱那一格的圖塊由 0x12263 就地 +1，繪圖端跟著讀那份可變緩衝；可編輯地圖的
	// field.Tiles 是初始值，直接拿去畫會讓箱子永遠關著。
	drawTiles := field.Tiles
	if mutable, ok := state.NativeMapDrawTiles(); ok {
		if len(mutable) != len(field.Tiles) {
			return indexedmap.NativeFrameInput{}, errors.New("native map frame: mutable map buffer does not match the editable field")
		}
		drawTiles = mutable
	}
	cells, err := indexedmap.BuildNativeTerrainCells(drawTiles, state.NativeTileBlitModes)
	if err != nil {
		return indexedmap.NativeFrameInput{}, fmt.Errorf("native map frame: terrain cells: %w", err)
	}
	roster, err := state.NativeMapFrameRoster()
	if err != nil {
		return indexedmap.NativeFrameInput{}, fmt.Errorf("native map frame: roster: %w", err)
	}
	lutIndex, err := fdother.NativeTerrainLUTIndex(roster.TerrainPhase)
	if err != nil || lutIndex < 0 || lutIndex >= len(assets.LUTs) || len(assets.LUTs[lutIndex]) != 256 {
		return indexedmap.NativeFrameInput{}, errors.New("native map frame: terrain phase LUT is incomplete")
	}
	frame := indexedmap.FrameInput{
		TerrainBank: assets.Terrain, RangeBank: assets.Range,
		UnitBank: assets.Units, ForegroundBank: assets.Terrain,
		SelectorCache: state.NativeMapSelectorCache,
		Cells:         cells, Controls: assets.Controls, LUT: assets.LUTs[lutIndex],
		MapWidth: field.W, CameraX: view.CameraX, CameraY: view.CameraY,
		Flip: roster.TerrainFlip, TerrainCycle: roster.Cycles.Idle,
		IdleCycle: roster.Cycles.Idle, MovingCycle: roster.Cycles.Moving,
		PixelShift: roster.UnitPixelShift,
		RangeMode:  state.NativeMapRangeMode, CursorX: view.CursorX, CursorY: view.CursorY,
		Units: roster.Units, ForegroundUnits: roster.Foreground,
		ChapterAux: assets.ChapterAux, ChapterAuxPhase: runtime.ChapterAuxPhase,
	}
	hud := runtime.HUD
	hud.DisplayGateA = state.NativeMapHUDState.DisplayGateA != 0
	hud.DisplayGateB = state.NativeMapHUDState.DisplayGateB != 0
	hud.AnchorX = state.NativeMapHUDState.AnchorX
	return indexedmap.NativeFrameInput{
		Frame: frame, HUD: hud, Frames: assets.Frames,
		HUDTerrain: assets.Terrain, HUDUnits: assets.Units,
		HUDCache: state.NativeMapSelectorCache,
	}, nil
}
