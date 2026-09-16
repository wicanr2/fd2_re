package battle

// native_end_turn_recovery.go — 0x1A30B 開頭的我方回復掃描。
//
// 0x1A332..0x1A3A2 先掃一次找出要回復的單位（畫圖示 0x1DA16、播 0x25A96(…,4,1)），
// 0x1A3F2..0x1A484 再掃一次把數值寫回：`+6`＝2、`(+5 & 0x81)`＝0、`+0x25`＝0、
// `+0x26`＝0、且 `+0x40`≠`+0x42` 的記錄，`+0x40 += +0x42 / 5`，超過就取 `+0x42`。
// 這段在 selector 1 的回合事件（0x1A813(1)）之前，玩家按 END、全員行動完的 0x13565
// 都會經過；已行動（0x80）的單位不回復，所以「全員行動完自動換手」那一回合沒有人回血。
// 回復過的單位由 0x13512 設 +5 bit7（0x1A477），橫幅之後 0x13536 才清。
// 數值部分由 fdother.NativeBattleEntryStep 釘住 raw 版面，這裡是 Unit 投影上的同一件事。

// NativeEndTurnRecovery 是一個單位在 0x1A30B 回復掃描的結果。
type NativeEndTurnRecovery struct {
	Unit   *Unit
	Before int
	After  int
}

// ApplyNativeEndTurnRecovery 對場上所有符合 0x1A30B 閘門的我方單位回復 MaxHP/5，
// 回傳有變動的單位（依記錄順序）。沒有原版 `+6`／`+5` 的舊可編輯單位用 Camp／Acted
// ／Alive 對應同一組條件。
func (s *State) ApplyNativeEndTurnRecovery() []NativeEndTurnRecovery {
	if s == nil {
		return nil
	}
	var out []NativeEndTurnRecovery
	for _, u := range s.Units {
		if u == nil || !u.OnField {
			continue
		}
		if u.HasNativeRecordByte6 {
			if u.NativeRecordByte6 != 2 {
				continue
			}
		} else if u.Camp != Own {
			continue
		}
		if !u.Alive() {
			continue
		}
		if u.HasNativeRecordByte5 {
			if u.NativeRecordByte5&0x81 != 0 {
				continue
			}
		} else if u.Acted {
			continue
		}
		if d, ok := u.NativeTransientDuration(0x25); ok && d != 0 {
			continue
		}
		if d, ok := u.NativeTransientDuration(0x26); ok && d != 0 {
			continue
		}
		if u.HP == u.MaxHP || u.MaxHP <= 0 {
			continue
		}
		before := u.HP
		next := u.HP + u.MaxHP/5
		if next > u.MaxHP {
			next = u.MaxHP
		}
		u.HP = next
		// 0x1A477：每個回復的單位接著 0x13512 把 +5 bit7 設起來（橫幅下畫成灰色，
		// r10 收據 seq 934），0x13536 在橫幅之後才整批清掉。
		if u.HasNativeRecordByte5 {
			u.NativeRecordByte5 |= 0x80
		}
		out = append(out, NativeEndTurnRecovery{Unit: u, Before: before, After: next})
	}
	return out
}

// MarkNativeDeadRecords 是 0x1DB65 的狀態部分：每次行動結算後（玩家路徑與敵方
// 路徑 0x15643 都呼叫）掃全部記錄，`+0x40` 為 0 的記錄把 `+5` 整個 byte 寫成 1
// （0x1DC61／0x1DD4C），之後的 AI 目標掃描、回合回復與勝負判定都以這個 bit 為準。
// 物理攻擊在扣血時已鏡射 bit0，這裡補的是指令傷害等只寫 HP 的路徑。回傳新標記的單位。
func (s *State) MarkNativeDeadRecords() []*Unit {
	if s == nil {
		return nil
	}
	var marked []*Unit
	for index, u := range s.Units {
		if u == nil || !u.HasNativeRecordByte5 || u.HP != 0 || u.NativeRecordByte5 == 1 {
			continue
		}
		u.NativeRecordByte5 = 1
		s.syncNativeRuntimeRecord(index, 0x05, 1, 1)
		marked = append(marked, u)
	}
	return marked
}
