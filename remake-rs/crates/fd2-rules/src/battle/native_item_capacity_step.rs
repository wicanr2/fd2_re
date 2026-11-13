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
    let (stat, field_offset, presentation_code) = match item_type {
        17 => (NATIVE_ITEM_MAX_HP, 0x42, 13),
        18 => (NATIVE_ITEM_MAX_MP, 0x46, 13),
        19 => (NATIVE_ITEM_MV, 0x3b, 19),
        _ => return (NativeItemCapacityStepRoute::default(), false),
    };
    (
        NativeItemCapacityStepRoute { item_type, stat, field_offset, amount, presentation_code, consumes_source: true },
        true,
    )
}

/// Go：`native_item_capacity_step.go:50`，`func ApplyNativeItemCapacityStep( records []byte, targetUnit int, route NativeItemCapacityStepRoute, sourceUnit, sourceSlot int, ) (uint16, error)`
pub fn apply_native_item_capacity_step(
    records: &mut [u8],
    target_unit: i64,
    route: NativeItemCapacityStepRoute,
    source_unit: i64,
    source_slot: i64,
) -> Result<u16> {
    let (verified, ok) = native_item_capacity_step_route_for_type(route.item_type, route.amount);
    if !ok || verified != route {
        return Err(battle_err!("native item capacity route is invalid: {}", route.go_sharp_v()));
    }
    let target_base = target_unit.wrapping_mul(NATIVE_RECORD_SIZE);
    if target_unit < 0 || target_base < 0 || target_base.wrapping_add(NATIVE_RECORD_SIZE) > records.len() as i64 {
        return Err(record_bounds_error(target_unit));
    }
    let target_base = target_base as usize;
    let preserved_exp = records[target_base + 0x3c];
    apply_native_word_delta_and_remove(
        records,
        target_unit,
        route.field_offset,
        route.amount as i64,
        source_unit,
        source_slot,
    )?;
    if route.stat == NATIVE_ITEM_MV {
        records[target_base + 0x3c] = preserved_exp;
        return Ok(records[target_base + 0x3b] as u16);
    }
    let off = target_base + route.field_offset as usize;
    Ok(u16::from_le_bytes([records[off], records[off + 1]]))
}

impl NativeItemCapacityStepRoute {
    /// Go `%#v`。
    pub(crate) fn go_sharp_v(&self) -> String {
        go_sharp_v(
            "NativeItemCapacityStepRoute",
            &[
                ("ItemType", GoV::U(self.item_type as u64)),
                ("Stat", GoV::S(self.stat.0)),
                ("FieldOffset", GoV::I(self.field_offset)),
                ("Amount", GoV::U(self.amount as u64)),
                ("PresentationCode", GoV::I(self.presentation_code)),
                ("ConsumesSource", GoV::B(self.consumes_source)),
            ],
        )
    }
}

#[cfg(test)]
mod native_item_capacity_step_test;
