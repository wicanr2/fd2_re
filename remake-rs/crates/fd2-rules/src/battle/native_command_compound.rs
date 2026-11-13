//! Go 來源：`remake/internal/battle/native_command_compound.go`（46 行）。

use super::*;

/// NativeCompoundStep is an evidence-only step in the 0x27fc9 wrapper.
/// Addresses and raw offsets are retained deliberately; this plan does not
/// execute a callee or assign a gameplay/effect name.
///
/// Go：`native_command_compound.go:6`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeCompoundStep {
    pub callee: u32,
    pub command_id: i64,
    pub marker_offset: i64,
    pub amount: i64,
}

/// NativeCompoundCommandPlan returns the verified helper order for IDs 32..35.
/// It remains a data-only evidence bridge: bounded player transactions for
/// IDs33..35 consume it or the same order elsewhere, while target selection,
/// presentation and mutation stay outside this plan itself.
///
/// Go：`native_command_compound.go:17`，`func NativeCompoundCommandPlan(commandID int) ([]NativeCompoundStep, bool)`
pub fn native_compound_command_plan(command_id: i64) -> (Vec<NativeCompoundStep>, bool) {
    todo!("battle/native_command_compound.go:NativeCompoundCommandPlan")
}
