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
    if source.len() as i64 != NATIVE_INVENTORY_CELLS {
        return Err(battle_err!(
            "native inventory flags: source length={}, want {}",
            source.len(),
            NATIVE_INVENTORY_CELLS
        ));
    }
    for (i, &value) in source.iter().enumerate() {
        if !(0..=0xff).contains(&value) {
            return Err(battle_err!("native inventory flags: source[{i}]={value} outside byte"));
        }
    }
    let mut flags = vec![0i64; NATIVE_INVENTORY_CELLS as usize];
    flags[0] = NATIVE_EQUIPPED_MASK;
    flags[1] = if source[0] == 0xff { 0x80 } else { NATIVE_EQUIPPED_MASK };
    for i in 2..NATIVE_INVENTORY_CELLS as usize {
        if source[i] == 0xff {
            flags[i] = 0x80;
        }
    }
    Ok(flags)
}

/// NativeInventoryCompactEligible maps one compact item index to its runtime
/// cell and applies the native signed-flag gate (flag byte signed >= 0). It rejects missing
/// or misaligned provenance instead of inferring a flag from the item ID.
///
/// Go：`native_inventory_flags.go:39`，`func NativeInventoryCompactEligible(flags, runtimeSlots []int, compactIndex int) (bool, error)`
pub fn native_inventory_compact_eligible(flags: &[i64], runtime_slots: &[i64], compact_index: i64) -> Result<bool> {
    if flags.len() as i64 != NATIVE_INVENTORY_CELLS || runtime_slots.len() as i64 != NATIVE_INVENTORY_CELLS {
        return Err(battle_err!("native inventory flags: need {} flags and slots", NATIVE_INVENTORY_CELLS));
    }
    if compact_index < 0 {
        return Err(battle_err!("native inventory flags: negative compact index"));
    }
    let mut seen = 0;
    for (slot, &item) in runtime_slots.iter().enumerate() {
        if item == 0xff {
            continue;
        }
        if !(0..=0xff).contains(&item) || !(0..=0xff).contains(&flags[slot]) {
            return Err(battle_err!("native inventory flags: malformed cell {slot}"));
        }
        if seen == compact_index {
            return Ok(flags[slot] & 0x80 == 0);
        }
        seen += 1;
    }
    Err(battle_err!("native inventory flags: compact index {compact_index} absent"))
}

#[cfg(test)]
mod native_inventory_flags_test;
