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
    let step = |callee: u32, command_id: i64, marker_offset: i64, amount: i64| NativeCompoundStep {
        callee,
        command_id,
        marker_offset,
        amount,
    };
    let steps = match command_id {
        32 => vec![step(0x2111a, 32, 0, 0)],
        33 => vec![
            // Callee==0 denotes the wrapper's direct byte clear, not a call.
            step(0, 33, 0x25, 0),
            step(0, 33, 0x26, 0),
            step(0, 33, 0x27, 0),
            step(0x211a4, 33, 0, 0x320),
        ],
        34 => vec![step(0x22721, 17, 0x22, 0), step(0x22866, 18, 0x23, 0), step(0x22997, 19, 0x24, 0)],
        35 => vec![step(0x22d1b, 26, 0x25, 0), step(0x22d1b, 22, 0x27, 0), step(0x22d1b, 27, 0x26, 0)],
        _ => return (Vec::new(), false),
    };
    (steps, true)
}

#[cfg(test)]
mod native_command_compound_test;
