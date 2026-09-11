package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// deathBattleIdle 是「沒有任何演出、事件、對白或 AI 在跑」。死亡程式不需要玩家的
// 操作權，只需要執行器是空的。
func deathBattleIdle(g *Game) bool {
	return g.st != nil && g.result == "" && !g.aiBusy && g.battleEvent == nil &&
		g.nativeTurnStaging == nil && len(g.dialog) == 0 && g.walk == nil && g.atk == nil &&
		!g.ring && g.spawnIntroTransition == nil && g.indexedTransition == nil &&
		g.nativeUnitPresent == nil && g.actJob == nil && g.camPan == nil && g.focusJob == nil
}

// loadChapterOneDeathBattle 直接進第一關戰場，推到執行器空下來。
func loadChapterOneDeathBattle(t *testing.T) *Game {
	t.Helper()
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_CAMP_NODE", journeyBattleNode)
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.st == nil || g.sc == nil {
		t.Fatal("第一關戰場或劇本沒有建立")
	}
	// 直接跳節點時劇本的開局事件留下一句一般對白（不是戰場事件對白，pump 不替它
	// 按鍵）；它與死亡程式無關，逐句推掉。
	if !pump(t, g, 20000, func() bool {
		if len(g.dialog) > 0 && g.battleEvent == nil {
			g.dlgAdvance()
		}
		return deathBattleIdle(g)
	}) {
		t.Fatalf("第一關的開場工作沒有收掉\n阻塞：%s", ch01Blockers(g))
	}
	return g
}

// runDeathAndCollect 讓 dead 在 killer 的行動裡倒下，走正式的收尾路徑，記下死亡程式
// 播過的每一段原生對白。
func runDeathAndCollect(t *testing.T, g *Game, dead, killer *battle.Unit) (lines []string, resultDuringLines string) {
	t.Helper()
	dead.ApplyHPDamage(dead.HP)
	g.awardDeathReward(dead, killer)
	g.checkResult()
	if g.result != "" {
		t.Fatalf("死亡程式還沒跑就判了勝負：%q", g.result)
	}
	g.finishSuccessfulUnitAction(killer, nil)
	done := pump(t, g, 20000, func() bool {
		if len(g.dialog) > 0 {
			if layout := g.dialog[len(g.dialog)-1].NativeDialogue; layout != nil {
				key := fmt.Sprintf("%s#%d.%d", layout.SourceDAT, layout.StringIndex, layout.Utterance)
				if n := len(lines); n == 0 || lines[n-1] != key {
					lines = append(lines, key)
					if g.result != "" {
						resultDuringLines = g.result
					}
				}
			}
		}
		return !g.nativeDeathProgramsPending() && g.battleEvent == nil && len(g.dialog) == 0
	})
	if !done {
		t.Fatalf("死亡程式沒有跑完\n阻塞：%s", ch01Blockers(g))
	}
	return lines, resultDuringLines
}

// 第一關海盜頭目的 [3, 8]：0x1AA1D 型態 3 播 FDTXT_001 第 8 句。勝負在輸入迴圈才判，
// 所以最後一個敵人是頭目時，死亡台詞播完才進勝利。
func TestNativeDeathTextPlaysBeforeVictory(t *testing.T) {
	g := loadChapterOneDeathBattle(t)
	before := len(g.st.Units)
	if n, err := g.st.AppendGroupWithNativePlacement(5, 0); err != nil || n <= 0 {
		t.Fatalf("放不上群組 5：%d %v", n, err)
	}
	var boss *battle.Unit
	for _, u := range g.st.Units[before:] {
		if kind, value, ok := battle.NativeDeathEffectOf(u); ok && kind == 3 && value == 8 {
			boss = u
		}
	}
	if boss == nil {
		t.Fatal("群組 5 裡找不到死亡效果 3:8 的海盜頭目")
	}
	var killer *battle.Unit
	for _, u := range g.st.Units {
		if u.Camp == battle.Own && u.Alive() && u.OnField {
			killer = u
			break
		}
	}
	for _, u := range g.st.Units {
		if u.Camp == battle.Enemy && u != boss {
			u.HP = 0
			if u.HasNativeRecordByte5 {
				u.NativeRecordByte5 = 1
			}
		}
	}
	g.st.PendingGroups = map[int]bool{}

	lines, during := runDeathAndCollect(t, g, boss, killer)
	if fmt.Sprint(lines) != "[FDTXT_001#8.0]" {
		t.Fatalf("死亡台詞 %v，應只有 FDTXT_001 第 8 句", lines)
	}
	if during != "" {
		t.Fatalf("台詞還在播就判了 %q", during)
	}
	if g.result != "win" {
		t.Fatalf("台詞播完應判勝利，結果 %q", g.result)
	}
}

// 事件 4（0x343E2）：哈諾倒下時記錄 13（哈瓦特）的 +6 從 2 改成 1，交給 AI；接著
// 播第 7 句的四段。記錄 13 寫死在處理器裡，這條同時驗證第一關的記錄順序。
func TestNativeDeathEventFourTurnsFatherToAlly(t *testing.T) {
	g := loadChapterOneDeathBattle(t)
	if len(g.st.Units) != 12 {
		t.Fatalf("第一關開局應有 12 筆 runtime 記錄（原版 FD2.SAV 第 1 回合），現在 %d", len(g.st.Units))
	}
	for _, group := range []int{3, 7} {
		if n, err := g.st.AppendGroupWithNativePlacement(group, 0); err != nil || n != 1 {
			t.Fatalf("放不上群組 %d：%d %v", group, n, err)
		}
	}
	hano, father := g.st.Units[12], g.st.Units[13]
	if kind, value, ok := battle.NativeDeathEffectOf(hano); !ok || kind != 2 || value != 4 {
		t.Fatalf("記錄 12 的死亡效果 %d:%d，應為哈諾的 2:4", kind, value)
	}
	if !father.HasNativeRecordByte6 || father.NativeRecordByte6 != 2 || father.Camp != battle.Own {
		t.Fatalf("記錄 13 開始時應是我方（raw 2）：%d %v", father.NativeRecordByte6, father.Camp)
	}
	var killer *battle.Unit
	for _, u := range g.st.Units {
		if u.Camp == battle.Enemy && u.Alive() && u.OnField {
			killer = u
			break
		}
	}

	lines, _ := runDeathAndCollect(t, g, hano, killer)
	want := "[FDTXT_001#7.0 FDTXT_001#7.1 FDTXT_001#7.2 FDTXT_001#7.3]"
	if fmt.Sprint(lines) != want {
		t.Fatalf("死亡對白 %v，應為 %s", lines, want)
	}
	if father.Camp != battle.Ally || father.NativeRecordByte6 != 1 {
		t.Fatalf("哈瓦特應轉成友軍：camp=%v raw=%d", father.Camp, father.NativeRecordByte6)
	}
}

// 找不到死亡程式時整段停下並說明是哪一個效果，不默默略過。
func TestNativeDeathProgramMissingFailsClosed(t *testing.T) {
	g := loadChapterOneDeathBattle(t)
	var dead, killer *battle.Unit
	for _, u := range g.st.Units {
		if u.Camp == battle.Enemy && dead == nil {
			dead = u
		}
		if u.Camp == battle.Own && killer == nil {
			killer = u
		}
	}
	dead.HasNativeRecordDeathEffect = false
	dead.DeathEffect = &battle.DeathEffect{Type: 2, Value: 64}
	dead.ApplyHPDamage(dead.HP)
	g.awardDeathReward(dead, killer)
	if !g.runPendingDeathPrograms(func() { t.Fatal("缺程式卻繼續往下") }) {
		t.Fatal("有待跑的死亡程式卻沒有執行")
	}
	if !strings.Contains(g.loadErr, "2:64") {
		t.Fatalf("錯誤訊息應指出 2:64：%q", g.loadErr)
	}
}

// 物品與金錢只給原版陣營 2 的擊殺者（0x1AC7B、0x1AB95 `cmp byte [esi+6], 2`）。
func TestNativeDeathRewardOnlyForPlayerKiller(t *testing.T) {
	g := &Game{}
	ally := &battle.Unit{Camp: battle.Ally, NativeRecordByte6: 1, HasNativeRecordByte6: true}
	g.grantNativeDeathReward(1, 1000, ally)
	g.grantNativeDeathReward(1, 1000, nil)
	if g.gold != 0 {
		t.Fatalf("友軍或狀態致死不該拿到金錢：%d", g.gold)
	}
	own := &battle.Unit{Camp: battle.Own, NativeRecordByte6: 2, HasNativeRecordByte6: true}
	g.grantNativeDeathReward(1, 1000, own)
	if g.gold != 1000 {
		t.Fatalf("我方擊殺應拿到 1000，現在 %d", g.gold)
	}
}

// 逐章實跑：每一章直接進戰場，找到帶該死亡效果的單位（還沒登場就把它的群組放上
// 來），由我方擊倒，走正式收尾路徑，確認程式跑完且沒有執行期錯誤。這條不比對
// 畫面，只擋「某章一倒下頭目就停住」這種串接缺口。
func TestNativeDeathProgramsRunInEveryChapter(t *testing.T) {
	// 每個子測試載入一整套遊戲，七十幾次累積會超過 4g 容器；一般回歸跳過，要跑時
	// 分批：FD2_DEATH_SMOKE=1 go test ./cmd/fd2 -run 'TestNativeDeathProgramsRunInEveryChapter/(ch0[1-9])'
	if os.Getenv("FD2_DEATH_SMOKE") == "" {
		t.Skip("需要 FD2_DEATH_SMOKE=1（逐章載入戰場，要分批跑）")
	}
	for chapter := 1; chapter <= 30; chapter++ {
		name := fmt.Sprintf("ch%02d", chapter)
		t.Run(name, func(t *testing.T) {
			t.Setenv("FD2_MUTE", "1")
			t.Setenv("FD2_TITLE", "0")
			t.Setenv("FD2_CAMPAIGN", "assets/scenarios/campaign_full.json")
			t.Setenv("FD2_CAMP_NODE", "battle_"+name)
			probe := loadGame()
			if probe.loadErr != "" || probe.sc == nil {
				t.Skipf("戰場沒有建立：%s", probe.loadErr)
			}
			keys := make([]string, 0, len(probe.sc.NativeDeathPrograms))
			for key := range probe.sc.NativeDeathPrograms {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				t.Run(key, func(t *testing.T) {
					g := loadGame()
					if !pump(t, g, 20000, func() bool {
						if len(g.dialog) > 0 && g.battleEvent == nil {
							g.dlgAdvance()
						}
						return deathBattleIdle(g)
					}) {
						t.Fatalf("開場工作沒有收掉\n阻塞：%s", ch01Blockers(g))
					}
					dead := findOrSpawnDeathUnit(t, g, key)
					var killer *battle.Unit
					for _, u := range g.st.Units {
						if u != dead && u.Camp == battle.Own && u.Alive() && u.OnField {
							killer = u
							break
						}
					}
					if killer == nil {
						t.Skip("沒有在場的我方單位可當擊殺者")
					}
					dead.ApplyHPDamage(dead.HP)
					g.awardDeathReward(dead, killer)
					g.finishSuccessfulUnitAction(killer, nil)
					done := pump(t, g, 40000, func() bool {
						return !g.nativeDeathProgramsPending() && g.battleEvent == nil &&
							g.nativeTurnStaging == nil && len(g.dialog) == 0
					})
					if !done {
						t.Fatalf("死亡程式沒有跑完\n阻塞：%s", ch01Blockers(g))
					}
				})
			}
		})
	}
}

// findOrSpawnDeathUnit 找帶這個死亡效果的現存單位；沒有就把名冊裡帶它的群組放上來。
func findOrSpawnDeathUnit(t *testing.T, g *Game, key string) *battle.Unit {
	t.Helper()
	match := func(u *battle.Unit) bool {
		kind, value, ok := battle.NativeDeathEffectOf(u)
		return ok && fmt.Sprintf("%d:%d", kind, value) == key && u.Alive()
	}
	for _, u := range g.st.Units {
		if match(u) && u.OnField {
			return u
		}
	}
	// 還沒遷移到原版記錄順序的章節，待登場單位放在 Units 裡、OnField 為 false。
	for _, u := range g.st.Units {
		if match(u) {
			g.st.SpawnGroup(u.Group, u.Camp, false, true)
			if u.OnField {
				return u
			}
		}
	}
	for _, u := range g.st.Roster {
		if !match(u) {
			continue
		}
		before := len(g.st.Units)
		if _, err := g.st.AppendGroupWithNativePlacement(u.Group, 0); err != nil {
			g.st.SpawnGroup(u.Group, u.Camp, false, true)
		}
		for _, candidate := range g.st.Units[before:] {
			if match(candidate) {
				return candidate
			}
		}
	}
	t.Fatalf("找不到帶 %s 的單位", key)
	return nil
}
