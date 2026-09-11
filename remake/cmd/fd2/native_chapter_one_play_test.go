package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 重製端自己把第一關打完，對應原版側的 docs/data/parity-plans/ch01-clear.jsonl。
//
// 兩側的驅動方式不同：原版那側只能送 BIOS 按鍵、可達範圍看不到，所以靠「送 enter
// 之後座標變了沒」試探；重製端這側直接讀 g.reach，挑格子是算出來的。走過的節點
// 相同，但**不是逐鍵重放**——這一條驗的是「重製端的戰鬥狀態機自己跑得完一整關」，
// 不是逐幀對拍。
//
// 預設跳過：要完整分離素材，而且一整關要跑幾十萬幀。
//
//	FD2_CH01_PLAY=<輸出目錄> go test ./cmd/fd2 -run TestPlayChapterOneToVictory
const (
	ch01MaxRounds    = 40
	ch01MaxUnitsTurn = 16
	ch01FrameBudget  = 4000
)

type ch01Recorder struct {
	out   string
	log   *os.File
	round int
}

func (r *ch01Recorder) note(g *Game, tag string) {
	if r.log == nil {
		return
	}
	own, enemy, enemyHP := 0, 0, 0
	for _, u := range g.st.Units {
		if u.HP <= 0 || !u.OnField {
			continue
		}
		switch u.Camp {
		case battle.Own:
			own++
		case battle.Enemy:
			enemy++
			enemyHP += u.HP
		}
	}
	view := g.st.NativeMapViewState
	var pendingGroups []int
	for _, u := range g.st.Roster {
		if u != nil && g.st.PendingGroups[u.Group] && u.Alive() && u.Camp == battle.Enemy {
			pendingGroups = append(pendingGroups, u.Group)
		}
	}
	offField := 0
	for _, u := range g.st.Units {
		if !u.OnField && u.Alive() && u.Camp == battle.Enemy {
			offField++
		}
	}
	record := map[string]any{
		"pending_enemy":         g.st.PendingCount(battle.Enemy),
		"pending_roster_groups": pendingGroups,
		"enemy_off_field":       offField,
		"round":                 r.round, "tag": tag, "own_alive": own,
		"enemy_alive": enemy, "enemy_hp": enemyHP, "result": g.result,
		"camera": []int{view.CameraX, view.CameraY},
		"cursor": []int{view.CursorX, view.CursorY},
		// 可見游標與 cursor-camera 的差：原版容許短暫不一致（0x149F8 只寫
		// 絕對游標），但走行步進會把它補回來。持續拉大就代表有一條路徑漏了。
		"visible": []int{view.VisibleCursorX, view.VisibleCursorY},
		"drift": []int{view.VisibleCursorX - (view.CursorX - view.CameraX),
			view.VisibleCursorY - (view.CursorY - view.CameraY)},
	}
	if enemy <= 3 { // 收尾階段才記位置，前面幾波人多沒有閱讀價值
		var where []any
		for _, u := range g.st.Units {
			if u.Camp != battle.Enemy || u.HP <= 0 || !u.OnField {
				continue
			}
			where = append(where, []any{
				fmt.Sprintf("%d,%d", u.NativePositionRecord.XWord, u.NativePositionRecord.YWord),
				u.X, u.Y, u.HP})
		}
		record["enemies"] = where
	}
	encoded, _ := json.Marshal(record)
	fmt.Fprintf(r.log, "%s\n", encoded)
}

// pendingOwn 回傳這一回合還沒行動的我方單位。Acted 是引擎自己的投影，和原版
// record +5 bit7 是兩回事——這裡用得起，因為驅動的是重製端自己的狀態機。
func pendingOwn(g *Game) []*battle.Unit {
	var pending []*battle.Unit
	for _, u := range g.st.Units {
		if u.Camp == battle.Own && u.HP > 0 && u.OnField && !u.Acted {
			pending = append(pending, u)
		}
	}
	sort.Slice(pending, func(a, b int) bool {
		if pending[a].Y != pending[b].Y {
			return pending[a].Y < pending[b].Y
		}
		return pending[a].X < pending[b].X
	})
	return pending
}

func livingEnemies(g *Game) []*battle.Unit {
	var enemies []*battle.Unit
	for _, u := range g.st.Units {
		if u.Camp == battle.Enemy && u.HP > 0 && u.OnField {
			enemies = append(enemies, u)
		}
	}
	return enemies
}

func manhattan(ax, ay, bx, by int) int {
	dx, dy := ax-bx, ay-by
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}

// isProtected 回傳這個單位是不是失敗條件綁的那一個。第一關沒有在節點上指定
// protect，checkResult 因此用預設值「索爾」——doc28 第 1 章：索爾死＝失敗，其餘
// 三人全倒也不算輸。自動戰術要是不知道這條規則，會把主角當成普通前鋒送上去。
func isProtected(u *battle.Unit) bool {
	return (u.HasNativeIdentity && u.NativeIdentity == 0) || u.Name == "索爾"
}

// estimateDamage 估 attacker 站在 from 打 target 的傷害。和引擎 AI 內部的
// estDamage 同一條規則（雙方地形 AP%／DP% 之後相減），但那是未匯出的；這裡重寫
// 一份只用來挑格子，實際結算仍由 AttackNativePhysicalWithExperience 擲骰。
func estimateDamage(g *Game, attacker *battle.Unit, from battle.Cell, target *battle.Unit) int {
	apPct, _ := g.st.TerrainAPDPPct(from.X, from.Y)
	_, dpPct := g.st.TerrainAPDPPct(target.X, target.Y)
	damage := attacker.AP*(100+apPct)/100 - target.DP*(100+dpPct)/100
	if damage < 0 {
		return 0
	}
	return damage
}

// incomingAt 估計站在這一格下回合會挨多少傷害。涵蓋判準是敵人的移動力加射程；
// 敵人那一側的地形加成算不了（牠還沒走），所以只把我方這一格的 DP% 算進去。
// 這是估計不是精算——精算要把敵方可達範圍連地形成本重跑一遍。
func incomingAt(g *Game, actor *battle.Unit, cell battle.Cell) int {
	_, dpPct := g.st.TerrainAPDPPct(cell.X, cell.Y)
	dp := actor.DP * (100 + dpPct) / 100
	total := 0
	for _, enemy := range livingEnemies(g) {
		span := enemy.MV + enemy.AtkMax
		if enemy.AtkMax == 0 {
			span = enemy.MV + 1
		}
		if manhattan(cell.X, cell.Y, enemy.X, enemy.Y) > span {
			continue
		}
		if damage := enemy.AP - dp; damage > 0 {
			total += damage
		}
	}
	return total
}

// candidateCells 把可達集合攤平並排序。g.reach 是 map，直接 range 的順序每次都
// 不一樣，分數打平時會選到不同格——整場就跟著發散，固定種子也重現不了。
func candidateCells(g *Game, actor *battle.Unit) []battle.Cell {
	cells := []battle.Cell{{X: actor.X, Y: actor.Y}}
	for cell := range g.reach {
		if cell.X != actor.X || cell.Y != actor.Y {
			cells = append(cells, cell)
		}
	}
	sort.Slice(cells, func(a, b int) bool {
		if cells[a].Y != cells[b].Y {
			return cells[a].Y < cells[b].Y
		}
		return cells[a].X < cells[b].X
	})
	return cells
}

// partyCenter 是我方活著的單位的重心，給保護目標當「不要落單」的參照。
func partyCenter(g *Game) (int, int, bool) {
	sumX, sumY, n := 0, 0, 0
	for _, u := range g.st.Units {
		if u.Camp != battle.Own || u.HP <= 0 || !u.OnField {
			continue
		}
		sumX, sumY, n = sumX+u.X, sumY+u.Y, n+1
	}
	if n == 0 {
		return 0, 0, false
	}
	return sumX / n, sumY / n, true
}

// chooseDestination 從可達範圍挑落腳格。原版側那條路只能靠「送 enter 之後座標變了
// 沒」試探，這裡 g.reach 直接給得出可達集合，所以挑得起來。
//
// 權衡四件事：這一擊的收益（能帶走一個最高）、站上去下回合會挨多少、原地待機的
// 回復（沒移動且未滿血回最大 HP 的 1/5，見 finishSelectedWait），以及離戰線遠近。
// 保護目標另一套：它的權重壓在「不會被打死」與「不要落單」，只在沒人打得到時出手。
func chooseDestination(g *Game, actor *battle.Unit) (battle.Cell, *battle.Unit) {
	enemies := livingEnemies(g)
	origin := battle.Cell{X: actor.X, Y: actor.Y}
	if len(enemies) == 0 {
		return origin, nil
	}
	protected := isProtected(actor)
	centerX, centerY, hasCenter := partyCenter(g)
	best, bestTarget := origin, (*battle.Unit)(nil)
	bestScore := -(1 << 40)
	for _, cell := range candidateCells(g, actor) {
		probe := *actor
		probe.X, probe.Y = cell.X, cell.Y
		var target *battle.Unit
		gain := 0
		for _, enemy := range enemies {
			if !g.st.InAttackRange(&probe, enemy.X, enemy.Y) {
				continue
			}
			damage := estimateDamage(g, actor, cell, enemy)
			value := damage
			if damage >= enemy.HP { // 能帶走就帶走：少一個人打我方
				value += 1000
			}
			if target == nil || value > gain {
				target, gain = enemy, value
			}
		}
		incoming := incomingAt(g, actor, cell)
		near := 1 << 20
		for _, enemy := range enemies {
			if d := manhattan(cell.X, cell.Y, enemy.X, enemy.Y); d < near {
				near = d
			}
		}
		stay := cell == origin

		var score int
		switch {
		case protected:
			// 保護目標的權重壓在「不會被打死」與「不要落單」。它還是會慢慢
			// 往戰線靠，但威脅的權重遠大於距離，所以不會自己走進包圍。
			score = -incoming * 50
			if incoming*2 >= actor.HP {
				score -= 6000
			}
			if target != nil && incoming <= actor.HP/4 {
				score += 1200 + gain*4
			} else {
				target = nil
			}
			if hasCenter {
				score -= manhattan(cell.X, cell.Y, centerX, centerY) * 10
			}
			if stay && actor.HP < actor.MaxHP {
				score += 300
			}
		default:
			// 接敵是主導項。威脅只當同分時的偏好——把威脅權重放在距離之上，
			// 單位就會停在敵人射程外一格不動，被動的敵人永遠等不到人打。
			score = -near * 40
			if target != nil {
				score += 2000 + gain*8
			}
			if incoming >= actor.HP { // 只有「站上去可能被打死」才是硬條件
				score -= 6000
			}
			score -= incoming
			if stay && actor.HP*2 < actor.MaxHP && incoming == 0 {
				score += 900 // 重傷又沒人打得到：原地待機回最大 HP 的 1/5
			}
		}
		if score > bestScore {
			best, bestTarget, bestScore = cell, target, score
		}
	}
	return best, bestTarget
}

// ch01FrameObserver 讓整條旅程的測試在每一幀記下節點、對白與戰況；單獨跑第一關
// 時是 nil。
var ch01FrameObserver func(g *Game)

func pump(t *testing.T, g *Game, budget int, done func() bool) bool {
	t.Helper()
	for frame := 0; frame < budget; frame++ {
		if ch01FrameObserver != nil {
			ch01FrameObserver(g)
		}
		if done() {
			return true
		}
		ackPresents(g)
		answerNativeTreasurePrompt(g)
		// 戰場事件的對白要玩家按 enter 才會往下走（正式路徑是
		// handleBattleEventDialogueInput）。第一關中途就有這種事件，沒人按就停在
		// 那裡，看起來像敵方回合收不掉。逐字與收框各自有閘門，逐幀按等同玩家連按。
		if (g.battleEvent != nil || g.nativeTurnStaging != nil) && len(g.dialog) > 0 {
			g.handleBattleEventDialogueInput(true)
		}
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
		if g.loadErr != "" {
			view := g.st.NativeMapViewState
			t.Fatalf("執行期錯誤：%s\n視圖：鏡頭(%d,%d) 游標(%d,%d) 可見(%d,%d)／繪製鏡頭(%.0f,%.0f)\n阻塞：%s",
				g.loadErr, view.CameraX, view.CameraY, view.CursorX, view.CursorY,
				view.VisibleCursorX, view.VisibleCursorY,
				g.camX/float64(g.m.TileW), g.camY/float64(g.m.TileH), ch01Blockers(g))
		}
	}
	return done()
}

// closeRing 推完指令環的收合動畫。開合掛在「這一幀被畫過」，離屏不走 Draw，
// 所以要自己標記——正式路徑由繪製那側標。
func closeRing(t *testing.T, g *Game, then func()) bool {
	t.Helper()
	g.beginActionOverlayClose(then)
	for frame := 0; frame < 120 && g.ring; frame++ {
		if err := g.composeNativeMapFrame(); err != nil {
			t.Fatalf("組不出整幀：%v", err)
		}
		g.markActionOverlayDrawn()
		if err := g.Update(); err != nil {
			t.Fatalf("Update：%v", err)
		}
	}
	return !g.ring
}

// playUnit 走完一個我方單位：選取 → 移動 → 攻擊或待機。
func playUnit(t *testing.T, g *Game, actor *battle.Unit, rec *ch01Recorder) bool {
	t.Helper()
	before := g.sel
	if !g.positionScreenshotCursor(actor.X, actor.Y) {
		t.Fatalf("游標移不到 (%d,%d)", actor.X, actor.Y)
	}
	g.confirm()
	if g.sel == nil || g.sel != actor {
		got := "nil"
		if g.sel != nil {
			got = fmt.Sprintf("%s(%d,%d)acted=%v", g.sel.Name, g.sel.X, g.sel.Y, g.sel.Acted)
		}
		had := "nil"
		if before != nil {
			had = fmt.Sprintf("%s(%d,%d)", before.Name, before.X, before.Y)
		}
		t.Fatalf("選取 %s(%d,%d) 失敗：游標=(%d,%d) 選取前=%s 選取後=%s err=%q\n阻塞：%s",
			actor.Name, actor.X, actor.Y, g.curX, g.curY, had, got, g.loadErr, ch01Blockers(g))
	}
	destination, target := chooseDestination(g, actor)

	if destination.X != actor.X || destination.Y != actor.Y {
		if !g.positionScreenshotCursor(destination.X, destination.Y) {
			t.Fatalf("游標移不到落腳格 (%d,%d)", destination.X, destination.Y)
		}
		g.confirm()
		if g.walk == nil {
			t.Fatalf("(%d,%d)→(%d,%d) 沒有開始移動 err=%q",
				actor.X, actor.Y, destination.X, destination.Y, g.loadErr)
		}
		if !pump(t, g, ch01FrameBudget, func() bool { return g.walk == nil }) {
			t.Fatal("移動動畫沒有結束")
		}
	} else {
		// 原地行動：原版也是在原格再按一次 enter 開指令環。
		g.confirm()
	}
	if !pump(t, g, 240, func() bool { return g.ring }) {
		t.Fatalf("走到 (%d,%d) 之後沒有開指令環 err=%q",
			destination.X, destination.Y, g.loadErr)
	}

	available := g.actionOverlayAvailability()
	if target != nil && nativeActionSelectable(available, 0) {
		g.ringSel = 0
		if !closeRing(t, g, func() {}) {
			t.Fatal("攻擊：指令環沒有收合")
		}
		if !g.positionScreenshotCursor(target.X, target.Y) {
			t.Fatalf("游標移不到目標 (%d,%d)", target.X, target.Y)
		}
		hp := target.HP
		g.confirm()
		pump(t, g, ch01FrameBudget, func() bool {
			return g.atk == nil && g.walk == nil && !g.ring && (target.HP != hp || target.HP <= 0)
		})
		// 攻擊演出跑完之後還可能接升級訊息等；等到這個單位真的行動完。
		if !pump(t, g, ch01FrameBudget, func() bool { return actor.Acted || actor.HP <= 0 }) {
			t.Fatalf("%s(%d,%d) 攻擊之後沒有行動完畢\n阻塞：%s",
				actor.Name, actor.X, actor.Y, ch01Blockers(g))
		}
		rec.note(g, fmt.Sprintf("attack (%d,%d)->(%d,%d)",
			destination.X, destination.Y, target.X, target.Y))
		return true
	}

	if !nativeActionSelectable(available, 3) {
		t.Fatalf("(%d,%d) 既打不到人也不能待機，availability=%v",
			destination.X, destination.Y, available)
	}
	g.ringSel = 3
	if !closeRing(t, g, g.finishSelectedWait) {
		t.Fatal("待機：指令環沒有收合")
	}
	if !pump(t, g, ch01FrameBudget, func() bool { return actor.Acted || actor.HP <= 0 }) {
		t.Fatalf("%s(%d,%d) 待機之後沒有行動完畢\n阻塞：%s",
			actor.Name, actor.X, actor.Y, ch01Blockers(g))
	}
	rec.note(g, fmt.Sprintf("wait (%d,%d)", destination.X, destination.Y))
	return true
}

func TestPlayChapterOneToVictory(t *testing.T) {
	out := os.Getenv("FD2_CH01_PLAY")
	if out == "" {
		t.Skip("需要 FD2_CH01_PLAY 指向輸出目錄")
	}
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_SHOT_AI", "1")     // 敵方回合要真的跑 AI，不是跳過
	t.Setenv("FD2_SEED", "20260911") // 固定種子：同一份戰術每次走同一局

	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatal(err)
	}
	if g.st == nil {
		t.Fatal("快進之後沒有戰場狀態")
	}

	log, err := os.Create(filepath.Join(out, "rounds.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()
	rec := &ch01Recorder{out: out, log: log}
	rec.note(g, "battle-start")

	playChapterOneRounds(t, g, rec)

	rec.note(g, "final")
	if g.result != "win" {
		t.Fatalf("第一關沒有打贏：result=%q 敵方存活 %d", g.result, len(livingEnemies(g)))
	}
}

// ch01Blockers 把 Game 上「還在進行中」的欄位列出來。卡住的時候光有 aiBusy 與
// staging 兩個旗標不夠——阻塞可能掛在任何一個演出工作上，用反射一次全掃比逐個
// 猜欄位名可靠。
func ch01Blockers(g *Game) string {
	v := reflect.ValueOf(g).Elem()
	tv := v.Type()
	var parts []string
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		name := tv.Field(i).Name
		switch f.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Func:
			if !f.IsNil() {
				parts = append(parts, name)
			}
		case reflect.Slice:
			if f.Type().Elem().Kind() == reflect.Ptr && f.Len() > 0 {
				parts = append(parts, fmt.Sprintf("%s[%d]", name, f.Len()))
			}
		case reflect.Bool:
			if f.Bool() {
				parts = append(parts, name)
			}
		case reflect.Int:
			if f.Int() != 0 {
				parts = append(parts, fmt.Sprintf("%s=%d", name, f.Int()))
			}
		case reflect.String:
			if s := f.String(); s != "" && len(s) < 40 {
				parts = append(parts, fmt.Sprintf("%s=%q", name, s))
			}
		}
	}
	return strings.Join(parts, " ")
}

func ch01UnitDump(g *Game) string {
	var parts []string
	for _, u := range g.st.Units {
		if u.HP <= 0 || !u.OnField {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s/camp%d/(%d,%d)/hp%d/acted%v",
			u.Name, u.Camp, u.X, u.Y, u.HP, u.Acted))
	}
	return strings.Join(parts, " ")
}

// ackPresents 承認演出工作的「這一幀已呈現」。離屏不走 Draw，而援軍進場、調色盤
// 斜坡這些工作都掛在 drawn 上；`fastForwardShotCampaign` 走的是同一種承認方式。
// 這裡只標 drawn，不動各自的 wait——等待幀數照跑，時序維持真實。
func ackPresents(g *Game) {
	if g.spawnIntroTransition != nil {
		g.spawnIntroTransition.drawn = true
	}
	if g.indexedTransition != nil {
		g.indexedTransition.drawn = true
	}
	if g.nativePaletteRamp != nil {
		g.nativePaletteRamp.drawn = true
	}
	if g.nativePalettePulse != nil {
		g.nativePalettePulse.drawn = true
	}
	if g.transitionReveal != nil {
		g.transitionReveal.drawn = true
	}
	if g.nativeUnitPresent != nil {
		g.nativeUnitPresent.drawn = true
	}
	if g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
	}
}

// answerNativeTreasurePrompt 回答踩到寶物時跳出的取得提示。單位待機在寶物格
// 會停在那個提示上（finishSelectedWait 先 return，等提示走完才提交行動），沒人
// 回答就等於這個單位永遠沒行動完。按的位置與 ringInput 的 enter 分支相同：
// 預設選項是 YES，收下之後還要再按一次確認取得訊息。
//
// 只回答寶物提示。系統選單的 END 確認不在這裡答——測試是直接呼叫 endTurn()，
// 沒有走那條路，真的冒出來代表有別的東西誤觸了系統覆蓋層。
func answerNativeTreasurePrompt(g *Game) {
	state := g.nativeSystemEndTurnUI
	if state == nil || state.treasure == nil || g.nativeClassUIJob != nil {
		return
	}
	if state.treasure.awaitAck {
		state.treasure.awaitAck = false
		g.nativeSystemEndTurnDelay = 1
		return
	}
	if g.nativeSystemEndTurnConfirm {
		g.confirmNativeSystemEndTurn()
	}
}

// playChapterOneRounds 從玩家取得操作權那一刻打到分出勝負：每回合掃完我方未行動
// 單位，結束回合，等敵方回合與回合末事件全部收掉。
func playChapterOneRounds(t *testing.T, g *Game, rec *ch01Recorder) {
	t.Helper()
	for round := 1; round <= ch01MaxRounds; round++ {
		rec.round = round
		if g.result != "" {
			break
		}
		handled := map[*battle.Unit]bool{}
		for step := 0; step < ch01MaxUnitsTurn; step++ {
			if g.result != "" || len(livingEnemies(g)) == 0 {
				break
			}
			var actor *battle.Unit
			for _, u := range pendingOwn(g) {
				if !handled[u] {
					actor = u
					break
				}
			}
			if actor == nil {
				break
			}
			handled[actor] = true
			playUnit(t, g, actor, rec)
		}
		rec.note(g, "own-phase-done")
		if g.result != "" {
			break
		}

		g.endTurn()
		if !pump(t, g, ch01FrameBudget*4, func() bool {
			return g.result != "" || (!g.aiBusy && g.nativeTurnStaging == nil && len(pendingOwn(g)) > 0)
		}) {
			t.Fatalf("第 %d 回合：敵方回合沒有結束（aiBusy=%v staging=%v turn=%d）\n阻塞：%s\n單位：%s",
				round, g.aiBusy, g.nativeTurnStaging != nil, g.st.Turn,
				ch01Blockers(g), ch01UnitDump(g))
		}
		rec.note(g, "enemy-phase-done")
	}
}
