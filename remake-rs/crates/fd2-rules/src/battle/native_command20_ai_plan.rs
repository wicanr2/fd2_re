//! Go 來源：`remake/internal/battle/native_command20_ai_plan.go`（252 行）。

use super::*;

/// NativeAICommand2022Plan is the private state transaction for the recovered
/// 0x22A85..0x22E41 handler families. Enemy and player constructors keep
/// their distinct target owners while sharing the native 16-bit RNG adapters;
/// presentation remains a caller concern. The historical AI name is retained
/// to avoid obscuring the first evidence-backed consumer.
///
/// Go：`native_command20_ai_plan.go:10`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAICommand2022Plan {
    pub actor: Option<UnitIdx>,
    pub command_id: i64,
    pub targets: Vec<UnitIdx>,
    pub results: Vec<NativeAICommand2022Result>,
    pub rng_state: u16,
    pub(crate) mp_before: i64,
    pub(crate) mp_after: i64,
    pub before: Vec<NativeAICommand2022TargetState>,
    pub after: Vec<NativeAICommand2022TargetState>,
    pub(crate) published: bool,
    pub(crate) completed: bool,
}

/// Go：`native_command20_ai_plan.go:24`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAICommand2022Result {
    pub target: Option<UnitIdx>,
    pub offset: i64,
    pub command25: bool,
    pub cleared_byte5: bool,
    pub restore: Option<NativeRawFlagRestoreResult>,
    pub apply: Option<NativeRawApplicationResult>,
}

/// Go：`native_command20_ai_plan.go:33`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeAICommand2022TargetState {
    pub target: Option<UnitIdx>,
    pub hp: i64,
    pub native_record_byte5: u8,
    pub native_transient: [u8; 6],
}

impl State {
    /// NativeAICommand2022Targets rebuilds the 0x15311 target array at the selected
    /// winner cell (AIPlan.NativeActionDestination); the actor stays put on this
    /// route.
    ///
    /// Go：`native_command20_ai_plan.go:43`，`func (s *State) NativeAICommand2022Targets(actor *Unit, origin Cell, commandID int) ([]*Unit, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_ai_command2022_targets(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
    ) -> Result<Vec<UnitIdx>> {
        todo!("battle/native_command20_ai_plan.go:State.NativeAICommand2022Targets")
    }
}

/// Go：`native_command20_ai_plan.go:89`，`func nativeAICommand2022States(targets []*Unit) []NativeAICommand2022TargetState`
pub(crate) fn native_ai_command2022_states(targets: &[Unit]) -> Vec<NativeAICommand2022TargetState> {
    todo!("battle/native_command20_ai_plan.go:nativeAICommand2022States")
}

/// Go：`native_command20_ai_plan.go:97`，`func nativeAICommand2022Equal(states []NativeAICommand2022TargetState) bool`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_ai_command2022_equal(units: &[Unit], states: &[NativeAICommand2022TargetState]) -> bool {
    todo!("battle/native_command20_ai_plan.go:nativeAICommand2022Equal")
}

/// Go：`native_command20_ai_plan.go:106`，`func nativeAICommand2022Publish(states []NativeAICommand2022TargetState)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_ai_command2022_publish(units: &mut [Unit], states: &[NativeAICommand2022TargetState]) {
    todo!("battle/native_command20_ai_plan.go:nativeAICommand2022Publish")
}

impl State {
    /// Go：`native_command20_ai_plan.go:112`，`func (s *State) PlanNativeAICommand2022(actor *Unit, origin Cell, commandID int, rngState uint16) (*NativeAICommand2022Plan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_ai_command2022(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeAICommand2022Plan> {
        todo!("battle/native_command20_ai_plan.go:State.PlanNativeAICommand2022")
    }

    /// PlanNativePlayerCommand2022 builds the same raw handler calculation for a
    /// normal player-confirmed cursor. Target selection stays
    /// in the existing player preflight functions; this method only shares the
    /// non-mutating raw-record calculation with the enemy route.
    ///
    /// Go：`native_command20_ai_plan.go:124`，`func (s *State) PlanNativePlayerCommand2022(actor, confirmed *Unit, commandID int, rngState uint16) (*NativeAICommand2022Plan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_player_command2022(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeAICommand2022Plan> {
        todo!("battle/native_command20_ai_plan.go:State.PlanNativePlayerCommand2022")
    }

    /// Go：`native_command20_ai_plan.go:143`，`func (s *State) planNativeCommand2022Targets(actor *Unit, commandID int, targets []*Unit, rngState uint16) (*NativeAICommand2022Plan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    pub(crate) fn plan_native_command2022_targets(
        &self,
        actor: UnitIdx,
        command_id: i64,
        targets: &[UnitIdx],
        rng_state: u16,
    ) -> Result<NativeAICommand2022Plan> {
        todo!("battle/native_command20_ai_plan.go:State.planNativeCommand2022Targets")
    }
}

/// Go：`native_command20_ai_plan.go:207`，`func nativeAICommand2022StatesFromRecords(targets []*Unit, records []byte) []NativeAICommand2022TargetState`
pub(crate) fn native_ai_command2022_states_from_records(
    targets: &[Unit],
    records: &[u8],
) -> Vec<NativeAICommand2022TargetState> {
    todo!("battle/native_command20_ai_plan.go:nativeAICommand2022StatesFromRecords")
}

/// Go：`native_command20_ai_plan.go:217`，`func PublishNativeAICommand2022(plan *NativeAICommand2022Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_ai_command2022(units: &mut [Unit], plan: Option<&mut NativeAICommand2022Plan>) -> Result<()> {
    todo!("battle/native_command20_ai_plan.go:PublishNativeAICommand2022")
}

/// Go：`native_command20_ai_plan.go:227`，`func CompleteNativeAICommand2022(plan *NativeAICommand2022Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_ai_command2022(units: &mut [Unit], plan: Option<&mut NativeAICommand2022Plan>) -> Result<()> {
    todo!("battle/native_command20_ai_plan.go:CompleteNativeAICommand2022")
}

/// Go：`native_command20_ai_plan.go:236`，`func AbortNativeAICommand2022(plan *NativeAICommand2022Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_ai_command2022(units: &mut [Unit], plan: Option<&mut NativeAICommand2022Plan>) -> Result<()> {
    todo!("battle/native_command20_ai_plan.go:AbortNativeAICommand2022")
}
