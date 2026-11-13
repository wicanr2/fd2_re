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
///
/// 所有權：Rust 版多一個開頭參數 `units`，`targets` 是其中的索引（Go 傳 `[]*Unit`）。
pub(crate) fn native_command35_states(units: &[Unit], targets: &[UnitIdx]) -> Vec<NativeCommand35TargetState> {
    targets
        .iter()
        .map(|&target| NativeCommand35TargetState {
            target: Some(target),
            hp: units[target].hp,
            native_transient: units[target].native_transient,
        })
        .collect()
}

/// Go：`native_command35.go:44`，`func nativeCommand35Publish(states []NativeCommand35TargetState)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command35_publish(units: &mut [Unit], states: &[NativeCommand35TargetState]) {
    for state in states {
        // Go 對 nil Target 會 panic；狀態都由本檔產生，一定有目標。
        let Some(target) = state.target else { continue };
        (units[target].hp, units[target].native_transient) = (state.hp, state.native_transient);
    }
}

/// Go：`native_command35.go:50`，`func nativeCommand35EqualCurrent(states []NativeCommand35TargetState) bool`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_command35_equal_current(units: &[Unit], states: &[NativeCommand35TargetState]) -> bool {
    states.iter().all(|state| {
        state.target.is_some_and(|t| units[t].hp == state.hp && units[t].native_transient == state.native_transient)
    })
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
        const COMMAND_ID: i64 = 35;
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
            return Err(battle_err!("native compound command 35 player provenance unavailable"));
        }
        let actor_unit = &self.units[actor.unwrap_or_default()];
        let record = self.native_command_book[COMMAND_ID as usize];
        if record.mp_cost < 0 || actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native compound command 35 MP gate failed"));
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
        let (mut records, indices) = native_compound_command_target_records(&self.units, &targets)?;
        let (steps, ok) = native_compound_command_plan(COMMAND_ID);
        if !ok || steps.len() != 3 {
            return Err(battle_err!("native compound command 35 plan unavailable"));
        }
        let mut plan = NativeCompoundCommand35Plan {
            actor,
            before: native_command35_states(&self.units, &targets),
            ..Default::default()
        };
        let size = NATIVE_RECORD_SIZE as usize;
        let mut state = rng_state;
        for step in steps {
            if step.callee != 0x22d1b || !(0x25..=0x27).contains(&step.marker_offset) {
                return Err(battle_err!("native compound command 35 plan diverged"));
            }
            let (applications, next_state, accumulator) =
                apply_native_raw_application(&mut records, &indices, step.marker_offset, state)?;
            let stage = NativeCompoundCommand35Stage {
                command_id: step.command_id,
                marker_offset: step.marker_offset,
                results: applications,
                accumulator,
                rng_state: next_state,
            };
            let stage_states = targets
                .iter()
                .enumerate()
                .map(|(index, &target)| {
                    let base = index * size;
                    let mut native_transient = [0u8; 6];
                    native_transient.copy_from_slice(&records[base + 0x22..base + 0x28]);
                    NativeCommand35TargetState {
                        target: Some(target),
                        hp: u16::from_le_bytes([records[base + 0x40], records[base + 0x41]]) as i64,
                        native_transient,
                    }
                })
                .collect();
            plan.result.stages.push(stage);
            plan.result.stage_states.push(stage_states);
            state = next_state;
        }
        plan.result.rng_state = state;
        Ok(plan)
    }
}

/// Go：`native_command35.go:109`，`func PublishNativeCompoundCommand35Stage(plan *NativeCompoundCommand35Plan, stage int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_compound_command35_stage(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand35Plan>,
    stage: i64,
) -> Result<()> {
    let Some(plan) = plan
        .filter(|p| !p.completed && stage == p.published && stage >= 0 && stage < p.result.stage_states.len() as i64)
    else {
        return Err(battle_err!("native command35 invalid publish stage {stage}"));
    };
    let expected = if stage > 0 { &plan.result.stage_states[(stage - 1) as usize] } else { &plan.before };
    if !native_command35_equal_current(units, expected) {
        return Err(battle_err!("native command35 target changed before stage {stage}"));
    }
    native_command35_publish(units, &plan.result.stage_states[stage as usize]);
    plan.published += 1;
    Ok(())
}

/// Go：`native_command35.go:125`，`func CompleteNativeCompoundCommand35(plan *NativeCompoundCommand35Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command35(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand35Plan>,
) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native command35 completion boundary unavailable"));
    };
    let Some(actor) = plan.actor.filter(|&a| !plan.completed && plan.published == 3 && !units[a].acted) else {
        return Err(battle_err!("native command35 completion boundary unavailable"));
    };
    units[actor].acted = true;
    plan.completed = true;
    Ok(())
}

/// Go：`native_command35.go:133`，`func AbortNativeCompoundCommand35(plan *NativeCompoundCommand35Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_compound_command35(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand35Plan>,
) -> Result<()> {
    let Some(plan) = plan else {
        return Ok(());
    };
    if plan.completed {
        return Err(battle_err!("native command35 completed plan cannot abort"));
    }
    native_command35_publish(units, &plan.before);
    if let Some(actor) = plan.actor {
        units[actor].acted = false;
    }
    plan.published = 0;
    Ok(())
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
        let mut plan = self.plan_native_compound_command35(actor, confirmed, rng_state)?;
        for stage in 0..3 {
            if let Err(err) = publish_native_compound_command35_stage(&mut self.units, Some(&mut plan), stage) {
                let _ = abort_native_compound_command35(&mut self.units, Some(&mut plan));
                return Err(err);
            }
        }
        if let Err(err) = complete_native_compound_command35(&mut self.units, Some(&mut plan)) {
            let _ = abort_native_compound_command35(&mut self.units, Some(&mut plan));
            return Err(err);
        }
        Ok(plan.result)
    }
}

#[cfg(test)]
mod native_command35_test;
