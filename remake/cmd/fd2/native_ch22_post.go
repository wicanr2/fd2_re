package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// native2189AJob is the exact ten-pass indexed presentation owned by 0x2189A.
// The address-shaped name is deliberate: evidence proves its buffer schedule,
// not a universal gameplay or narrative meaning.
type native2189AJob struct {
	workFrames [][]byte
	vgaFrames  [][]byte
	palette    color.Palette
	index      int
	drawn      bool
	then       func()
	rollback   func()
}

func nativeTransitionInput(in indexedmap.NativeFrameInput) indexedmap.NativeTransitionFrameInput {
	f := in.Frame
	return indexedmap.NativeTransitionFrameInput{
		TerrainBank: f.TerrainBank, UnitBank: f.UnitBank, ForegroundBank: f.ForegroundBank,
		SelectorCache: f.SelectorCache, Cells: f.Cells, Controls: f.Controls, TerrainLUT: f.LUT,
		MapWidth: f.MapWidth, CameraX: f.CameraX, CameraY: f.CameraY,
		Flip: f.Flip, TerrainCycle: f.TerrainCycle, IdleCycle: f.IdleCycle,
		MovingCycle: f.MovingCycle, PixelShift: f.PixelShift,
		Units: f.Units, ForegroundUnits: f.ForegroundUnits,
	}
}

func (g *Game) startNative2189A(loop campaign.Native2189ALoop, then func()) error {
	if g == nil || g.native2189A != nil || g.nativeCh20SkyKey != nil ||
		g.nativeCh23Loop != nil || g.indexedTransition != nil {
		return errors.New("native 0x2189A presentation is already active")
	}
	slot, initialRadius, radiusStep := loop.Slot, loop.InitialRadius, loop.RadiusStep
	if !((slot == 10 && initialRadius == 15 && radiusStep == 1) ||
		(slot == 16 && initialRadius == 30 && radiusStep == 1)) || loop.Repeat != 10 ||
		loop.WorkOffset != 0x8088 || loop.WorkStride != 456 ||
		loop.ClipWidth != 312 || loop.ClipHeight != 192 {
		return errors.New("native 0x2189A call differs from recovered ch22 contract")
	}
	if g.st == nil || g.m == nil || !nativeMapAssetsAvailable(g.nativeMapAssets) ||
		!g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return errors.New("native 0x2189A indexed map state is unavailable")
	}
	unit := g.handlerUnitAt(slot)
	if unit == nil || !unit.HasNativeMapPresentation {
		return fmt.Errorf("native 0x2189A slot %d lacks raw +0/+1 presentation coordinates", slot)
	}
	for i := 0; i < 10; i++ {
		if i >= len(g.nativeMapAssets.LUTs) || len(g.nativeMapAssets.LUTs[i]) != 256 {
			return fmt.Errorf("native 0x2189A FDOTHER#3 LUT %d is unavailable", i)
		}
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native 0x2189A HUD provenance is unavailable")
	}
	in, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, g.st, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	transition := nativeTransitionInput(in)
	snapshot := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	copy(snapshot, g.nativeMapWork)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(snapshot, transition); err != nil {
		return err
	}
	centerX := (int(unit.NativeMapPresentation.X)-transition.CameraX)*24 + 12
	centerY := (int(unit.NativeMapPresentation.Y)-transition.CameraY)*24 + 18
	palette := append(color.Palette(nil), g.nativeMapAssets.Palette...)
	if len(g.nativeMapDAC) == 256*3 {
		if current, paletteErr := fdother.VGAPaletteFromDAC(g.nativeMapDAC); paletteErr == nil {
			palette = current
		}
	}
	job := &native2189AJob{palette: palette, then: then}
	for i := 0; i < 10; i++ {
		work := append([]byte(nil), snapshot...)
		vga := append([]byte(nil), g.nativeMapVGA...)
		radius := initialRadius + i*radiusStep
		if err := fdother.ApplyRadialLUTRemap(work[loop.WorkOffset:], loop.WorkStride, g.nativeMapAssets.LUTs[i], fdother.RadialLUTRemap{
			CenterX: centerX, CenterY: centerY, Radius: radius, Scale: 12,
			StartY: 0, EndY: loop.ClipHeight, ClipWidth: loop.ClipWidth,
		}); err != nil {
			return fmt.Errorf("native 0x2189A pass %d: %w", i, err)
		}
		if err := indexedmap.RedrawNativeUnitPresentObjects(work, transition); err != nil {
			return fmt.Errorf("native 0x2189A pass %d object redraw: %w", i, err)
		}
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
			return fmt.Errorf("native 0x2189A pass %d viewport: %w", i, err)
		}
		job.workFrames = append(job.workFrames, work)
		job.vgaFrames = append(job.vgaFrames, vga)
	}
	// 0x219A3 invokes the ordinary 0x11CAC(0) owner after all ten LUT passes.
	// Build that complete steady frame before publishing the first effect frame,
	// so a tail failure cannot leave a partially completed beat.
	steadyWork := append([]byte(nil), g.nativeMapWork...)
	steadyVGA := append([]byte(nil), g.nativeMapVGA...)
	if err := indexedmap.ComposeNativeFrame(steadyWork, steadyVGA, in); err != nil {
		return fmt.Errorf("native 0x2189A steady restore: %w", err)
	}
	job.workFrames = append(job.workFrames, steadyWork)
	job.vgaFrames = append(job.vgaFrames, steadyVGA)
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	job.rollback = func() {
		g.nativeMapWork = append(g.nativeMapWork[:0], beforeWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], beforeVGA...)
	}
	g.native2189A = job
	g.publishNative2189AFrame()
	return nil
}

func (g *Game) publishNative2189AFrame() {
	j := g.native2189A
	if j == nil || j.index < 0 || j.index >= len(j.vgaFrames) {
		return
	}
	g.nativeMapWork = append(g.nativeMapWork[:0], j.workFrames[j.index]...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.vgaFrames[j.index]...)
	j.drawn = false
}

func (g *Game) stepNative2189A() {
	j := g.native2189A
	if j == nil || !j.drawn {
		return
	}
	j.index++
	if j.index < len(j.vgaFrames) {
		g.publishNative2189AFrame()
		return
	}
	then := j.then
	j.rollback = nil
	g.native2189A = nil
	if then != nil {
		then()
	}
}

func (g *Game) failNative2189A(err error) {
	if j := g.native2189A; j != nil && j.rollback != nil {
		j.rollback()
	}
	g.native2189A = nil
	g.loadErr = "native 0x2189A: " + err.Error()
}

func (g *Game) drawNative2189A(screen *ebiten.Image) bool {
	j := g.native2189A
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
