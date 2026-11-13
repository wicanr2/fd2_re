//! Go 來源：`remake/internal/battle/native_command32.go`（172 行）。

use super::*;

/// NativeCompoundCommand32Target preserves the per-target 0x1C75E transaction
/// result consumed by the separate indexed presentation owner.
///
/// Go：`native_command32.go:7`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand32Target {
    pub target: Option<UnitIdx>,
    pub damage: NativeCommandDamage,
    pub hp_before: i64,
    pub hp_after: i64,
}

/// NativeCompoundCommand32Result is the bounded class-19 player state result.
///
/// Go：`native_command32.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand32Result {
    pub targets: Vec<NativeCompoundCommand32Target>,
    pub rng_state: u16,
}

/// NativeCompoundCommand32Plan separates the proven per-target 0x1C75E
/// publication markers from final actor completion so the indexed owner can
/// fail closed without rerolling RNG or partially committing HP.
///
/// Go：`native_command32.go:23`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCompoundCommand32Plan {
    pub actor: Option<UnitIdx>,
    pub result: NativeCompoundCommand32Result,
    pub actor_acted_before: bool,
    pub rng_before: u16,
    pub(crate) targets: Vec<NativeCompoundCommand32Target>,
    pub(crate) published: Vec<bool>,
    pub(crate) completed: bool,
}

impl State {
    /// ExecuteNativeCompoundCommand32 stages the proven 0x2111A -> 0x1C75E target
    /// loop privately. The five reachable FIGANI resources bypass the known MP
    /// debit sink, so record32 is an availability gate only on this player path.
    ///
    /// Go：`native_command32.go:36`，`func (s *State) ExecuteNativeCompoundCommand32(actor, confirmed *Unit, rngState uint16) (NativeCompoundCommand32Result, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_compound_command32(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand32Result> {
        todo!("battle/native_command32.go:State.ExecuteNativeCompoundCommand32")
    }

    /// PlanNativeCompoundCommand32 performs the complete target/RNG preflight
    /// without mutating actor, targets, or the caller-owned RNG state.
    ///
    /// Go：`native_command32.go:56`，`func (s *State) PlanNativeCompoundCommand32(actor, confirmed *Unit, rngState uint16) (*NativeCompoundCommand32Plan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_compound_command32(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCompoundCommand32Plan> {
        todo!("battle/native_command32.go:State.PlanNativeCompoundCommand32")
    }
}

/// Go：`native_command32.go:123`，`func ApplyNativeCompoundCommand32Target(plan *NativeCompoundCommand32Plan, index int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_compound_command32_target(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand32Plan>,
    index: i64,
) -> Result<()> {
    todo!("battle/native_command32.go:ApplyNativeCompoundCommand32Target")
}

/// Go：`native_command32.go:137`，`func CompleteNativeCompoundCommand32(plan *NativeCompoundCommand32Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command32(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand32Plan>,
) -> Result<()> {
    todo!("battle/native_command32.go:CompleteNativeCompoundCommand32")
}

/// AbortNativeCompoundCommand32 restores every already-published target after
/// preflighting the complete rollback. It never overwrites state changed by an
/// outside owner and therefore remains fail closed at presentation boundaries.
///
/// Go：`native_command32.go:155`，`func AbortNativeCompoundCommand32(plan *NativeCompoundCommand32Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_compound_command32(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand32Plan>,
) -> Result<()> {
    todo!("battle/native_command32.go:AbortNativeCompoundCommand32")
}
