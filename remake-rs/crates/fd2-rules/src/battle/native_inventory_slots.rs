//! Go 來源：`remake/internal/battle/native_inventory_slots.go`（23 行）。

use super::*;

/// NativeInventoryOccupiedCount reproduces 0x1b8a6: eight two-byte inventory
/// cells begin at record+0x0a, and the native count increments when a cell's
/// flag byte bit7 is clear. Some callers then scan the first count-sized raw
/// slots, but 0x1b8a6 itself does not verify that occupied cells form a prefix.
/// Bit7 set is the reserved empty state used by 0x1bb8c. Item bytes and
/// higher-level item meaning stay opaque.
///
/// Go：`native_inventory_slots.go:11`，`func NativeInventoryOccupiedCount(records []byte, unitIndex int) (int, error)`
pub fn native_inventory_occupied_count(records: &[u8], unit_index: i64) -> Result<i64> {
    todo!("battle/native_inventory_slots.go:NativeInventoryOccupiedCount")
}
