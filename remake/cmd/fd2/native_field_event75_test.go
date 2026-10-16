package main

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestEvent75PersistentLoadCHPreservesRawIdentitySource(t *testing.T) {
	if os.Getenv("FD2_ORIGINAL_FDOTHER") == "" {
		t.Skip("正常LOADCH揭幕需要唯讀原版素材")
	}
	g := newChapter29RuntimeBattle(t)
	source, ok := g.partyRoster[9]
	if !ok || !source.HasNativeRecordByte8 || source.NativeRecordByte8 != 9 {
		t.Fatal("真實persistent constructor沒有悠妮raw8來源")
	}
	// 此局部fixture從story進場，未走19人選擇前置，按persistent身份找悠妮。
	var actor *battle.Unit
	for _, unit := range g.st.Units {
		if unit != nil && unit.HasNativeIdentity && unit.NativeIdentity == 9 {
			actor = unit
			break
		}
	}
	if actor == nil {
		t.Fatal("LOADCH沒有持續隊伍的悠妮")
	}
	if !actor.HasNativeRecordByte6 || actor.NativeRecordByte6 != 2 ||
		!actor.HasNativeRecordByte8 || actor.NativeRecordByte8 != 9 {
		t.Fatalf("LOADCH actor來源遺失：raw6=%d/%v raw8=%d/%v", actor.NativeRecordByte6,
			actor.HasNativeRecordByte6, actor.NativeRecordByte8, actor.HasNativeRecordByte8)
	}
	// 局部consumer測試移至事件格；正常移動可達性由完整章重播驗證。
	actor.X, actor.Y = 15, 21
	plan, err := battle.PlanNativeFieldEvent75(g.st, actor, 15, 21)
	if err != nil || !plan.Activate || plan.TextIndex != 1 {
		t.Fatalf("persistent→LOADCH→75 plan=%+v err=%v", plan, err)
	}
	// 正規化identity不能替代已遺失的raw來源，也不能保留目的端舊旗標。
	source.HasNativeRecordByte8 = false
	applyPersistentStats(actor, &source)
	if _, err := battle.PlanNativeFieldEvent75(g.st, actor, 15, 21); err == nil {
		t.Fatal("缺raw8來源卻由NativeIdentity推造通過")
	}
}

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
	attachOfficialTestLocale(t, g, "zh-Hant")
	g.finishSuccessfulUnitAction(actor, nil)
	// 0x13512 先設 bit7（Acted），0x1198A 的事件對話期間 state／rows 都還沒提交。
	if g.battleEvent == nil || len(g.dialog) != 1 || !actor.Acted ||
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
	attachOfficialTestLocale(t, g, "zh-Hant")
	g.finishSuccessfulUnitAction(actor, nil)
	if len(g.dialog) != 1 || g.dialog[0].Speaker != 23 || !actor.Acted {
		t.Fatalf("event75 mismatch dialogue=%#v acted=%v", g.dialog, actor.Acted)
	}
	g.dialog = nil
	g.advanceBattleEvent()
	if !actor.Acted || st.NativeEventState[16] != 0 || st.NativeEventState[17] != 0 ||
		st.NativeTurnEventControls[0].Turn != 0xff || st.NativeTurnEventControls[1].Turn != 0xff {
		t.Fatalf("event75 mismatch activated state=%v rows=%#v", st.NativeEventState[16:18], st.NativeTurnEventControls[:2])
	}
}
