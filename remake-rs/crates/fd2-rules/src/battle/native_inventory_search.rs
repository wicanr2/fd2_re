//! Go 來源：`remake/internal/battle/native_inventory_search.go`（46 行）。

use super::*;

/// FindNativeInventoryItemInUnit reproduces the raw search used by 0x31860.
/// Native inventory cells are ordered as eight two-byte cells at record+0x0a;
/// 0x31860 first obtains the occupied count from 0x1b8a6 and then compares the
/// first count-sized raw slots' item bytes at +0x0b. 0x1b8a6 does not verify
/// prefix compactness. The flag/item bytes remain opaque; this helper does not
/// consume or mutate a cell.
///
/// Go：`native_inventory_search.go:11`，`func FindNativeInventoryItemInUnit(records []byte, unitIndex int, item byte) (int, error)`
pub fn find_native_inventory_item_in_unit(records: &[u8], unit_index: i64, item: u8) -> Result<i64> {
    let base = unit_index.wrapping_mul(NATIVE_RECORD_SIZE);
    if unit_index < 0 || base < 0 || base.wrapping_add(0x1a) > records.len() as i64 {
        return Err(record_bounds_error(unit_index));
    }
    let count = native_inventory_occupied_count(records, unit_index)?;
    for slot in 0..count {
        if records[(base + 0x0b + slot * 2) as usize] == item {
            return Ok(slot);
        }
    }
    Ok(-1)
}

/// FindNativeInventoryItem reproduces 0x24b14(item): scan runtime units 0..15
/// and return the first matching unit/slot.  A missing item returns (-1,-1),
/// matching the native -1 return while preserving the useful raw location for
/// the successful editable gate.  The search is read-only and fail-closed.
///
/// Go：`native_inventory_search.go:32`，`func FindNativeInventoryItem(records []byte, item byte) (unitIndex, slot int, err error)`
/// Go 具名回傳值依序為：unitIndex、slot、err。
pub fn find_native_inventory_item(records: &[u8], item: u8) -> Result<(i64, i64)> {
    if (records.len() as i64) < 16 * NATIVE_RECORD_SIZE {
        return Err(battle_err!("native inventory search needs 16 records, got {} bytes", records.len()));
    }
    for unit in 0..16 {
        let found = find_native_inventory_item_in_unit(records, unit, item)?;
        if found >= 0 {
            return Ok((unit, found));
        }
    }
    Ok((-1, -1))
}

#[cfg(test)]
mod native_inventory_search_test;
