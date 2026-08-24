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

func nativeCommand34PlanFixture(t *testing.T) (*battle.NativeCompoundCommand34Plan, *battle.Unit, *battle.Unit) {
	t.Helper()
	actor := &battle.Unit{Camp: battle.Own, OnField: true, X: 0, Y: 0, HP: 40, MaxHP: 100, MP: 60, AP: 100, DP: 80, HIT: 60, EV: 50, Lv: 20, BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true}
	target := &battle.Unit{Camp: battle.Own, OnField: true, X: 1, Y: 0, HP: 10, MaxHP: 90, AP: 70, DP: 60, HIT: 50, EV: 40, Lv: 12, BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[34] = battle.NativeCommandRecord{ID: 34, SelectionMode: 5, EffectMode: 3, MPCost: 28, TargetCode: 1}
	st := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := st.PlanNativeCompoundCommand34(actor, target, 7)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestStartNativeCommand34PresentationUsesOriginalAssetsEndToEnd(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	paths := map[string]string{"FD2_ORIGINAL_FIGANI": filepath.Join(base, "FIGANI.DAT"), "FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"), "FD2_ORIGINAL_FDTXT": filepath.Join(base, "FDTXT.DAT"), "FD2_ORIGINAL_BG": filepath.Join(base, "BG.DAT"), "FD2_ORIGINAL_TAI": filepath.Join(base, "TAI.DAT")}
	for key, path := range paths {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	g := &Game{rng: rand.New(rand.NewSource(34)), nativeUIPalette: loadNativeUIPalette()}
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" || len(g.st.Units) < 5 {
		t.Fatalf("fixture err=%q units=%d", g.loadErr, len(g.st.Units))
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
	actor.Camp, actor.MP, actor.AP, actor.DP, actor.HIT, actor.EV = battle.Own, 60, 100, 80, 60, 50
	actor.BattleFig, actor.HasBattleFig, actor.NativeRecordClass, actor.HasNativeRecordClass = 4, true, 19, true
	target.Camp, target.AP, target.DP, target.HIT, target.EV = battle.Own, 70, 60, 50, 40
	target.BattleFig, target.HasBattleFig, target.NativeRecordClass, target.HasNativeRecordClass = 5, true, 2, true
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[34] = battle.NativeCommandRecord{ID: 34, SelectionMode: 5, EffectMode: 3, MPCost: 28, TargetCode: 1}
	g.st.NativeCommandBook, g.st.NativeCompositionEventBytes = book, make([]byte, g.st.W*g.st.H)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 1, CursorY: 0, VisibleCursorX: 1, VisibleCursorY: 0}); err != nil {
		t.Fatal(err)
	}
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("HUD/range unavailable")
	}
	g.curX, g.curY = 1, 0
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = 7
	done := 0
	if err := g.startNativeCommand34Presentation(actor, target, func(result battle.NativeCompoundCommand34Result) {
		if len(result.StageStates) != 3 {
			t.Fatalf("stages=%d", len(result.StageStates))
		}
		done++
	}); err != nil {
		t.Fatal(err)
	}
	if len(g.nativeCmd34Presentation.frames) < 150 || g.nativeCmd34Presentation.publishAt[0] <= 0 {
		t.Fatalf("owner=%#v", g.nativeCmd34Presentation)
	}
	for index := range g.nativeCmd34Presentation.frames {
		g.nativeCmd34Presentation.frames[index].delay = 1
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd34Presentation != nil && steps < 2000; steps++ {
		if !g.drawNativeCommand34Presentation(screen) {
			t.Fatalf("draw failed at %d", steps)
		}
		g.stepNativeCommand34Presentation()
		if g.nativeCmd34Presentation != nil && g.nativeCmd34Presentation.holding {
			g.nativeCmd34Presentation.hold = 0
		}
	}
	if g.nativeCmd34Presentation != nil || !actor.Acted || actor.NativeTransient[0] == 0 || actor.NativeTransient[1] == 0 || actor.NativeTransient[2] == 0 || actor.MP != 60 || done != 1 {
		t.Fatalf("actor=%#v done=%d", actor, done)
	}
}

func TestNativeCommand34PresentationPublishesStagesOnlyAfterDraw(t *testing.T) {
	plan, actor, _ := nativeCommand34PlanFixture(t)
	baseWork, baseVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 7, nativeMapWork: append([]byte(nil), baseWork...), nativeMapVGA: append([]byte(nil), baseVGA...)}
	j := &nativeCommand34PresentationJob{actor: actor, plan: plan, frames: []nativeCompoundPresentedFrame{{image: ebiten.NewImage(1, 1), delay: 1}, {image: ebiten.NewImage(1, 1), delay: 1}}, publishAt: [3]int{1, 9, 10}, stageEnd: [3]int{2, 9, 10}, rngBefore: 7, baselineWork: baseWork, baselineVGA: baseVGA}
	j.stageWork[0], j.stageVGA[0] = append([]byte(nil), baseWork...), append([]byte(nil), baseVGA...)
	g.nativeCmd34Presentation = j
	g.stepNativeCommand34Presentation()
	if actor.NativeTransient[0] != 0 {
		t.Fatal("published without draw")
	}
	j.drawn = true
	g.stepNativeCommand34Presentation()
	if actor.NativeTransient[0] == 0 || actor.NativeTransient[1] != 0 || actor.Acted {
		t.Fatalf("stage boundary actor=%#v", actor)
	}
}

func TestNativeCommand34PresentationLateFailureRollsBackStages(t *testing.T) {
	plan, actor, target := nativeCommand34PlanFixture(t)
	baseWork, baseVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 7, nativeMapWork: baseWork, nativeMapVGA: baseVGA}
	g.nativeCmd34Presentation = &nativeCommand34PresentationJob{actor: actor, plan: plan, rngBefore: 7, baselineWork: append([]byte(nil), baseWork...), baselineVGA: append([]byte(nil), baseVGA...)}
	if err := battle.PublishNativeCompoundCommand34Stage(plan, 0); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = plan.Result.Stages[0].RNGState
	g.failNativeCommand34Presentation(errors.New("late failure"))
	if g.nativeCmd34Presentation != nil || actor.NativeTransient[0] != 0 || target.NativeTransient[0] != 0 || actor.Acted || g.nativeRNGState != 7 || g.loadErr == "" {
		t.Fatalf("rollback actor=%#v target=%#v", actor, target)
	}
}

func TestResetActionOverlayCancelsNativeCommand34Stages(t *testing.T) {
	plan, actor, target := nativeCommand34PlanFixture(t)
	baseWork, baseVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 7, nativeMapWork: baseWork, nativeMapVGA: baseVGA}
	g.nativeCmd34Presentation = &nativeCommand34PresentationJob{actor: actor, plan: plan, rngBefore: 7, baselineWork: append([]byte(nil), baseWork...), baselineVGA: append([]byte(nil), baseVGA...)}
	if err := battle.PublishNativeCompoundCommand34Stage(plan, 0); err != nil {
		t.Fatal(err)
	}
	g.resetActionOverlayLifecycle()
	if g.nativeCmd34Presentation != nil || actor.NativeTransient[0] != 0 || target.NativeTransient[0] != 0 || actor.Acted || g.nativeRNGState != 7 {
		t.Fatalf("overlay reset retained command34 actor=%#v target=%#v", actor, target)
	}
}
