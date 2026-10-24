package main

import (
	"fmt"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

// READY: fd2_ch30_flow_20261017.json event82_contract。
// 戰場33F78與故事版共用12CEA／22253，名冊保持正式battle.State。
func (g *Game) startNativeBattleStagingPresent(spec *battle.NativeStagingPlacement, then func()) error {
	if g == nil || spec == nil || g.st == nil || g.m == nil || !g.st.HasNativeMapViewState ||
		spec.Slot < 0 || spec.Slot >= len(g.st.Units) || g.st.Units[spec.Slot] == nil ||
		g.nativeUnitPresent != nil || g.focusJob != nil {
		return fmt.Errorf("battle staging: runtime source unavailable")
	}
	shadow := *g
	shadow.hasStoryNativeMapView = false
	camX, camY, view, err := shadow.nativeFocusEndpoint(spec.X, spec.Y)
	if err != nil {
		return err
	}
	candidate, err := cloneNativeUnitPresentState(g.st)
	if err != nil {
		return err
	}
	if err := candidate.MaterializeNativeMapViewState(view); err != nil {
		return err
	}
	shadow.camX, shadow.camY = float64(camX), float64(camY)
	shadow.curX, shadow.curY = spec.X, spec.Y
	input, err := shadow.buildNativeIndexedTransitionInputForState(candidate)
	if err != nil {
		return err
	}
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	if err := indexedmap.ComposeNativeUnitPresentTerrainSnapshot(work, input); err != nil {
		return err
	}
	if err := indexedmap.RedrawNativeUnitPresentObjects(work, input); err != nil {
		return err
	}
	vga := make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.CopyNativeUnitPresentViewport(vga, work); err != nil {
		return err
	}
	present := campaign.NativeUnitPresent{Slot: spec.Slot, NewX: spec.X, NewY: spec.Y, VisualX: spec.X, VisualY: spec.Y}
	job, err := shadow.buildNativeUnitPresentJob(present, candidate, work, vga, then)
	if err != nil {
		return err
	}
	beforeUnit := *g.st.Units[spec.Slot]
	beforeView := g.st.NativeMapViewState
	beforeCamX, beforeCamY, beforeCurX, beforeCurY := g.camX, g.camY, g.curX, g.curY
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	job.mutate = func() bool {
		return g.st != nil && spec.Slot < len(g.st.Units) && g.st.Units[spec.Slot] != nil &&
			g.st.Units[spec.Slot].SetNativeMapCoordinatesRaw(spec.X, spec.Y)
	}
	job.rollback = func() {
		*g.st.Units[spec.Slot] = beforeUnit
		g.st.NativeMapViewState = beforeView
		g.camX, g.camY, g.curX, g.curY = beforeCamX, beforeCamY, beforeCurX, beforeCurY
		g.nativeMapWork, g.nativeMapVGA = beforeWork, beforeVGA
	}
	g.focusJob = &focusUnitJob{targetX: spec.X, targetY: spec.Y, nativeView: true, unitPresent: job, then: func() {
		g.nativeUnitPresent = job
		g.publishNativeUnitPresentFrame()
	}}
	return nil
}

func (g *Game) beginNativeFieldEvent80(actor *battle.Unit) bool {
	if g == nil || g.st == nil || actor == nil {
		return false
	}
	id, ok := battle.NativeFieldEventIDAt(g.st, actor.X, actor.Y, 1)
	if !ok || id != 80 {
		return false
	}
	if _, applied := battle.ApplyNativeFieldModeEvent(g.st, actor, actor.X, actor.Y, 1); !applied {
		g.loadErr = "event80: sourced mode rule unavailable"
	}
	return true
}
