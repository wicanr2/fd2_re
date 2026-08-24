package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestExecuteNativeAICommand17UsesOriginalIndexedTail(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	paths := map[string]string{
		"FD2_ORIGINAL_FIGANI":  filepath.Join(base, "FIGANI.DAT"),
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
	g := &Game{rng: rand.New(rand.NewSource(17)), nativeUIPalette: loadNativeUIPalette()}
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" || len(g.st.Units) < 6 {
		t.Fatalf("fixture err=%q units=%d", g.loadErr, len(g.st.Units))
	}
	actor, target := g.st.Units[4], g.st.Units[5]
	g.st.Units = []*battle.Unit{actor, target}
	actor.SetMapPlacement(0, 0, 3)
	target.SetMapPlacement(1, 0, 3)
	if err := actor.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	if err := target.MaterializeNativeMapPresentation(); err != nil {
		t.Fatal(err)
	}
	actor.Camp, actor.HP, actor.MaxHP, actor.MP, actor.AP, actor.Acted = battle.Enemy, 100, 100, 10, 100, false
	actor.NativeRecordByte5, actor.NativeRecordByte6 = 0, 0
	actor.HasNativeRecordByte5, actor.HasNativeRecordByte6 = true, true
	actor.NativeTransient = [6]byte{}
	target.Camp, target.HP, target.MaxHP, target.AP = battle.Enemy, 100, 100, 200
	target.NativeRecordByte5, target.NativeRecordByte6 = 0, 0
	target.HasNativeRecordByte5, target.HasNativeRecordByte6 = true, true
	target.NativeTransient = [6]byte{}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[17] = battle.NativeCommandRecord{ID: 17, SelectionMode: 1, EffectMode: 1, MPCost: 99, TargetCode: 1}
	book[18] = battle.NativeCommandRecord{ID: 18, SelectionMode: 1, EffectMode: 0, MPCost: 4, TargetCode: 1}
	g.st.NativeCommandBook = book
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: 0, CameraY: 0, CursorX: 1, CursorY: 0, VisibleCursorX: 1, VisibleCursorY: 0}); err != nil {
		t.Fatal(err)
	}
	if !g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatal("HUD/range unavailable")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if len(g.nativeMapWork) != indexedmap.NativeUnitPresentWorkSize || len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		t.Fatal("indexed map buffers unavailable")
	}
	g.nativeRNGState = 0
	continued := 0
	plan := &battle.AIPlan{U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand, NativeCommandID: 17}
	if err := g.executeNativeAIActionWithContinuation(plan, func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.nativeAICommandModifier == nil || actor.AP != 100 || target.AP != 200 || actor.MP != 10 || actor.Acted {
		t.Fatalf("AI modifier crossed first Draw boundary: actor=%#v target=%#v", actor, target)
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeAICommandModifier != nil && steps < 512; steps++ {
		if !g.drawNativeAICommandModifierPresentation(screen) {
			t.Fatalf("draw failed at %d", steps)
		}
		g.stepNativeAICommandModifierPresentation()
		if g.nativeAICommandModifier != nil && g.nativeAICommandModifier.holding {
			g.nativeAICommandModifier.hold = 0
		}
	}
	if g.nativeAICommandModifier != nil || actor.AP != 116 || target.AP != 231 || actor.MP != 6 || !actor.Acted || continued != 1 {
		t.Fatalf("AI modifier owner incomplete: actor=%#v target=%#v continued=%d", actor, target, continued)
	}
}

func TestNativeAICommandModifierLateCancelRollsBack(t *testing.T) {
	actor := completeNativeAIExecutorUnit()
	actor.Camp, actor.MP, actor.AP, actor.NativeRecordByte6 = battle.Enemy, 10, 100, 0
	target := completeNativeAIExecutorUnit()
	target.Camp, target.AP, target.NativeRecordByte6 = battle.Enemy, 200, 0
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[17] = battle.NativeCommandRecord{ID: 17, EffectMode: 1, TargetCode: 1}
	book[18] = battle.NativeCommandRecord{ID: 18, MPCost: 4}
	st := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book}
	actor.NativeMapPresentation.X, target.NativeMapPresentation.X = 0, 1
	plan, err := st.PlanNativeAICommandModifier(actor, 17, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := battle.PublishNativeAICommandModifier(plan); err != nil {
		t.Fatal(err)
	}
	baseWork, baseVGA := make([]byte, indexedmap.NativeUnitPresentWorkSize), make([]byte, indexedmap.NativeMapVGASize)
	g := &Game{st: st, nativeRNGState: plan.Result.RNGState, nativeMapWork: baseWork, nativeMapVGA: baseVGA}
	g.nativeAICommandModifier = &nativeAICommandModifierPresentationJob{actor: actor, plan: plan, rngBefore: 0, baselineWork: append([]byte(nil), baseWork...), baselineVGA: append([]byte(nil), baseVGA...)}
	g.cancelNativeAICommandModifierPresentation()
	if actor.AP != 100 || target.AP != 200 || actor.MP != 10 || actor.Acted || g.nativeRNGState != 0 || g.nativeAICommandModifier != nil {
		t.Fatalf("cancel retained state actor=%#v target=%#v", actor, target)
	}
}
