package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativeCh28PostPresentJob owns sub_1DB65's 13 pose frames, two six-frame
// FDOTHER#5 overlay loops, raw sample index 3, and final 0x11CAC redraw. The
// address-shaped name avoids inventing a player-facing effect name.
type nativeCh28PostPresentJob struct {
	workFrames  [][]byte
	vgaFrames   [][]byte
	stateFrames []*battle.State
	waits       []int
	palette     color.Palette
	frame       int
	wait        int
	drawn       bool
	sfxAt       int
	then        func()
	rollback    func()
}

func cloneNativeCh28PostState(source *battle.State) (*battle.State, error) {
	return cloneNativeTurnStagingState(source)
}

func (g *Game) nativeCh28PostFrameInput(state *battle.State) (indexedmap.NativeFrameInput, error) {
	if g == nil || state == nil {
		return indexedmap.NativeFrameInput{}, errors.New("native ch28 post: state unavailable")
	}
	before := g.st
	g.st = state
	hud, ok := g.nativeMapHUDInput()
	g.st = before
	if !ok {
		return indexedmap.NativeFrameInput{}, errors.New("native ch28 post: HUD provenance unavailable")
	}
	return buildNativeMapFrameInput(g.nativeMapAssets, g.m, state, nativeMapFrameRuntime{HUD: hud})
}

func (g *Game) startNativeCh28PostPresent(then func()) error {
	if g == nil || g.nativeCh28PostPresent != nil || g.nativeUnitPresent != nil ||
		g.native2189A != nil || g.transitionReveal != nil || g.indexedTransition != nil ||
		g.nativePaletteRamp != nil || g.nativePalettePulse != nil ||
		g.nativeCh20SkyKey != nil || g.nativeCh23Loop != nil {
		return errors.New("native 0x1DB65 presentation is already active")
	}
	if g.st == nil || g.m == nil || !nativeMapAssetsAvailable(g.nativeMapAssets) ||
		len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize ||
		!g.st.HasNativeMapViewState {
		return errors.New("native 0x1DB65 indexed map state is unavailable")
	}
	if err := fdother.ValidateNativeCh28PostLMI(g.nativeMapAssets.CommandHealDigits); err != nil {
		return err
	}
	if !osMuteOrShot(g) && (g.sfx == nil || len(g.sfx[fdother.NativeCh28PostSFXIndex]) == 0) {
		return fmt.Errorf(
			"native 0x1DB65 raw sample unavailable: resource=%d index=%d",
			fdother.NativeCh28PostSFXResource, fdother.NativeCh28PostSFXIndex,
		)
	}

	candidate, err := cloneNativeCh28PostState(g.st)
	if err != nil {
		return err
	}
	if err := battle.ApplyNativeCh28PostRawPrelude(candidate); err != nil {
		return err
	}
	view := candidate.NativeMapViewState
	plan, err := battle.PlanNativeCh28PostPresentation(candidate, view.CameraX, view.CameraY)
	if err != nil {
		return err
	}

	job := &nativeCh28PostPresentJob{
		palette: append(color.Palette(nil), g.nativeMapAssets.Palette...),
		sfxAt:   fdother.NativeCh28PostPoseFrames,
		then:    then,
	}
	if len(g.nativeMapDAC) == 256*3 {
		if current, paletteErr := fdother.VGAPaletteFromDAC(g.nativeMapDAC); paletteErr == nil {
			job.palette = current
		}
	}
	appendFrame := func(work, vga []byte, state *battle.State, wait int) error {
		snapshot, cloneErr := cloneNativeCh28PostState(state)
		if cloneErr != nil {
			return cloneErr
		}
		job.workFrames = append(job.workFrames, append([]byte(nil), work...))
		job.vgaFrames = append(job.vgaFrames, append([]byte(nil), vga...))
		job.stateFrames = append(job.stateFrames, snapshot)
		if wait < 1 {
			wait = 1
		}
		job.waits = append(job.waits, wait)
		return nil
	}

	work := append([]byte(nil), g.nativeMapWork...)
	vga := append([]byte(nil), g.nativeMapVGA...)
	for frame := 0; frame < fdother.NativeCh28PostPoseFrames; frame++ {
		if err := battle.ApplyNativeCh28PostPoseFrame(candidate, frame); err != nil {
			return err
		}
		in, err := g.nativeCh28PostFrameInput(candidate)
		if err != nil {
			return err
		}
		transition := nativeTransitionInput(in)
		if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(work, transition); err != nil {
			return fmt.Errorf("native 0x1DB65 pose frame %d terrain: %w", frame, err)
		}
		if err := indexedmap.RedrawNativeUnitPresentObjects(work, transition); err != nil {
			return fmt.Errorf("native 0x1DB65 pose frame %d objects: %w", frame, err)
		}
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
			return fmt.Errorf("native 0x1DB65 pose frame %d viewport: %w", frame, err)
		}
		if err := appendFrame(work, vga, candidate, 3); err != nil {
			return err
		}
	}
	if err := battle.ApplyNativeCh28PostInactiveMark(candidate); err != nil {
		return err
	}
	afterInput, err := g.nativeCh28PostFrameInput(candidate)
	if err != nil {
		return err
	}
	transition := nativeTransitionInput(afterInput)
	staged := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(staged, transition); err != nil {
		return fmt.Errorf("native 0x1DB65 staged terrain: %w", err)
	}
	if err := indexedmap.RedrawNativeUnitPresentObjects(staged, transition); err != nil {
		return fmt.Errorf("native 0x1DB65 staged objects: %w", err)
	}
	for index, frame := range fdother.NativeCh28PostOverlayFrames() {
		if frame.Phase == "overlay_b" {
			work = append(work[:0], staged...)
		}
		for _, target := range plan.Targets {
			if err := fdother.BlitNativeCh28PostOverlay(
				g.nativeMapAssets.CommandHealDigits[frame.Entry], work, target.Origin,
			); err != nil {
				return fmt.Errorf("native 0x1DB65 overlay frame %d slot%d: %w", index, target.Slot, err)
			}
		}
		if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
			return fmt.Errorf("native 0x1DB65 overlay frame %d viewport: %w", index, err)
		}
		if err := appendFrame(work, vga, candidate, 3); err != nil {
			return err
		}
	}
	steadyWork := append([]byte(nil), work...)
	steadyVGA := append([]byte(nil), vga...)
	if err := indexedmap.ComposeNativeFrame(steadyWork, steadyVGA, afterInput); err != nil {
		return fmt.Errorf("native 0x1DB65 final steady redraw: %w", err)
	}
	if err := appendFrame(steadyWork, steadyVGA, candidate, 1); err != nil {
		return err
	}
	if len(job.vgaFrames) != 26 || len(job.workFrames) != len(job.vgaFrames) ||
		len(job.stateFrames) != len(job.vgaFrames) || len(job.waits) != len(job.vgaFrames) {
		return errors.New("native 0x1DB65 preflight produced an invalid 13+6+6+steady schedule")
	}

	beforeState, err := cloneNativeCh28PostState(g.st)
	if err != nil {
		return err
	}
	beforeWork := append([]byte(nil), g.nativeMapWork...)
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	job.rollback = func() {
		g.st = beforeState
		g.nativeMapWork = append(g.nativeMapWork[:0], beforeWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], beforeVGA...)
	}
	g.nativeCh28PostPresent = job
	g.publishNativeCh28PostFrame()
	return nil
}

func (g *Game) publishNativeCh28PostFrame() {
	j := g.nativeCh28PostPresent
	if j == nil || j.frame < 0 || j.frame >= len(j.vgaFrames) {
		return
	}
	g.st = j.stateFrames[j.frame]
	g.nativeMapWork = append(g.nativeMapWork[:0], j.workFrames[j.frame]...)
	g.nativeMapVGA = append(g.nativeMapVGA[:0], j.vgaFrames[j.frame]...)
	j.drawn = false
	j.wait = j.waits[j.frame] - 1
	if j.wait < 0 {
		j.wait = 0
	}
}

func (g *Game) stepNativeCh28PostPresent() {
	j := g.nativeCh28PostPresent
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
		g.nativeCh28PostPresent = nil
		if then != nil {
			then()
		}
		return
	}
	if next == j.sfxAt {
		g.playSFX(fdother.NativeCh28PostSFXIndex)
	}
	j.frame = next
	g.publishNativeCh28PostFrame()
}

func (g *Game) failNativeCh28PostPresent(err error) {
	if j := g.nativeCh28PostPresent; j != nil && j.rollback != nil {
		j.rollback()
	}
	g.nativeCh28PostPresent = nil
	g.loadErr = "native 0x1DB65: " + err.Error()
}

func (g *Game) drawNativeCh28PostPresent(screen *ebiten.Image) bool {
	j := g.nativeCh28PostPresent
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
