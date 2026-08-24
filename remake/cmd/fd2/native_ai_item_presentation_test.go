package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestExecuteNativeAIItemRestoreUsesOriginalIndexedTail(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	paths := map[string]string{
		"FD2_ORIGINAL_FIGANI":  filepath.Join(base, "FIGANI.DAT"),
		"FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"),
		"FD2_ORIGINAL_FDTXT":   filepath.Join(base, "FDTXT.DAT"),
		"FD2_ORIGINAL_BG":      filepath.Join(base, "BG.DAT"),
	}
	for key, path := range paths {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	g := &Game{rng: rand.New(rand.NewSource(13)), nativeUIPalette: loadNativeUIPalette()}
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
	actor.Camp, actor.HP, actor.MaxHP, actor.Acted = battle.Enemy, 80, 80, false
	actor.NativeRecordByte5, actor.NativeRecordByte6 = 0, 1
	actor.HasNativeRecordByte5, actor.HasNativeRecordByte6 = true, true
	actor.InventorySlots = []int{192, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	actor.NativeInventoryFlags = []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	target.Camp, target.HP, target.MaxHP = battle.Own, 1, 80
	target.NativeRecordByte5, target.NativeRecordByte6 = 0, 0
	target.HasNativeRecordByte5, target.HasNativeRecordByte6 = true, true
	target.InventorySlots = []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	target.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
	rows, err := battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	g.nativeItemEffectRows = rows
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 0, CameraY: 0, CursorX: 1, CursorY: 0, VisibleCursorX: 1, VisibleCursorY: 0,
	}); err != nil {
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
	g.nativeRNGState = 7
	continued := 0
	plan := &battle.AIPlan{
		U: actor, Target: target, NativeActionKind: battle.NativeAIActionItem,
		NativeItemSlot: 0, NativeItemID: 192, NativeItemTargetIndices: []byte{1},
	}
	if err := g.executeNativeAIActionWithContinuation(plan, func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.nativeAIItemPresentation == nil || target.HP != 1 || actor.InventorySlots[0] != 192 || actor.Acted {
		t.Fatalf("AI item crossed first Draw boundary: actor=%#v target=%#v", actor, target)
	}
	for index := range g.nativeAIItemPresentation.frames {
		g.nativeAIItemPresentation.frames[index].delay = 1
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeAIItemPresentation != nil && steps < 512; steps++ {
		if !g.drawNativeAIItemPresentation(screen) {
			t.Fatalf("draw failed at %d", steps)
		}
		g.stepNativeAIItemPresentation()
		if g.nativeAIItemPresentation != nil && g.nativeAIItemPresentation.holding {
			g.nativeAIItemPresentation.hold = 0
		}
	}
	if g.nativeAIItemPresentation != nil || target.HP <= 1 || target.HP > target.MaxHP ||
		actor.InventorySlots[0] != 0xff || actor.NativeInventoryFlags[0]&0x80 == 0 ||
		!actor.Acted || continued != 1 {
		t.Fatalf("AI item indexed owner incomplete: actor=%#v target=%#v continued=%d", actor, target, continued)
	}
}

func TestNativeAIItemPresentationLateCancelRollsBack(t *testing.T) {
	actor := completeNativeAIExecutorUnit()
	target := completeNativeAIExecutorUnit()
	actor.InventorySlots[0], actor.NativeInventoryFlags[0] = 1, 0x40
	target.HP = 10
	st := &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}}
	rows := make([]byte, 2*battle.NativeItemEffectRowSize)
	rows[battle.NativeItemEffectRowSize+0x0d], rows[battle.NativeItemEffectRowSize+0x0e] = 13, 30
	g := &Game{st: st, nativeItemEffectRows: rows, nativeRNGState: 7}
	plan := &battle.AIPlan{U: actor, Target: target, NativeActionKind: battle.NativeAIActionItem,
		NativeItemSlot: 0, NativeItemID: 1, NativeItemTargetIndices: []byte{1}}
	tx, err := g.planNativeAITargetItem(plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := g.commitNativeAIItemTransaction(tx); err != nil {
		t.Fatal(err)
	}
	g.nativeAIItemPresentation = &nativeAIItemPresentationJob{
		transaction: tx, published: true,
		baselineWork: make([]byte, indexedmap.NativeUnitPresentWorkSize),
		baselineVGA:  make([]byte, indexedmap.NativeMapVGASize),
	}
	g.nativeMapWork = make([]byte, indexedmap.NativeUnitPresentWorkSize)
	g.nativeMapVGA = make([]byte, indexedmap.NativeMapVGASize)
	g.cancelNativeAIItemPresentation()
	if target.HP != 10 || g.nativeRNGState != 7 || actor.InventorySlots[0] != 1 {
		t.Fatalf("late cancel did not roll back: hp=%d rng=%#x slot=%d", target.HP, g.nativeRNGState, actor.InventorySlots[0])
	}
}

func TestNativeAIItemPresentationRosterChangeCancelsWithoutPanic(t *testing.T) {
	g := &Game{
		st: &battle.State{Units: []*battle.Unit{completeNativeAIExecutorUnit()}},
		nativeAIItemPresentation: &nativeAIItemPresentationJob{
			transaction:  &nativeAIItemTransaction{before: make([]byte, 160), rngBefore: 7},
			published:    true,
			baselineWork: make([]byte, indexedmap.NativeUnitPresentWorkSize),
			baselineVGA:  make([]byte, indexedmap.NativeMapVGASize),
		},
		nativeMapWork: make([]byte, indexedmap.NativeUnitPresentWorkSize),
		nativeMapVGA:  make([]byte, indexedmap.NativeMapVGASize),
	}
	g.cancelNativeAIItemPresentation()
	if g.nativeAIItemPresentation != nil || g.loadErr == "" {
		t.Fatalf("roster-change cancel job=%v err=%q", g.nativeAIItemPresentation, g.loadErr)
	}
}
