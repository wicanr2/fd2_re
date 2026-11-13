//! Go 來源：`remake/internal/battle/native_inventory_remove.go`（21 行）。

use super::*;

/// RemoveNativeInventorySlot reproduces IDA's 0x1b8e7(unit, slot): the eight
/// raw two-byte cells begin at record+0x0a.  It shifts later cells left with
/// memmove semantics, then writes 0x80 to the final cell's flag byte.  Item
/// bytes remain opaque, including the stale final-cell byte left by native.
///
/// Go：`native_inventory_remove.go:9`，`func RemoveNativeInventorySlot(records []byte, unitIndex, slot int) error`
pub fn remove_native_inventory_slot(records: &mut [u8], unit_index: i64, slot: i64) -> Result<()> {
    todo!("battle/native_inventory_remove.go:RemoveNativeInventorySlot")
}
