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

func nativeCommand35PlanFixture(t *testing.T) (*battle.NativeCompoundCommand35Plan, *battle.Unit, *battle.Unit) {
	t.Helper()
	actor := &battle.Unit{Camp: battle.Own, OnField: true, X: 0, Y: 0, HP: 100, MaxHP: 100, MP: 40, Lv: 20, BattleFig: 4, HasBattleFig: true, NativeRecordClass: 19, HasNativeRecordClass: true}
	target := &battle.Unit{Camp: battle.Own, OnField: true, X: 1, Y: 0, HP: 90, MaxHP: 90, Lv: 12, BattleFig: 5, HasBattleFig: true, NativeRecordClass: 2, HasNativeRecordClass: true}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[35] = battle.NativeCommandRecord{ID: 35, SelectionMode: 5, EffectMode: 3, MPCost: 36, TargetCode: 1}
	st := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	plan, err := st.PlanNativeCompoundCommand35(actor, target, 1)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestStartNativeCommand35PresentationUsesOriginalAssetsEndToEnd(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	paths := map[string]string{"FD2_ORIGINAL_FIGANI": filepath.Join(base, "FIGANI.DAT"), "FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"), "FD2_ORIGINAL_FDTXT": filepath.Join(base, "FDTXT.DAT"), "FD2_ORIGINAL_BG": filepath.Join(base, "BG.DAT"), "FD2_ORIGINAL_TAI": filepath.Join(base, "TAI.DAT")}
	for key, path := range paths {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	g := &Game{rng: rand.New(rand.NewSource(35)), nativeUIPalette: loadNativeUIPalette()}
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
	actor.Camp, actor.HP, actor.MaxHP, actor.MP = battle.Own, 100, 100, 40
	actor.BattleFig, actor.HasBattleFig, actor.NativeRecordClass, actor.HasNativeRecordClass = 4, true, 19, true
	target.Camp, target.HP, target.MaxHP = battle.Own, 90, 90
	target.BattleFig, target.HasBattleFig, target.NativeRecordClass, target.HasNativeRecordClass = 5, true, 2, true
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[35] = battle.NativeCommandRecord{ID: 35, SelectionMode: 5, EffectMode: 3, MPCost: 36, TargetCode: 1}
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
	g.nativeRNGState = 1
	done := 0
	if err := g.startNativeCommand35Presentation(actor, target, func(result battle.NativeCompoundCommand35Result) {
		if len(result.StageStates) != 3 {
			t.Fatalf("stages=%d", len(result.StageStates))
		}
		done++
	}); err != nil {
		t.Fatal(err)
	}
	if len(g.nativeCmd35Presentation.frames) < 150 || g.nativeCmd35Presentation.publishAt[0] <= 0 {
		t.Fatalf("owner=%#v", g.nativeCmd35Presentation)
	}
	for index := range g.nativeCmd35Presentation.frames {
		g.nativeCmd35Presentation.frames[index].delay = 1
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd35Presentation != nil && steps < 2000; steps++ {
		if !g.drawNativeCommand35Presentation(screen) {
			t.Fatalf("draw failed at %d", steps)
		}
		g.stepNativeCommand35Presentation()
		if g.nativeCmd35Presentation != nil && g.nativeCmd35Presentation.holding {
			g.nativeCmd35Presentation.hold = 0
		}
	}
	if g.nativeCmd35Presentation != nil || !actor.Acted || actor.NativeTransient[3] == 0 || actor.NativeTransient[4] == 0 || actor.NativeTransient[5] == 0 || actor.MP != 40 || done != 1 {
		t.Fatalf("actor=%#v done=%d", actor, done)
	}
}

func TestNativeCommand35PresentationPublishesStagesOnlyAfterDraw(t *testing.T) {
	plan, actor, _ := nativeCommand35PlanFixture(t)
	baseWork, baseVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 1, nativeMapWork: append([]byte(nil), baseWork...), nativeMapVGA: append([]byte(nil), baseVGA...)}
	j := &nativeCommand35PresentationJob{actor: actor, plan: plan, frames: []nativeCompoundPresentedFrame{{image: ebiten.NewImage(1, 1), delay: 1}, {image: ebiten.NewImage(1, 1), delay: 1}}, publishAt: [3]int{1, 9, 10}, stageEnd: [3]int{2, 9, 10}, rngBefore: 1, baselineWork: baseWork, baselineVGA: baseVGA}
	j.stageWork[0], j.stageVGA[0] = append([]byte(nil), baseWork...), append([]byte(nil), baseVGA...)
	g.nativeCmd35Presentation = j
	g.stepNativeCommand35Presentation()
	if actor.NativeTransient[3] != 0 {
		t.Fatal("published without draw")
	}
	j.drawn = true
	g.stepNativeCommand35Presentation()
	if actor.NativeTransient[3] == 0 || actor.NativeTransient[5] != 0 || actor.Acted {
		t.Fatalf("stage boundary actor=%#v", actor)
	}
}

func TestNativeCommand35PresentationLateFailureRollsBackStages(t *testing.T) {
	plan, actor, target := nativeCommand35PlanFixture(t)
	baseWork, baseVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 1, nativeMapWork: baseWork, nativeMapVGA: baseVGA}
	g.nativeCmd35Presentation = &nativeCommand35PresentationJob{actor: actor, plan: plan, rngBefore: 1, baselineWork: append([]byte(nil), baseWork...), baselineVGA: append([]byte(nil), baseVGA...)}
	if err := battle.PublishNativeCompoundCommand35Stage(plan, 0); err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = plan.Result.Stages[0].RNGState
	g.failNativeCommand35Presentation(errors.New("late failure"))
	if g.nativeCmd35Presentation != nil || actor.NativeTransient[3] != 0 || target.NativeTransient[3] != 0 || actor.Acted || g.nativeRNGState != 1 || g.loadErr == "" {
		t.Fatalf("rollback actor=%#v target=%#v", actor, target)
	}
}

func TestResetActionOverlayCancelsNativeCommand35Stages(t *testing.T) {
	plan, actor, target := nativeCommand35PlanFixture(t)
	baseWork, baseVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{nativeRNGState: 1, nativeMapWork: baseWork, nativeMapVGA: baseVGA}
	g.nativeCmd35Presentation = &nativeCommand35PresentationJob{actor: actor, plan: plan, rngBefore: 1, baselineWork: append([]byte(nil), baseWork...), baselineVGA: append([]byte(nil), baseVGA...)}
	if err := battle.PublishNativeCompoundCommand35Stage(plan, 0); err != nil {
		t.Fatal(err)
	}
	g.resetActionOverlayLifecycle()
	if g.nativeCmd35Presentation != nil || actor.NativeTransient[3] != 0 || target.NativeTransient[3] != 0 || actor.Acted || g.nativeRNGState != 1 {
		t.Fatalf("overlay reset retained command35 actor=%#v target=%#v", actor, target)
	}
}
