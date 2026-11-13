//! Go 來源：`remake/internal/battle/native_equipped_item.go`（25 行）。

use super::*;

/// NativeEquippedInventorySlot reproduces 0x1b83d(unit,a2). It scans runtime
/// cells in order, requires flag bit 0x40, then selects item bytes below 0x80
/// when a2==0 and bytes >=0x80 when a2!=0. The return value is the raw runtime
/// slot, or -1 for native not-found; no weapon/effect name is inferred.
///
/// Go：`native_equipped_item.go:9`，`func NativeEquippedInventorySlot(flags, slots []int, a2 int) (int, error)`
pub fn native_equipped_inventory_slot(flags: &[i64], slots: &[i64], a2: i64) -> Result<i64> {
    if flags.len() as i64 != NATIVE_INVENTORY_CELLS || slots.len() as i64 != NATIVE_INVENTORY_CELLS {
        return Err(battle_err!("native equipped item: need {} flags and slots", NATIVE_INVENTORY_CELLS));
    }
    for slot in 0..NATIVE_INVENTORY_CELLS as usize {
        if !(0..=0xff).contains(&flags[slot]) || !(0..=0xff).contains(&slots[slot]) {
            return Err(battle_err!("native equipped item: malformed cell {slot}"));
        }
        if flags[slot] & 0x40 == 0 {
            continue;
        }
        if (a2 == 0 && slots[slot] < 0x80) || (a2 != 0 && slots[slot] >= 0x80) {
            return Ok(slot as i64);
        }
    }
    Ok(-1)
}

#[cfg(test)]
mod native_equipped_item_test;
