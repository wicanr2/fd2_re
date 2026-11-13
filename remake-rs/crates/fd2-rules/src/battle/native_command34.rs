//! Go 來源：`remake/internal/battle/native_command34.go`（161 行）。

use super::*;

/// Go：`native_command34.go:5`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand34TargetState {
    pub target: Option<UnitIdx>,
    pub native_transient: [u8; 6],
    pub ap: i64,
    pub dp: i64,
    pub hit: i64,
    pub ev: i64,
}

/// Go：`native_command34.go:11`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand34Result {
    pub stages: Vec<NativeCommandModifierResult>,
    pub stage_states: Vec<Vec<NativeCommand34TargetState>>,
    pub rng_state: u16,
}

/// Go：`native_command34.go:17`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand34Plan {
    pub actor: Option<UnitIdx>,
    pub result: NativeCompoundCommand34Result,
    pub(crate) before: Vec<NativeCommand34TargetState>,
    pub(crate) published: i64,
    pub(crate) completed: bool,
}

/// Go：`native_command34.go:25`，`func nativeCommand34States(targets []*Unit) []NativeCommand34TargetState`
pub(crate) fn native_command34_states(targets: &[Unit]) -> Vec<NativeCommand34TargetState> {
    todo!("battle/native_command34.go:nativeCommand34States")
}

/// Go：`native_command34.go:33`，`func nativeCommand34Publish(states []NativeCommand34TargetState)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command34_publish(units: &mut [Unit], states: &[NativeCommand34TargetState]) {
    todo!("battle/native_command34.go:nativeCommand34Publish")
}

/// Go：`native_command34.go:40`，`func nativeCommand34EqualCurrent(states []NativeCommand34TargetState) bool`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command34_equal_current(units: &[Unit], states: &[NativeCommand34TargetState]) -> bool {
    todo!("battle/native_command34.go:nativeCommand34EqualCurrent")
}

impl State {
    /// PlanNativeCompoundCommand34 computes the three raw stages without mutation.
    /// The indexed owner publishes each stage only after its mask frames are drawn.
    ///
    /// Go：`native_command34.go:51`，`func (s *State) PlanNativeCompoundCommand34(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand34Plan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_compound_command34(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand34Plan> {
        todo!("battle/native_command34.go:State.PlanNativeCompoundCommand34")
    }
}

/// Go：`native_command34.go:97`，`func PublishNativeCompoundCommand34Stage(plan *NativeCompoundCommand34Plan, stage int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_compound_command34_stage(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand34Plan>,
    stage: i64,
) -> Result<()> {
    todo!("battle/native_command34.go:PublishNativeCompoundCommand34Stage")
}

/// Go：`native_command34.go:113`，`func CompleteNativeCompoundCommand34(plan *NativeCompoundCommand34Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command34(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand34Plan>,
) -> Result<()> {
    todo!("battle/native_command34.go:CompleteNativeCompoundCommand34")
}

/// Go：`native_command34.go:121`，`func AbortNativeCompoundCommand34(plan *NativeCompoundCommand34Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_compound_command34(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand34Plan>,
) -> Result<()> {
    todo!("battle/native_command34.go:AbortNativeCompoundCommand34")
}

impl State {
    /// Go：`native_command34.go:136`，`func (s *State) ExecuteNativeCompoundCommand34(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand34Result, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_compound_command34(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand34Result> {
        todo!("battle/native_command34.go:State.ExecuteNativeCompoundCommand34")
    }
}

/// Go：`native_command34.go:154`，`func nativeCompoundPlayerSelector(selector int) bool`
pub(crate) fn native_compound_player_selector(selector: i64) -> bool {
    todo!("battle/native_command34.go:nativeCompoundPlayerSelector")
}
