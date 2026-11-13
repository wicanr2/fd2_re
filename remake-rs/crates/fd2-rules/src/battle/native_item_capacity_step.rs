//! Go 來源：`remake/internal/battle/native_item_capacity_step.go`（75 行）。

use super::*;

/// Go：`native_item_capacity_step.go:8`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct NativeItemCapacityStat(pub &'static str);

pub const NATIVE_ITEM_MAX_HP: NativeItemCapacityStat = NativeItemCapacityStat("max_hp");

pub const NATIVE_ITEM_MAX_MP: NativeItemCapacityStat = NativeItemCapacityStat("max_mp");

pub const NATIVE_ITEM_MV: NativeItemCapacityStat = NativeItemCapacityStat("mv");

/// NativeItemCapacityStepRoute closes type 17–19 around 0x21082. All three
/// consume the source slot. Type 19 uses a word add at +0x3b but its caller
/// restores byte +0x3c, so only MV changes and EXP is preserved.
///
/// Go：`native_item_capacity_step.go:19`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeItemCapacityStepRoute {
    pub item_type: u8,
    pub stat: NativeItemCapacityStat,
    pub field_offset: i64,
    pub amount: u16,
    pub presentation_code: i64,
    pub consumes_source: bool,
}

/// Go：`native_item_capacity_step.go:28`，`func NativeItemCapacityStepRouteForType(itemType byte, amount uint16) (NativeItemCapacityStepRoute, bool)`
pub fn native_item_capacity_step_route_for_type(item_type: u8, amount: u16) -> (NativeItemCapacityStepRoute, bool) {
    todo!("battle/native_item_capacity_step.go:NativeItemCapacityStepRouteForType")
}

/// Go：`native_item_capacity_step.go:50`，`func ApplyNativeItemCapacityStep( records []byte, targetUnit int, route NativeItemCapacityStepRoute, sourceUnit, sourceSlot int, ) (uint16, error)`
pub fn apply_native_item_capacity_step(
    records: &mut [u8],
    target_unit: i64,
    route: NativeItemCapacityStepRoute,
    source_unit: i64,
    source_slot: i64,
) -> Result<u16> {
    todo!("battle/native_item_capacity_step.go:ApplyNativeItemCapacityStep")
}
