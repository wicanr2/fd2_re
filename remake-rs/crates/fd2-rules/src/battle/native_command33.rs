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
        let mut plan = self.plan_native_compound_command33(actor, confirmed, rng_state)?;
        publish_native_compound_command33(&mut self.units, Some(&mut plan))?;
        if let Err(err) = complete_native_compound_command33(&mut self.units, Some(&mut plan)) {
            let _ = abort_native_compound_command33(&mut self.units, Some(&mut plan));
            return Err(err);
        }
        Ok(plan.result)
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
        const COMMAND_ID: i64 = 33;
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
            return Err(battle_err!("native compound command 33 player provenance unavailable"));
        }
        let actor_unit = &self.units[actor.unwrap_or_default()];
        let record = self.native_command_book[COMMAND_ID as usize];
        if record.mp_cost < 0 || actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native compound command 33 MP gate failed"));
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
        let mut seen = BTreeSet::new();
        for (index, &target) in targets.iter().enumerate() {
            if !seen.insert(target) {
                return Err(battle_err!("native compound command 33 target {index} duplicated"));
            }
        }
        let size = NATIVE_RECORD_SIZE as usize;
        for index in 0..targets.len() {
            let base = index * size;
            records[base + 0x25..base + 0x28].fill(0);
        }
        let restore = apply_native_raw_hp_restore_list(&mut records, &indices, 0x320, rng_state)?;
        let planned: Vec<NativeCompoundCommand33TargetInner> = targets
            .iter()
            .enumerate()
            .map(|(index, &target)| {
                let base = index * size;
                let mut after = [0u8; 6];
                after.copy_from_slice(&records[base + 0x22..base + 0x28]);
                NativeCompoundCommand33TargetInner {
                    target: Some(target),
                    hp_before: self.units[target].hp,
                    hp_after: u16::from_le_bytes([records[base + 0x40], records[base + 0x41]]) as i64,
                    transient_before: self.units[target].native_transient,
                    after,
                }
            })
            .collect();
        let result_targets = planned
            .iter()
            .map(|target| NativeCompoundCommand33Target {
                target: target.target,
                hp_before: target.hp_before,
                hp_after: target.hp_after,
                transient_before: target.transient_before,
                transient_after: target.after,
            })
            .collect();
        Ok(NativeCompoundCommand33Plan {
            actor,
            result: NativeCompoundCommand33Result { restore, targets: result_targets },
            actor_acted_before: actor_unit.acted,
            rng_before: rng_state,
            targets: planned,
            published: false,
            completed: false,
        })
    }
}

/// Go：`native_command33.go:125`，`func PublishNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_compound_command33(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand33Plan>,
) -> Result<()> {
    let Some(plan) = plan.filter(|p| {
        p.actor.is_some_and(|a| units[a].acted == p.actor_acted_before)
            && !p.completed
            && !p.published
            && !p.targets.is_empty()
            && p.result.restore.results.len() == p.targets.len()
    }) else {
        return Err(battle_err!("native compound command 33 publication unavailable"));
    };
    for (index, target) in plan.targets.iter().enumerate() {
        let unchanged = target
            .target
            .is_some_and(|t| units[t].hp == target.hp_before && units[t].native_transient == target.transient_before);
        if !unchanged {
            return Err(battle_err!("native compound command 33 target {index} state changed"));
        }
    }
    for target in &plan.targets {
        let Some(t) = target.target else { continue };
        units[t].hp = target.hp_after;
        units[t].native_transient = target.after;
    }
    plan.published = true;
    Ok(())
}

/// Go：`native_command33.go:145`，`func CompleteNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command33(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand33Plan>,
) -> Result<()> {
    let Some(plan) =
        plan.filter(|p| p.actor.is_some_and(|a| units[a].acted == p.actor_acted_before) && !p.completed && p.published)
    else {
        return Err(battle_err!("native compound command 33 completion unavailable"));
    };
    for (index, target) in plan.targets.iter().enumerate() {
        let complete =
            target.target.is_some_and(|t| units[t].hp == target.hp_after && units[t].native_transient == target.after);
        if !complete {
            return Err(battle_err!("native compound command 33 target {index} incomplete"));
        }
    }
    if let Some(actor) = plan.actor {
        units[actor].acted = true;
    }
    plan.completed = true;
    Ok(())
}

/// Go：`native_command33.go:160`，`func AbortNativeCompoundCommand33(plan *NativeCompoundCommand33Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_compound_command33(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand33Plan>,
) -> Result<()> {
    let Some(plan) = plan.filter(|p| p.actor.is_some_and(|a| units[a].acted == p.actor_acted_before) && !p.completed)
    else {
        return Err(battle_err!("native compound command 33 rollback unavailable"));
    };
    if !plan.published {
        return Ok(());
    }
    for (index, target) in plan.targets.iter().enumerate() {
        let complete =
            target.target.is_some_and(|t| units[t].hp == target.hp_after && units[t].native_transient == target.after);
        if !complete {
            return Err(battle_err!("native compound command 33 target {index} rollback state changed"));
        }
    }
    for target in plan.targets.iter().rev() {
        let Some(t) = target.target else { continue };
        units[t].hp = target.hp_before;
        units[t].native_transient = target.transient_before;
    }
    plan.published = false;
    Ok(())
}

/// Go：`native_command33.go:181`，`func nativeCompoundCommandTargetRecords(targets []*Unit) ([]byte, []byte, error)`
///
/// 所有權：Rust 版多一個開頭參數 `units`，`targets` 是其中的索引（Go 傳 `[]*Unit`）。
pub(crate) fn native_compound_command_target_records(
    units: &[Unit],
    targets: &[UnitIdx],
) -> Result<(Vec<u8>, Vec<u8>)> {
    if targets.is_empty() || targets.len() > 256 {
        return Err(battle_err!("native compound command target count={}", targets.len()));
    }
    let size = NATIVE_RECORD_SIZE as usize;
    let mut records = vec![0u8; targets.len() * size];
    let mut indices = vec![0u8; targets.len()];
    for (index, &target) in targets.iter().enumerate() {
        let target = &units[target];
        let byte_range = 0..=u8::MAX as i64;
        let word_range = 0..=u16::MAX as i64;
        if !target.has_native_record_class
            || !target.has_battle_fig
            || !byte_range.contains(&target.lv)
            || !byte_range.contains(&target.battle_fig)
            || !word_range.contains(&target.hp)
            || !word_range.contains(&target.max_hp)
        {
            return Err(battle_err!("native compound command target {index} lacks raw provenance"));
        }
        let base = index * size;
        records[base + 5] = target.native_record_byte5;
        records[base + 7] = target.battle_fig as u8;
        records[base + 0x20] = target.native_record_class;
        records[base + 0x21] = target.lv as u8;
        records[base + 0x22..base + 0x28].copy_from_slice(&target.native_transient);
        records[base + 0x40..base + 0x42].copy_from_slice(&(target.hp as u16).to_le_bytes());
        records[base + 0x42..base + 0x44].copy_from_slice(&(target.max_hp as u16).to_le_bytes());
        indices[index] = index as u8;
    }
    Ok((records, indices))
}

#[cfg(test)]
mod native_command33_test;
