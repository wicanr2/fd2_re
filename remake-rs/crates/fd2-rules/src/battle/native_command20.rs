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
    todo!("battle/native_command20.go:ApplyNativeCommandRestore")
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
    todo!("battle/native_command20.go:ApplyNativeCommandRestoreNative")
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
        todo!("battle/native_command20.go:State.ExecuteNativeCommandClearRestore")
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
        todo!("battle/native_command20.go:State.NativeCommandClearRestoreTargets")
    }
}
