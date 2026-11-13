//! Go 來源：`remake/internal/battle/native_item_base_stat_unit.go`（80 行）。

use super::*;

/// ApplyNativeItemBaseStatDeltaToUnit is the editable Unit projection of the
/// closed 0x21082 transaction used by item types 8/9/10. It preserves native
/// 16-bit wrapping and 0x1b8e7 compact source removal. AP/DP require an
/// initialized equipment base; DX is already the independently established raw
/// +0x3e field. The caller remains responsible for 0x1145a recomputation.
///
/// Go：`native_item_base_stat_unit.go:10`，`func ApplyNativeItemBaseStatDeltaToUnit( target, source *Unit, route NativeItemWordDeltaRoute, delta uint16, sourceRawSlot int, ) (int, error)`
/// 所有權：`units` 是 Rust 新增的參數。Go 允許這幾個 `*Unit` 指向同一個單位（例如對自己使用道具），Rust 改傳同一陣列的索引。
/// 所有權：`target`、`source` 是 `units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_item_base_stat_delta_to_unit(
    units: &mut [Unit],
    target: Option<UnitIdx>,
    source: Option<UnitIdx>,
    route: NativeItemWordDeltaRoute,
    delta: u16,
    source_raw_slot: i64,
) -> Result<i64> {
    let (verified, ok) = native_item_word_delta_route_for_type(route.item_type);
    if !ok || verified != route {
        return Err(battle_err!("native item base-stat route is invalid: {}", go_sharp_v_word_delta_route(&route)));
    }
    let (Some(target), Some(source)) = (target.filter(|&i| i < units.len()), source.filter(|&i| i < units.len()))
    else {
        return Err(battle_err!("native item base-stat unit is missing"));
    };
    let compact = native_item_source_compact_index(Some(&units[source]), source_raw_slot)?;

    let unit = &units[target];
    let current = if route.base_stat == NATIVE_ITEM_BASE_AP {
        if !unit.equipment_base_set {
            return Err(battle_err!("native item AP base provenance is unavailable"));
        }
        unit.base_ap
    } else if route.base_stat == NATIVE_ITEM_BASE_DP {
        if !unit.equipment_base_set {
            return Err(battle_err!("native item DP base provenance is unavailable"));
        }
        unit.base_dp
    } else if route.base_stat == NATIVE_ITEM_BASE_DX {
        unit.dx
    } else {
        return Err(battle_err!("native item base stat {} is unsupported", go_quote(route.base_stat.0)));
    };

    let next = (current as i16 as u16).wrapping_add(delta) as i16 as i64;
    let unit = &mut units[target];
    if route.base_stat == NATIVE_ITEM_BASE_AP {
        unit.base_ap = next;
    } else if route.base_stat == NATIVE_ITEM_BASE_DP {
        unit.base_dp = next;
    } else if route.base_stat == NATIVE_ITEM_BASE_DX {
        unit.dx = next;
        (unit.base_hit, unit.base_ev) = (next, next);
    }
    if !remove_native_compact_inventory_inner(Some(&mut units[source]), compact) {
        return Err(battle_err!("native item source removal failed after preflight"));
    }
    Ok(next)
}

/// Go：`native_item_base_stat_unit.go:62`，`func nativeItemSourceCompactIndex(source *Unit, sourceRawSlot int) (int, error)`
pub(crate) fn native_item_source_compact_index(source: Option<&Unit>, source_raw_slot: i64) -> Result<i64> {
    let Some(source) = source.filter(|s| {
        (0..NATIVE_INVENTORY_CELLS).contains(&source_raw_slot)
            && s.inventory_slots.len() as i64 == NATIVE_INVENTORY_CELLS
            && s.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS
            && s.inventory_slots[source_raw_slot as usize] != 0xff
            && s.native_inventory_flags[source_raw_slot as usize] & 0x80 == 0
    }) else {
        return Err(battle_err!("native item source raw slot {source_raw_slot} is unavailable"));
    };
    let compact = source.inventory_slots[..source_raw_slot as usize].iter().filter(|&&s| s != 0xff).count() as i64;
    if compact >= source.inventory.len() as i64 {
        return Err(battle_err!("native item source compact slot is inconsistent"));
    }
    Ok(compact)
}

/// Go `%#v` 對 `NativeItemWordDeltaRoute`（A 組型別）的輸出。
pub(crate) fn go_sharp_v_word_delta_route(route: &NativeItemWordDeltaRoute) -> String {
    go_sharp_v(
        "NativeItemWordDeltaRoute",
        &[
            ("ItemType", GoV::I(route.item_type)),
            ("BaseStat", GoV::S(route.base_stat.0)),
            ("FieldOffset", GoV::I(route.field_offset)),
            ("PresentationCode", GoV::I(route.presentation_code)),
        ],
    )
}

#[cfg(test)]
mod native_item_base_stat_unit_test;
