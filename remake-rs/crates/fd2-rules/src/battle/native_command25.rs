//! Go 來源：`remake/internal/battle/native_command25.go`（72 行）。

use super::*;

/// NativeCommand25Result is one final-effect target in the recovered ID25
/// route. Cleared reports the direct unit+5 bit0x80 mutation performed by
/// 0x22C04; it does not assign a gameplay name to the command.
///
/// Go：`native_command25.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCommand25Result {
    pub target: Option<UnitIdx>,
    pub cleared: bool,
}

impl State {
    /// ExecuteNativeCommand25 mirrors the verified state portion of
    /// 0x22C04: generic 0x1CFF0 two-stage targets, record-25 MP debit, then clear
    /// unit+5 bit0x80 only when it was already set. The wrapper's successful
    /// command completion still consumes the actor's action after the effect.
    /// Renderer, message feedback and UI remain outside this fail-closed slice.
    ///
    /// Go：`native_command25.go:18`，`func (s *State) ExecuteNativeCommand25(actor, confirmed *Unit) ([]NativeCommand25Result, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn execute_native_command25(
        &mut self,
        actor: Option<UnitIdx>,
        confirmed: Option<UnitIdx>,
    ) -> Result<Vec<NativeCommand25Result>> {
        const COMMAND_ID: usize = 25;
        let targets = self.native_command25_targets(actor, confirmed)?;
        let record = self.native_command_book[COMMAND_ID];
        if !spend_native_command_mp(actor.map(|i| &mut self.units[i]), record.mp_cost) {
            return Err(battle_err!("native command 25 insufficient MP"));
        }
        let mut results = Vec::with_capacity(targets.len());
        for target in targets {
            let unit = &mut self.units[target];
            let cleared = unit.native_record_byte5 & 0x80 != 0;
            if cleared {
                unit.native_record_byte5 &= !0x80;
            }
            results.push(NativeCommand25Result { target: Some(target), cleared });
        }
        // 0x18D8C sets the invoking actor's completion bit only after the handler
        // returns success. It is independent from the target bit that ID25 clears.
        if let Some(actor) = actor {
            self.units[actor].acted = true;
        }
        Ok(results)
    }

    /// NativeCommand25Targets performs the complete non-mutating preflight before
    /// the player-only 0x1D6C8 presentation. Target raw +5 provenance is mandatory:
    /// its bit7 must never be projected from or written through Unit.Acted.
    ///
    /// Go：`native_command25.go:45`，`func (s *State) NativeCommand25Targets(actor, confirmed *Unit) ([]*Unit, error)`
    /// 所有權：`actor`、`confirmed` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// 所有權：回傳的單位是 `self.units` 的索引。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_command25_targets(&self, actor: Option<UnitIdx>, confirmed: Option<UnitIdx>) -> Result<Vec<UnitIdx>> {
        const COMMAND_ID: i64 = 25;
        let Some(actor_idx) = actor.filter(|_| {
            self.native_command_book.len() == 36 && self.native_command_book[COMMAND_ID as usize].id == COMMAND_ID
        }) else {
            return Err(battle_err!("native command 25 record unavailable"));
        };
        let record = self.native_command_book[COMMAND_ID as usize];
        if record.mp_cost < 0 || self.units[actor_idx].mp < record.mp_cost {
            return Err(battle_err!("native command 25 insufficient MP"));
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
        if targets.iter().any(|&target| !self.units[target].has_native_record_byte5) {
            return Err(battle_err!("native command 25 target lacks raw +5 provenance"));
        }
        Ok(targets)
    }
}

#[cfg(test)]
mod native_command25_test;
