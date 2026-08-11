package main

import (
	"slices"
	"testing"

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
		st:              &battle.State{HasNativeMapViewState: true},
		ring:            true,
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
	for _, id := range []int{0, 13, 16, 20, 21, 22, 24, 25, 26, 27, 28, 29, 31} {
		if !g.nativeCommandTargetSupported(id) {
			t.Fatalf("verified target/effect id %d was rejected", id)
		}
	}
	for _, id := range []int{-1, 1, 9, 10, 17, 18, 19, 23, 30, 32, 35, 36} {
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
