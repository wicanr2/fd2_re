package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func newNativeCommand24PresentationTestGame(t *testing.T) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	figaniPath, fdotherPath := filepath.Join(base, "FIGANI.DAT"), filepath.Join(base, "FDOTHER.DAT")
	if !fileExists(figaniPath) || !fileExists(fdotherPath) {
		t.Skip("player-provided FIGANI.DAT/FDOTHER.DAT unavailable")
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", figaniPath)
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_MUTE", "1")
	actor := &battle.Unit{Camp: battle.Own, OnField: true, X: 0, Y: 0, AP: 100, MP: 30, MaxMP: 30, HP: 120, MaxHP: 120, BattleFig: 32, HasBattleFig: true}
	target := &battle.Unit{Camp: battle.Enemy, OnField: true, X: 1, Y: 0, DP: 20, HP: 200, MaxHP: 200, BattleFig: 0, HasBattleFig: true}
	book := make([]battle.NativeCommandRecord, 36)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[24] = battle.NativeCommandRecord{ID: 24, SelectionMode: 5, EffectMode: 1, MPCost: 22, TargetCode: 0}
	st := &battle.State{
		W: 2, H: 1, Units: []*battle.Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
	}
	g := &Game{
		st: st, rng: rand.New(rand.NewSource(2)), nativeUIPalette: loadNativeUIPalette(),
		bg: ebiten.NewImage(320, 100), tai: ebiten.NewImage(1, 1), panel: ebiten.NewImage(149, 42),
	}
	return g, actor, target
}

func TestNativeCommand24PresentationPublishesAtRawMarkers(t *testing.T) {
	g, actor, target := newNativeCommand24PresentationTestGame(t)
	done := 0
	if err := g.startNativeCommand24Presentation(actor, target, func([]battle.NativeCommand24Damage) { done++ }); err != nil {
		t.Fatal(err)
	}
	if actor.MP != 30 || actor.Acted || target.HP != 200 {
		t.Fatalf("start crossed mutation boundary actor=%#v target=%#v", actor, target)
	}
	screen := ebiten.NewImage(640, 400)
	for g.nativeCmd24Presentation != nil && g.nativeCmd24Presentation.frame < 4 {
		if !g.drawNativeCommand24Presentation(screen) {
			t.Fatal("actor prelude draw failed")
		}
		g.stepNativeCommand24Presentation()
		if actor.MP != 30 || actor.Acted || target.HP != 200 {
			t.Fatalf("mutation before actor marker actor=%#v target=%#v", actor, target)
		}
	}
	if !g.drawNativeCommand24Presentation(screen) {
		t.Fatal("actor marker draw failed")
	}
	g.stepNativeCommand24Presentation()
	if actor.MP != 8 || actor.Acted || target.HP != 200 {
		t.Fatalf("actor marker state actor=%#v target=%#v", actor, target)
	}
	for g.nativeCmd24Presentation != nil && g.nativeCmd24Presentation.frame < 10 {
		if !g.drawNativeCommand24Presentation(screen) {
			t.Fatal("target prelude draw failed")
		}
		g.stepNativeCommand24Presentation()
		if target.HP != 200 || actor.Acted {
			t.Fatalf("damage before target marker actor=%#v target=%#v", actor, target)
		}
	}
	if !g.drawNativeCommand24Presentation(screen) {
		t.Fatal("target marker draw failed")
	}
	g.stepNativeCommand24Presentation()
	if target.HP >= 200 || actor.Acted {
		t.Fatalf("target marker state actor=%#v target=%#v", actor, target)
	}
	for steps := 0; g.nativeCmd24Presentation != nil && steps < 128; steps++ {
		if !g.drawNativeCommand24Presentation(screen) {
			t.Fatal("tail draw failed")
		}
		g.stepNativeCommand24Presentation()
	}
	if g.nativeCmd24Presentation != nil || !actor.Acted || done != 1 {
		t.Fatalf("presentation did not complete job=%v actor=%#v done=%d", g.nativeCmd24Presentation != nil, actor, done)
	}
}

func TestNativeCommand24PresentationRejectsUnprovenActorWithoutMutation(t *testing.T) {
	g, actor, target := newNativeCommand24PresentationTestGame(t)
	actor.BattleFig = 4
	if err := g.startNativeCommand24Presentation(actor, target, nil); err == nil {
		t.Fatal("unproven command24 actor selector was accepted")
	}
	if actor.MP != 30 || actor.Acted || target.HP != 200 || g.nativeCmd24Presentation != nil {
		t.Fatalf("failed start mutated state actor=%#v target=%#v", actor, target)
	}
}
