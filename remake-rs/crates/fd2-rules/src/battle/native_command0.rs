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
        if self.native_command_resistances.is_empty() {
            return Err(battle_err!("native command 0 resistances unavailable"));
        }
        let resistances = self.native_command_resistances.clone();
        self.execute_native_command_damage(actor, confirmed, 0, &resistances, rng_state)
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
        if self.native_command_resistances.is_empty() {
            return Err(battle_err!("native command 0 resistances unavailable"));
        }
        self.plan_native_command_damage(actor, confirmed, 0, &self.native_command_resistances, rng_state)
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
        let mut plan = self.plan_native_command_damage(actor, confirmed, command_id, resist_by_class, rng_state)?;
        apply_native_command_damage_mp(&mut self.units, Some(&mut plan))?;
        let stages = plan.damage_stages;
        for index in 0..plan.results.len() {
            apply_native_command_damage_stage(&mut self.units, Some(&mut plan), index as i64, stages)?;
        }
        complete_native_command_damage(&mut self.units, Some(&mut plan))?;
        Ok((plan.results.clone(), plan.rng_after))
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
        if !(0..=12).contains(&command_id)
            || self.native_command_book.len() != 36
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native command damage record unavailable id={command_id}"));
        }
        let record = self.native_command_book[command_id as usize];
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
        self.plan_native_command_damage_targets(actor, command_id, record, &targets, resist_by_class, rng_state, walk)
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
        let actor_ok = actor.is_some_and(|i| self.units[i].camp != ENEMY);
        if !actor_ok
            || (command_id != 6 && !(10..=12).contains(&command_id))
            || self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native player command{command_id} cursor record/actor unavailable"));
        }
        let record = self.native_command_book[command_id as usize];
        let flags = self.native_command_base_flags()?;
        let targets = if record.selection_mode == 0 {
            native_command_self_centered_targets(
                self.w,
                self.h,
                actor,
                cursor,
                record.effect_mode,
                &flags,
                &self.units,
            )?
        } else {
            native_command_effect_targets_at_cursor(
                self.w,
                self.h,
                actor,
                cursor,
                record.selection_mode,
                record.effect_mode,
                record.target_code,
                &flags,
                &self.units,
            )?
        };
        self.plan_native_command_damage_targets(actor, command_id, record, &targets, resist_by_class, rng_state, walk)
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
        let mut record = record;
        // The original table is total for valid runtime class IDs.  Validate every
        // target before 0x1CA89-equivalent MP mutation to keep a missing editable
        // table entry fail-closed rather than making a partial command transaction.
        for &target in targets {
            let class = native_command_resistance_class(&self.units[target]);
            match resist_by_class.get(&class) {
                Some(raw) if (0..=10).contains(raw) => {}
                _ => return Err(battle_err!("native command damage missing resistance class={class}")),
            }
        }
        let Some(actor_idx) = actor else {
            return Err(battle_err!("native command damage insufficient MP"));
        };
        let actor_unit = &self.units[actor_idx];
        if actor_unit.acted || !(0..=0xff).contains(&record.mp_cost) || actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native command damage insufficient MP"));
        }
        let damage_stages = native_command_damage_stages(command_id)?;
        let mut plan = NativeCommandDamagePlan {
            actor,
            command_id,
            mp_before: actor_unit.mp,
            mp_after: actor_unit.mp - record.mp_cost,
            actor_acted_before: actor_unit.acted,
            rng_before: rng_state,
            damage_stages,
            ..Default::default()
        };
        record.hit = self.command_hit(actor, command_id, record.hit);
        resolve_native_command_damage_plan(&mut plan, record, &self.units, targets, resist_by_class, rng_state, walk)?;
        Ok(plan)
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
/// 所有權：`units` 是 Rust 新增的參數，`targets` 是其中的索引（Go 傳 `[]*Unit`）；結果的
/// `target` 直接存這些索引。
pub(crate) fn resolve_native_command_damage_plan(
    plan: &mut NativeCommandDamagePlan,
    record: NativeCommandRecord,
    units: &[Unit],
    targets: &[UnitIdx],
    resist_by_class: &BTreeMap<i64, i64>,
    rng_state: u16,
    walk: Option<&mut NativeCommandDamageRNGWalk<'_>>,
) -> Result<()> {
    plan.results = targets
        .iter()
        .map(|&target| NativeCommandDamageResult {
            target: Some(target),
            hp_before: units[target].hp,
            hp_after: units[target].hp,
            ..Default::default()
        })
        .collect();
    plan.published_target_stages = vec![0; targets.len()];
    let mut resolved = vec![false; targets.len()];
    let mut resolve = |index: i64, rng: u16| -> Result<(u16, bool)> {
        if index < 0 || index >= targets.len() as i64 || resolved[index as usize] {
            return Err(battle_err!("native command damage walk resolved target {index} out of order"));
        }
        let target = &units[targets[index as usize]];
        let resist = resist_by_class.get(&native_command_resistance_class(target)).copied().unwrap_or(0);
        let (result, next) = resolve_native_command_damage(record.damage, record.hit, resist, rng)?;
        let mut hp_after = target.hp;
        if result.hit {
            hp_after -= result.damage;
            if hp_after < 0 {
                hp_after = 0;
            }
        }
        plan.results[index as usize].native_command_damage = result;
        plan.results[index as usize].hp_after = hp_after;
        resolved[index as usize] = true;
        Ok((next, result.hit))
    };
    let target_count = targets.len() as i64;
    let after = match walk {
        Some(walk) => walk(target_count, &mut resolve)?,
        None => {
            let mut rng = rng_state;
            for index in 0..target_count {
                let (next, _) = resolve(index, rng)?;
                rng = next;
            }
            rng
        }
    };
    if let Some(index) = resolved.iter().position(|done| !done) {
        return Err(battle_err!("native command damage walk left target {index} unresolved"));
    }
    plan.rng_after = after;
    Ok(())
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
        let (Some(actor_idx), Some(target_idx)) = (actor, target) else {
            return Err(battle_err!("native AI command single target unavailable"));
        };
        let (actor_unit, target_unit) = (&self.units[actor_idx], &self.units[target_idx]);
        if actor_unit.camp != ENEMY
            || target_unit.camp == ENEMY
            || !actor_unit.on_field
            || !target_unit.on_field
            || actor_unit.hp <= 0
            || target_unit.hp <= 0
        {
            return Err(battle_err!("native AI command single target unavailable"));
        }
        if !(0..=9).contains(&command_id)
            || self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native AI command damage record unavailable id={command_id}"));
        }
        let record = self.native_command_book[command_id as usize];
        if record.effect_mode != 0
            || record.target_code != 0
            || actor_unit.acted
            || !(0..=0xff).contains(&record.mp_cost)
            || actor_unit.mp < record.mp_cost
        {
            return Err(battle_err!("native AI command single-target contract unavailable id={command_id}"));
        }
        let class = native_command_resistance_class(target_unit);
        let resistance = match resist_by_class.get(&class) {
            Some(&raw) if (0..=10).contains(&raw) => raw,
            _ => return Err(battle_err!("native AI command damage missing resistance class={class}")),
        };
        let stages = native_command_damage_stages(command_id)?;
        let (resolved, next_rng) = resolve_native_command_damage(record.damage, record.hit, resistance, rng_state)?;
        let mut hp_after = target_unit.hp;
        if resolved.hit {
            hp_after -= resolved.damage;
            if hp_after < 0 {
                hp_after = 0;
            }
        }
        Ok(NativeCommandDamagePlan {
            actor,
            command_id,
            mp_before: actor_unit.mp,
            mp_after: actor_unit.mp - record.mp_cost,
            actor_acted_before: actor_unit.acted,
            rng_before: rng_state,
            rng_after: next_rng,
            damage_stages: stages,
            results: vec![NativeCommandDamageResult {
                target,
                native_command_damage: resolved,
                hp_before: target_unit.hp,
                hp_after,
            }],
            published_target_stages: vec![0; 1],
            ..Default::default()
        })
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
        let actor_ok = actor.is_some_and(|i| {
            let u = &self.units[i];
            u.camp == ENEMY && u.has_native_record_byte6 && u.has_native_map_presentation
        });
        if !actor_ok
            || !(0..=9).contains(&command_id)
            || self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native AI command damage selector unavailable id={command_id}"));
        }
        let actor_unit = &self.units[actor.unwrap_or_default()];
        self.native_ai_command_origin_in_grid(origin)?;
        let selector = actor_unit.native_record_byte6 as i64;
        if selector != 0 && selector != 1 {
            return Err(battle_err!("native AI command selector={selector} is outside 0/1"));
        }
        let records = native_ai_scoring_records(&self.units)?;
        let mut record = self.native_command_book[command_id as usize];
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
            return Err(battle_err!(
                "native AI command damage target array is empty（actor ({},{}) origin ({},{}) selector={} command={} effect_mode={} target_code={}）",
                actor_unit.native_map_presentation.x,
                actor_unit.native_map_presentation.y,
                origin.x,
                origin.y,
                selector,
                command_id,
                record.effect_mode,
                target_code
            ));
        }
        if actor_unit.acted || !(0..=0xff).contains(&record.mp_cost) || actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native AI command damage insufficient MP"));
        }
        let stages = native_command_damage_stages(command_id)?;
        let mut targets = Vec::with_capacity(indices.len());
        for &index in &indices {
            let target = index as usize;
            let class = native_command_resistance_class(&self.units[target]);
            match resist_by_class.get(&class) {
                Some(raw) if (0..=10).contains(raw) => {}
                _ => return Err(battle_err!("native AI command damage missing resistance class={class}")),
            }
            targets.push(target);
        }
        let mut plan = NativeCommandDamagePlan {
            actor,
            command_id,
            mp_before: actor_unit.mp,
            mp_after: actor_unit.mp - record.mp_cost,
            actor_acted_before: actor_unit.acted,
            rng_before: rng_state,
            damage_stages: stages,
            ..Default::default()
        };
        record.hit = self.command_hit(actor, command_id, record.hit);
        resolve_native_command_damage_plan(&mut plan, record, &self.units, &targets, resist_by_class, rng_state, walk)?;
        let debug = DEBUG_AI.read().unwrap_or_else(|e| e.into_inner()).clone();
        if let Some(debug) = debug {
            // Go 以 %+v 印出結果（含單位指標位址）；Rust 印 Debug 格式，只供診斷。
            debug(format_args!(
                "AI command {} actor ({},{}) origin ({},{}) targets={} rng {}→{} results={:?}",
                command_id,
                actor_unit.native_map_presentation.x,
                actor_unit.native_map_presentation.y,
                origin.x,
                origin.y,
                indices.len(),
                plan.rng_before,
                plan.rng_after,
                plan.results
            ));
        }
        Ok(plan)
    }

    /// nativeAICommandOriginInGrid rejects a winner cell that the 0x15311 route
    /// could never have produced; the presentation owners pass
    /// AIPlan.NativeActionDestination here.
    ///
    /// Go：`native_command0.go:343`，`func (s *State) nativeAICommandOriginInGrid(origin Cell) error`
    pub(crate) fn native_ai_command_origin_in_grid(&self, origin: Cell) -> Result<()> {
        if origin.x < 0 || origin.y < 0 || origin.x >= self.w || origin.y >= self.h {
            return Err(battle_err!(
                "native AI command origin ({},{}) is outside the {}x{} grid",
                origin.x,
                origin.y,
                self.w,
                self.h
            ));
        }
        Ok(())
    }
}

/// nativeCommandDamageStages preserves the 0x525AF command0..9 table consumed
/// by 0x2AEC9. Commands10..12 do not use that indexed target presenter in the
/// current engine slice, so their state-only transaction publishes once.
///
/// Go：`native_command0.go:353`，`func nativeCommandDamageStages(commandID int) (int, error)`
pub(crate) fn native_command_damage_stages(command_id: i64) -> Result<i64> {
    const SHARED: [i64; 10] = [7, 8, 6, 13, 6, 6, 5, 5, 16, 20];
    if (0..SHARED.len() as i64).contains(&command_id) {
        return Ok(SHARED[command_id as usize]);
    }
    if (10..=12).contains(&command_id) {
        return Ok(1);
    }
    Err(battle_err!("native command damage stages unavailable id={command_id}"))
}

/// Go：`native_command0.go:364`，`func ApplyNativeCommandDamageMP(plan *NativeCommandDamagePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command_damage_mp(units: &mut [Unit], plan: Option<&mut NativeCommandDamagePlan>) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native command damage MP publish state changed"));
    };
    let Some(actor) = plan.actor else {
        return Err(battle_err!("native command damage MP publish state changed"));
    };
    if plan.mp_published
        || plan.completed
        || units[actor].mp != plan.mp_before
        || units[actor].acted != plan.actor_acted_before
    {
        return Err(battle_err!("native command damage MP publish state changed"));
    }
    units[actor].mp = plan.mp_after;
    plan.mp_published = true;
    Ok(())
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
    let Some(plan) = plan else {
        return Err(battle_err!("native command damage stage unavailable"));
    };
    if !plan.mp_published
        || plan.completed
        || plan.damage_stages < 1
        || index < 0
        || index >= plan.results.len() as i64
        || stage < 1
        || stage > plan.damage_stages
    {
        return Err(battle_err!("native command damage stage unavailable"));
    }
    let result = plan.results[index as usize];
    let previous = plan.published_target_stages[index as usize];
    let Some(target) =
        result.target.filter(|_| stage >= previous && (stage == plan.damage_stages || stage == previous + 1))
    else {
        return Err(battle_err!("native command damage stage order changed"));
    };
    let want_current = result.hp_before - (result.hp_before - result.hp_after) * previous / plan.damage_stages;
    if units[target].hp != want_current {
        return Err(battle_err!("native command damage target publish state changed"));
    }
    units[target].hp = result.hp_before - (result.hp_before - result.hp_after) * stage / plan.damage_stages;
    plan.published_target_stages[index as usize] = stage;
    Ok(())
}

/// Go：`native_command0.go:396`，`func CompleteNativeCommandDamage(plan *NativeCommandDamagePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_command_damage(units: &mut [Unit], plan: Option<&mut NativeCommandDamagePlan>) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native command damage completion state changed"));
    };
    let Some(actor) = plan.actor else {
        return Err(battle_err!("native command damage completion state changed"));
    };
    if !plan.mp_published
        || plan.completed
        || units[actor].mp != plan.mp_after
        || units[actor].acted != plan.actor_acted_before
    {
        return Err(battle_err!("native command damage completion state changed"));
    }
    for (index, result) in plan.results.iter().enumerate() {
        let complete = result.target.is_some_and(|target| {
            plan.published_target_stages[index] == plan.damage_stages && units[target].hp == result.hp_after
        });
        if !complete {
            return Err(battle_err!("native command damage target is not fully published"));
        }
    }
    units[actor].acted = true;
    plan.completed = true;
    Ok(())
}

#[cfg(test)]
mod native_command0_test;

#[cfg(test)]
mod native_command9_ai_test;
