//! Go 來源：`remake/internal/battle/native_item_marker_application.go`（48 行）。

use super::*;

/// NativeItemMarkerApplicationRoute closes the type-14/type-22 item callers
/// around 0x22d1b. Both retain the source, use a 50-percent RNG gate, apply
/// HP damage with base amount 10 on success, then write a third-RNG marker.
/// The marker's player-facing status name remains unknown.
///
/// Go：`native_item_marker_application.go:9`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemMarkerApplicationRoute {
    pub item_type: u8,
    pub presentation_code: i64,
    pub marker_offset: i64,
    pub hp_damage_base: i64,
    pub consumes_source: bool,
}

/// Go：`native_item_marker_application.go:17`，`func NativeItemMarkerApplicationRouteForType(itemType byte) (NativeItemMarkerApplicationRoute, bool)`
pub fn native_item_marker_application_route_for_type(item_type: u8) -> (NativeItemMarkerApplicationRoute, bool) {
    todo!("battle/native_item_marker_application.go:NativeItemMarkerApplicationRouteForType")
}

/// ApplyNativeItemMarkerApplication executes the proven retained-item mutation.
/// ApplyNativeRawApplication preserves marker/class gates, all three RNG steps,
/// HP damage, list order, 16-bit HP behavior and atomic target preflight.
///
/// Go：`native_item_marker_application.go:37`，`func ApplyNativeItemMarkerApplication( records []byte, targetIndices []byte, route NativeItemMarkerApplicationRoute, rngState uint16, ) ([]NativeRawApplicationResult, uint16, int, error)`
pub fn apply_native_item_marker_application(
    records: &mut [u8],
    target_indices: &[u8],
    route: NativeItemMarkerApplicationRoute,
    rng_state: u16,
) -> Result<(Vec<NativeRawApplicationResult>, u16, i64)> {
    todo!("battle/native_item_marker_application.go:ApplyNativeItemMarkerApplication")
}
