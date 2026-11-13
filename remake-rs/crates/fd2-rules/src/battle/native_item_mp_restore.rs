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
    todo!("battle/native_item_mp_restore.go:NativeItemMPRestoreRouteForType")
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
    todo!("battle/native_item_mp_restore.go:ApplyNativeItemMPRestore")
}
