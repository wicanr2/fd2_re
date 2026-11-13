//! Go 來源：`remake/internal/battle/native_inventory_records.go`（38 行）。

use super::*;

/// NativeInventoryRecords materializes only the proven inventory cells of a
/// runtime roster. The surrounding 0x50-byte records stay zeroed because this
/// adapter is intentionally scoped to 0x24b14/0x31860's +0x0a inventory ABI.
/// It fails closed when any requested unit lacks the source slots/flags needed
/// to reproduce the native prefix gate.
///
/// Go：`native_inventory_records.go:10`，`func NativeInventoryRecords(units []*Unit, count int) ([]byte, error)`
pub fn native_inventory_records(units: &[Unit], count: i64) -> Result<Vec<u8>> {
    todo!("battle/native_inventory_records.go:NativeInventoryRecords")
}

/// Go：`native_inventory_records.go:33`，`func minInt(a, b int) int`
pub(crate) fn min_int(a: i64, b: i64) -> i64 {
    todo!("battle/native_inventory_records.go:minInt")
}
