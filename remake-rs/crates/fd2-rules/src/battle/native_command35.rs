//! Go 來源：`remake/internal/battle/native_command35.go`（164 行）。

use super::*;

/// Go：`native_command35.go:8`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand35Stage {
    pub command_id: i64,
    pub marker_offset: i64,
    pub results: Vec<NativeRawApplicationResult>,
    pub accumulator: i64,
    pub rng_state: u16,
}

/// Go：`native_command35.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand35TargetState {
    pub target: Option<UnitIdx>,
    pub hp: i64,
    pub native_transient: [u8; 6],
}

/// Go：`native_command35.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand35Result {
    pub stages: Vec<NativeCompoundCommand35Stage>,
    pub stage_states: Vec<Vec<NativeCommand35TargetState>>,
    pub rng_state: u16,
}

/// Go：`native_command35.go:28`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand35Plan {
    pub actor: Option<UnitIdx>,
    pub result: NativeCompoundCommand35Result,
    pub(crate) before: Vec<NativeCommand35TargetState>,
    pub(crate) published: i64,
    pub(crate) completed: bool,
}

/// Go：`native_command35.go:36`，`func nativeCommand35States(targets []*Unit) []NativeCommand35TargetState`
pub(crate) fn native_command35_states(targets: &[Unit]) -> Vec<NativeCommand35TargetState> {
    todo!("battle/native_command35.go:nativeCommand35States")
}

/// Go：`native_command35.go:44`，`func nativeCommand35Publish(states []NativeCommand35TargetState)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command35_publish(units: &mut [Unit], states: &[NativeCommand35TargetState]) {
    todo!("battle/native_command35.go:nativeCommand35Publish")
}

/// Go：`native_command35.go:50`，`func nativeCommand35EqualCurrent(states []NativeCommand35TargetState) bool`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command35_equal_current(units: &[Unit], states: &[NativeCommand35TargetState]) -> bool {
    todo!("battle/native_command35.go:nativeCommand35EqualCurrent")
}

impl State {
    /// Go：`native_command35.go:59`，`func (s *State) PlanNativeCompoundCommand35(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand35Plan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_compound_command35(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand35Plan> {
        todo!("battle/native_command35.go:State.PlanNativeCompoundCommand35")
    }
}

/// Go：`native_command35.go:109`，`func PublishNativeCompoundCommand35Stage(plan *NativeCompoundCommand35Plan, stage int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_compound_command35_stage(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand35Plan>,
    stage: i64,
) -> Result<()> {
    todo!("battle/native_command35.go:PublishNativeCompoundCommand35Stage")
}

/// Go：`native_command35.go:125`，`func CompleteNativeCompoundCommand35(plan *NativeCompoundCommand35Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command35(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand35Plan>,
) -> Result<()> {
    todo!("battle/native_command35.go:CompleteNativeCompoundCommand35")
}

/// Go：`native_command35.go:133`，`func AbortNativeCompoundCommand35(plan *NativeCompoundCommand35Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_compound_command35(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand35Plan>,
) -> Result<()> {
    todo!("battle/native_command35.go:AbortNativeCompoundCommand35")
}

impl State {
    /// Go：`native_command35.go:148`，`func (s *State) ExecuteNativeCompoundCommand35(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand35Result, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_compound_command35(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand35Result> {
        todo!("battle/native_command35.go:State.ExecuteNativeCompoundCommand35")
    }
}
