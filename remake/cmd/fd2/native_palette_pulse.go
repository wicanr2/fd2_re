package main

import (
	"errors"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativePalettePulseJob owns 0x35E5A's inclusive 0..63 / hold / 62..0 DAC
// sequence. Every DAC write must be drawn before advancing; the indexed VGA
// bytes remain constant throughout the pulse.
type nativePalettePulseJob struct {
	deltas   []int
	waits    []int
	step     int
	wait     int
	vga      []byte
	dac      []byte
	baseline []byte
	palette  color.Palette
	drawn    bool
	then     func()
}

func validateNativePalettePulse(spec campaign.NativePalettePulse) bool {
	return spec.RiseStart == 0 && spec.RiseEnd == 63 && spec.RiseDelayMs == 8 &&
		spec.HoldMs == 400 && spec.FallStart == 62 && spec.FallEnd == 0 && spec.FallDelayMs == 8
}

func (g *Game) startNativePalettePulse(spec campaign.NativePalettePulse, then func()) error {
	if g == nil || g.nativePalettePulse != nil || g.nativePaletteRamp != nil ||
		g.nativeUnitPresent != nil || g.transitionReveal != nil || g.indexedTransition != nil {
		return errors.New("native 0x35E5A palette presenter is already active")
	}
	if !validateNativePalettePulse(spec) || !nativeMapAssetsAvailable(g.nativeMapAssets) {
		return errors.New("native indexed framebuffer or DAC baseline unavailable")
	}
	if err := g.ensureNativePaletteFrame(); err != nil {
		return err
	}
	baseline := append([]byte(nil), g.nativeMapAssets.PaletteDAC...)
	dac := append([]byte(nil), baseline...)
	if len(g.nativeMapDAC) == len(baseline) {
		copy(dac, g.nativeMapDAC)
	}
	job := &nativePalettePulseJob{
		vga: append([]byte(nil), g.nativeMapVGA...), dac: dac, baseline: baseline, then: then,
	}
	for delta := spec.RiseStart; delta <= spec.RiseEnd; delta++ {
		job.deltas = append(job.deltas, delta)
		wait := nativeDelayTicks(spec.RiseDelayMs)
		if delta == spec.RiseEnd {
			wait += nativeDelayTicks(spec.HoldMs)
		}
		job.waits = append(job.waits, wait)
	}
	for delta := spec.FallStart; delta >= spec.FallEnd; delta-- {
		job.deltas = append(job.deltas, delta)
		job.waits = append(job.waits, nativeDelayTicks(spec.FallDelayMs))
	}
	if len(job.deltas) != 127 || len(job.waits) != len(job.deltas) {
		return errors.New("native 0x35E5A schedule is incomplete")
	}
	// Preflight every immutable-baseline write and palette conversion before
	// publishing step zero. No partially valid editable schedule can flash.
	probe := append([]byte(nil), dac...)
	for _, delta := range job.deltas {
		if err := fdother.ApplyVGAPaletteDelta(probe, baseline, 0, 255, delta); err != nil {
			return err
		}
		if _, err := fdother.VGAPaletteFromDAC(probe); err != nil {
			return err
		}
	}
	if err := job.applyCurrent(); err != nil {
		return err
	}
	g.nativePalettePulse = job
	return nil
}

func (j *nativePalettePulseJob) applyCurrent() error {
	if j == nil || j.step < 0 || j.step >= len(j.deltas) || len(j.waits) != len(j.deltas) {
		return errors.New("native 0x35E5A palette step unavailable")
	}
	if err := fdother.ApplyVGAPaletteDelta(j.dac, j.baseline, 0, 255, j.deltas[j.step]); err != nil {
		return err
	}
	palette, err := fdother.VGAPaletteFromDAC(j.dac)
	if err != nil {
		return err
	}
	j.palette, j.drawn = palette, false
	j.wait = j.waits[j.step] - 1
	if j.wait < 0 {
		j.wait = 0
	}
	return nil
}

func (g *Game) stepNativePalettePulse() {
	j := g.nativePalettePulse
	if j == nil || !j.drawn {
		return
	}
	if j.wait > 0 {
		j.wait--
		return
	}
	j.step++
	if j.step >= len(j.deltas) {
		then := j.then
		g.nativeMapDAC = append(g.nativeMapDAC[:0], j.dac...)
		g.nativePalettePulse = nil
		if then != nil {
			then()
		}
		return
	}
	if err := j.applyCurrent(); err != nil {
		g.loadErr = "native 0x35E5A palette pulse: " + err.Error()
		g.nativePalettePulse = nil
	}
}

func (g *Game) drawNativePalettePulse(screen *ebiten.Image) bool {
	j := g.nativePalettePulse
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
