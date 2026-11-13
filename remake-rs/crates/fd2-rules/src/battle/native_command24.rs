//! Go 來源：`remake/internal/battle/native_command24.go`（430 行）。

use super::*;

/// NativeCommand24Damage is the state result of player-dispatched native
/// command 24.  It is deliberately separate from NativeCommandDamage: this
/// route does not read the command's damage/hit bytes or class resistance.
///
/// Go：`native_command24.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand24Damage {
    pub target: Option<UnitIdx>,
    pub amount: i64,
    pub damage: i64,
    pub hp_before: i64,
    pub hp_after: i64,
    pub native_record_byte5_before: u8,
    pub native_record_byte5_after: u8,
    pub(crate) death_flag_before: u8,
    pub(crate) death_flag_valid_before: bool,
}

/// NativeCommandDerivedStrikePlan is a mutation-free result of the recovered
/// 0x276EC selector, damage RNG and HP clamp. Presentation owners may publish
/// MP and each target HP at their raw FIGANI markers without rerolling damage.
///
/// Go：`native_command24.go:26`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeCommandDerivedStrikePlan {
    pub actor: Option<UnitIdx>,
    pub command_id: i64,
    pub mp_before: i64,
    pub mp_after: i64,
    pub results: Vec<NativeCommand24Damage>,
    pub target_impact_count: i64,
    pub(crate) published_target_stages: Vec<i64>,
}

/// Go：`native_command24.go:36`，`func nativeCommandDerivedStrikeMultiplier(commandID int) (int, bool)`
pub(crate) fn native_command_derived_strike_multiplier(command_id: i64) -> (i64, bool) {
    todo!("battle/native_command24.go:nativeCommandDerivedStrikeMultiplier")
}

/// ResolveNativeCommandDerivedStrikeDamage mirrors the signed amount path in
/// 0x276EC -> 0x1C81F: trunc(actor derived AP * multiplier / 10) minus target
/// derived DP, then trunc(amount * 9 / 10) plus trunc(rand()%100 * amount / 1000).
/// Go integer division truncates toward zero, matching x86 IDIV here.
///
/// Go：`native_command24.go:55`，`func ResolveNativeCommandDerivedStrikeDamage(actorAP, targetDP, multiplier int, rng *rand.Rand) (amount, damage int, err error)`
/// Go 具名回傳值依序為：amount、damage、err。
pub fn resolve_native_command_derived_strike_damage(
    actor_ap: i64,
    target_dp: i64,
    multiplier: i64,
    rng: Option<&mut dyn GoRand>,
) -> Result<(i64, i64)> {
    todo!("battle/native_command24.go:ResolveNativeCommandDerivedStrikeDamage")
}

/// ResolveNativeCommand24Damage preserves the ID24-specific public helper.
///
/// Go：`native_command24.go:68`，`func ResolveNativeCommand24Damage(actorAP, targetDP int, rng *rand.Rand) (amount, damage int, err error)`
/// Go 具名回傳值依序為：amount、damage、err。
pub fn resolve_native_command24_damage(
    actor_ap: i64,
    target_dp: i64,
    rng: Option<&mut dyn GoRand>,
) -> Result<(i64, i64)> {
    resolve_native_command_derived_strike_damage(actor_ap, target_dp, 15, rng)
}

impl State {
    /// ExecuteNativeCommand24 is the recovered non-UI player state slice for
    /// 0x1CFF0 command 0x18 -> 0x2A6BD -> 0x276EC.  The generic two-stage target
    /// list and record24 MP debit are followed by the derived-AP/DP damage route.
    /// The original resource has separate actor and target markers, but command24
    /// uses denominator 1 and publishes the complete HP delta at its target marker.
    /// This state-only helper applies that final delta without claiming presentation.
    ///
    /// Go：`native_command24.go:78`，`func (s *State) ExecuteNativeCommand24(actor, confirmed *Unit, rng *rand.Rand) ([]NativeCommand24Damage, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command24(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<Vec<NativeCommand24Damage>> {
        self.execute_native_command_derived_strike(actor, confirmed, 24, rng)
    }

    /// ExecuteNativeCommandDerivedStrike is the state-only 0x276EC family with
    /// proven player dispatches 24, 28, 29 and 31. ID30 has its own special
    /// cursor-selector entry point below; IDs32..35 use 0x27FC9 and stay closed.
    ///
    /// Go：`native_command24.go:85`，`func (s *State) ExecuteNativeCommandDerivedStrike(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommand24Damage, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command_derived_strike(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<Vec<NativeCommand24Damage>> {
        todo!("battle/native_command24.go:State.ExecuteNativeCommandDerivedStrike")
    }

    /// PlanNativeCommandDerivedStrike validates every target and consumes the
    /// caller-owned RNG, but leaves MP, HP and completion state unchanged.
    ///
    /// Go：`native_command24.go:114`，`func (s *State) PlanNativeCommandDerivedStrike(actor, confirmed *Unit, commandID int, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn plan_native_command_derived_strike(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<NativeCommandDerivedStrikePlan> {
        todo!("battle/native_command24.go:State.PlanNativeCommandDerivedStrike")
    }

    /// NativeCommand29EffectTargets keeps the original self-centered cursor separate
    /// from the affected enemy units. The direct-unit entry above remains available.
    ///
    /// Go：`native_command24.go:135`，`func (s *State) NativeCommand29EffectTargets(actor *Unit, cursor Cell) ([]*Unit, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command29_effect_targets(&self, actor: Option<UnitIdx>, cursor: Cell) -> Result<Vec<UnitIdx>> {
        todo!("battle/native_command24.go:State.NativeCommand29EffectTargets")
    }

    /// Go：`native_command24.go:150`，`func (s *State) PlanNativeCommand29AtCursor(actor *Unit, cursor Cell, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_command29_at_cursor(
        &self,
        actor: Option<UnitIdx>,
        cursor: Cell,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<NativeCommandDerivedStrikePlan> {
        todo!("battle/native_command24.go:State.PlanNativeCommand29AtCursor")
    }

    /// Go：`native_command24.go:158`，`func (s *State) planNativeCommandDerivedStrikeTargets(actor *Unit, targets []*Unit, commandID int, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn plan_native_command_derived_strike_targets(
        &self,
        actor: Option<UnitIdx>,
        targets: &[UnitIdx],
        command_id: i64,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<NativeCommandDerivedStrikePlan> {
        todo!("battle/native_command24.go:State.planNativeCommandDerivedStrikeTargets")
    }
}

/// Go：`native_command24.go:215`，`func ApplyNativeCommandDerivedStrikeMP(plan *NativeCommandDerivedStrikePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command_derived_strike_mp(
    units: &mut [Unit],
    plan: Option<&NativeCommandDerivedStrikePlan>,
) -> Result<()> {
    todo!("battle/native_command24.go:ApplyNativeCommandDerivedStrikeMP")
}

/// Go：`native_command24.go:223`，`func ApplyNativeCommandDerivedStrikeTarget(plan *NativeCommandDerivedStrikePlan, index int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command_derived_strike_target(
    units: &mut [Unit],
    plan: Option<&NativeCommandDerivedStrikePlan>,
    index: i64,
) -> Result<()> {
    todo!("battle/native_command24.go:ApplyNativeCommandDerivedStrikeTarget")
}

/// NativeCommand28StageResult is a frozen panel/HP projection. Stage0 denotes
/// the original target; stages1..8 use the original capped-damage denominator.
///
/// Go：`native_command24.go:248`，`func NativeCommand28StageResult(plan *NativeCommandDerivedStrikePlan, index, stage int) (NativeCommand24Damage, error)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn native_command28_stage_result(
    units: &[Unit],
    plan: Option<&NativeCommandDerivedStrikePlan>,
    index: i64,
    stage: i64,
) -> Result<NativeCommand24Damage> {
    todo!("battle/native_command24.go:NativeCommand28StageResult")
}

/// Go：`native_command24.go:265`，`func ApplyNativeCommand28TargetStage(plan *NativeCommandDerivedStrikePlan, index, stage int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command28_target_stage(
    units: &[Unit],
    plan: Option<&mut NativeCommandDerivedStrikePlan>,
    index: i64,
    stage: i64,
) -> Result<()> {
    todo!("battle/native_command24.go:ApplyNativeCommand28TargetStage")
}

/// RestoreNativeCommand30Targets restores the death-presentation provenance as
/// well as HP/raw +5. It is confined to the ID30 owner transaction.
///
/// Go：`native_command24.go:291`，`func RestoreNativeCommand30Targets(plan *NativeCommandDerivedStrikePlan)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn restore_native_command30_targets(units: &mut [Unit], plan: Option<&NativeCommandDerivedStrikePlan>) {
    todo!("battle/native_command24.go:RestoreNativeCommand30Targets")
}

/// Go：`native_command24.go:306`，`func CompleteNativeCommandDerivedStrike(plan *NativeCommandDerivedStrikePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_command_derived_strike(
    units: &mut [Unit],
    plan: Option<&NativeCommandDerivedStrikePlan>,
) -> Result<()> {
    todo!("battle/native_command24.go:CompleteNativeCommandDerivedStrike")
}

impl State {
    /// NativeCommand30EffectTargets validates the bounded player selector for command30:
    /// 0x1CFF0 first validates a normal 0x14818 candidate, then 0x149F8 traces
    /// from the saved pre-confirm cursor toward the confirmed cursor for
    /// record+3-0x10 steps, before 0x2A6BD -> 0x276EC applies multiplier 18.
    /// Its indexed multi-hit presentation and UI cursor lifecycle remain outside
    /// this method; callers must supply both recovered cursor positions.
    ///
    /// Go：`native_command24.go:338`，`func (s *State) NativeCommand30EffectTargets(actor *Unit, savedCursor, confirmedCursor Cell) ([]*Unit, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command30_effect_targets(
        &self,
        actor: Option<UnitIdx>,
        saved_cursor: Cell,
        confirmed_cursor: Cell,
    ) -> Result<Vec<UnitIdx>> {
        todo!("battle/native_command24.go:State.NativeCommand30EffectTargets")
    }

    /// PlanNativeCommand30 shares the proven selector with the existing state-only
    /// control, while freezing each HP result for the indexed/HD marker publisher.
    ///
    /// Go：`native_command24.go:372`，`func (s *State) PlanNativeCommand30(actor *Unit, savedCursor, confirmedCursor Cell, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_command30(
        &self,
        actor: Option<UnitIdx>,
        saved_cursor: Cell,
        confirmed_cursor: Cell,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<NativeCommandDerivedStrikePlan> {
        todo!("battle/native_command24.go:State.PlanNativeCommand30")
    }

    /// Go：`native_command24.go:407`，`func (s *State) ExecuteNativeCommand30(actor *Unit, savedCursor, confirmedCursor Cell, rng *rand.Rand) ([]NativeCommand24Damage, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command30(
        &mut self,
        actor: Option<UnitIdx>,
        saved_cursor: Cell,
        confirmed_cursor: Cell,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<Vec<NativeCommand24Damage>> {
        todo!("battle/native_command24.go:State.ExecuteNativeCommand30")
    }
}
