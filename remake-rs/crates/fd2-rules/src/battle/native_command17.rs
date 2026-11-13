//! Go 來源：`remake/internal/battle/native_command17.go`（301 行）。

use super::*;

impl State {
    /// NativeCommandModifierTargets validates the recovered player selector for
    /// command IDs 17..19 without mutating state. The player renderer remains
    /// intentionally disconnected until its indexed presentation is recovered.
    ///
    /// Go：`native_command17.go:12`，`func (s *State) NativeCommandModifierTargets(actor, confirmed *Unit, commandID int) ([]*Unit, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_modifier_targets(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
    ) -> Result<Vec<UnitIdx>> {
        if !(17..=19).contains(&command_id)
            || self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native command modifier record unavailable id={command_id}"));
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
        self.preflight_native_command_modifier_actor(actor, command_id)?;
        Ok(targets)
    }

    /// NativeAICommandModifierTargets rebuilds the final target array through the
    /// recovered 0x15311 raw-selector owner. It does not reuse the player's
    /// confirmed-cursor predicate.
    ///
    /// Go：`native_command17.go:38`，`func (s *State) NativeAICommandModifierTargets(actor *Unit, origin Cell, commandID int) ([]*Unit, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_ai_command_modifier_targets(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
    ) -> Result<Vec<UnitIdx>> {
        let Some(actor_unit) = actor.map(|i| &self.units[i]).filter(|u| u.has_native_record_byte6) else {
            return Err(battle_err!("native AI command modifier selector unavailable"));
        };
        self.native_ai_command_origin_in_grid(origin)?;
        if !(17..=19).contains(&command_id)
            || self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native AI command modifier record unavailable id={command_id}"));
        }
        let selector = actor_unit.native_record_byte6 as i64;
        if selector != 0 && selector != 1 {
            return Err(battle_err!("native AI command modifier selector={selector} is outside 0/1"));
        }
        let records = native_ai_scoring_records(&self.units)?;
        let record = self.native_command_book[command_id as usize];
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
            return Err(battle_err!("native AI command modifier target array is empty"));
        }
        let targets: Vec<UnitIdx> = indices.iter().map(|&index| index as usize).collect();
        self.preflight_native_command_modifier_actor(actor, command_id)?;
        Ok(targets)
    }

    /// ExecuteNativeCommandModifier applies the player selector and the exact
    /// ID17..19 raw transaction. No presentation or status name is inferred.
    ///
    /// Go：`native_command17.go:88`，`func (s *State) ExecuteNativeCommandModifier(actor, confirmed *Unit, commandID int, rngState uint16) (NativeCommandModifierResult, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command_modifier(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeCommandModifierResult> {
        let targets = self.native_command_modifier_targets(actor, confirmed, command_id)?;
        self.execute_native_command_modifier_targets(actor, &targets, command_id, rng_state)
    }

    /// ExecuteNativeAICommandModifier consumes only the raw target array rebuilt
    /// for the recovered mode-11 owner.
    ///
    /// Go：`native_command17.go:98`，`func (s *State) ExecuteNativeAICommandModifier(actor *Unit, origin Cell, commandID int, rngState uint16) (NativeCommandModifierResult, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_ai_command_modifier(
        &mut self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeCommandModifierResult> {
        let mut plan = self.plan_native_ai_command_modifier(actor, origin, command_id, rng_state)?;
        if let Err(err) = publish_native_ai_command_modifier(&mut self.units, Some(&mut plan)) {
            let _ = abort_native_ai_command_modifier(&mut self.units, Some(&mut plan));
            return Err(err);
        }
        if let Err(err) = complete_native_ai_command_modifier(&mut self.units, Some(&mut plan)) {
            let _ = abort_native_ai_command_modifier(&mut self.units, Some(&mut plan));
            return Err(err);
        }
        Ok(plan.result)
    }
}

/// NativeAICommandModifierPlan keeps the single 0x226EA/0x2282F/0x22960
/// transaction private until the handler-owned mask boundary has been drawn.
///
/// Go：`native_command17.go:116`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeAICommandModifierPlan {
    pub actor: Option<UnitIdx>,
    pub command_id: i64,
    pub targets: Vec<UnitIdx>,
    pub result: NativeCommandModifierResult,
    pub after: Vec<NativeCommand34TargetState>,
    pub(crate) mp_before: i64,
    pub(crate) mp_after: i64,
    pub(crate) before: Vec<NativeCommand34TargetState>,
    pub(crate) published: bool,
    pub(crate) completed: bool,
}

impl State {
    /// Go：`native_command17.go:129`，`func (s *State) PlanNativeAICommandModifier(actor *Unit, origin Cell, commandID int, rngState uint16) (*NativeAICommandModifierPlan, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn plan_native_ai_command_modifier(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeAICommandModifierPlan> {
        let Some(actor_idx) = actor.filter(|&i| !self.units[i].acted) else {
            return Err(battle_err!("native AI command modifier actor unavailable"));
        };
        let targets = self.native_ai_command_modifier_targets(actor, origin, command_id)?;
        let (mut records, indices) = native_command_modifier_records(&self.units, &targets)?;
        let result = apply_native_command_modifier(&mut records, &indices, command_id, rng_state)?;
        let debit_id = if command_id == 17 { 18 } else { command_id };
        let cost = self.native_command_book[debit_id as usize].mp_cost;
        let actor_unit = &self.units[actor_idx];
        if cost < 0 || actor_unit.mp < cost {
            return Err(battle_err!("native AI command modifier MP debit unavailable"));
        }
        let after = native_command_modifier_shadow_states(&self.units, &targets, &records);
        let debug = DEBUG_AI.read().unwrap_or_else(|e| e.into_inner()).clone();
        if let Some(debug) = debug {
            debug(format_args!(
                "AI command {} actor ({},{}) origin ({},{}) targets={} rng {}→{} mp {}→{}",
                command_id,
                actor_unit.x,
                actor_unit.y,
                origin.x,
                origin.y,
                targets.len(),
                rng_state,
                result.rng_state,
                actor_unit.mp,
                actor_unit.mp - cost
            ));
        }
        Ok(NativeAICommandModifierPlan {
            actor,
            command_id,
            before: native_command34_states(&self.units, &targets),
            targets,
            result,
            after,
            mp_before: actor_unit.mp,
            mp_after: actor_unit.mp - cost,
            published: false,
            completed: false,
        })
    }
}

/// Go：`native_command17.go:173`，`func PublishNativeAICommandModifier(plan *NativeAICommandModifierPlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn publish_native_ai_command_modifier(
    units: &mut [Unit],
    plan: Option<&mut NativeAICommandModifierPlan>,
) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native AI command modifier publish boundary unavailable"));
    };
    let Some(actor) = plan.actor.filter(|&a| {
        !plan.published
            && !plan.completed
            && !units[a].acted
            && units[a].mp == plan.mp_before
            && native_command34_equal_current(units, &plan.before)
    }) else {
        return Err(battle_err!("native AI command modifier publish boundary unavailable"));
    };
    native_command34_publish(units, &plan.after);
    units[actor].mp = plan.mp_after;
    plan.published = true;
    Ok(())
}

/// Go：`native_command17.go:184`，`func CompleteNativeAICommandModifier(plan *NativeAICommandModifierPlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn complete_native_ai_command_modifier(
    units: &mut [Unit],
    plan: Option<&mut NativeAICommandModifierPlan>,
) -> Result<()> {
    let Some(plan) = plan else {
        return Err(battle_err!("native AI command modifier completion boundary unavailable"));
    };
    let Some(actor) = plan.actor.filter(|&a| {
        plan.published
            && !plan.completed
            && !units[a].acted
            && units[a].mp == plan.mp_after
            && native_command34_equal_current(units, &plan.after)
    }) else {
        return Err(battle_err!("native AI command modifier completion boundary unavailable"));
    };
    units[actor].acted = true;
    plan.completed = true;
    Ok(())
}

/// Go：`native_command17.go:194`，`func AbortNativeAICommandModifier(plan *NativeAICommandModifierPlan) error`
/// 所有權：`units` 是 Rust 新增的參數。Go 經由結構欄位裡的 `*Unit`（Rust 是 `UnitIdx`）讀寫單位，Rust 需要單位陣列才能解參照；傳入產生那些索引時的同一個陣列（通常是 `state.units`）。
pub fn abort_native_ai_command_modifier(
    units: &mut [Unit],
    plan: Option<&mut NativeAICommandModifierPlan>,
) -> Result<()> {
    let Some(plan) = plan else {
        return Ok(());
    };
    if plan.completed {
        return Err(battle_err!("native AI command modifier completed plan cannot abort"));
    }
    native_command34_publish(units, &plan.before);
    if let Some(actor) = plan.actor {
        units[actor].mp = plan.mp_before;
        units[actor].acted = false;
    }
    plan.published = false;
    Ok(())
}

impl State {
    /// ValidateNativeCommandModifierTargets performs the complete raw projection
    /// preflight without consuming RNG or mutating any unit. Presentation owners
    /// call this before publishing sample/palette frames.
    ///
    /// Go：`native_command17.go:213`，`func (s *State) ValidateNativeCommandModifierTargets(actor *Unit, targets []*Unit, commandID int) error`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    pub fn validate_native_command_modifier_targets(
        &self,
        actor: Option<UnitIdx>,
        targets: &[UnitIdx],
        command_id: i64,
    ) -> Result<()> {
        self.preflight_native_command_modifier_actor(actor, command_id)?;
        native_command_modifier_records(&self.units, targets)?;
        Ok(())
    }

    /// Go：`native_command17.go:221`，`func (s *State) preflightNativeCommandModifierActor(actor *Unit, commandID int) error`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn preflight_native_command_modifier_actor(
        &self,
        actor: Option<UnitIdx>,
        command_id: i64,
    ) -> Result<()> {
        let Some(actor) = actor.filter(|_| {
            (17..=19).contains(&command_id) && self.native_command_book.len() as i64 == NATIVE_COMMAND_RECORD_COUNT
        }) else {
            return Err(battle_err!("native command modifier transaction unavailable id={command_id}"));
        };
        let debit_id = if command_id == 17 { 18 } else { command_id };
        if self.native_command_book[debit_id as usize].id != debit_id {
            return Err(battle_err!("native command modifier debit record unavailable id={debit_id}"));
        }
        if self.units[actor].mp < self.native_command_book[debit_id as usize].mp_cost {
            return Err(battle_err!("native command modifier insufficient MP"));
        }
        Ok(())
    }

    /// Go：`native_command17.go:239`，`func (s *State) executeNativeCommandModifierTargets(actor *Unit, targets []*Unit, commandID int, rngState uint16) (NativeCommandModifierResult, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    pub(crate) fn execute_native_command_modifier_targets(
        &mut self,
        actor: Option<UnitIdx>,
        targets: &[UnitIdx],
        command_id: i64,
        rng_state: u16,
    ) -> Result<NativeCommandModifierResult> {
        self.preflight_native_command_modifier_actor(actor, command_id)?;
        let (mut records, indices) = native_command_modifier_records(&self.units, targets)?;
        let result = apply_native_command_modifier(&mut records, &indices, command_id, rng_state)?;
        let debit_id = if command_id == 17 { 18 } else { command_id };
        let cost = self.native_command_book[debit_id as usize].mp_cost;
        if !spend_native_command_mp(actor.map(|i| &mut self.units[i]), cost) {
            return Err(battle_err!("native command modifier insufficient MP after preflight"));
        }
        publish_native_command_modifier_records(&mut self.units, targets, &records);
        if let Some(actor) = actor {
            self.units[actor].acted = true;
        }
        Ok(result)
    }
}

/// Go：`native_command17.go:263`，`func publishNativeCommandModifierRecords(targets []*Unit, records []byte)`
///
/// 所有權：Rust 版多一個開頭參數 `units`，`targets` 是其中的索引（Go 傳 `[]*Unit`）；第 `i` 個目標
/// 讀 `records` 的第 `i` 筆。
pub(crate) fn publish_native_command_modifier_records(units: &mut [Unit], targets: &[UnitIdx], records: &[u8]) {
    for (i, &target) in targets.iter().enumerate() {
        let base = i * NATIVE_RECORD_SIZE as usize;
        let word = |off: usize| i16::from_le_bytes([records[base + off], records[base + off + 1]]) as i64;
        let unit = &mut units[target];
        unit.native_transient.copy_from_slice(&records[base + 0x22..base + 0x28]);
        unit.ap = word(0x48);
        unit.dp = word(0x4a);
        unit.hit = word(0x4c);
        unit.ev = word(0x4e);
    }
}

/// Go：`native_command17.go:274`，`func nativeCommandModifierRecords(targets []*Unit) ([]byte, []byte, error)`
///
/// 所有權：Rust 版多一個開頭參數 `units`，`targets` 是其中的索引（Go 傳 `[]*Unit`）。
pub(crate) fn native_command_modifier_records(units: &[Unit], targets: &[UnitIdx]) -> Result<(Vec<u8>, Vec<u8>)> {
    if targets.is_empty() || targets.len() > 256 {
        return Err(battle_err!("native command modifier target count={}", targets.len()));
    }
    let size = NATIVE_RECORD_SIZE as usize;
    let mut records = vec![0u8; targets.len() * size];
    let mut indices = vec![0u8; targets.len()];
    for (i, &target) in targets.iter().enumerate() {
        let target = &units[target];
        if !target.has_native_record_class || target.lv < 0 || target.lv > u8::MAX as i64 {
            return Err(battle_err!("native command modifier target {i} lacks raw class/level provenance"));
        }
        let words = [target.ap, target.dp, target.hit, target.ev];
        if words.iter().any(|&value| value < i16::MIN as i64 || value > i16::MAX as i64) {
            return Err(battle_err!("native command modifier target {i} derived word outside range"));
        }
        let base = i * size;
        records[base + 0x20] = target.native_record_class;
        records[base + 0x21] = target.lv as u8;
        records[base + 0x22..base + 0x28].copy_from_slice(&target.native_transient);
        for (j, value) in words.iter().enumerate() {
            records[base + 0x48 + j * 2..base + 0x4a + j * 2].copy_from_slice(&(*value as i16).to_le_bytes());
        }
        indices[i] = i as u8;
    }
    Ok((records, indices))
}

/// Go 在 `PlanNativeAICommandModifier`、`PlanNativeCompoundCommand34` 等處的影子寫法：
/// 把每個目標淺複製一份（`copied := *target`）、把記錄寫進副本，再取副本的狀態並把 `Target`
/// 換回真正的目標。Rust 用深複製的副本陣列做同一件事（沒有 Go 對應函式）。
pub(crate) fn native_command_modifier_shadow_states(
    units: &[Unit],
    targets: &[UnitIdx],
    records: &[u8],
) -> Vec<NativeCommand34TargetState> {
    let mut shadow: Vec<Unit> = targets.iter().map(|&target| units[target].clone()).collect();
    let local: Vec<UnitIdx> = (0..targets.len()).collect();
    publish_native_command_modifier_records(&mut shadow, &local, records);
    let mut states = native_command34_states(&shadow, &local);
    for (state, &target) in states.iter_mut().zip(targets) {
        state.target = Some(target);
    }
    states
}

#[cfg(test)]
mod native_command17_test;
