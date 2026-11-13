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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeActionWhen {
    /// NativeEventState[i] == v
    /// Go 標籤：`json:"state_eq,omitempty"`
    pub state_eq: Option<[i64; 2]>,
    /// NativeEventState[i] != v
    /// Go 標籤：`json:"state_ne,omitempty"`
    pub state_ne: Option<[i64; 2]>,
    /// [0x53BEF] < n（0x34883 jge）
    /// Go 標籤：`json:"round_lt,omitempty"`
    pub round_lt: Option<i64>,
    /// 索引區間內有任何一筆 +5 bit0 未設（0x3453E）
    /// Go 標籤：`json:"any_active,omitempty"`
    pub any_active: Option<[i64; 2]>,
}

/// NativeDeathOp 的欄位依 Op 取用；名稱只描述運算，不替 raw byte 猜玩法語意。
///
/// Go：`native_death_program.go:28`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeDeathOp {
    /// Go 標籤：`json:"op"`
    pub op: String,
    /// Go 標籤：`json:"unit,omitempty"`
    pub unit: i64,
    /// Go 標籤：`json:"first,omitempty"`
    pub first: i64,
    /// Go 標籤：`json:"last,omitempty"`
    pub last: i64,
    /// Go 標籤：`json:"mode,omitempty"`
    pub mode: i64,
    /// Go 標籤：`json:"value,omitempty"`
    pub value: i64,
    /// Go 標籤：`json:"mask,omitempty"`
    pub mask: i64,
    /// Go 標籤：`json:"slot,omitempty"`
    pub slot: i64,
    /// Go 標籤：`json:"delta,omitempty"`
    pub delta: i64,
    /// Go 標籤：`json:"index,omitempty"`
    pub index: i64,
    /// Go 標籤：`json:"x,omitempty"`
    pub x: i64,
    /// Go 標籤：`json:"y,omitempty"`
    pub y: i64,
    /// Go 標籤：`json:"group,omitempty"`
    pub group: i64,
    /// Go 標籤：`json:"kind,omitempty"`
    pub kind: i64,
    /// Go 標籤：`json:"writes,omitempty"`
    pub writes: Vec<[i64; 3]>,
}

/// NativeDeathEffectOf 回傳單位目前的死亡效果（型態, 值）。原始 +0x31..+0x33 優先，
/// 因為處理器可以把它改成 0xFF（事件 30）；沒有原始欄位時用資產的 death_effect。
///
/// Go：`native_death_program.go:48`，`func NativeDeathEffectOf(u *Unit) (int, int, bool)`
pub fn native_death_effect_of(u: Option<&Unit>) -> (i64, i64, bool) {
    todo!("battle/native_death_program.go:NativeDeathEffectOf")
}

impl Scenario {
    /// NativeDeathProgram 取出死亡效果對應的動作清單。型態 0／1 不走程式。
    ///
    /// Go：`native_death_program.go:66`，`func (sc *Scenario) NativeDeathProgram(u *Unit) ([]Action, string, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_death_program(&self, u: Option<&Unit>) -> (Vec<Action>, String, bool) {
        todo!("battle/native_death_program.go:Scenario.NativeDeathProgram")
    }

    /// NativeDeathCancelsExp 回答「擊倒這個單位會不會把本次行動的經驗清零」。
    /// 事件 30 的 0x34B48 在分派時寫 [0x53EC8]=0，而經驗要到 0x1196D 才發；重製端在
    /// 攻擊結算時就發經驗，所以在那之前先查。只有沒有條件的 exp_cancel 算數。
    ///
    /// Go：`native_death_program.go:79`，`func (sc *Scenario) NativeDeathCancelsExp(u *Unit) bool`
    pub fn native_death_cancels_exp(&self, u: Option<&Unit>) -> bool {
        todo!("battle/native_death_program.go:Scenario.NativeDeathCancelsExp")
    }
}
impl NativeActionWhen {
    /// Match 判斷條件。缺少原版狀態時回錯誤，不以假值猜測。
    ///
    /// Go：`native_death_program.go:93`，`func (w *NativeActionWhen) Match(st *State) (bool, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn r#match(&self, st: Option<&State>) -> Result<bool> {
        todo!("battle/native_death_program.go:NativeActionWhen.Match")
    }
}

/// NativeRecordInactive 是 0x3453E：記錄 +5 bit0。沒有原版 +5 的單位退回 HP 與在場。
///
/// Go：`native_death_program.go:144`，`func NativeRecordInactive(u *Unit) bool`
pub fn native_record_inactive(u: &Unit) -> bool {
    todo!("battle/native_death_program.go:NativeRecordInactive")
}

impl State {
    /// ApplyNativeDeathOp 執行純狀態的原語。對白、演出、staging、全員倒下的呈現與
    /// 給物品由介面擁有者執行，這裡對它們回錯誤，避免有人以為已經生效。
    ///
    /// Go：`native_death_program.go:153`，`func (st *State) ApplyNativeDeathOp(op NativeDeathOp) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn apply_native_death_op(&mut self, op: &NativeDeathOp) -> Result<()> {
        todo!("battle/native_death_program.go:State.ApplyNativeDeathOp")
    }

    /// ClearNativeHPFrom 是 0x35BBA 的狀態部分：從 first 起每筆 +0x40 清零。之後的
    /// 0x1DB65 呈現與 +5 標記由介面擁有者執行。
    ///
    /// Go：`native_death_program.go:259`，`func (st *State) ClearNativeHPFrom(first int) []*Unit`
    /// 所有權：回傳的單位是 `self.units` 的索引。
    pub fn clear_native_hp_from(&mut self, first: i64) -> Vec<UnitIdx> {
        todo!("battle/native_death_program.go:State.ClearNativeHPFrom")
    }

    /// syncNativeRuntimeRecord 在戰場帶完整 0x50 raw 投影時（CONTINUE 載入）把同一筆
    /// 寫入也落到投影，讓後續以投影驗證的呈現與存檔看到同一個值。
    ///
    /// Go：`native_death_program.go:273`，`func (st *State) syncNativeRuntimeRecord(index, offset, value, width int)`
    pub(crate) fn sync_native_runtime_record(&mut self, index: i64, offset: i64, value: i64, width: i64) {
        todo!("battle/native_death_program.go:State.syncNativeRuntimeRecord")
    }
}

/// applyNativeRecordWrites 把直接記錄寫入落到型別化欄位。每個 offset 都要有對應
/// 的欄位，沒有就整筆拒絕，不留半套。
///
/// Go：`native_death_program.go:289`，`func applyNativeRecordWrites(u *Unit, writes [][3]int) error`
pub(crate) fn apply_native_record_writes(u: &mut Unit, writes: &[[i64; 3]]) -> Result<()> {
    todo!("battle/native_death_program.go:applyNativeRecordWrites")
}

/// nativeRawCamp 是地圖資料實測的對應：+6 為 0 敵方、1 友軍、2 我方。
///
/// Go：`native_death_program.go:352`，`func nativeRawCamp(raw byte) (Camp, error)`
pub(crate) fn native_raw_camp(raw: u8) -> Result<Camp> {
    todo!("battle/native_death_program.go:nativeRawCamp")
}

impl State {
    /// killCancelsExp 是發經驗前的查詢：本次行動擊倒的任何一個單位帶無條件的
    /// exp_cancel，整次行動的經驗就是 0（原版清的是整次行動累積的 [0x53EC8]）。
    ///
    /// Go：`native_death_program.go:366`，`func (st *State) killCancelsExp(targets ...*Unit) bool`
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn kill_cancels_exp(&self, targets: &[UnitIdx]) -> bool {
        todo!("battle/native_death_program.go:State.killCancelsExp")
    }
}
