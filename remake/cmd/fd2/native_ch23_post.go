package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativeCh23AdapterState is the caller-owned raw state shared by the two
// separated native_ch23_loop beats around FDTXT_024 index 3. Names remain
// address-shaped because the original evidence only proves staging, latch and
// timer contracts, not a higher-level transition meaning.
type nativeCh23AdapterState struct {
	staging             []byte
	latch               int
	tickSnapshot        int
	paletteTickSnapshot int
	initialComplete     bool
}

type nativeCh23LoopJob struct {
	spec       fdother.NativeCh23LoopSpec
	stage      int
	draw       int
	rawESI     int
	waitFrames int
	drawn      bool
	palette    color.Palette
	then       func()
	rollback   func()
}

func cloneNativeCh23AdapterState(src *nativeCh23AdapterState) *nativeCh23AdapterState {
	if src == nil {
		return nil
	}
	dst := *src
	dst.staging = append([]byte(nil), src.staging...)
	return &dst
}

func nativeCh23LoopSpec(loop campaign.NativeCh23Loop) fdother.NativeCh23LoopSpec {
	return fdother.NativeCh23LoopSpec{
		Phase: loop.Phase, Repeat: loop.Repeat,
		StageValues: append([]int(nil), loop.StageValues...),
		Palette:     loop.Palette != nil,
	}
}

func (g *Game) snapshotNativeCh23LoopState() func() {
	state := g.st
	beforeState := *state
	beforeAdapter := cloneNativeCh23AdapterState(g.nativeCh23State)
	beforeWork := append([]byte(nil), g.nativeMapWork...)
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	beforeDAC := append([]byte(nil), g.nativeMapDAC...)
	beforeClock := g.nativeMapClock
	beforePhase := g.nativeFDOTHERPalettePhase
	return func() {
		if g.st == state {
			*state = beforeState
		}
		g.nativeCh23State = cloneNativeCh23AdapterState(beforeAdapter)
		g.nativeMapWork = append(g.nativeMapWork[:0], beforeWork...)
		g.nativeMapVGA = append(g.nativeMapVGA[:0], beforeVGA...)
		g.nativeMapDAC = append(g.nativeMapDAC[:0], beforeDAC...)
		g.nativeMapClock = beforeClock
		g.nativeFDOTHERPalettePhase = beforePhase
	}
}

func (g *Game) startNativeCh23Loop(loop campaign.NativeCh23Loop, then func()) error {
	if g == nil || g.nativeCh23Loop != nil || g.nativeCh20SkyKey != nil || g.indexedTransition != nil {
		return errors.New("native ch23 loop is already active")
	}
	spec := nativeCh23LoopSpec(loop)
	if !spec.IsRecoveredContract() {
		return errors.New("native ch23 loop differs from recovered contract")
	}
	if g.st == nil || g.m == nil || !nativeMapAssetsAvailable(g.nativeMapAssets) ||
		!g.st.HasNativeMapViewState || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		len(g.nativeMapVGA) != indexedmap.NativeMapVGASize || len(g.nativeMapDAC) != 256*3 {
		return errors.New("native ch23 indexed map state is unavailable")
	}
	currentTick, ok := g.nativeMapClock.Current()
	if !ok {
		return errors.New("native ch23 BIOS tick provenance is unavailable")
	}

	rollback := g.snapshotNativeCh23LoopState()
	if spec.Phase == "initial" {
		if g.nativeCh23State != nil {
			return errors.New("native ch23 initial loop state already exists")
		}
		path := nativeFDOTHERPath()
		if path == "" {
			return errors.New("native ch23 requires player-provided FDOTHER.DAT")
		}
		frame, err := fdother.DecodeNativeCh23Stage(path)
		if err != nil {
			return fmt.Errorf("FDOTHER #42: %w", err)
		}
		staging := make([]byte, fdother.NativeCh23StageStride*fdother.NativeCh23StageHeight)
		if err := fdother.BlitNativeCh23Stage(frame, staging); err != nil {
			return fmt.Errorf("FDOTHER #42 staging: %w", err)
		}
		g.nativeCh23State = &nativeCh23AdapterState{
			staging: staging, tickSnapshot: currentTick,
			paletteTickSnapshot: currentTick,
		}
	} else if g.nativeCh23State == nil || !g.nativeCh23State.initialComplete {
		return errors.New("native ch23 palette loop lacks completed initial state")
	}

	job := &nativeCh23LoopJob{spec: spec, then: then, rollback: rollback}
	g.nativeCh23Loop = job
	if err := g.prepareNativeCh23Draw(time.Now()); err != nil {
		g.failNativeCh23Loop(err)
		return err
	}
	return nil
}

func (g *Game) prepareNativeCh23Draw(now time.Time) error {
	j, state := g.nativeCh23Loop, g.nativeCh23State
	if j == nil || state == nil || j.stage < 0 || j.stage >= len(j.spec.StageValues) ||
		j.draw < 0 || j.draw >= j.spec.Repeat {
		return errors.New("native ch23 loop cursor is invalid")
	}
	if j.draw == 0 {
		state.latch = j.spec.StageValues[j.stage]
	}

	candidateState := *g.st
	candidateAdapter := cloneNativeCh23AdapterState(state)
	candidateGame := *g
	candidateGame.st = &candidateState
	candidateGame.nativeMapClock = g.nativeMapClock
	candidateGame.nativeMapWork = append([]byte(nil), g.nativeMapWork...)
	candidateGame.nativeMapVGA = append([]byte(nil), g.nativeMapVGA...)
	candidateGame.nativeMapDAC = append([]byte(nil), g.nativeMapDAC...)
	candidatePhase := g.nativeFDOTHERPalettePhase

	rawTick := candidateGame.nativeMapClock.Sample(now)
	if rawTick != candidateAdapter.tickSnapshot {
		if err := fdother.RotateNativeCh23Rows(candidateAdapter.staging, candidateAdapter.latch); err != nil {
			return err
		}
		candidateAdapter.tickSnapshot = rawTick
	}
	if j.spec.Palette {
		if err := fdother.ApplyVGAPaletteSubtraction(
			candidateGame.nativeMapDAC, g.nativeMapAssets.PaletteDAC, 0, 255, j.rawESI,
		); err != nil {
			return err
		}
		if uint16(rawTick)-uint16(candidateAdapter.paletteTickSnapshot) >= 2 {
			candidatePhase = (candidatePhase + 1) & 15
			if err := fdother.ApplyNativeDACPaletteCycleE0EF(candidateGame.nativeMapDAC, candidatePhase); err != nil {
				return err
			}
			candidateAdapter.paletteTickSnapshot = rawTick
		}
	}
	if err := indexedmap.SeedNativeCh23Staging(candidateGame.nativeMapWork, candidateAdapter.staging); err != nil {
		return err
	}
	if err := candidateGame.composeNativeMapFrameAt(now); err != nil {
		return fmt.Errorf("indexed draw: %w", err)
	}
	palette, err := fdother.VGAPaletteFromDAC(candidateGame.nativeMapDAC)
	if err != nil {
		return err
	}

	*g.st = candidateState
	g.nativeCh23State = candidateAdapter
	g.nativeMapClock = candidateGame.nativeMapClock
	g.nativeMapWork = candidateGame.nativeMapWork
	g.nativeMapVGA = candidateGame.nativeMapVGA
	g.nativeMapDAC = candidateGame.nativeMapDAC
	g.nativeFDOTHERPalettePhase = candidatePhase
	j.palette = palette
	j.waitFrames = 3
	j.drawn = false
	return nil
}

func (g *Game) failNativeCh23Loop(err error) {
	if job := g.nativeCh23Loop; job != nil && job.rollback != nil {
		job.rollback()
	}
	g.nativeCh23Loop = nil
	g.loadErr = "native ch23 post: " + err.Error()
}

func (g *Game) finishNativeCh23Loop() {
	j := g.nativeCh23Loop
	if j == nil {
		return
	}
	if j.spec.Phase == "initial" {
		g.nativeCh23State.initialComplete = true
	}
	then := j.then
	j.rollback = nil
	g.nativeCh23Loop = nil
	if then != nil {
		then()
	}
}

func (g *Game) stepNativeCh23LoopAt(now time.Time) {
	j := g.nativeCh23Loop
	if j == nil || !j.drawn {
		return
	}
	if j.waitFrames > 0 {
		j.waitFrames--
		if j.waitFrames > 0 {
			return
		}
	}
	if j.spec.Palette {
		j.rawESI++
	}
	j.draw++
	if j.draw >= j.spec.Repeat {
		j.draw = 0
		j.stage++
	}
	if j.stage >= len(j.spec.StageValues) {
		g.finishNativeCh23Loop()
		return
	}
	if err := g.prepareNativeCh23Draw(now); err != nil {
		g.failNativeCh23Loop(err)
	}
}

func (g *Game) stepNativeCh23Loop() {
	g.stepNativeCh23LoopAt(time.Now())
}

func (g *Game) drawNativeCh23Loop(screen *ebiten.Image) bool {
	j := g.nativeCh23Loop
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
