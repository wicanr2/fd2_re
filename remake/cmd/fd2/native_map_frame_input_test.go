package main

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func nativeFrameTestSprite(pixel byte) fdicon.Sprite {
	pixels, mask := make([]byte, 24*24), make([]byte, 24*24)
	for i := range pixels {
		pixels[i], mask[i] = pixel, 1
	}
	return fdicon.Sprite{Pixels: pixels, Mask: mask, RemapMask: make([]byte, 24*24)}
}

func nativeFrameTestBank(count int, pixel byte) *fdicon.Bank {
	bank := &fdicon.Bank{Sprites: make([]fdicon.Sprite, count)}
	for i := range bank.Sprites {
		bank.Sprites[i] = nativeFrameTestSprite(pixel)
	}
	return bank
}

func nativeFrameTestHUDFrame(width, height int, pixel byte) fdother.Frame {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint16(raw, uint16(width))
	binary.LittleEndian.PutUint16(raw[2:], uint16(height))
	for row := 0; row < height; row++ {
		for remaining := width; remaining > 0; {
			count := remaining
			if count > 64 {
				count = 64
			}
			raw = append(raw, byte(count-1), pixel)
			remaining -= count
		}
	}
	return fdother.Frame{Width: width, Height: height, Pixels: raw}
}

func nativeFrameTestHUDFrames() indexedmap.NativeMapHUDFrames {
	frames := indexedmap.NativeMapHUDFrames{
		Panel:        nativeFrameTestHUDFrame(69, 34, 0x5a),
		PositiveSign: nativeFrameTestHUDFrame(6, 7, 0x31),
		NegativeSign: nativeFrameTestHUDFrame(6, 5, 0x42),
	}
	for digit := range frames.Digits {
		frames.Digits[digit] = nativeFrameTestHUDFrame(6, 8, byte(0x50+digit))
		frames.HPMismatchDigits[digit] = nativeFrameTestHUDFrame(6, 8, byte(0x70+digit))
	}
	frames.Digits[1] = nativeFrameTestHUDFrame(5, 8, 0x51)
	frames.HPMismatchDigits[1] = nativeFrameTestHUDFrame(5, 8, 0x71)
	frames.HPEqualOverflow = nativeFrameTestHUDFrame(18, 8, 0x7a)
	frames.HPMismatchOverflow = nativeFrameTestHUDFrame(18, 8, 0x7b)
	return frames
}

func completeNativeMapFrameFixture(t *testing.T) (*nativeMapAssets, *MapData, *battle.State) {
	t.Helper()
	controls := []byte{0, 0, 0, 0}
	luts := make([][]byte, 10)
	for i := range luts {
		luts[i] = make([]byte, 256)
	}
	assets := &nativeMapAssets{
		Terrain: nativeFrameTestBank(2, 1), Range: nativeFrameTestBank(20, 2), Units: nativeFrameTestBank(96, 3),
		Controls: controls, LUTs: luts, Palette: make(color.Palette, 256), PaletteDAC: make([]byte, 256*3), Frames: nativeFrameTestHUDFrames(),
		CommandHealDigits: make([]fdother.LMI1Entry, fdother.NativeCommandHealTailDigitBias+10),
		FDOTHER6:          make([]fdother.LMI1Entry, 0x7d),
	}
	for i := range assets.CommandHealDigits {
		assets.CommandHealDigits[i] = fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{byte(i)}}
	}
	for i := range assets.FDOTHER6 {
		assets.FDOTHER6[i] = fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{byte(i)}}
	}
	for i := range assets.Palette {
		assets.Palette[i] = color.RGBA{byte(i), byte(i), byte(i), 0xff}
	}
	field := &MapData{
		W: 13, H: 8, TileW: 24, TileH: 24, Tiles: make([]int, 13*8),
		NativeTileBlitModes:  make([]byte, 13*8),
		NativeTerrainControl: append([]byte(nil), controls...),
	}
	for i := range field.NativeTileBlitModes {
		field.NativeTileBlitModes[i] = 0xff
	}
	unit := &battle.Unit{
		X: 0, Y: 0, MapSelectorKey: 7, HasMapSelectorKey: true,
		NativeRecordByte5: 0, HasNativeRecordByte5: true,
		BattleFig: 9, HasBattleFig: true,
		NativeRecordRace: 3, HasNativeRecordRace: true,
		NativeRecordClass: 4, HasNativeRecordClass: true,
		HP: 5, NativeRecordWord42: 10, HasNativeRecordWord42: true,
		NativeRecordByte6: 0, HasNativeRecordByte6: true,
	}
	state := &battle.State{}
	if err := state.AppendNativeMapSelectorBatch([]*battle.Unit{unit}); err != nil {
		t.Fatal(err)
	}
	if !state.MaterializeNativeMapHUDState(2, 3, 1) {
		t.Fatal("HUD state materialization rejected")
	}
	state.W, state.H = field.W, field.H
	state.NativeTileBlitModes = append([]byte(nil), field.NativeTileBlitModes...)
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil {
		t.Fatal(err)
	}
	if !state.MaterializeNativeMapRangeMode(1) {
		t.Fatal("range mode materialization rejected")
	}
	return assets, field, state
}

func TestDrawNativeMapFramePresentsCorrectedVGABorder(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	g := &Game{
		nativeMapAssets: assets,
		m:               field,
		st:              state,
	}
	screen := ebiten.NewImage(640, 400)
	if !g.drawNativeMapFrame(screen) {
		t.Fatal("complete materialized native map frame was not presented")
	}
	if len(g.nativeMapVGA) != indexedmap.NativeMapVGASize {
		t.Fatalf("VGA bytes=%d", len(g.nativeMapVGA))
	}
	if g.nativeMapVGA[0] != 0 || g.nativeMapVGA[4*320+3] != 0 {
		t.Fatal("native four-pixel border was overwritten")
	}
	if got := g.nativeMapVGA[4*320+4]; got != 3 {
		t.Fatalf("first viewport pixel=%d, want unit layer 3", got)
	}
}

func TestComposeNativeMapFrameAdmitsOpeningAndInteractiveSelectors(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	for i := range assets.Range.Sprites {
		assets.Range.Sprites[i] = nativeFrameTestSprite(byte(0x20 + i))
	}
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CursorX: 4, CursorY: 4, VisibleCursorX: 4, VisibleCursorY: 4,
	}); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		nativeMapAssets: assets,
		m:               field,
		st:              state,
	}
	var steady []byte
	for mode := 1; mode <= 5; mode++ {
		if !state.MaterializeNativeMapRangeMode(mode) {
			t.Fatalf("selector %d rejected by raw state", mode)
		}
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatalf("drawable selector %d: %v", mode, err)
		}
		if mode == 1 {
			steady = append([]byte(nil), g.nativeMapVGA...)
		} else if bytes.Equal(steady, g.nativeMapVGA) {
			t.Fatalf("dynamic selector %d did not change the indexed frame", mode)
		}
	}
	if !state.MaterializeNativeMapRangeMode(0) {
		t.Fatal("opening selector 0 rejected by raw state")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatalf("opening selector 0: %v", err)
	}
	for _, mode := range []int{6, 11} {
		if !state.MaterializeNativeMapRangeMode(mode) {
			t.Fatalf("selector %d rejected by raw state", mode)
		}
		if err := g.composeNativeMapFrame(); err == nil {
			t.Fatalf("non-drawable production selector %d was admitted", mode)
		}
	}
}

func TestNativeItemTargetEntryOwnsIndexedMapFrameAndFailsClosed(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	for i := range assets.Range.Sprites {
		assets.Range.Sprites[i] = nativeFrameTestSprite(byte(0x20 + i))
	}
	g := &Game{
		nativeMapAssets: assets,
		m:               field,
		st:              state,
		sel:             state.Units[0],
		itemOpen:        true,
	}
	state.NativeCompositionEventBytes = make([]byte, state.W*state.H)
	var err error
	g.nativeItemEffectRows, err = battle.LoadNativeItemEffectRowPrefix("../../assets/data/native_item_effect_rows.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := g.composeNativeMapFrameAt(time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	steady := append([]byte(nil), g.nativeMapVGA...)
	started, err := g.beginNativeTargetItem(0, 192)
	if err != nil || !started {
		t.Fatalf("item target entry started=%v err=%v", started, err)
	}
	if !g.nativeMapFrameAdmission(false, true) {
		t.Fatal("item target entry was not admitted to the indexed map owner")
	}
	if err := g.composeNativeMapFrameAt(time.Unix(0, int64(time.Second/60))); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(steady, g.nativeMapVGA) {
		t.Fatal("item target field did not change the indexed map frame")
	}

	before := append([]byte(nil), g.nativeMapVGA...)
	assets.Range = nil
	if err := g.composeNativeMapFrameAt(time.Unix(0, int64(2*time.Second/60))); err == nil {
		t.Fatal("missing range sprites admitted an item target frame")
	}
	if !bytes.Equal(before, g.nativeMapVGA) {
		t.Fatal("failed item target frame partially published pixels")
	}
}

func TestComposeNativeMapFrameAdvancesTimingOnceAndFailsAtomically(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	g := &Game{nativeMapAssets: assets, m: field, st: state}
	start := time.Unix(500, 0)
	if err := g.composeNativeMapFrameAt(start); err != nil {
		t.Fatal(err)
	}
	if got := state.NativeMapCycleState; got.Idle != 0 || got.Moving != 1 ||
		got.LastTimerTick != 0 {
		t.Fatalf("first compositor timing=%+v", got)
	}
	if err := g.composeNativeMapFrameAt(start.Add(5 * nativeBIOSTickPeriod)); err != nil {
		t.Fatal(err)
	}
	if got := state.NativeMapCycleState; got.Idle != 1 || got.Moving != 2 ||
		got.LastTimerTick != 5 {
		t.Fatalf("second compositor timing=%+v", got)
	}

	if !state.MaterializeNativeMapRangeMode(6) {
		t.Fatal("raw selector 6 rejected by State")
	}
	beforeCycles := state.NativeMapCycleState
	beforeTerrain := state.NativeTerrainPhaseState
	beforeFlip := state.NativeTerrainFlipState
	beforeShift := state.NativeUnitPixelShiftState
	beforeClock := g.nativeMapClock
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	if err := g.composeNativeMapFrameAt(start.Add(10 * nativeBIOSTickPeriod)); err == nil {
		t.Fatal("non-drawable selector was accepted")
	}
	if state.NativeMapCycleState != beforeCycles ||
		state.NativeTerrainPhaseState != beforeTerrain ||
		state.NativeTerrainFlipState != beforeFlip ||
		state.NativeUnitPixelShiftState != beforeShift ||
		g.nativeMapClock != beforeClock || !bytes.Equal(g.nativeMapVGA, beforeVGA) {
		t.Fatal("failed compositor transaction changed timing or pixels")
	}
}

func TestPlayerChapterOneNativeFrameAdmission(t *testing.T) {
	fdotherPath, err := filepath.Abs("../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT unavailable")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", "assets/scenarios/campaign_full.json")
	t.Setenv("FD2_CAMP_NODE", "battle_ch01")
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.camp == nil || g.camp.Cur != "battle_ch01" {
		t.Fatalf("campaign node=%v", g.camp)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		for i, unit := range g.st.Units {
			t.Logf("unit %d fig=%d hp=%d on=%v presentation=%v slot=%v byte5=%#x/%v battleFig=%d/%v race=%d/%v class=%d/%v",
				i, unit.Fig, unit.HP, unit.OnField,
				unit.HasNativeMapPresentation, unit.HasMapSelectorSlot,
				unit.NativeRecordByte5, unit.HasNativeRecordByte5,
				unit.BattleFig, unit.HasBattleFig,
				unit.NativeRecordRace, unit.HasNativeRecordRace,
				unit.NativeRecordClass, unit.HasNativeRecordClass)
		}
		t.Fatal(err)
	}
	if got := g.nativeMapVGA[4*320+4]; got == 0 {
		t.Fatal("player-backed native viewport remained empty")
	}
}

func TestBuildNativeMapFrameInputUsesOnlyRawMaterializedState(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	if !state.AdvanceNativeTerrainPhase(3, -1) {
		t.Fatal("terrain phase advance rejected")
	}
	state.NativeMapCycleState = fdicon.NativeMapSpriteCycleState{Idle: 2, Moving: 3}
	if !state.AdvanceNativeTerrainFlip(1) || !state.AdvanceNativeUnitPixelShift(1) {
		t.Fatal("binary map timing advance rejected")
	}
	runtime := nativeMapFrameRuntime{HUD: indexedmap.NativeMapHUDInput{}}
	got, err := buildNativeMapFrameInput(assets, field, state, runtime)
	if err != nil {
		t.Fatal(err)
	}
	if got.Frame.TerrainCycle != 2 || got.Frame.IdleCycle != 2 || got.Frame.MovingCycle != 3 ||
		got.Frame.Flip != 1 || got.Frame.PixelShift != 1 ||
		got.Frame.RangeMode != 1 || got.Frame.ForegroundBank != assets.Terrain ||
		got.Frame.SelectorCache != state.NativeMapSelectorCache ||
		len(got.Frame.Units) != 1 || len(got.Frame.ForegroundUnits) != 1 ||
		got.HUDCache != state.NativeMapSelectorCache ||
		!got.HUD.DisplayGateA || !got.HUD.DisplayGateB || got.HUD.AnchorX != 1 {
		t.Fatalf("frame input=%+v", got.Frame)
	}
}

func TestBuildNativeMapFrameInputRejectsControlDriftAndMissingRawRoster(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	field.NativeTerrainControl[0] = 1
	if _, err := buildNativeMapFrameInput(assets, field, state, nativeMapFrameRuntime{}); err == nil {
		t.Fatal("accepted editable/native control-table drift")
	}
	field.NativeTerrainControl[0] = 0
	state.Units[0].HasBattleFig = false
	if got, err := buildNativeMapFrameInput(assets, field, state, nativeMapFrameRuntime{}); err == nil ||
		got.Frame.Units != nil || got.Frame.ForegroundUnits != nil {
		t.Fatalf("partial frame=%+v err=%v", got.Frame, err)
	}
}
