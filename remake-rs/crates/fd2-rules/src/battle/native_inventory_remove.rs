//! Go 來源：`remake/internal/battle/native_inventory_remove.go`（21 行）。

use super::*;

/// RemoveNativeInventorySlot reproduces IDA's 0x1b8e7(unit, slot): the eight
/// raw two-byte cells begin at record+0x0a.  It shifts later cells left with
/// memmove semantics, then writes 0x80 to the final cell's flag byte.  Item
/// bytes remain opaque, including the stale final-cell byte left by native.
///
/// Go：`native_inventory_remove.go:9`，`func RemoveNativeInventorySlot(records []byte, unitIndex, slot int) error`
pub fn remove_native_inventory_slot(records: &mut [u8], unit_index: i64, slot: i64) -> Result<()> {
    let base = unit_index.wrapping_mul(NATIVE_RECORD_SIZE);
    if unit_index < 0 || base < 0 || base.wrapping_add(0x1a) > records.len() as i64 {
        return Err(record_bounds_error(unit_index));
    }
    if !(0..8).contains(&slot) {
        return Err(battle_err!("native inventory slot {slot} is out of bounds"));
    }
    let start = (base + 0x0a + slot * 2) as usize;
    let n = (2 * (7 - slot)) as usize;
    records.copy_within(start + 2..start + 2 + n, start);
    records[base as usize + 0x18] = 0x80;
    Ok(())
}

#[cfg(test)]
mod native_inventory_remove_test;
