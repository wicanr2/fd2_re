package main

import (
	"os"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 第五章首領的死亡效果 2:19（0x345AB 同型的死亡程式）把執行期記錄 7..36 改成模式 7，
// 台詞「糟啦！首領被打倒了！快逃啊！」。撤退者走回 raw +0x35/+0x36 後由 0x32975 離場，
// 最後一名離場時原版在同一個敵方階段判勝（第四章同型機制見 56「第四章撤退離場與
// 勝負判定」）。這個受控抽測走正式玩家攻擊打倒首領，之後只按 END，不再出手。
//
// 修改範圍：首領 HP 改成 1 並移到我方旁邊；不在撤退範圍內的其餘敵軍 HP 歸零。
// 結果是 RUNTIME-E1，不是一般玩家路徑。
func TestChapterFiveBossDeathRetreatEndsBattleProbe(t *testing.T) {
	save := os.Getenv("FD2_CH05_PROBE_SAVE")
	if save == "" {
		t.Skip("FD2_CH05_PROBE_SAVE: 第五章建構槽")
	}
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_SEED", "4")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if err := g.loadNativeGameFromSlot(save, 0); err != nil {
		t.Fatal(err)
	}
	g.camp.Cur = "battle_ch05"
	g.enterNode()
	if !pump(t, g, 18000, func() bool {
		if len(g.dialog) > 0 && g.battleEvent == nil && g.nativeLevelUpDialogue == nil {
			g.dlgAdvance()
		}
		return deathBattleIdle(g)
	}) {
		t.Fatal(ch01Blockers(g))
	}

	var boss *battle.Unit
	bossIndex := -1
	for index, u := range g.st.Units {
		if u.Camp == battle.Enemy && u.OnField && u.Alive() && u.HasNativeRecordDeathEffect &&
			u.NativeRecordDeathEffect[0] == 2 && u.NativeRecordDeathEffect[1] == 19 {
			boss, bossIndex = u, index
		}
	}
	if boss == nil {
		t.Fatal("chapter five boss with death effect 2:19 not found")
	}
	inRetreatRange := func(index int) bool { return index >= 7 && index <= 36 }
	for index, u := range g.st.Units {
		if u.Camp == battle.Enemy && u.OnField && u.Alive() && u != boss && !inRetreatRange(index) {
			u.HP = 0
		}
	}
	g.st.MarkNativeDeadRecords()

	actor := g.st.Units[0]
	var cell battle.Cell
	found := false
	for _, p := range []battle.Cell{{X: actor.X + 1, Y: actor.Y}, {X: actor.X - 1, Y: actor.Y}, {X: actor.X, Y: actor.Y + 1}, {X: actor.X, Y: actor.Y - 1}} {
		if p.X >= 0 && p.X < g.st.W && p.Y >= 0 && p.Y < g.st.H && g.st.UnitAt(p.X, p.Y) == nil {
			cell, found = p, true
			break
		}
	}
	if !found || !actor.Alive() || !actor.OnField {
		t.Fatalf("invalid setup: adjacent=%v actor=%+v", found, actor)
	}
	boss.ApplyHPDamage(boss.HP - 1)
	boss.SetMapPlacement(cell.X, cell.Y, boss.Dir)
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	if !g.positionScreenshotCursor(actor.X, actor.Y) {
		t.Fatal("cursor did not reach actor")
	}
	g.confirm()
	if g.sel != actor {
		t.Fatal("selection rejected")
	}
	g.confirm()
	if !pump(t, g, 240, func() bool { return g.ring }) || !nativeActionSelectable(g.actionOverlayAvailability(), 0) {
		t.Fatal("attack not available")
	}
	g.ringSel = 0
	if !closeRing(t, g, g.beginPlayerAttackTargeting) || !g.positionScreenshotCursor(boss.X, boss.Y) {
		t.Fatal("attack targeting failed")
	}
	g.confirm()
	if g.atk == nil || boss.HP != 0 {
		t.Fatalf("normal player attack did not kill the boss: animation=%v hp=%d error=%s", g.atk != nil, boss.HP, g.loadErr)
	}
	if !pump(t, g, 18000, func() bool {
		if g.nativeDeathRewardMessageAwaitingKey() {
			g.acknowledgeNativeDeathRewardMessage()
		}
		if g.nativeLevelUpDialogue != nil {
			g.stepNativeLevelUpDialogue(true)
		}
		if len(g.dialog) > 0 && g.battleEvent == nil && g.nativeLevelUpDialogue == nil {
			g.dlgAdvance()
		}
		return g.atk == nil && !g.nativeActionDeathPresentationPending() && !g.nativeDeathProgramsPending() &&
			g.nativeLevelUpDialogue == nil && g.nativeClassUIJob == nil && !g.transientUI && deathBattleIdle(g)
	}) {
		t.Fatalf("boss death program did not settle: result=%q atk=%v actionDeath=%v deathProg=%v levelUp=%v classJob=%v transient=%v idle=%v sel=%v ring=%v ai=%v dialog=%d msg=%q",
			g.result, g.atk != nil, g.nativeActionDeathPresentationPending(), g.nativeDeathProgramsPending(), g.nativeLevelUpDialogue != nil, g.nativeClassUIJob != nil, g.transientUI, deathBattleIdle(g), g.sel != nil, g.ring, g.aiBusy, len(g.dialog), g.msg)
	}

	var retreaters []*battle.Unit
	for index, u := range g.st.Units {
		if inRetreatRange(index) && index != bossIndex && u.Camp == battle.Enemy && u.OnField && u.Alive() {
			if u.NativeRecordByte34&0x0f != 7 {
				t.Fatalf("record %d did not switch to retreat mode: +0x34=%#x", index, u.NativeRecordByte34)
			}
			retreaters = append(retreaters, u)
		}
	}
	if len(retreaters) == 0 || g.result != "" {
		t.Fatalf("boss death must leave counted retreaters: retreaters=%d result=%q", len(retreaters), g.result)
	}
	t.Logf("boss record %d down at turn %d; %d retreaters switched to mode 7", bossIndex, g.st.Turn, len(retreaters))

	for round := 0; round < 12 && g.result == ""; round++ {
		turn := g.st.Turn
		g.endTurn()
		if !pump(t, g, 60000, func() bool {
			if g.nativeLevelUpDialogue != nil {
				g.stepNativeLevelUpDialogue(true)
			} else if len(g.dialog) > 0 && g.battleEvent == nil {
				g.dlgAdvance()
			}
			return g.result != "" || (g.st.Turn > turn && deathBattleIdle(g) && g.nativeLevelUpDialogue == nil && !g.transientUI)
		}) {
			t.Fatalf("round after turn %d did not settle: %s", turn, ch01Blockers(g))
		}
		departed, dead, left := 0, 0, 0
		for _, u := range retreaters {
			switch {
			case !u.Alive():
				dead++
			case u.NativeRecordByte5&1 != 0:
				if u.OnField {
					t.Fatalf("departed retreater still counted on field at (%d,%d)", u.X, u.Y)
				}
				departed++
			default:
				left++
			}
		}
		t.Logf("after END at turn %d: departed=%d dead=%d still walking=%d result=%q turn=%d", turn, departed, dead, left, g.result, g.st.Turn)
		if g.result != "" {
			if g.result != "win" || left != 0 || g.st.Turn != turn {
				t.Fatalf("retreat victory: result=%q walking=%d turn %d→%d", g.result, left, turn, g.st.Turn)
			}
		}
	}
	if g.result != "win" {
		t.Fatalf("retreat never ended the battle: result=%q", g.result)
	}
	if !pump(t, g, 16000, func() bool { return resultInputReachable(g) }) {
		t.Fatalf("victory confirmation not reachable: %s", ch01Blockers(g))
	}
	if !g.confirmBattleResult() || g.camp.NodeID() != "postbattle_ch05_persist" {
		t.Fatalf("win handoff: %s %s", g.camp.NodeID(), g.loadErr)
	}
}
