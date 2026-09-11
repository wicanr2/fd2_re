package battle

import "fmt"

// 死亡效果（FDFIELD b22..b24 → runtime +0x31..+0x33）的執行期規則。
//
// 0x1B6B7 依記錄順序收集「+5 bit0 未設、+0x31 != 0xFF、HP <= 0」的三個 byte，
// 0x1AA1D(擊殺者, 筆數, 緩衝) 逐筆分派：
//
//	型態 0／1：物品／金錢，只給原版陣營 +6 == 2 的擊殺者（0x1AC7B、0x1AB95）；
//	型態 2：delay(200) 之後呼叫全域事件表 0x51B91[id](擊殺者)（0x1AC0A..0x1AC1A）；
//	型態 3：以 payload 當索引顯示章節戰場文字庫的一句（0x1AC2B..0x1AC49）。
//
// 型態 2 的 23 個處理器由 tools/extract_native_death_events.py 逐指令轉寫並核對，
// 再由 tools/sync_native_death_programs.py 降成各章劇本的 NativeDeathPrograms。
// 對白、pan、演出、生成群組沿用既有動作種類；其餘原語放在 NativeDeathOp。

// NativeActionWhen 是處理器在動作之間讀取的條件。每個條件都在執行到該動作時
// 才判斷，與原版讀取狀態的時間點相同。
type NativeActionWhen struct {
	StateEq   *[2]int `json:"state_eq,omitempty"`   // NativeEventState[i] == v
	StateNe   *[2]int `json:"state_ne,omitempty"`   // NativeEventState[i] != v
	RoundLt   *int    `json:"round_lt,omitempty"`   // [0x53BEF] < n（0x34883 jge）
	AnyActive *[2]int `json:"any_active,omitempty"` // 索引區間內有任何一筆 +5 bit0 未設（0x3453E）
}

// NativeDeathOp 的欄位依 Op 取用；名稱只描述運算，不替 raw byte 猜玩法語意。
type NativeDeathOp struct {
	Op     string   `json:"op"`
	Unit   int      `json:"unit,omitempty"`
	First  int      `json:"first,omitempty"`
	Last   int      `json:"last,omitempty"`
	Mode   int      `json:"mode,omitempty"`
	Value  int      `json:"value,omitempty"`
	Mask   int      `json:"mask,omitempty"`
	Slot   int      `json:"slot,omitempty"`
	Delta  int      `json:"delta,omitempty"`
	Index  int      `json:"index,omitempty"`
	X      int      `json:"x,omitempty"`
	Y      int      `json:"y,omitempty"`
	Group  int      `json:"group,omitempty"`
	Kind   int      `json:"kind,omitempty"`
	Writes [][3]int `json:"writes,omitempty"`
}

// NativeDeathEffectOf 回傳單位目前的死亡效果（型態, 值）。原始 +0x31..+0x33 優先，
// 因為處理器可以把它改成 0xFF（事件 30）；沒有原始欄位時用資產的 death_effect。
func NativeDeathEffectOf(u *Unit) (int, int, bool) {
	if u == nil {
		return 0, 0, false
	}
	if u.HasNativeRecordDeathEffect {
		raw := u.NativeRecordDeathEffect
		if raw[0] == 0xff {
			return 0, 0, false
		}
		return int(raw[0]), int(raw[1]) | int(raw[2])<<8, true
	}
	if u.DeathEffect == nil {
		return 0, 0, false
	}
	return u.DeathEffect.Type, u.DeathEffect.Value, true
}

// NativeDeathProgram 取出死亡效果對應的動作清單。型態 0／1 不走程式。
func (sc *Scenario) NativeDeathProgram(u *Unit) ([]Action, string, bool) {
	kind, value, ok := NativeDeathEffectOf(u)
	if sc == nil || !ok || (kind != 2 && kind != 3) {
		return nil, "", false
	}
	key := fmt.Sprintf("%d:%d", kind, value)
	program, found := sc.NativeDeathPrograms[key]
	return program, key, found
}

// NativeDeathCancelsExp 回答「擊倒這個單位會不會把本次行動的經驗清零」。
// 事件 30 的 0x34B48 在分派時寫 [0x53EC8]=0，而經驗要到 0x1196D 才發；重製端在
// 攻擊結算時就發經驗，所以在那之前先查。只有沒有條件的 exp_cancel 算數。
func (sc *Scenario) NativeDeathCancelsExp(u *Unit) bool {
	program, _, ok := sc.NativeDeathProgram(u)
	if !ok {
		return false
	}
	for _, action := range program {
		if action.NativeDeathOp != nil && action.NativeDeathOp.Op == "exp_cancel" && action.NativeWhen == nil {
			return true
		}
	}
	return false
}

// Match 判斷條件。缺少原版狀態時回錯誤，不以假值猜測。
func (w *NativeActionWhen) Match(st *State) (bool, error) {
	if w == nil {
		return true, nil
	}
	if st == nil {
		return false, fmt.Errorf("native death condition requires battle state")
	}
	state := func(pair *[2]int) (int, error) {
		index := pair[0]
		if index < 0 || index >= len(st.NativeEventState) {
			return 0, fmt.Errorf("native event-state index %d outside table", index)
		}
		return int(st.NativeEventState[index]), nil
	}
	if w.StateEq != nil {
		value, err := state(w.StateEq)
		if err != nil || value != w.StateEq[1] {
			return false, err
		}
	}
	if w.StateNe != nil {
		value, err := state(w.StateNe)
		if err != nil || value == w.StateNe[1] {
			return false, err
		}
	}
	if w.RoundLt != nil {
		if st.NativeRoundCounter <= 0 {
			return false, fmt.Errorf("native round predicate requires [0x53BEF] provenance")
		}
		if st.NativeRoundCounter >= *w.RoundLt {
			return false, nil
		}
	}
	if w.AnyActive != nil {
		any := false
		for index := w.AnyActive[0]; index <= w.AnyActive[1] && index < len(st.Units); index++ {
			unit := st.Units[index]
			if unit != nil && !NativeRecordInactive(unit) {
				any = true
				break
			}
		}
		if !any {
			return false, nil
		}
	}
	return true, nil
}

// NativeRecordInactive 是 0x3453E：記錄 +5 bit0。沒有原版 +5 的單位退回 HP 與在場。
func NativeRecordInactive(u *Unit) bool {
	if u.HasNativeRecordByte5 {
		return u.NativeRecordByte5&1 != 0
	}
	return !u.OnField || !u.Alive()
}

// ApplyNativeDeathOp 執行純狀態的原語。對白、演出、staging、全員倒下的呈現與
// 給物品由介面擁有者執行，這裡對它們回錯誤，避免有人以為已經生效。
func (st *State) ApplyNativeDeathOp(op NativeDeathOp) error {
	if st == nil {
		return fmt.Errorf("native death op %s: no battle state", op.Op)
	}
	each := func(first, last int, f func(index int, u *Unit)) {
		for index := first; index <= last && index < len(st.Units); index++ {
			if index >= 0 && st.Units[index] != nil {
				f(index, st.Units[index])
			}
		}
	}
	switch op.Op {
	case "ai_mode_range":
		// 0x3419C：只改 +0x34 的低四位，保留高四位旗標。還沒建構的記錄之後由
		// 建構器整筆寫入，所以只處理現存單位。
		if op.Mode < 0 || op.Mode > 0x0f || op.First < 0 || op.Last < op.First {
			return fmt.Errorf("ai_mode_range %d..%d mode %d outside raw range", op.First, op.Last, op.Mode)
		}
		each(op.First, op.Last, func(index int, u *Unit) {
			u.NativeRecordByte34 = u.NativeRecordByte34&0xf0 | byte(op.Mode)
			u.HasNativeRecordByte34 = true
			st.syncNativeRuntimeRecord(index, 0x34, int(u.NativeRecordByte34), 1)
		})
	case "ai_byte_set_range":
		// 0x34A85..0x34AA5：整個 +0x34 覆寫。
		each(op.First, op.Last, func(index int, u *Unit) {
			u.NativeRecordByte34, u.HasNativeRecordByte34 = byte(op.Value), true
			st.syncNativeRuntimeRecord(index, 0x34, op.Value, 1)
		})
	case "ai_byte_and_range":
		// 0x34A0E：+0x34 &= mask。
		each(op.First, op.Last, func(index int, u *Unit) {
			u.NativeRecordByte34 &= byte(op.Mask)
			u.HasNativeRecordByte34 = true
			st.syncNativeRuntimeRecord(index, 0x34, int(u.NativeRecordByte34), 1)
		})
	case "record_bytes":
		if op.Unit < 0 {
			return fmt.Errorf("record_bytes: runtime slot %d outside array", op.Unit)
		}
		if op.Unit >= len(st.Units) || st.Units[op.Unit] == nil {
			// 原版照寫 raw 記錄陣列；那一筆之後被 0x10B4E 建構時會整筆覆寫，
			// 所以寫入還沒建構的記錄在重製端是無作用。
			return nil
		}
		if err := applyNativeRecordWrites(st.Units[op.Unit], op.Writes); err != nil {
			return err
		}
		for _, write := range op.Writes {
			st.syncNativeRuntimeRecord(op.Unit, write[0], write[1], write[2])
		}
	case "control_turn":
		// [0x53A55]+3+3×slot = [0x53BEF] + delta。
		if !st.HasNativeTurnEventControlState || st.NativeRoundCounter <= 0 {
			return fmt.Errorf("control_turn requires native turn-control and round provenance")
		}
		if op.Slot < 0 || op.Slot >= len(st.NativeTurnEventControls) {
			return fmt.Errorf("control_turn slot %d outside table", op.Slot)
		}
		turn := st.NativeRoundCounter + op.Delta
		if turn < 0 || turn > 0xff {
			return fmt.Errorf("control_turn value %d outside byte", turn)
		}
		st.NativeTurnEventControls[op.Slot].Turn = byte(turn)
		offset := 3 + 3*op.Slot
		if st.HasNativeFieldControlState && offset < len(st.NativeFieldControlRaw) {
			st.NativeFieldControlRaw[offset] = byte(turn)
		}
	case "state_set", "state_inc":
		if op.Index < 0 || op.Index >= len(st.NativeEventState) {
			return fmt.Errorf("%s index %d outside table", op.Op, op.Index)
		}
		if op.Op == "state_set" {
			st.NativeEventState[op.Index] = byte(op.Value)
		} else {
			st.NativeEventState[op.Index]++
		}
	case "mark_inactive":
		// 0x32975：+5 整個 byte 寫成 1。還沒建構的記錄同 record_bytes，無作用。
		if op.Unit < 0 {
			return fmt.Errorf("mark_inactive: runtime slot %d outside array", op.Unit)
		}
		if op.Unit >= len(st.Units) || st.Units[op.Unit] == nil {
			return nil
		}
		u := st.Units[op.Unit]
		u.OnField = false
		u.NativeRecordByte5, u.HasNativeRecordByte5 = 1, true
		st.syncNativeRuntimeRecord(op.Unit, 0x05, 1, 1)
	case "range_one":
		// 0x35C18：互動時的地圖範圍選擇值。
		st.NativeMapRangeMode, st.HasNativeMapRangeModeState = 1, true
	case "exp_cancel":
		// 效果在攻擊結算時由 NativeDeathCancelsExp 先套用；這裡只保留位置。
	default:
		return fmt.Errorf("native death op %q has no state owner", op.Op)
	}
	return nil
}

// ClearNativeHPFrom 是 0x35BBA 的狀態部分：從 first 起每筆 +0x40 清零。之後的
// 0x1DB65 呈現與 +5 標記由介面擁有者執行。
func (st *State) ClearNativeHPFrom(first int) []*Unit {
	var cleared []*Unit
	for index := first; index >= 0 && index < len(st.Units); index++ {
		if u := st.Units[index]; u != nil {
			u.HP = 0
			st.syncNativeRuntimeRecord(index, 0x40, 0, 2)
			cleared = append(cleared, u)
		}
	}
	return cleared
}

// syncNativeRuntimeRecord 在戰場帶完整 0x50 raw 投影時（CONTINUE 載入）把同一筆
// 寫入也落到投影，讓後續以投影驗證的呈現與存檔看到同一個值。
func (st *State) syncNativeRuntimeRecord(index, offset, value, width int) {
	if !st.HasNativeRuntimeUnitProjection || index < 0 || index >= len(st.NativeRuntimeRecords) {
		return
	}
	raw := st.NativeRuntimeRecords[index].Raw[:]
	if offset < 0 || offset+width > len(raw) {
		return
	}
	raw[offset] = byte(value)
	if width == 2 {
		raw[offset+1] = byte(value >> 8)
	}
}

// applyNativeRecordWrites 把直接記錄寫入落到型別化欄位。每個 offset 都要有對應
// 的欄位，沒有就整筆拒絕，不留半套。
func applyNativeRecordWrites(u *Unit, writes [][3]int) error {
	type setter func()
	var apply []setter
	for _, write := range writes {
		offset, value, width := write[0], write[1], write[2]
		if (width == 1 && (value < 0 || value > 0xff)) || (width == 2 && (value < 0 || value > 0xffff)) {
			return fmt.Errorf("record write +%#x value %d outside width %d", offset, value, width)
		}
		switch {
		case offset == 0x05 && width == 1:
			apply = append(apply, func() {
				u.NativeRecordByte5, u.HasNativeRecordByte5 = byte(value), true
				if value&1 == 0 {
					u.OnField = true
				}
			})
		case offset == 0x06 && width == 1:
			camp, err := nativeRawCamp(byte(value))
			if err != nil {
				return err
			}
			apply = append(apply, func() {
				u.NativeRecordByte6, u.HasNativeRecordByte6 = byte(value), true
				u.Camp = camp
			})
		case offset == 0x07 && width == 1:
			apply = append(apply, func() { u.BattleFig, u.HasBattleFig = value, true })
		case offset == 0x08 && width == 1:
			apply = append(apply, func() {
				u.NativeRecordByte8, u.HasNativeRecordByte8 = byte(value), true
				if u.HasNativeIdentity {
					u.NativeIdentity = value
				}
			})
		case offset == 0x31 && width == 1:
			apply = append(apply, func() {
				u.NativeRecordDeathEffect[0], u.HasNativeRecordDeathEffect = byte(value), true
				if value == 0xff {
					u.DeathEffect, u.DeathReward = nil, nil
				}
			})
		case offset == 0x34 && width == 1:
			apply = append(apply, func() { u.NativeRecordByte34, u.HasNativeRecordByte34 = byte(value), true })
		case offset == 0x40 && width == 2:
			apply = append(apply, func() { u.HP = value })
		default:
			return fmt.Errorf("record write +%#x width %d has no typed field", offset, width)
		}
	}
	for _, f := range apply {
		f()
	}
	return nil
}

// nativeRawCamp 是地圖資料實測的對應：+6 為 0 敵方、1 友軍、2 我方。
func nativeRawCamp(raw byte) (Camp, error) {
	switch raw {
	case 0:
		return Enemy, nil
	case 1:
		return Ally, nil
	case 2:
		return Own, nil
	}
	return Own, fmt.Errorf("raw camp %d has no engine camp", raw)
}

// killCancelsExp 是發經驗前的查詢：本次行動擊倒的任何一個單位帶無條件的
// exp_cancel，整次行動的經驗就是 0（原版清的是整次行動累積的 [0x53EC8]）。
func (st *State) killCancelsExp(targets ...*Unit) bool {
	if st == nil || st.NativeDeathExpCancel == nil {
		return false
	}
	for _, target := range targets {
		if target != nil && target.HP <= 0 && st.NativeDeathExpCancel(target) {
			return true
		}
	}
	return false
}
