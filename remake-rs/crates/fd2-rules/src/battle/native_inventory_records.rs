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
    if count < 0 || count > units.len() as i64 || count > 16 {
        return Err(battle_err!(
            "native inventory records: count {count} outside 0..{}",
            min_int(units.len() as i64, 16)
        ));
    }
    let mut records = vec![0u8; (count * NATIVE_RECORD_SIZE) as usize];
    for (i, unit) in units.iter().enumerate().take(count as usize) {
        if unit.inventory_slots.len() as i64 != NATIVE_INVENTORY_CELLS
            || unit.native_inventory_flags.len() as i64 != NATIVE_INVENTORY_CELLS
        {
            return Err(battle_err!("native inventory records: unit {i} lacks raw inventory provenance"));
        }
        let base = i * NATIVE_RECORD_SIZE as usize;
        for slot in 0..NATIVE_INVENTORY_CELLS as usize {
            let (item, flag) = (unit.inventory_slots[slot], unit.native_inventory_flags[slot]);
            if !(0..=0xff).contains(&item) || !(0..=0xff).contains(&flag) {
                return Err(battle_err!("native inventory records: unit {i} cell {slot} malformed"));
            }
            records[base + 0x0a + slot * 2] = flag as u8;
            records[base + 0x0b + slot * 2] = item as u8;
        }
    }
    Ok(records)
}

/// Go：`native_inventory_records.go:33`，`func minInt(a, b int) int`
pub(crate) fn min_int(a: i64, b: i64) -> i64 {
    if a < b { a } else { b }
}

#[cfg(test)]
mod native_inventory_records_test;
