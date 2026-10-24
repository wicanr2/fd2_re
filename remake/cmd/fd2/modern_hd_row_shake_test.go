package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func rowShakeHDTestGame(t *testing.T, story, modern bool) *Game {
	t.Helper()
	g := indexedHDTestGame(t, modern)
	if !story {
		g.st, g.storyBG = g.storyNativeMapState, false
		g.st.MaterializeNativeMapHUDState(1, 1, 1)
		v := g.st.NativeMapViewState
		v.CursorX, v.CursorY = 6, 5
		g.curX, g.curY = 6, 5
		v.VisibleCursorX, v.VisibleCursorY = 6-v.CameraX, 5-v.CameraY
		if err := g.st.MaterializeNativeMapViewState(v); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

func rowShakeNativeState(g *Game) any {
	view := *g
	if g.st != nil {
		view.storyNativeMapState = g.st
		view.storyActors = make([]battle.Unit, len(g.st.Units))
		for i, u := range g.st.Units {
			view.storyActors[i] = *u
		}
	}
	var state any
	if view.storyNativeMapState != nil {
		state = storyHDNativeSnapshot(&view)
	} else {
		raw, err := json.Marshal(g.storyActors)
		if err != nil {
			panic(err)
		}
		state = []any{string(raw), g.nativeMapClock, append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...), append([]byte(nil), g.nativeMapDAC...)}
	}
	j := g.transitionReveal
	if j == nil {
		return []any{state, nil}
	}
	raw, err := json.Marshal([]any{j.work, j.frames, j.palette, j.index, j.remaining, j.delay, j.ticks})
	if err != nil {
		panic(err)
	}
	return []any{state, string(raw)}
}

func TestModernHDRowShakeNativeControlAndCadence(t *testing.T) {
	for _, story := range []bool{false, true} {
		for _, count := range []int{20, 30} {
			g, control := rowShakeHDTestGame(t, story, true), rowShakeHDTestGame(t, story, false)
			a, b := 0, 0
			if err := g.startNative24B4D(count, 20, func() { a++ }); err != nil {
				t.Fatal(err)
			}
			if err := control.startNative24B4D(count, 20, func() { b++ }); err != nil {
				t.Fatal(err)
			}
			if len(g.transitionReveal.hdFrames) != 2 || !reflect.DeepEqual(rowShakeNativeState(g), rowShakeNativeState(control)) {
				t.Fatal("HD changed native producer")
			}
			for n := 0; n < count; n++ {
				before := rowShakeNativeState(g)
				g.stepTransitionReveal()
				if !reflect.DeepEqual(before, rowShakeNativeState(g)) {
					t.Fatal("advanced without Draw")
				}
				for _, game := range []*Game{g, control} {
					game.transitionReveal.drawn = true
					game.stepTransitionReveal()
				}
				if !reflect.DeepEqual(rowShakeNativeState(g), rowShakeNativeState(control)) {
					t.Fatal("native control diverged", n)
				}
			}
			g.stepTransitionReveal()
			if a != 1 || b != 1 || g.transitionReveal != nil || story && g.st != nil {
				t.Fatal("wrong completion boundary")
			}
		}
	}
}

func TestModernHDRowShakeLayersAndIncomingRow(t *testing.T) {
	g := rowShakeHDTestGame(t, false, true)
	// Explicit synthetic foreground carrier: tile zero has the same 0x88
	// control as hdForegroundPlanFixture, with a complete twelve-frame bank.
	g.nativeMapAssets.Terrain = nativeFrameTestBank(12, 1)
	g.nativeMapAssets.Controls[0], g.m.NativeTerrainControl[0] = 0x88, 0x88
	for _, u := range g.st.Units {
		u.NativeRecordRace, u.NativeRecordClass = 0, 0
	}
	if err := g.startNative24B4D(30, 20, nil); err != nil {
		t.Fatal(err)
	}
	j := g.transitionReveal
	a, b := j.hdFrames[0], j.hdFrames[1]
	if len(a.Units) == 0 || len(a.Foreground) == 0 || a.HUDUnit == nil || b.HUDUnit == nil {
		t.Fatalf("fixture must contain units=%d, foreground=%d and HD HUD icon=%v/%v", len(a.Units), len(a.Foreground), a.HUDUnit, b.HUDUnit)
	}
	for i, u := range a.Units {
		if b.Units[i].Rect != u.Rect.Add(image.Pt(0, -6)) {
			t.Fatal("unit missed row displacement")
		}
	}
	for i, f := range a.Foreground {
		if b.Foreground[i].Rect != f.Rect.Add(image.Pt(0, -6)) {
			t.Fatal("foreground missed row displacement")
		}
	}
	if b.HUDUnit.Unit.Rect != a.HUDUnit.Unit.Rect.Add(image.Pt(0, -6)) {
		t.Fatal("HUD icon missed row displacement")
	}
	for y := 4; y < 195; y++ {
		for x := 4; x < 316; x++ {
			if b.Background.RGBAAt(x, y) != a.Background.RGBAAt(x, y+1) || b.UI.RGBAAt(x, y) != a.UI.RGBAAt(x, y+1) {
				t.Fatal("background or HUD did not use source row", x, y)
			}
		}
	}
	for x := 4; x < 316; x++ {
		want := j.palette[j.work[0x8088+192*456+x-4]]
		if b.Background.At(x, 195) != want { // At returns RGBA, so compare channels below.
			r, gc, bc, ac := b.Background.At(x, 195).RGBA()
			wr, wg, wb, wa := want.RGBA()
			if r != wr || gc != wg || bc != wb || ac != wa {
				t.Fatal("lost incoming ninth terrain row")
			}
		}
	}
	before := rowShakeNativeState(g)
	if _, err := g.prepareModernHDRowShake(j, g.modernStoryPortraits); err != nil || !reflect.DeepEqual(before, rowShakeNativeState(g)) {
		t.Fatal("preparation mutated native source", err)
	}
}

func TestModernHDRowShakeRejectsSourceAndThemeAtomically(t *testing.T) {
	for _, kind := range []string{"world", "image", "size", "work", "row1", "seed", "hud", "palette", "bank"} {
		g := rowShakeHDTestGame(t, false, true)
		set := g.modernStoryPortraits
		g.modernStoryPortraits = nil
		if err := g.startNative24B4D(30, 20, nil); err != nil {
			t.Fatal(err)
		}
		j := g.transitionReveal
		unit := j.hdSource.Input.Frame.Units[0]
		group, err := j.hdSource.Input.Frame.SelectorCache.KeyForSlot(unit.Slot)
		if err != nil {
			t.Fatal(err)
		}
		switch kind {
		case "world":
			delete(set.mapSpritesHD, group)
		case "image":
			for i := range set.mapSpritesHD[group] {
				set.mapSpritesHD[group][i] = nil
			}
		case "size":
			for i := range set.mapSpritesHD[group] {
				set.mapSpritesHD[group][i] = image.NewRGBA(image.Rect(0, 0, 24, 24))
			}
		case "work":
			j.work[0]++
		case "row1":
			j.frames[1][100]++
		case "seed":
			j.hdSource.Seed[0]++
		case "hud":
			j.hdSource.Input.HUD.AnchorX++
		case "palette":
			j.palette[1] = j.palette[2]
		case "bank":
			j.hdSource.Input.Frame.UnitBank = g.nativeMapAssets.Terrain
		}
		before := rowShakeNativeState(g)
		if err := g.applyTheme(set); err == nil || g.modernStoryPortraits != nil || j.hdFrames != nil || j.hdTheme != nil || !reflect.DeepEqual(before, rowShakeNativeState(g)) {
			t.Fatal("failed F2 published or advanced", kind, err)
		}
	}
}

func TestModernHDRowShakeStoryColdFailureRestoresInitialization(t *testing.T) {
	for _, fresh := range []bool{false, true} {
		g := rowShakeHDTestGame(t, true, true)
		if fresh {
			g.storyNativeMapSource = g.storyNativeMapState
			g.storyNativeMapState = nil
			for i := range g.storyActors {
				g.storyActors[i].HasNativeMapPresentation = false
				g.storyActors[i].HasMapSelectorSlot = false
			}
		}
		// Native initialization succeeds, then HD rejects the partial world.
		delete(g.modernStoryPortraits.mapSpritesHD, 0)
		before := rowShakeNativeState(g)
		carrier := g.storyNativeMapState
		if err := g.startNative24B4D(20, 20, nil); err == nil || g.transitionReveal != nil || g.st != nil || carrier != g.storyNativeMapState || !reflect.DeepEqual(before, rowShakeNativeState(g)) {
			t.Fatal("cold failure leaked story initialization", fresh, err)
		}
	}
}

func TestModernHDRowShakeStagesAllOwnersBeforePublish(t *testing.T) {
	g := rowShakeHDTestGame(t, true, false)
	if err := g.startNative24B4D(20, 20, nil); err != nil {
		t.Fatal(err)
	}
	set := modernHDMapRuntimeFixture(t).modernStoryPortraits
	g.storyNativeMapState.NativeTileBlitModes = g.storyNativeMapState.NativeTileBlitModes[:1]
	j := g.transitionReveal
	before := rowShakeNativeState(g)
	if err := g.applyTheme(set); err == nil || g.modernStoryPortraits != nil || j.hdFrames != nil || !reflect.DeepEqual(before, rowShakeNativeState(g)) {
		t.Fatal("later owner failure published row shake", err)
	}
}

func TestModernHDRowShakeRuntimeFailureRestoresBuffers(t *testing.T) {
	g := rowShakeHDTestGame(t, false, true)
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	continued := 0
	if err := g.startNative24B4D(30, 20, func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	g.failNative24B4D(errors.New("controlled draw failure"))
	if g.transitionReveal != nil || continued != 0 || !bytes.Equal(work, g.nativeMapWork) || !bytes.Equal(vga, g.nativeMapVGA) {
		t.Fatal("runtime failure lost existing rollback")
	}
}
