//! Go 來源：`remake/internal/battle/native_command20.go`（128 行）。

use super::*;

/// NativeCommandRestore is the recovered 0x1C916 HP mutation. Display code in
/// the original receives the uncapped rolled amount; this data result instead
/// exposes the actual HP delta so an engine caller cannot mistake an over-cap
/// display number for a state mutation.
///
/// Go：`native_command20.go:14`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandRestore {
    pub rolled: i64,
    pub actual: i64,
}

/// ApplyNativeCommandRestore mirrors 0x1C916 for a supplied raw amount:
/// amount*9/10 + (rand()%100)*amount/1000, add current HP and clamp max HP.
///
/// Go：`native_command20.go:21`，`func ApplyNativeCommandRestore(target *Unit, amount int, rng *rand.Rand) (NativeCommandRestore, error)`
pub fn apply_native_command_restore(
    target: Option<&mut Unit>,
    amount: i64,
    rng: Option<&mut dyn GoRand>,
) -> Result<NativeCommandRestore> {
    let (Some(target), Some(rng)) = (target, rng) else {
        return Err(battle_err!("invalid native command restore state"));
    };
    if amount < 0 || target.max_hp < 0 || target.hp < 0 || target.hp > target.max_hp {
        return Err(battle_err!("invalid native command restore state"));
    }
    let rolled = amount.wrapping_mul(9) / 10 + rng.intn(100).wrapping_mul(amount) / 1000;
    let before = target.hp;
    target.hp += rolled;
    if target.hp > target.max_hp {
        target.hp = target.max_hp;
    }
    Ok(NativeCommandRestore { rolled, actual: target.hp - before })
}

/// ApplyNativeCommandRestoreNative 是 ApplyNativeCommandRestore 的原版亂數版：0x1C916 先
/// 走一步 0x4E893，再 `amount*9/10 + (rng%100)*amount/1000`、上限 MaxHP。回傳走完的
/// 亂數字組（原版 r4 seq 603：指令 13 amount 70，rng 45295%100=95 → 63+6=69）。
///
/// Go：`native_command20.go:37`，`func ApplyNativeCommandRestoreNative(target *Unit, amount int, rngState uint16) (NativeCommandRestore, uint16, error)`
pub fn apply_native_command_restore_native(
    target: Option<&mut Unit>,
    amount: i64,
    rng_state: u16,
) -> Result<(NativeCommandRestore, u16)> {
    let Some(target) = target else {
        return Err(battle_err!("invalid native command restore state"));
    };
    if amount < 0 || target.max_hp < 0 || target.hp < 0 || target.hp > target.max_hp {
        return Err(battle_err!("invalid native command restore state"));
    }
    let next = fdother::native_rng_step(rng_state);
    let rolled = amount.wrapping_mul(9) / 10 + ((next % 100) as i64).wrapping_mul(amount) / 1000;
    let before = target.hp;
    target.hp += rolled;
    if target.hp > target.max_hp {
        target.hp = target.max_hp;
    }
    Ok((NativeCommandRestore { rolled, actual: target.hp - before }, next))
}

/// NativeCommandClearRestoreResult records one final target in the common
/// ID20/21 route. Cleared identifies the raw +0x25/+0x26 flag; restore is
/// present only if that flag was nonzero.
///
/// Go：`native_command20.go:54`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommandClearRestoreResult {
    pub target: Option<UnitIdx>,
    pub offset: i64,
    pub cleared: bool,
    pub restore: NativeCommandRestore,
}

impl State {
    /// ExecuteNativeCommandClearRestore mirrors 0x22A85/0x22BC6 -> 0x22AA8 ->
    /// 0x22AF6 for IDs20/21. The command's own record supplies target/MP data;
    /// native 0x1C916 is deliberately invoked with record10's damage field, then
    /// the nonzero target raw interval is cleared. No named status is inferred.
    ///
    /// Go：`native_command20.go:65`，`func (s *State) ExecuteNativeCommandClearRestore(actor, confirmed *Unit, commandID int, rng *rand.Rand) ([]NativeCommandClearRestoreResult, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command_clear_restore(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
        rng: Option<&mut dyn GoRand>,
    ) -> Result<Vec<NativeCommandClearRestoreResult>> {
        let Some(rng) = rng else {
            return Err(battle_err!("missing native command clear/restore state/rng"));
        };
        let (targets, offset) = self.native_command_clear_restore_targets(actor, confirmed, command_id)?;
        let record = self.native_command_book[command_id as usize];
        if !spend_native_command_mp(actor.map(|i| &mut self.units[i]), record.mp_cost) {
            return Err(battle_err!("native command clear/restore insufficient MP"));
        }
        let amount = self.native_command_book[10].damage;
        let mut results = Vec::with_capacity(targets.len());
        for target in targets {
            let mut result = NativeCommandClearRestoreResult { target: Some(target), offset, ..Default::default() };
            let (duration, _) = self.units[target].native_transient_duration(offset);
            if duration != 0 {
                // Go 忽略這裡的錯誤（`restore, _ :=`）；預檢已排除 HP 不合法的目標。
                let restore = apply_native_command_restore(Some(&mut self.units[target]), amount, Some(&mut *rng))
                    .unwrap_or_default();
                self.units[target].set_native_transient_duration(offset, 0);
                (result.cleared, result.restore) = (true, restore);
            }
            results.push(result);
        }
        if let Some(actor) = actor {
            self.units[actor].acted = true;
        }
        Ok(results)
    }

    /// NativeCommandClearRestoreTargets performs the complete non-mutating
    /// transaction preflight needed before the player-only 0x1D6C8 presentation.
    ///
    /// Go：`native_command20.go:94`，`func (s *State) NativeCommandClearRestoreTargets(actor, confirmed *Unit, commandID int) ([]*Unit, int, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command_clear_restore_targets(
        &self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
        command_id: i64,
    ) -> Result<(Vec<UnitIdx>, i64)> {
        let Some(actor_idx) = actor else {
            return Err(battle_err!("missing native command clear/restore state/actor"));
        };
        let offset = match command_id {
            20 => 0x25,
            21 => 0x26,
            _ => return Err(battle_err!("native command clear/restore unavailable id={command_id}")),
        };
        if self.native_command_book.len() != 36
            || self.native_command_book[command_id as usize].id != command_id
            || self.native_command_book[10].id != 10
        {
            return Err(battle_err!("native command clear/restore records unavailable id={command_id}"));
        }
        let record = self.native_command_book[command_id as usize];
        if record.mp_cost < 0 || self.units[actor_idx].mp < record.mp_cost || self.native_command_book[10].damage < 0 {
            return Err(battle_err!("native command clear/restore cost/amount unavailable"));
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
        for &target in &targets {
            let t = &self.units[target];
            if t.hp < 0 || t.max_hp < 0 || t.hp > t.max_hp {
                return Err(battle_err!("invalid native command clear/restore target state"));
            }
        }
        Ok((targets, offset))
    }
}

#[cfg(test)]
mod native_command20_test;
