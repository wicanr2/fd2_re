//! Go 來源：`remake/internal/battle/native_command_availability.go`（50 行）。

use super::*;

/// NativeCommandAvailable is the narrow selector gate recovered from
/// 0x159fa: the command bit must exist in unit+0x1a..+0x1e, its verified
/// 0x4e516 record must be present, and record+5 (MP cost) must be <= runtime
/// unit+0x44. It deliberately does not include the action-direction +0x27
/// gate, target geometry, or any command/status name.
///
/// Go：`native_command_availability.go:8`，`func NativeCommandAvailable(unit *Unit, book []NativeCommandRecord, commandID int) bool`
pub fn native_command_available(unit: Option<&Unit>, book: &[NativeCommandRecord], command_id: i64) -> bool {
    let Some(unit) = unit else {
        return false;
    };
    if !(0..36).contains(&command_id) || book.len() != 36 || book[command_id as usize].id != command_id {
        return false;
    }
    if unit.native_command_mask[(command_id / 8) as usize] & (1 << (command_id % 8)) == 0 {
        return false;
    }
    let record = &book[command_id as usize];
    record.mp_cost >= 0 && record.mp_cost <= unit.mp
}

/// NativeAvailableCommandIDs returns only IDs with a closed command record and
/// sufficient current MP. Unknown physical bits 36..39 are omitted rather
/// than promoted to executable commands.
///
/// Go：`native_command_availability.go:21`，`func NativeAvailableCommandIDs(unit *Unit, book []NativeCommandRecord) []int`
pub fn native_available_command_ids(unit: Option<&Unit>, book: &[NativeCommandRecord]) -> Vec<i64> {
    if unit.is_none() || book.len() != 36 {
        return Vec::new();
    }
    (0..36).filter(|&id| native_command_available(unit, book, id)).collect()
}

/// NativeAvailableAICommandIDs adds the 0x1598a dispatcher gate to the
/// bounded command scan: raw unit+0x27 must be zero before command bytes are
/// enumerated. Unknown physical IDs 36..39 remain omitted.
///
/// Go：`native_command_availability.go:37`，`func NativeAvailableAICommandIDs(unit *Unit, book []NativeCommandRecord) []int`
pub fn native_available_ai_command_ids(unit: Option<&Unit>, book: &[NativeCommandRecord]) -> Vec<i64> {
    match unit {
        Some(u) if u.native_transient[5] == 0 => native_available_command_ids(unit, book),
        _ => Vec::new(),
    }
}

/// NativeAvailableAIScoredCommandIDs is the bounded command-index list consumed
/// directly by 0x1598a.  0x1c269 returns set-bit indices and 0x1598a passes each
/// index unchanged to both 0x4e516 and 0x15b77.  It must therefore retain IDs
/// below 0x10; command-0x10 belongs to the separate 0x1567e item-command path.
///
/// Go：`native_command_availability.go:48`，`func NativeAvailableAIScoredCommandIDs(unit *Unit, book []NativeCommandRecord) []int`
pub fn native_available_ai_scored_command_ids(unit: Option<&Unit>, book: &[NativeCommandRecord]) -> Vec<i64> {
    native_available_ai_command_ids(unit, book)
}

#[cfg(test)]
mod native_command_availability_test;
