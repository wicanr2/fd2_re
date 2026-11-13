//! Go 來源：`remake/internal/battle/native_item_marker_clear_restore.go`（62 行）。

use super::*;

/// NativeItemMarkerClearRestoreRoute closes type 6/7 around 0x22af6.
/// Nonzero target markers are cleared after a base-10 HP restore, and the
/// dispatcher consumes the source slot. Player-facing status names are unknown.
///
/// Go：`native_item_marker_clear_restore.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemMarkerClearRestoreRoute {
    pub item_type: u8,
    pub presentation_code: i64,
    pub marker_offset: i64,
    pub hp_restore_base: i64,
    pub consumes_source: bool,
}

/// Go：`native_item_marker_clear_restore.go:16`，`func NativeItemMarkerClearRestoreRouteForType(itemType byte) (NativeItemMarkerClearRestoreRoute, bool)`
pub fn native_item_marker_clear_restore_route_for_type(item_type: u8) -> (NativeItemMarkerClearRestoreRoute, bool) {
    let (presentation_code, marker_offset) = match item_type {
        6 => (20, 0x25),
        7 => (21, 0x26),
        _ => return (NativeItemMarkerClearRestoreRoute::default(), false),
    };
    (
        NativeItemMarkerClearRestoreRoute {
            item_type,
            presentation_code,
            marker_offset,
            hp_restore_base: 10,
            consumes_source: true,
        },
        true,
    )
}

/// Go：`native_item_marker_clear_restore.go:33`，`func ApplyNativeItemMarkerClearRestore( records []byte, targetIndices []byte, route NativeItemMarkerClearRestoreRoute, rngState uint16, sourceUnit, sourceSlot int, ) ([]NativeRawFlagRestoreResult, uint16, int, error)`
pub fn apply_native_item_marker_clear_restore(
    records: &mut [u8],
    target_indices: &[u8],
    route: NativeItemMarkerClearRestoreRoute,
    rng_state: u16,
    source_unit: i64,
    source_slot: i64,
) -> Result<(Vec<NativeRawFlagRestoreResult>, u16, i64)> {
    let (verified, ok) = native_item_marker_clear_restore_route_for_type(route.item_type);
    if !ok || verified != route {
        return Err(battle_err!("native item marker-clear route is invalid: {}", route.go_sharp_v()));
    }
    let source_base = source_unit.wrapping_mul(NATIVE_RECORD_SIZE);
    if source_unit < 0 || source_base < 0 || source_base.wrapping_add(0x1a) > records.len() as i64 {
        return Err(record_bounds_error(source_unit));
    }
    if !(0..8).contains(&source_slot) {
        return Err(battle_err!("native inventory slot {source_slot} is out of bounds"));
    }
    let (results, next_rng, score) =
        apply_native_raw_flag_restore(records, target_indices, route.marker_offset, rng_state)?;
    // Removal cannot fail after the preflight above.
    remove_native_inventory_slot(records, source_unit, source_slot)?;
    Ok((results, next_rng, score))
}

impl NativeItemMarkerClearRestoreRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemMarkerClearRestoreRoute",
            &[
                ("ItemType", GoV::U(self.item_type as u64)),
                ("PresentationCode", GoV::I(self.presentation_code)),
                ("MarkerOffset", GoV::I(self.marker_offset)),
                ("HPRestoreBase", GoV::I(self.hp_restore_base)),
                ("ConsumesSource", GoV::B(self.consumes_source)),
            ],
        )
    }
}

#[cfg(test)]
mod native_item_marker_clear_restore_test;
