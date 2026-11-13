//! Go 來源：`remake/internal/battle/native_result.go`（125 行）。

use super::*;

/// NativeResultRule 保存已閉合的章節結果覆寫；Source 只是原始 writer 的
/// 出處，正式執行依具型別條件，不依 FD2.EXE 位址分派。
/// READY 契約與主證據見56及fd2_ch13_result_conditions_20261001.json、
/// fd2_ch15_result_conditions_20261002.json；原版無對白列的Actions為空。
///
/// Go：`native_result.go:9`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeResultRule {
    /// Go 標籤：`json:"id"`
    pub id: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"records_all_inactive"`
    pub records_all_inactive: Vec<i64>,
    /// Go 標籤：`json:"event_states_all_nonzero,omitempty"`
    pub event_states_all_nonzero: Vec<i64>,
    /// Go 標籤：`json:"round_greater_than,omitempty"`
    pub round_greater_than: Option<i64>,
    /// Go 標籤：`json:"code"`
    pub code: i64,
    /// Go 標籤：`json:"actions"`
    pub actions: Vec<Action>,
}

impl NativeResultRule {
    /// Go：`native_result.go:19`，`func (r NativeResultRule) Validate() error`
    pub fn validate(&self) -> Result<()> {
        todo!("battle/native_result.go:NativeResultRule.Validate")
    }

    /// Match 在該列真正執行時讀 raw 狀態；回合閘先於記錄查詢，不能因
    /// 尚未生成的59而提前拒絕，也不能用HP代替sub_3453E的+5 bit0。
    ///
    /// Go：`native_result.go:55`，`func (r NativeResultRule) Match(st *State) (bool, error)`
    pub fn r#match(&self, st: Option<&State>) -> Result<bool> {
        todo!("battle/native_result.go:NativeResultRule.Match")
    }
}

/// NativeInitialResultCode preserves 20B72's event-state-only result owner.
/// READY: fd2_ch29_flow_20261017.json chapter_result_contract.
///
/// Go：`native_result.go:92`，`func NativeInitialResultCode(st *State, sc *Scenario) (int, error)`
pub fn native_initial_result_code(st: Option<&State>, sc: Option<&Scenario>) -> Result<i64> {
    todo!("battle/native_result.go:NativeInitialResultCode")
}

/// NativeDefaultResultCode 是sub_205BE的已閉合三值基礎規則：僅掃已生成的
/// runtime陣列；raw+6=0且+5 bit0=0使結果2改0，最後記錄0的bit0使其改1。
///
/// Go：`native_result.go:108`，`func NativeDefaultResultCode(st *State) (int, error)`
pub fn native_default_result_code(st: Option<&State>) -> Result<i64> {
    todo!("battle/native_result.go:NativeDefaultResultCode")
}
