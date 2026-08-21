package main

import (
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestEvent75SuccessfulActionCommitsOnlyAfterEditableDialogue(t *testing.T) {
	st, err := battle.Load(assetPath("assets/maps/map28/map28_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch29.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sc.SetupChecked(st); err != nil {
		t.Fatal(err)
	}
	st.NativeRoundCounter = 8
	actor := &battle.Unit{
		X: 15, Y: 21,
		HasNativeRecordByte6: true, NativeRecordByte6: 1,
		HasNativeRecordByte8: true, NativeRecordByte8: 9,
	}
	g := &Game{st: st, sc: sc}
	g.finishSuccessfulUnitAction(actor, nil)
	if g.battleEvent == nil || len(g.dialog) != 1 || actor.Acted ||
		st.NativeEventState[16] != 0 || st.NativeTurnEventControls[0].Turn != 0xff {
		t.Fatalf("event75 started job=%v dialogue=%d acted=%v state16=%d row0=%#v", g.battleEvent != nil, len(g.dialog), actor.Acted, st.NativeEventState[16], st.NativeTurnEventControls[0])
	}
	for index := 0; index < 5; index++ {
		g.dialog = nil
		g.advanceBattleEvent()
	}
	if g.battleEvent != nil || !actor.Acted || st.NativeEventState[16] != 4 ||
		st.NativeEventState[17] != 1 ||
		st.NativeTurnEventControls[0] != (battle.NativeTurnEventControl{Turn: 8, EventID: 74, RawCamp: 0}) ||
		st.NativeTurnEventControls[1] != (battle.NativeTurnEventControl{Turn: 9, EventID: 76, RawCamp: 2}) {
		t.Fatalf("event75 completion job=%v acted=%v state=%v rows=%#v", g.battleEvent != nil, actor.Acted, st.NativeEventState[16:18], st.NativeTurnEventControls[:2])
	}
}

func TestEvent75MismatchUsesTriggerRawByte7AndDoesNotActivate(t *testing.T) {
	st, err := battle.Load(assetPath("assets/maps/map28/map28_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch29.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sc.SetupChecked(st); err != nil {
		t.Fatal(err)
	}
	actor := &battle.Unit{
		X: 15, Y: 21, BattleFig: 23, HasBattleFig: true,
		HasNativeRecordByte6: true, NativeRecordByte6: 1,
		HasNativeRecordByte8: true, NativeRecordByte8: 8,
	}
	g := &Game{st: st, sc: sc}
	g.finishSuccessfulUnitAction(actor, nil)
	if len(g.dialog) != 1 || g.dialog[0].Speaker != 23 || actor.Acted {
		t.Fatalf("event75 mismatch dialogue=%#v acted=%v", g.dialog, actor.Acted)
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if !actor.Acted || st.NativeEventState[16] != 0 || st.NativeEventState[17] != 0 ||
		st.NativeTurnEventControls[0].Turn != 0xff || st.NativeTurnEventControls[1].Turn != 0xff {
		t.Fatalf("event75 mismatch activated state=%v rows=%#v", st.NativeEventState[16:18], st.NativeTurnEventControls[:2])
	}
}
