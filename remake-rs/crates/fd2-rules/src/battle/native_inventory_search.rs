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
    todo!("battle/native_inventory_search.go:FindNativeInventoryItemInUnit")
}

/// FindNativeInventoryItem reproduces 0x24b14(item): scan runtime units 0..15
/// and return the first matching unit/slot.  A missing item returns (-1,-1),
/// matching the native -1 return while preserving the useful raw location for
/// the successful editable gate.  The search is read-only and fail-closed.
///
/// Go：`native_inventory_search.go:32`，`func FindNativeInventoryItem(records []byte, item byte) (unitIndex, slot int, err error)`
/// Go 具名回傳值依序為：unitIndex、slot、err。
pub fn find_native_inventory_item(records: &[u8], item: u8) -> Result<(i64, i64)> {
    todo!("battle/native_inventory_search.go:FindNativeInventoryItem")
}
