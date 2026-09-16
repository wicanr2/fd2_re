package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// nativeMapSelectionOverlay 決定這一幀比 0x11CAC 多畫什麼：
//   - 玩家選了自己還沒行動的單位、正在挑目的地（移動範圍亮著、還沒移動、指令環
//     沒開）：0x18B84 的重繪，多一個單位資訊視窗（0x18C6D）；
//   - 指令環開著：0x1741C 把四張 FDOTHER#2 圖示貼在游標四周（穩態用 0x179D5 的
//     停留位移，開合中用各自的動畫位移），被選方向依 [0x53C13] 閃爍。
//
// 回傳 nil 表示照 0x11CAC。
func (g *Game) nativeMapSelectionOverlay(state *battle.State) (func([]byte, int, int) error, error) {
	if g == nil || state == nil || !state.HasNativeMapViewState {
		return nil, nil
	}
	if g.ring && g.sel != nil && g.walk == nil && len(g.nativeActionCellsRaw) == nativeActionOverlayCellCount {
		return g.nativeMapActionOverlay(state), nil
	}
	if g.sel == nil || g.moved || g.ring || g.walk != nil ||
		g.castSp != nil || g.nativeCommand0Targeting || g.reach == nil ||
		g.sel.Camp != battle.Own {
		return nil, nil
	}
	if g.nativeMapUnitWindowAssets == nil {
		assets, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
		if err != nil {
			return nil, fmt.Errorf("native map unit window: %w", err)
		}
		g.nativeMapUnitWindowAssets = &assets
	}
	record, err := battle.NativeItemPanelRecordForUnit(g.sel)
	if err != nil {
		return nil, fmt.Errorf("native map unit window: %w", err)
	}
	visibleX := state.NativeMapViewState.VisibleCursorX
	assets := g.nativeMapUnitWindowAssets
	return func(frame []byte, stride, viewportBase int) error {
		return battle.RenderNativeMapUnitWindow(*assets, record, frame[viewportBase:], stride, visibleX)
	}, nil
}

// nativeMapActionOverlay 把指令環貼進工作緩衝，與 drawNativeActionOverlay 同一組
// 位移、格子索引與閃爍規則，只是目的地換成 indexed 畫面。
func (g *Game) nativeMapActionOverlay(state *battle.State) func([]byte, int, int) error {
	overlay := g.nativeActionOverlayState()
	frame, closing := g.actionOverlayRenderState()
	offsets, err := fdother.ActionOverlayFrameOffsets(frame, closing)
	if err != nil {
		return func([]byte, int, int) error { return err }
	}
	steady := !g.actionOverlayBlocksInput()
	if steady {
		offsets = [4]int{-0x23a0, 0x378, 0x3a8, 0x2ac0}
	}
	view := state.NativeMapViewState
	ringSel, blink := g.ringSel, g.actionOverlayBlink.Phase
	cells := g.nativeActionCellsRaw
	return func(buf []byte, stride, viewportBase int) error {
		origin, err := fdother.ActionOverlayOrigin(view.VisibleCursorX, view.VisibleCursorY)
		if err != nil {
			return err
		}
		for direction, offset := range offsets {
			index, err := overlay.CellIndex(direction)
			if steady && direction == ringSel && fdother.ActionOverlayAcceptsDirection(overlay.Availability, direction) {
				index, err = overlay.SelectedCellIndex(direction, blink)
			}
			if err != nil {
				return err
			}
			if index < 0 || index >= len(cells) {
				return fmt.Errorf("native action overlay cell %d is absent", index)
			}
			pos := origin + offset
			if pos < 0 || pos >= len(buf) {
				return fmt.Errorf("native action overlay position %d is invalid", pos)
			}
			if err := cells[index].BlitAt(buf, stride, pos%stride, pos/stride); err != nil {
				return err
			}
		}
		return nil
	}
}
