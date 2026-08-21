package main

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func completeNativeCh28PostGame(t *testing.T) *Game {
	t.Helper()
	assets, field, seed := completeNativeMapFrameFixture(t)
	base := *seed.Units[0]
	units := make([]*battle.Unit, 76)
	for index := range units {
		clone := base
		clone.X, clone.Y = index%field.W, (index/field.W)%field.H
		clone.Group = 0
		if index >= 20 {
			clone.Group = 8
		}
		clone.MapSelectorSlot = 0
		clone.HasMapSelectorSlot = false
		clone.NativeMapPresentation = battle.NativeMapPresentationState{}
		clone.HasNativeMapPresentation = false
		units[index] = &clone
	}
	state := &battle.State{W: field.W, H: field.H}
	if err := state.AppendNativeMapSelectorBatch(units); err != nil {
		t.Fatal(err)
	}
	state.Roster = []*battle.Unit{{Group: 2}, {Group: 3}, {Group: 9}}
	state.NativeTileBlitModes = append([]byte(nil), field.NativeTileBlitModes...)
	if !state.MaterializeNativeMapHUDState(2, 3, 1) || !state.MaterializeNativeMapRangeMode(1) {
		t.Fatal("native HUD/range state materialization rejected")
	}
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil {
		t.Fatal(err)
	}
	g := &Game{nativeMapAssets: assets, m: field, st: state}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestNativeCh28PostPresentPrecomputesExactScheduleAndCommits(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	continued := 0
	if err := g.startNativeCh28PostPresent(func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	j := g.nativeCh28PostPresent
	if j == nil || len(j.vgaFrames) != 26 || len(j.stateFrames) != 26 || j.sfxAt != 13 {
		t.Fatalf("job=%+v", j)
	}
	if g.st.Units[19].HP != 5 || g.st.Units[20].HP != 0 ||
		g.st.Units[20].BattleFig != 0x7e || g.st.Units[20].NativeRecordByte8 != 0x7e {
		t.Fatalf("first published state did not include raw prelude: slot19=%+v slot20=%+v", g.st.Units[19], g.st.Units[20])
	}
	for steps := 0; g.nativeCh28PostPresent != nil && steps < 26; steps++ {
		g.nativeCh28PostPresent.drawn = true
		g.nativeCh28PostPresent.wait = 0
		g.stepNativeCh28PostPresent()
	}
	if g.nativeCh28PostPresent != nil || continued != 1 {
		t.Fatalf("job=%v continued=%d", g.nativeCh28PostPresent, continued)
	}
	if g.st.Units[20].NativeRecordByte5 != 1 || g.st.Units[75].NativeRecordByte5 != 1 {
		t.Fatalf("final inactive marks were not committed: slot20=%#x slot75=%#x", g.st.Units[20].NativeRecordByte5, g.st.Units[75].NativeRecordByte5)
	}
}

func TestNativeCh28PostPresentMissingLMIRejectsBeforeMutation(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	beforeHP := g.st.Units[20].HP
	g.nativeMapAssets.CommandHealDigits[fdother.NativeCh28PostLMIFirst] = fdother.LMI1Entry{}
	if err := g.startNativeCh28PostPresent(nil); err == nil {
		t.Fatal("missing required FDOTHER#5 entry was accepted")
	}
	if g.nativeCh28PostPresent != nil || g.st.Units[20].HP != beforeHP ||
		!bytes.Equal(beforeWork, g.nativeMapWork) || !bytes.Equal(beforeVGA, g.nativeMapVGA) {
		t.Fatal("failed preflight changed state or indexed buffers")
	}
}

func TestNativeCh28PostPresentFailureRollsBackStateAndBuffers(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	beforeWork, beforeVGA := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	beforeHP := g.st.Units[20].HP
	if err := g.startNativeCh28PostPresent(nil); err != nil {
		t.Fatal(err)
	}
	g.failNativeCh28PostPresent(bytes.ErrTooLarge)
	if g.nativeCh28PostPresent != nil || g.st.Units[20].HP != beforeHP ||
		!bytes.Equal(beforeWork, g.nativeMapWork) || !bytes.Equal(beforeVGA, g.nativeMapVGA) {
		t.Fatal("runtime presentation failure did not restore state and indexed buffers")
	}
}
