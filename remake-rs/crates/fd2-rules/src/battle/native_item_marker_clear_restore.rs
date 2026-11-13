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
    todo!("battle/native_item_marker_clear_restore.go:NativeItemMarkerClearRestoreRouteForType")
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
    todo!("battle/native_item_marker_clear_restore.go:ApplyNativeItemMarkerClearRestore")
}
