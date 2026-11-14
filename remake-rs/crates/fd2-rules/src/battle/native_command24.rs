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
    /// Go：Rolled 是 0x1C81F 內的擲出值（截斷到 0 與剩餘 HP 之前），經驗公式用它。
    pub rolled: i64,
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
    match command_id {
        24 => (15, true),
        28 => (20, true),
        29 => (12, true),
        31 => (18, true),
        _ => (0, false),
    }
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
    let Some(rng) = rng else {
        return Err(battle_err!("nil rng"));
    };
    if multiplier <= 0 {
        return Err(battle_err!("invalid native derived-strike multiplier={multiplier}"));
    }
    let amount = actor_ap.wrapping_mul(multiplier).wrapping_div(10).wrapping_sub(target_dp);
    let damage =
        amount.wrapping_mul(9).wrapping_div(10).wrapping_add(rng.intn(100).wrapping_mul(amount).wrapping_div(1000));
    Ok((amount, damage))
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
        let mut plan = self.plan_native_command_derived_strike(actor, confirmed, command_id, rng)?;
        apply_native_command_derived_strike_mp(&mut self.units, Some(&plan))?;
        for i in 0..plan.results.len() as i64 {
            if command_id == 28 {
                for stage in 1..=plan.target_impact_count {
                    apply_native_command28_target_stage(&mut self.units, Some(&mut plan), i, stage)?;
                }
            } else {
                apply_native_command_derived_strike_target(&mut self.units, Some(&plan), i)?;
            }
        }
        complete_native_command_derived_strike(&mut self.units, Some(&plan))?;
        Ok(plan.results)
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
        if rng.is_none() {
            return Err(battle_err!("missing native derived-strike state/rng"));
        }
        let book = &self.native_command_book;
        if book.len() != 36
            || command_id < 0
            || command_id >= book.len() as i64
            || book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native derived-strike record unavailable id={command_id}"));
        }
        let record = book[command_id as usize];
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
        self.plan_native_command_derived_strike_targets(actor, &targets, command_id, rng)
    }

    /// NativeCommand29EffectTargets keeps the original self-centered cursor separate
    /// from the affected enemy units. The direct-unit entry above remains available.
    ///
    /// Go：`native_command24.go:135`，`func (s *State) NativeCommand29EffectTargets(actor *Unit, cursor Cell) ([]*Unit, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command29_effect_targets(&self, actor: Option<UnitIdx>, cursor: Cell) -> Result<Vec<UnitIdx>> {
        let book = &self.native_command_book;
        let Some(actor_idx) = actor.filter(|&i| {
            self.units[i].camp != ENEMY && book.len() as i64 == NATIVE_COMMAND_RECORD_COUNT && book[29].id == 29
        }) else {
            return Err(battle_err!("native player command29 record/actor unavailable"));
        };
        let record = book[29];
        let flags = self.native_command_base_flags()?;
        if record.selection_mode == 0 {
            return native_command_self_centered_targets(
                self.w,
                self.h,
                Some(actor_idx),
                cursor,
                record.effect_mode,
                &flags,
                &self.units,
            );
        }
        native_command_effect_targets_at_cursor(
            self.w,
            self.h,
            Some(actor_idx),
            cursor,
            record.selection_mode,
            record.effect_mode,
            record.target_code,
            &flags,
            &self.units,
        )
    }

    /// Go：`native_command24.go:150`，`func (s *State) PlanNativeCommand29AtCursor(actor *Unit, cursor Cell, rng *rand.Rand) (*NativeCommandDerivedStrikePlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_command29_at_cursor(
        &self,
        actor: Option<UnitIdx>,
        cursor: Cell,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<NativeCommandDerivedStrikePlan> {
        let targets = self.native_command29_effect_targets(actor, cursor)?;
        self.plan_native_command_derived_strike_targets(actor, &targets, 29, rng)
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
        let book = &self.native_command_book;
        let Some(rng) = rng.filter(|_| {
            command_id >= 0
                && command_id < book.len() as i64
                && book.len() as i64 == NATIVE_COMMAND_RECORD_COUNT
                && book[command_id as usize].id == command_id
        }) else {
            return Err(battle_err!("native derived-strike record/rng unavailable id={command_id}"));
        };
        let (multiplier, ok) = native_command_derived_strike_multiplier(command_id);
        if !ok {
            return Err(battle_err!("native derived-strike command unavailable id={command_id}"));
        }
        let record = book[command_id as usize];
        let Some(actor_idx) =
            actor.filter(|&i| (0..=0xff).contains(&record.mp_cost) && self.units[i].mp >= record.mp_cost)
        else {
            return Err(battle_err!("native command {command_id} insufficient MP"));
        };
        let actor_unit = &self.units[actor_idx];
        if command_id == 28 && (!actor_unit.has_battle_fig || actor_unit.battle_fig != 19) {
            return Err(battle_err!("native command28 requires proven selector19 impact contract"));
        }
        let mut plan = NativeCommandDerivedStrikePlan {
            actor: Some(actor_idx),
            command_id,
            mp_before: actor_unit.mp,
            mp_after: actor_unit.mp - record.mp_cost,
            results: Vec::with_capacity(targets.len()),
            ..Default::default()
        };
        if command_id == 28 {
            // FIGANI59 has eight target raw+4 markers. Other selectors require
            // their own reviewed contract rather than a single-marker guess.
            plan.target_impact_count = 8;
            plan.published_target_stages = vec![0; targets.len()];
        }
        // Go 另檢查 nil 目標；Rust 的目標是 `self.units` 的索引，不會是 nil。
        for &target_idx in targets {
            let target = &self.units[target_idx];
            let (amount, damage) =
                resolve_native_command_derived_strike_damage(actor_unit.ap, target.dp, multiplier, Some(&mut *rng))?;
            let mut applied_damage = damage.max(0);
            // 0x27C44 clamps the roll to original HP before command28 divides
            // its cumulative eight impact stages. Keep the final capped damage.
            if applied_damage > target.hp {
                applied_damage = target.hp;
            }
            let hp_after = target.hp - applied_damage;
            let mut raw_byte5_after = target.native_record_byte5;
            if hp_after == 0 && target.has_native_record_byte5 {
                raw_byte5_after = 1;
            }
            plan.results.push(NativeCommand24Damage {
                target: Some(target_idx),
                amount,
                rolled: damage,
                damage: applied_damage,
                hp_before: target.hp,
                hp_after,
                native_record_byte5_before: target.native_record_byte5,
                native_record_byte5_after: raw_byte5_after,
                ..Default::default()
            });
        }
        Ok(plan)
    }
}

/// Go：`native_command24.go:215`，`func ApplyNativeCommandDerivedStrikeMP(plan *NativeCommandDerivedStrikePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command_derived_strike_mp(
    units: &mut [Unit],
    plan: Option<&NativeCommandDerivedStrikePlan>,
) -> Result<()> {
    let Some((plan, actor)) = plan.and_then(|p| p.actor.map(|a| (p, a))) else {
        return Err(battle_err!("native derived-strike MP publish state changed"));
    };
    let unit = &mut units[actor];
    if unit.mp != plan.mp_before || unit.acted {
        return Err(battle_err!("native derived-strike MP publish state changed"));
    }
    unit.mp = plan.mp_after;
    Ok(())
}

/// Go：`native_command24.go:223`，`func ApplyNativeCommandDerivedStrikeTarget(plan *NativeCommandDerivedStrikePlan, index int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command_derived_strike_target(
    units: &mut [Unit],
    plan: Option<&NativeCommandDerivedStrikePlan>,
    index: i64,
) -> Result<()> {
    let Some(plan) = plan.filter(|p| index >= 0 && index < p.results.len() as i64) else {
        return Err(battle_err!("native derived-strike target publish index unavailable"));
    };
    if plan.command_id == 28 {
        return Err(battle_err!("native command28 requires ordered impact-stage publication"));
    }
    let result = plan.results[index as usize];
    let Some(target) = result.target.map(|t| &mut units[t]).filter(|t| {
        t.hp == result.hp_before
            && !(t.has_native_record_byte5 && t.native_record_byte5 != result.native_record_byte5_before)
    }) else {
        return Err(battle_err!("native derived-strike target publish state changed"));
    };
    if plan.command_id == 30 {
        target.apply_hp_damage(result.hp_before - result.hp_after);
        return Ok(());
    }
    target.hp = result.hp_after;
    if target.has_native_record_byte5 {
        target.native_record_byte5 = result.native_record_byte5_after;
    }
    Ok(())
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
    let Some(plan) = plan.filter(|p| {
        p.command_id == 28
            && p.target_impact_count == 8
            && index >= 0
            && index < p.results.len() as i64
            && (0..=8).contains(&stage)
    }) else {
        return Err(battle_err!("native command28 stage projection unavailable"));
    };
    let mut result = plan.results[index as usize];
    let Some(target) = result.target.map(|t| &units[t]) else {
        return Err(battle_err!("native command28 target projection unavailable"));
    };
    result.damage = result.damage * stage / 8;
    result.hp_after = result.hp_before - result.damage;
    result.native_record_byte5_after = result.native_record_byte5_before;
    if result.hp_after == 0 && target.has_native_record_byte5 {
        result.native_record_byte5_after = 1;
    }
    Ok(result)
}

/// Go：`native_command24.go:265`，`func ApplyNativeCommand28TargetStage(plan *NativeCommandDerivedStrikePlan, index, stage int) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn apply_native_command28_target_stage(
    units: &mut [Unit],
    plan: Option<&mut NativeCommandDerivedStrikePlan>,
    index: i64,
    stage: i64,
) -> Result<()> {
    let Some(plan) = plan.filter(|p| {
        p.actor.is_some_and(|a| units[a].mp == p.mp_after && !units[a].acted)
            && index >= 0
            && index < p.published_target_stages.len() as i64
            && stage == p.published_target_stages[index as usize] + 1
    }) else {
        return Err(battle_err!("native command28 stage publication state changed"));
    };
    let before = native_command28_stage_result(units, Some(&*plan), index, stage - 1)?;
    let after = native_command28_stage_result(units, Some(&*plan), index, stage)?;
    let Some(target) = after.target.map(|t| &mut units[t]).filter(|t| {
        t.hp == before.hp_after
            && !(t.has_native_record_byte5 && t.native_record_byte5 != before.native_record_byte5_after)
    }) else {
        return Err(battle_err!("native command28 target changed before impact"));
    };
    target.hp = after.hp_after;
    if target.has_native_record_byte5 {
        target.native_record_byte5 = after.native_record_byte5_after;
    }
    plan.published_target_stages[index as usize] = stage;
    Ok(())
}

/// RestoreNativeCommand30Targets restores the death-presentation provenance as
/// well as HP/raw +5. It is confined to the ID30 owner transaction.
///
/// Go：`native_command24.go:291`，`func RestoreNativeCommand30Targets(plan *NativeCommandDerivedStrikePlan)`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn restore_native_command30_targets(units: &mut [Unit], plan: Option<&NativeCommandDerivedStrikePlan>) {
    let Some(plan) = plan.filter(|p| p.command_id == 30) else {
        return;
    };
    for result in &plan.results {
        let Some(target) = result.target.map(|t| &mut units[t]) else {
            continue;
        };
        target.hp = result.hp_before;
        target.native_record_byte5 = result.native_record_byte5_before;
        target.native_death_flag_before = result.death_flag_before;
        target.native_death_flag_valid = result.death_flag_valid_before;
    }
}

/// Go：`native_command24.go:306`，`func CompleteNativeCommandDerivedStrike(plan *NativeCommandDerivedStrikePlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_command_derived_strike(
    units: &mut [Unit],
    plan: Option<&NativeCommandDerivedStrikePlan>,
) -> Result<()> {
    let Some((plan, actor)) = plan.and_then(|p| p.actor.map(|a| (p, a))) else {
        return Err(battle_err!("native derived-strike completion state changed"));
    };
    if units[actor].mp != plan.mp_after || units[actor].acted {
        return Err(battle_err!("native derived-strike completion state changed"));
    }
    if plan.command_id == 28 {
        if plan.published_target_stages.len() != plan.results.len() {
            return Err(battle_err!("native command28 impact publication unavailable"));
        }
        if plan.published_target_stages.iter().any(|&stage| stage != 8) {
            return Err(battle_err!("native command28 impacts are incomplete"));
        }
    }
    for result in &plan.results {
        let published = result.target.map(|t| &units[t]).is_some_and(|t| {
            t.hp == result.hp_after
                && !(t.has_native_record_byte5 && t.native_record_byte5 != result.native_record_byte5_after)
        });
        if !published {
            return Err(battle_err!("native derived-strike target is not published"));
        }
    }
    // 0x18D8C applies the invoking actor's completion bit only after 0x1CFF0
    // returns success; presentation owners call this after their final frame.
    units[actor].acted = true;
    Ok(())
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
        let book = &self.native_command_book;
        let Some(actor_idx) = actor.filter(|_| book.len() == 36 && book[30].id == 30) else {
            return Err(battle_err!("native command 30 state unavailable"));
        };
        let record = book[30];
        if record.selection_mode < 0x10 {
            return Err(battle_err!("native command 30 invalid selector mode={}", record.selection_mode));
        }
        // The preceding 0x14818 -> 0x115B6 confirmation is still required. It
        // proves the provided confirmed cursor is a valid target candidate before
        // the special line selector is allowed to mutate MP or HP.
        let flags = self.native_command_base_flags()?;
        let actor_unit = &self.units[actor_idx];
        let selection = native_command_targets(
            self.w,
            self.h,
            Cell { x: actor_unit.x, y: actor_unit.y },
            record.selection_mode,
            record.target_code,
            &flags,
            &self.units,
        )?;
        let confirmed = selection.iter().any(|&candidate| {
            self.units[candidate].x == confirmed_cursor.x && self.units[candidate].y == confirmed_cursor.y
        });
        if !confirmed {
            return Err(battle_err!("confirmed cursor is not a native command 30 candidate"));
        }
        native_command30_targets(
            self.w,
            self.h,
            saved_cursor,
            confirmed_cursor,
            record.selection_mode - 0x10,
            &self.units,
        )
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
        let Some(rng) = rng else {
            return Err(battle_err!("native command 30 rng unavailable"));
        };
        let targets = self.native_command30_effect_targets(actor, saved_cursor, confirmed_cursor)?;
        let record = self.native_command_book[30];
        // `native_command30_effect_targets` 成功代表 actor 不是 None。
        let actor_idx = actor.expect("native_command30_effect_targets 已檢查 actor");
        let actor_unit = &self.units[actor_idx];
        if actor_unit.acted || record.mp_cost < 0 || actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native command 30 insufficient MP or completed actor"));
        }
        let mut plan = NativeCommandDerivedStrikePlan {
            actor: Some(actor_idx),
            command_id: 30,
            mp_before: actor_unit.mp,
            mp_after: actor_unit.mp - record.mp_cost,
            ..Default::default()
        };
        for target_idx in targets {
            let target = &self.units[target_idx];
            let (amount, damage) =
                resolve_native_command_derived_strike_damage(actor_unit.ap, target.dp, 18, Some(&mut *rng))?;
            let applied = damage.max(0).min(target.hp);
            let mut after = target.native_record_byte5;
            if applied > 0 && target.hp - applied == 0 && target.has_native_record_byte5 {
                after = 1;
            }
            // Damage keeps the existing ID30 state-only result's unclamped roll.
            plan.results.push(NativeCommand24Damage {
                target: Some(target_idx),
                amount,
                rolled: damage,
                damage,
                hp_before: target.hp,
                hp_after: target.hp - applied,
                native_record_byte5_before: target.native_record_byte5,
                native_record_byte5_after: after,
                death_flag_before: target.native_death_flag_before,
                death_flag_valid_before: target.native_death_flag_valid,
            });
        }
        Ok(plan)
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
        let Some(rng) = rng else {
            return Err(battle_err!("native command 30 state unavailable"));
        };
        let targets = self.native_command30_effect_targets(actor, saved_cursor, confirmed_cursor)?;
        let record = self.native_command_book[30];
        // `native_command30_effect_targets` 成功代表 actor 不是 None。
        let actor_idx = actor.expect("native_command30_effect_targets 已檢查 actor");
        if !spend_native_command_mp(Some(&mut self.units[actor_idx]), record.mp_cost) {
            return Err(battle_err!("native command 30 insufficient MP"));
        }
        let mut results = Vec::with_capacity(targets.len());
        for target_idx in targets {
            let (actor_ap, target_dp) = (self.units[actor_idx].ap, self.units[target_idx].dp);
            let (amount, damage) =
                resolve_native_command_derived_strike_damage(actor_ap, target_dp, 18, Some(&mut *rng))?;
            self.units[target_idx].apply_hp_damage(damage);
            results.push(NativeCommand24Damage {
                target: Some(target_idx),
                amount,
                rolled: damage,
                damage,
                ..Default::default()
            });
        }
        self.units[actor_idx].acted = true;
        Ok(results)
    }
}

#[cfg(test)]
mod native_command24_test;
