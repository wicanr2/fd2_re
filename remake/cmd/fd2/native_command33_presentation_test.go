package main

import (
	"errors"
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func nativeCompound33FixtureForMain() (*battle.State, *battle.Unit, *battle.Unit) {
	actor := &battle.Unit{
		Camp: battle.Own, OnField: true, X: 0, Y: 0, HP: 40, MaxHP: 100, MP: 60, Lv: 20,
		BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true,
		NativeTransient: [6]byte{1, 2, 3, 4, 5, 6},
	}
	target := &battle.Unit{
		Camp: battle.Own, OnField: true, X: 1, Y: 0, HP: 10, MaxHP: 90, Lv: 12,
		BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true,
		NativeTransient: [6]byte{7, 8, 9, 10, 11, 12},
	}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[33] = battle.NativeCommandRecord{ID: 33, SelectionMode: 5, EffectMode: 3, MPCost: 52, TargetCode: 1}
	return &battle.State{
		W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2),
		NativeCommandBook: book,
	}, actor, target
}

func nativeCommand33PresentationPlan(t *testing.T) (*battle.NativeCompoundCommand33Plan, *battle.Unit, *battle.Unit) {
	t.Helper()
	st, actor, target := nativeCompound33FixtureForMain()
	plan, err := st.PlanNativeCompoundCommand33(actor, target, 7)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestStartNativeCommand33PresentationUsesOriginalAssetsEndToEnd(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	paths := map[string]string{
		"FD2_ORIGINAL_FIGANI": filepath.Join(base, "FIGANI.DAT"), "FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"),
		"FD2_ORIGINAL_FDTXT": filepath.Join(base, "FDTXT.DAT"), "FD2_ORIGINAL_BG": filepath.Join(base, "BG.DAT"),
		"FD2_ORIGINAL_TAI": filepath.Join(base, "TAI.DAT"),
	}
	for key, path := range paths {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	g := &Game{rng: rand.New(rand.NewSource(33)), nativeUIPalette: loadNativeUIPalette()}
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" || len(g.st.Units) < 5 {
		t.Fatalf("command33 fixture reset err=%q units=%d", g.loadErr, len(g.st.Units))
	}
	actor, target := g.st.Units[0], g.st.Units[4]
	for index := 1; index < 4; index++ {
		g.st.Units[index].OnField = false
	}
	actor.SetMapPlacement(0, 0, 3)
	target.SetMapPlacement(1, 0, 3)
	if err := actor.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	if err := target.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	actor.Camp, actor.HP, actor.MaxHP, actor.MP = battle.Own, 40, 100, 60
	actor.BattleFig, actor.HasBattleFig, actor.NativeRecordClass, actor.HasNativeRecordClass = 4, true, 19, true
	actor.NativeTransient = [6]byte{1, 2, 3, 4, 5, 6}
	target.Camp, target.HP, target.MaxHP = battle.Own, 10, 90
	target.BattleFig, target.HasBattleFig, target.NativeRecordClass, target.HasNativeRecordClass = 5, true, 2, true
	target.NativeTransient = [6]byte{7, 8, 9, 10, 11, 12}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[33] = battle.NativeCommandRecord{ID: 33, SelectionMode: 5, EffectMode: 3, MPCost: 52, TargetCode: 1}
	g.st.NativeCommandBook = book
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 1, CursorY: 0, VisibleCursorX: 1, VisibleCursorY: 0}); err != nil {
		t.Fatal(err)
	}
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("command33 fixture native HUD/range unavailable")
	}
	g.curX, g.curY = 1, 0
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = 7
	done := 0
	if err := g.startNativeCommand33Presentation(actor, target, func(result battle.NativeCompoundCommand33Result) {
		if len(result.Targets) != 2 {
			t.Fatalf("command33 callback targets=%d", len(result.Targets))
		}
		done++
	}); err != nil {
		t.Fatal(err)
	}
	if len(g.nativeCmd33Presentation.frames) < 100 || g.nativeCmd33Presentation.publishAt <= 0 {
		t.Fatalf("command33 did not preflight complete owner: %#v", g.nativeCmd33Presentation)
	}
	for index := range g.nativeCmd33Presentation.frames {
		g.nativeCmd33Presentation.frames[index].delay = 1
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd33Presentation != nil && steps < 1000; steps++ {
		if !g.drawNativeCommand33Presentation(screen) {
			t.Fatalf("command33 draw failed at %d", steps)
		}
		g.stepNativeCommand33Presentation()
		if g.nativeCmd33Presentation != nil && g.nativeCmd33Presentation.holding {
			g.nativeCmd33Presentation.hold = 0
		}
	}
	if g.nativeCmd33Presentation != nil || !actor.Acted || actor.HP != actor.MaxHP || target.HP != target.MaxHP ||
		actor.NativeTransient[3] != 0 || target.NativeTransient[5] != 0 || actor.MP != 60 || done != 1 {
		t.Fatalf("command33 end-to-end actor=%#v target=%#v done=%d", actor, target, done)
	}
}

func TestNativeCommand33PresentationPublishesOnlyAfterDrawBoundary(t *testing.T) {
	plan, actor, target := nativeCommand33PresentationPlan(t)
	baselineWork := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	baselineVGA := make([]byte, indexedmap.NativeMapVGASize)
	postVGA := append([]byte(nil), baselineVGA...)
	postVGA[0] = 9
	g := &Game{nativeRNGState: 7, nativeMapWork: append([]byte(nil), baselineWork...), nativeMapVGA: append([]byte(nil), baselineVGA...)}
	g.nativeCmd33Presentation = &nativeCommand33PresentationJob{
		actor: actor, plan: plan, frames: []nativeCompoundPresentedFrame{{image: ebiten.NewImage(1, 1), delay: 1}, {image: ebiten.NewImage(1, 1), delay: 1}},
		publishAt: 1, rngBefore: 7, baselineWork: baselineWork, baselineVGA: baselineVGA,
		postTransactionWork: append([]byte(nil), baselineWork...), postVGA: postVGA,
	}
	g.stepNativeCommand33Presentation()
	if actor.HP != 40 || target.HP != 10 || actor.Acted {
		t.Fatal("command33 published without Draw acknowledgement")
	}
	g.nativeCmd33Presentation.drawn = true
	g.stepNativeCommand33Presentation()
	if actor.HP != actor.MaxHP || target.HP != target.MaxHP || actor.Acted || g.nativeRNGState != plan.Result.Restore.RNGState || g.nativeMapVGA[0] != 9 {
		t.Fatalf("command33 publication boundary actor=%#v target=%#v", actor, target)
	}
	g.nativeCmd33Presentation.drawn = true
	g.stepNativeCommand33Presentation()
	if g.nativeCmd33Presentation == nil || !g.nativeCmd33Presentation.holding || actor.Acted {
		t.Fatal("command33 completed before result hold")
	}
	g.nativeCmd33Presentation.hold = 0
	g.stepNativeCommand33Presentation()
	if g.nativeCmd33Presentation != nil || !actor.Acted {
		t.Fatalf("command33 completion actor=%#v", actor)
	}
}

func TestNativeCommand33PresentationFailureRollsBackPublishedState(t *testing.T) {
	plan, actor, target := nativeCommand33PresentationPlan(t)
	baselineWork := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	baselineVGA := make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 7, nativeMapWork: append([]byte(nil), baselineWork...), nativeMapVGA: append([]byte(nil), baselineVGA...)}
	g.nativeCmd33Presentation = &nativeCommand33PresentationJob{actor: actor, plan: plan, rngBefore: 7, baselineWork: baselineWork, baselineVGA: baselineVGA}
	if err := battle.PublishNativeCompoundCommand33(plan); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = plan.Result.Restore.RNGState
	g.failNativeCommand33Presentation(errors.New("late draw failure"))
	if g.nativeCmd33Presentation != nil || actor.HP != 40 || target.HP != 10 || actor.Acted || g.nativeRNGState != 7 || g.loadErr == "" {
		t.Fatalf("command33 rollback actor=%#v target=%#v rng=%#x err=%q", actor, target, g.nativeRNGState, g.loadErr)
	}
}

func TestResetActionOverlayCancelsNativeCommand33Transaction(t *testing.T) {
	plan, actor, target := nativeCommand33PresentationPlan(t)
	g := &Game{nativeRNGState: 7, nativeMapWork: make([]byte, indexedmap.NativeUnitPresentWorkSize), nativeMapVGA: make([]byte, indexedmap.NativeMapVGASize)}
	g.nativeCmd33Presentation = &nativeCommand33PresentationJob{actor: actor, plan: plan, rngBefore: 7, baselineWork: append([]byte(nil), g.nativeMapWork...), baselineVGA: append([]byte(nil), g.nativeMapVGA...)}
	if err := battle.PublishNativeCompoundCommand33(plan); err != nil {
		t.Fatal(err)
	}
	g.resetActionOverlayLifecycle()
	if g.nativeCmd33Presentation != nil || actor.HP != 40 || target.HP != 10 || actor.Acted {
		t.Fatalf("overlay reset retained command33 actor=%#v target=%#v", actor, target)
	}
}
