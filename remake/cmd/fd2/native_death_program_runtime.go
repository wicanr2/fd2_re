package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 死亡效果型態 2／3 的介面擁有者。
//
// 擊倒發生在攻擊結算（awardDeathReward），程式則在行動收尾
// （finishSuccessfulUnitAction）依 runtime 記錄順序執行：原版 0x1CFF0 在攻擊演出
// 之後才呼叫 0x1B6B7 收集、0x1AA1D 分派，而 0x1B6B7 由索引 0 往上掃。勝敗判定
// 要等程式跑完，頭目的死亡台詞才會在勝利畫面之前播出。

// nativeDeathStagingHandler 標示由死亡程式建立的 0x35822 呼叫，讓 staging 預檢
// 走死亡事件的驗證器而不是回合事件的逐 id 驗證器。
const nativeDeathStagingHandler = "death:0x35822"

// nativeDeathEventDelayMS 是 0x1AC0A 在查表呼叫前的 delay(0xC8)。
const nativeDeathEventDelayMS = 200

type pendingDeathProgram struct {
	dead, killer *battle.Unit
}

// bindNativeDeathPrograms 在劇本綁到戰場後呼叫，讓攻擊結算能先查「這次擊殺會不會
// 取消經驗」。
func (g *Game) bindNativeDeathPrograms() {
	if g.st == nil {
		return
	}
	g.pendingDeathPrograms, g.deathProgramKiller, g.deathProgramRunning, g.deathProgramDead = nil, nil, false, nil
	sc := g.sc
	if sc == nil || len(sc.NativeDeathPrograms) == 0 {
		g.st.NativeDeathExpCancel = nil
		return
	}
	g.st.NativeDeathExpCancel = sc.NativeDeathCancelsExp
}

// queueNativeDeathProgram 在擊倒當下登記。沒有擊殺者（中毒等狀態致死）不登記：
// 原版只有 0x1548E、0x18D8C、0x1CFF0、0x20C6F 四個行動結算點呼叫 0x1B6B7。
func (g *Game) queueNativeDeathProgram(dead, killer *battle.Unit) {
	if dead == nil || killer == nil {
		return
	}
	kind, _, ok := battle.NativeDeathEffectOf(dead)
	if !ok || (kind != 2 && kind != 3) {
		return
	}
	g.pendingDeathPrograms = append(g.pendingDeathPrograms, pendingDeathProgram{dead: dead, killer: killer})
}

// nativeDeathProgramsPending 讓勝敗判定知道還有死亡程式要跑。
func (g *Game) nativeDeathProgramsPending() bool {
	return len(g.pendingDeathPrograms) > 0 || g.deathProgramRunning
}

// runPendingDeathPrograms 依記錄順序逐一執行，全部跑完才呼叫 then。沒有待跑的
// 程式回 false，呼叫端照常往下。
//
// 佇列在開始時換算成記錄索引，每一步再從 g.st.Units 取當下的單位：staging 與
// 0x1DB65 呈現會把 Units 換成新的快照指標，舊指標在那之後就不在戰場上了。記錄
// 索引在原版與重製端都不會移動。
func (g *Game) runPendingDeathPrograms(then func()) bool {
	if len(g.pendingDeathPrograms) == 0 || g.st == nil {
		return false
	}
	queue := g.pendingDeathPrograms
	g.pendingDeathPrograms = nil
	indexOf := func(u *battle.Unit) int {
		for i, candidate := range g.st.Units {
			if candidate == u {
				return i
			}
		}
		return -1
	}
	type slotJob struct{ dead, killer int }
	jobs := make([]slotJob, 0, len(queue))
	for _, job := range queue {
		if dead := indexOf(job.dead); dead >= 0 {
			jobs = append(jobs, slotJob{dead: dead, killer: indexOf(job.killer)})
		}
	}
	sort.SliceStable(jobs, func(i, j int) bool { return jobs[i].dead < jobs[j].dead })

	var step func(int)
	step = func(i int) {
		if i >= len(jobs) {
			g.deathProgramKiller, g.deathProgramRunning, g.deathProgramDead = nil, false, nil
			then()
			return
		}
		dead := g.st.Units[jobs[i].dead]
		var killer *battle.Unit
		if k := jobs[i].killer; k >= 0 && k < len(g.st.Units) {
			killer = g.st.Units[k]
		}
		program, key, ok := g.sc.NativeDeathProgram(dead)
		if !ok {
			kind, value, _ := battle.NativeDeathEffectOf(dead)
			if key == "" {
				key = fmt.Sprintf("%d:%d", kind, value)
			}
			g.deathProgramKiller, g.deathProgramRunning, g.deathProgramDead = nil, false, nil
			g.loadErr = fmt.Sprintf("死亡效果 %s 沒有可執行的原生程式", key)
			return
		}
		actions := make([]battle.Action, 0, len(program)+1)
		if strings.HasPrefix(key, "2:") {
			actions = append(actions, battle.Action{Type: "delay", Ms: nativeDeathEventDelayMS, NativeSource: "0x1ac0a"})
		}
		actions = append(actions, program...)
		// 擊殺者在同一次行動裡倒下也照樣分派；查不到擊殺者時給物品的動作不給。
		g.deathProgramKiller, g.deathProgramRunning, g.deathProgramDead = killer, true, dead
		g.startBattleEvent(actions, func() { step(i + 1) })
	}
	step(0)
	return true
}

// runNativeDeathOp 執行一個 native_death_op。回傳 true 表示已交給阻塞式工作（或已
// 失敗），執行器要停下；false 表示是瞬時狀態寫入，繼續下一個動作。
func (g *Game) runNativeDeathOp(action battle.Action) bool {
	op := action.NativeDeathOp
	if op == nil || action.NativeSource == "" || g.st == nil {
		g.finishBattleEventWithError("死亡程式動作缺少來源或戰場")
		return true
	}
	switch op.Op {
	case "staging":
		id := 0
		if action.NativeEventID != nil {
			id = *action.NativeEventID
		}
		if err := g.beginNativeStagingJob(nativeDeathStagingEvent(id, *op, action.NativeSource), g.advanceBattleEvent); err != nil {
			g.finishBattleEventWithError(err.Error())
		}
		return true
	case "clear_hp_from":
		if err := g.startNativeHPClear(op.First, g.advanceBattleEvent); err != nil {
			g.finishBattleEventWithError(err.Error())
		}
		return true
	case "reward":
		g.grantNativeDeathReward(op.Kind, op.Value, g.deathProgramKiller)
		return false
	}
	if err := g.st.ApplyNativeDeathOp(*op); err != nil {
		g.finishBattleEventWithError(err.Error())
		return true
	}
	if op.Op == "record_bytes" || op.Op == "mark_inactive" {
		// 0x32975 與事件 30 的復活都改變了誰在場上；下一幀重畫。
		g.nativeMapVGA = nil
	}
	return false
}

// nativeDeathStagingEvent 以 0x35822 本體的常數建立一個呼叫。常數與回合事件 63
// 的 helper 相同，因為是同一支函式。
func nativeDeathStagingEvent(id int, op battle.NativeDeathOp, source string) battle.NativeTurnEvent {
	return battle.NativeTurnEvent{
		EventID: id, RawCamp: -1, Handler: nativeDeathStagingHandler,
		Staging: battle.NativeTurnStaging{
			Helper: "0x35822", PanHelper: "0x135dd", SpawnHelper: "0x10b4e",
			DelayBeforeFlashMS: 300, PaletteHelper: "0x11df2", PaletteStart: 0, PaletteEnd: 255,
			FlashDelta: 255, FlashHoldMS: 200, RestoreDelta: 0, RedrawHelper: "0x11cac",
			RawPlacementGate: 0,
			Calls:            []battle.NativeTurnStagingCall{{Group: op.Group, X: op.X, Y: op.Y, Source: source}},
		},
	}
}

func validateNativeDeathStaging(event battle.NativeTurnEvent) error {
	want := nativeDeathStagingEvent(event.EventID, battle.NativeDeathOp{}, "")
	s, w := event.Staging, want.Staging
	if event.Handler != nativeDeathStagingHandler || s.Helper != w.Helper || s.PanHelper != w.PanHelper ||
		s.SpawnHelper != w.SpawnHelper || s.DelayBeforeFlashMS != w.DelayBeforeFlashMS ||
		s.PaletteHelper != w.PaletteHelper || s.PaletteStart != w.PaletteStart ||
		s.PaletteEnd != w.PaletteEnd || s.FlashDelta != w.FlashDelta || s.FlashHoldMS != w.FlashHoldMS ||
		s.RestoreDelta != w.RestoreDelta || s.RedrawHelper != w.RedrawHelper ||
		s.RawPlacementGate != w.RawPlacementGate || len(s.Calls) != 1 || s.Calls[0].Source == "" {
		return fmt.Errorf("event%d death staging differs from the 0x35822 helper", event.EventID)
	}
	return nil
}

// startNativeHPClear 是 0x35BBA：從 first 起清 +0x40，接 0x1DB65 呈現。戰場沒有
// 宣告 indexed 狀態時只落狀態（和 staging 的非 indexed 分支一樣），宣告了卻畫不出來
// 就是錯誤。
func (g *Game) startNativeHPClear(first int, then func()) error {
	prelude := func(st *battle.State) error {
		st.ClearNativeHPFrom(first)
		return nil
	}
	indexed := g.st.HasNativeMapViewState && nativeMapAssetsAvailable(g.nativeMapAssets)
	if indexed {
		return g.startNativeUnitDeathPresent(prelude, then)
	}
	for _, u := range g.st.ClearNativeHPFrom(first) {
		if u.HasNativeRecordByte5 {
			u.NativeRecordByte5 = 1
		}
	}
	then()
	return nil
}

// nativeKillerIsPlayer 是 0x1AC7B／0x1AB95 的 `cmp byte [esi+6], 2`：物品與金錢只給
// 原版陣營 2 的擊殺者。沒有 raw +6 的單位退回引擎陣營。
func nativeKillerIsPlayer(killer *battle.Unit) bool {
	if killer == nil {
		return false
	}
	if killer.HasNativeRecordByte6 {
		return killer.NativeRecordByte6 == 2
	}
	return killer.Camp == battle.Own
}

// grantNativeDeathReward 是 0x1AA1D 的型態 0／1。物品優先放進擊殺者，滿了才交給
// 隊伍空格（原版的轉交提示尚未接）。
func (g *Game) grantNativeDeathReward(kind, value int, killer *battle.Unit) {
	if !nativeKillerIsPlayer(killer) {
		return
	}
	switch kind {
	case 0:
		awarded := false
		if len(killer.Inventory) < 8 {
			awarded = killer.AddInventoryItem(value, false)
		} else {
			awarded = g.grantItemToParty(value)
		}
		key := "battle.reward.item_full"
		if awarded {
			key = "battle.reward.item"
		}
		if message, ok := g.localeMessage(key, value); ok {
			g.msg = message
		}
	case 1:
		g.gold += value
		if message, ok := g.localeMessage("battle.reward.gold", value); ok {
			g.msg = message
		}
	}
}
