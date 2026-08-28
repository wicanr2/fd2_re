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

func nativeCommand32PresentationPlan(t *testing.T) (*battle.NativeCompoundCommand32Plan, *battle.Unit, *battle.Unit) {
	t.Helper()
	st, actor, target := nativeCompound32FixtureForMain()
	plan, err := st.PlanNativeCompoundCommand32(actor, target, 7)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestStartNativeCommand32PresentationUsesOriginalAssetsEndToEnd(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	paths := map[string]string{
		"FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"),
		"FD2_ORIGINAL_FDTXT":   filepath.Join(base, "FDTXT.DAT"),
		"FD2_ORIGINAL_BG":      filepath.Join(base, "BG.DAT"),
		"FD2_ORIGINAL_TAI":     filepath.Join(base, "TAI.DAT"),
	}
	for key, path := range paths {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing.dat"))
	g := &Game{rng: rand.New(rand.NewSource(32)), nativeUIPalette: loadNativeUIPalette()}
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" || len(g.st.Units) < 5 {
		t.Fatalf("command32 fixture reset err=%q units=%d", g.loadErr, len(g.st.Units))
	}
	actor, target := g.st.Units[0], g.st.Units[4]
	actor.SetMapPlacement(0, 0, 3)
	target.SetMapPlacement(1, 0, 1)
	if err := actor.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	if err := target.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	actor.Camp, actor.HP, actor.MaxHP, actor.MP = battle.Own, 100, 100, 80
	actor.ClassID, actor.BattleFig, actor.HasBattleFig = 5, 4, true
	actor.NativeRecordClass, actor.HasNativeRecordClass = 19, true
	target.Camp, target.HP, target.MaxHP = battle.Enemy, 100, 100
	target.ClassID, target.BattleFig, target.HasBattleFig = 5, 5, true
	target.NativeRecordClass, target.HasNativeRecordClass = 2, true
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[32] = battle.NativeCommandRecord{ID: 32, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 76, TargetCode: 0}
	g.st.NativeCommandBook = book
	g.st.NativeCommandResistances = map[int]int{5: 10}
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 1, CursorY: 0, VisibleCursorX: 1, VisibleCursorY: 0}); err != nil {
		t.Fatal(err)
	}
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("command32 fixture native HUD/range unavailable")
	}
	g.curX, g.curY = 1, 0
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = 7
	done := 0
	if err := g.startNativeCommand32Presentation(actor, target, func(result battle.NativeCompoundCommand32Result) {
		if len(result.Targets) != 1 {
			t.Fatalf("command32 callback targets=%d", len(result.Targets))
		}
		done++
	}); err != nil {
		t.Fatal(err)
	}
	if len(g.nativeCmd32Presentation.frames) < 100 || g.nativeCmd32Presentation.publishAt <= 0 {
		t.Fatalf("command32 did not preflight complete owner: %#v", g.nativeCmd32Presentation)
	}
	for index := range g.nativeCmd32Presentation.frames {
		g.nativeCmd32Presentation.frames[index].delay = 1
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd32Presentation != nil && steps < 1000; steps++ {
		if !g.drawNativeCommand32Presentation(screen) {
			t.Fatalf("command32 draw failed at %d", steps)
		}
		g.stepNativeCommand32Presentation()
		if g.nativeCmd32Presentation != nil && g.nativeCmd32Presentation.holding {
			g.nativeCmd32Presentation.hold = 0
		}
	}
	if g.nativeCmd32Presentation != nil || !actor.Acted || target.HP >= 100 || actor.MP != 80 || done != 1 {
		t.Fatalf("command32 end-to-end actor=%#v target=%#v done=%d", actor, target, done)
	}
}

func nativeCompound32FixtureForMain() (*battle.State, *battle.Unit, *battle.Unit) {
	actor := &battle.Unit{
		Camp: battle.Own, OnField: true, X: 0, Y: 0, HP: 100, MaxHP: 100, MP: 80, Lv: 20,
		ClassID: 5, BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true,
	}
	target := &battle.Unit{
		Camp: battle.Enemy, OnField: true, X: 1, Y: 0, HP: 100, MaxHP: 100, Lv: 12,
		ClassID: 5, BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true,
	}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[32] = battle.NativeCommandRecord{ID: 32, Damage: 50, Hit: 100, SelectionMode: 1, EffectMode: 0, MPCost: 76, TargetCode: 0}
	return &battle.State{
		W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2),
		NativeCommandBook: book, NativeCommandResistances: map[int]int{5: 10},
	}, actor, target
}

func TestNativeCommand32PresentationPublishesOnlyAfterDrawBoundary(t *testing.T) {
	plan, actor, target := nativeCommand32PresentationPlan(t)
	callback := 0
	baselineWork := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	baselineVGA := make([]byte, indexedmap.NativeMapVGASize)
	postWork := append([]byte(nil), baselineWork...)
	postVGA := append([]byte(nil), baselineVGA...)
	postVGA[0] = 9
	g := &Game{nativeRNGState: 7, nativeMapWork: append([]byte(nil), baselineWork...), nativeMapVGA: append([]byte(nil), baselineVGA...)}
	g.nativeCmd32Presentation = &nativeCommand32PresentationJob{
		actor: actor, plan: plan,
		frames:    []nativeCompoundPresentedFrame{{image: ebiten.NewImage(1, 1), delay: 1}, {image: ebiten.NewImage(1, 1), delay: 1}},
		publishAt: 1, rngBefore: 7, baselineWork: baselineWork, baselineVGA: baselineVGA,
		postTransactionWork: postWork, postVGA: postVGA,
		then: func(battle.NativeCompoundCommand32Result) { callback++ },
	}
	g.stepNativeCommand32Presentation()
	if target.HP != 100 || actor.Acted || g.nativeRNGState != 7 {
		t.Fatal("command32 published without Draw acknowledgement")
	}
	g.nativeCmd32Presentation.drawn = true
	g.stepNativeCommand32Presentation()
	if target.HP != plan.Result.Targets[0].HPAfter || actor.Acted || g.nativeRNGState != plan.Result.RNGState || g.nativeMapVGA[0] != 9 {
		t.Fatalf("command32 damage boundary mismatch actor=%#v target=%#v rng=%#x", actor, target, g.nativeRNGState)
	}
	g.nativeCmd32Presentation.drawn = true
	g.stepNativeCommand32Presentation()
	if g.nativeCmd32Presentation == nil || !g.nativeCmd32Presentation.holding || actor.Acted {
		t.Fatal("command32 completed before result hold")
	}
	g.nativeCmd32Presentation.hold = 0
	g.stepNativeCommand32Presentation()
	if g.nativeCmd32Presentation != nil || !actor.Acted || callback != 1 {
		t.Fatalf("command32 completion actor=%#v callback=%d", actor, callback)
	}
}

func TestNativeCommand32PresentationFailureRollsBackPublishedState(t *testing.T) {
	plan, actor, target := nativeCommand32PresentationPlan(t)
	baselineWork := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	baselineVGA := make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 7, nativeMapWork: append([]byte(nil), baselineWork...), nativeMapVGA: append([]byte(nil), baselineVGA...)}
	g.nativeCmd32Presentation = &nativeCommand32PresentationJob{
		actor: actor, plan: plan, frames: []nativeCompoundPresentedFrame{{image: ebiten.NewImage(1, 1), delay: 1}},
		publishAt: 0, published: true, rngBefore: 7, baselineWork: baselineWork, baselineVGA: baselineVGA,
	}
	if err := battle.ApplyNativeCompoundCommand32Target(plan, 0); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = plan.Result.RNGState
	g.failNativeCommand32Presentation(errors.New("late draw failure"))
	if g.nativeCmd32Presentation != nil || target.HP != 100 || actor.Acted || g.nativeRNGState != 7 || g.loadErr == "" {
		t.Fatalf("command32 rollback actor=%#v target=%#v rng=%#x err=%q", actor, target, g.nativeRNGState, g.loadErr)
	}
}

func TestResetActionOverlayCancelsNativeCommand32Transaction(t *testing.T) {
	plan, actor, target := nativeCommand32PresentationPlan(t)
	baselineWork := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	baselineVGA := make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 7, nativeMapWork: append([]byte(nil), baselineWork...), nativeMapVGA: append([]byte(nil), baselineVGA...)}
	g.nativeCmd32Presentation = &nativeCommand32PresentationJob{
		actor: actor, plan: plan, rngBefore: 7, baselineWork: baselineWork, baselineVGA: baselineVGA,
	}
	if err := battle.ApplyNativeCompoundCommand32Target(plan, 0); err != nil {
		t.Fatal(err)
	}
	g.resetActionOverlayLifecycle()
	if g.nativeCmd32Presentation != nil || target.HP != 100 || actor.Acted {
		t.Fatalf("overlay reset retained command32 actor=%#v target=%#v", actor, target)
	}
}
