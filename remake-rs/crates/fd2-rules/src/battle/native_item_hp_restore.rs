//! Go 來源：`remake/internal/battle/native_item_hp_restore.go`（65 行）。

use super::*;

/// NativeItemHPRestoreRoute is the closed type-5/type-13 branch contract in
/// 0x20c6f. Both routes call 0x211a4 with row word +0x0e as the HP restore
/// amount. Only type 5 continues through 0x1b8e7 and consumes the source slot;
/// type 13 retains it.
///
/// Go：`native_item_hp_restore.go:9`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemHPRestoreRoute {
    pub item_type: u8,
    pub amount: u16,
    pub consumes_source: bool,
}

/// Go：`native_item_hp_restore.go:15`，`func NativeItemHPRestoreRouteForType(itemType byte, amount uint16) (NativeItemHPRestoreRoute, bool)`
pub fn native_item_hp_restore_route_for_type(item_type: u8, amount: u16) -> (NativeItemHPRestoreRoute, bool) {
    match item_type {
        5 => (NativeItemHPRestoreRoute { item_type, amount, consumes_source: true }, true),
        13 => (NativeItemHPRestoreRoute { item_type, amount, consumes_source: false }, true),
        _ => (NativeItemHPRestoreRoute::default(), false),
    }
}

/// ApplyNativeItemHPRestore executes the proven mutation transaction without
/// renderer/SFX callbacks. Target-list bounds and the optional source removal
/// are preflighted before any HP is changed.
///
/// Go：`native_item_hp_restore.go:29`，`func ApplyNativeItemHPRestore( records []byte, targetIndices []byte, route NativeItemHPRestoreRoute, rngState uint16, sourceUnit, sourceSlot int, ) (NativeRawRestoreBatch, error)`
pub fn apply_native_item_hp_restore(
    records: &mut [u8],
    target_indices: &[u8],
    route: NativeItemHPRestoreRoute,
    rng_state: u16,
    source_unit: i64,
    source_slot: i64,
) -> Result<NativeRawRestoreBatch> {
    let (verified, ok) = native_item_hp_restore_route_for_type(route.item_type, route.amount);
    if !ok || verified != route {
        return Err(battle_err!("native item HP restore route is invalid: {}", route.go_sharp_v()));
    }
    for &raw_index in target_indices {
        if raw_index as i64 >= records.len() as i64 / NATIVE_RECORD_SIZE {
            return Err(record_bounds_error(raw_index as i64));
        }
    }
    if route.consumes_source {
        let base = source_unit.wrapping_mul(NATIVE_RECORD_SIZE);
        if source_unit < 0 || base < 0 || base.wrapping_add(0x1a) > records.len() as i64 {
            return Err(record_bounds_error(source_unit));
        }
        if !(0..8).contains(&source_slot) {
            return Err(battle_err!("native inventory slot {source_slot} is out of bounds"));
        }
    }
    let result = apply_native_raw_hp_restore_list(records, target_indices, route.amount as i64, rng_state)?;
    if route.consumes_source {
        // Removal cannot fail after the preflight above.
        remove_native_inventory_slot(records, source_unit, source_slot)?;
    }
    Ok(result)
}

impl NativeItemHPRestoreRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemHPRestoreRoute",
            &[
                ("ItemType", GoV::U(self.item_type as u64)),
                ("Amount", GoV::U(self.amount as u64)),
                ("ConsumesSource", GoV::B(self.consumes_source)),
            ],
        )
    }
}

#[cfg(test)]
mod native_item_hp_restore_test;
