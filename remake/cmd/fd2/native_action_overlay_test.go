package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func TestNativeActionOffsetXYMatchesFinalOpenFrame(t *testing.T) {
	offsets, err := fdother.ActionOverlayFrameOffsets(3, false)
	if err != nil {
		t.Fatal(err)
	}
	want := [4][2]int{{0, -13}, {-18, 2}, {18, 2}, {0, 17}}
	for direction, offset := range offsets {
		x, y := nativeActionOffsetXY(offset)
		if got := [2]int{x, y}; got != want[direction] {
			t.Fatalf("direction %d offset=%#x gives %v, want %v", direction, offset, got, want[direction])
		}
	}
}

func TestNativeSystemOverlayUses16F55CellState(t *testing.T) {
	g := &Game{nativeSystemCursorOverlay: true}
	state := g.nativeActionOverlayState()
	want := [4]int{21, 15, 18, 12}
	for direction := range want {
		got, err := state.CellIndex(direction)
		if err != nil || got != want[direction] {
			t.Fatalf("direction %d: index=%d err=%v, want %d", direction, got, err, want[direction])
		}
	}
}

func TestSharedBattleEmptyCursorOpensNativeSystemOverlay(t *testing.T) {
	g := &Game{
		m:                 &MapData{W: 20, H: 20, TileW: 24, TileH: 24},
		st:                &battle.State{W: 20, H: 20},
		curX:              9,
		curY:              7,
		nativeActionCells: nativeSystemOverlayTestCells(),
	}
	g.confirm()
	if !g.ring || !g.nativeSystemCursorOverlay || g.ringSel != 0 ||
		g.actionOverlayPhase != actionOverlayOpening || g.sel != nil {
		t.Fatalf("shared 0x117E7 empty-cursor overlay = %+v", g)
	}
}

func TestSharedBattleEmptyCursorRejectsIncompleteNativeOverlay(t *testing.T) {
	cells := nativeSystemOverlayTestCells()
	cells[18] = nil
	g := &Game{
		m:                 &MapData{W: 20, H: 20, TileW: 24, TileH: 24},
		st:                &battle.State{W: 20, H: 20},
		curX:              9,
		curY:              7,
		nativeActionCells: cells,
	}
	g.confirm()
	if g.ring || g.nativeSystemCursorOverlay || g.sel != nil {
		t.Fatalf("incomplete FDOTHER #2 opened invisible system overlay: %+v", g)
	}
}

func TestLoadNativeActionCellsKeepsFullReferenceBank(t *testing.T) {
	path := filepath.Join("../../../org_game/炎龍騎士團/FLAME2", "FDOTHER.DAT")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("player-provided FDOTHER.DAT is absent: %v", err)
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", path)
	palette := loadNativeUIPalette()
	cells := loadNativeActionCells(palette)
	if len(cells) != nativeActionOverlayCellCount {
		t.Fatalf("cell bank length=%d, want %d", len(cells), nativeActionOverlayCellCount)
	}
	for _, index := range []int{12, 15, 18, 21} {
		if cells[index] == nil {
			t.Fatalf("current-runtime cell %d was not loaded", index)
		}
	}
}

func TestNativeMapFrameAdmissionKeepsActionOverlayOnCompleteFrame(t *testing.T) {
	if !(&Game{ring: true}).nativeMapFrameAdmission(false, true) {
		t.Fatal("action overlay was excluded from the complete native map frame")
	}
	if !(&Game{nativeCommandOpen: true}).nativeMapFrameAdmission(false, true) {
		t.Fatal("native command grid was excluded from the complete native map frame")
	}
	if (&Game{sel: &battle.Unit{}}).nativeMapFrameAdmission(false, true) {
		t.Fatal("ordinary selected-unit movement was admitted without a complete modal state")
	}
	for _, state := range []*Game{
		{ring: true, spellOpen: true},
		{nativeCommandOpen: true, itemOpen: true},
		{nativeCommand0Targeting: true, castSp: &battle.Spell{}},
	} {
		if state.nativeMapFrameAdmission(false, true) {
			t.Fatalf("incompatible modal state was admitted: %+v", state)
		}
	}
	if (&Game{ring: true}).nativeMapFrameAdmission(true, true) {
		t.Fatal("legacy viewport was admitted")
	}
}

func TestNativeActionOverlayAnchorUsesPresentationScale(t *testing.T) {
	unit := &battle.Unit{X: 8, Y: 16}
	base := &Game{
		m:               &MapData{TileW: 24, TileH: 24},
		camX:            24,
		camY:            312,
		nativeMapAssets: &nativeMapAssets{},
		st: &battle.State{HasNativeMapViewState: true, NativeMapViewState: battle.NativeMapViewState{
			CursorX: 8, CursorY: 16,
		}},
		ring: true,
	}
	x, y, scale := base.actionOverlayAnchor(unit)
	if scale != 2 || x != 336 || y != 144 {
		t.Fatalf("native anchor=(%.0f,%.0f) scale=%.0f, want (336,144) scale=2", x, y, scale)
	}
	base.nativeMapAssets = nil
	x, y, scale = base.actionOverlayAnchor(unit)
	if scale != 1 || x != 168 || y != 72 {
		t.Fatalf("normalized anchor=(%.0f,%.0f) scale=%.0f, want (168,72) scale=1", x, y, scale)
	}
}

func TestActionOverlayLifecyclePresentsAllOpeningAndClosingFrames(t *testing.T) {
	g := &Game{}
	g.beginActionOverlayOpen(2)
	if !g.ring || g.ringSel != 2 || !g.actionOverlayBlocksInput() {
		t.Fatalf("opening state = ring:%v sel:%d phase:%q", g.ring, g.ringSel, g.actionOverlayPhase)
	}
	for want := 0; want < 4; want++ {
		frame, closing := g.actionOverlayRenderState()
		if frame != want || closing {
			t.Fatalf("opening present %d = frame%d closing=%v", want, frame, closing)
		}
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	if !g.ring || g.actionOverlayPhase != actionOverlayOpen || g.actionOverlayBlocksInput() {
		t.Fatalf("settled open state = ring:%v phase:%q", g.ring, g.actionOverlayPhase)
	}

	completed := false
	g.beginActionOverlayClose(func() { completed = true })
	for want := 0; want < 4; want++ {
		frame, closing := g.actionOverlayRenderState()
		if frame != want || !closing || completed {
			t.Fatalf("closing present %d = frame%d closing=%v completed=%v", want, frame, closing, completed)
		}
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	if g.ring || g.actionOverlayPhase != "" || !completed {
		t.Fatalf("completed close state = ring:%v phase:%q completed=%v", g.ring, g.actionOverlayPhase, completed)
	}
}

func TestActionOverlayCloseDefersChildMenuUntilFourthPresent(t *testing.T) {
	g := &Game{}
	g.beginActionOverlayOpen(1)
	for g.actionOverlayBlocksInput() {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	g.beginActionOverlayClose(func() { g.nativeCommandOpen = true })
	for present := 0; present < 4; present++ {
		if g.nativeCommandOpen {
			t.Fatalf("child menu opened beneath close frame %d", present)
		}
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	if !g.nativeCommandOpen || g.ring {
		t.Fatalf("close did not hand off to child menu: command=%v ring=%v", g.nativeCommandOpen, g.ring)
	}
}

func TestActionOverlayLifecycleCannotSkipAnUndrawnFrame(t *testing.T) {
	g := &Game{}
	g.beginActionOverlayOpen(1)
	g.stepActionOverlayLifecycle()
	g.stepActionOverlayLifecycle()
	if frame, closing := g.actionOverlayRenderState(); frame != 0 || closing {
		t.Fatalf("undrawn opening frame advanced to frame%d closing=%v", frame, closing)
	}
	g.markActionOverlayDrawn()
	g.stepActionOverlayLifecycle()
	if frame, _ := g.actionOverlayRenderState(); frame != 1 {
		t.Fatalf("drawn opening frame did not advance: frame%d", frame)
	}
}

func TestActionOverlayAvailabilityUsesCurrentRemakeGates(t *testing.T) {
	g := &Game{st: &battle.State{}, sel: &battle.Unit{Spells: []int{1}, Inventory: []int{2}}}
	if got := g.actionOverlayAvailability(); got != [4]int{1, 0, 0, 0} {
		t.Fatalf("availability=%v", got)
	}
	g.sel.Sealed = true
	g.sel.Inventory = nil
	if got := g.actionOverlayAvailability(); got != [4]int{1, 1, 1, 0} {
		t.Fatalf("availability=%v", got)
	}
}

func TestActionOverlayAvailabilityRequiresRecoveredNativeInventoryGates(t *testing.T) {
	selected := battle.Unit{OnField: true, HP: 10, AtkMin: 1, AtkMax: 1, Inventory: []int{3}, Equipped: []bool{false}}
	enemy := &battle.Unit{OnField: true, HP: 10, Camp: battle.Enemy, X: 1, Y: 0}
	g := &Game{st: &battle.State{Units: []*battle.Unit{enemy}}, sel: &selected}
	if got := g.actionOverlayAvailability(); got != [4]int{1, 1, 0, 0} {
		t.Fatalf("without equipped weapon/raw command availability=%v", got)
	}
	selected.Equipped[0] = true
	selected.NativeCommandMask[0] = 0x01
	if got := g.actionOverlayAvailability(); got != [4]int{0, 0, 0, 0} {
		t.Fatalf("with equipped weapon/raw command availability=%v", got)
	}
}

func TestActionOverlayNativeCommandGateUsesRawOffset27NotLegacySeal(t *testing.T) {
	selected := battle.Unit{
		OnField: true, HP: 10, AtkMin: 1, AtkMax: 1,
		Inventory: []int{3}, Equipped: []bool{true},
		NativeCommandMask: [5]byte{1},
	}
	enemy := &battle.Unit{OnField: true, HP: 10, Camp: battle.Enemy, X: 1, Y: 0}
	g := &Game{st: &battle.State{Units: []*battle.Unit{enemy}}, sel: &selected}
	selected.Sealed = true // normalized legacy state is not the native +0x27 gate.
	if got := g.actionOverlayAvailability(); got != [4]int{0, 0, 0, 0} {
		t.Fatalf("legacy seal must not disable raw command: availability=%v", got)
	}
	selected.NativeTransient[5] = 1 // exact raw unit+0x27.
	if got := g.actionOverlayAvailability(); got != [4]int{0, 1, 0, 0} {
		t.Fatalf("raw +0x27 must disable command: availability=%v", got)
	}
}

func TestNativeActionSelectableRejectsDisabledWordAndInvalidDirection(t *testing.T) {
	availability := [4]int{0, 1, 0, 0}
	for _, direction := range []int{-1, 1, 4} {
		if nativeActionSelectable(availability, direction) {
			t.Fatalf("direction %d must not be selectable", direction)
		}
	}
	if !nativeActionSelectable(availability, 0) || !nativeActionSelectable(availability, 3) {
		t.Fatal("zero disabled-word must be selectable")
	}
}

func TestNativeCommandTargetWhitelistKeepsUnresolvedIDsFailClosed(t *testing.T) {
	g := &Game{}
	for _, id := range []int{0, 13, 16, 17, 18, 19, 20, 21, 22, 24, 25, 26, 27, 28, 29, 31} {
		if !g.nativeCommandTargetSupported(id) {
			t.Fatalf("verified target/effect id %d was rejected", id)
		}
	}
	for _, id := range []int{-1, 1, 9, 10, 23, 30, 32, 35, 36} {
		if g.nativeCommandTargetSupported(id) {
			t.Fatalf("unresolved target/effect id %d was enabled", id)
		}
	}
}

func TestNativeCommandTargetProjectionUsesSelectedRawCommandRecord(t *testing.T) {
	book := make([]battle.NativeCommandRecord, 36)
	book[0] = battle.NativeCommandRecord{ID: 0, SelectionMode: 1, TargetCode: 0}
	// An invalid selected record must fail; hard-coding record 0 would
	// incorrectly return a target list here.
	book[13] = battle.NativeCommandRecord{ID: 13, SelectionMode: -1, TargetCode: 0}
	actor := &battle.Unit{Camp: battle.Own, OnField: true, HP: 10, X: 0, Y: 0}
	target := &battle.Unit{Camp: battle.Enemy, OnField: true, HP: 10, X: 1, Y: 0}
	g := &Game{
		st:  &battle.State{W: 2, H: 1, Units: []*battle.Unit{actor, target}, NativeCommandBook: book, NativeCompositionEventBytes: make([]byte, 2)},
		sel: actor, nativeCommandTargetID: 13,
	}
	if _, err := g.nativeCommandTargetUnits(); err == nil {
		t.Fatal("selected raw command record was not used")
	}
}

func TestNativeCommandTargetFieldMaterializeAndReset(t *testing.T) {
	actor := &battle.Unit{X: 0, Y: 0}
	g := &Game{
		st: &battle.State{
			W: 3, H: 1,
			NativeCompositionEventBytes: make([]byte, 3),
			NativeTileBlitModes:         []byte{0xff, 0xff, 0xff},
		},
		sel: actor,
	}
	record := battle.NativeCommandRecord{SelectionMode: 1}
	if err := g.materializeNativeCommandTargetField(record); err != nil {
		t.Fatal(err)
	}
	if got, want := g.st.NativeTileBlitModes, []byte{1, 0, 0xff}; !slices.Equal(got, want) {
		t.Fatalf("target field=%#v want %#v", got, want)
	}
	if !g.st.HasNativeMapRangeModeState || g.st.NativeMapRangeMode != 2 {
		t.Fatalf("target selector=%d/%v", g.st.NativeMapRangeMode, g.st.HasNativeMapRangeModeState)
	}
	if !g.resetNativeTargetField() ||
		!slices.Equal(g.st.NativeTileBlitModes, []byte{0xff, 0xff, 0xff}) {
		t.Fatalf("target field reset=%#v", g.st.NativeTileBlitModes)
	}
}

func TestPlayerNativeCommand13RunsCursorTransactionThroughConfirm(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	for i := range assets.LUTs {
		for p := range assets.LUTs[i] {
			assets.LUTs[i][p] = byte(p)
		}
	}
	book := make([]battle.NativeCommandRecord, 36)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[13] = battle.NativeCommandRecord{
		ID: 13, Damage: 70, SelectionMode: 1, EffectMode: 0,
		MPCost: 3, TargetCode: 1,
	}
	actor := state.Units[0]
	actor.Camp, actor.OnField = battle.Own, true
	actor.HP, actor.MaxHP, actor.MP = 40, 100, 5
	actor.NativeRecordWord42, actor.HasNativeRecordWord42 = 100, true
	state.NativeCommandBook = book
	state.NativeCompositionEventBytes = make([]byte, state.W*state.H)
	state.NativeTileBlitModes[0], field.NativeTileBlitModes[0] = 0, 0
	state.MaterializeNativeMapRangeMode(3)
	g := &Game{
		nativeMapAssets: assets, nativeMapDAC: append([]byte(nil), assets.PaletteDAC...),
		m: field, st: state, sel: actor, curX: 0, curY: 0,
		nativeCommand0Targeting: true, nativeCommandTargetID: 13,
		rng: rand.New(rand.NewSource(2)),
	}
	g.confirm()
	if g.nativeHealPresentation != nil || actor.MP != 5 || actor.HP != 40 || actor.Acted {
		t.Fatalf("missing indexed baseline did not fail closed: job=%v actor=%#v",
			g.nativeHealPresentation != nil, actor)
	}
	if err := g.composeNativeMapFrameAt(time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	digits := assets.CommandHealDigits
	assets.CommandHealDigits = nil
	g.confirm()
	if g.nativeHealPresentation != nil || actor.MP != 5 || actor.HP != 40 || actor.Acted {
		t.Fatalf("missing FDOTHER#5 did not fail closed: job=%v actor=%#v", g.nativeHealPresentation != nil, actor)
	}
	assets.CommandHealDigits = digits

	g.confirm()
	if g.nativeHealPresentation == nil || actor.MP != 5 || actor.HP != 40 || actor.Acted {
		t.Fatalf("command13 did not stop at presentation boundary: job=%v actor=%#v",
			g.nativeHealPresentation != nil, actor)
	}
	for step := 0; g.nativeHealPresentation != nil && step < 256; step++ {
		if phase := g.nativeHealPresentation.phase; phase == nativeCommandHealFrames || phase == nativeCommandHealEffectFrames || phase == nativeCommandHealMaskFrames || phase == nativeCommandHealDigitFrames {
			g.nativeHealPresentation.drawn = true
		}
		g.stepNativeCommandHealPresentation()
	}
	if g.nativeHealPresentation != nil {
		t.Fatal("command13 presentation did not complete")
	}

	if actor.MP != 2 || !actor.Acted || actor.HP != 100 {
		t.Fatalf("player command13 transaction actor=%#v", actor)
	}
	if g.nativeCommand0Targeting || g.sel != nil || state.NativeMapRangeMode != 1 {
		t.Fatalf("player command13 cleanup targeting=%v sel=%#v range=%d",
			g.nativeCommand0Targeting, g.sel, state.NativeMapRangeMode)
	}
	if g.msg != "原始指令 13：回復 60" {
		t.Fatalf("player command13 message=%q", g.msg)
	}
}

func TestPlayerNativeCommand17WaitsForEightPalettePhasesBeforeTransaction(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[17] = battle.NativeCommandRecord{ID: 17, SelectionMode: 1, EffectMode: 0, MPCost: 99, TargetCode: 1}
	book[18] = battle.NativeCommandRecord{ID: 18, SelectionMode: 1, EffectMode: 0, MPCost: 4, TargetCode: 1}
	actor := state.Units[0]
	actor.Camp, actor.OnField = battle.Own, true
	actor.HP, actor.MaxHP, actor.MP, actor.AP = 40, 100, 10, 100
	actor.Lv, actor.NativeRecordClass, actor.HasNativeRecordClass = 2, 9, true
	actor.NativeTransient = [6]byte{}
	state.NativeCommandBook = book
	state.NativeCompositionEventBytes = make([]byte, state.W*state.H)
	state.NativeTileBlitModes[0], field.NativeTileBlitModes[0] = 0, 0
	state.MaterializeNativeMapRangeMode(3)
	table, err := battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		nativeMapAssets: assets, nativeMapDAC: append([]byte(nil), assets.PaletteDAC...),
		m: field, st: state, sel: actor, curX: 0, curY: 0,
		nativeCommand0Targeting: true, nativeCommandTargetID: 17,
		nativeCommandPaletteFlash: table,
	}

	// Missing framebuffer and then missing exact sample must both reject before
	// any palette frame, MP debit or target mutation.
	g.confirm()
	if g.nativeModifierPresentation != nil || actor.MP != 10 || actor.AP != 100 || actor.Acted {
		t.Fatalf("missing framebuffer crossed boundary: job=%v actor=%#v", g.nativeModifierPresentation != nil, actor)
	}
	if err := g.composeNativeMapFrameAt(time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	g.confirm()
	if g.nativeModifierPresentation != nil || actor.MP != 10 || actor.AP != 100 || actor.Acted {
		t.Fatalf("missing sample crossed boundary: job=%v actor=%#v", g.nativeModifierPresentation != nil, actor)
	}

	t.Setenv("FD2_MUTE", "1")
	g.confirm()
	if g.nativeModifierPresentation == nil || actor.MP != 10 || actor.AP != 100 || actor.Acted {
		t.Fatalf("command17 did not stop at palette boundary: job=%v actor=%#v", g.nativeModifierPresentation != nil, actor)
	}
	if len(g.nativeModifierPresentation.palettes) != 8 {
		t.Fatalf("palette phases=%d want 8", len(g.nativeModifierPresentation.palettes))
	}
	for phase := 0; phase < 8; phase++ {
		job := g.nativeModifierPresentation
		if job == nil || job.phase != phase || actor.MP != 10 || actor.AP != 100 || actor.Acted {
			t.Fatalf("phase %d mutated early: job=%#v actor=%#v", phase, job, actor)
		}
		entry := job.palettes[phase][0]
		r, green, blue, _ := entry.RGBA()
		black := r == 0 && green == 0 && blue == 0
		if phase%2 == 0 {
			if black {
				t.Fatalf("phase %d command color is black", phase)
			}
		} else if !black {
			t.Fatalf("phase %d black entry=%v", phase, entry)
		}
		if !g.drawNativeCommandModifierPresentation(ebiten.NewImage(640, 400)) {
			t.Fatalf("phase %d did not draw", phase)
		}
		g.stepNativeCommandModifierPresentation()
	}
	if g.nativeModifierPresentation != nil || actor.MP != 6 || actor.AP != 116 ||
		actor.NativeTransient[0] != 2 || !actor.Acted {
		t.Fatalf("command17 transaction actor=%#v job=%v", actor, g.nativeModifierPresentation != nil)
	}
}

func TestPlayerNativeCommands20And22UseSharedPaletteBoundary(t *testing.T) {
	for _, commandID := range []int{20, 22} {
		t.Run(fmt.Sprintf("command_%d", commandID), func(t *testing.T) {
			assets, field, state := completeNativeMapFrameFixture(t)
			book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
			for id := range book {
				book[id] = battle.NativeCommandRecord{ID: id}
			}
			book[10].Damage = 100
			book[commandID] = battle.NativeCommandRecord{
				ID: commandID, SelectionMode: 1, EffectMode: 0, MPCost: 2, TargetCode: 1,
			}
			actor := state.Units[0]
			actor.Camp, actor.OnField, actor.MP = battle.Own, true, 5
			actor.HP, actor.MaxHP = 10, 100
			state.NativeCommandBook = book
			state.NativeCompositionEventBytes = make([]byte, state.W*state.H)
			state.NativeTileBlitModes[0], field.NativeTileBlitModes[0] = 0, 0
			state.MaterializeNativeMapRangeMode(3)

			if commandID == 22 {
				book[22].TargetCode = 0
			}
			table, err := battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
			if err != nil {
				t.Fatal(err)
			}
			g := &Game{
				nativeMapAssets: assets, nativeMapDAC: append([]byte(nil), assets.PaletteDAC...),
				m: field, st: state, sel: actor,
				nativeCommand0Targeting: true, nativeCommandTargetID: commandID,
				nativeCommandPaletteFlash: table, rng: rand.New(rand.NewSource(0)),
			}
			if err := g.composeNativeMapFrameAt(time.Unix(0, 0)); err != nil {
				t.Fatal(err)
			}
			target := &battle.Unit{
				Camp: battle.Own, OnField: true, HP: 1, MaxHP: 100,
				X: 1, Y: 0, NativeTransient: [6]byte{0, 0, 0, 3},
				NativeRecordByte5: 0, HasNativeRecordByte5: true,
				NativeRecordByte6: 0, HasNativeRecordByte6: true,
			}
			if commandID == 22 {
				target.Camp, target.ClassID, target.HP, target.MaxHP = battle.Enemy, 2, 20, 20
				target.NativeRecordByte6 = 1
				target.NativeTransient = [6]byte{}
			}
			state.Units = append(state.Units, target)
			state.NativeTileBlitModes[target.Y*state.W+target.X] = 0
			g.curX, g.curY = target.X, target.Y
			if allowed, gateErr := battle.NativeCursorConfirmationAllowed(
				battle.Cell{X: g.curX, Y: g.curY}, 0, state.NativeMapRangeMode,
				book[commandID].TargetCode, state.Units,
			); gateErr != nil || !allowed {
				t.Fatalf("command %d fixture cursor gate allowed=%v err=%v range=%d targetCode=%d",
					commandID, allowed, gateErr, state.NativeMapRangeMode, book[commandID].TargetCode)
			}
			t.Setenv("FD2_MUTE", "1")
			g.confirm()
			if g.nativeModifierPresentation == nil || actor.MP != 5 || actor.Acted {
				t.Fatalf("command %d crossed pre-presentation boundary: actor=%#v msg=%q", commandID, actor, g.msg)
			}
			for phase := 0; phase < 8; phase++ {
				if !g.drawNativeCommandModifierPresentation(ebiten.NewImage(640, 400)) {
					t.Fatalf("command %d phase %d did not draw", commandID, phase)
				}
				g.stepNativeCommandModifierPresentation()
			}
			if g.nativeModifierPresentation != nil || actor.MP != 3 || !actor.Acted ||
				g.nativeCommand0Targeting || g.sel != nil || state.NativeMapRangeMode != 1 {
				t.Fatalf("command %d transaction/cleanup actor=%#v job=%v targeting=%v sel=%#v range=%d",
					commandID, actor, g.nativeModifierPresentation != nil,
					g.nativeCommand0Targeting, g.sel, state.NativeMapRangeMode)
			}
			if commandID == 20 && target.NativeTransient[3] != 0 {
				t.Fatalf("command 20 did not clear raw +0x25: %#v", target.NativeTransient)
			}
		})
	}
}
