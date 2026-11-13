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
    let base = target_unit.wrapping_mul(NATIVE_RECORD_SIZE);
    if target_unit < 0 || base < 0 || base.wrapping_add(NATIVE_RECORD_SIZE) > records.len() as i64 {
        return Err(record_bounds_error(target_unit));
    }
    if word_offset < 0 || word_offset.wrapping_add(2) > NATIVE_RECORD_SIZE {
        return Err(battle_err!("native word offset {} is out of bounds", go_sharp_x(word_offset)));
    }
    let removal_base = removal_unit.wrapping_mul(NATIVE_RECORD_SIZE);
    if removal_unit < 0 || removal_base < 0 || removal_base.wrapping_add(0x1a) > records.len() as i64 {
        return Err(record_bounds_error(removal_unit));
    }
    if !(0..8).contains(&removal_slot) {
        return Err(battle_err!("native inventory slot {removal_slot} is out of bounds"));
    }
    let offset = (base + word_offset) as usize;
    let value = u16::from_le_bytes([records[offset], records[offset + 1]]).wrapping_add(delta as i16 as u16);
    records[offset..offset + 2].copy_from_slice(&value.to_le_bytes());
    remove_native_inventory_slot(records, removal_unit, removal_slot)?;
    Ok(value)
}

#[cfg(test)]
mod native_word_delta_remove_test;
