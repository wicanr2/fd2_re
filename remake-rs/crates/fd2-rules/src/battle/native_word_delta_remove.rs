//! Go 來源：`remake/internal/battle/native_word_delta_remove.go`（37 行）。

use super::*;

/// ApplyNativeWordDeltaAndRemove reproduces the mutation portion of 0x21082:
/// add a signed caller-supplied delta to a raw unit word, then invoke the
/// proven 0x1b8e7 inventory-slot removal on the caller's unit/slot. Types
/// 8/9/0xa identify offsets +0x37/+0x39/+0x3e as base AP/DP/DX; other callers
/// must retain their own evidence boundary. Native renderer/recalculation
/// callbacks remain outside this adapter.
///
/// Go：`native_word_delta_remove.go:14`，`func ApplyNativeWordDeltaAndRemove(records []byte, targetUnit, wordOffset int, delta int, removalUnit, removalSlot int) (uint16, error)`
pub fn apply_native_word_delta_and_remove(
    records: &mut [u8],
    target_unit: i64,
    word_offset: i64,
    delta: i64,
    removal_unit: i64,
    removal_slot: i64,
) -> Result<u16> {
    todo!("battle/native_word_delta_remove.go:ApplyNativeWordDeltaAndRemove")
}
