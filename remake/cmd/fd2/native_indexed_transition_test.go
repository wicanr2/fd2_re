package main

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func nativeIndexedTransitionSpecForTest() campaign.HandlerIndexedTransition {
	return campaign.HandlerIndexedTransition{
		TileX: 6, TileY: 6, RadialRadius: 10, RadialRadiusStep: 8,
		StartY: 0, EndY: 192, ClipWidth: 312, ClipHeight: 192,
		Frames: 9, FrameDelayMs: 5, TailDelayMs: 500,
		PaletteRangeStart: 0, PaletteRangeEnd: 255,
		PaletteDeltaStart: 0, PaletteDeltaEnd: 62, PaletteDeltaStep: 2,
		PaletteDelayMs: 4,
	}
}

func TestResolveNativeIndexedTransitionUsesProvenRelativeCursor(t *testing.T) {
	state := &battle.State{W: 20, H: 20}
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 4, CameraY: 5, CursorX: 15, CursorY: 11,
		VisibleCursorX: 11, VisibleCursorY: 6,
	}); err != nil {
		t.Fatal(err)
	}
	g := &Game{st: state}
	spec := nativeIndexedTransitionSpecForTest()
	spec.CursorSource = "native_relative_cursor"
	spec.CursorYOffset = 3
	resolved, err := g.resolveNativeIndexedTransitionSpec(spec, "0x245ce")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.TileX != 11 || resolved.TileY != 9 {
		t.Fatalf("resolved relative cursor=(%d,%d), want (11,9)", resolved.TileX, resolved.TileY)
	}
	if resolved.CursorSource != spec.CursorSource || resolved.CursorYOffset != 3 {
		t.Fatalf("cursor provenance was lost: %#v", resolved)
	}
}

func TestResolveNativeIndexedTransitionUsesLoadCHSceneView(t *testing.T) {
	g := &Game{
		storyNativeMapView: battle.NativeMapViewState{
			CameraX: 14, CameraY: 32, CursorX: 14, CursorY: 32,
			VisibleCursorX: 0, VisibleCursorY: 0,
		},
		hasStoryNativeMapView: true,
	}
	spec := nativeIndexedTransitionSpecForTest()
	spec.CursorSource = "native_relative_cursor"
	spec.CursorYOffset = 5
	resolved, err := g.resolveNativeIndexedTransitionSpec(spec, "0x336e5")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.TileX != 0 || resolved.TileY != 5 {
		t.Fatalf("resolved ch22 scene cursor=(%d,%d), want (0,5)", resolved.TileX, resolved.TileY)
	}
}

func TestResolveNativeIndexedTransitionUsesChapter27TwoAxisOffset(t *testing.T) {
	g := &Game{
		storyNativeMapView:    battle.NativeMapViewState{VisibleCursorX: 2, VisibleCursorY: 3},
		hasStoryNativeMapView: true,
	}
	spec := nativeIndexedTransitionSpecForTest()
	spec.CursorSource = "native_relative_cursor"
	spec.CursorXOffset = 6
	spec.CursorYOffset = 5
	resolved, err := g.resolveNativeIndexedTransitionSpec(spec, "0x33ce2")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.TileX != 8 || resolved.TileY != 8 {
		t.Fatalf("resolved ch27 scene cursor=(%d,%d), want (8,8)", resolved.TileX, resolved.TileY)
	}
}

func TestResolveNativeIndexedTransitionRejectsMissingOrUnprovenCursor(t *testing.T) {
	spec := nativeIndexedTransitionSpecForTest()
	spec.CursorSource = "native_relative_cursor"
	spec.CursorYOffset = 3
	if _, err := (&Game{}).resolveNativeIndexedTransitionSpec(spec, "0x245ce"); err == nil {
		t.Fatal("missing native relative cursor provenance was accepted")
	}
	state := &battle.State{W: 20, H: 20}
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 4, CameraY: 5, CursorX: 15, CursorY: 11,
		VisibleCursorX: 11, VisibleCursorY: 6,
	}); err != nil {
		t.Fatal(err)
	}
	g := &Game{st: state}
	badSource := spec
	badSource.CursorSource = "absolute_cursor"
	if _, err := g.resolveNativeIndexedTransitionSpec(badSource, "0x245ce"); err == nil {
		t.Fatal("unproven cursor source was accepted")
	}
	badOffset := spec
	badOffset.CursorYOffset = 2
	if _, err := g.resolveNativeIndexedTransitionSpec(badOffset, "0x245ce"); err == nil {
		t.Fatal("unproven cursor y offset was accepted")
	}
	ch22 := spec
	ch22.CursorYOffset = 5
	if _, err := g.resolveNativeIndexedTransitionSpec(ch22, "0x336e5"); err != nil {
		t.Fatalf("proven ch22 cursor offset rejected: %v", err)
	}
	if _, err := g.resolveNativeIndexedTransitionSpec(ch22, "0x245ce"); err == nil {
		t.Fatal("ch22 cursor offset was accepted at ch21 call-site")
	}
	ch27 := spec
	ch27.CursorXOffset, ch27.CursorYOffset = 6, 5
	if _, err := g.resolveNativeIndexedTransitionSpec(ch27, "0x33ce2"); err != nil {
		t.Fatalf("proven ch27 cursor offsets rejected: %v", err)
	}
	ch27.CursorXOffset = 5
	if _, err := g.resolveNativeIndexedTransitionSpec(ch27, "0x33ce2"); err == nil {
		t.Fatal("unproven ch27 cursor x offset was accepted")
	}
}

func TestNativeIndexedTransitionRequiresEveryDrawBeforeContinuation(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	for i := range assets.LUTs {
		for p := range assets.LUTs[i] {
			assets.LUTs[i][p] = byte(p)
		}
	}
	actor := *state.Units[0]
	actor.SetMapPlacement(6, 6, 0)
	g := &Game{
		nativeMapAssets: assets, m: field,
		storyActors: []battle.Unit{actor},
	}
	continued := 0
	if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.indexedTransition == nil || g.indexedTransition.frame != 0 {
		t.Fatal("first pass was not atomically precomposed")
	}
	for i := 0; i < 20; i++ {
		g.stepNativeIndexedTransition()
	}
	if g.indexedTransition.frame != 0 || continued != 0 {
		t.Fatal("Update advanced an unacknowledged transition frame")
	}
	for want := 1; want < 9; want++ {
		g.indexedTransition.drawn = true
		g.stepNativeIndexedTransition()
		if g.indexedTransition == nil || g.indexedTransition.frame != want {
			t.Fatalf("pass=%d, want %d", g.indexedTransition.frame, want)
		}
	}
	g.indexedTransition.drawn = true
	g.stepNativeIndexedTransition()
	if g.indexedTransition.phase != nativeTransitionTail || g.indexedTransition.tailTicks != 30 {
		t.Fatalf("tail=%d phase=%d", g.indexedTransition.tailTicks, g.indexedTransition.phase)
	}
	for i := 0; i < 31; i++ {
		g.stepNativeIndexedTransition()
	}
	if g.indexedTransition.phase != nativeTransitionPalette || g.indexedTransition.paletteStep != 0 {
		t.Fatalf("palette did not start after exact tail: phase=%d step=%d", g.indexedTransition.phase, g.indexedTransition.paletteStep)
	}
	pixels := append([]byte(nil), g.indexedTransition.vga...)
	for step := 0; step < 32; step++ {
		if step > 0 && g.indexedTransition.paletteStep != step {
			t.Fatalf("palette step=%d want %d", g.indexedTransition.paletteStep, step)
		}
		g.indexedTransition.drawn = true
		g.stepNativeIndexedTransition()
	}
	if g.indexedTransition != nil || continued != 1 {
		t.Fatalf("completion job=%v continuation=%d", g.indexedTransition, continued)
	}
	if !bytes.Equal(g.nativeMapVGA, pixels) {
		t.Fatal("palette phase changed indexed pixels")
	}
}

func TestNativeIndexedTransitionStartFailsClosedOnMissingRawAssets(t *testing.T) {
	g := &Game{nativeMapWork: []byte{7}, nativeMapVGA: []byte{8}}
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err == nil {
		t.Fatal("missing native source was accepted")
	}
	if g.indexedTransition != nil || !bytes.Equal(g.nativeMapWork, beforeWork) || !bytes.Equal(g.nativeMapVGA, beforeVGA) {
		t.Fatal("rejected transition partially mutated runtime")
	}
}

func TestNativeIndexedTransitionPreflightsAllNineLUTsBeforePublish(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	assets.LUTs[2] = assets.LUTs[2][:255]
	actor := *state.Units[0]
	g := &Game{
		nativeMapAssets: assets, m: field,
		storyActors:   []battle.Unit{actor},
		nativeMapWork: []byte{7}, nativeMapVGA: []byte{8},
	}
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if err := g.startNativeIndexedTransition(nativeIndexedTransitionSpecForTest(), "", nil); err == nil {
		t.Fatal("malformed second-to-last LUT was accepted")
	}
	if g.indexedTransition != nil || !bytes.Equal(g.nativeMapWork, beforeWork) || !bytes.Equal(g.nativeMapVGA, beforeVGA) {
		t.Fatal("failed nine-LUT preflight partially published transition")
	}
}
