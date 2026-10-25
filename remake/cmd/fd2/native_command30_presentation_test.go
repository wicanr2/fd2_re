package main

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// Controlled two-enemy presentation fixture. Legal acquisition is tested
// separately against the real map16 constructor and native growth tables.
func newNativeCommand30PresentationTestGame(t *testing.T) (*Game, *battle.Unit, []*battle.Unit) {
	t.Helper()
	g, actor, targets := newNativeCommand29PresentationTestGame(t)
	actor.BattleFig, actor.NativeRecordByte8 = 18, 18
	actor.Lv, actor.MP, actor.MaxMP = 4, 29, 29
	actor.NativeRecordByte6 = 2
	actor.HasNativeRecordByte5 = true
	book, err := battle.LoadNativeCommandRecords(assetPath("assets/spells.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.st.NativeCommandBook = book
	g.rng = rand.New(rand.NewSource(30))
	return g, actor, targets
}

func TestNativeCommand30PresentationMatchesStateControlAndDrawMarkers(t *testing.T) {
	g, actor, targets := newNativeCommand30PresentationTestGame(t)
	control, ca, ct := newNativeCommand30PresentationTestGame(t)
	saved, confirmed := battle.Cell{X: 0, Y: 0}, battle.Cell{X: 1, Y: 0}
	want, err := control.st.ExecuteNativeCommand30(ca, saved, confirmed, control.rng)
	if err != nil {
		t.Fatal(err)
	}
	done := 0
	if err := g.startNativeCommand30PresentationAtCursor(actor, saved, confirmed, func(got []battle.NativeCommand24Damage) {
		done++
		if len(got) != len(want) {
			t.Fatal("target count changed")
		}
		for i := range got {
			if got[i].Target != targets[i] || got[i].Amount != want[i].Amount || got[i].Damage != want[i].Damage {
				t.Fatal("state-only control and presentation plan differ")
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	j := g.nativeCmd29Presentation
	if j.plan.CommandID != 30 || len(j.preludeFrames) != 9 || len(j.effectFrames) != 14 || len(j.targets) != 2 || len(j.targets[1].transitionFrames) != 20 || len(j.actorCues) != 2 {
		t.Fatal("incomplete source56 owner")
	}
	if _, ok := j.actorCues[2]; !ok {
		t.Fatal("early non-impact sound cue was lost")
	}
	for i := 0; i < 3; i++ {
		g.stepNativeCommand29Presentation()
	}
	if j.preludeFrame != 0 || actor.MP != 29 || actor.Acted {
		t.Fatal("advanced without Draw")
	}
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	seen := [2]bool{}
	for step := 0; g.nativeCmd29Presentation != nil && step < 1000; step++ {
		prelude, transition, frame, index := j.preludeFrame, j.transitionFrame, j.frame, j.targetIndex
		mp := actor.MP
		hp := [2]int{targets[0].HP, targets[1].HP}
		if !g.drawNativeCommand29Presentation(screen) || !g.drawNativeCommand29Presentation(screen) {
			t.Fatal("Draw failed")
		}
		if actor.MP != mp || targets[0].HP != hp[0] || targets[1].HP != hp[1] {
			t.Fatal("Draw mutated rules")
		}
		g.stepNativeCommand29Presentation()
		if actor.MP != mp && (prelude != 9 || transition >= 0 || frame != 4 || actor.MP != 5) {
			t.Fatal("MP published outside source marker4")
		}
		for i := range targets {
			if targets[i].HP != hp[i] {
				if prelude != 9 || transition >= 0 || frame != 11 || index != i || seen[i] || i == 1 && !seen[0] {
					t.Fatal("damage order/marker11/repeat differs")
				}
				seen[i] = true
			}
		}
	}
	if g.nativeCmd29Presentation != nil || done != 1 || !seen[0] || !seen[1] || !reflect.DeepEqual(*actor, *ca) || !reflect.DeepEqual(*targets[0], *ct[0]) || !reflect.DeepEqual(*targets[1], *ct[1]) || g.rng.Int63() != control.rng.Int63() {
		t.Fatal("final transaction/control/RNG/continuation differs")
	}
}

func TestNativeCommand30PresentationColdRejectAndLateRollback(t *testing.T) {
	for _, field := range []string{"actor-selector", "second-target-record", "second-target-selector", "MP"} {
		t.Run(field, func(t *testing.T) {
			g, actor, targets := newNativeCommand30PresentationTestGame(t)
			switch field {
			case "actor-selector":
				actor.BattleFig = 19
			case "second-target-record":
				targets[1].HasNativeRecordByte8 = false
			case "second-target-selector":
				targets[1].BattleFig = 4096
			case "MP":
				actor.MP = 23
			}
			before := []battle.Unit{*actor, *targets[0], *targets[1]}
			rng := &splitStrikeCountedSource{Source: rand.NewSource(30)}
			g.rng = rand.New(rng)
			if err := startSplitStrikeHD(g, 30, actor, targets[0], nil); err == nil || rng.calls != 0 || g.nativeCmd29Presentation != nil || !reflect.DeepEqual(before, []battle.Unit{*actor, *targets[0], *targets[1]}) {
				t.Fatalf("cold rejection changed state/RNG: %v", err)
			}
		})
	}
	g, actor, targets := newNativeCommand30PresentationTestGame(t)
	actor.AP, targets[0].HP = 1000, 5
	before := []battle.Unit{*actor, *targets[0], *targets[1]}
	done := 0
	if err := startSplitStrikeHD(g, 30, actor, targets[0], func([]battle.NativeCommand24Damage) { done++ }); err != nil {
		t.Fatal(err)
	}
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	for steps := 0; g.nativeCmd29Presentation != nil && !g.nativeCmd29Presentation.targets[0].damagePublished && steps < 600; steps++ {
		g.drawNativeCommand29Presentation(screen)
		g.stepNativeCommand29Presentation()
	}
	if g.nativeCmd29Presentation == nil || !g.nativeCmd29Presentation.targets[0].damagePublished {
		t.Fatal("first marker not reached")
	}
	targets[1].HP--
	for steps := 0; g.nativeCmd29Presentation != nil && steps < 600; steps++ {
		g.drawNativeCommand29Presentation(screen)
		g.stepNativeCommand29Presentation()
	}
	if g.nativeCmd29Presentation != nil || done != 0 || g.loadErr == "" || !reflect.DeepEqual(before, []battle.Unit{*actor, *targets[0], *targets[1]}) {
		t.Fatal("late failure did not restore actor and every target")
	}
}

func TestNativeCommand30FixedSourceScheduleIncludesEarlyCue(t *testing.T) {
	_, _, _ = newNativeCommand30PresentationTestGame(t)
	a, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), 56)
	if err != nil {
		t.Fatal(err)
	}
	s, err := figani.BuildNativeCommandDerivedStrikeSchedule(a, 30)
	if err != nil || s.TargetStart != 9 || s.ActorImpactFrame != 4 || s.TargetImpactFrame != 11 || s.AudioResource != 49 || s.ActorSample != 4 || s.TargetSample != 3 || !reflect.DeepEqual(s.ActorCues, []figani.NativeCommandFrameCue{{Frame: 2, Sample: 2}, {Frame: 4, Sample: 4}}) {
		t.Fatalf("source56 schedule differs: %+v %v", s, err)
	}
}

func TestNativeCommand30PlayerTargetModalSavesCursorAndCancels(t *testing.T) {
	g, actor, targets := newNativeCommand30PresentationTestGame(t)
	g.sel, g.curX, g.curY = actor, 0, 0
	g.st.NativeTileBlitModes = make([]byte, 3)
	before := []battle.Unit{*actor, *targets[0], *targets[1]}
	if err := g.beginNativeCommandTargetSelection(30); err != nil {
		t.Fatal(err)
	}
	if !g.nativeCommand0Targeting || g.nativeCommand30SavedCursor == nil || *g.nativeCommand30SavedCursor != (battle.Cell{X: 0, Y: 0}) {
		t.Fatal("menu did not capture the pre-modal cursor")
	}
	g.curX = 1
	g.cancelNativeCommandTargetSelection()
	if g.nativeCommand0Targeting || !g.nativeCommandOpen || g.nativeCommand30SavedCursor != nil || !reflect.DeepEqual(before, []battle.Unit{*actor, *targets[0], *targets[1]}) {
		t.Fatal("cancel retained cursor or changed transaction")
	}
	// Re-enter from the original actor cell, then move the confirmed cursor.
	// Using the current cursor as the line origin would omit target0.
	g.curX = 0
	if err := g.beginNativeCommandTargetSelection(30); err != nil {
		t.Fatal(err)
	}
	g.curX = 1
	g.confirm()
	j := g.nativeCmd29Presentation
	if j == nil || j.plan.CommandID != 30 || len(j.targets) != 2 || j.targets[0].unit != targets[0] || j.targets[1].unit != targets[1] || !reflect.DeepEqual(before, []battle.Unit{*actor, *targets[0], *targets[1]}) {
		t.Fatalf("formal confirm lost saved origin: %s", g.msg)
	}
}
