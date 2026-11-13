//! Go 來源：`remake/internal/battle/native_item_mp_restore.go`（82 行）。

use super::*;

/// NativeItemMPRestoreRoute is the closed type-11 branch in 0x20c6f.
/// Row word +0x0e is passed to 0x1c9dd, and the source slot is consumed after
/// the target loop.
///
/// Go：`native_item_mp_restore.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemMPRestoreRoute {
    pub item_type: u8,
    pub amount: u16,
}

/// Go：`native_item_mp_restore.go:16`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeItemMPRestoreBatch {
    pub results: Vec<NativeRawRestoreResult>,
    pub skipped_targets: Vec<i64>,
    pub rng_state: u16,
    pub score: i64,
}

/// Go：`native_item_mp_restore.go:23`，`func NativeItemMPRestoreRouteForType(itemType byte, amount uint16) (NativeItemMPRestoreRoute, bool)`
pub fn native_item_mp_restore_route_for_type(item_type: u8, amount: u16) -> (NativeItemMPRestoreRoute, bool) {
    if item_type != 11 {
        return (NativeItemMPRestoreRoute::default(), false);
    }
    (NativeItemMPRestoreRoute { item_type, amount }, true)
}

/// ApplyNativeItemMPRestore reproduces the mutation transaction of the type-11
/// branch without presentation callbacks. A target whose max-MP word +0x46 is
/// zero is skipped without advancing RNG. Valid targets use 0x1c9dd in list
/// order, then 0x1b8e7 consumes the source slot.
///
/// Go：`native_item_mp_restore.go:34`，`func ApplyNativeItemMPRestore( records []byte, targetIndices []byte, route NativeItemMPRestoreRoute, rngState uint16, sourceUnit, sourceSlot int, ) (NativeItemMPRestoreBatch, error)`
pub fn apply_native_item_mp_restore(
    records: &mut [u8],
    target_indices: &[u8],
    route: NativeItemMPRestoreRoute,
    rng_state: u16,
    source_unit: i64,
    source_slot: i64,
) -> Result<NativeItemMPRestoreBatch> {
    let (verified, ok) = native_item_mp_restore_route_for_type(route.item_type, route.amount);
    if !ok || verified != route {
        return Err(battle_err!("native item MP restore route is invalid: {}", route.go_sharp_v()));
    }
    for &raw_index in target_indices {
        if raw_index as i64 >= records.len() as i64 / NATIVE_RECORD_SIZE {
            return Err(record_bounds_error(raw_index as i64));
        }
    }
    let source_base = source_unit.wrapping_mul(NATIVE_RECORD_SIZE);
    if source_unit < 0 || source_base < 0 || source_base.wrapping_add(0x1a) > records.len() as i64 {
        return Err(record_bounds_error(source_unit));
    }
    if !(0..8).contains(&source_slot) {
        return Err(battle_err!("native inventory slot {source_slot} is out of bounds"));
    }
    let mut result = NativeItemMPRestoreBatch {
        results: Vec::with_capacity(target_indices.len()),
        skipped_targets: Vec::new(),
        rng_state,
        score: 0,
    };
    for &raw_index in target_indices {
        let unit = raw_index as usize;
        let off = unit * NATIVE_RECORD_SIZE as usize + 0x46;
        if u16::from_le_bytes([records[off], records[off + 1]]) == 0 {
            result.skipped_targets.push(unit as i64);
            continue;
        }
        let entry = apply_native_raw_mp_restore(records, unit as i64, route.amount as i64, result.rng_state)?;
        result.rng_state = entry.rng_state;
        result.score = result.score.wrapping_add(entry.score);
        result.results.push(entry);
    }
    // Removal cannot fail after the preflight above.
    remove_native_inventory_slot(records, source_unit, source_slot)?;
    Ok(result)
}

impl NativeItemMPRestoreRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemMPRestoreRoute",
            &[("ItemType", GoV::U(self.item_type as u64)), ("Amount", GoV::U(self.amount as u64))],
        )
    }
}

#[cfg(test)]
mod native_item_mp_restore_test;
