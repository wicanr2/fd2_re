package main

import (
	"math/rand"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestExecuteNativeAICommand20UsesOriginalIndexedTail(t *testing.T) {
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	for key, path := range map[string]string{
		"FD2_ORIGINAL_FIGANI": filepath.Join(base, "FIGANI.DAT"), "FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"),
		"FD2_ORIGINAL_FDTXT": filepath.Join(base, "FDTXT.DAT"), "FD2_ORIGINAL_BG": filepath.Join(base, "BG.DAT"), "FD2_ORIGINAL_TAI": filepath.Join(base, "TAI.DAT"),
	} {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	g := &Game{rng: rand.New(rand.NewSource(20)), nativeUIPalette: loadNativeUIPalette()}
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
	actor.Camp, actor.HP, actor.MaxHP, actor.MP, actor.Acted = battle.Enemy, 100, 100, 10, false
	actor.NativeRecordByte5, actor.NativeRecordByte6, actor.HasNativeRecordByte5, actor.HasNativeRecordByte6 = 0, 0, true, true
	actor.NativeTransient = [6]byte{}
	target.Camp, target.HP, target.MaxHP = battle.Enemy, 50, 100
	target.NativeRecordByte5, target.NativeRecordByte6, target.HasNativeRecordByte5, target.HasNativeRecordByte6 = 0, 0, true, true
	target.NativeTransient = [6]byte{0, 0, 0, 3}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[10] = battle.NativeCommandRecord{ID: 10, Damage: 10}
	book[20] = battle.NativeCommandRecord{ID: 20, SelectionMode: 1, EffectMode: 1, MPCost: 2, TargetCode: 1}
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
	if err := g.executeNativeAIActionWithContinuation(&battle.AIPlan{U: actor, Target: target, NativeActionKind: battle.NativeAIActionCommand, NativeCommandID: 20}, func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.nativeAICommandModifier == nil || target.HP != 50 || target.NativeTransient[3] != 3 || actor.MP != 10 || actor.Acted {
		t.Fatal("transaction crossed first Draw boundary")
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
	if g.nativeAICommandModifier != nil || target.HP <= 50 || target.NativeTransient[3] != 0 || actor.MP != 8 || !actor.Acted || continued != 1 {
		t.Fatalf("owner incomplete actor=%#v target=%#v continued=%d", actor, target, continued)
	}
}

func TestPlayerNativeCommand20UsesPaletteThenOriginalIndexedTail(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	for key, path := range map[string]string{
		"FD2_ORIGINAL_FIGANI": filepath.Join(base, "FIGANI.DAT"), "FD2_ORIGINAL_FDOTHER": filepath.Join(base, "FDOTHER.DAT"),
		"FD2_ORIGINAL_FDTXT": filepath.Join(base, "FDTXT.DAT"), "FD2_ORIGINAL_BG": filepath.Join(base, "BG.DAT"), "FD2_ORIGINAL_TAI": filepath.Join(base, "TAI.DAT"),
	} {
		if !fileExists(path) {
			t.Skip("player-provided original archives unavailable")
		}
		t.Setenv(key, path)
	}
	g := &Game{rng: rand.New(rand.NewSource(20)), nativeUIPalette: loadNativeUIPalette()}
	if err := g.loadMap("assets/maps/map0"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json")
	if g.loadErr != "" || len(g.st.Units) < 6 {
		t.Fatalf("fixture err=%q units=%d", g.loadErr, len(g.st.Units))
	}
	paletteTable, err := battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
	if err != nil {
		t.Fatal(err)
	}
	g.nativeCommandPaletteFlash = paletteTable
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
	actor.Camp, actor.HP, actor.MaxHP, actor.MP, actor.Acted = battle.Own, 100, 100, 10, false
	target.Camp, target.HP, target.MaxHP, target.NativeTransient = battle.Own, 50, 100, [6]byte{0, 0, 0, 3}
	for _, unit := range []*battle.Unit{actor, target} {
		unit.NativeRecordByte5, unit.NativeRecordByte6 = 0, 0
		unit.HasNativeRecordByte5, unit.HasNativeRecordByte6 = true, true
	}
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[10] = battle.NativeCommandRecord{ID: 10, Damage: 10}
	book[20] = battle.NativeCommandRecord{ID: 20, SelectionMode: 1, EffectMode: 0, MPCost: 2, TargetCode: 1}
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
	g.nativeRNGState = 0
	continued := 0
	if err := g.startNativeCommand2022Presentation(actor, target, 20, true, func(*battle.NativeAICommand2022Plan) { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.nativeAICommandModifier == nil || target.HP != 50 || target.NativeTransient[3] != 3 || actor.MP != 10 || actor.Acted {
		t.Fatal("player transaction crossed first Draw boundary")
	}
	if len(g.nativeAICommandModifier.frames) < 8 {
		t.Fatal("player palette phases unavailable")
	}
	screen := ebiten.NewImage(640, 400)
	for step := 0; g.nativeAICommandModifier != nil && step < 512; step++ {
		if !g.drawNativeAICommandModifierPresentation(screen) {
			t.Fatalf("draw failed at %d", step)
		}
		g.stepNativeAICommandModifierPresentation()
		if g.nativeAICommandModifier != nil && g.nativeAICommandModifier.holding {
			g.nativeAICommandModifier.hold = 0
		}
	}
	if target.HP <= 50 || target.NativeTransient[3] != 0 || actor.MP != 8 || !actor.Acted || continued != 1 {
		t.Fatalf("player owner incomplete actor=%#v target=%#v continued=%d", actor, target, continued)
	}
}
