//! Go 來源：`remake/internal/battle/native_inventory_assign.go`（28 行）。

use super::*;

/// Go：`native_inventory_assign.go:5`，`func recordBoundsError(index int) error`
pub(crate) fn record_bounds_error(index: i64) -> BattleError {
    battle_err!("native inventory record {index} is out of bounds")
}

/// AssignNativeReservedItem reproduces 0x1bb8c(unit,item): it scans eight
/// cells at record+0x0a, consumes the first cell whose flag byte has bit7 set,
/// clears that flag, and writes the caller-supplied item byte. The returned
/// bool is the native 1/-1 success result.
///
/// Go：`native_inventory_assign.go:13`，`func AssignNativeReservedItem(records []byte, unitIndex int, itemID byte) (bool, error)`
pub fn assign_native_reserved_item(records: &mut [u8], unit_index: i64, item_id: u8) -> Result<bool> {
    let base = unit_index.wrapping_mul(NATIVE_RECORD_SIZE);
    if unit_index < 0 || base < 0 || base.wrapping_add(0x1a) > records.len() as i64 {
        return Err(record_bounds_error(unit_index));
    }
    for slot in 0..8 {
        let offset = (base + 0x0a + slot * 2) as usize;
        if records[offset] & 0x80 == 0 {
            continue;
        }
        records[offset] = 0;
        records[offset + 1] = item_id;
        return Ok(true);
    }
    Ok(false)
}

#[cfg(test)]
mod native_inventory_assign_test;
