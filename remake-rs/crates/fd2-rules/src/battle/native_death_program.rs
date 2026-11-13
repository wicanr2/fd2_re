//! Go 來源：`remake/internal/battle/native_death_program.go`（376 行）。

use super::*;

// 死亡效果（FDFIELD b22..b24 → runtime +0x31..+0x33）的執行期規則。
//
// 0x1B6B7 依記錄順序收集「+5 bit0 未設、+0x31 != 0xFF、HP <= 0」的三個 byte，
// 0x1AA1D(擊殺者, 筆數, 緩衝) 逐筆分派：
//
//     型態 0／1：物品／金錢，只給原版陣營 +6 == 2 的擊殺者（0x1AC7B、0x1AB95）；
//     型態 2：delay(200) 之後呼叫全域事件表 0x51B91[id](擊殺者)（0x1AC0A..0x1AC1A）；
//     型態 3：以 payload 當索引顯示章節戰場文字庫的一句（0x1AC2B..0x1AC49）。
//
// 型態 2 的 23 個處理器由 tools/extract_native_death_events.py 逐指令轉寫並核對，
// 再由 tools/sync_native_death_programs.py 降成各章劇本的 NativeDeathPrograms。
// 對白、pan、演出、生成群組沿用既有動作種類；其餘原語放在 NativeDeathOp。

/// NativeActionWhen 是處理器在動作之間讀取的條件。每個條件都在執行到該動作時
/// 才判斷，與原版讀取狀態的時間點相同。
///
/// Go：`native_death_program.go:20`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeActionWhen {
    /// NativeEventState[i] == v
    /// Go 標籤：`json:"state_eq,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub state_eq: Option<[i64; 2]>,
    /// NativeEventState[i] != v
    /// Go 標籤：`json:"state_ne,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub state_ne: Option<[i64; 2]>,
    /// [0x53BEF] < n（0x34883 jge）
    /// Go 標籤：`json:"round_lt,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub round_lt: Option<i64>,
    /// 索引區間內有任何一筆 +5 bit0 未設（0x3453E）
    /// Go 標籤：`json:"any_active,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub any_active: Option<[i64; 2]>,
}

/// NativeDeathOp 的欄位依 Op 取用；名稱只描述運算，不替 raw byte 猜玩法語意。
///
/// Go：`native_death_program.go:28`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeDeathOp {
    /// Go 標籤：`json:"op"`
    #[serde(deserialize_with = "null_as_default")]
    pub op: String,
    /// Go 標籤：`json:"unit,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub unit: i64,
    /// Go 標籤：`json:"first,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub first: i64,
    /// Go 標籤：`json:"last,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub last: i64,
    /// Go 標籤：`json:"mode,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub mode: i64,
    /// Go 標籤：`json:"value,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub value: i64,
    /// Go 標籤：`json:"mask,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub mask: i64,
    /// Go 標籤：`json:"slot,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub slot: i64,
    /// Go 標籤：`json:"delta,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub delta: i64,
    /// Go 標籤：`json:"index,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub index: i64,
    /// Go 標籤：`json:"x,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub x: i64,
    /// Go 標籤：`json:"y,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub y: i64,
    /// Go 標籤：`json:"group,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub group: i64,
    /// Go 標籤：`json:"kind,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub kind: i64,
    /// Go 標籤：`json:"writes,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub writes: Vec<[i64; 3]>,
}

/// NativeDeathEffectOf 回傳單位目前的死亡效果（型態, 值）。原始 +0x31..+0x33 優先，
/// 因為處理器可以把它改成 0xFF（事件 30）；沒有原始欄位時用資產的 death_effect。
///
/// Go：`native_death_program.go:48`，`func NativeDeathEffectOf(u *Unit) (int, int, bool)`
pub fn native_death_effect_of(u: Option<&Unit>) -> (i64, i64, bool) {
    let Some(u) = u else {
        return (0, 0, false);
    };
    if u.has_native_record_death_effect {
        let raw = u.native_record_death_effect;
        if raw[0] == 0xff {
            return (0, 0, false);
        }
        return (raw[0] as i64, raw[1] as i64 | (raw[2] as i64) << 8, true);
    }
    match &u.death_effect {
        None => (0, 0, false),
        Some(effect) => (effect.r#type, effect.value, true),
    }
}

impl Scenario {
    /// NativeDeathProgram 取出死亡效果對應的動作清單。型態 0／1 不走程式。
    ///
    /// Go：`native_death_program.go:66`，`func (sc *Scenario) NativeDeathProgram(u *Unit) ([]Action, string, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_death_program(&self, u: Option<&Unit>) -> (Vec<Action>, String, bool) {
        let (kind, value, ok) = native_death_effect_of(u);
        if !ok || (kind != 2 && kind != 3) {
            return (Vec::new(), String::new(), false);
        }
        let key = format!("{kind}:{value}");
        match self.native_death_programs.get(&key) {
            Some(program) => (program.clone(), key, true),
            None => (Vec::new(), key, false),
        }
    }

    /// NativeDeathCancelsExp 回答「擊倒這個單位會不會把本次行動的經驗清零」。
    /// 事件 30 的 0x34B48 在分派時寫 [0x53EC8]=0，而經驗要到 0x1196D 才發；重製端在
    /// 攻擊結算時就發經驗，所以在那之前先查。只有沒有條件的 exp_cancel 算數。
    ///
    /// Go：`native_death_program.go:79`，`func (sc *Scenario) NativeDeathCancelsExp(u *Unit) bool`
    pub fn native_death_cancels_exp(&self, u: Option<&Unit>) -> bool {
        let (program, _, ok) = self.native_death_program(u);
        if !ok {
            return false;
        }
        program.iter().any(|action| {
            action.native_death_op.as_ref().is_some_and(|op| op.op == "exp_cancel") && action.native_when.is_none()
        })
    }
}
impl NativeActionWhen {
    /// Match 判斷條件。缺少原版狀態時回錯誤，不以假值猜測。
    ///
    /// Go：`native_death_program.go:93`，`func (w *NativeActionWhen) Match(st *State) (bool, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn r#match(&self, st: Option<&State>) -> Result<bool> {
        let Some(st) = st else {
            return Err(battle_err!("native death condition requires battle state"));
        };
        let state = |pair: &[i64; 2]| -> Result<i64> {
            let index = pair[0];
            if index < 0 || index >= st.native_event_state.len() as i64 {
                return Err(battle_err!("native event-state index {index} outside table"));
            }
            Ok(st.native_event_state[index as usize] as i64)
        };
        if let Some(pair) = &self.state_eq
            && state(pair)? != pair[1]
        {
            return Ok(false);
        }
        if let Some(pair) = &self.state_ne
            && state(pair)? == pair[1]
        {
            return Ok(false);
        }
        if let Some(round_lt) = self.round_lt {
            if st.native_round_counter <= 0 {
                return Err(battle_err!("native round predicate requires [0x53BEF] provenance"));
            }
            if st.native_round_counter >= round_lt {
                return Ok(false);
            }
        }
        if let Some([first, last]) = self.any_active {
            let mut any = false;
            let mut index = first;
            while index <= last && index < st.units.len() as i64 {
                // Go 對負索引會 panic；這裡同樣以越界索引 panic。
                if !native_record_inactive(&st.units[index as usize]) {
                    any = true;
                    break;
                }
                index += 1;
            }
            if !any {
                return Ok(false);
            }
        }
        Ok(true)
    }
}

/// NativeRecordInactive 是 0x3453E：記錄 +5 bit0。沒有原版 +5 的單位退回 HP 與在場。
///
/// Go：`native_death_program.go:144`，`func NativeRecordInactive(u *Unit) bool`
pub fn native_record_inactive(u: &Unit) -> bool {
    if u.has_native_record_byte5 {
        return u.native_record_byte5 & 1 != 0;
    }
    !u.on_field || !u.alive()
}

impl State {
    /// ApplyNativeDeathOp 執行純狀態的原語。對白、演出、staging、全員倒下的呈現與
    /// 給物品由介面擁有者執行，這裡對它們回錯誤，避免有人以為已經生效。
    ///
    /// Go：`native_death_program.go:153`，`func (st *State) ApplyNativeDeathOp(op NativeDeathOp) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn apply_native_death_op(&mut self, op: &NativeDeathOp) -> Result<()> {
        // Go 的 each：first..=last 內、非負且在陣列內的索引。
        let range = |st: &State, first: i64, last: i64| -> Vec<usize> {
            let mut out = Vec::new();
            let mut index = first;
            while index <= last && index < st.units.len() as i64 {
                if index >= 0 {
                    out.push(index as usize);
                }
                index += 1;
            }
            out
        };
        match op.op.as_str() {
            "ai_mode_range" => {
                // 0x3419C：只改 +0x34 低四位。保留原始含端點範圍，只投影物化單位；
                // 未物化槽的 +0x34 在 0x10FB9 由 FDFIELD row+17 整 byte 覆寫，之後
                // 0x1100B 才增加有效筆數。證據：fd2_ch14_event10_20261002.json。
                if op.mode < 0 || op.mode > 0x0f || op.first < 0 || op.last < op.first {
                    return Err(battle_err!(
                        "ai_mode_range {}..{} mode {} outside raw range",
                        op.first,
                        op.last,
                        op.mode
                    ));
                }
                for index in range(self, op.first, op.last) {
                    let u = &mut self.units[index];
                    u.native_record_byte34 = u.native_record_byte34 & 0xf0 | op.mode as u8;
                    u.has_native_record_byte34 = true;
                    let value = u.native_record_byte34 as i64;
                    self.sync_native_runtime_record(index as i64, 0x34, value, 1);
                }
            }
            "ai_byte_set_range" => {
                // 0x34A85..0x34AA5：整個 +0x34 覆寫。
                for index in range(self, op.first, op.last) {
                    let u = &mut self.units[index];
                    (u.native_record_byte34, u.has_native_record_byte34) = (op.value as u8, true);
                    self.sync_native_runtime_record(index as i64, 0x34, op.value, 1);
                }
            }
            "ai_byte_and_range" => {
                // 0x34A0E：+0x34 &= mask。
                for index in range(self, op.first, op.last) {
                    let u = &mut self.units[index];
                    u.native_record_byte34 &= op.mask as u8;
                    u.has_native_record_byte34 = true;
                    let value = u.native_record_byte34 as i64;
                    self.sync_native_runtime_record(index as i64, 0x34, value, 1);
                }
            }
            "record_bytes" => {
                if op.unit < 0 {
                    return Err(battle_err!("record_bytes: runtime slot {} outside array", op.unit));
                }
                if op.unit >= self.units.len() as i64 {
                    // 原版照寫 raw 記錄陣列；那一筆之後被 0x10B4E 建構時會整筆覆寫，
                    // 所以寫入還沒建構的記錄在重製端是無作用。
                    return Ok(());
                }
                apply_native_record_writes(&mut self.units[op.unit as usize], &op.writes)?;
                for write in &op.writes {
                    self.sync_native_runtime_record(op.unit, write[0], write[1], write[2]);
                }
            }
            "control_turn" => {
                // [0x53A55]+3+3×slot = [0x53BEF] + delta。
                if !self.has_native_turn_event_control_state || self.native_round_counter <= 0 {
                    return Err(battle_err!("control_turn requires native turn-control and round provenance"));
                }
                if op.slot < 0 || op.slot >= self.native_turn_event_controls.len() as i64 {
                    return Err(battle_err!("control_turn slot {} outside table", op.slot));
                }
                let turn = self.native_round_counter.wrapping_add(op.delta);
                if !(0..=0xff).contains(&turn) {
                    return Err(battle_err!("control_turn value {turn} outside byte"));
                }
                self.native_turn_event_controls[op.slot as usize].turn = turn as u8;
                let offset = 3 + 3 * op.slot;
                if self.has_native_field_control_state && offset < self.native_field_control_raw.len() as i64 {
                    self.native_field_control_raw[offset as usize] = turn as u8;
                }
            }
            "state_set" | "state_inc" => {
                if op.index < 0 || op.index >= self.native_event_state.len() as i64 {
                    return Err(battle_err!("{} index {} outside table", op.op, op.index));
                }
                let slot = &mut self.native_event_state[op.index as usize];
                if op.op == "state_set" {
                    *slot = op.value as u8;
                } else {
                    *slot = slot.wrapping_add(1);
                }
            }
            "mark_inactive" => {
                // 0x32975：+5 整個 byte 寫成 1。還沒建構的記錄同 record_bytes，無作用。
                if op.unit < 0 {
                    return Err(battle_err!("mark_inactive: runtime slot {} outside array", op.unit));
                }
                if op.unit >= self.units.len() as i64 {
                    return Ok(());
                }
                let u = &mut self.units[op.unit as usize];
                u.on_field = false;
                (u.native_record_byte5, u.has_native_record_byte5) = (1, true);
                self.sync_native_runtime_record(op.unit, 0x05, 1, 1);
            }
            "range_one" => {
                // 0x35C18：互動時的地圖範圍選擇值。
                (self.native_map_range_mode, self.has_native_map_range_mode_state) = (1, true);
            }
            "range_zero" => {
                // 0x34638（第五章事件 15）：登場演出前把地圖範圍選擇值關掉。
                (self.native_map_range_mode, self.has_native_map_range_mode_state) = (0, true);
            }
            "exp_cancel" => {
                // 效果在攻擊結算時由 NativeDeathCancelsExp 先套用；這裡只保留位置。
            }
            _ => return Err(battle_err!("native death op {} has no state owner", go_quote(&op.op))),
        }
        Ok(())
    }

    /// ClearNativeHPFrom 是 0x35BBA 的狀態部分：從 first 起每筆 +0x40 清零。之後的
    /// 0x1DB65 呈現與 +5 標記由介面擁有者執行。
    ///
    /// Go：`native_death_program.go:259`，`func (st *State) ClearNativeHPFrom(first int) []*Unit`
    /// 所有權：回傳的單位是 `self.units` 的索引。
    pub fn clear_native_hp_from(&mut self, first: i64) -> Vec<UnitIdx> {
        let mut cleared = Vec::new();
        let mut index = first;
        while index >= 0 && index < self.units.len() as i64 {
            self.units[index as usize].hp = 0;
            self.sync_native_runtime_record(index, 0x40, 0, 2);
            cleared.push(index as usize);
            index += 1;
        }
        cleared
    }

    /// syncNativeRuntimeRecord 在戰場帶完整 0x50 raw 投影時（CONTINUE 載入）把同一筆
    /// 寫入也落到投影，讓後續以投影驗證的呈現與存檔看到同一個值。
    ///
    /// Go：`native_death_program.go:273`，`func (st *State) syncNativeRuntimeRecord(index, offset, value, width int)`
    pub(crate) fn sync_native_runtime_record(&mut self, index: i64, offset: i64, value: i64, width: i64) {
        if !self.has_native_runtime_unit_projection || index < 0 || index >= self.native_runtime_records.len() as i64 {
            return;
        }
        let raw = &mut self.native_runtime_records[index as usize].raw;
        if offset < 0 || offset + width > raw.len() as i64 {
            return;
        }
        raw[offset as usize] = value as u8;
        if width == 2 {
            raw[offset as usize + 1] = (value >> 8) as u8;
        }
    }
}

/// applyNativeRecordWrites 把直接記錄寫入落到型別化欄位。每個 offset 都要有對應
/// 的欄位，沒有就整筆拒絕，不留半套。
///
/// Go：`native_death_program.go:289`，`func applyNativeRecordWrites(u *Unit, writes [][3]int) error`
pub(crate) fn apply_native_record_writes(u: &mut Unit, writes: &[[i64; 3]]) -> Result<()> {
    // Go 先把每筆寫入收成 setter 閉包，全部通過才依序套用；Rust 用列舉表示同一件事。
    enum Setter {
        Byte5(u8),
        Byte6(u8, Camp),
        BattleFig(i64),
        Byte8(u8, i64),
        DeathEffect0(u8),
        Byte34(u8),
        Byte35(u8),
        Byte36(u8),
        Hp(i64),
    }
    let mut apply = Vec::with_capacity(writes.len());
    for write in writes {
        let (offset, value, width) = (write[0], write[1], write[2]);
        if (width == 1 && !(0..=0xff).contains(&value)) || (width == 2 && !(0..=0xffff).contains(&value)) {
            return Err(battle_err!("record write +{} value {} outside width {}", go_sharp_x(offset), value, width));
        }
        apply.push(match (offset, width) {
            (0x05, 1) => Setter::Byte5(value as u8),
            (0x06, 1) => Setter::Byte6(value as u8, native_raw_camp(value as u8)?),
            (0x07, 1) => Setter::BattleFig(value),
            (0x08, 1) => Setter::Byte8(value as u8, value),
            (0x31, 1) => Setter::DeathEffect0(value as u8),
            (0x34, 1) => Setter::Byte34(value as u8),
            // 第十五章event13在0x34ED9整byte寫0；沿用raw欄位及mode3消費端。
            (0x35, 1) => Setter::Byte35(value as u8),
            // event6 344B5直接寫入；本章mode4以+35／+36消費目的格。
            // READY來源見fd2_ch02_flow_20261017.json，不改其他caller的欄位語意。
            (0x36, 1) => Setter::Byte36(value as u8),
            (0x40, 2) => Setter::Hp(value),
            _ => return Err(battle_err!("record write +{} width {} has no typed field", go_sharp_x(offset), width)),
        });
    }
    for setter in apply {
        match setter {
            Setter::Byte5(v) => {
                (u.native_record_byte5, u.has_native_record_byte5) = (v, true);
                if v & 1 == 0 {
                    u.on_field = true;
                }
            }
            Setter::Byte6(v, camp) => {
                (u.native_record_byte6, u.has_native_record_byte6) = (v, true);
                u.camp = camp;
            }
            Setter::BattleFig(v) => (u.battle_fig, u.has_battle_fig) = (v, true),
            Setter::Byte8(v, identity) => {
                (u.native_record_byte8, u.has_native_record_byte8) = (v, true);
                if u.has_native_identity {
                    u.native_identity = identity;
                }
            }
            Setter::DeathEffect0(v) => {
                (u.native_record_death_effect[0], u.has_native_record_death_effect) = (v, true);
                if v == 0xff {
                    (u.death_effect, u.death_reward) = (None, None);
                }
            }
            Setter::Byte34(v) => (u.native_record_byte34, u.has_native_record_byte34) = (v, true),
            Setter::Byte35(v) => (u.native_record_byte35, u.has_native_record_byte35) = (v, true),
            Setter::Byte36(v) => (u.native_record_byte36, u.has_native_record_byte36) = (v, true),
            Setter::Hp(v) => u.hp = v,
        }
    }
    Ok(())
}

/// nativeRawCamp 是地圖資料實測的對應：+6 為 0 敵方、1 友軍、2 我方。
///
/// Go：`native_death_program.go:352`，`func nativeRawCamp(raw byte) (Camp, error)`
pub(crate) fn native_raw_camp(raw: u8) -> Result<Camp> {
    match raw {
        0 => Ok(ENEMY),
        1 => Ok(ALLY),
        2 => Ok(OWN),
        _ => Err(battle_err!("raw camp {raw} has no engine camp")),
    }
}

impl State {
    /// killCancelsExp 是發經驗前的查詢：本次行動擊倒的任何一個單位帶無條件的
    /// exp_cancel，整次行動的經驗就是 0（原版清的是整次行動累積的 [0x53EC8]）。
    ///
    /// Go：`native_death_program.go:366`，`func (st *State) killCancelsExp(targets ...*Unit) bool`
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn kill_cancels_exp(&self, targets: &[UnitIdx]) -> bool {
        let Some(cancel) = &self.native_death_exp_cancel else {
            return false;
        };
        targets.iter().any(|&target| self.units[target].hp <= 0 && cancel(&self.units[target]))
    }
}

#[cfg(test)]
mod native_death_program_test;
