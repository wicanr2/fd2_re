//! Go 來源：`remake/internal/battle/native_inventory_flags.go`（60 行）。

use super::*;

/// NativeInventoryFlagsFromSource reproduces the constructor writes at
/// 0x10c50 for the eight FDFIELD inventory source bytes.  The first two cells
/// are a special pair: a 0xff first byte collapses the second source item into
/// cell 0 and reserves cell 1; otherwise both cells receive the equipped flag.
/// Cells 2..7 are unequipped when their source item is present and reserved
/// (bit 7 set) when it is 0xff.  The returned values are raw flag bytes, not
/// normalized equipped booleans.
///
/// Go：`native_inventory_flags.go:12`，`func NativeInventoryFlagsFromSource(source []int) ([]int, error)`
pub fn native_inventory_flags_from_source(source: &[i64]) -> Result<Vec<i64>> {
    todo!("battle/native_inventory_flags.go:NativeInventoryFlagsFromSource")
}

/// NativeInventoryCompactEligible maps one compact item index to its runtime
/// cell and applies the native signed-flag gate (flag byte signed >= 0). It rejects missing
/// or misaligned provenance instead of inferring a flag from the item ID.
///
/// Go：`native_inventory_flags.go:39`，`func NativeInventoryCompactEligible(flags, runtimeSlots []int, compactIndex int) (bool, error)`
pub fn native_inventory_compact_eligible(flags: &[i64], runtime_slots: &[i64], compact_index: i64) -> Result<bool> {
    todo!("battle/native_inventory_flags.go:NativeInventoryCompactEligible")
}
