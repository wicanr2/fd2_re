package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func (g *Game) startNative24B4D(frameCount, delayMs int, then func()) error {
	if g == nil || g.transitionReveal != nil || g.native2189A != nil ||
		g.nativeCh20SkyKey != nil || g.nativeCh23Loop != nil || g.indexedTransition != nil {
		return errors.New("native 0x24B4D presentation is already active")
	}
	if frameCount <= 0 || frameCount > 255 || delayMs != 20 {
		return errors.New("native 0x24B4D frame count or 20ms delay differs from recovered ABI")
	}
	if g.st == nil || g.m == nil || !nativeMapAssetsAvailable(g.nativeMapAssets) ||
		len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return errors.New("native 0x24B4D indexed map state is unavailable")
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		return errors.New("native 0x24B4D HUD provenance is unavailable")
	}
	in, err := buildNativeMapFrameInput(g.nativeMapAssets, g.m, g.st, nativeMapFrameRuntime{HUD: hud})
	if err != nil {
		return err
	}
	work := append([]byte(nil), g.nativeMapWork...)
	if err := indexedmap.ComposeNative24B4DStaging(work, nativeTransitionInput(in)); err != nil {
		return err
	}
	steadyVGA := append([]byte(nil), g.nativeMapVGA...)
	if err := indexedmap.ComposeNativeFrame(work, steadyVGA, in); err != nil {
		return fmt.Errorf("native 0x24B4D steady 0x11CAC: %w", err)
	}
	var frames [2][]byte
	for row := 0; row < 2; row++ {
		frames[row] = append([]byte(nil), steadyVGA...)
		if err := fdother.CopyNativeTransitionViewport(
			frames[row][0x504:], 320, work,
			fdother.NativeTransitionStageOffset+row*fdother.NativeTransitionStageStride,
			fdother.NativeTransitionStageStride,
			fdother.NativeTransitionStageWidth, fdother.NativeTransitionStageHeight,
		); err != nil {
			return fmt.Errorf("native 0x24B4D row %d viewport: %w", row, err)
		}
	}
	palette := append(color.Palette(nil), g.nativeMapAssets.Palette...)
	if len(g.nativeMapDAC) == 256*3 {
		if current, paletteErr := fdother.VGAPaletteFromDAC(g.nativeMapDAC); paletteErr == nil {
			palette = current
		}
	}
	delay := delayMs * 60 / 1000
	if delay < 1 {
		delay = 1
	}
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	job := &transitionRevealJob{
		work: work, frames: frames, palette: palette,
		remaining: frameCount, delay: delay, then: then,
	}
	job.rollback = func() {
		g.nativeMapWork = append(g.nativeMapWork[:0], beforeWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], beforeVGA...)
	}
	g.nativeMapWork = append(g.nativeMapWork[:0], work...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], frames[0]...)
	g.transitionReveal = job
	return nil
}

func (g *Game) failNative24B4D(err error) {
	if job := g.transitionReveal; job != nil && job.rollback != nil {
		job.rollback()
	}
	g.transitionReveal = nil
	g.loadErr = "native 0x24B4D: " + err.Error()
}

func (g *Game) drawNative24B4D(screen *ebiten.Image) bool {
	job := g.transitionReveal
	if job == nil || len(job.palette) != 256 || len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), job.palette)
	copy(img.Pix, g.nativeMapVGA)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	job.drawn = true
	return true
}
