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

// nativeUnitPresentJob owns one complete 0x22253 transaction. Frames are
// precomputed before publication; mutationAt is the first bridge row, after
// all 11 intro and six contract full presents have actually been drawn.
type nativeUnitPresentJob struct {
	workFrames [][]byte
	vgaFrames  [][]byte
	waits      []int
	palette    color.Palette
	frame      int
	wait       int
	drawn      bool

	mutationAt int
	unitSlot   int
	newX       int
	newY       int
	mutated    bool
	then       func()
	rollback   func()
}

func nativeUnitPresentAssetsAvailable(a *nativeMapAssets) bool {
	if !nativeMapAssetsAvailable(a) || len(a.FDOTHER6) <= 0x7c {
		return false
	}
	for index := 0x72; index <= 0x7c; index++ {
		entry := a.FDOTHER6[index]
		if entry.Width <= 0 || entry.Height <= 0 || len(entry.Pixels) != entry.Width*entry.Height {
			return false
		}
	}
	return true
}

func resolveNativeUnitPresentCall(spec campaign.NativeUnitPresent, slotCount int) (fdother.NativeUnitPresentCall, error) {
	if slotCount <= 0 || (spec.LastRuntimeSlot && spec.Slot != 0) {
		return fdother.NativeUnitPresentCall{}, errors.New("native 0x22253 slot selector is invalid")
	}
	slot := spec.Slot
	if spec.LastRuntimeSlot {
		slot = slotCount - 1
	}
	return fdother.PlanNativeUnitPresentCall(slot, spec.NewX, spec.NewY, spec.VisualX, spec.VisualY)
}

func cloneNativeUnitPresentState(source *battle.State) (*battle.State, error) {
	if source == nil || source.NativeMapSelectorCache == nil || source.NativeMapSelectorError != nil {
		return nil, errors.New("native 0x22253 selector state is unavailable")
	}
	candidate := *source
	candidate.Units = cloneBattleUnitPointers(source.Units)
	candidate.Roster = cloneBattleUnitPointers(source.Roster)
	candidate.NativeMapSelectorCache = source.NativeMapSelectorCache.Clone()
	candidate.NativeCompositionEventBytes = append([]byte(nil), source.NativeCompositionEventBytes...)
	return &candidate, nil
}

func (g *Game) startNativeUnitPresent(spec campaign.NativeUnitPresent, then func()) error {
	if g == nil || g.nativeUnitPresent != nil || g.native2189A != nil ||
		g.transitionReveal != nil || g.indexedTransition != nil || g.nativePaletteRamp != nil ||
		g.nativePalettePulse != nil ||
		g.nativeCh20SkyKey != nil || g.nativeCh23Loop != nil {
		return errors.New("native 0x22253 presentation is already active")
	}
	job, err := g.buildNativeUnitPresentJob(spec, g.st, g.nativeMapWork, g.nativeMapVGA, then)
	if err != nil {
		return err
	}
	call, err := resolveNativeUnitPresentCall(spec, len(g.st.Units))
	if err != nil {
		return err
	}
	beforeWork := append([]byte(nil), g.nativeMapWork...)
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	beforeUnit := *g.st.Units[call.UnitSlot]
	job.rollback = func() {
		g.nativeMapWork = append(g.nativeMapWork[:0], beforeWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], beforeVGA...)
		if g.st != nil && call.UnitSlot < len(g.st.Units) && g.st.Units[call.UnitSlot] != nil {
			*g.st.Units[call.UnitSlot] = beforeUnit
		}
	}
	g.nativeUnitPresent = job
	g.publishNativeUnitPresentFrame()
	return nil
}

// buildNativeUnitPresentJob precomputes one complete 0x22253 call without
// publishing frames or mutating live state. A caller which owns a multi-call
// sequence can therefore validate every leg before the first visible output.
func (g *Game) buildNativeUnitPresentJob(
	spec campaign.NativeUnitPresent,
	source *battle.State,
	initialWork, initialVGA []byte,
	then func(),
) (*nativeUnitPresentJob, error) {
	if g == nil || source == nil || g.m == nil || !nativeUnitPresentAssetsAvailable(g.nativeMapAssets) ||
		!source.HasNativeMapViewState || len(initialWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(initialVGA) != indexedmap.NativeMapVGASize {
		return nil, errors.New("native 0x22253 indexed map state is unavailable")
	}
	call, err := resolveNativeUnitPresentCall(spec, len(source.Units))
	if err != nil {
		return nil, err
	}
	if call.UnitSlot < 0 || call.UnitSlot >= len(source.Units) || source.Units[call.UnitSlot] == nil ||
		!source.Units[call.UnitSlot].HasNativeMapPresentation {
		return nil, fmt.Errorf("native 0x22253 slot %d lacks raw map presentation", call.UnitSlot)
	}
	if err := fdother.ValidateNativeUnitPresentSchedule(fdother.NativeUnitPresentSchedule()); err != nil {
		return nil, err
	}
	bridgeLUT, err := fdother.NativeUnitPresentBridgeLUT(g.nativeMapAssets.LUTs)
	if err != nil {
		return nil, err
	}
	beforeInput, err := g.buildNativeIndexedTransitionInputForState(source)
	if err != nil {
		return nil, fmt.Errorf("native 0x22253 pre-mutation input: %w", err)
	}
	candidate, err := cloneNativeUnitPresentState(source)
	if err != nil {
		return nil, err
	}
	if !candidate.Units[call.UnitSlot].SetNativeMapCoordinatesRaw(int(call.NewX), int(call.NewY)) {
		return nil, errors.New("native 0x22253 candidate coordinate write failed")
	}
	afterInput, err := g.buildNativeIndexedTransitionInputForState(candidate)
	if err != nil {
		return nil, fmt.Errorf("native 0x22253 post-mutation input: %w", err)
	}

	terrainSnapshot := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(terrainSnapshot, beforeInput); err != nil {
		return nil, err
	}
	job := &nativeUnitPresentJob{
		palette:  append(color.Palette(nil), g.nativeMapAssets.Palette...),
		unitSlot: call.UnitSlot, newX: int(call.NewX), newY: int(call.NewY), then: then,
	}
	if len(g.nativeMapDAC) == 256*3 {
		if current, paletteErr := fdother.VGAPaletteFromDAC(g.nativeMapDAC); paletteErr == nil {
			job.palette = current
		}
	}
	appendFrame := func(work, vga []byte, wait int) {
		job.workFrames = append(job.workFrames, append([]byte(nil), work...))
		job.vgaFrames = append(job.vgaFrames, append([]byte(nil), vga...))
		if wait < 1 {
			wait = 1
		}
		job.waits = append(job.waits, wait)
	}

	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	vga := append([]byte(nil), initialVGA...)
	for index := 0x72; index <= 0x7c; index++ {
		if err := indexedmap.ComposeNativeUnitPresentIntroFrame(
			work, vga, terrainSnapshot, beforeInput, g.nativeMapAssets.FDOTHER6[index],
			int(call.VisualX), int(call.VisualY),
		); err != nil {
			return nil, fmt.Errorf("native 0x22253 intro %#x: %w", index, err)
		}
		appendFrame(work, vga, 3) // one native BIOS tick, approximated on 60 Hz host
	}
	lutSnapshot := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.ComposeNativeUnitPresentLUTSnapshot(
		lutSnapshot, terrainSnapshot, beforeInput, g.nativeMapAssets.FDOTHER6[0x7c],
		int(call.VisualX), int(call.VisualY),
	); err != nil {
		return nil, err
	}
	view := source.NativeMapViewState
	lutFrames, err := fdother.NativeUnitPresentLUTFrames(view.VisibleCursorX, view.VisibleCursorY)
	if err != nil {
		return nil, err
	}
	for index, frame := range lutFrames[:6] {
		lut := g.nativeMapAssets.LUTs[frame.LUTIndex]
		if err := indexedmap.ComposeNativeUnitPresentLUTFrame(work, vga, lutSnapshot, lut, beforeInput, frame); err != nil {
			return nil, fmt.Errorf("native 0x22253 contract %d: %w", index, err)
		}
		wait := nativeDelayTicks(frame.DelayMs) + frame.DelayTicks*3
		appendFrame(work, vga, wait)
	}
	job.mutationAt = len(job.vgaFrames)
	bridgeWork := append([]byte(nil), work...)
	bridgeVGA := append([]byte(nil), vga...)
	if err := indexedmap.ComposeNativeUnitPresentStripBridge(
		bridgeWork, bridgeVGA, lutSnapshot, bridgeLUT, afterInput,
		int(call.VisualX), int(call.VisualY), view.VisibleCursorX, view.VisibleCursorY,
		func(_ int) error {
			appendFrame(bridgeWork, bridgeVGA, nativeDelayTicks(10))
			return nil
		},
	); err != nil {
		return nil, err
	}
	work, vga = bridgeWork, bridgeVGA
	for index, frame := range lutFrames[6:] {
		lut := g.nativeMapAssets.LUTs[frame.LUTIndex]
		if err := indexedmap.ComposeNativeUnitPresentLUTFrame(work, vga, lutSnapshot, lut, afterInput, frame); err != nil {
			return nil, fmt.Errorf("native 0x22253 release %d: %w", index, err)
		}
		appendFrame(work, vga, frame.DelayTicks*3)
	}
	if len(job.workFrames) != len(job.vgaFrames) || len(job.waits) != len(job.vgaFrames) ||
		job.mutationAt != 17 || len(job.vgaFrames) < 45 {
		return nil, errors.New("native 0x22253 preflight produced an invalid schedule")
	}
	return job, nil
}

func (g *Game) publishNativeUnitPresentFrame() {
	j := g.nativeUnitPresent
	if j == nil || j.frame < 0 || j.frame >= len(j.vgaFrames) {
		return
	}
	g.nativeMapWork = append(g.nativeMapWork[:0], j.workFrames[j.frame]...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.vgaFrames[j.frame]...)
	j.drawn = false
	j.wait = j.waits[j.frame] - 1
	if j.wait < 0 {
		j.wait = 0
	}
}

func (g *Game) stepNativeUnitPresent() {
	j := g.nativeUnitPresent
	if j == nil || !j.drawn {
		return
	}
	if j.wait > 0 {
		j.wait--
		return
	}
	next := j.frame + 1
	if next >= len(j.vgaFrames) {
		then := j.then
		j.rollback = nil
		g.nativeUnitPresent = nil
		if then != nil {
			then()
		}
		return
	}
	if next == j.mutationAt && !j.mutated {
		if g.st == nil || j.unitSlot >= len(g.st.Units) || g.st.Units[j.unitSlot] == nil ||
			!g.st.Units[j.unitSlot].SetNativeMapCoordinatesRaw(j.newX, j.newY) {
			g.failNativeUnitPresent(errors.New("coordinate mutation boundary unavailable"))
			return
		}
		j.mutated = true
	}
	j.frame = next
	g.publishNativeUnitPresentFrame()
}

func (g *Game) failNativeUnitPresent(err error) {
	if j := g.nativeUnitPresent; j != nil && j.rollback != nil {
		j.rollback()
	}
	g.nativeUnitPresent = nil
	g.loadErr = "native 0x22253: " + err.Error()
}

func (g *Game) drawNativeUnitPresent(screen *ebiten.Image) bool {
	j := g.nativeUnitPresent
	if j == nil || len(j.palette) != 256 || len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palette)
	copy(img.Pix, g.nativeMapVGA)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	j.drawn = true
	return true
}
