package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func storySpawnCandidate(t *testing.T, g *Game) []battle.Unit {
	t.Helper()
	actors := append([]battle.Unit(nil), g.storyActors...)
	unit := actors[0]
	unit.OnField = true
	unit.HP = 1
	if !unit.SetMapPlacement(int(g.camX)/24+3, int(g.camY)/24+3, unit.Dir) {
		t.Fatal("fixture placement")
	}
	return append(actors, unit)
}

func TestNativeStorySpawnMapStateFeedsHDMotion(t *testing.T) {
	g := storyHDTestGame(t)
	before := storyHDNativeSnapshot(g)
	actors := storySpawnCandidate(t, g)
	// Reproduce the old publication: only the actor array grows. The real
	// compositor must reject this before any replacement frame is published.
	stale := *g
	stale.storyActors, stale.camX = actors, g.camX+0.5
	if _, err := stale.prepareModernHDStoryMapFrame(g.modernStoryPortraits); err == nil ||
		!strings.Contains(err.Error(), "visual roster provenance mismatch") {
		t.Fatal("old spawn publication did not reproduce the observed refusal", err)
	}
	state, err := g.prepareNativeStoryMapSpawnState(actors, len(g.storyActors))
	if err != nil {
		t.Fatal(err)
	}
	if state == g.storyNativeMapState || state.NativeMapSelectorCache == g.storyNativeMapState.NativeMapSelectorCache ||
		len(state.Units) != len(actors) || !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
		t.Fatal("spawn preparation changed the live carrier or missed the new roster")
	}
	for i, unit := range state.Units {
		if unit != &actors[i] {
			t.Fatal("candidate carrier points to an old actor array", i)
		}
	}
	view := *g
	view.storyActors, view.storyNativeMapState, view.camX = actors, state, g.camX+0.5
	frame, err := view.prepareModernHDStoryMapFrame(g.modernStoryPortraits)
	if err != nil || frame == nil || len(frame.Units) < 2 {
		t.Fatalf("synchronized roster did not reach the native snapshot and HD motion: frame=%+v actors=%+v err=%v", frame, actors, err)
	}
	if !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
		t.Fatal("private spawn rendering changed original actors or timing")
	}
}

func TestNativeStorySpawnMapStateRejectsWithoutLiveMutation(t *testing.T) {
	for _, failure := range []string{"boundary", "cache", "key", "coordinate"} {
		t.Run(failure, func(t *testing.T) {
			g := storyHDTestGame(t)
			before := storyHDNativeSnapshot(g)
			actors := storySpawnCandidate(t, g)
			view, oldCount := *g, len(g.storyActors)
			state := *g.storyNativeMapState
			view.storyNativeMapState = &state
			switch failure {
			case "boundary":
				oldCount--
			case "cache":
				state.NativeMapSelectorCache = nil
			case "key":
				actors[len(actors)-1].HasMapSelectorKey = false
			case "coordinate":
				actors[len(actors)-1].X = -1
			}
			if _, err := view.prepareNativeStoryMapSpawnState(actors, oldCount); err == nil {
				t.Fatal("invalid spawn source accepted")
			}
			if !reflect.DeepEqual(before, storyHDNativeSnapshot(g)) {
				t.Fatal("failed preparation changed original actors, selector cache or timing")
			}
		})
	}
}
