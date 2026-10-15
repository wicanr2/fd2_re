package main

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func completeNative24B4DGame(t *testing.T) *Game {
	t.Helper()
	g := completeNative2189AGame(t)
	// 0x24B4D asks 0x11EEE for a ninth terrain row. Extend the compact steady
	// fixture by one complete native row without inventing any normalized data.
	g.m.H = 9
	g.m.Tiles = append(g.m.Tiles, make([]int, g.m.W)...)
	for i := 0; i < g.m.W; i++ {
		g.m.NativeTileBlitModes = append(g.m.NativeTileBlitModes, 0xff)
	}
	g.st.H = 9
	g.st.NativeTileBlitModes = append(g.st.NativeTileBlitModes, g.m.NativeTileBlitModes[len(g.st.NativeTileBlitModes):]...)
	return g
}

func TestNative24B4DPublishesThirtyAlternatingFramesBeforeContinuation(t *testing.T) {
	g := completeNative24B4DGame(t)
	continued := 0
	if err := g.startNative24B4D(30, 20, func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.transitionReveal == nil || g.transitionReveal.remaining != 30 || g.transitionReveal.index != 0 || continued != 0 {
		t.Fatalf("job=%+v continued=%d", g.transitionReveal, continued)
	}
	frame0 := append([]byte(nil), g.transitionReveal.frames[0]...)
	frame1 := append([]byte(nil), g.transitionReveal.frames[1]...)
	for step := 0; g.transitionReveal != nil && step < 30; step++ {
		job := g.transitionReveal
		job.drawn = true
		job.ticks = 0
		g.stepTransitionReveal()
		if g.transitionReveal != nil {
			want := (step + 1) & 1
			if g.transitionReveal.index != want {
				t.Fatalf("step%d index=%d want%d", step, g.transitionReveal.index, want)
			}
			wantFrame := frame0
			if want == 1 {
				wantFrame = frame1
			}
			if !bytes.Equal(g.nativeMapVGA, wantFrame) {
				t.Fatalf("step%d did not publish row-shifted frame%d", step, want)
			}
		}
	}
	if g.transitionReveal != nil || continued != 1 {
		t.Fatalf("job=%v continued=%d", g.transitionReveal, continued)
	}
}

func TestNative24B4DFailsBeforeMutationWithoutNinthTerrainRow(t *testing.T) {
	g := completeNative2189AGame(t)
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if err := g.startNative24B4D(30, 20, nil); err == nil {
		t.Fatal("eight-row map unexpectedly satisfied 13x9 staging")
	}
	if g.transitionReveal != nil || len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize ||
		!bytes.Equal(beforeWork, g.nativeMapWork) || !bytes.Equal(beforeVGA, g.nativeMapVGA) {
		t.Fatal("failed 0x24B4D preflight changed indexed state")
	}
}

func TestNative24B4DStoryPresentationKeepsBattleStateAbsent(t *testing.T) {
	for _, missingAssets := range []bool{false, true} {
		name := "complete"
		if missingAssets {
			name = "missing-assets"
		}
		t.Run(name, func(t *testing.T) {
			g := completeNative24B4DGame(t)
			g.storyNativeMapSource = g.st
			g.storyNativeMapView = g.st.NativeMapViewState
			g.hasStoryNativeMapView = true
			for _, unit := range g.st.Units {
				g.storyActors = append(g.storyActors, *unit)
			}
			g.st = nil
			if missingAssets {
				g.nativeMapAssets = nil
			}
			continued := 0
			err := g.startNative24B4D(20, 20, func() { continued++ })
			if g.st != nil {
				t.Fatal("story presentation published an active battle state")
			}
			if missingAssets {
				if err == nil || g.transitionReveal != nil || continued != 0 {
					t.Fatalf("missing assets: err=%v job=%v continued=%d", err, g.transitionReveal, continued)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for step := 0; g.transitionReveal != nil && step < 20; step++ {
				g.transitionReveal.drawn = true
				g.transitionReveal.ticks = 0
				g.stepTransitionReveal()
			}
			if g.st != nil || g.transitionReveal != nil || continued != 1 {
				t.Fatalf("story completion: battle=%v job=%v continued=%d", g.st, g.transitionReveal, continued)
			}
		})
	}
}
