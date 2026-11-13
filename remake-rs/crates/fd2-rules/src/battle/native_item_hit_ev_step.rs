//! Go 來源：`remake/internal/battle/native_item_hit_ev_step.go`（35 行）。

use super::*;

/// NativeItemHITEVStepRoute is the closed type-12 branch in 0x20c6f. It calls
/// 0x22997, which gates on marker +0x24 and adds 15 to the derived HIT/EV words
/// +0x4c/+0x4e. The dispatcher retains the source item.
///
/// Go：`native_item_hit_ev_step.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemHITEVStepRoute {
    pub item_type: u8,
    pub increment: u16,
    pub consumes_source: bool,
}

/// Go：`native_item_hit_ev_step.go:14`，`func NativeItemHITEVStepRouteForType(itemType byte) (NativeItemHITEVStepRoute, bool)`
pub fn native_item_hitev_step_route_for_type(item_type: u8) -> (NativeItemHITEVStepRoute, bool) {
    if item_type != 12 {
        return (NativeItemHITEVStepRoute::default(), false);
    }
    (NativeItemHITEVStepRoute { item_type, increment: 15, consumes_source: false }, true)
}

/// ApplyNativeItemHITEVStep executes the proven type-12 mutation without its
/// presentation callbacks. ApplyNativeRawPairStep preserves list order,
/// marker-gated RNG, 16-bit wrapping, and atomic target preflight.
///
/// Go：`native_item_hit_ev_step.go:24`，`func ApplyNativeItemHITEVStep( records []byte, unitIndices []byte, route NativeItemHITEVStepRoute, rngState uint16, ) ([]NativeRawPairStepResult, uint16, int, error)`
pub fn apply_native_item_hitev_step(
    records: &mut [u8],
    unit_indices: &[u8],
    route: NativeItemHITEVStepRoute,
    rng_state: u16,
) -> Result<(Vec<NativeRawPairStepResult>, u16, i64)> {
    let (verified, ok) = native_item_hitev_step_route_for_type(route.item_type);
    if !ok || verified != route {
        return Err(battle_err!("native item HIT/EV route is invalid: {}", route.go_sharp_v()));
    }
    apply_native_raw_pair_step(records, unit_indices, rng_state)
}

impl NativeItemHITEVStepRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemHITEVStepRoute",
            &[
                ("ItemType", GoV::U(self.item_type as u64)),
                ("Increment", GoV::U(self.increment as u64)),
                ("ConsumesSource", GoV::B(self.consumes_source)),
            ],
        )
    }
}

#[cfg(test)]
mod native_item_hit_ev_step_test;
