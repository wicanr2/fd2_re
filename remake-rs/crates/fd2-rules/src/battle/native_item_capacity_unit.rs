//! Go 來源：`remake/internal/battle/native_item_capacity_unit.go`（56 行）。

use super::*;

/// ApplyNativeItemCapacityToUnit projects the closed type17/18/19 0x21082
/// transaction onto editable Unit fields. Max HP/MP use native signed-word
/// wrapping while leaving current HP/MP unchanged. MV uses the low byte of the
/// +0x3b word add and preserves EXP exactly, matching the caller save/restore.
///
/// Go：`native_item_capacity_unit.go:9`，`func ApplyNativeItemCapacityToUnit( target, source *Unit, route NativeItemCapacityStepRoute, sourceRawSlot int, ) (int, error)`
/// 所有權：`units` 是 Rust 新增的參數。Go 允許這幾個 `*Unit` 指向同一個單位（例如對自己使用道具），Rust 改傳同一陣列的索引。
/// 所有權：`target`、`source` 是 `units` 的索引（Go 傳 `*Unit`）。
pub fn apply_native_item_capacity_to_unit(
    units: &mut [Unit],
    target: Option<UnitIdx>,
    source: Option<UnitIdx>,
    route: NativeItemCapacityStepRoute,
    source_raw_slot: i64,
) -> Result<i64> {
    let (verified, ok) = native_item_capacity_step_route_for_type(route.item_type, route.amount);
    if !ok || verified != route {
        return Err(battle_err!("native item capacity route is invalid: {}", route.go_sharp_v()));
    }
    // Go 的 nil 指標對應 `None`；索引超出 `units` 視同 nil。
    let (Some(target), Some(source)) = (target.filter(|&i| i < units.len()), source.filter(|&i| i < units.len()))
    else {
        return Err(battle_err!("native item capacity unit is missing"));
    };
    let compact = native_item_source_compact_index(Some(&units[source]), source_raw_slot)?;

    let unit = &units[target];
    let next = if route.stat == NATIVE_ITEM_MAX_HP {
        (unit.max_hp as i16 as u16).wrapping_add(route.amount) as i16 as i64
    } else if route.stat == NATIVE_ITEM_MAX_MP {
        (unit.max_mp as i16 as u16).wrapping_add(route.amount) as i16 as i64
    } else if route.stat == NATIVE_ITEM_MV {
        if unit.mv < 0 || unit.mv > 0xff {
            return Err(battle_err!("native item MV byte is out of range"));
        }
        (unit.mv as u8).wrapping_add(route.amount as u8) as i64
    } else {
        return Err(battle_err!("native item capacity stat {} is unsupported", go_quote(route.stat.0)));
    };

    let unit = &mut units[target];
    if route.stat == NATIVE_ITEM_MAX_HP {
        unit.max_hp = next;
    } else if route.stat == NATIVE_ITEM_MAX_MP {
        unit.max_mp = next;
    } else if route.stat == NATIVE_ITEM_MV {
        unit.mv = next;
        if unit.equipment_base_set {
            unit.base_mv = next;
        }
    }
    if !remove_native_compact_inventory_inner(Some(&mut units[source]), compact) {
        return Err(battle_err!("native item source removal failed after preflight"));
    }
    Ok(next)
}

#[cfg(test)]
mod native_item_capacity_unit_test;
