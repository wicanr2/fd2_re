//! Go 來源：`remake/internal/battle/native_inventory_available.go`（23 行）。

use super::*;

/// NativeInventoryAvailableCount is the Unit-facing form of 0x1b8a6. It
/// counts the eight raw flags whose sign bit is clear, independently of item
/// bytes. Callers must supply the constructor's complete eight-byte flag view;
/// a compact inventory length is not an equivalent substitute.
///
/// Go：`native_inventory_available.go:9`，`func NativeInventoryAvailableCount(flags []int) (int, error)`
pub fn native_inventory_available_count(flags: &[i64]) -> Result<i64> {
    todo!("battle/native_inventory_available.go:NativeInventoryAvailableCount")
}
