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

type nativeCommandHealPresentationPhase uint8

const (
	nativeCommandHealFrames nativeCommandHealPresentationPhase = iota
	nativeCommandHealMidHold
	nativeCommandHealTailHold
)

// nativeCommandHealPresentationJob owns the recovered 0x21EB1 presentation.
// Every frame is precomposed from the same strict native map input, matching
// the original full-buffer restore before each 0x22046 pass.
type nativeCommandHealPresentationJob struct {
	schedule fdother.NativeCommandHealPresentationSchedule
	frames   [][]byte
	baseline []byte
	palette  color.Palette
	frame    int
	phase    nativeCommandHealPresentationPhase
	hold     int
	drawn    bool
	then     func()
}

func (g *Game) startNativeCommandHealPresentation(commandID int, then func()) error {
	if g == nil {
		return errors.New("native command heal presentation game unavailable")
	}
	if g.nativeHealPresentation != nil || g.indexedTransition != nil {
		return errors.New("native command heal presentation already active")
	}
	if g.st == nil || !g.st.HasNativeMapViewState ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return errors.New("native command heal presentation raw view/frame unavailable")
	}
	schedule, err := fdother.BuildNativeCommandHealPresentationSchedule(commandID)
	if err != nil {
		return err
	}
	in, err := g.buildNativeIndexedTransitionInputForState(g.st)
	if err != nil {
		return err
	}
	view := g.st.NativeMapViewState
	centerX := 24*view.VisibleCursorX + 12
	centerY := 24*view.VisibleCursorY + 16
	frames := make([][]byte, 0, len(schedule.Frames))
	for index, spec := range schedule.Frames {
		lut, err := fdother.NativeIndexedTransitionLUT(g.nativeMapAssets.LUTs, spec.LUTIndex)
		if err != nil {
			return fmt.Errorf("native command heal presentation frame %d LUT: %w", index, err)
		}
		pass, err := fdother.BuildNativeIndexedTransitionPass(
			centerX, centerY, spec.Radius, 0, fdother.NativeTransitionStageHeight,
		)
		if err != nil {
			return fmt.Errorf("native command heal presentation frame %d geometry: %w", index, err)
		}
		work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
		vga := make([]byte, indexedmap.NativeMapVGASize)
		if err := indexedmap.ComposeNativeTransitionFrame(work, vga, in, pass, lut); err != nil {
			return fmt.Errorf("native command heal presentation frame %d: %w", index, err)
		}
		frames = append(frames, vga)
	}
	palette := g.nativeMapAssets.Palette
	if len(g.nativeMapDAC) == 256*3 {
		palette, err = fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			return err
		}
	}
	if len(palette) != 256 {
		return errors.New("native command heal presentation palette unavailable")
	}
	g.nativeHealPresentation = &nativeCommandHealPresentationJob{
		schedule: schedule,
		frames:   frames,
		baseline: append([]byte(nil), g.nativeMapVGA...),
		palette:  append(color.Palette(nil), palette...),
		phase:    nativeCommandHealFrames,
		then:     then,
	}
	g.playSFX(schedule.SampleIndex)
	return nil
}

func (g *Game) stepNativeCommandHealPresentation() {
	j := g.nativeHealPresentation
	if j == nil {
		return
	}
	switch j.phase {
	case nativeCommandHealFrames:
		if !j.drawn {
			return
		}
		if j.frame+1 == j.schedule.MidFrame {
			j.phase = nativeCommandHealMidHold
			j.hold = nativeDelayTicks(j.schedule.MidDelayMs)
			return
		}
		if j.frame+1 >= len(j.frames) {
			j.phase = nativeCommandHealTailHold
			j.hold = nativeDelayTicks(j.schedule.TailDelayMs)
			j.drawn = false
			return
		}
		j.frame++
		j.drawn = false
	case nativeCommandHealMidHold:
		if j.hold > 0 {
			j.hold--
			return
		}
		j.frame = j.schedule.MidFrame
		j.phase = nativeCommandHealFrames
		j.drawn = false
	case nativeCommandHealTailHold:
		if j.hold > 0 {
			j.hold--
			return
		}
		then := j.then
		g.nativeMapVGA = append(g.nativeMapVGA[:0], j.baseline...)
		g.nativeHealPresentation = nil
		if then != nil {
			then()
		}
	}
}

func (g *Game) drawNativeCommandHealPresentation(screen *ebiten.Image) bool {
	j := g.nativeHealPresentation
	if j == nil || len(j.palette) != 256 {
		return false
	}
	pixels := j.baseline
	if j.phase != nativeCommandHealTailHold {
		if j.frame < 0 || j.frame >= len(j.frames) {
			return false
		}
		pixels = j.frames[j.frame]
		j.drawn = true
	}
	if len(pixels) != indexedmap.NativeMapVGASize {
		return false
	}
	img := image.NewPaletted(image.Rect(0, 0, 320, 200), j.palette)
	copy(img.Pix, pixels)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(2, 2)
	screen.DrawImage(ebiten.NewImageFromImage(img), op)
	return true
}
