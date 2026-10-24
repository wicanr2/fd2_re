package main

import (
	"bytes"
	"encoding/json"
	"image"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func storyHDTestGame(t *testing.T) *Game {
	t.Helper()
	g, _ := unitPresentHDTestGame(t, false, true, true)
	source := g.st
	g.storyActors = make([]battle.Unit, len(source.Units))
	for i, unit := range source.Units {
		g.storyActors[i] = *unit
	}
	g.storyNativeMapView, g.hasStoryNativeMapView = source.NativeMapViewState, true
	g.storyBG, g.st = true, nil
	if err := g.materializeNativeStoryMapState(source); err != nil {
		t.Fatal(err)
	}
	return g
}

func storyHDNativeSnapshot(g *Game) any {
	// Detach the render carrier's public values. Gameplay callbacks and maps
	// keyed by grid cells are not JSON data and are not compositor inputs.
	s := g.storyNativeMapState
	raw, err := json.Marshal([]any{g.storyActors, s.W, s.H, s.NativeMapViewState, s.HasNativeMapViewState,
		s.NativeMapCycleState, s.HasNativeMapCycleState, s.NativeTerrainPhaseState, s.HasNativeTerrainPhaseState,
		s.NativeTerrainFlipState, s.NativeUnitPixelShiftState, s.HasNativeMapBinaryTimingState,
		s.NativeMapHUDState, s.HasNativeMapHUDState, s.NativeMapRangeMode, s.HasNativeMapRangeModeState,
		s.NativeTileBlitModes, s.NativeCompositionEventBytes, g.storyNativeMapView,
		g.camX, g.camY, g.curX, g.curY, g.nativeRNGState,
		g.nativeFDOTHERPalettePhase, g.nativeFDOTHERPaletteTick})
	if err != nil {
		panic(err)
	}
	return []any{string(raw), g.nativeMapClock, g.storyNativeMapState.NativeMapSelectorCache.Clone(),
		append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...), append([]byte(nil), g.nativeMapDAC...)}
}

func TestModernHDStoryMapPrivateComposition(t *testing.T) {
	g := storyHDTestGame(t)
	// Keep the already documented approximation private as well.
	g.storyActors[0].HasNativeRecordRace, g.storyActors[0].HasNativeRecordClass = false, false
	before := storyHDNativeSnapshot(g)
	previous := g.modernHDMapFrame
	frame, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	if frame == nil || len(frame.Units) == 0 || frame.HUDUnit != nil || frame.Range != nil || previous != g.modernHDMapFrame {
		t.Fatal("private story frame absent, contains battle UI or was prematurely published")
	}
	if !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) || g.storyActors[0].HasNativeRecordRace || g.storyActors[0].HasNativeRecordClass {
		t.Fatal("HD preparation changed native story source or approximation provenance")
	}
	first := healHDCPUWorld(frame, g.modernStoryPortraits)
	again, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil || !bytes.Equal(first.Pix, healHDCPUWorld(again, g.modernStoryPortraits).Pix) || !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
		t.Fatal("repeated private preparation advanced the scene", err)
	}
	viewport := frame.Viewport
	if viewport.Empty() {
		viewport = modernHDMapViewport()
	}
	if viewport != modernHDMapViewport() {
		t.Fatal("story world lost the native inset viewport")
	}
}

func TestModernHDStoryMapAdmission(t *testing.T) {
	g := storyHDTestGame(t)
	if !g.modernHDMapFrameAdmission(true, false) {
		t.Fatal("complete native story source was not admitted")
	}
	for _, missing := range []string{"story", "view", "carrier", "theme", "walking", "camera"} {
		bad := *g
		switch missing {
		case "story":
			bad.storyBG = false
		case "view":
			bad.hasStoryNativeMapView = false
		case "carrier":
			bad.storyNativeMapState = nil
		case "theme":
			bad.modernStoryPortraits = nil
		case "walking":
			bad.storyWalks = []*storyWalkJob{{actor: 0}}
		case "camera":
			bad.camX++
		}
		if bad.modernHDMapFrameAdmission(true, false) {
			t.Fatal("unsupported story source admitted:", missing)
		}
	}
	if (*Game)(nil).modernHDMapFrameAdmission(true, false) {
		t.Fatal("nil game admitted")
	}
}

func TestModernHDStoryMapRejectsBeforePublication(t *testing.T) {
	for _, failure := range []string{"selector", "world", "image", "size", "native"} {
		g := storyHDTestGame(t)
		frame, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
		if err != nil {
			t.Fatal(err)
		}
		unit := frame.Units[0]
		switch failure {
		case "selector":
			g.storyNativeMapState.NativeMapSelectorCache = nil
		case "world":
			delete(g.modernStoryPortraits.mapSpritesHD, unit.Group)
		case "image":
			g.modernStoryPortraits.mapSpritesHD[unit.Group][unit.Frame] = nil
		case "size":
			g.modernStoryPortraits.mapSpritesHD[unit.Group][unit.Frame] = image.NewRGBA(image.Rect(0, 0, 143, 144))
		case "native":
			g.storyNativeMapState.NativeTileBlitModes = g.storyNativeMapState.NativeTileBlitModes[:1]
		}
		// A missing selector cannot be cloned by the snapshot helper.
		cache := g.storyNativeMapState.NativeMapSelectorCache
		var before any
		if cache != nil {
			before = storyHDNativeSnapshot(g)
		}
		previous, theme := g.modernHDMapFrame, g.modernStoryPortraits
		if _, err := g.prepareModernHDStoryMapFrame(theme); err == nil {
			t.Fatal("invalid source accepted:", failure)
		}
		if previous != g.modernHDMapFrame || theme != g.modernStoryPortraits || cache != g.storyNativeMapState.NativeMapSelectorCache ||
			before != nil && !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
			t.Fatal("rejected source changed live state:", failure)
		}
	}
}

func TestModernHDStoryMapThemeSwitchAtomic(t *testing.T) {
	g := storyHDTestGame(t)
	frame, err := g.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil {
		t.Fatal(err)
	}
	g.modernHDMapFrame = frame
	set := *g.modernStoryPortraits
	set.mapSpritesHD = make(map[int][]image.Image)
	for group, frames := range g.modernStoryPortraits.mapSpritesHD {
		set.mapSpritesHD[group] = append([]image.Image(nil), frames...)
	}
	unit := frame.Units[0]
	set.mapSpritesHD[unit.Group][unit.Frame] = nil
	before := storyHDNativeSnapshot(g)
	previous := g.modernStoryPortraits
	if err := g.applyTheme(&set); err == nil {
		t.Fatal("theme switch accepted a missing story HD unit")
	}
	if g.modernStoryPortraits != previous || g.modernHDMapFrame != frame || !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
		t.Fatal("failed story theme switch published resources or native state")
	}
}
