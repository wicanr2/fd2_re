//! Go 來源：`remake/internal/battle/native_item_ap_dp_step.go`（56 行）。

use super::*;

/// Go：`native_item_ap_dp_step.go:5`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeItemDerivedStat(pub &'static str);

pub const NATIVE_ITEM_DERIVED_AP: NativeItemDerivedStat = NativeItemDerivedStat("ap");

pub const NATIVE_ITEM_DERIVED_DP: NativeItemDerivedStat = NativeItemDerivedStat("dp");

/// NativeItemAPDPStepRoute closes the type-15/type-16 branches in 0x20c6f.
/// Both retain the source item. Type 15 gates marker +0x23 and increases
/// derived DP +0x4a; type 16 gates +0x22 and increases derived AP +0x48.
///
/// Go：`native_item_ap_dp_step.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemAPDPStepRoute {
    pub item_type: u8,
    pub derived_stat: NativeItemDerivedStat,
    pub marker_offset: i64,
    pub word_offset: i64,
    pub consumes_source: bool,
}

/// Go：`native_item_ap_dp_step.go:23`，`func NativeItemAPDPStepRouteForType(itemType byte) (NativeItemAPDPStepRoute, bool)`
pub fn native_item_apdp_step_route_for_type(item_type: u8) -> (NativeItemAPDPStepRoute, bool) {
    todo!("battle/native_item_ap_dp_step.go:NativeItemAPDPStepRouteForType")
}

/// ApplyNativeItemAPDPStep executes the proven retained-item mutation without
/// presentation callbacks. The shared primitive preserves marker-gated RNG,
/// trunc(current*0.15+1), 16-bit wrapping, list order and atomic preflight.
///
/// Go：`native_item_ap_dp_step.go:43`，`func ApplyNativeItemAPDPStep( records []byte, unitIndices []byte, route NativeItemAPDPStepRoute, rngState uint16, ) ([]NativeRawWordStepResult, uint16, int, error)`
pub fn apply_native_item_apdp_step(
    records: &mut [u8],
    unit_indices: &[u8],
    route: NativeItemAPDPStepRoute,
    rng_state: u16,
) -> Result<(Vec<NativeRawWordStepResult>, u16, i64)> {
    todo!("battle/native_item_ap_dp_step.go:ApplyNativeItemAPDPStep")
}
