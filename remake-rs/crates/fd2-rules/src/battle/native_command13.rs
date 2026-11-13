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
        todo!("battle/native_command13.go:State.NativeCommandHealTargets")
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
        todo!("battle/native_command13.go:State.NativeAICommandHealTargets")
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
        todo!("battle/native_command13.go:State.ExecuteNativeCommandHeal")
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
        todo!("battle/native_command13.go:State.ExecuteNativeAICommandHeal")
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
        todo!("battle/native_command13.go:State.ExecuteNativeCommandHealNative")
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
        todo!("battle/native_command13.go:State.ExecuteNativeAICommandHealNative")
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
        todo!("battle/native_command13.go:State.executeNativeCommandHealTargetsNative")
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
        todo!("battle/native_command13.go:State.executeNativeCommandHealTargets")
    }
}
