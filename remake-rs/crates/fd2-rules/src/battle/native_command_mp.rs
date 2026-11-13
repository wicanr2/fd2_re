//! Go 來源：`remake/internal/battle/native_command_mp.go`（18 行）。

use super::*;

/// SpendNativeCommandMP mirrors the successful post-confirm portion of
/// 0x1CA89: subtract command record byte+5 from runtime unit+0x44 (current
/// MP).  The native selector has already checked current MP >= byte+5 before
/// reaching this helper; return false and leave state unchanged if that
/// precondition is absent in the remake caller.
///
/// This intentionally takes the raw command-record cost, not a legacy Spell:
/// the two tables are byte-identical only for the verified IDs 0..35, while
/// their normalized gameplay semantics are not interchangeable.
///
/// Go：`native_command_mp.go:12`，`func SpendNativeCommandMP(unit *Unit, cost int) bool`
pub fn spend_native_command_mp(unit: Option<&mut Unit>, cost: i64) -> bool {
    todo!("battle/native_command_mp.go:SpendNativeCommandMP")
}
