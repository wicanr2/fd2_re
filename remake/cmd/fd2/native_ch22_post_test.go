package main

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func completeNative2189AGame(t *testing.T) *Game {
	t.Helper()
	assets, field, state := completeNativeMapFrameFixture(t)
	batch := make([]*battle.Unit, 10)
	for i := range batch {
		clone := *state.Units[0]
		clone.X, clone.Y = i+1, 1
		clone.MapSelectorSlot, clone.HasMapSelectorSlot = 0, false
		clone.NativeMapPresentation = battle.NativeMapPresentationState{}
		clone.HasNativeMapPresentation = false
		batch[i] = &clone
	}
	if err := state.AppendNativeMapSelectorBatch(batch); err != nil {
		t.Fatal(err)
	}
	if len(state.Units) != 11 || !state.Units[10].HasNativeMapPresentation {
		t.Fatalf("slot10 native presentation was not materialized: slots=%d", len(state.Units))
	}
	for table := range assets.LUTs {
		for value := range assets.LUTs[table] {
			assets.LUTs[table][value] = byte((value + table + 1) & 0xff)
		}
	}
	g := &Game{nativeMapAssets: assets, m: field, st: state}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g
}

func recoveredNative2189ALoop() campaign.Native2189ALoop {
	return campaign.Native2189ALoop{
		Slot: 10, InitialRadius: 15, RadiusStep: 1, Repeat: 10,
		WorkOffset: 0x8088, WorkStride: 456, ClipWidth: 312, ClipHeight: 192,
	}
}

func TestNative2189APrecomputesTenFramesAndContinuesAfterPresentation(t *testing.T) {
	g := completeNative2189AGame(t)
	continued := 0
	if err := g.startNative2189A(recoveredNative2189ALoop(), func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.native2189A == nil || len(g.native2189A.vgaFrames) != 11 || continued != 0 {
		t.Fatalf("job=%+v continued=%d", g.native2189A, continued)
	}
	first := append([]byte(nil), g.nativeMapVGA...)
	lastEffect := append([]byte(nil), g.native2189A.vgaFrames[9]...)
	steady := append([]byte(nil), g.native2189A.vgaFrames[10]...)
	if bytes.Equal(first, lastEffect) {
		t.Fatal("recovered radius/LUT schedule produced no distinct indexed frame")
	}
	for steps := 0; g.native2189A != nil && steps < 11; steps++ {
		g.native2189A.drawn = true
		g.stepNative2189A()
	}
	if g.native2189A != nil || continued != 1 {
		t.Fatalf("job=%v continued=%d", g.native2189A, continued)
	}
	if len(first) != indexedmap.NativeMapVGASize || !bytes.Equal(steady, g.nativeMapVGA) {
		t.Fatal("ten-pass job did not finish on the steady 0x11CAC frame")
	}
}

func TestNative2189AMissingLUTFailsBeforeMutation(t *testing.T) {
	g := completeNative2189AGame(t)
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	g.nativeMapAssets.LUTs[0] = nil
	if err := g.startNative2189A(recoveredNative2189ALoop(), nil); err == nil {
		t.Fatal("missing LUT0 was accepted")
	}
	if g.native2189A != nil || !bytes.Equal(beforeWork, g.nativeMapWork) || !bytes.Equal(beforeVGA, g.nativeMapVGA) {
		t.Fatal("failed preflight changed indexed state")
	}
}

func TestNative2189AAcceptsRecoveredCommand11CallShape(t *testing.T) {
	g := completeNative2189AGame(t)
	loop := recoveredNative2189ALoop()
	loop.RadiusStep = 10
	if err := g.startNative2189A(loop, nil); err != nil {
		t.Fatal(err)
	}
	if got := len(g.native2189A.vgaFrames); got != 11 {
		t.Fatalf("command11 frames=%d want 11", got)
	}
}

func TestNative2189ARejectsUnrecoveredCallShape(t *testing.T) {
	g := completeNative2189AGame(t)
	loop := recoveredNative2189ALoop()
	loop.RadiusStep = 9
	if err := g.startNative2189A(loop, nil); err == nil {
		t.Fatal("unrecovered radius step was accepted")
	}
}
