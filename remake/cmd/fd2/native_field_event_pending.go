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
// mode-range／event62沿用既有規則；已轉寫的全域程式沿用battleEvent阻塞擁有者。
// 回傳true表示已接管續行或遇到錯誤，其餘id保持失敗即關閉。
func (g *Game) dispatchNativeFieldEventPending(actor *battle.Unit) bool {
	if g == nil {
		return false
	}
	pending := g.nativeFieldEventPending
	g.nativeFieldEventPending = nil
	if pending == nil || g.st == nil || actor == nil || pending.trigger != actor {
		return false
	}
	if pending.eventID == 62 {
		if _, err := battle.ApplyNativeFieldTurnActivationEvent(g.st, pending.x, pending.y, 0); err != nil {
			g.loadErr = "battle field event62: " + err.Error()
		}
		return g.loadErr != ""
	}
	if _, applied := battle.ApplyNativeFieldModeEvent(g.st, actor, pending.x, pending.y, 0); applied {
		return false
	}
	for _, rule := range g.st.NativeFieldEventRules {
		if rule.EventID == int(pending.eventID) && rule.Selector == 0 {
			// 規則存在但閘門沒過（例如 event 26 要求觸發單位 raw +6 != 0）：原版處理器
			// 自己回傳，不是錯誤。
			return false
		}
	}
	// selector0與死亡型態2呼叫同一張0x51B91全域表。歷史欄位2:id保存handler
	// 本體，這裡不加入死亡caller的200ms delay，也不建立死亡／擊殺者暫態。
	if g.sc != nil {
		if actions, ok := g.sc.NativeDeathPrograms["2:"+itoa(int(pending.eventID))]; ok && len(actions) > 0 {
			if g.battleEvent != nil {
				g.loadErr = "battle field event: event owner is already active"
				return true
			}
			for _, action := range actions {
				if action.NativeSource == "" || action.NativeEventID == nil || *action.NativeEventID != int(pending.eventID) {
					g.loadErr = "battle field event: global program provenance mismatch"
					return true
				}
				active, err := action.NativeWhen.Match(g.st)
				if err != nil {
					g.loadErr = "battle field event: " + err.Error()
					return true
				}
				if op := action.NativeDeathOp; active && op != nil && op.Op == "ai_mode_range" {
					for index := op.First; index <= op.Last && index < len(g.st.Units); index++ {
						if index < 0 || g.st.Units[index] == nil || !g.st.Units[index].HasNativeRecordByte34 {
							g.loadErr = "battle field event: raw +0x34 provenance is absent"
							return true
						}
					}
				}
			}
			if g.st.HasNativeMapViewState {
				if err := g.composeNativeMapFrame(); err != nil {
					g.loadErr = "battle field event redraw: " + err.Error()
					return true
				}
			}
			g.startBattleEvent(actions, func() {
				if !g.beginNativeFieldEvent61(actor, nil) {
					g.beginNativeFieldEvent75(actor, nil)
				}
			})
			return true
		}
	}
	g.loadErr = "battle field event: selector 0 event " + itoa(int(pending.eventID)) + " has no transcribed owner"
	return true
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
