package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeActionDeathPresentationRetainsSettlementAndContinuesOnce(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	g.modernStoryPortraits = deathHDTestSet(g)
	actor, dead := g.st.Units[0], g.st.Units[1]
	actor.Acted, actor.Camp, g.sel = false, battle.Own, actor
	dead.NativeRecordByte5 = 0x80
	before := captureNativeActionDeathFlags(actor, dead)
	dead.ApplyHPDamage(dead.HP)
	g.queueNativeActionDeathPresentation(before)
	state, units := g.st, append([]*battle.Unit(nil), g.st.Units...)
	g.nativeRNGState, g.gold = 123, 77
	continued := 0
	g.finishSuccessfulUnitAction(actor, func() { continued++ })
	job := g.nativeCh28PostPresent
	if job == nil || job.kind != "action" || job.retainedState != state || continued != 0 || actor.Acted {
		t.Fatalf("missing synchronous action death: %s", g.loadErr)
	}
	if job.stateFrames[0].Units[1].NativeRecordByte5 != 0x80 ||
		job.stateFrames[13].Units[1].NativeRecordByte5 != 1 || len(job.hdFrames) != 26 {
		t.Fatal("private frames lost the exact pre-settlement flag or HD schedule")
	}
	for steps := 0; g.nativeCh28PostPresent != nil && steps < 26; steps++ {
		if g.st != state || !sameNativeDeathUnitPointers(g.st.Units, units) || dead.HP != 0 || dead.NativeRecordByte5 != 1 ||
			g.nativeRNGState != 123 || g.gold != 77 {
			t.Fatal("visual frame replaced live identities or replayed settlement")
		}
		g.nativeCh28PostPresent.drawn, g.nativeCh28PostPresent.wait = true, 0
		g.stepNativeCh28PostPresent()
	}
	g.stepNativeCh28PostPresent()
	if g.loadErr != "" || continued != 1 || !actor.Acted || g.nativeCh28PostPresent != nil ||
		len(g.nativeActionDeathBefore) != 0 || len(g.pendingDeathPrograms) != 0 {
		t.Fatalf("continuation: %s calls%d acted%v", g.loadErr, continued, actor.Acted)
	}
}

func sameNativeDeathUnitPointers(actual, expected []*battle.Unit) bool {
	if len(actual) != len(expected) {
		return false
	}
	for index := range actual {
		if actual[index] != expected[index] {
			return false
		}
	}
	return true
}

func TestNativeActionDeathPresentationRejectsBeforeInactiveWrite(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	actor, dead := g.st.Units[0], g.st.Units[1]
	dead.HP, dead.NativeRecordByte5 = 0, 0x80
	g.nativeMapAssets.CommandHealDigits[79].Pixels = nil
	state := g.st
	g.finishSuccessfulUnitAction(actor, nil)
	if g.loadErr == "" || g.nativeCh28PostPresent != nil || g.st != state || dead.NativeRecordByte5 != 0x80 || actor.Acted {
		t.Fatal("missing source was accepted or partially committed")
	}
}

func TestNativeCommandDamageDeathUsesWriterFlagWithoutPhysicalEntry(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	g := completeNativeCh28PostGame(t)
	actor, dead := g.st.Units[0], g.st.Units[1]
	dead.NativeRecordByte5 = 0x80
	result, seed, err := battle.ApplyNativeCommandDamage(dead, 1000, 100, 10, 1)
	if err != nil || !result.Hit || dead.HP != 0 || dead.NativeRecordByte5 != 1 {
		t.Fatal("controlled native command did not kill its target")
	}
	g.nativeRNGState = seed
	if !g.nativeActionDeathPresentationPending() || len(g.nativeActionDeathBefore) != 0 {
		t.Fatal("command death still depends on the physical entry snapshot")
	}
	continued := 0
	g.finishSuccessfulUnitAction(actor, func() { continued++ })
	job := g.nativeCh28PostPresent
	if job == nil || job.stateFrames[0].Units[1].NativeRecordByte5 != 0x80 {
		t.Fatalf("native command lost its real writer flag: %s", g.loadErr)
	}
	for steps := 0; g.nativeCh28PostPresent != nil && steps < 26; steps++ {
		g.nativeCh28PostPresent.drawn, g.nativeCh28PostPresent.wait = true, 0
		g.stepNativeCh28PostPresent()
	}
	if continued != 1 || g.nativeRNGState != seed || g.nativeActionDeathPresentationPending() {
		t.Fatal("command death repeated RNG or left a pending corpse")
	}
}

func TestNativeActionDeathPresentationSkipsOffscreenAndOldInactive(t *testing.T) {
	t.Setenv("FD2_MUTE", "1")
	for _, old := range []bool{false, true} {
		g := completeNativeCh28PostGame(t)
		actor, dead := g.st.Units[0], g.st.Units[1]
		dead.HP, dead.NativeRecordByte5 = 0, 1
		if !old {
			dead.NativeRecordByte5 = 0
			dead.NativeMapPresentation.X, dead.X = 250, 250
		}
		continued := 0
		g.finishSuccessfulUnitAction(actor, func() { continued++ })
		if g.loadErr != "" || g.nativeCh28PostPresent != nil || continued != 1 || dead.NativeRecordByte5 != 1 {
			t.Fatalf("old%v: empty presentation should not block: %s", old, g.loadErr)
		}
	}
}

func TestNativeTransientDeathOwnerRunsAfterMessageWithoutRewards(t *testing.T) {
	g := chapter13ResultGame(t)
	ackPresents(g)
	dead := g.st.Units[0]
	dead.HP, dead.MaxHP, dead.NativeRecordByte5 = 1, 100, 0
	dead.HasNativeRecordByte6, dead.NativeRecordByte6 = true, 2
	dead.NativeTransient = [6]byte{0, 0, 0, 2}
	if g.st.HasNativeRuntimeUnitProjection {
		t.Fatal("this explicit chapter constructor must exercise the typed path")
	}
	state, gold := g.st, g.gold
	continued := 0
	if err := g.beginNativeTransientPhases([]byte{2}, func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if dead.HP != 0 || dead.NativeRecordByte5 != 1 || dead.NativeTransient[3] != 2 ||
		g.nativeLevelUpDialogue == nil || g.nativeCh28PostPresent != nil || continued != 0 {
		t.Fatal("status death did not preserve the damage-message-first boundary")
	}
	for steps := 0; g.nativeCh28PostPresent == nil && steps < 1200; steps++ {
		ackPresents(g)
		g.stepNativeStoryDialogueProgress()
		g.stepNativeLevelUpDialogue(true)
		g.stepNativeClassUILifecycle(time.Time{})
		if g.loadErr != "" {
			t.Fatal(g.loadErr)
		}
	}
	job := g.nativeCh28PostPresent
	if job == nil || job.kind != "status" || !g.nativeTransientPresentationActive() || continued != 0 {
		t.Fatal("status message did not hand off to the synchronous death owner")
	}
	seed := g.nativeRNGState
	for steps := 0; g.nativeCh28PostPresent != nil && steps < 26; steps++ {
		g.nativeCh28PostPresent.drawn, g.nativeCh28PostPresent.wait = true, 0
		g.stepNativeCh28PostPresent()
	}
	if g.st != state || g.st.Units[0] != dead || continued != 1 || g.gold != gold || g.nativeRNGState != seed ||
		len(g.pendingDeathPrograms) != 0 || len(g.pendingNativeDeathRewards) != 0 || g.deathProgramKiller != nil {
		t.Fatal("status visual changed identities, RNG, rewards or phase continuation")
	}
}

func TestNativeTransientDeathPreflightRejectsBeforeHPTransaction(t *testing.T) {
	g := chapter13ResultGame(t)
	ackPresents(g)
	dead := g.st.Units[0]
	dead.HP, dead.MaxHP, dead.NativeRecordByte5 = 1, 100, 0
	dead.NativeTransient = [6]byte{0, 0, 0, 2}
	g.nativeMapAssets.CommandHealDigits[79].Pixels = nil
	state, seed, gold := g.st, g.nativeRNGState, g.gold
	if err := g.beginNativeTransientPhases([]byte{2}, nil); err == nil {
		t.Fatal("missing death frame was accepted")
	}
	if g.st != state || g.st.Units[0] != dead || dead.HP != 1 || dead.NativeRecordByte5 != 0 ||
		dead.NativeTransient[3] != 2 || g.nativeRNGState != seed || g.gold != gold ||
		g.nativeCh28PostPresent != nil || g.nativeLevelUpDialogue != nil {
		t.Fatal("death preflight failure published HP, counters or presentation")
	}
}

func TestNativeTransientDeathPreviewMatchesFocusWithoutPublishingOrTracing(t *testing.T) {
	g := chapter13ResultGame(t)
	ackPresents(g)
	candidate, err := cloneNativeCh28PostState(g.st)
	if err != nil {
		t.Fatal(err)
	}
	candidate.Units[0].SetMapPlacement(1, 1, 0)
	candidate.Units[1].SetMapPlacement(candidate.W-2, candidate.H-2, 0)
	damage := []battle.NativeTransientDamage{{Unit: candidate.Units[0], Slot: 0}, {Unit: candidate.Units[1], Slot: 1}}
	state, view, hud, seed := g.st, g.st.NativeMapViewState, g.st.NativeMapHUDState, g.nativeRNGState
	work, vga := append([]byte(nil), g.nativeMapWork...), append([]byte(nil), g.nativeMapVGA...)
	oldHook := battle.NativeFocusTraceHook
	defer func() { battle.NativeFocusTraceHook = oldHook }()
	traces := 0
	battle.NativeFocusTraceHook = func(int, int, battle.NativeMapViewState) { traces++ }
	preview, err := g.nativeTransientDeathPreview(candidate, damage)
	if err != nil {
		t.Fatal(err)
	}
	if traces != 0 || g.st != state || g.st.NativeMapViewState != view || g.st.NativeMapHUDState != hud ||
		g.nativeRNGState != seed || !bytes.Equal(g.nativeMapWork, work) || !bytes.Equal(g.nativeMapVGA, vga) {
		t.Fatal("private preview published state or emitted a real focus trace")
	}
	control := *g
	control.st, err = cloneNativeCh28PostState(candidate)
	if err != nil {
		t.Fatal(err)
	}
	control.nativeMapWork, control.nativeMapVGA = work, vga
	for _, event := range damage {
		if err := control.focusNativeTransientRecord(control.st.Units[event.Slot]); err != nil {
			t.Fatal(err)
		}
	}
	if traces == 0 || control.st.NativeMapViewState != preview.st.NativeMapViewState ||
		control.st.NativeMapHUDState != preview.st.NativeMapHUDState ||
		!bytes.Equal(control.nativeMapWork, preview.nativeMapWork) || !bytes.Equal(control.nativeMapVGA, preview.nativeMapVGA) {
		t.Fatal("preview differs from the actual ordered damage-message focus")
	}
}
