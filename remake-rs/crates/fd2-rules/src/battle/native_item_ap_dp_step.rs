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
    match item_type {
        15 => (
            NativeItemAPDPStepRoute {
                item_type,
                derived_stat: NATIVE_ITEM_DERIVED_DP,
                marker_offset: 0x23,
                word_offset: 0x4a,
                consumes_source: false,
            },
            true,
        ),
        16 => (
            NativeItemAPDPStepRoute {
                item_type,
                derived_stat: NATIVE_ITEM_DERIVED_AP,
                marker_offset: 0x22,
                word_offset: 0x48,
                consumes_source: false,
            },
            true,
        ),
        _ => (NativeItemAPDPStepRoute::default(), false),
    }
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
    let (verified, ok) = native_item_apdp_step_route_for_type(route.item_type);
    if !ok || verified != route {
        return Err(battle_err!("native item AP/DP route is invalid: {}", route.go_sharp_v()));
    }
    apply_native_raw_word_step_at_offsets(records, unit_indices, rng_state, route.marker_offset, route.word_offset)
}

impl NativeItemAPDPStepRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemAPDPStepRoute",
            &[
                ("ItemType", GoV::U(self.item_type as u64)),
                ("DerivedStat", GoV::S(self.derived_stat.0)),
                ("MarkerOffset", GoV::I(self.marker_offset)),
                ("WordOffset", GoV::I(self.word_offset)),
                ("ConsumesSource", GoV::B(self.consumes_source)),
            ],
        )
    }
}

#[cfg(test)]
mod native_item_ap_dp_step_test;
