//! Go 來源：`remake/internal/battle/native_future_constructor.go`（155 行）。

use super::*;

/// NativeFutureConstructorBase is the table-derived subset written by
/// 0x10d7f..0x10ebd before inventory/equipment recomputation. Values retain
/// the original 16-bit write boundary.
///
/// Go：`native_future_constructor.go:11`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeFutureConstructorBase {
    pub record_race: u8,
    pub record_class: u8,
    pub base_ap: u16,
    pub base_dp: u16,
    pub base_dx: u16,
    pub mobility: u8,
    pub max_hp: u16,
    pub max_mp: u16,
}

/// DecodeNativeFutureConstructorBase transcribes the two b1-selected table
/// branches in 0x10d99..0x10ebd. It deliberately stops before 0x1b750:
/// effective AP/DP/HIT/EV require the separately verified inventory table and
/// are not inferred here.
///
/// Go：`native_future_constructor.go:26`，`func DecodeNativeFutureConstructorBase( table *NativeConstructorTable, level byte, ) (NativeFutureConstructorBase, error)`
pub fn decode_native_future_constructor_base(
    table: Option<&NativeConstructorTable>,
    level: u8,
) -> Result<NativeFutureConstructorBase> {
    // Go 先對可能為 nil 的 table 呼叫 validate（nil 時回 nil），再檢查 nil；Rust 合併成一步。
    let Some(table) = table else {
        return Err(battle_err!("native future constructor table unavailable"));
    };
    table.validate()?;
    let record = &table.record;
    let word = |at: usize| u16::from_le_bytes([record[at], record[at + 1]]);
    if table.branch == "high_class" {
        return Ok(NativeFutureConstructorBase {
            record_race: record[0],
            record_class: record[1],
            base_ap: mul_word(record[5], level),
            base_dp: mul_word(record[6], level),
            base_dx: mul_word(record[7], level),
            mobility: record[8],
            max_hp: (word(2) as u32).wrapping_mul(level as u32) as u16,
            max_mp: mul_word(record[4], level),
        });
    }
    let aux = &table.aux_record;
    let level_minus_one = level as i32 - 1;
    Ok(NativeFutureConstructorBase {
        record_race: record[0],
        record_class: record[1],
        base_ap: add_scaled_word(word(0x12), aux[0], level as i32),
        base_dp: add_scaled_word(word(0x14), aux[2], level as i32),
        base_dx: add_scaled_word(word(0x16), aux[4], level as i32),
        mobility: record[7],
        max_hp: add_scaled_word(word(3), aux[6], level_minus_one),
        max_mp: add_scaled_word(word(5), aux[8], level_minus_one),
    })
}

/// MaterializeNativeFutureConstructor applies the typed fields closed by
/// 0x10c50's table/inventory tail and its 0x1b750 call. It is intentionally a
/// Unit projection rather than a claim of byte-identical 0x50-byte storage:
/// fields without a typed consumer remain in their existing raw provenance.
/// Every source and selected item row is validated before unit is replaced.
///
/// Go：`native_future_constructor.go:87`，`func MaterializeNativeFutureConstructor(unit *Unit, itemTable []byte) error`
pub fn materialize_native_future_constructor(unit: Option<&mut Unit>, item_table: &[u8]) -> Result<()> {
    let Some(unit) = unit.filter(|u| u.native_constructor.is_some()) else {
        return Err(battle_err!("native future constructor provenance unavailable"));
    };
    if unit.lv < 0 || unit.lv > 0xff {
        return Err(battle_err!("native future constructor level {} outside byte", unit.lv));
    }
    if !unit.has_native_record_byte6 || unit.native_record_byte6 > 2 {
        return Err(battle_err!("native future constructor raw camp unavailable"));
    }
    validate_native_inventory_projection(Some(unit)).map_err(|e| e.wrap("native future constructor inventory: "))?;
    let base = decode_native_future_constructor_base(unit.native_constructor.as_ref(), unit.lv as u8)?;
    let mut records = native_inventory_records(std::slice::from_ref(unit), 1)?;
    let record = &mut records[..NATIVE_EQUIPMENT_RECORD_SIZE as usize];
    native_put_word(record, 0x37, base.base_ap as i16);
    native_put_word(record, 0x39, base.base_dp as i16);
    native_put_word(record, 0x3e, base.base_dx as i16);
    // 0x10f6b clears +0x22..+0x27 before 0x11003 calls 0x1b750.
    apply_native_runtime_equipment_recalc(record, item_table)?;

    let mut staged = unit.clone();
    (staged.native_record_race, staged.has_native_record_race) = (base.record_race, true);
    (staged.native_record_class, staged.has_native_record_class) = (base.record_class, true);
    (staged.native_record_byte5, staged.has_native_record_byte5) = (0, true);
    staged.native_transient = [0; 6];
    staged.base_ap = base.base_ap as i16 as i64;
    staged.base_dp = base.base_dp as i16 as i64;
    staged.base_hit = base.base_dx as i16 as i64;
    staged.base_ev = base.base_dx as i16 as i64;
    staged.base_mv = base.mobility as i64;
    staged.equipment_base_set = true;
    staged.ap = native_signed_word(record, 0x48) as i64;
    staged.dp = native_signed_word(record, 0x4a) as i64;
    staged.hit = native_signed_word(record, 0x4c) as i64;
    staged.ev = native_signed_word(record, 0x4e) as i64;
    staged.dx = base.base_dx as i16 as i64;
    staged.mv = base.mobility as i64;
    staged.hp = base.max_hp as i16 as i64;
    staged.max_hp = staged.hp;
    staged.mp = base.max_mp as i16 as i64;
    staged.max_mp = staged.mp;
    (staged.native_record_word42, staged.has_native_record_word42) = (base.max_hp, true);
    (staged.native_record_word46, staged.has_native_record_word46) = (base.max_mp, true);
    staged.exp = if staged.native_record_byte6 == 2 { 0.0 } else { 255.0 };
    *unit = staged;
    Ok(())
}

/// Go：`native_future_constructor.go:149`，`func mulWord(value, multiplier byte) uint16`
pub(crate) fn mul_word(value: u8, multiplier: u8) -> u16 {
    (value as u32 * multiplier as u32) as u16
}

/// Go：`native_future_constructor.go:153`，`func addScaledWord(base uint16, value byte, multiplier int32) uint16`
pub(crate) fn add_scaled_word(base: u16, value: u8, multiplier: i32) -> u16 {
    (base as i32).wrapping_add((value as i32).wrapping_mul(multiplier)) as u16
}

#[cfg(test)]
mod native_future_constructor_test;
