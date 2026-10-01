package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 直接進章、明示raw測試狀態只驗E1；同槽正常玩家動作由章重播另驗。
func chapter13ResultGame(t *testing.T) *Game {
	t.Helper()
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "animations/fdother_079_pending_code1/bank.json")); err != nil {
		t.Skip(err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_SEED", "4")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_CAMP_NODE", "battle_ch13")
	t.Setenv("FD2_NATIVE_SAVE", filepath.Join(t.TempDir(), "FD2.SAV"))
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	g.st.Turn, g.st.NativeRoundCounter = 4, 4
	actions := g.sc.TriggerActions(g.st, "on_turn_end", "")
	if len(actions) == 0 || actions[0].Type != "spawn_group" {
		t.Fatal("第四回合援軍動作缺漏")
	}
	if _, _, err := g.sc.ExecuteActionChecked(g.st, actions[0]); err != nil {
		t.Fatal(err)
	}
	g.st.Turn, g.st.NativeRoundCounter = 6, 6
	return g
}

func TestNativeChapterResultReadsSecondRuleAfterFirstDialogue(t *testing.T) {
	g := chapter13ResultGame(t)
	for i := 15; i <= 26; i++ {
		g.st.Units[i].NativeRecordByte5 |= 1
	}
	g.checkResult()
	firstJob, firstEvent := g.nativeChapterResult, g.battleEvent
	if firstJob == nil || firstJob.nextRule != 1 || firstEvent == nil || g.result != "" || g.nativeDefeat != nil {
		t.Fatalf("第一列尚未收框即判敗或讀下一列：%s", g.loadErr)
	}
	if !pump(t, g, journeyStoryFrames, func() bool {
		return len(g.dialog) > 0 && g.dialog[0].NativeDialogue != nil && g.dialog[0].NativeDialogue.StringIndex == 10
	}) {
		t.Fatalf("第一列未顯示原始文字10：%s", g.loadErr)
	}
	// 在對白擁有期間改測試狀態，證明下一條件不是預先計算。
	g.st.Units[59].NativeRecordByte5 |= 1
	g.checkResult()
	if g.nativeChapterResult != firstJob || g.battleEvent != firstEvent {
		t.Fatal("對白期間重複啟動結果判定")
	}
	if !pump(t, g, journeyStoryFrames, func() bool {
		if g.nativeChapterResult != nil && g.nativeChapterResult.nextRule == 2 &&
			len(g.dialog) > 0 && g.dialog[0].NativeDialogue != nil && g.dialog[0].NativeDialogue.StringIndex == 2 {
			return true
		}
		if len(g.dialog) > 0 && storyEnterReady(g) {
			g.handleBattleEventDialogueInput(true)
		}
		return false
	}) {
		t.Fatalf("第一列收框後沒有重讀第二列：%s", g.loadErr)
	}
	if g.result != "" || g.nativeDefeat != nil {
		t.Fatal("第二列對白尚未收框就開始敗北")
	}
	if !pump(t, g, journeyStoryFrames, func() bool {
		if g.nativeDefeat != nil {
			return true
		}
		if len(g.dialog) > 0 && storyEnterReady(g) {
			g.handleBattleEventDialogueInput(true)
		}
		return false
	}) {
		t.Fatalf("全部對白結束後未交接原生敗北：%s", g.loadErr)
	}
	if g.result != "lose" || !reflect.DeepEqual(g.nativeResultMatchedRules, []string{"initial_allies_inactive", "late_record59_inactive"}) {
		t.Fatalf("順序或結果不符：%s %v", g.result, g.nativeResultMatchedRules)
	}
	g.clearDefeatedBattleTransientState()
	if g.nativeChapterResult != nil || g.nativeResultMatchedRules != nil {
		t.Fatal("戰鬥重設留下結果擁有者")
	}
}

func TestNativeChapterResultMissingDialogueStateStopsBeforeDefeat(t *testing.T) {
	g := chapter13ResultGame(t)
	g.st.Units[59].NativeRecordByte5 |= 1
	g.nativeMapAssets = nil
	g.checkResult()
	if g.loadErr == "" || !g.startupBlocked || g.result != "" || g.nativeDefeat != nil {
		t.Fatal("原生對白缺漏卻跳過對白進入敗北")
	}
	if g.sc.NativeResultRules[1].Code != 1 || g.st.Units[0].Camp != battle.Own {
		t.Fatal("錯誤流程改寫正式規則或我方狀態")
	}
}

func TestChapter15SilentResultHandsOffToNativeDefeat(t *testing.T) {
	// 直接進章與注入raw測試條件只驗E1；普通輸入仍由章重播另驗。
	t.Setenv("FD2_ASSET_PACK", filepath.Clean("../../generated-assets/fd2-original-b97caf22"))
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_SEED", "4")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_CAMP_NODE", "battle_ch15")
	t.Setenv("FD2_NATIVE_SAVE", filepath.Join(t.TempDir(), "FD2.SAV"))
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if len(g.sc.NativeResultRules) != 1 || len(g.sc.NativeResultRules[0].Actions) != 0 {
		t.Fatal("canonical正式入口未載入無對白結果規則")
	}
	g.st.Units[65].NativeRecordByte5 |= 1
	g.checkResult()
	if g.result != "" || g.nativeDefeat != nil || g.loadErr != "" {
		t.Fatalf("其他友軍倒下被判敗：%s", g.loadErr)
	}
	// 無對白列不產生對白owner；敗北提示沿用FDOTHER79。
	g.st.Units[64].NativeRecordByte5 |= 1
	g.checkResult()
	if g.result != "lose" || g.nativeDefeat == nil || g.battleEvent != nil ||
		g.nativeChapterResult != nil || g.loadErr != "" ||
		!reflect.DeepEqual(g.nativeResultMatchedRules, []string{"record64_inactive"}) {
		t.Fatalf("無對白結果沒有直接交接原生敗北：result=%q error=%s", g.result, g.loadErr)
	}
	// 純結果owner沒有對白素材也能判定；正式敗北owner仍要求戰場／色盤。
	probe := *g
	probe.nativeChapterResult, probe.nativeDefeat, probe.battleEvent = nil, nil, nil
	probe.result, probe.loadErr = "", ""
	probe.camp, probe.nativeMapAssets = nil, nil
	probe.beginNativeChapterResult()
	if probe.loadErr != "" || probe.result != "lose" || probe.battleEvent != nil {
		t.Fatal("無對白結果列錯誤要求戰場對白素材")
	}
}

func TestChapter13Event7FinishesActingBeforeDialogueAndContinuation(t *testing.T) {
	// 直接進章與回合設定僅驗RUNTIME-E1；正常章內輸入由#61章重播驗。
	g := chapter13ResultGame(t)
	g.st.Turn, g.st.NativeRoundCounter = 9, 9
	actions := g.sc.TriggerActions(g.st, "on_turn_end", "")
	want := []string{"pan", "spawn_group", "native_acting", "reset_pose", "dialogue"}
	if len(actions) != len(want) {
		t.Fatalf("event7動作數=%d，預期%d", len(actions), len(want))
	}
	for i, kind := range want {
		if actions[i].Type != kind || actions[i].Camp != "enemy" ||
			actions[i].NativeEventID == nil || *actions[i].NativeEventID != 7 {
			t.Fatalf("event7動作%d的順序／camp／來源不符：%+v", i, actions[i])
		}
	}
	if actions[0].Grid == nil || *actions[0].Grid != [2]int{27, 5} ||
		len(actions[1].NativeSpawns) != 1 || actions[1].NativeSpawns[0].Source != "0x34d91" ||
		actions[1].NativeSpawns[0].RawPlacementGate == nil || *actions[1].NativeSpawns[0].RawPlacementGate != 1 ||
		actions[2].NativeActing == nil || actions[2].NativeActing.Resource != 46 {
		t.Fatal("event7缺少原始鏡頭／配置閘門／ACTING46")
	}
	continued, sawActing := false, false
	g.startBattleEvent(actions, func() { continued = true })
	if !pump(t, g, journeyStoryFrames, func() bool {
		if g.actJob != nil {
			sawActing = true
			if len(g.dialog) != 0 || continued {
				t.Fatal("ACTING未結束即交出對白或敵方續行")
			}
		}
		return len(g.dialog) > 0
	}) {
		t.Fatalf("event7未走完演出到文字8：%s", g.loadErr)
	}
	if !sawActing || continued || g.actJob != nil || len(g.st.Units) != 72 ||
		g.dialog[0].NativeDialogue == nil || g.dialog[0].NativeDialogue.StringIndex != 8 {
		t.Fatalf("event7演出／收框交接不完整：%s", g.loadErr)
	}
	for i := 60; i <= 71; i++ {
		u := g.st.Units[i]
		if u.Group != 2 || !u.HasNativeMapPresentation || u.NativeMapPresentation.Pose != 0 || u.NativeMapPresentation.Motion != 0 {
			t.Fatalf("event7的原生record%d未生成或姿態未收尾：%+v", i, u)
		}
	}
	if !pump(t, g, journeyStoryFrames, func() bool {
		if len(g.dialog) > 0 && storyEnterReady(g) {
			g.handleBattleEventDialogueInput(true)
		}
		return continued
	}) {
		t.Fatalf("event7對白完整收框後未續行：%s", g.loadErr)
	}
	if g.battleEvent != nil || len(g.dialog) != 0 || len(g.sc.TriggerActions(g.st, "on_turn_end", "")) != 0 {
		t.Fatal("event7留下對白／擁有者或重複觸發")
	}
}
