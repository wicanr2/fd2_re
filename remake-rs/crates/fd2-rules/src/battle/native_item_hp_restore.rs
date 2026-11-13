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
    todo!("battle/native_item_hp_restore.go:NativeItemHPRestoreRouteForType")
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
    todo!("battle/native_item_hp_restore.go:ApplyNativeItemHPRestore")
}
