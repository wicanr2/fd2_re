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
///
/// 所有權：Rust 版多一個開頭參數 `units`，`targets` 是其中的索引（Go 傳 `[]*Unit`）；回傳狀態的
/// `target` 就是這些索引。
pub(crate) fn native_command34_states(units: &[Unit], targets: &[UnitIdx]) -> Vec<NativeCommand34TargetState> {
    targets
        .iter()
        .map(|&target| {
            let unit = &units[target];
            NativeCommand34TargetState {
                target: Some(target),
                native_transient: unit.native_transient,
                ap: unit.ap,
                dp: unit.dp,
                hit: unit.hit,
                ev: unit.ev,
            }
        })
        .collect()
}

/// Go：`native_command34.go:33`，`func nativeCommand34Publish(states []NativeCommand34TargetState)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command34_publish(units: &mut [Unit], states: &[NativeCommand34TargetState]) {
    for state in states {
        // Go 對 nil Target 會 panic；狀態都由 native_command34_states 產生，一定有目標。
        let Some(target) = state.target else { continue };
        let unit = &mut units[target];
        unit.native_transient = state.native_transient;
        (unit.ap, unit.dp, unit.hit, unit.ev) = (state.ap, state.dp, state.hit, state.ev);
    }
}

/// Go：`native_command34.go:40`，`func nativeCommand34EqualCurrent(states []NativeCommand34TargetState) bool`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command34_equal_current(units: &[Unit], states: &[NativeCommand34TargetState]) -> bool {
    states.iter().all(|state| {
        state.target.is_some_and(|target| {
            let unit = &units[target];
            unit.native_transient == state.native_transient
                && unit.ap == state.ap
                && unit.dp == state.dp
                && unit.hit == state.hit
                && unit.ev == state.ev
        })
    })
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
        const COMMAND_ID: i64 = 34;
        let actor_ok = actor.is_some_and(|i| {
            let a = &self.units[i];
            !a.acted
                && a.has_native_record_class
                && a.native_record_class == 19
                && a.has_battle_fig
                && native_compound_player_selector(a.battle_fig)
        });
        if !actor_ok
            || self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_command_book[COMMAND_ID as usize].id != COMMAND_ID
        {
            return Err(battle_err!("native compound command 34 player provenance unavailable"));
        }
        let actor_unit = &self.units[actor.unwrap_or_default()];
        let record = self.native_command_book[COMMAND_ID as usize];
        if record.mp_cost < 0 || actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native compound command 34 MP gate failed"));
        }
        let flags = self.native_command_base_flags()?;
        let targets = native_command_effect_targets(
            self.w,
            self.h,
            actor,
            confirmed,
            record.selection_mode,
            record.effect_mode,
            record.target_code,
            &flags,
            &self.units,
        )?;
        let (mut records, indices) = native_command_modifier_records(&self.units, &targets)?;
        let mut plan = NativeCompoundCommand34Plan {
            actor,
            before: native_command34_states(&self.units, &targets),
            ..Default::default()
        };
        let mut state = rng_state;
        for stage_id in [17, 18, 19] {
            let stage = apply_native_command_modifier(&mut records, &indices, stage_id, state)?;
            let stage_states = native_command_modifier_shadow_states(&self.units, &targets, &records);
            state = stage.rng_state;
            plan.result.stages.push(stage);
            plan.result.stage_states.push(stage_states);
        }
        plan.result.rng_state = state;
        Ok(plan)
    }
}

/// Go：`native_command34.go:97`，`func PublishNativeCompoundCommand34Stage(plan *NativeCompoundCommand34Plan, stage int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_compound_command34_stage(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand34Plan>,
    stage: i64,
) -> Result<()> {
    let Some(plan) = plan
        .filter(|p| !p.completed && stage == p.published && stage >= 0 && stage < p.result.stage_states.len() as i64)
    else {
        return Err(battle_err!("native command34 invalid publish stage {stage}"));
    };
    let expected = if stage > 0 { &plan.result.stage_states[(stage - 1) as usize] } else { &plan.before };
    if !native_command34_equal_current(units, expected) {
        return Err(battle_err!("native command34 target changed before stage {stage}"));
    }
    native_command34_publish(units, &plan.result.stage_states[stage as usize]);
    plan.published += 1;
    Ok(())
}

/// Go：`native_command34.go:113`，`func CompleteNativeCompoundCommand34(plan *NativeCompoundCommand34Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command34(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand34Plan>,
) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native command34 completion boundary unavailable"));
    };
    let Some(actor) = plan.actor.filter(|&a| !plan.completed && plan.published == 3 && !units[a].acted) else {
        return Err(battle_err!("native command34 completion boundary unavailable"));
    };
    units[actor].acted = true;
    plan.completed = true;
    Ok(())
}

/// Go：`native_command34.go:121`，`func AbortNativeCompoundCommand34(plan *NativeCompoundCommand34Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_compound_command34(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand34Plan>,
) -> Result<()> {
    let Some(plan) = plan else {
        return Ok(());
    };
    if plan.completed {
        return Err(battle_err!("native command34 completed plan cannot abort"));
    }
    native_command34_publish(units, &plan.before);
    if let Some(actor) = plan.actor {
        units[actor].acted = false;
    }
    plan.published = 0;
    Ok(())
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
        let mut plan = self.plan_native_compound_command34(actor, confirmed, rng_state)?;
        for stage in 0..3 {
            if let Err(err) = publish_native_compound_command34_stage(&mut self.units, Some(&mut plan), stage) {
                let _ = abort_native_compound_command34(&mut self.units, Some(&mut plan));
                return Err(err);
            }
        }
        if let Err(err) = complete_native_compound_command34(&mut self.units, Some(&mut plan)) {
            let _ = abort_native_compound_command34(&mut self.units, Some(&mut plan));
            return Err(err);
        }
        Ok(plan.result)
    }
}

/// Go：`native_command34.go:154`，`func nativeCompoundPlayerSelector(selector int) bool`
pub(crate) fn native_compound_player_selector(selector: i64) -> bool {
    matches!(selector, 4 | 5 | 6 | 7 | 20)
}

#[cfg(test)]
mod native_command34_test;
