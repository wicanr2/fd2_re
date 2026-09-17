package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativePaletteRampJob owns the exact whole-DAC 0x11d40 writes used by
// 0x1f882 and 0x1f525. Each step must be presented before it advances; this
// intentionally stretches sub-frame DOS waits on a 60Hz host rather than
// collapsing the complete effect into one unobservable update.
type nativePaletteRampJob struct {
	deltas   []int
	delayMs  int
	step     int
	vga      []byte
	dac      []byte
	baseline []byte
	palette  color.Palette
	drawn    bool
	then     func()
}

func buildInclusivePaletteDeltas(start, end int) ([]int, error) {
	if start < 0 || start > 255 || end < 0 || end > 255 {
		return nil, errors.New("palette delta endpoint outside byte range")
	}
	step := 1
	if start > end {
		step = -1
	}
	deltas := make([]int, 0, absInt(start-end)+1)
	for value := start; ; value += step {
		deltas = append(deltas, value)
		if value == end {
			break
		}
	}
	return deltas, nil
}

func (g *Game) startNativePaletteRamp(start, end, delayMs int, then func()) error {
	if g.nativePaletteRamp != nil {
		return errors.New("native palette ramp already active")
	}
	if delayMs != 2 || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native indexed framebuffer or DAC baseline unavailable")
	}
	if err := g.ensureNativePaletteFrame(); err != nil {
		return err
	}
	deltas, err := buildInclusivePaletteDeltas(start, end)
	if err != nil {
		return err
	}
	baseline := append([]byte(nil), g.nativeMapAssets.PaletteDAC...)
	dac := append([]byte(nil), baseline...)
	if len(g.nativeMapDAC) == len(baseline) {
		copy(dac, g.nativeMapDAC)
	}
	job := &nativePaletteRampJob{
		deltas: deltas, delayMs: delayMs,
		vga: append([]byte(nil), g.nativeMapVGA...),
		dac: dac, baseline: baseline, then: then,
	}
	if err := job.applyCurrent(); err != nil {
		return err
	}
	g.nativePaletteRamp = job
	return nil
}

// ensureNativePaletteFrame uses the already published steady battle frame
// when available. A handler cutscene has no battle State/HUD; its acting
// redraw path is the separately recovered terrain → unit/foreground →
// 0x11eb0 compositor, represented by the three strict indexedmap operations
// below. No normalized PNG pixels are converted into fabricated palette
// indices.
func (g *Game) ensureNativePaletteFrame() error {
	if len(g.nativeMapVGA) == indexedmap.NativeMapVGASize {
		return nil
	}
	if len(g.storyActors) == 0 {
		return errors.New("native indexed framebuffer unavailable")
	}
	in, err := g.buildNativeIndexedTransitionInput()
	if err != nil {
		return fmt.Errorf("native handler frame: %w", err)
	}
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(work, in); err != nil {
		return fmt.Errorf("native handler terrain: %w", err)
	}
	if err := indexedmap.RedrawNativeUnitPresentObjects(work, in); err != nil {
		return fmt.Errorf("native handler objects: %w", err)
	}
	if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
		return fmt.Errorf("native handler viewport: %w", err)
	}
	g.nativeMapWork = append(g.nativeMapWork[:0], work...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], vga...)
	return nil
}

func (j *nativePaletteRampJob) applyCurrent() error {
	if j == nil || j.step < 0 || j.step >= len(j.deltas) || j.delayMs != 2 {
		return errors.New("native palette ramp step unavailable")
	}
	if err := fdother.ApplyVGAPaletteSubtraction(j.dac, j.baseline, 0, 255, j.deltas[j.step]); err != nil {
		return err
	}
	palette, err := fdother.VGAPaletteFromDAC(j.dac)
	if err != nil {
		return err
	}
	j.palette, j.drawn = palette, false
	return nil
}

func (g *Game) stepNativePaletteRamp() {
	j := g.nativePaletteRamp
	if j == nil || !j.drawn {
		return
	}
	j.step++
	if j.step >= len(j.deltas) {
		then := j.then
		g.nativeMapDAC = append(g.nativeMapDAC[:0], j.dac...)
		g.nativePaletteRamp = nil
		if then != nil {
			then()
		}
		return
	}
	if err := j.applyCurrent(); err != nil {
		g.loadErr = "native palette ramp: " + err.Error()
		g.nativePaletteRamp = nil
	}
}

func (g *Game) drawNativePaletteRamp(screen *ebiten.Image) bool {
	j := g.nativePaletteRamp
	if j == nil || len(j.vga) != indexedmap.NativeMapVGASize || len(j.palette) != 256 {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palette)
	copy(img.Pix, j.vga)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
	return true
}

func nativeDACIsBlack(dac []byte) bool {
	if len(dac) != 256*3 {
		return false
	}
	for _, component := range dac {
		if component != 0 {
			return false
		}
	}
	return true
}

func (g *Game) applyHandlerDirectRecordPatch(patch *campaign.HandlerDirectRecordPatch) error {
	// 戰場上寫 g.st；戰前處理器（例如 0x3332B 在 LOADCH 之後）寫同一份 runtime slot 的
	// 劇情演員陣列，戰場接手時由 AdoptHandlerBattleState 原樣帶進去。
	if patch == nil || g.handlerUnitCount() == 0 || len(patch.Units) == 0 ||
		(patch.View != nil && g.st == nil) {
		return errors.New("runtime battle records unavailable")
	}
	seen := make(map[int]bool, len(patch.Units))
	for _, entry := range patch.Units {
		if entry.Slot < 0 || entry.Slot >= g.handlerUnitCount() || seen[entry.Slot] {
			return fmt.Errorf("slot%d unavailable or duplicated", entry.Slot)
		}
		if (entry.X == nil) != (entry.Y == nil) ||
			(entry.X == nil && entry.Pose == nil && len(entry.RawBytes) == 0) {
			return fmt.Errorf("slot%d has an incomplete or empty sparse write", entry.Slot)
		}
		if entry.X != nil && (*entry.X < 0 || *entry.X > 0xff || *entry.Y < 0 || *entry.Y > 0xff) {
			return fmt.Errorf("slot%d coordinate outside byte range", entry.Slot)
		}
		if entry.Pose != nil && (*entry.Pose < 0 || *entry.Pose > 3) {
			return fmt.Errorf("slot%d pose outside native range", entry.Slot)
		}
		seen[entry.Slot] = true
		unit := g.handlerUnitAt(entry.Slot)
		if unit == nil || ((entry.X != nil || entry.Pose != nil) && !unit.HasNativeMapPresentation) {
			return fmt.Errorf("slot%d lacks native map-record provenance", entry.Slot)
		}
		seenOffsets := make(map[int]bool, len(entry.RawBytes))
		for _, raw := range entry.RawBytes {
			if raw.Value < 0 || raw.Value > 0xff || seenOffsets[raw.Offset] {
				return fmt.Errorf("slot%d raw offset %#x has invalid or duplicate byte value", entry.Slot, raw.Offset)
			}
			seenOffsets[raw.Offset] = true
			if raw.Offset == 5 && !unit.HasNativeRecordByte5 {
				return fmt.Errorf("slot%d lacks raw byte+5 provenance", entry.Slot)
			}
			if raw.Offset != 5 && (raw.Offset < 0x22 || raw.Offset > 0x27) {
				return fmt.Errorf("slot%d raw offset %#x unsupported", entry.Slot, raw.Offset)
			}
		}
	}
	if patch.View != nil {
		candidate := *g.st
		if err := candidate.MaterializeNativeMapViewState(battleNativeMapView(*patch.View)); err != nil {
			return fmt.Errorf("native view: %w", err)
		}
	}
	for _, entry := range patch.Units {
		unit := g.handlerUnitAt(entry.Slot)
		if entry.X != nil && !unit.SetNativeMapCoordinatesRaw(*entry.X, *entry.Y) {
			return fmt.Errorf("slot%d coordinate write failed", entry.Slot)
		}
		if entry.Pose != nil && !unit.SetMapPose(*entry.Pose) {
			return fmt.Errorf("slot%d pose write failed", entry.Slot)
		}
		for _, raw := range entry.RawBytes {
			switch raw.Offset {
			case 5:
				unit.NativeRecordByte5 = byte(raw.Value)
			default:
				unit.NativeTransient[raw.Offset-0x22] = byte(raw.Value)
			}
		}
	}
	if patch.View != nil {
		if err := g.st.MaterializeNativeMapViewState(battleNativeMapView(*patch.View)); err != nil {
			return fmt.Errorf("native view commit: %w", err)
		}
		g.syncNativeMapView()
	}
	return nil
}

func battleNativeMapView(view campaign.NativeMapViewConfig) battle.NativeMapViewState {
	return battle.NativeMapViewState{
		CameraX: view.CameraX, CameraY: view.CameraY,
		CursorX: view.CursorX, CursorY: view.CursorY,
		VisibleCursorX: view.VisibleCursorX, VisibleCursorY: view.VisibleCursorY,
	}
}
