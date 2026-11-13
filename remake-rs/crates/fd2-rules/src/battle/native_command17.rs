//! Go 來源：`remake/internal/battle/native_command17.go`（301 行）。

use super::*;

impl State {
    /// NativeCommandModifierTargets validates the recovered player selector for
    /// command IDs 17..19 without mutating state. The player renderer remains
    /// intentionally disconnected until its indexed presentation is recovered.
    ///
    /// Go：`native_command17.go:12`，`func (s *State) NativeCommandModifierTargets(actor, confirmed *Unit, commandID int) ([]*Unit, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_modifier_targets(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
    ) -> Result<Vec<UnitIdx>> {
        todo!("battle/native_command17.go:State.NativeCommandModifierTargets")
    }

    /// NativeAICommandModifierTargets rebuilds the final target array through the
    /// recovered 0x15311 raw-selector owner. It does not reuse the player's
    /// confirmed-cursor predicate.
    ///
    /// Go：`native_command17.go:38`，`func (s *State) NativeAICommandModifierTargets(actor *Unit, origin Cell, commandID int) ([]*Unit, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_ai_command_modifier_targets(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
    ) -> Result<Vec<UnitIdx>> {
        todo!("battle/native_command17.go:State.NativeAICommandModifierTargets")
    }

    /// ExecuteNativeCommandModifier applies the player selector and the exact
    /// ID17..19 raw transaction. No presentation or status name is inferred.
    ///
    /// Go：`native_command17.go:88`，`func (s *State) ExecuteNativeCommandModifier(actor, confirmed *Unit, commandID int, rngState uint16) (NativeCommandModifierResult, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command_modifier(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeCommandModifierResult> {
        todo!("battle/native_command17.go:State.ExecuteNativeCommandModifier")
    }

    /// ExecuteNativeAICommandModifier consumes only the raw target array rebuilt
    /// for the recovered mode-11 owner.
    ///
    /// Go：`native_command17.go:98`，`func (s *State) ExecuteNativeAICommandModifier(actor *Unit, origin Cell, commandID int, rngState uint16) (NativeCommandModifierResult, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_ai_command_modifier(
        &mut self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeCommandModifierResult> {
        todo!("battle/native_command17.go:State.ExecuteNativeAICommandModifier")
    }
}

/// NativeAICommandModifierPlan keeps the single 0x226EA/0x2282F/0x22960
/// transaction private until the handler-owned mask boundary has been drawn.
///
/// Go：`native_command17.go:116`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAICommandModifierPlan {
    pub actor: Option<UnitIdx>,
    pub command_id: i64,
    pub targets: Vec<UnitIdx>,
    pub result: NativeCommandModifierResult,
    pub after: Vec<NativeCommand34TargetState>,
    pub(crate) mp_before: i64,
    pub(crate) mp_after: i64,
    pub(crate) before: Vec<NativeCommand34TargetState>,
    pub(crate) published: bool,
    pub(crate) completed: bool,
}

impl State {
    /// Go：`native_command17.go:129`，`func (s *State) PlanNativeAICommandModifier(actor *Unit, origin Cell, commandID int, rngState uint16) (*NativeAICommandModifierPlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_ai_command_modifier(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeAICommandModifierPlan> {
        todo!("battle/native_command17.go:State.PlanNativeAICommandModifier")
    }
}

/// Go：`native_command17.go:173`，`func PublishNativeAICommandModifier(plan *NativeAICommandModifierPlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_ai_command_modifier(
    units: &mut [Unit],
    plan: Option<&mut NativeAICommandModifierPlan>,
) -> Result<()> {
    todo!("battle/native_command17.go:PublishNativeAICommandModifier")
}

/// Go：`native_command17.go:184`，`func CompleteNativeAICommandModifier(plan *NativeAICommandModifierPlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_ai_command_modifier(
    units: &mut [Unit],
    plan: Option<&mut NativeAICommandModifierPlan>,
) -> Result<()> {
    todo!("battle/native_command17.go:CompleteNativeAICommandModifier")
}

/// Go：`native_command17.go:194`，`func AbortNativeAICommandModifier(plan *NativeAICommandModifierPlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_ai_command_modifier(
    units: &mut [Unit],
    plan: Option<&mut NativeAICommandModifierPlan>,
) -> Result<()> {
    todo!("battle/native_command17.go:AbortNativeAICommandModifier")
}

impl State {
    /// ValidateNativeCommandModifierTargets performs the complete raw projection
    /// preflight without consuming RNG or mutating any unit. Presentation owners
    /// call this before publishing sample/palette frames.
    ///
    /// Go：`native_command17.go:213`，`func (s *State) ValidateNativeCommandModifierTargets(actor *Unit, targets []*Unit, commandID int) error`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    pub fn validate_native_command_modifier_targets(
        &self,
        actor: Option<UnitIdx>,
        targets: &[UnitIdx],
        command_id: i64,
    ) -> Result<()> {
        todo!("battle/native_command17.go:State.ValidateNativeCommandModifierTargets")
    }

    /// Go：`native_command17.go:221`，`func (s *State) preflightNativeCommandModifierActor(actor *Unit, commandID int) error`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn preflight_native_command_modifier_actor(
        &self,
        actor: Option<UnitIdx>,
        command_id: i64,
    ) -> Result<()> {
        todo!("battle/native_command17.go:State.preflightNativeCommandModifierActor")
    }

    /// Go：`native_command17.go:239`，`func (s *State) executeNativeCommandModifierTargets(actor *Unit, targets []*Unit, commandID int, rngState uint16) (NativeCommandModifierResult, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    pub(crate) fn execute_native_command_modifier_targets(
        &mut self,
        actor: Option<UnitIdx>,
        targets: &[UnitIdx],
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeCommandModifierResult> {
        todo!("battle/native_command17.go:State.executeNativeCommandModifierTargets")
    }
}

/// Go：`native_command17.go:263`，`func publishNativeCommandModifierRecords(targets []*Unit, records []byte)`
pub(crate) fn publish_native_command_modifier_records(targets: &mut [Unit], records: &[u8]) {
    todo!("battle/native_command17.go:publishNativeCommandModifierRecords")
}

/// Go：`native_command17.go:274`，`func nativeCommandModifierRecords(targets []*Unit) ([]byte, []byte, error)`
pub(crate) fn native_command_modifier_records(targets: &[Unit]) -> Result<(Vec<u8>, Vec<u8>)> {
    todo!("battle/native_command17.go:nativeCommandModifierRecords")
}
