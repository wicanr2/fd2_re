package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

func nativeCurrentSaveTestGame(t *testing.T) (*Game, string, []byte) {
	t.Helper()
	plain := make([]byte, fdsave.FileSize)
	plain[fdsave.CurrentFieldControlOffset+2] = 1
	header := plain[fdsave.CurrentRuntimeHeaderOffset : fdsave.CurrentRuntimeHeaderOffset+fdsave.CurrentRuntimeHeaderSize]
	copy(header, []byte{3, 1, 0, 1, 2, 8, 7, 7, 5, 1, 10, 0, 0, 0, 0, 1, 1, 1})
	baseline := fdsave.PersistentRecord{}
	baseline.Raw[0] = 8
	baseline.Raw[1] = 7
	baseline.Raw[3] = 2
	baseline.Raw[4] = 3
	baseline.Raw[5] = 0
	baseline.Raw[6] = 2
	baseline.Raw[7] = 9
	baseline.Raw[8] = 4
	baseline.Raw[0x1f] = 1
	baseline.Raw[0x20] = 2
	baseline.Raw[0x21] = 12
	baseline.Raw[0x3b] = 5
	for slot := 0; slot < 8; slot++ {
		baseline.Raw[0x0a+slot*2] = 0x80
		baseline.Raw[0x0b+slot*2] = 0xff
	}
	copy(plain[fdsave.CurrentRuntimeRosterOffset:], baseline.Raw[:])
	copy(plain[fdsave.CurrentPersistentRosterOffset:], baseline.Raw[:])
	stored, err := fdsave.Encode(plain)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "FD2.SAV")
	if err := os.WriteFile(path, stored, 0o600); err != nil {
		t.Fatal(err)
	}
	plain, err = fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_SAVE", path)
	unit := &battle.Unit{
		X: 8, Y: 7, OnField: true,
		NativeMapPresentation:    battle.NativeMapPresentationState{X: 8, Y: 7, Pose: 2, Motion: 3},
		HasNativeMapPresentation: true,
		BattleFig:                9, HasBattleFig: true,
		MapSelectorKey: 9, HasMapSelectorKey: true,
		MapSelectorSlot: 0, HasMapSelectorSlot: true,
		NativeRecordByte5: 0, HasNativeRecordByte5: true,
		NativeRecordByte6: 2, HasNativeRecordByte6: true,
		NativeRecordByte8: 4, HasNativeRecordByte8: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 2, HasNativeRecordClass: true,
		InventorySlots:       []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
		Lv:                   12, MV: 5, Exp: 0, DX: 0, HP: 20, MaxHP: 30, MP: 4, MaxMP: 8,
	}
	state := &battle.State{
		W: 20, H: 20, Units: []*battle.Unit{unit}, NativeRoundCounter: 4,
		NativeFieldControlRaw:          make([]byte, fdsave.CurrentFieldControlSize),
		HasNativeFieldControlState:     true,
		NativeRuntimeRecords:           []battle.NativeRuntimeRecordState{{Raw: baseline.Raw, SelectorKey: 9, SelectorSlot: 0}},
		HasNativeRuntimeUnitProjection: true,
		NativeMapViewState: battle.NativeMapViewState{
			CameraX: 1, CameraY: 2, CursorX: 8, CursorY: 7, VisibleCursorX: 7, VisibleCursorY: 5,
		},
		HasNativeMapViewState: true,
	}
	state.NativeFieldControlRaw[2] = 1
	g := &Game{
		st: state, sc: &battle.Scenario{Chapter: 1}, curX: 8, curY: 7, gold: 77,
		nativeCurrentSavePlain: append([]byte(nil), plain...),
	}
	return g, path, stored
}

func drainNativeCurrentSaveUI(g *Game) {
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
}

func TestNativeCurrentSaveBuildAndAtomicReplaceRoundTrip(t *testing.T) {
	g, path, before := nativeCurrentSaveTestGame(t)
	pathGot, stored, err := g.buildNativeCurrentSaveStored()
	if err != nil || pathGot != path {
		t.Fatalf("build path=%q err=%v", pathGot, err)
	}
	if bytes.Equal(stored, before) {
		t.Fatal("live current snapshot did not change encoded save")
	}
	if err := replaceNativeCurrentSaveAtomic(path, stored); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := fdsave.Decode(written)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Header.TurnCounter != 4 || snapshot.Header.Currency != 77 ||
		snapshot.RuntimeRecords[0].View().HP != 20 || snapshot.RuntimeRecords[0].View().MaxHP != 30 {
		t.Fatalf("written current snapshot header=%+v record=%+v", snapshot.Header, snapshot.RuntimeRecords[0].View())
	}
}

func TestNativeCurrentSaveFailureLeavesOriginalFileUnchanged(t *testing.T) {
	g, path, before := nativeCurrentSaveTestGame(t)
	g.st.Units[0].X++
	if _, _, err := g.buildNativeCurrentSaveStored(); err == nil {
		t.Fatal("stale native map presentation unexpectedly produced save")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("failed preflight changed FD2.SAV")
	}
}

func TestNativeNestedCurrentSaveWritesOnlyAfterYESAndCancelPreservesFile(t *testing.T) {
	prepare := func(t *testing.T) (*Game, string, []byte) {
		t.Helper()
		g, path, before := nativeCurrentSaveTestGame(t)
		base := "../../../org_game/炎龍騎士團/FLAME2"
		t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
		var err error
		g.nativePreparationUI, err = loadNativePreparationUIAssets()
		if err != nil {
			t.Skipf("player-provided original UI assets are absent: %v", err)
		}
		g.nativeClassUI, err = loadNativeClassUIAssets()
		if err != nil {
			t.Fatal(err)
		}
		g.nativeMapVGA = make([]byte, 320*200)
		g.nativeSystemCursorOverlay = true
		g.nativeSystemNestedOpen = true
		g.ring = true
		g.ringSel = 1
		return g, path, before
	}
	closeNested := func(g *Game) {
		for present := 0; present < 4; present++ {
			g.markActionOverlayDrawn()
			g.stepActionOverlayLifecycle()
		}
		drainNativeCurrentSaveUI(g)
	}
	finishDialogue := func(g *Game) {
		drainNativeCurrentSaveUI(g)
		for g.nativeSystemEndTurnDelay > 0 {
			g.stepNativeSystemEndTurn()
		}
		drainNativeCurrentSaveUI(g)
	}

	cancel, cancelPath, cancelBefore := prepare(t)
	if !cancel.activateNativeSystemDirectionOne() {
		t.Fatal("nested SAVE route rejected")
	}
	if got, _ := os.ReadFile(cancelPath); !bytes.Equal(got, cancelBefore) {
		t.Fatal("SAVE preflight changed FD2.SAV before confirmation")
	}
	closeNested(cancel)
	cancel.cancelNativeSystemEndTurn()
	finishDialogue(cancel)
	if got, _ := os.ReadFile(cancelPath); !bytes.Equal(got, cancelBefore) {
		t.Fatal("SAVE cancellation changed FD2.SAV")
	}

	accept, acceptPath, acceptBefore := prepare(t)
	if !accept.activateNativeSystemDirectionOne() {
		t.Fatal("nested SAVE route rejected")
	}
	closeNested(accept)
	accept.confirmNativeSystemEndTurn()
	finishDialogue(accept)
	got, err := os.ReadFile(acceptPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(got, acceptBefore) {
		t.Fatal("SAVE YES did not replace FD2.SAV")
	}
	if accept.nativeSystemEndTurnUI != nil || accept.st.Turn != 0 || accept.aiBusy {
		t.Fatalf("SAVE completion leaked UI or ended turn: ui=%#v turn=%d ai=%v", accept.nativeSystemEndTurnUI, accept.st.Turn, accept.aiBusy)
	}
}
