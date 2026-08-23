package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func newNativeCommand29PresentationTestGame(t *testing.T) (*Game, *battle.Unit, []*battle.Unit) {
	t.Helper()
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	figaniPath, fdotherPath := filepath.Join(base, "FIGANI.DAT"), filepath.Join(base, "FDOTHER.DAT")
	bgPath, fdtxtPath, taiPath := filepath.Join(base, "BG.DAT"), filepath.Join(base, "FDTXT.DAT"), filepath.Join(base, "TAI.DAT")
	if !fileExists(figaniPath) || !fileExists(fdotherPath) || !fileExists(bgPath) || !fileExists(fdtxtPath) || !fileExists(taiPath) {
		t.Skip("player-provided FIGANI.DAT/FDOTHER.DAT/BG.DAT/FDTXT.DAT/TAI.DAT unavailable")
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", figaniPath)
	t.Setenv("FD2_ORIGINAL_BG", bgPath)
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_ORIGINAL_FDTXT", fdtxtPath)
	t.Setenv("FD2_ORIGINAL_TAI", taiPath)
	t.Setenv("FD2_MUTE", "1")
	actor := &battle.Unit{Camp: battle.Own, OnField: true, X: 0, Y: 0, Lv: 20, AP: 120, MP: 40, MaxMP: 40, HP: 180, MaxHP: 180, BattleFig: 34, HasBattleFig: true, NativeRecordByte6: 0, HasNativeRecordByte6: true, NativeRecordByte8: 0, HasNativeRecordByte8: true}
	targets := []*battle.Unit{
		{Camp: battle.Enemy, OnField: true, X: 1, Y: 0, Lv: 12, DP: 20, HP: 210, MaxHP: 210, BattleFig: 0, HasBattleFig: true, NativeRecordByte6: 1, HasNativeRecordByte6: true, NativeRecordByte8: 1, HasNativeRecordByte8: true, NativeRecordByte5: 0, HasNativeRecordByte5: true},
		{Camp: battle.Enemy, OnField: true, X: 2, Y: 0, Lv: 13, DP: 25, HP: 220, MaxHP: 220, BattleFig: 3, HasBattleFig: true, NativeRecordByte6: 1, HasNativeRecordByte6: true, NativeRecordByte8: 2, HasNativeRecordByte8: true, NativeRecordByte5: 0, HasNativeRecordByte5: true},
	}
	book := make([]battle.NativeCommandRecord, 36)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[29] = battle.NativeCommandRecord{ID: 29, SelectionMode: 5, EffectMode: 2, MPCost: 26, TargetCode: 0}
	units := []*battle.Unit{actor, targets[0], targets[1]}
	st := &battle.State{W: 3, H: 1, Units: units, NativeCompositionEventBytes: make([]byte, 3), NativeCommandBook: book}
	g := &Game{
		m:  &MapData{W: 3, H: 1, Tiles: []int{0, 0, 0}, NativeTerrainControl: []byte{0, 0, 4, 0}},
		st: st, rng: rand.New(rand.NewSource(29)), nativeUIPalette: loadNativeUIPalette(),
	}
	return g, actor, targets
}

func TestNativeCommand29PresentationPublishesMultipleTargetsInOrder(t *testing.T) {
	g, actor, targets := newNativeCommand29PresentationTestGame(t)
	done := 0
	if err := g.startNativeCommand29Presentation(actor, targets[0], func(results []battle.NativeCommand24Damage) {
		if len(results) != 2 {
			t.Fatalf("callback results=%d want 2", len(results))
		}
		done++
	}); err != nil {
		t.Fatal(err)
	}
	job := g.nativeCmd29Presentation
	if len(job.targets) != 2 || len(job.targets[0].transitionFrames) != 20 || len(job.targets[1].transitionFrames) != 20 {
		t.Fatalf("multi-target presentation was not fully preflighted: %#v", job)
	}
	screen := ebiten.NewImage(640, 400)
	firstHP, secondHP := targets[0].HP, targets[1].HP
	seenFirst, seenSecond := false, false
	for steps := 0; g.nativeCmd29Presentation != nil && steps < 1000; steps++ {
		if !g.drawNativeCommand29Presentation(screen) {
			t.Fatalf("draw failed at step %d", steps)
		}
		g.stepNativeCommand29Presentation()
		if targets[0].HP < firstHP {
			seenFirst = true
		}
		if targets[1].HP < secondHP {
			seenSecond = true
			if !seenFirst {
				t.Fatal("second target published before first target")
			}
		}
		if seenFirst && !seenSecond && actor.MP != 14 {
			t.Fatalf("actor MP was not published exactly once: %d", actor.MP)
		}
	}
	if g.nativeCmd29Presentation != nil || !seenFirst || !seenSecond || !actor.Acted || actor.MP != 14 || done != 1 {
		t.Fatalf("presentation incomplete job=%v first=%v second=%v actor=%#v done=%d", g.nativeCmd29Presentation != nil, seenFirst, seenSecond, actor, done)
	}
}

func TestNativeCommand29PresentationRollsBackAllTargetsOnLateFailure(t *testing.T) {
	g, actor, targets := newNativeCommand29PresentationTestGame(t)
	if err := g.startNativeCommand29Presentation(actor, targets[0], nil); err != nil {
		t.Fatal(err)
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd29Presentation != nil && !g.nativeCmd29Presentation.targets[0].damagePublished && steps < 600; steps++ {
		if !g.drawNativeCommand29Presentation(screen) {
			t.Fatal("draw failed before first target marker")
		}
		g.stepNativeCommand29Presentation()
	}
	if g.nativeCmd29Presentation == nil || !g.nativeCmd29Presentation.targets[0].damagePublished {
		t.Fatal("first target marker was not reached")
	}
	targets[1].HP--
	for steps := 0; g.nativeCmd29Presentation != nil && steps < 600; steps++ {
		if !g.drawNativeCommand29Presentation(screen) {
			t.Fatal("draw failed before rollback boundary")
		}
		g.stepNativeCommand29Presentation()
	}
	if g.nativeCmd29Presentation != nil || actor.MP != 40 || actor.Acted || targets[0].HP != 210 || targets[1].HP != 220 {
		t.Fatalf("late failure was not atomic actor=%#v targets=%#v", actor, targets)
	}
}

func TestNativeCommand29PresentationRejectsUnprovenSelectorWithoutMutation(t *testing.T) {
	g, actor, targets := newNativeCommand29PresentationTestGame(t)
	actor.BattleFig = 33
	if err := g.startNativeCommand29Presentation(actor, targets[0], nil); err == nil {
		t.Fatal("unproven command29 selector was accepted")
	}
	if actor.MP != 40 || actor.Acted || targets[0].HP != 210 || targets[1].HP != 220 || g.nativeCmd29Presentation != nil {
		t.Fatalf("failed preflight mutated state actor=%#v targets=%#v", actor, targets)
	}
}
