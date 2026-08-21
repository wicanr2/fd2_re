package main

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func completeNativeUnitPresentGame(t *testing.T) *Game {
	t.Helper()
	assets, field, state := completeNativeMapFrameFixture(t)
	field.W, field.H = 20, 15
	field.Tiles = make([]int, field.W*field.H)
	field.NativeTileBlitModes = make([]byte, len(field.Tiles))
	for i := range field.NativeTileBlitModes {
		field.NativeTileBlitModes[i] = 0xff
	}
	state.W, state.H = field.W, field.H
	state.NativeTileBlitModes = append([]byte(nil), field.NativeTileBlitModes...)
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 4, CameraY: 3, CursorX: 10, CursorY: 7,
		VisibleCursorX: 6, VisibleCursorY: 4,
	}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		clone := *state.Units[0]
		clone.MapSelectorSlot, clone.HasMapSelectorSlot = 0, false
		clone.NativeMapPresentation = battle.NativeMapPresentationState{}
		clone.HasNativeMapPresentation = false
		clone.X, clone.Y = 5+i, 5
		if err := state.AppendNativeMapSelectorBatch([]*battle.Unit{&clone}); err != nil {
			t.Fatal(err)
		}
	}
	for table := range assets.LUTs {
		for value := range assets.LUTs[table] {
			assets.LUTs[table][value] = byte((value + table + 1) & 0xff)
		}
	}
	g := &Game{
		nativeMapAssets: assets, m: field, st: state,
		camX: 4 * 24, camY: 3 * 24,
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g
}

func recoveredCh28UnitPresent() campaign.NativeUnitPresent {
	return campaign.NativeUnitPresent{
		LastRuntimeSlot: true,
		NewX:            15, NewY: 10, VisualX: 15, VisualY: 10,
	}
}

func stepNativeUnitPresentNow(g *Game) {
	g.nativeUnitPresent.drawn = true
	g.nativeUnitPresent.wait = 0
	g.stepNativeUnitPresent()
}

func TestNativeUnitPresentMutatesOnlyAfterContractAndCompletesAllFrames(t *testing.T) {
	g := completeNativeUnitPresentGame(t)
	unit := g.st.Units[len(g.st.Units)-1]
	oldX, oldY := unit.X, unit.Y
	continued := 0
	if err := g.startNativeUnitPresent(recoveredCh28UnitPresent(), func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	job := g.nativeUnitPresent
	if job == nil || job.mutationAt != 17 || len(job.vgaFrames) != 51 || len(job.workFrames) != 51 {
		t.Fatalf("job=%+v", job)
	}
	for step := 0; step < 16; step++ {
		stepNativeUnitPresentNow(g)
	}
	if unit.X != oldX || unit.Y != oldY || g.nativeUnitPresent.frame != 16 {
		t.Fatalf("coordinate changed before contract boundary: (%d,%d) frame=%d", unit.X, unit.Y, g.nativeUnitPresent.frame)
	}
	stepNativeUnitPresentNow(g)
	if unit.X != 15 || unit.Y != 10 || g.nativeUnitPresent.frame != 17 {
		t.Fatalf("coordinate was not committed at bridge boundary: (%d,%d) frame=%d", unit.X, unit.Y, g.nativeUnitPresent.frame)
	}
	for steps := 0; g.nativeUnitPresent != nil && steps < 100; steps++ {
		stepNativeUnitPresentNow(g)
	}
	if g.nativeUnitPresent != nil || continued != 1 || unit.X != 15 || unit.Y != 10 {
		t.Fatalf("job=%v continued=%d unit=(%d,%d)", g.nativeUnitPresent, continued, unit.X, unit.Y)
	}
}

func TestNativeUnitPresentPreflightFailureLeavesStateAndBuffersUntouched(t *testing.T) {
	g := completeNativeUnitPresentGame(t)
	unit := g.st.Units[len(g.st.Units)-1]
	beforeUnit := *unit
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	g.nativeMapAssets.FDOTHER6[0x72].Pixels = nil
	if err := g.startNativeUnitPresent(recoveredCh28UnitPresent(), nil); err == nil {
		t.Fatal("malformed FDOTHER #6 intro entry was accepted")
	}
	if g.nativeUnitPresent != nil || !reflect.DeepEqual(*unit, beforeUnit) ||
		!bytes.Equal(beforeWork, g.nativeMapWork) || !bytes.Equal(beforeVGA, g.nativeMapVGA) {
		t.Fatal("failed preflight changed live state or indexed buffers")
	}
}

func TestNativeUnitPresentRuntimeFailureRollsBackCoordinateAndBuffers(t *testing.T) {
	g := completeNativeUnitPresentGame(t)
	unit := g.st.Units[len(g.st.Units)-1]
	oldX, oldY := unit.X, unit.Y
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	if err := g.startNativeUnitPresent(recoveredCh28UnitPresent(), nil); err != nil {
		t.Fatal(err)
	}
	for g.nativeUnitPresent.frame < g.nativeUnitPresent.mutationAt {
		stepNativeUnitPresentNow(g)
	}
	if unit.X != 15 || unit.Y != 10 {
		t.Fatal("test did not reach mutation boundary")
	}
	g.failNativeUnitPresent(assertionError("forced renderer failure"))
	if unit.X != oldX || unit.Y != oldY || !bytes.Equal(beforeWork, g.nativeMapWork) ||
		!bytes.Equal(beforeVGA, g.nativeMapVGA) {
		t.Fatal("runtime failure did not restore coordinate/work/VGA")
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }
