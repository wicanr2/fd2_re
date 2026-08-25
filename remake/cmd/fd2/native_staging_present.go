package main

import (
	"errors"
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// nativeFocusEndpoint predicts the exact 0x12CEA safe-band endpoint without
// publishing a focus step.  sub_33F78 needs this private endpoint to preflight
// the following 0x22253 transaction before the first visible mutation.
func (g *Game) nativeFocusEndpoint(targetX, targetY int) (int, int, battle.NativeMapViewState, error) {
	if g == nil || g.m == nil || g.m.TileW <= 0 || g.m.TileH <= 0 ||
		targetX < 0 || targetY < 0 || targetX >= g.m.W || targetY >= g.m.H ||
		int(g.camX)%g.m.TileW != 0 || int(g.camY)%g.m.TileH != 0 {
		return 0, 0, battle.NativeMapViewState{}, errors.New("native 0x33F78 focus endpoint is unavailable")
	}
	originX, originY := int(g.camX)/g.m.TileW, int(g.camY)/g.m.TileH
	cursorX, cursorY := g.curX, g.curY
	screenX, screenY := cursorX-originX, cursorY-originY
	if g.hasStoryNativeMapView {
		screenX, screenY = g.storyNativeMapView.VisibleCursorX, g.storyNativeMapView.VisibleCursorY
	}
	maxOriginX, maxOriginY := g.m.W-13, g.m.H-8
	if maxOriginX < 0 {
		maxOriginX = 0
	}
	if maxOriginY < 0 {
		maxOriginY = 0
	}
	for steps := 0; cursorX != targetX || cursorY != targetY; steps++ {
		if steps > g.m.W+g.m.H+32 {
			return 0, 0, battle.NativeMapViewState{}, errors.New("native 0x33F78 focus endpoint exceeded map bound")
		}
		switch {
		case cursorX > targetX:
			cursorX--
			if screenX < 2 && originX > 0 {
				originX--
			} else {
				screenX--
			}
		case cursorX < targetX:
			cursorX++
			if screenX > 10 && originX < maxOriginX {
				originX++
			} else {
				screenX++
			}
		case cursorY > targetY:
			cursorY--
			if screenY < 2 && originY > 0 {
				originY--
			} else {
				screenY--
			}
		case cursorY < targetY:
			cursorY++
			if screenY > 5 && originY < maxOriginY {
				originY++
			} else {
				screenY++
			}
		}
	}
	view := battle.NativeMapViewState{
		CameraX: originX, CameraY: originY,
		CursorX: targetX, CursorY: targetY,
		VisibleCursorX: screenX, VisibleCursorY: screenY,
	}
	carrier := &battle.State{W: g.m.W, H: g.m.H}
	if err := carrier.MaterializeNativeMapViewState(view); err != nil {
		return 0, 0, battle.NativeMapViewState{}, err
	}
	return originX * g.m.TileW, originY * g.m.TileH, carrier.NativeMapViewState, nil
}

func (g *Game) buildNativeStagingPresentJob(spec campaign.NativeStagingPresent) (*nativeUnitPresentJob, error) {
	if g == nil || spec.Slot < 0 || spec.X < 0 || spec.Y < 0 ||
		spec.Slot >= len(g.storyActors) || !g.hasStoryNativeMapView ||
		g.nativeUnitPresent != nil || g.focusJob != nil {
		return nil, errors.New("native 0x33F78 story runtime array or focus owner is unavailable")
	}
	predictedCamX, predictedCamY, predictedView, err := g.nativeFocusEndpoint(spec.X, spec.Y)
	if err != nil {
		return nil, err
	}
	state := &battle.State{W: g.m.W, H: g.m.H}
	actors := cloneStoryUnitPointers(g.storyActors)
	if err := state.AppendNativeMapSelectorBatch(actors); err != nil {
		return nil, fmt.Errorf("native 0x33F78 selector array: %w", err)
	}
	if err := state.MaterializeNativeMapViewState(predictedView); err != nil {
		return nil, err
	}
	shadow := *g
	shadow.camX, shadow.camY = float64(predictedCamX), float64(predictedCamY)
	shadow.curX, shadow.curY = spec.X, spec.Y
	input, err := shadow.buildNativeIndexedTransitionInputForState(state)
	if err != nil {
		return nil, fmt.Errorf("native 0x33F78 predicted frame: %w", err)
	}
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(work, input); err != nil {
		return nil, err
	}
	if err := indexedmap.RedrawNativeUnitPresentObjects(work, input); err != nil {
		return nil, err
	}
	vga := make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
		return nil, err
	}
	present := campaign.NativeUnitPresent{
		Slot: spec.Slot,
		NewX: spec.X, NewY: spec.Y,
		VisualX: spec.X, VisualY: spec.Y,
	}
	var then func()
	if g.camp != nil {
		then = g.beatAdvance
	}
	job, err := shadow.buildNativeUnitPresentJob(present, state, work, vga, then)
	if err != nil {
		return nil, err
	}
	beforeUnit := g.storyActors[spec.Slot]
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	beforeCamX, beforeCamY, beforeCurX, beforeCurY := g.camX, g.camY, g.curX, g.curY
	beforeView := g.storyNativeMapView
	job.mutate = func() bool {
		return spec.Slot < len(g.storyActors) &&
			g.storyActors[spec.Slot].SetNativeMapCoordinatesRaw(spec.X, spec.Y)
	}
	job.rollback = func() {
		if spec.Slot < len(g.storyActors) {
			g.storyActors[spec.Slot] = beforeUnit
		}
		g.nativeMapWork, g.nativeMapVGA = append(g.nativeMapWork[:0], beforeWork...), append(g.nativeMapVGA[:0], beforeVGA...)
		g.camX, g.camY, g.curX, g.curY = beforeCamX, beforeCamY, beforeCurX, beforeCurY
		g.storyNativeMapView = beforeView
	}
	return job, nil
}
func (g *Game) startNativeStagingPresent(spec campaign.NativeStagingPresent) error {
	job, err := g.buildNativeStagingPresentJob(spec)
	if err != nil {
		return err
	}
	g.focusJob = &focusUnitJob{targetX: spec.X, targetY: spec.Y, nativeView: true, then: func() {
		g.nativeUnitPresent = job
		g.publishNativeUnitPresentFrame()
	}}
	return nil
}
