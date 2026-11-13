//! Go 來源：`remake/internal/battle/native_inventory_assign.go`（28 行）。

use super::*;

/// Go：`native_inventory_assign.go:5`，`func recordBoundsError(index int) error`
pub(crate) fn record_bounds_error(index: i64) -> Result<()> {
    todo!("battle/native_inventory_assign.go:recordBoundsError")
}

/// AssignNativeReservedItem reproduces 0x1bb8c(unit,item): it scans eight
/// cells at record+0x0a, consumes the first cell whose flag byte has bit7 set,
/// clears that flag, and writes the caller-supplied item byte. The returned
/// bool is the native 1/-1 success result.
///
/// Go：`native_inventory_assign.go:13`，`func AssignNativeReservedItem(records []byte, unitIndex int, itemID byte) (bool, error)`
pub fn assign_native_reserved_item(records: &mut [u8], unit_index: i64, item_id: u8) -> Result<bool> {
    todo!("battle/native_inventory_assign.go:AssignNativeReservedItem")
}
