package main

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func newNativeCommand28PresentationTestGame(t *testing.T) (*Game, *battle.Unit, []*battle.Unit) {
	t.Helper()
	g, actor, targets := newNativeCommand29PresentationTestGame(t)
	book, err := battle.LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	g.st.NativeCommandBook = book
	actor.BattleFig, actor.NativeRecordByte8 = 19, 19
	actor.HasNativeRecordByte5 = true
	targets[0].HP, targets[0].MaxHP = 900, 900
	return g, actor, targets
}

func TestNativeCommand28FormalConfirmPresentsEightImpactsAndReturns(t *testing.T) {
	for _, side := range []byte{0, 2} {
		t.Run(fmt.Sprintf("side%d", side), func(t *testing.T) {
			g, actor, targets := newNativeCommand28PresentationTestGame(t)
			actor.NativeRecordByte6 = side
			g.st.HasNativeMapViewState = true
			g.st.NativeTileBlitModes = make([]byte, 3)
			g.sel, g.curX, g.curY = actor, 1, 0
			if err := g.beginNativeCommandTargetSelection(28); err != nil {
				t.Fatal(err)
			}
			g.confirm()
			job := g.nativeCmd29Presentation
			if job == nil || job.command28 == nil || job.plan.CommandID != 28 || len(job.targets) != 1 || len(job.command28.frames) != 49 {
				t.Fatalf("command28 did not enter complete resource59 sequence: %s", g.msg)
			}
			g.stepNativeCommand29Presentation()
			if actor.MP != 40 || actor.Acted || targets[0].HP != 900 {
				t.Fatal("undrawn command28 frame published state")
			}
			mpMarkers, targetMarkers, preludeFrames := 0, 0, 0
			screen := ebiten.NewImage(640, 400)
			for steps := 0; g.nativeCmd29Presentation != nil && steps < 100; steps++ {
				event := job.command28.events[job.command28.index]
				if event.prelude < 9 {
					preludeFrames++
				}
				if !g.drawNativeCommand29Presentation(screen) {
					t.Fatalf("command28 frame%d did not draw", steps)
				}
				g.stepNativeCommand29Presentation()
				if event.mp {
					mpMarkers++
				}
				if event.impact > 0 {
					targetMarkers++
					if event.impact != targetMarkers || targets[0].HP != 900-job.plan.Results[0].Damage*targetMarkers/8 {
						t.Fatal("command28 impact order or cumulative HP differs")
					}
				}
			}
			if g.nativeCmd29Presentation != nil || mpMarkers != 1 || targetMarkers != 8 || preludeFrames != 9 || actor.MP != 18 || !actor.Acted {
				t.Fatalf("command28 incomplete: MP markers%d HP markers%d prelude%d error%s", mpMarkers, targetMarkers, preludeFrames, g.loadErr)
			}
			if g.nativeCommand0Targeting || g.sel != nil || g.st.NativeMapRangeMode != 1 || targets[1].HP != 220 {
				t.Fatal("command28 did not restore player state or changed excluded target")
			}
		})
	}
}

func TestNativeCommand28LateImpactFailureRestoresWholeTransaction(t *testing.T) {
	g, actor, targets := newNativeCommand28PresentationTestGame(t)
	if err := g.startNativeCommand28Presentation(actor, targets[0], nil); err != nil {
		t.Fatal(err)
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd29Presentation != nil && targets[0].HP == 900 && steps < 60; steps++ {
		g.drawNativeCommand29Presentation(screen)
		g.stepNativeCommand29Presentation()
	}
	if targets[0].HP == 900 || g.nativeCmd29Presentation == nil {
		t.Fatal("first command28 impact was not presented")
	}
	targets[0].HP--
	for steps := 0; g.nativeCmd29Presentation != nil && steps < 60; steps++ {
		g.drawNativeCommand29Presentation(screen)
		g.stepNativeCommand29Presentation()
	}
	if g.nativeCmd29Presentation != nil || actor.MP != 40 || actor.Acted || targets[0].HP != 900 || targets[0].NativeRecordByte5 != 0 || g.loadErr == "" {
		t.Fatal("command28 late failure did not restore HP/MP/raw state")
	}
}

func TestNativeCommand28MissingTerrainRejectsBeforeDamageRNG(t *testing.T) {
	g, actor, targets := newNativeCommand28PresentationTestGame(t)
	g.m.NativeTerrainControl = nil
	if err := g.startNativeCommand28Presentation(actor, targets[0], nil); err == nil {
		t.Fatal("missing terrain source accepted")
	}
	if g.nativeCmd29Presentation != nil || actor.MP != 40 || actor.Acted || targets[0].HP != 900 || g.rng.Int63() != rand.New(rand.NewSource(29)).Int63() {
		t.Fatal("command28 rejected source changed state or RNG")
	}
}
