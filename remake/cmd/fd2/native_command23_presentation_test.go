package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func nativeCommand23TestGame(t *testing.T) (*Game, *battle.Unit, *battle.Unit) {
	t.Helper()
	g := completeNativeUnitPresentGame(t)
	actor, target := g.st.Units[0], g.st.Units[1]
	g.st.Units = g.st.Units[:2]
	for i, unit := range g.st.Units {
		unit.Camp, unit.OnField = battle.Own, true
		unit.NativeIdentity, unit.HasNativeIdentity = 7+i, true
		unit.NativeRecordClass, unit.HasNativeRecordClass = 1, true
		unit.NativeRecordRace, unit.HasNativeRecordRace = 1, true
		unit.NativeRecordByte5, unit.HasNativeRecordByte5 = 0, true
		unit.NativeRecordByte6, unit.HasNativeRecordByte6 = 0, true
		unit.InventorySlots = []int{255, 255, 255, 255, 255, 255, 255, 255}
		unit.NativeInventoryFlags = []int{128, 128, 128, 128, 128, 128, 128, 128}
		if !unit.SetNativeMapCoordinatesRaw(i, 0) {
			t.Fatal("raw map coordinates unavailable")
		}
	}
	actor.MP, actor.MaxMP = 30, 40
	var err error
	g.st.NativeCommandBook, err = battle.LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	g.nativeCommandBook = g.st.NativeCommandBook
	g.nativeCommandPaletteFlash, err = battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
	if err != nil {
		t.Fatal(err)
	}
	g.st.NativeCompositionEventBytes = make([]byte, g.st.W*g.st.H)
	g.st.NativeTerrainMoveCodes = make([]byte, g.st.W*g.st.H)
	for i := range g.st.NativeTerrainMoveCodes {
		g.st.NativeTerrainMoveCodes[i] = 2
	}
	g.nativeMapDAC = append([]byte(nil), g.nativeMapAssets.PaletteDAC...)
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil {
		t.Fatal(err)
	}
	g.sel, g.moved, g.curX, g.curY = actor, true, 0, 0
	g.camX, g.camY = 0, 0
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g, actor, target
}

func TestNativeCommand23FormalConfirmCancelAndFullPresentation(t *testing.T) {
	g, actor, target := nativeCommand23TestGame(t)
	if err := g.beginNativeCommandTargetSelection(23); err != nil {
		t.Fatal(err)
	}
	g.confirm()
	if g.nativeItemRelocating || actor.MP != 30 {
		t.Fatal("spell23 accepted its caster despite inner radius1")
	}
	g.curX, g.curY = target.X, target.Y
	g.confirm()
	if !g.nativeCommand23Relocating || !g.nativeItemRelocating || g.nativeCommand0Targeting || actor.MP != 30 {
		t.Fatalf("spell23 did not enter its independent destination stage: %s", g.msg)
	}
	if !g.cancelNativeItemTargetModal() || !g.nativeCommandOpen || g.itemOpen || g.nativeCommand23Relocating {
		t.Fatal("spell23 destination cancel did not return to command grid")
	}
	if err := g.beginNativeCommandTargetSelection(23); err != nil {
		t.Fatal(err)
	}
	g.confirm()
	g.curX, g.curY = actor.X, actor.Y
	g.confirm()
	if g.nativeModifierPresentation != nil || target.X != 1 || actor.MP != 30 {
		t.Fatal("spell23 accepted occupied destination")
	}
	g.curX, g.curY = 8, 7
	g.confirm()
	if g.nativeModifierPresentation == nil || target.X != 1 || actor.MP != 30 {
		t.Fatalf("spell23 did not preflight complete palette/disappear/appear chain: %s", g.msg)
	}
	screen := ebiten.NewImage(640, 400)
	for i := 0; g.nativeModifierPresentation != nil && i < 20; i++ {
		g.drawNativeCommandModifierPresentation(screen)
		g.stepNativeCommandModifierPresentation()
	}
	stages := 0
	sawOffMap := false
	for i := 0; g.nativeUnitPresent != nil && i < 120; i++ {
		g.drawNativeUnitPresent(screen)
		stepNativeUnitPresentNow(g)
		stages++
		sawOffMap = sawOffMap || target.X == 255 && target.Y == 255
	}
	if g.nativeModifierPresentation != nil || g.nativeUnitPresent != nil || stages != 96 || !sawOffMap || target.X != 8 || target.Y != 7 || actor.MP != 10 || !actor.Acted || g.nativeCommand23Relocating || g.nativeItemRelocating || g.sel != nil || g.loadErr != "" {
		t.Fatalf("spell23 incomplete: stages%d offmap%v MP%d acted%v target(%d,%d) error%s", stages, sawOffMap, actor.MP, actor.Acted, target.X, target.Y, g.loadErr)
	}
}

func TestNativeCommand23InsufficientMPOrMissingArtIsAtomic(t *testing.T) {
	for _, missing := range []bool{false, true} {
		g, actor, target := nativeCommand23TestGame(t)
		if err := g.beginNativeCommandTargetSelection(23); err != nil {
			t.Fatal(err)
		}
		g.curX, g.curY = target.X, target.Y
		g.confirm()
		if missing {
			g.nativeMapAssets.FDOTHER6[0x72].Pixels = nil
		} else {
			actor.MP = 19
		}
		beforeMP := actor.MP
		g.curX, g.curY = 8, 7
		g.confirm()
		if g.nativeModifierPresentation != nil || g.nativeUnitPresent != nil || actor.MP != beforeMP || actor.Acted || target.X != 1 || target.Y != 0 || !g.nativeCommand23Relocating {
			t.Fatal("rejected spell23 changed transaction state")
		}
	}
}
