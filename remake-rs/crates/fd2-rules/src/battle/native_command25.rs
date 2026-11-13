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
        todo!("battle/native_command25.go:State.ExecuteNativeCommand25")
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
        todo!("battle/native_command25.go:State.NativeCommand25Targets")
    }
}
