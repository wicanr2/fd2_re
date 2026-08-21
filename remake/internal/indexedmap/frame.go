// Package indexedmap composes the verified steady native tactical-map passes.
// It intentionally has no Ebiten dependency: callers supply indexed buffers
// and the still-separate native HUD pass.
package indexedmap

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const (
	workStride = fdicon.NativeMapStride
	workBase   = 0x8088
	viewWidth  = 320 // VGA row stride, not the steady copied width
	viewHeight = 192
	// 0x11cac calls 0x11eb0(dst=A000:0504, dstStride=320,
	// src=work+0x8088, srcStride=456, width=312, height=192).
	steadyViewportOffset = 0x504
	steadyViewportWidth  = 312
	NativeMapVGASize     = 320 * 200
	// NativeUnitPresentWorkSize is 0x22253's exact temporary allocation.
	NativeUnitPresentWorkSize = 0x25680
)

// SeedNativeCh23Staging reproduces the zero-offset chapter-23 case of
// 0x11EEE before the shared terrain/range/object/HUD pipeline runs. The raw
// source is exactly 312x192 with stride 312; the destination begins at
// work+0x8088 with stride 456. The helper commits atomically and deliberately
// does not name the staging pixels as a background or transition.
func SeedNativeCh23Staging(work, staging []byte) error {
	if len(work) != NativeUnitPresentWorkSize || len(staging) != 312*viewHeight {
		return errors.New("indexedmap: incomplete native ch23 staging buffers")
	}
	last := workBase + (viewHeight-1)*workStride + steadyViewportWidth
	if last > len(work) {
		return errors.New("indexedmap: native ch23 staging exceeds work buffer")
	}
	next := append([]byte(nil), work...)
	for row := 0; row < viewHeight; row++ {
		src := row * steadyViewportWidth
		dst := workBase + row*workStride
		copy(next[dst:dst+steadyViewportWidth], staging[src:src+steadyViewportWidth])
	}
	copy(work, next)
	return nil
}

// FrameInput is the raw, already-selected input required by 0x11cac's normal
// redraw.  It deliberately keeps all native selectors raw: the caller owns
// resource selection, map lifetime, palette phase, and unit materialization.
type FrameInput struct {
	TerrainBank, RangeBank, UnitBank, ForegroundBank *fdicon.Bank
	SelectorCache                                    *fdicon.NativeSelectorCache
	Cells                                            []fdicon.NativeTerrainCell
	Controls, LUT                                    []byte
	MapWidth, CameraX, CameraY                       int
	Flip, TerrainCycle, IdleCycle, MovingCycle       int
	PixelShift                                       int
	RangeMode, CursorX, CursorY                      int
	Units                                            []fdicon.NativeUnitLayerEntry
	ForegroundUnits                                  []fdicon.NativeForegroundLayerEntry
}

// NativeFrameInput is the complete, directly composable steady redraw slice.
// It binds the separately verified 0x1acf3 resources/input to 0x11cac's
// terrain/range/unit/foreground scheduler without allowing a caller to swap
// in an approximation at the HUD boundary.
type NativeFrameInput struct {
	Frame                FrameInput
	HUD                  NativeMapHUDInput
	Frames               NativeMapHUDFrames
	HUDTerrain, HUDUnits *fdicon.Bank
	HUDCache             *fdicon.NativeSelectorCache
}

// NativeTransitionFrameInput is the raw layer subset used by 0x24618. Unlike
// the steady tactical redraw, the transition starts with terrain only; the
// middle 0x127a9 callback adds unit and foreground layers between two LUT
// remaps. Range and HUD are intentionally absent from this contract.
type NativeTransitionFrameInput struct {
	TerrainBank, UnitBank, ForegroundBank *fdicon.Bank
	SelectorCache                         *fdicon.NativeSelectorCache
	Cells                                 []fdicon.NativeTerrainCell
	Controls, TerrainLUT                  []byte
	MapWidth, CameraX, CameraY            int
	Flip, TerrainCycle, IdleCycle         int
	MovingCycle, PixelShift               int
	Units                                 []fdicon.NativeUnitLayerEntry
	ForegroundUnits                       []fdicon.NativeForegroundLayerEntry
}

// NativeUnitPresentStripLayout is 0x22390..0x22434's direct work-buffer to
// VGA bridge. Offsets are relative to their supplied buffers, not original
// process addresses.
type NativeUnitPresentStripLayout struct {
	WorkOffset int
	VGAOffset  int
	Rows       int
}

// NativeUnitPresentStripLayoutFor reproduces the two native row ranges.
// A target on cameraY copies 18 rows beginning at the target row. Every other
// target copies 24 rows beginning six rows above it.
func NativeUnitPresentStripLayoutFor(mapX, mapY, cameraX, cameraY int) NativeUnitPresentStripLayout {
	dx, dy := mapX-cameraX, mapY-cameraY
	work := workBase + dx*24 + dy*24*workStride
	vga := dx*24 + dy*24*viewWidth
	rows := 24
	if mapY == cameraY {
		rows = 18
	} else {
		work -= 6 * workStride
		vga -= 6 * viewWidth
	}
	return NativeUnitPresentStripLayout{WorkOffset: work, VGAOffset: vga, Rows: rows}
}

// RunNativeUnitPresentStripBridge reproduces the progressive direct VGA writes
// after 0x22253's bridge-only 0x22046 pass. Each row copies 24 bytes from the
// 456-stride work buffer to the 320-stride VGA buffer, then invokes delay10ms.
// Unlike a full 0x11eb0 viewport present, every completed row is immediately
// observable. Bounds are preflighted before the first write.
func RunNativeUnitPresentStripBridge(
	work, vga []byte,
	mapX, mapY, cameraX, cameraY int,
	delay10ms func(row int) error,
) error {
	if len(work) != NativeUnitPresentWorkSize || len(vga) < viewWidth*viewHeight || delay10ms == nil {
		return errors.New("indexedmap: incomplete native unit-present strip bridge")
	}
	layout := NativeUnitPresentStripLayoutFor(mapX, mapY, cameraX, cameraY)
	lastWork := layout.WorkOffset + (layout.Rows-1)*workStride + 24
	lastVGA := layout.VGAOffset + (layout.Rows-1)*viewWidth + 24
	if layout.WorkOffset < 0 || layout.VGAOffset < 0 || lastWork > len(work) || lastVGA > len(vga) {
		return errors.New("indexedmap: native unit-present strip exceeds buffer")
	}
	for row := 0; row < layout.Rows; row++ {
		src := layout.WorkOffset + row*workStride
		dst := layout.VGAOffset + row*viewWidth
		copy(vga[dst:dst+24], work[src:src+24])
		if err := delay10ms(row); err != nil {
			return fmt.Errorf("indexedmap: unit-present strip row %d delay: %w", row, err)
		}
	}
	return nil
}

// ComposeNativeUnitPresentTerrainSnapshot reproduces 0x22253's initial
// 0x11eee terrain-only draw into its exact 0x25680-byte buffer. Units,
// foreground, range and HUD are intentionally excluded: 0x22470/0x22046 call
// 0x127a9 at their own recovered points.
func ComposeNativeUnitPresentTerrainSnapshot(work []byte, in NativeTransitionFrameInput) error {
	if len(work) != NativeUnitPresentWorkSize {
		return errors.New("indexedmap: native unit-present work must be exactly 0x25680 bytes")
	}
	if in.TerrainBank == nil || in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 ||
		len(in.TerrainLUT) != 256 {
		return errors.New("indexedmap: incomplete native unit-present terrain input")
	}
	frame := append([]byte(nil), work...)
	baseX, baseY := workBase%workStride, workBase/workStride
	if err := in.TerrainBank.BlitNativeTerrainRegion(
		frame, workStride, baseX, baseY,
		in.MapWidth, in.Cells, in.Controls,
		in.CameraX, in.CameraY, 13, 8,
		in.Flip, in.TerrainCycle, in.TerrainLUT,
	); err != nil {
		return fmt.Errorf("indexedmap: unit-present terrain: %w", err)
	}
	copy(work, frame)
	return nil
}

// ComposeNative24B4DStaging reproduces 0x24B4D's one-time 0x11EEE terrain
// draw. It differs from the unit-present snapshot only in the proven ninth
// tile row, which supplies the second row-shifted 312x192 viewport.
func ComposeNative24B4DStaging(work []byte, in NativeTransitionFrameInput) error {
	if len(work) != NativeUnitPresentWorkSize {
		return errors.New("indexedmap: native 0x24B4D work must be exactly 0x25680 bytes")
	}
	if in.TerrainBank == nil || in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 ||
		len(in.TerrainLUT) != 256 {
		return errors.New("indexedmap: incomplete native 0x24B4D terrain input")
	}
	frame := append([]byte(nil), work...)
	baseX, baseY := workBase%workStride, workBase/workStride
	if err := in.TerrainBank.BlitNativeTerrainRegion(
		frame, workStride, baseX, baseY,
		in.MapWidth, in.Cells, in.Controls,
		in.CameraX, in.CameraY, 13, 9,
		in.Flip, in.TerrainCycle, in.TerrainLUT,
	); err != nil {
		return fmt.Errorf("indexedmap: native 0x24B4D terrain: %w", err)
	}
	copy(work, frame)
	return nil
}

// RedrawNativeUnitPresentObjects is the isolated 0x127a9-equivalent callback
// required after the LMI intro writes and between 0x22046's radial passes.
// It commits atomically and never redraws terrain/range/HUD.
func RedrawNativeUnitPresentObjects(work []byte, in NativeTransitionFrameInput) error {
	if len(work) != NativeUnitPresentWorkSize {
		return errors.New("indexedmap: native unit-present work must be exactly 0x25680 bytes")
	}
	if in.UnitBank == nil || in.ForegroundBank == nil || in.SelectorCache == nil ||
		in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 || len(in.TerrainLUT) != 256 {
		return errors.New("indexedmap: incomplete native unit-present object input")
	}
	frame := append([]byte(nil), work...)
	if err := in.UnitBank.BlitNativeUnitLayer(
		frame, workStride, in.SelectorCache, in.Units,
		in.CameraX, in.CameraY, 12, 7,
		in.IdleCycle, in.MovingCycle, in.PixelShift,
	); err != nil {
		return fmt.Errorf("indexedmap: unit-present units: %w", err)
	}
	if err := in.ForegroundBank.BlitNativeForegroundLayer(
		frame, workStride, in.ForegroundUnits,
		in.MapWidth, in.Cells, in.Controls,
		in.CameraX, in.CameraY, 12, 7,
		in.Flip, in.TerrainLUT,
	); err != nil {
		return fmt.Errorf("indexedmap: unit-present foreground: %w", err)
	}
	copy(work, frame)
	return nil
}

// CopyNativeUnitPresentViewport reproduces the 0x11eb0 calls in
// 0x22470/0x22547/0x22656: source work+0x8088, source stride456,
// width312, height192, destination stride320.
func CopyNativeUnitPresentViewport(vga, work []byte) error {
	if len(work) != NativeUnitPresentWorkSize || len(vga) < viewWidth*viewHeight {
		return errors.New("indexedmap: incomplete native unit-present viewport buffers")
	}
	return fdother.CopyNativeTransitionViewport(
		vga, viewWidth, work,
		workBase, workStride, 312, viewHeight,
	)
}

// ComposeNativeUnitPresentIntroFrame executes one 0x22470 frame atomically:
// restore terrain snapshot, blit one FDOTHER#6 LMI entry at the recovered map
// coordinate, redraw objects, then copy the 312×192 viewport.
func ComposeNativeUnitPresentIntroFrame(
	work, vga, snapshot []byte,
	in NativeTransitionFrameInput,
	entry fdother.LMI1Entry,
	mapX, mapY int,
) error {
	if len(work) != NativeUnitPresentWorkSize ||
		len(snapshot) != NativeUnitPresentWorkSize ||
		len(vga) < viewWidth*viewHeight {
		return errors.New("indexedmap: incomplete native unit-present intro buffers")
	}
	frame, viewport := append([]byte(nil), snapshot...), append([]byte(nil), vga...)
	if err := fdother.BlitNativeUnitPresentLMI(
		entry, frame, mapX, mapY, in.CameraX, in.CameraY,
	); err != nil {
		return fmt.Errorf("indexedmap: unit-present intro LMI: %w", err)
	}
	if err := RedrawNativeUnitPresentObjects(frame, in); err != nil {
		return err
	}
	if err := CopyNativeUnitPresentViewport(viewport, frame); err != nil {
		return err
	}
	copy(work, frame)
	copy(vga, viewport)
	return nil
}

// ComposeNativeUnitPresentLUTSnapshot reproduces the phase boundary at the
// start of 0x22547. Native keeps the original 0x22253 allocation, restores its
// terrain-only contents, and blits the final intro LMI entry #0x7c into that
// allocation once. The resulting terrain+LMI snapshot is then shared by all
// six contract frames and all ten release frames; object layers are redrawn
// only after each frame restores this snapshot.
func ComposeNativeUnitPresentLUTSnapshot(
	dst, terrainSnapshot []byte,
	in NativeTransitionFrameInput,
	finalIntro fdother.LMI1Entry,
	mapX, mapY int,
) error {
	if len(dst) != NativeUnitPresentWorkSize || len(terrainSnapshot) != NativeUnitPresentWorkSize {
		return errors.New("indexedmap: incomplete native unit-present LUT snapshot buffers")
	}
	frame := append([]byte(nil), terrainSnapshot...)
	if err := fdother.BlitNativeUnitPresentLMI(
		finalIntro, frame, mapX, mapY, in.CameraX, in.CameraY,
	); err != nil {
		return fmt.Errorf("indexedmap: unit-present LUT snapshot LMI: %w", err)
	}
	copy(dst, frame)
	return nil
}

// ComposeNativeUnitPresentLUTFrame binds fdother's exact snapshot/remap
// transaction to indexedmap's native object redraw and viewport copy.
// Contract and release both consume the one snapshot built at 0x22547.
func ComposeNativeUnitPresentLUTFrame(
	work, vga, snapshot, lut []byte,
	in NativeTransitionFrameInput,
	frame fdother.UnitPresentLUTFrame,
) error {
	if len(work) != NativeUnitPresentWorkSize ||
		len(snapshot) != NativeUnitPresentWorkSize ||
		len(vga) < viewWidth*viewHeight {
		return errors.New("indexedmap: incomplete native unit-present LUT buffers")
	}
	nextWork, nextVGA := append([]byte(nil), work...), append([]byte(nil), vga...)
	if err := fdother.RunNativeUnitPresentLUTFrame(
		nextWork, snapshot, lut, frame,
		func(full []byte) error { return RedrawNativeUnitPresentObjects(full, in) },
		func(full []byte, _ fdother.UnitPresentLUTFrame) error {
			return CopyNativeUnitPresentViewport(nextVGA, full)
		},
	); err != nil {
		return fmt.Errorf("indexedmap: unit-present LUT frame: %w", err)
	}
	copy(work, nextWork)
	copy(vga, nextVGA)
	return nil
}

// ComposeNativeUnitPresentStripBridge binds 0x22253's transition between the
// six contract and ten release frames. Native restores the shared
// terrain+final-LMI snapshot, applies one extra 0x22046 pass with the
// pointer+1 LUT returned by 0x22547, redraws objects between radial passes,
// then progressively copies the target strip directly to VGA. There is no
// full-viewport 0x11eb0 present in this transaction.
func ComposeNativeUnitPresentStripBridge(
	work, vga, snapshot, bridgeLUT []byte,
	in NativeTransitionFrameInput,
	mapX, mapY, raw53AB9, raw53ABD int,
	delay10ms func(row int) error,
) error {
	if len(work) != NativeUnitPresentWorkSize ||
		len(snapshot) != NativeUnitPresentWorkSize ||
		len(vga) < viewWidth*viewHeight ||
		len(bridgeLUT) != 256 ||
		delay10ms == nil {
		return errors.New("indexedmap: incomplete native unit-present strip composition")
	}
	pass, err := fdother.NativeUnitPresentLUTPass(24*raw53AB9+12, 24*raw53ABD+15, 0)
	if err != nil {
		return fmt.Errorf("indexedmap: unit-present strip geometry: %w", err)
	}
	frame := append([]byte(nil), snapshot...)
	if err := fdother.ApplyIndexedTransitionPass(
		frame[workBase:], workStride, bridgeLUT, pass,
		func([]byte) error { return RedrawNativeUnitPresentObjects(frame, in) },
	); err != nil {
		return fmt.Errorf("indexedmap: unit-present strip LUT: %w", err)
	}
	// Match native observability: work has completed the remap before the first
	// progressive VGA row becomes visible.
	copy(work, frame)
	return RunNativeUnitPresentStripBridge(
		work, vga, mapX, mapY, in.CameraX, in.CameraY, delay10ms,
	)
}

// BuildNativeTerrainCells materializes the exact per-cell pair exported by
// tools/export_engine_assets.py. The tile word and event high byte are kept
// separate because native 0x11eee consumes both; missing or mismatched arrays
// are rejected instead of defaulting old PNG-only maps into a native path.
func BuildNativeTerrainCells(tiles []int, blitModes []byte) ([]fdicon.NativeTerrainCell, error) {
	if len(tiles) == 0 || len(tiles) != len(blitModes) {
		return nil, errors.New("indexedmap: native terrain cell arrays are incomplete")
	}
	cells := make([]fdicon.NativeTerrainCell, len(tiles))
	for i, tile := range tiles {
		if tile < 0 || tile > 0x3ff {
			return nil, fmt.Errorf("indexedmap: native terrain tile %d=%d outside 10-bit range", i, tile)
		}
		cells[i] = fdicon.NativeTerrainCell{Tile: uint16(tile), BlitMode: blitModes[i]}
	}
	return cells, nil
}

// ComposeNativeFrame is the strict native-HUD form of ComposeFrame. It uses
// the exact indexed HUD assembly at its recovered position, rather than
// accepting an arbitrary callback. All source data remain explicit and any
// rejection keeps work/VGA unchanged through ComposeFrame's transaction.
func ComposeNativeFrame(work, vga []byte, in NativeFrameInput) error {
	return ComposeFrame(work, vga, in.Frame, func(dst []byte) error {
		// 0x11cfa passes [0x53a49]+0x8088 to 0x1acf3, not the allocation
		// base. HUD row/column offsets are therefore viewport-relative, just
		// like the preceding 0x11eee terrain destination.
		if len(dst) < workBase {
			return errors.New("indexedmap: native HUD viewport base outside work buffer")
		}
		return BlitNativeMapHUD(in.Frames, in.HUDTerrain, in.HUDUnits, in.HUDCache, dst[workBase:], in.HUD)
	})
}

// ComposeNativeTransitionFrame performs one verified indexed 0x24618 frame:
// terrain redraw → first LUT pass → 0x127a9 unit/foreground redraw → second
// LUT pass → centered rectangle LUT → 312×192 viewport copy. It clones the
// work buffer before every operation, so missing raw banks or malformed
// coordinates cannot partially mutate either caller buffer.
func ComposeNativeTransitionFrame(work, vga []byte, in NativeTransitionFrameInput, pass fdother.IndexedTransitionPass, lut []byte) error {
	if len(work) < (fdother.NativeTransitionStageOffset+fdother.NativeTransitionStageHeight*fdother.NativeTransitionStageStride) || len(vga) < fdother.NativeTransitionPresentStride*fdother.NativeTransitionStageHeight {
		return errors.New("indexedmap: incomplete native transition buffers")
	}
	if in.TerrainBank == nil || in.UnitBank == nil || in.ForegroundBank == nil || in.SelectorCache == nil || in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 || len(in.TerrainLUT) != 256 || len(lut) != 256 {
		return errors.New("indexedmap: incomplete native transition input")
	}
	frame := append([]byte(nil), work...)
	baseX, baseY := workBase%workStride, workBase/workStride
	if err := in.TerrainBank.BlitNativeTerrainRegion(frame, workStride, baseX, baseY, in.MapWidth, in.Cells, in.Controls, in.CameraX, in.CameraY, 13, 8, in.Flip, in.TerrainCycle, in.TerrainLUT); err != nil {
		return fmt.Errorf("indexedmap: transition terrain: %w", err)
	}
	redraw := func(dst []byte) error {
		if err := in.UnitBank.BlitNativeUnitLayer(dst, workStride, in.SelectorCache, in.Units, in.CameraX, in.CameraY, 12, 7, in.IdleCycle, in.MovingCycle, in.PixelShift); err != nil {
			return fmt.Errorf("indexedmap: transition units: %w", err)
		}
		if err := in.ForegroundBank.BlitNativeForegroundLayer(dst, workStride, in.ForegroundUnits, in.MapWidth, in.Cells, in.Controls, in.CameraX, in.CameraY, 12, 7, in.Flip, in.TerrainLUT); err != nil {
			return fmt.Errorf("indexedmap: transition foreground: %w", err)
		}
		return nil
	}
	if err := fdother.ApplyIndexedTransitionPass(frame, workStride, lut, pass, redraw); err != nil {
		return fmt.Errorf("indexedmap: transition pass: %w", err)
	}
	viewport := make([]byte, fdother.NativeTransitionPresentStride*fdother.NativeTransitionStageHeight)
	if err := fdother.CopyNativeTransitionViewport(viewport, fdother.NativeTransitionPresentStride, frame, fdother.NativeTransitionStageOffset, fdother.NativeTransitionStageStride, fdother.NativeTransitionStageWidth, fdother.NativeTransitionStageHeight); err != nil {
		return fmt.Errorf("indexedmap: transition viewport: %w", err)
	}
	copy(work, frame)
	copy(vga, viewport)
	return nil
}

// ComposeFrame performs the recovered steady order:
//
//	0x11eee terrain → 0x122dc range → 0x127a9 unit/foreground
//	→ required 0x1acf3-equivalent HUD callback → 0x11eb0 viewport copy.
//
// HUD is mandatory because copying before it would silently alter native draw
// order. All work happens on a private clone first, so rejected editable input
// or a HUD error never leaves either caller buffer partially changed.
func ComposeFrame(work, vga []byte, in FrameInput, renderHUD func([]byte) error) error {
	if renderHUD == nil || len(work)%workStride != 0 || len(vga) < NativeMapVGASize || in.MapWidth <= 0 || len(in.Cells)%in.MapWidth != 0 {
		return errors.New("indexedmap: incomplete native frame input")
	}
	if in.TerrainBank == nil || in.RangeBank == nil || in.UnitBank == nil || in.ForegroundBank == nil || in.SelectorCache == nil {
		return errors.New("indexedmap: missing native frame bank")
	}
	frame := append([]byte(nil), work...)
	cells := append([]fdicon.NativeTerrainCell(nil), in.Cells...)
	baseX, baseY := workBase%workStride, workBase/workStride
	if err := in.TerrainBank.BlitNativeTerrainRegion(frame, workStride, baseX, baseY, in.MapWidth, cells, in.Controls, in.CameraX, in.CameraY, 13, 8, in.Flip, in.TerrainCycle, in.LUT); err != nil {
		return fmt.Errorf("indexedmap: terrain: %w", err)
	}
	if in.RangeMode == 6 {
		if in.CursorX < 0 || in.CursorY < 0 || in.CursorX >= in.MapWidth ||
			in.CursorY >= len(cells)/in.MapWidth {
			return errors.New("indexedmap: mode6 cursor outside field")
		}
		// 0x122dc mutates composition byte+3 after 0x11eee terrain and before
		// 0x127a9 foreground. Commit to caller state only after the full frame
		// succeeds, retaining the adapter's failure-atomic boundary.
		cells[in.CursorY*in.MapWidth+in.CursorX].BlitMode = 0
	} else if err := fdother.BlitNativeRangeOverlay(in.RangeBank, frame, in.CameraX, in.CameraY, 13, 8, in.RangeMode, in.CursorX, in.CursorY); err != nil {
		return fmt.Errorf("indexedmap: range: %w", err)
	}
	if err := in.UnitBank.BlitNativeUnitLayer(frame, workStride, in.SelectorCache, in.Units, in.CameraX, in.CameraY, 12, 7, in.IdleCycle, in.MovingCycle, in.PixelShift); err != nil {
		return fmt.Errorf("indexedmap: units: %w", err)
	}
	if err := in.ForegroundBank.BlitNativeForegroundLayer(frame, workStride, in.ForegroundUnits, in.MapWidth, cells, in.Controls, in.CameraX, in.CameraY, 12, 7, in.Flip, in.LUT); err != nil {
		return fmt.Errorf("indexedmap: foreground: %w", err)
	}
	if err := renderHUD(frame); err != nil {
		return fmt.Errorf("indexedmap: HUD: %w", err)
	}
	copyFrame := append([]byte(nil), vga...)
	if err := fdicon.CopyNativeIndexedRegion(
		copyFrame[steadyViewportOffset:], viewWidth,
		frame[workBase:], workStride,
		steadyViewportWidth, viewHeight,
	); err != nil {
		return fmt.Errorf("indexedmap: viewport copy: %w", err)
	}
	copy(work, frame)
	copy(vga, copyFrame)
	copy(in.Cells, cells)
	return nil
}
