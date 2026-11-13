//! Go 來源：`remake/internal/battle/native_command0.go`（409 行）。

use super::*;

/// NativeCommandDamageResult is one final-effect candidate resolved by the
/// verified numeric damage path. Animation and post-resolution messaging are
/// intentionally outside this state mutation.
///
/// Go：`native_command0.go:10`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandDamageResult {
    pub target: Option<UnitIdx>,
    /// Go 內嵌欄位 `NativeCommandDamage`；Go 的欄位提升（`x.Amount`）在 Rust 寫成 `x.native_command_damage.amount`。
    pub native_command_damage: NativeCommandDamage,
    pub hp_before: i64,
    pub hp_after: i64,
}

/// NativeCommandDamagePlan is the mutation-free half of 0x2A6BD's numeric
/// transaction.  Presentation owners use it to publish MP before the target
/// loop and HP at the command-specific raw impact markers without rerolling RNG.
///
/// Go：`native_command0.go:20`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandDamagePlan {
    pub actor: Option<UnitIdx>,
    pub command_id: i64,
    pub mp_before: i64,
    pub mp_after: i64,
    pub actor_acted_before: bool,
    pub rng_before: u16,
    pub rng_after: u16,
    pub results: Vec<NativeCommandDamageResult>,
    pub damage_stages: i64,
    pub(crate) published_target_stages: Vec<i64>,
    pub(crate) mp_published: bool,
    pub(crate) completed: bool,
}

impl State {
    /// ExecuteBoundNativeCommand0 uses the state-bound verified resistance table.
    /// A missing table remains fail-closed rather than falling back to the legacy
    /// normalized magic approximation.
    ///
    /// Go：`native_command0.go:35`，`func (s *State) ExecuteBoundNativeCommand0(actor, confirmed *Unit, rngState uint16) ([]NativeCommandDamageResult, uint16, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn execute_bound_native_command0(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<(Vec<NativeCommandDamageResult>, u16)> {
        todo!("battle/native_command0.go:State.ExecuteBoundNativeCommand0")
    }

    /// PlanBoundNativeCommand0 validates and resolves command0 without mutation.
    ///
    /// Go：`native_command0.go:43`，`func (s *State) PlanBoundNativeCommand0(actor, confirmed *Unit, rngState uint16) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_bound_native_command0(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng_state: u16,
    ) -> Result<NativeCommandDamagePlan> {
        todo!("battle/native_command0.go:State.PlanBoundNativeCommand0")
    }

    /// ExecuteNativeCommandDamage covers the byte-for-byte numeric route proven
    /// for player-dispatched command IDs 0..12. IDs0..8 dispatch directly to
    /// 0x2A6BD, which runs sub_2B659's MP event and its final-target loop directly
    /// calls 0x1C75E(targetSlot, commandID). ID9 invokes 0x1CA89 -> 0x1C75E;
    /// IDs10..12 run their distinct indexed compositor (0x21548) before the same
    /// state sequence. Other IDs stay fail-closed.
    ///
    /// Go：`native_command0.go:56`，`func (s *State) ExecuteNativeCommandDamage(actor, confirmed *Unit, commandID int, resistByClass map[int]int, rngState uint16) ([]NativeCommandDamageResult, uint16, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command_damage(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
    ) -> Result<(Vec<NativeCommandDamageResult>, u16)> {
        todo!("battle/native_command0.go:State.ExecuteNativeCommandDamage")
    }

    /// PlanNativeCommandDamage consumes the verified RNG sequence but leaves all
    /// Unit fields unchanged.  The result captures exact pre/post HP for atomic
    /// command-indexed publication denominator for atomic presentation admission.
    ///
    /// Go：`native_command0.go:78`，`func (s *State) PlanNativeCommandDamage(actor, confirmed *Unit, commandID int, resistByClass map[int]int, rngState uint16) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_command_damage(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
    ) -> Result<NativeCommandDamagePlan> {
        self.plan_native_command_damage_walk(actor, confirmed, command_id, resist_by_class, rng_state, None)
    }

    /// PlanNativeCommandDamageWalk 是 PlanNativeCommandDamage 加上演出亂數交錯的版本。
    ///
    /// Go：`native_command0.go:83`，`func (s *State) PlanNativeCommandDamageWalk(actor, confirmed *Unit, commandID int, resistByClass map[int]int, rngState uint16, walk NativeCommandDamageRNGWalk) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_command_damage_walk(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
        walk: Option<&mut NativeCommandDamageRNGWalk<'_>>,
    ) -> Result<NativeCommandDamagePlan> {
        todo!("battle/native_command0.go:State.PlanNativeCommandDamageWalk")
    }

    /// PlanNativeCommand6DamageAtCursor 驗證玩家游標中心並沿用完整損傷預檢交易。
    /// 指令6的其他演出與發布時點仍由原有 owner 處理。
    ///
    /// Go：`native_command0.go:104`，`func (s *State) PlanNativeCommand6DamageAtCursor(actor *Unit, cursor Cell, resistByClass map[int]int, rngState uint16, walk NativeCommandDamageRNGWalk) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_command6_damage_at_cursor(
        &self,
        actor: Option<UnitIdx>,
        cursor: Cell,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
        walk: Option<&mut NativeCommandDamageRNGWalk<'_>>,
    ) -> Result<NativeCommandDamagePlan> {
        self.plan_native_command_damage_at_cursor(actor, cursor, 6, resist_by_class, rng_state, walk)
    }

    /// Player commands 6 and10..12 use the common confirmed-cursor caller. In
    /// particular, selection mode0 must not require an enemy on the actor's cell.
    ///
    /// Go：`native_command0.go:110`，`func (s *State) PlanNativeCommandDamageAtCursor(actor *Unit, cursor Cell, commandID int, resistByClass map[int]int, rngState uint16, walk NativeCommandDamageRNGWalk) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_command_damage_at_cursor(
        &self,
        actor: Option<UnitIdx>,
        cursor: Cell,
        command_id: i64,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
        walk: Option<&mut NativeCommandDamageRNGWalk<'_>>,
    ) -> Result<NativeCommandDamagePlan> {
        todo!("battle/native_command0.go:State.PlanNativeCommandDamageAtCursor")
    }

    /// Go：`native_command0.go:131`，`func (s *State) planNativeCommandDamageTargets(actor *Unit, commandID int, record NativeCommandRecord, targets []*Unit, resistByClass map[int]int, rngState uint16, walk NativeCommandDamageRNGWalk) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    pub(crate) fn plan_native_command_damage_targets(
        &self,
        actor: Option<UnitIdx>,
        command_id: i64,
        record: NativeCommandRecord,
        targets: &[UnitIdx],
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
        walk: Option<&mut NativeCommandDamageRNGWalk<'_>>,
    ) -> Result<NativeCommandDamagePlan> {
        todo!("battle/native_command0.go:State.planNativeCommandDamageTargets")
    }
}

/// NativeCommandDamageRNGWalk 讓演出 owner 決定 0x2A6BD 目標迴圈裡每個目標的
/// 0x1C75E 擲骰落在整條 0x4E893 序列的哪一點：handler 的 mode 0 初始化、mode 5
/// 重置與 0x2AF40 的畫面抖動都吃同一個全域亂數。walk 必須對 0..targetCount-1
/// 各呼叫 resolve 恰好一次（依目標順序），並回傳整段演出結束時的亂數狀態。
/// nil 表示沒有演出亂數，逐目標依序擲骰。
///
/// Go：`native_command0.go:164`
pub type NativeCommandDamageRNGWalk<'a> =
    dyn FnMut(i64, &mut dyn FnMut(i64, u16) -> Result<(u16, bool)>) -> Result<u16> + 'a;

/// resolveNativeCommandDamagePlan 依 walk 指定的順序對每個目標跑 0x1C75E→0x1C81F
/// 的命中／傷害擲骰，並把演出結束時的亂數寫進 plan.RNGAfter。
///
/// Go：`native_command0.go:168`，`func resolveNativeCommandDamagePlan(plan *NativeCommandDamagePlan, record NativeCommandRecord, targets []*Unit, resistByClass map[int]int, rngState uint16, walk NativeCommandDamageRNGWalk) error`
pub(crate) fn resolve_native_command_damage_plan(
    plan: &mut NativeCommandDamagePlan,
    record: NativeCommandRecord,
    targets: &[Unit],
    resist_by_class: &BTreeMap<i64, i64>,
    rng_state: u16,
    walk: Option<&mut NativeCommandDamageRNGWalk<'_>>,
) -> Result<()> {
    todo!("battle/native_command0.go:resolveNativeCommandDamagePlan")
}

impl State {
    /// PlanNativeAICommandDamageSingleTarget consumes the explicit target array
    /// already produced by 0x15311 for the 0x2A6BD route. It must not re-run the
    /// player-only 0x1CFF0 two-stage selector, whose native camp predicate is not
    /// relative to the enemy actor. The narrow entry currently accepts only a
    /// proven one-target record and preserves the same MP/resistance/RNG contract.
    ///
    /// Go：`native_command0.go:227`，`func (s *State) PlanNativeAICommandDamageSingleTarget(actor, target *Unit, commandID int, resistByClass map[int]int, rngState uint16) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor`、`target` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_ai_command_damage_single_target(
        &self,
        actor: Option<UnitIdx>,
        target: Option<UnitIdx>,
        command_id: i64,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
    ) -> Result<NativeCommandDamagePlan> {
        todo!("battle/native_command0.go:State.PlanNativeAICommandDamageSingleTarget")
    }

    /// PlanNativeAICommandDamage rebuilds the exact final target array selected by
    /// 0x1598A/0x15311 from the actor's raw +6 selector and the selected winner
    /// cell. 0x15311 keeps the actor at its current cell and feeds the winner
    /// coordinates [0x53C27]/[0x53C2B] to the 0x14818 target geometry, so origin
    /// must be AIPlan.NativeActionDestination, never the actor's presentation
    /// cell. It intentionally bypasses the player-only confirmed-cursor admission.
    ///
    /// Go：`native_command0.go:272`，`func (s *State) PlanNativeAICommandDamage(actor *Unit, origin Cell, commandID int, resistByClass map[int]int, rngState uint16) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_ai_command_damage(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
    ) -> Result<NativeCommandDamagePlan> {
        self.plan_native_ai_command_damage_walk(actor, origin, command_id, resist_by_class, rng_state, None)
    }

    /// PlanNativeAICommandDamageWalk 是 PlanNativeAICommandDamage 加上演出亂數交錯的版本。
    ///
    /// Go：`native_command0.go:277`，`func (s *State) PlanNativeAICommandDamageWalk(actor *Unit, origin Cell, commandID int, resistByClass map[int]int, rngState uint16, walk NativeCommandDamageRNGWalk) (*NativeCommandDamagePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_ai_command_damage_walk(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        resist_by_class: &BTreeMap<i64, i64>,
        rng_state: u16,
        walk: Option<&mut NativeCommandDamageRNGWalk<'_>>,
    ) -> Result<NativeCommandDamagePlan> {
        todo!("battle/native_command0.go:State.PlanNativeAICommandDamageWalk")
    }

    /// nativeAICommandOriginInGrid rejects a winner cell that the 0x15311 route
    /// could never have produced; the presentation owners pass
    /// AIPlan.NativeActionDestination here.
    ///
    /// Go：`native_command0.go:343`，`func (s *State) nativeAICommandOriginInGrid(origin Cell) error`
    pub(crate) fn native_ai_command_origin_in_grid(&self, origin: Cell) -> Result<()> {
        todo!("battle/native_command0.go:State.nativeAICommandOriginInGrid")
    }
}

/// nativeCommandDamageStages preserves the 0x525AF command0..9 table consumed
/// by 0x2AEC9. Commands10..12 do not use that indexed target presenter in the
/// current engine slice, so their state-only transaction publishes once.
///
/// Go：`native_command0.go:353`，`func nativeCommandDamageStages(commandID int) (int, error)`
pub(crate) fn native_command_damage_stages(command_id: i64) -> Result<i64> {
    todo!("battle/native_command0.go:nativeCommandDamageStages")
}

/// Go：`native_command0.go:364`，`func ApplyNativeCommandDamageMP(plan *NativeCommandDamagePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command_damage_mp(units: &mut [Unit], plan: Option<&mut NativeCommandDamagePlan>) -> Result<()> {
    todo!("battle/native_command0.go:ApplyNativeCommandDamageMP")
}

/// ApplyNativeCommandDamageStage publishes the exact intermediate HP formula
/// used by 0x2A6BD with the command-indexed denominator from 0x525AF. A caller
/// may jump directly to the final stage for state-only execution.
///
/// Go：`native_command0.go:377`，`func ApplyNativeCommandDamageStage(plan *NativeCommandDamagePlan, index, stage int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command_damage_stage(
    units: &mut [Unit],
    plan: Option<&mut NativeCommandDamagePlan>,
    index: i64,
    stage: i64,
) -> Result<()> {
    todo!("battle/native_command0.go:ApplyNativeCommandDamageStage")
}

/// Go：`native_command0.go:396`，`func CompleteNativeCommandDamage(plan *NativeCommandDamagePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_command_damage(units: &mut [Unit], plan: Option<&mut NativeCommandDamagePlan>) -> Result<()> {
    todo!("battle/native_command0.go:CompleteNativeCommandDamage")
}
