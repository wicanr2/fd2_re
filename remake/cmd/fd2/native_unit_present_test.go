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

func storyStagingFixture(t *testing.T) *Game {
	t.Helper()
	g := completeNativeUnitPresentGame(t)
	view := g.st.NativeMapViewState
	g.storyActors = make([]battle.Unit, len(g.st.Units))
	for index, unit := range g.st.Units {
		g.storyActors[index] = *unit
	}
	g.storyNativeMapView, g.hasStoryNativeMapView = view, true
	g.curX, g.curY = view.CursorX, view.CursorY
	g.st = nil
	return g
}

func TestMaterializeNativeStoryMapStateUsesCurrentLOADCHTerrain(t *testing.T) {
	g := storyStagingFixture(t)
	source := completeNativeUnitPresentGame(t).st
	g.m.NativeTileBlitModes[0] = 0x2a
	if err := g.materializeNativeStoryMapState(source); err != nil {
		t.Fatal(err)
	}
	if g.storyNativeMapState.W != g.m.W || g.storyNativeMapState.H != g.m.H ||
		!bytes.Equal(g.storyNativeMapState.NativeTileBlitModes, g.m.NativeTileBlitModes) {
		t.Fatalf("story terrain=%dx%d/%#v, LOADCH=%dx%d/%#v",
			g.storyNativeMapState.W, g.storyNativeMapState.H, g.storyNativeMapState.NativeTileBlitModes,
			g.m.W, g.m.H, g.m.NativeTileBlitModes)
	}
	g.storyNativeMapState.NativeTileBlitModes[0] = 0x11
	if g.m.NativeTileBlitModes[0] != 0x2a {
		t.Fatal("story terrain state aliases editable LOADCH data")
	}
}

func TestMaterializeNativeStoryMapStateRejectsIncompleteLOADCHTerrain(t *testing.T) {
	g := storyStagingFixture(t)
	source := completeNativeUnitPresentGame(t).st
	g.m.NativeTileBlitModes = g.m.NativeTileBlitModes[:len(g.m.NativeTileBlitModes)-1]
	if err := g.materializeNativeStoryMapState(source); err == nil {
		t.Fatal("incomplete LOADCH terrain renderer inputs were accepted")
	}
	if g.storyNativeMapState != nil {
		t.Fatal("failed materialization published a partial story state")
	}
}

func TestComposeNativeStoryMapFrameKeepsApproximationPrivate(t *testing.T) {
	g := storyStagingFixture(t)
	source := completeNativeUnitPresentGame(t).st
	if err := g.materializeNativeStoryMapState(source); err != nil {
		t.Fatal(err)
	}
	actor := g.storyNativeMapState.Units[0]
	actor.HasNativeRecordRace, actor.HasNativeRecordClass = false, false
	if err := g.composeNativeStoryMapFrame(); err != nil {
		t.Fatal(err)
	}
	if actor.HasNativeRecordRace || actor.HasNativeRecordClass {
		t.Fatal("story foreground approximation leaked into canonical actor state")
	}
}

func TestNativeStagingFocusUsesOneStoryViewAfterLoadCH(t *testing.T) {
	g := storyStagingFixture(t)
	// A preceding battle may leave the generic cursor at an unrelated value.
	// Once LOADCH has published its six-field story view, focus must not mix
	// this stale pair with the story-visible cursor.
	g.curX, g.curY = 0, 0
	_, _, view, err := g.nativeFocusEndpoint(8, 7)
	if err != nil {
		t.Fatal(err)
	}
	if view.CursorX != 8 || view.CursorY != 7 ||
		view.VisibleCursorX != view.CursorX-view.CameraX ||
		view.VisibleCursorY != view.CursorY-view.CameraY {
		t.Fatalf("story focus view=%+v", view)
	}
}

func TestNativeStagingPresentFocusesXYThenMutatesStorySlotAtBridge(t *testing.T) {
	g := storyStagingFixture(t)
	spec := campaign.NativeStagingPresent{Slot: 2, X: 8, Y: 7}
	oldX, oldY := g.storyActors[spec.Slot].X, g.storyActors[spec.Slot].Y
	if err := g.startNativeStagingPresent(spec); err != nil {
		t.Fatal(err)
	}
	if g.curX != g.storyNativeMapView.CursorX || g.curY != g.storyNativeMapView.CursorY {
		t.Fatalf("focus owner cursor=(%d,%d), story=%+v", g.curX, g.curY, g.storyNativeMapView)
	}
	if g.focusJob == nil || g.focusJob.targetX != spec.X || g.focusJob.targetY != spec.Y || g.nativeUnitPresent != nil {
		t.Fatalf("focus=%#v present=%#v", g.focusJob, g.nativeUnitPresent)
	}
	for steps := 0; g.focusJob != nil && steps < 100; steps++ {
		g.stepFocusUnit()
	}
	if g.focusJob != nil || g.nativeUnitPresent == nil || g.curX != spec.X || g.curY != spec.Y {
		t.Fatalf("focus did not hand off: focus=%#v present=%#v cursor=(%d,%d)", g.focusJob, g.nativeUnitPresent, g.curX, g.curY)
	}
	if !g.hasStoryNativeMapView || g.storyNativeMapView.CursorX != spec.X || g.storyNativeMapView.CursorY != spec.Y {
		t.Fatalf("focus did not publish native story view: %#v", g.storyNativeMapView)
	}
	for g.nativeUnitPresent.frame < g.nativeUnitPresent.mutationAt {
		stepNativeUnitPresentNow(g)
	}
	if g.storyActors[spec.Slot].X != spec.X || g.storyActors[spec.Slot].Y != spec.Y {
		t.Fatalf("story slot did not mutate at bridge: %#v", g.storyActors[spec.Slot])
	}
	if oldX == spec.X && oldY == spec.Y {
		t.Fatal("fixture did not exercise a coordinate change")
	}
	for steps := 0; g.nativeUnitPresent != nil && steps < 100; steps++ {
		stepNativeUnitPresentNow(g)
	}
	if g.nativeUnitPresent != nil || g.loadErr != "" {
		t.Fatalf("staging did not finish: present=%#v err=%q", g.nativeUnitPresent, g.loadErr)
	}
}

func TestNativeStagingPresentPreflightFailureStartsNoFocus(t *testing.T) {
	g := storyStagingFixture(t)
	spec := campaign.NativeStagingPresent{Slot: 2, X: 8, Y: 7}
	beforeUnit := g.storyActors[spec.Slot]
	beforeCamX, beforeCamY, beforeCurX, beforeCurY := g.camX, g.camY, g.curX, g.curY
	g.nativeMapAssets.FDOTHER6[0x72].Pixels = nil
	if err := g.startNativeStagingPresent(spec); err == nil {
		t.Fatal("malformed staging assets were accepted")
	}
	if g.focusJob != nil || g.nativeUnitPresent != nil || !reflect.DeepEqual(g.storyActors[spec.Slot], beforeUnit) ||
		g.camX != beforeCamX || g.camY != beforeCamY || g.curX != beforeCurX || g.curY != beforeCurY {
		t.Fatal("failed staging preflight changed focus, story slot, or camera")
	}
}

type assertionError string

func (e assertionError) Error() string { return string(e) }
