//! Go 來源：`remake/internal/battle/native_system_group_march.go`（214 行）。

use super::*;

/// NativeSystemGroupMarchStep 保存 sub_16F55 selector1 對一筆合格 runtime
/// record 的可重播結果。Path 含起點與終點；本切片只接受不觸發全域事件的路徑。
///
/// Go：`native_system_group_march.go:7`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSystemGroupMarchStep {
    pub unit_index: i64,
    pub path: Vec<Cell>,
    pub events: Vec<NativeSystemGroupMarchEvent>,
}

/// NativeSystemGroupMarchEvent 保存一個已在私有投影驗證的 selector1 中途事件。
/// PathIndex 指向 Path 內事件格；TextIndex／Presentation 只供正式 UI owner 在
/// 關閉確認框前驗證既有可編輯素材，不替 handler 新增語意。
///
/// Go：`native_system_group_march.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeSystemGroupMarchEvent {
    pub path_index: i64,
    pub event_id: u8,
    pub text_index: i64,
    pub presentation: bool,
}

/// NativeSystemGroupMarchPlan 是確認視窗關閉前完成的私有預演結果。
///
/// Go：`native_system_group_march.go:24`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSystemGroupMarchPlan {
    pub destination: Cell,
    pub steps: Vec<NativeSystemGroupMarchStep>,
}

impl State {
    /// PlanNativeSystemGroupMarch 重現 0x16FF4..0x1716A 的無事件產品子集：依
    /// runtime record 順序選 (raw+5)&0x85==0、raw+6==2 的單位，逐筆以 selector1
    /// 執行已閉合的 0x14B78 尋路。占位會在私有投影逐筆更新，任一未知事件或來源
    /// 缺失都讓整批失敗，不發布部分移動。
    ///
    /// Go：`native_system_group_march.go:33`，`func (s *State) PlanNativeSystemGroupMarch(destination Cell) (NativeSystemGroupMarchPlan, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_system_group_march(&mut self, destination: Cell) -> Result<NativeSystemGroupMarchPlan> {
        todo!("battle/native_system_group_march.go:State.PlanNativeSystemGroupMarch")
    }
}

/// Go：`native_system_group_march.go:112`，`func cloneNativeSystemGroupMarchUnit(unit *Unit) *Unit`
pub(crate) fn clone_native_system_group_march_unit(unit: Option<&Unit>) -> Option<Unit> {
    todo!("battle/native_system_group_march.go:cloneNativeSystemGroupMarchUnit")
}

/// Go：`native_system_group_march.go:125`，`func cloneNativeSystemGroupMarchState(source *State) (State, error)`
pub(crate) fn clone_native_system_group_march_state(source: &mut State) -> Result<State> {
    todo!("battle/native_system_group_march.go:cloneNativeSystemGroupMarchState")
}

/// Go：`native_system_group_march.go:148`，`func planNativeSystemGroupMarchEvent( working *State, trigger *Unit, pathIndex int, eventID byte, ) (NativeSystemGroupMarchEvent, error)`
/// 所有權：`trigger` 是 `working.units` 的索引（Go 傳 `*Unit`）。
pub(crate) fn plan_native_system_group_march_event(
    working: Option<&mut State>,
    trigger: Option<UnitIdx>,
    path_index: i64,
    event_id: u8,
) -> Result<NativeSystemGroupMarchEvent> {
    todo!("battle/native_system_group_march.go:planNativeSystemGroupMarchEvent")
}

impl State {
    /// CommitNativeSystemGroupMarchStep 原子發布一筆已預演移動；動畫 owner 必須在
    /// path 完整呈現後呼叫。它對應 selector1 每筆的姿態清零與 0x13512 bit7 writer。
    ///
    /// Go：`native_system_group_march.go:189`，`func (s *State) CommitNativeSystemGroupMarchStep(step NativeSystemGroupMarchStep) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn commit_native_system_group_march_step(&mut self, step: &NativeSystemGroupMarchStep) -> Result<()> {
        todo!("battle/native_system_group_march.go:State.CommitNativeSystemGroupMarchStep")
    }
}
