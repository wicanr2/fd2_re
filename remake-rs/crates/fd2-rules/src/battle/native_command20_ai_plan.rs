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
        let unsupported = !(20..=27).contains(&command_id) || matches!(command_id, 23..=25);
        let Some(actor_unit) = actor.map(|i| &self.units[i]).filter(|u| {
            u.has_native_record_byte6
                && !unsupported
                && self.native_command_book.len() as i64 == NATIVE_COMMAND_RECORD_COUNT
                && self.native_command_book[command_id as usize].id == command_id
        }) else {
            return Err(battle_err!("native AI command tail record unavailable id={command_id}"));
        };
        let selector = actor_unit.native_record_byte6 as i64;
        if selector != 0 && selector != 1 {
            return Err(battle_err!("native AI command tail selector={selector} is outside 0/1"));
        }
        self.native_ai_command_origin_in_grid(origin)?;
        let records = native_ai_scoring_records(&self.units)?;
        let record = self.native_command_book[command_id as usize];
        if actor_unit.acted || record.mp_cost < 0 {
            return Err(battle_err!("native AI command tail actor/cost gate failed"));
        }
        let target_code = native_ai_scored_command_target_code(record.target_code, selector)?;
        let flags = self.native_command_base_flags()?;
        let indices = native_ai_scored_command_target_indices(
            self.w,
            self.h,
            &records,
            self.units.len() as i64,
            origin,
            record.effect_mode,
            target_code,
            &flags,
        )?;
        if indices.is_empty() {
            return Err(battle_err!("native AI command tail target array is empty"));
        }
        let targets: Vec<UnitIdx> = indices.iter().map(|&index| index as usize).collect();
        if actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native AI command tail insufficient MP"));
        }
        Ok(targets)
    }
}

/// Go：`native_command20_ai_plan.go:89`，`func nativeAICommand2022States(targets []*Unit) []NativeAICommand2022TargetState`
///
/// 所有權：Rust 版多一個開頭參數 `units`，`targets` 是其中的索引（Go 傳 `[]*Unit`）。
pub(crate) fn native_ai_command2022_states(units: &[Unit], targets: &[UnitIdx]) -> Vec<NativeAICommand2022TargetState> {
    targets
        .iter()
        .map(|&target| {
            let unit = &units[target];
            NativeAICommand2022TargetState {
                target: Some(target),
                hp: unit.hp,
                native_record_byte5: unit.native_record_byte5,
                native_transient: unit.native_transient,
            }
        })
        .collect()
}

/// Go：`native_command20_ai_plan.go:97`，`func nativeAICommand2022Equal(states []NativeAICommand2022TargetState) bool`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_ai_command2022_equal(units: &[Unit], states: &[NativeAICommand2022TargetState]) -> bool {
    states.iter().all(|state| {
        state.target.is_some_and(|target| {
            let unit = &units[target];
            unit.hp == state.hp
                && unit.native_record_byte5 == state.native_record_byte5
                && unit.native_transient == state.native_transient
        })
    })
}

/// Go：`native_command20_ai_plan.go:106`，`func nativeAICommand2022Publish(states []NativeAICommand2022TargetState)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub(crate) fn native_ai_command2022_publish(units: &mut [Unit], states: &[NativeAICommand2022TargetState]) {
    for state in states {
        // Go 對 nil Target 會 panic；狀態都由本檔產生，一定有目標。
        let Some(target) = state.target else { continue };
        let unit = &mut units[target];
        (unit.hp, unit.native_record_byte5, unit.native_transient) =
            (state.hp, state.native_record_byte5, state.native_transient);
    }
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
        let targets = self.native_ai_command2022_targets(actor, origin, command_id)?;
        // 上一步已確認 actor 存在。
        self.plan_native_command2022_targets(actor.unwrap_or_default(), command_id, &targets, rng_state)
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
        let targets = match command_id {
            20 | 21 => self.native_command_clear_restore_targets(actor, confirmed, command_id)?.0,
            22 | 26 | 27 => self.native_command_application_targets(actor, confirmed, command_id)?.0,
            25 => self.native_command25_targets(actor, confirmed)?,
            _ => return Err(battle_err!("native player command tail unsupported id={command_id}")),
        };
        // 三個預檢都先拒絕 nil actor。
        self.plan_native_command2022_targets(actor.unwrap_or_default(), command_id, &targets, rng_state)
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
        let (records, indices) = native_compound_command_target_records(&self.units, targets)?;
        let marker = match command_id {
            20 => 0x25,
            21 => 0x26,
            22 => 0x27,
            26 => 0x25,
            27 => 0x26,
            25 => 5,
            _ => return Err(battle_err!("native command tail unsupported id={command_id}")),
        };
        let actor_unit = &self.units[actor];
        let mut plan = NativeAICommand2022Plan {
            actor: Some(actor),
            command_id,
            targets: targets.to_vec(),
            before: native_ai_command2022_states(&self.units, targets),
            mp_before: actor_unit.mp,
            mp_after: actor_unit.mp - self.native_command_book[command_id as usize].mp_cost,
            ..Default::default()
        };
        let mut shadow = records.clone();
        let size = NATIVE_RECORD_SIZE as usize;
        if command_id == 25 {
            for (index, &target_index) in indices.iter().enumerate() {
                let base = target_index as usize * size;
                let cleared = shadow[base + 5] & 0x80 != 0;
                if cleared {
                    shadow[base + 5] &= !0x80;
                }
                plan.results.push(NativeAICommand2022Result {
                    target: Some(targets[index]),
                    offset: marker,
                    command25: true,
                    cleared_byte5: cleared,
                    ..Default::default()
                });
            }
            plan.rng_state = rng_state;
        } else if command_id == 20 || command_id == 21 {
            if self.native_command_book[10].id != 10 || self.native_command_book[10].damage < 0 {
                return Err(battle_err!("native AI command restore amount unavailable"));
            }
            let (results, next, _) = apply_native_raw_flag_restore(&mut shadow, &indices, marker, rng_state)?;
            plan.rng_state = next;
            for result in results {
                plan.results.push(NativeAICommand2022Result {
                    target: Some(targets[result.target_index as usize]),
                    offset: marker,
                    restore: Some(result),
                    ..Default::default()
                });
            }
        } else {
            let (results, next, _) = apply_native_raw_application(&mut shadow, &indices, marker, rng_state)?;
            plan.rng_state = next;
            for result in results {
                plan.results.push(NativeAICommand2022Result {
                    target: Some(targets[result.target_index as usize]),
                    offset: marker,
                    apply: Some(result),
                    ..Default::default()
                });
            }
        }
        plan.after = native_ai_command2022_states_from_records(targets, &shadow);
        Ok(plan)
    }
}

/// Go：`native_command20_ai_plan.go:207`，`func nativeAICommand2022StatesFromRecords(targets []*Unit, records []byte) []NativeAICommand2022TargetState`
///
/// 所有權：Go 只用 `targets` 的指標填 `Target`；Rust 直接收 `state.units` 的索引。
pub(crate) fn native_ai_command2022_states_from_records(
    targets: &[UnitIdx],
    records: &[u8],
) -> Vec<NativeAICommand2022TargetState> {
    let size = NATIVE_RECORD_SIZE as usize;
    targets
        .iter()
        .enumerate()
        .map(|(i, &target)| {
            let base = i * size;
            let mut native_transient = [0u8; 6];
            native_transient.copy_from_slice(&records[base + 0x22..base + 0x28]);
            NativeAICommand2022TargetState {
                target: Some(target),
                hp: records[base + 0x40] as i64 | (records[base + 0x41] as i64) << 8,
                native_record_byte5: records[base + 5],
                native_transient,
            }
        })
        .collect()
}

/// Go：`native_command20_ai_plan.go:217`，`func PublishNativeAICommand2022(plan *NativeAICommand2022Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_ai_command2022(units: &mut [Unit], plan: Option<&mut NativeAICommand2022Plan>) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native command tail publish boundary unavailable"));
    };
    let Some(actor) = plan.actor.filter(|&a| {
        !plan.published
            && !plan.completed
            && !units[a].acted
            && units[a].mp == plan.mp_before
            && native_ai_command2022_equal(units, &plan.before)
    }) else {
        return Err(battle_err!("native command tail publish boundary unavailable"));
    };
    native_ai_command2022_publish(units, &plan.after);
    units[actor].mp = plan.mp_after;
    plan.published = true;
    Ok(())
}

/// Go：`native_command20_ai_plan.go:227`，`func CompleteNativeAICommand2022(plan *NativeAICommand2022Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_ai_command2022(units: &mut [Unit], plan: Option<&mut NativeAICommand2022Plan>) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native command tail completion boundary unavailable"));
    };
    let Some(actor) = plan.actor.filter(|&a| {
        plan.published
            && !plan.completed
            && !units[a].acted
            && units[a].mp == plan.mp_after
            && native_ai_command2022_equal(units, &plan.after)
    }) else {
        return Err(battle_err!("native command tail completion boundary unavailable"));
    };
    units[actor].acted = true;
    plan.completed = true;
    Ok(())
}

/// Go：`native_command20_ai_plan.go:236`，`func AbortNativeAICommand2022(plan *NativeAICommand2022Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_ai_command2022(units: &mut [Unit], plan: Option<&mut NativeAICommand2022Plan>) -> Result<()> {
    let Some(plan) = plan else {
        return Ok(());
    };
    if plan.completed {
        return Err(battle_err!("native command tail completed plan cannot abort"));
    }
    if plan.published {
        native_ai_command2022_publish(units, &plan.before);
        // Go 在 Actor 為 nil 時會 panic；Rust 略過。
        if let Some(actor) = plan.actor {
            units[actor].mp = plan.mp_before;
        }
    }
    if let Some(actor) = plan.actor {
        units[actor].acted = false;
    }
    plan.published = false;
    Ok(())
}

#[cfg(test)]
mod native_command20_ai_plan_test;
