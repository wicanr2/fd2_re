//! Go 來源：`remake/internal/battle/native_command33.go`（203 行）。

use super::*;

/// NativeCompoundCommand33Result exposes the proven raw HP batch, target
/// snapshots and RNG boundary. The indexed owner consumes these snapshots;
/// the score/EXP consumer remains unresolved.
///
/// Go：`native_command33.go:12`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand33Result {
    pub restore: NativeRawRestoreBatch,
    pub targets: Vec<NativeCompoundCommand33Target>,
}

/// Go：`native_command33.go:17`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand33Target {
    pub target: Option<UnitIdx>,
    pub hp_before: i64,
    pub hp_after: i64,
    pub transient_before: [u8; 6],
    pub transient_after: [u8; 6],
}

/// Go：`native_command33.go:23`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct NativeCompoundCommand33TargetInner {
    pub(crate) target: Option<UnitIdx>,
    pub(crate) hp_before: i64,
    pub(crate) hp_after: i64,
    pub(crate) transient_before: [u8; 6],
    pub(crate) after: [u8; 6],
}

/// NativeCompoundCommand33Plan keeps the 0x211A4 clear/restore batch private
/// until the indexed owner reaches the post-mask publication boundary.
///
/// Go：`native_command33.go:31`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand33Plan {
    pub actor: Option<UnitIdx>,
    pub result: NativeCompoundCommand33Result,
    pub actor_acted_before: bool,
    pub rng_before: u16,
    pub(crate) targets: Vec<NativeCompoundCommand33TargetInner>,
    pub(crate) published: bool,
    pub(crate) completed: bool,
}

impl State {
    /// ExecuteNativeCompoundCommand33 implements the bounded player class-19 state
    /// transaction at 0x285A1..0x285ED. Proven player FIGANI resources bypass the
    /// known MP debit sink, so record33 remains an availability gate only.
    ///
    /// Go：`native_command33.go:44`，`func (s *State) ExecuteNativeCompoundCommand33(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand33Result, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_compound_command33(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand33Result> {
        todo!("battle/native_command33.go:State.ExecuteNativeCompoundCommand33")
    }

    /// PlanNativeCompoundCommand33 computes the complete raw clear/restore result
    /// without mutating the actor, targets or caller-owned RNG state.
    ///
    /// Go：`native_command33.go:61`，`func (s *State) PlanNativeCompoundCommand33(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand33Plan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_compound_command33(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand33Plan> {
        todo!("battle/native_command33.go:State.PlanNativeCompoundCommand33")
    }
}

/// Go：`native_command33.go:125`，`func PublishNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_compound_command33(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand33Plan>,
) -> Result<()> {
    todo!("battle/native_command33.go:PublishNativeCompoundCommand33")
}

/// Go：`native_command33.go:145`，`func CompleteNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command33(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand33Plan>,
) -> Result<()> {
    todo!("battle/native_command33.go:CompleteNativeCompoundCommand33")
}

/// Go：`native_command33.go:160`，`func AbortNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_compound_command33(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand33Plan>,
) -> Result<()> {
    todo!("battle/native_command33.go:AbortNativeCompoundCommand33")
}

/// Go：`native_command33.go:181`，`func nativeCompoundCommandTargetRecords(targets []*Unit) ([]byte, []byte, error)`
pub(crate) fn native_compound_command_target_records(targets: &[Unit]) -> Result<(Vec<u8>, Vec<u8>)> {
    todo!("battle/native_command33.go:nativeCompoundCommandTargetRecords")
}
