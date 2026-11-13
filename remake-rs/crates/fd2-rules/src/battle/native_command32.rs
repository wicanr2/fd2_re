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
        let mut plan = self.plan_native_compound_command32(actor, confirmed, rng_state)?;
        for index in 0..plan.result.targets.len() {
            if let Err(err) = apply_native_compound_command32_target(&mut self.units, Some(&mut plan), index as i64) {
                let _ = abort_native_compound_command32(&mut self.units, Some(&mut plan));
                return Err(err);
            }
        }
        if let Err(err) = complete_native_compound_command32(&mut self.units, Some(&mut plan)) {
            let _ = abort_native_compound_command32(&mut self.units, Some(&mut plan));
            return Err(err);
        }
        Ok(plan.result)
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
        const COMMAND_ID: i64 = 32;
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
            return Err(battle_err!("native compound command 32 player provenance unavailable"));
        }
        let actor_unit = &self.units[actor.unwrap_or_default()];
        let record = self.native_command_book[COMMAND_ID as usize];
        if record.mp_cost < 0 || actor_unit.mp < record.mp_cost || record.damage < 0 || !(0..=100).contains(&record.hit)
        {
            return Err(battle_err!("native compound command 32 record gate failed"));
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
        native_compound_command_target_records(&self.units, &targets)?;
        let mut seen_targets = BTreeSet::new();
        for (index, &target) in targets.iter().enumerate() {
            if !seen_targets.insert(target) {
                return Err(battle_err!("native compound command 32 target {index} duplicated"));
            }
            match self.native_command_resistances.get(&native_command_resistance_class(&self.units[target])) {
                Some(resistance) if (0..=10).contains(resistance) => {}
                _ => return Err(battle_err!("native compound command 32 target {index} resistance unavailable")),
            }
        }

        let mut result = NativeCompoundCommand32Result { targets: Vec::with_capacity(targets.len()), rng_state: 0 };
        let mut state = rng_state;
        for &target in &targets {
            let unit = &self.units[target];
            let resistance =
                self.native_command_resistances.get(&native_command_resistance_class(unit)).copied().unwrap_or(0);
            let (damage, next_state) = resolve_native_command_damage(
                record.damage,
                self.command_hit(actor, COMMAND_ID, record.hit),
                resistance,
                state,
            )?;
            let mut hp_after = unit.hp;
            if damage.hit {
                hp_after -= damage.damage;
                if hp_after < 0 {
                    hp_after = 0;
                }
            }
            result.targets.push(NativeCompoundCommand32Target {
                target: Some(target),
                damage,
                hp_before: unit.hp,
                hp_after,
            });
            state = next_state;
        }
        result.rng_state = state;
        let planned_targets = result.targets.clone();
        let published = vec![false; result.targets.len()];
        Ok(NativeCompoundCommand32Plan {
            actor,
            result,
            actor_acted_before: actor_unit.acted,
            rng_before: rng_state,
            targets: planned_targets,
            published,
            completed: false,
        })
    }
}

/// Go：`native_command32.go:123`，`func ApplyNativeCompoundCommand32Target(plan *NativeCompoundCommand32Plan, index int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_compound_command32_target(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand32Plan>,
    index: i64,
) -> Result<()> {
    let Some(plan) = plan.filter(|p| {
        p.actor.is_some_and(|a| units[a].acted == p.actor_acted_before)
            && !p.completed
            && index >= 0
            && index < p.targets.len() as i64
            && p.published.len() == p.targets.len()
            && !p.published[index as usize]
    }) else {
        return Err(battle_err!("native compound command 32 target publication unavailable"));
    };
    let target = plan.targets[index as usize];
    let Some(t) = target.target.filter(|&t| units[t].hp == target.hp_before) else {
        return Err(battle_err!("native compound command 32 target state changed"));
    };
    units[t].hp = target.hp_after;
    plan.published[index as usize] = true;
    Ok(())
}

/// Go：`native_command32.go:137`，`func CompleteNativeCompoundCommand32(plan *NativeCompoundCommand32Plan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_compound_command32(
    units: &mut [Unit],
    plan: Option<&mut NativeCompoundCommand32Plan>,
) -> Result<()> {
    let Some(plan) = plan.filter(|p| {
        p.actor.is_some_and(|a| units[a].acted == p.actor_acted_before)
            && !p.completed
            && !p.targets.is_empty()
            && p.published.len() == p.targets.len()
    }) else {
        return Err(battle_err!("native compound command 32 completion unavailable"));
    };
    for (index, &published) in plan.published.iter().enumerate() {
        let target = plan.targets[index];
        if !published || !target.target.is_some_and(|t| units[t].hp == target.hp_after) {
            return Err(battle_err!("native compound command 32 target {index} incomplete"));
        }
    }
    if let Some(actor) = plan.actor {
        units[actor].acted = true;
    }
    plan.completed = true;
    Ok(())
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
    let Some(plan) = plan.filter(|p| {
        p.actor.is_some_and(|a| units[a].acted == p.actor_acted_before)
            && !p.completed
            && p.published.len() == p.targets.len()
    }) else {
        return Err(battle_err!("native compound command 32 rollback unavailable"));
    };
    for (index, &published) in plan.published.iter().enumerate() {
        let target = plan.targets[index];
        if published && !target.target.is_some_and(|t| units[t].hp == target.hp_after) {
            return Err(battle_err!("native compound command 32 target {index} rollback state changed"));
        }
    }
    for index in (0..plan.published.len()).rev() {
        if plan.published[index] {
            if let Some(t) = plan.targets[index].target {
                units[t].hp = plan.targets[index].hp_before;
            }
            plan.published[index] = false;
        }
    }
    Ok(())
}

#[cfg(test)]
mod native_command32_test;
