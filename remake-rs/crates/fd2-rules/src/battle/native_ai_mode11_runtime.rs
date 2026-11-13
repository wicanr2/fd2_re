//! Go 來源：`remake/internal/battle/native_ai_mode11_runtime.go`（164 行）。

use super::*;

impl State {
    /// nextNativeAIMode11Plan lowers the direct 0x13A9F mode-11 branch without
    /// routing it through 0x14EF0.  The two producers are evaluated before any
    /// action is committed, then their raw gate order is preserved in
    /// NativeMode11Stages.  A missing producer/table/record is an error rather
    /// than permission to use the normalized planner.
    ///
    /// Go：`native_ai_mode11_runtime.go:10`，`func (s *State) nextNativeAIMode11Plan(u *Unit) (*AIPlan, bool, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn next_native_ai_mode11_plan(&mut self, u: Option<UnitIdx>) -> Result<(Option<AIPlan>, bool)> {
        todo!("battle/native_ai_mode11_runtime.go:State.nextNativeAIMode11Plan")
    }
}

/// nativeAIPhysicalGate is the raw [0x53C4F] threshold producer.  The helper
/// keeps the no-candidate value below the signed >=6 gate; it never invents a
/// target when 0x14237 did not return one.
///
/// Go：`native_ai_mode11_runtime.go:114`，`func nativeAIPhysicalGate(selection NativePhysicalAttackSelection, ok bool) int`
pub(crate) fn native_ai_physical_gate(selection: NativePhysicalAttackSelection, ok: bool) -> i64 {
    todo!("battle/native_ai_mode11_runtime.go:nativeAIPhysicalGate")
}

impl State {
    /// nativeAIMode11BlockedOrRecovery is the direct 0x14121 -> 0x13FD4 edge.  A
    /// found raw blocked cell becomes movement-only.  If no cell is found, the
    /// raw HP/gate decision is retained; an unaccepted decision is a legitimate
    /// common-tail no-op, not a normalized wait or attack.
    ///
    /// Go：`native_ai_mode11_runtime.go:125`，`func (s *State) nativeAIMode11BlockedOrRecovery( u *Unit, actor, selector int, records, actorRecord, baseFlags, costRow []byte, ) (*AIPlan, *NativeAIIdleRecoveryDecision, error)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn native_ai_mode11_blocked_or_recovery(
        &mut self,
        u: Option<UnitIdx>,
        actor: i64,
        selector: i64,
        records: &[u8],
        actor_record: &[u8],
        base_flags: &mut [u8],
        cost_row: &mut [u8],
    ) -> Result<(Option<AIPlan>, Option<NativeAIIdleRecoveryDecision>)> {
        todo!("battle/native_ai_mode11_runtime.go:State.nativeAIMode11BlockedOrRecovery")
    }
}
