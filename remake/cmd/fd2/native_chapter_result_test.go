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
