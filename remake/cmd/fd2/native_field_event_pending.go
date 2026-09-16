package main

import "github.com/wicanr2/fd2_re/remake/internal/battle"

// nativeFieldEventPending 是原版 [0x51A8F]：走行的每一步（0x1300D→0x13175）都用新座標
// 呼叫 0x13A44(x, y, 0)，格子有 selector 0 的事件就把 event id 記下來；行動收尾
// （玩家路徑 0x1198A、AI 三遍 0x1D855／0x1D94C／0x1D9DC）才以行動單位為引數呼叫全域
// 事件表 0x51B91 的那一項，然後寫回 0xff。玩家選取單位開始行動時（0x188BF）先清成
// 0xff，所以取消的移動不會觸發。第七章 r2 收據：騎士直線向上走進 (12,15)，待機收尾
// 之後記錄 9..27 的 +0x34 低四位才清 0——不是只有向左踏入那一拍。
type nativeFieldEventPending struct {
	eventID byte
	x, y    int
	trigger *battle.Unit
}

// noteNativeFieldEventStep 是 0x13175：走到 (x,y) 就查 selector 0 的格子事件，有就蓋掉
// 之前記的（一格只留最後一個）。
func (g *Game) noteNativeFieldEventStep(trigger *battle.Unit, x, y int) {
	if g == nil || g.st == nil || trigger == nil {
		return
	}
	eventID, ok := battle.NativeFieldEventIDAt(g.st, x, y, 0)
	if !ok {
		return
	}
	g.nativeFieldEventPending = &nativeFieldEventPending{eventID: eventID, x: x, y: y, trigger: trigger}
}

// dispatchNativeFieldEventPending 是行動收尾的 `cmp [0x51A8F],0xff / call [0x51B91+id*4]`：
// 只分派已轉寫成 native_field_event_rules 的處理器（mode-range 家族與 event62），其餘
// id 保留失敗即關閉——沒有 owner 的事件不能靜默吞掉。
func (g *Game) dispatchNativeFieldEventPending(actor *battle.Unit) {
	pending := g.nativeFieldEventPending
	g.nativeFieldEventPending = nil
	if pending == nil || g == nil || g.st == nil || actor == nil || pending.trigger != actor {
		return
	}
	if pending.eventID == 62 {
		if _, err := battle.ApplyNativeFieldTurnActivationEvent(g.st, pending.x, pending.y, 0); err != nil {
			g.loadErr = "battle field event62: " + err.Error()
		}
		return
	}
	if _, applied := battle.ApplyNativeFieldModeEvent(g.st, actor, pending.x, pending.y, 0); applied {
		return
	}
	for _, rule := range g.st.NativeFieldEventRules {
		if rule.EventID == int(pending.eventID) && rule.Selector == 0 {
			// 規則存在但閘門沒過（例如 event 26 要求觸發單位 raw +6 != 0）：原版處理器
			// 自己回傳，不是錯誤。
			return
		}
	}
	g.loadErr = "battle field event: selector 0 event " + itoa(int(pending.eventID)) + " has no transcribed owner"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
