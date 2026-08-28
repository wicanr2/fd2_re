package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type nativeCh20SkyKeyPhase uint8

const (
	nativeCh20SkyKeyPan nativeCh20SkyKeyPhase = iota
	nativeCh20SkyKeyFirstFrames
	nativeCh20SkyKeyANI
	nativeCh20SkyKeyFlash
	nativeCh20SkyKeyRestore
	nativeCh20SkyKeyTailFrames
)

// nativeCh20SkyKeyJob owns the complete blocking presentation called at
// 0x242c9. Indexed framebuffer and DAC snapshots remain separate throughout:
// the white flash and baseline restore must recolour the final ANI indices,
// not replace them with a generic RGBA overlay.
type nativeCh20SkyKeyJob struct {
	spec      campaign.NativeCh20SkyKeySequence
	fdFrames  []fdother.Frame
	aniPixels [][]byte
	aniDAC    [][]byte

	base    []byte
	vga     []byte
	dac     []byte
	palette color.Palette

	phase    nativeCh20SkyKeyPhase
	frame    int
	ticks    int
	drawn    bool
	then     func()
	rollback func()
}

func validateNativeCh20SkyKeyAssets(spec campaign.NativeCh20SkyKeySequence, frames []fdother.Frame, clip *afm.Clip) error {
	if !spec.IsRecoveredContract() {
		return errors.New("native 0x24336 payload differs from recovered contract")
	}
	if len(frames) != spec.FDOTHERFrameCount {
		return fmt.Errorf("FDOTHER #%d frames=%d, want %d", spec.FDOTHERResource, len(frames), spec.FDOTHERFrameCount)
	}
	// Validate every RLE stream before the camera or campaign state changes.
	// Actual compositing happens only after the native pan reaches (14,8).
	for index, frame := range frames {
		if err := frame.Blit(make([]byte, indexedmap.NativeMapVGASize), 320, -1); err != nil {
			return fmt.Errorf("FDOTHER #%d frame %d: %w", spec.FDOTHERResource, index, err)
		}
	}
	if clip == nil || len(clip.IndexedFrames) != spec.ANIFrameCount ||
		len(clip.Palettes) != spec.ANIFrameCount {
		return fmt.Errorf("ANI #%d frame snapshots are incomplete", spec.ANIResource)
	}
	for index := 0; index < spec.ANIFrameCount; index++ {
		if len(clip.IndexedFrames[index]) != indexedmap.NativeMapVGASize || len(clip.Palettes[index]) != 256*3 {
			return fmt.Errorf("ANI #%d frame %d geometry is incomplete", spec.ANIResource, index)
		}
		if _, err := fdother.VGAPaletteFromDAC(clip.Palettes[index]); err != nil {
			return fmt.Errorf("ANI #%d frame %d palette: %w", spec.ANIResource, index, err)
		}
	}
	return nil
}

func loadNativeCh20SkyKeyANI(spec campaign.NativeCh20SkyKeySequence) (*afm.Clip, error) {
	return afm.LoadSeparatedResource(separatedAssetPath("animations"), spec.ANIResource)
}

func (g *Game) startNativeCh20SkyKeySequence(spec campaign.NativeCh20SkyKeySequence, then func()) error {
	if g == nil || g.nativeCh20SkyKey != nil {
		return errors.New("native 0x24336 sequence is already active")
	}
	if !spec.IsRecoveredContract() || !nativeMapAssetsAvailable(g.nativeMapAssets) ||
		g.m == nil || g.st == nil || !g.st.HasNativeMapViewState {
		return errors.New("native 0x24336 indexed map state is unavailable")
	}
	fdPath := nativeFDOTHERPath()
	if fdPath == "" {
		return errors.New("native 0x24336 requires player-provided FDOTHER.DAT")
	}
	frames, err := fdother.DecodeResource(fdPath, spec.FDOTHERResource)
	if err != nil {
		return fmt.Errorf("FDOTHER #%d: %w", spec.FDOTHERResource, err)
	}
	clip, err := loadNativeCh20SkyKeyANI(spec)
	if err != nil {
		return fmt.Errorf("ANI #%d: %w", spec.ANIResource, err)
	}
	if err := validateNativeCh20SkyKeyAssets(spec, frames, clip); err != nil {
		return err
	}
	rollback := snapshotNativeCh20SkyKeyState(g)

	// 0x135dd changes camera and absolute cursor together while preserving the
	// visible cursor. Validate the final state before the first visible step.
	view := g.st.NativeMapViewState
	view.CameraX, view.CameraY = spec.PanGridX, spec.PanGridY
	view.CursorX = view.CameraX + view.VisibleCursorX
	view.CursorY = view.CameraY + view.VisibleCursorY
	carrier := &battle.State{W: g.st.W, H: g.st.H}
	if err := carrier.MaterializeNativeMapViewState(view); err != nil {
		return fmt.Errorf("native 0x24336 pan target: %w", err)
	}
	if !g.syncNativeMapView() {
		rollback()
		return errors.New("native 0x24336 current map view is unavailable")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		rollback()
		return fmt.Errorf("native 0x24336 baseline frame: %w", err)
	}

	job := &nativeCh20SkyKeyJob{
		spec: spec, fdFrames: frames,
		aniPixels: clip.IndexedFrames, aniDAC: clip.Palettes,
		dac:   append([]byte(nil), g.nativeMapDAC...),
		phase: nativeCh20SkyKeyPan, then: then, rollback: rollback,
	}
	g.nativeCh20SkyKey = job
	if err := job.advancePan(g); err != nil {
		job.rollback()
		g.nativeCh20SkyKey = nil
		return err
	}
	return nil
}

// snapshotNativeCh20SkyKeyState protects the caller-visible map transaction.
// Asset decoding is already fully preflighted, but a later indexed HUD/frame
// rejection must still restore the view, timing and framebuffer atomically.
func snapshotNativeCh20SkyKeyState(g *Game) func() {
	state := g.st
	beforeState := *state
	beforeCurX, beforeCurY := g.curX, g.curY
	beforeCamX, beforeCamY := g.camX, g.camY
	beforeWork := append([]byte(nil), g.nativeMapWork...)
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	beforeDAC := append([]byte(nil), g.nativeMapDAC...)
	beforeClock := g.nativeMapClock
	beforePhase := g.nativeFDOTHERPalettePhase
	return func() {
		if g.st == state {
			*state = beforeState
		}
		g.curX, g.curY = beforeCurX, beforeCurY
		g.camX, g.camY = beforeCamX, beforeCamY
		g.nativeMapWork = append(g.nativeMapWork[:0], beforeWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], beforeVGA...)
		g.nativeMapDAC = append(g.nativeMapDAC[:0], beforeDAC...)
		g.nativeMapClock = beforeClock
		g.nativeFDOTHERPalettePhase = beforePhase
	}
}

func (j *nativeCh20SkyKeyJob) refreshPalette() error {
	palette, err := fdother.VGAPaletteFromDAC(j.dac)
	if err != nil {
		return err
	}
	j.palette, j.drawn = palette, false
	return nil
}

func (j *nativeCh20SkyKeyJob) advancePan(g *Game) error {
	view := g.st.NativeMapViewState
	if view.CameraX == j.spec.PanGridX && view.CameraY == j.spec.PanGridY {
		return j.beginFirstFrames(g)
	}
	if view.CameraX < j.spec.PanGridX {
		view.CameraX++
		view.CursorX++
	} else if view.CameraX > j.spec.PanGridX {
		view.CameraX--
		view.CursorX--
	} else if view.CameraY < j.spec.PanGridY {
		view.CameraY++
		view.CursorY++
	} else {
		view.CameraY--
		view.CursorY--
	}
	if err := g.st.MaterializeNativeMapViewState(view); err != nil {
		return fmt.Errorf("pan step: %w", err)
	}
	if !g.syncNativeMapView() {
		return errors.New("pan step could not publish native map view")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		return fmt.Errorf("pan redraw: %w", err)
	}
	j.vga = append(j.vga[:0], g.nativeMapVGA...)
	j.dac = append(j.dac[:0], g.nativeMapDAC...)
	j.ticks = 0
	return j.refreshPalette()
}

func (j *nativeCh20SkyKeyJob) beginFirstFrames(g *Game) error {
	if len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		return errors.New("native 0x24336 baseline framebuffer is incomplete")
	}
	j.base = append(j.base[:0], g.nativeMapVGA...)
	if err := j.fdFrames[j.spec.BaseFrame].Blit(j.base, 320, -1); err != nil {
		return fmt.Errorf("base frame %d: %w", j.spec.BaseFrame, err)
	}
	j.phase, j.frame = nativeCh20SkyKeyFirstFrames, j.spec.FirstFrameStart
	return j.prepareFDFrame(g, true)
}

func (j *nativeCh20SkyKeyJob) prepareFDFrame(g *Game, cycle bool) error {
	j.vga = append(j.vga[:0], j.base...)
	if err := j.fdFrames[j.frame].Blit(j.vga, 320, -1); err != nil {
		return fmt.Errorf("frame %d: %w", j.frame, err)
	}
	if cycle {
		// 0x4DFCC advances a process-global 0..15 phase behind a BIOS tick gate.
		// The remake preserves its relative cycle and process lifetime; exact
		// phase parity remains a separate dynamic-E2 comparison.
		g.nativeFDOTHERPalettePhase = (g.nativeFDOTHERPalettePhase + 1) & 15
		if err := fdother.ApplyNativeDACPaletteCycleE0EF(j.dac, g.nativeFDOTHERPalettePhase); err != nil {
			return err
		}
	}
	j.ticks = nativeDelayTicks(j.spec.FrameWaitBIOSTicks * 55)
	return j.refreshPalette()
}

func (j *nativeCh20SkyKeyJob) prepareANIFrame() error {
	j.vga = append(j.vga[:0], j.aniPixels[j.frame]...)
	j.dac = append(j.dac[:0], j.aniDAC[j.frame]...)
	j.ticks = nativeDelayTicks(j.spec.ANIFrameDelayMs)
	return j.refreshPalette()
}

func (j *nativeCh20SkyKeyJob) beginFlash(baseline []byte) error {
	j.phase = nativeCh20SkyKeyFlash
	j.dac = append(j.dac[:0], baseline...)
	if err := fdother.ApplyVGAPaletteDelta(j.dac, baseline, j.spec.PaletteStart, j.spec.PaletteEnd, j.spec.FlashDelta); err != nil {
		return err
	}
	j.ticks = nativeDelayTicks(j.spec.FlashHoldMs)
	return j.refreshPalette()
}

func (j *nativeCh20SkyKeyJob) beginRestore(baseline []byte) error {
	j.phase = nativeCh20SkyKeyRestore
	j.dac = append(j.dac[:0], baseline...)
	if err := fdother.ApplyVGAPaletteDelta(j.dac, baseline, j.spec.PaletteStart, j.spec.PaletteEnd, j.spec.RestoreDelta); err != nil {
		return err
	}
	j.ticks = nativeDelayTicks(j.spec.RestoreHoldMs)
	return j.refreshPalette()
}

func (j *nativeCh20SkyKeyJob) beginTail(g *Game) error {
	j.phase, j.frame = nativeCh20SkyKeyTailFrames, j.spec.TailFrameStart
	j.dac = append(j.dac[:0], g.nativeMapAssets.PaletteDAC...)
	return j.prepareFDFrame(g, false)
}

func (g *Game) failNativeCh20SkyKey(err error) {
	if job := g.nativeCh20SkyKey; job != nil && job.rollback != nil {
		job.rollback()
	}
	g.nativeCh20SkyKey = nil
	g.loadErr = "native 0x24336: " + err.Error()
}

func (g *Game) stepNativeCh20SkyKey() {
	j := g.nativeCh20SkyKey
	if j == nil || !j.drawn {
		return
	}
	if j.ticks > 0 {
		j.ticks--
		if j.ticks > 0 {
			return
		}
	}
	var err error
	switch j.phase {
	case nativeCh20SkyKeyPan:
		err = j.advancePan(g)
	case nativeCh20SkyKeyFirstFrames:
		if j.frame < j.spec.FirstFrameEnd {
			j.frame++
			err = j.prepareFDFrame(g, true)
		} else {
			j.phase, j.frame = nativeCh20SkyKeyANI, 0
			err = j.prepareANIFrame()
		}
	case nativeCh20SkyKeyANI:
		if j.frame+1 < j.spec.ANIFrameCount {
			j.frame++
			err = j.prepareANIFrame()
		} else {
			err = j.beginFlash(g.nativeMapAssets.PaletteDAC)
		}
	case nativeCh20SkyKeyFlash:
		err = j.beginRestore(g.nativeMapAssets.PaletteDAC)
	case nativeCh20SkyKeyRestore:
		err = j.beginTail(g)
	case nativeCh20SkyKeyTailFrames:
		if j.frame < j.spec.TailFrameEnd {
			j.frame++
			err = j.prepareFDFrame(g, false)
		} else {
			then := j.then
			j.rollback = nil
			g.nativeMapVGA = append(g.nativeMapVGA[:0], j.vga...)
			g.nativeMapDAC = append(g.nativeMapDAC[:0], j.dac...)
			g.nativeCh20SkyKey = nil
			if then != nil {
				then()
			}
			return
		}
	default:
		err = errors.New("unknown presentation phase")
	}
	if err != nil {
		g.failNativeCh20SkyKey(err)
	}
}

func (g *Game) drawNativeCh20SkyKey(screen *ebiten.Image) bool {
	j := g.nativeCh20SkyKey
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
