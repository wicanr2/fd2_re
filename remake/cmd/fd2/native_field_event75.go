package main

import (
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func event75DialogueActions(textIndex int, trigger *battle.Unit) ([]battle.Action, bool) {
	line, count := 0, 0
	switch textIndex {
	case 0:
		line, count = 0, 1
	case 1:
		line, count = 1, 5
	default:
		return nil, false
	}
	sceneIndex := 0
	lines := loadStoryScriptAt("assets/story/ch29.json", "", &sceneIndex)
	if line < 0 || count <= 0 || line+count > len(lines) {
		return nil, false
	}
	actions := make([]battle.Action, 0, count)
	for _, source := range lines[line : line+count] {
		if source.SpeakerSlot != nil {
			return nil, false
		}
		actions = append(actions, battle.Action{
			Type: "dialogue", Speaker: source.Speaker, Text: source.Text,
		})
	}
	if textIndex == 0 {
		// sub_35C79 在 FDTXT_029 index0 前，先把觸發 runtime record 的 raw +7
		// 交給 0x1956B；BattleFig 是這個 +7 的具型別來源。
		if trigger == nil || !trigger.HasBattleFig {
			return nil, false
		}
		actions[0].Speaker = trigger.BattleFig
	}
	return actions, true
}

// beginNativeFieldEvent75 只由成功動作 owner 消費 map28 selector1。
// FDTXT_029 index1 必須先完成，之後才可一起提交 live rows 與 0x20-byte
// event-state table。
func (g *Game) beginNativeFieldEvent75(actor *battle.Unit, after func()) bool {
	if g == nil || g.st == nil || actor == nil {
		return false
	}
	eventID, bound := battle.NativeFieldEventIDAt(g.st, actor.X, actor.Y, 1)
	if !bound || eventID != 75 {
		return false
	}
	plan, err := battle.PlanNativeFieldEvent75(g.st, actor, actor.X, actor.Y)
	if err != nil {
		g.loadErr = err.Error()
		g.msg = "事件 75 的原版證據不完整，已停止事件寫入"
		if after != nil {
			after()
		}
		return true
	}
	if plan.Noop {
		if after != nil {
			after()
		}
		return true
	}
	actions, ok := event75DialogueActions(plan.TextIndex, actor)
	if !ok {
		g.loadErr = "event75: FDTXT_029 editable dialogue unavailable"
		if after != nil {
			after()
		}
		return true
	}
	then := after
	if plan.Activate {
		then = func() {
			if err := battle.CommitNativeFieldEvent75(g.st, plan); err != nil {
				g.loadErr = err.Error()
			}
			if after != nil {
				after()
			}
		}
	}
	g.startBattleEvent(actions, then)
	return true
}
