//! Go 來源：`remake/internal/battle/native_command13.go`（193 行）。

use super::*;

/// NativeCommandHealResult is one final target in the shared recovered
/// ID13..16 HP-restore route. The command family has separate indexed
/// presentation wrappers, which are intentionally not represented here.
///
/// Go：`native_command13.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandHealResult {
    pub target: Option<UnitIdx>,
    pub restore: NativeCommandRestore,
}

impl State {
    /// NativeCommandHealTargets validates the recovered command-13..16 selector
    /// and MP boundary without mutating combat state. The presentation owner uses
    /// this preflight so a missing indexed adapter cannot debit MP or change HP.
    ///
    /// Go：`native_command13.go:19`，`func (s *State) NativeCommandHealTargets(actor, confirmed *Unit, commandID int) ([]*Unit, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_heal_targets(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
    ) -> Result<Vec<UnitIdx>> {
        if !(13..=16).contains(&command_id)
            || self.native_command_book.len() != 36
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native command heal record unavailable id={command_id}"));
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
        if !actor.is_some_and(|i| self.units[i].mp >= record.mp_cost) {
            return Err(battle_err!("native command heal insufficient MP"));
        }
        Ok(targets)
    }

    /// NativeAICommandHealTargets reproduces the 0x15311 consumer: it rebuilds the
    /// 0x14818 target array at the selected winner cell (AIPlan.NativeActionDestination,
    /// the raw [0x53C27]/[0x53C2B] pair), using the raw +6 selector transform
    /// recovered for the 0x1598A/0x15311 path. The actor does not move on this
    /// route, so its presentation cell is not the geometry origin. It does not
    /// reuse the player's confirmed-cursor predicate.
    ///
    /// Go：`native_command13.go:47`，`func (s *State) NativeAICommandHealTargets(actor *Unit, origin Cell, commandID int) ([]*Unit, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_ai_command_heal_targets(
        &self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
    ) -> Result<Vec<UnitIdx>> {
        let Some(actor_unit) = actor.map(|i| &self.units[i]).filter(|u| u.has_native_record_byte6) else {
            return Err(battle_err!("native AI command heal selector unavailable"));
        };
        self.native_ai_command_origin_in_grid(origin)?;
        if !(13..=16).contains(&command_id)
            || self.native_command_book.len() as i64 != NATIVE_COMMAND_RECORD_COUNT
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native AI command heal record unavailable id={command_id}"));
        }
        let selector = actor_unit.native_record_byte6 as i64;
        if selector != 0 && selector != 1 {
            return Err(battle_err!("native AI command heal selector={selector} is outside 0/1"));
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
            return Err(battle_err!("native AI command heal target array is empty"));
        }
        let targets: Vec<UnitIdx> = indices.iter().map(|&index| index as usize).collect();
        if actor_unit.mp < record.mp_cost {
            return Err(battle_err!("native AI command heal insufficient MP"));
        }
        Ok(targets)
    }

    /// ExecuteNativeCommandHeal executes the state portion shared by native IDs
    /// 13..16 (0x21AD9/0x21B99/0x2211C/0x22153 -> 0x21B18): generic two-stage
    /// targets, that record's MP debit, and per-target 0x1C916 restore using the
    /// same record's u16 damage field. It remains fail-closed outside this exact
    /// bounded family and does not attach its renderer/SFX/UI.
    ///
    /// Go：`native_command13.go:98`，`func (s *State) ExecuteNativeCommandHeal(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommandHealResult, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn execute_native_command_heal(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<Vec<NativeCommandHealResult>> {
        let Some(rng) = rng else {
            return Err(battle_err!("missing native command heal state/rng"));
        };
        if !(13..=16).contains(&command_id)
            || self.native_command_book.len() != 36
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native command heal record unavailable id={command_id}"));
        }
        let targets = self.native_command_heal_targets(actor, confirmed, command_id)?;
        let mut restore =
            |target: &mut Unit, amount: i64| apply_native_command_restore(Some(target), amount, Some(&mut *rng));
        self.execute_native_command_heal_targets(actor, &targets, command_id, Some(&mut restore))
    }

    /// ExecuteNativeAICommandHeal consumes only the target array rebuilt by the
    /// recovered 0x15311 AI owner. Presentation remains owned by the caller.
    ///
    /// Go：`native_command13.go:116`，`func (s *State) ExecuteNativeAICommandHeal(actor *Unit, origin Cell, commandID int, rng *rand.Rand) ([]NativeCommandHealResult, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn execute_native_ai_command_heal(
        &mut self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<Vec<NativeCommandHealResult>> {
        let Some(rng) = rng else {
            return Err(battle_err!("missing native AI command heal state/rng"));
        };
        let targets = self.native_ai_command_heal_targets(actor, origin, command_id)?;
        let mut restore =
            |target: &mut Unit, amount: i64| apply_native_command_restore(Some(target), amount, Some(&mut *rng));
        self.execute_native_command_heal_targets(actor, &targets, command_id, Some(&mut restore))
    }

    /// ExecuteNativeCommandHealNative 是 ExecuteNativeCommandHeal 的原版亂數版：每個目標的
    /// 0x1C916 只走一步 0x4E893（第七章 r4 追蹤：0x1C965 一次，eax=amount*9/10），回傳
    /// 走完的亂數字組。有原版 `+6` 出處的隊伍一律用這一版。
    ///
    /// Go：`native_command13.go:132`，`func (s *State) ExecuteNativeCommandHealNative(actor, confirmed *Unit, commandID int, rngState uint16) ([]NativeCommandHealResult, uint16, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn execute_native_command_heal_native(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng_state: u16,
    ) -> Result<(Vec<NativeCommandHealResult>, u16)> {
        if !(13..=16).contains(&command_id)
            || self.native_command_book.len() != 36
            || self.native_command_book[command_id as usize].id != command_id
        {
            return Err(battle_err!("native command heal record unavailable id={command_id}"));
        }
        let targets = self.native_command_heal_targets(actor, confirmed, command_id)?;
        self.execute_native_command_heal_targets_native(actor, &targets, command_id, rng_state)
    }

    /// ExecuteNativeAICommandHealNative 是 ExecuteNativeAICommandHeal 的原版亂數版。
    ///
    /// Go：`native_command13.go:147`，`func (s *State) ExecuteNativeAICommandHealNative(actor *Unit, origin Cell, commandID int, rngState uint16) ([]NativeCommandHealResult, uint16, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn execute_native_ai_command_heal_native(
        &mut self,
        actor: Option<UnitIdx>,
        origin: Cell,
        command_id: i64,
        rng_state: u16,
    ) -> Result<(Vec<NativeCommandHealResult>, u16)> {
        let targets = self.native_ai_command_heal_targets(actor, origin, command_id)?;
        self.execute_native_command_heal_targets_native(actor, &targets, command_id, rng_state)
    }

    /// Go：`native_command13.go:158`，`func (s *State) executeNativeCommandHealTargetsNative(actor *Unit, targets []*Unit, commandID int, rngState uint16) ([]NativeCommandHealResult, uint16, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    pub(crate) fn execute_native_command_heal_targets_native(
        &mut self,
        actor: Option<UnitIdx>,
        targets: &[UnitIdx],
        command_id: i64,
        rng_state: u16,
    ) -> Result<(Vec<NativeCommandHealResult>, u16)> {
        let mut state = rng_state;
        let mut restore = |target: &mut Unit, amount: i64| {
            let (restore, next) = apply_native_command_restore_native(Some(target), amount, state)?;
            state = next;
            Ok(restore)
        };
        let results = self.execute_native_command_heal_targets(actor, targets, command_id, Some(&mut restore))?;
        Ok((results, state))
    }

    /// Go：`native_command13.go:174`，`func (s *State) executeNativeCommandHealTargets(actor *Unit, targets []*Unit, commandID int, restoreFn func(target *Unit, amount int) (NativeCommandRestore, error)) ([]NativeCommandHealResult, error)`
    /// 所有權：`actor` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：`targets` 是 `self.units` 的索引清單（Go 傳 `[]*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn execute_native_command_heal_targets(
        &mut self,
        actor: Option<UnitIdx>,
        targets: &[UnitIdx],
        command_id: i64,
        restore_fn: Option<&mut dyn FnMut(&mut Unit, i64) -> Result<NativeCommandRestore>>,
    ) -> Result<Vec<NativeCommandHealResult>> {
        let Some(restore_fn) = restore_fn.filter(|_| {
            (13..=16).contains(&command_id)
                && self.native_command_book.len() as i64 == NATIVE_COMMAND_RECORD_COUNT
                && self.native_command_book[command_id as usize].id == command_id
        }) else {
            return Err(battle_err!("native command heal target transaction unavailable id={command_id}"));
        };
        let record = self.native_command_book[command_id as usize];
        if !spend_native_command_mp(actor.map(|i| &mut self.units[i]), record.mp_cost) {
            return Err(battle_err!("native command heal insufficient MP"));
        }
        let mut results = Vec::with_capacity(targets.len());
        for &target in targets {
            let restore = restore_fn(&mut self.units[target], record.damage)?;
            results.push(NativeCommandHealResult { target: Some(target), restore });
        }
        if let Some(actor) = actor {
            self.units[actor].acted = true;
        }
        Ok(results)
    }
}

#[cfg(test)]
mod native_command13_test;
