package main

import (
	"bytes"
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

type nativeIndexedTransitionPhase uint8

const (
	nativeTransitionPass nativeIndexedTransitionPhase = iota
	nativeTransitionTail
	nativeTransitionPalette
)

// nativeIndexedTransitionJob owns one complete 0x24618 presentation. Indexed
// pixels and the six-bit DAC remain separate, just as they do in mode 13h.
// Draw acknowledges every native present so multiple 5/4ms operations cannot
// collapse into an unobservable final frame on the host's 60Hz update clock.
type nativeIndexedTransitionJob struct {
	input    indexedmap.NativeTransitionFrameInput
	schedule fdother.NativeIndexedTransitionSchedule
	spec     campaign.HandlerIndexedTransition
	work     []byte
	vga      []byte
	frameVGA [][]byte
	dac      []byte
	baseline []byte
	palette  color.Palette

	phase       nativeIndexedTransitionPhase
	frame       int
	paletteStep int
	tailTicks   int
	drawn       bool
	then        func()
}

func nativeDelayTicks(ms int) int {
	if ms <= 0 {
		return 0
	}
	return (ms*60 + 999) / 1000
}

func validateNativeIndexedTransitionSpec(s campaign.HandlerIndexedTransition) error {
	if s.Frames != 9 || s.FrameDelayMs != 5 || s.TailDelayMs != 500 ||
		s.StartY != 0 || s.EndY != fdother.NativeTransitionStageHeight ||
		s.ClipWidth != fdother.NativeTransitionStageWidth ||
		s.ClipHeight != fdother.NativeTransitionStageHeight ||
		s.PaletteRangeStart != 0 || s.PaletteRangeEnd != 255 ||
		s.PaletteDeltaStart != 0 || s.PaletteDeltaEnd != 62 ||
		s.PaletteDeltaStep != 2 || s.PaletteDelayMs != 4 {
		return errors.New("native 0x24618 payload differs from recovered geometry/timing")
	}
	return nil
}

// resolveNativeIndexedTransitionSpec keeps the dynamic first two arguments
// of 0x24618 distinct from static authored geometry.  The recovered handlers
// use three recovered raw cursor expressions: ch21 call-site 0x245ce pushes
// [0x53ab9] and [0x53abd]+3; ch22 pre call-site 0x336e5 pushes
// [0x53ab9] and [0x53abd]+5; ch27 pre call-site 0x33ce2 pushes
// [0x53ab9]+6 and [0x53abd]+5.  The call-site is part of Beat.Source so an
// authored offset cannot silently acquire a different native meaning.
func (g *Game) resolveNativeIndexedTransitionSpec(spec campaign.HandlerIndexedTransition, source string) (campaign.HandlerIndexedTransition, error) {
	if spec.CursorSource == "" {
		return spec, nil
	}
	if spec.CursorSource != "native_relative_cursor" {
		return campaign.HandlerIndexedTransition{}, errors.New("native 0x24618 cursor source is not proven")
	}
	validOffset := (source == "0x245ce" && spec.CursorXOffset == 0 && spec.CursorYOffset == 3) ||
		(source == "0x336e5" && spec.CursorXOffset == 0 && spec.CursorYOffset == 5) ||
		(source == "0x33ce2" && spec.CursorXOffset == 6 && spec.CursorYOffset == 5)
	if !validOffset {
		return campaign.HandlerIndexedTransition{}, fmt.Errorf("native 0x24618 cursor offset is not proven for source %s", source)
	}
	if g == nil {
		return campaign.HandlerIndexedTransition{}, errors.New("native 0x24618 relative cursor provenance unavailable")
	}
	var view battle.NativeMapViewState
	switch {
	case g.st != nil && g.st.HasNativeMapViewState:
		view = g.st.NativeMapViewState
	case g.hasStoryNativeMapView:
		// LOADCH scenes keep the six raw map-view globals outside battle.State;
		// this is the same provenance, without inventing a battle runtime array.
		view = g.storyNativeMapView
	default:
		return campaign.HandlerIndexedTransition{}, errors.New("native 0x24618 relative cursor provenance unavailable")
	}
	x, y := view.VisibleCursorX+spec.CursorXOffset, view.VisibleCursorY+spec.CursorYOffset
	if x < 0 || x >= fdother.NativeTransitionStageWidth || y < 0 || y >= fdother.NativeTransitionStageHeight {
		return campaign.HandlerIndexedTransition{}, errors.New("native 0x24618 relative cursor is outside indexed stage")
	}
	spec.TileX, spec.TileY = x, y
	return spec, nil
}

func (g *Game) buildNativeIndexedTransitionInput() (indexedmap.NativeTransitionFrameInput, error) {
	return g.buildNativeIndexedTransitionInputForActors(g.storyActors)
}

func (g *Game) buildNativeIndexedTransitionInputForActors(actorsSource []battle.Unit) (indexedmap.NativeTransitionFrameInput, error) {
	if g.m == nil {
		return indexedmap.NativeTransitionFrameInput{}, errors.New("native map unavailable")
	}
	state := &battle.State{W: g.m.W, H: g.m.H}
	actors := make([]battle.Unit, len(actorsSource))
	units := make([]*battle.Unit, len(actors))
	for i := range actorsSource {
		actors[i] = actorsSource[i]
		units[i] = &actors[i]
	}
	if err := state.AppendNativeMapSelectorBatch(units); err != nil {
		return indexedmap.NativeTransitionFrameInput{}, fmt.Errorf("selector roster: %w", err)
	}
	return g.buildNativeIndexedTransitionInputForState(state)
}

// buildNativeIndexedTransitionInputForState preserves the battle-session
// selector cache and animation/timing globals.  Reconstructing those from a
// flat unit slice would reset a turn-four transition to the chapter-opening
// phase even when every unit record was otherwise correct.
func (g *Game) buildNativeIndexedTransitionInputForState(state *battle.State) (indexedmap.NativeTransitionFrameInput, error) {
	a, field := g.nativeMapAssets, g.m
	if !nativeMapAssetsAvailable(a) || field == nil || field.TileW <= 0 || field.TileH <= 0 ||
		field.W <= 0 || field.H <= 0 || len(field.Tiles) != field.W*field.H ||
		!bytes.Equal(field.NativeTerrainControl, a.Controls) || state == nil ||
		state.W != field.W || state.H != field.H {
		return indexedmap.NativeTransitionFrameInput{}, errors.New("native map assets/field unavailable")
	}
	if int(g.camX)%field.TileW != 0 || int(g.camY)%field.TileH != 0 ||
		g.camX != float64(int(g.camX)) || g.camY != float64(int(g.camY)) {
		return indexedmap.NativeTransitionFrameInput{}, errors.New("native camera is not tile-aligned")
	}
	cells, err := indexedmap.BuildNativeTerrainCells(field.Tiles, field.NativeTileBlitModes)
	if err != nil {
		return indexedmap.NativeTransitionFrameInput{}, fmt.Errorf("terrain cells: %w", err)
	}
	roster, err := state.NativeMapFrameRoster()
	if err != nil {
		return indexedmap.NativeTransitionFrameInput{}, fmt.Errorf("native roster: %w", err)
	}
	lutIndex, err := fdother.NativeTerrainLUTIndex(roster.TerrainPhase)
	if err != nil || lutIndex < 0 || lutIndex >= len(a.LUTs) || len(a.LUTs[lutIndex]) != 256 {
		return indexedmap.NativeTransitionFrameInput{}, errors.New("terrain phase LUT unavailable")
	}
	return indexedmap.NativeTransitionFrameInput{
		TerrainBank: a.Terrain, UnitBank: a.Units, ForegroundBank: a.Terrain,
		SelectorCache: state.NativeMapSelectorCache,
		Cells:         cells, Controls: a.Controls, TerrainLUT: a.LUTs[lutIndex],
		MapWidth: field.W, CameraX: int(g.camX) / field.TileW, CameraY: int(g.camY) / field.TileH,
		Flip: roster.TerrainFlip, TerrainCycle: roster.Cycles.Idle,
		IdleCycle: roster.Cycles.Idle, MovingCycle: roster.Cycles.Moving,
		PixelShift: roster.UnitPixelShift,
		Units:      roster.Units, ForegroundUnits: roster.Foreground,
	}, nil
}

func (g *Game) startNativeIndexedTransition(spec campaign.HandlerIndexedTransition, source string, then func()) error {
	if g.indexedTransition != nil {
		return errors.New("native 0x24618 transition already active")
	}
	resolvedSpec, err := g.resolveNativeIndexedTransitionSpec(spec, source)
	if err != nil {
		return err
	}
	spec = resolvedSpec
	if err := validateNativeIndexedTransitionSpec(spec); err != nil {
		return err
	}
	schedule, err := fdother.BuildNativeIndexedTransitionSchedule(spec.RadialRadius, spec.RadialRadiusStep)
	if err != nil {
		return err
	}
	in, err := g.buildNativeIndexedTransitionInput()
	if err != nil {
		return err
	}
	a := g.nativeMapAssets
	job := &nativeIndexedTransitionJob{
		input: in, schedule: schedule, spec: spec,
		work:     make([]byte, indexedmap.NativeUnitPresentWorkSize),
		vga:      make([]byte, indexedmap.NativeMapVGASize),
		dac:      append([]byte(nil), a.PaletteDAC...),
		baseline: append([]byte(nil), a.PaletteDAC...),
		phase:    nativeTransitionPass, then: then,
	}
	for i := range schedule.Frames {
		if err := job.composePass(i, a.LUTs); err != nil {
			return fmt.Errorf("preflight pass %d: %w", i, err)
		}
	}
	palette, err := fdother.VGAPaletteFromDAC(job.dac)
	if err != nil {
		return err
	}
	job.frame, job.palette, job.drawn = 0, palette, false
	copy(job.vga, job.frameVGA[0])
	g.indexedTransition = job
	return nil
}

func (j *nativeIndexedTransitionJob) composePass(index int, luts [][]byte) error {
	if index < 0 || index >= len(j.schedule.Frames) {
		return errors.New("native 0x24618 pass outside schedule")
	}
	frame := j.schedule.Frames[index]
	lut, err := fdother.NativeIndexedTransitionLUT(luts, frame.LUTIndex)
	if err != nil {
		return err
	}
	pass, err := fdother.BuildNativeIndexedTransitionPass(
		j.spec.TileX, j.spec.TileY, frame.Radius, j.spec.StartY, j.spec.EndY,
	)
	if err != nil {
		return err
	}
	if err := indexedmap.ComposeNativeTransitionFrame(j.work, j.vga, j.input, pass, lut); err != nil {
		return err
	}
	j.frameVGA = append(j.frameVGA, append([]byte(nil), j.vga...))
	return nil
}

func (g *Game) stepNativeIndexedTransition() {
	j := g.indexedTransition
	if j == nil {
		return
	}
	switch j.phase {
	case nativeTransitionPass:
		if !j.drawn {
			return
		}
		if j.frame+1 < len(j.schedule.Frames) {
			j.frame++
			copy(j.vga, j.frameVGA[j.frame])
			j.drawn = false
			return
		}
		j.phase = nativeTransitionTail
		j.tailTicks = nativeDelayTicks(j.schedule.TailDelayMs)
	case nativeTransitionTail:
		if j.tailTicks > 0 {
			j.tailTicks--
			return
		}
		j.phase = nativeTransitionPalette
		j.paletteStep = 0
		j.drawn = false
		if err := g.applyNativeIndexedTransitionPalette(); err != nil {
			g.loadErr = "beat indexed_transition: " + err.Error()
			g.indexedTransition = nil
		}
	case nativeTransitionPalette:
		if !j.drawn {
			return
		}
		j.paletteStep++
		if j.paletteStep >= len(j.schedule.PaletteDeltas) {
			then := j.then
			g.nativeMapWork = append(g.nativeMapWork[:0], j.work...)
			g.nativeMapVGA = append(g.nativeMapVGA[:0], j.vga...)
			g.indexedTransition = nil
			if then != nil {
				then()
			}
			return
		}
		if err := g.applyNativeIndexedTransitionPalette(); err != nil {
			g.loadErr = "beat indexed_transition: " + err.Error()
			g.indexedTransition = nil
		}
	}
}

func (g *Game) applyNativeIndexedTransitionPalette() error {
	j := g.indexedTransition
	if j == nil || j.paletteStep < 0 || j.paletteStep >= len(j.schedule.PaletteDeltas) {
		return errors.New("native 0x24618 palette step unavailable")
	}
	if err := fdother.ApplyVGAPaletteDelta(
		j.dac, j.baseline, j.spec.PaletteRangeStart, j.spec.PaletteRangeEnd,
		j.schedule.PaletteDeltas[j.paletteStep],
	); err != nil {
		return err
	}
	palette, err := fdother.VGAPaletteFromDAC(j.dac)
	if err != nil {
		return err
	}
	j.palette, j.drawn = palette, false
	return nil
}

func (g *Game) drawNativeIndexedTransition(screen *ebiten.Image) bool {
	j := g.indexedTransition
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
