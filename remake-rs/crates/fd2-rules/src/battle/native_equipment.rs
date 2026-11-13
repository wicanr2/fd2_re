//! Go 來源：`remake/internal/battle/native_equipment.go`（132 行）。

use super::*;

pub(crate) const NATIVE_EQUIPMENT_RECORD_SIZE: i64 = 0x50;

pub(crate) const NATIVE_INVENTORY_OFFSET: i64 = 0xa;

pub(crate) const NATIVE_INVENTORY_CELLS: i64 = 8;

pub(crate) const NATIVE_EQUIPPED_MASK: i64 = 0x40;

/// Go：`native_equipment.go:16`，`func nativeSignedWord(buf []byte, off int) int16`
pub(crate) fn native_signed_word(buf: &[u8], off: i64) -> i16 {
    let off = off as usize;
    i16::from_le_bytes([buf[off], buf[off + 1]])
}

/// Go：`native_equipment.go:20`，`func nativePutWord(buf []byte, off int, value int16)`
pub(crate) fn native_put_word(buf: &mut [u8], off: i64, value: i16) {
    let off = off as usize;
    buf[off..off + 2].copy_from_slice(&value.to_le_bytes());
}

/// FD2.EXE 0x5018d stores the IEEE-754 double 1.15 exactly as
/// 2589569785738035 / 2251799813685248. 0x1b750 multiplies in x87 extended
/// precision and 0x377a4 rounds toward zero, so a host binary64 multiply is
/// not interchangeable (for example, native maps 20 to 22, not 23).
/// Go 原式：`uint64(2589569785738035)`
pub(crate) const NATIVE_SCALE115_NUMERATOR: u64 = 2589569785738035;

/// Go 原式：`uint64(2251799813685248)`
pub(crate) const NATIVE_SCALE115_DENOMINATOR: u64 = 2251799813685248;

/// Go：`native_equipment.go:33`，`func nativeScale115TowardZero(value int32) int32`
pub(crate) fn native_scale115_toward_zero(value: i32) -> i32 {
    let negative = value < 0;
    let magnitude = (value as i64).unsigned_abs();
    let quotient = (magnitude as u128 * NATIVE_SCALE115_NUMERATOR as u128 / NATIVE_SCALE115_DENOMINATOR as u128) as u64;
    if negative { (quotient as i32).wrapping_neg() } else { quotient as i32 }
}

/// Go：`native_equipment.go:47`，`func nativeEquipmentTotals(record, itemTable []byte, runtimeModifiers bool) ([4]int32, error)`
pub(crate) fn native_equipment_totals(record: &[u8], item_table: &[u8], runtime_modifiers: bool) -> Result<[i32; 4]> {
    if (record.len() as i64) < NATIVE_EQUIPMENT_RECORD_SIZE {
        return Err(battle_err!(
            "native equipment: record={}, want at least {}",
            record.len(),
            go_sharp_x(NATIVE_EQUIPMENT_RECORD_SIZE)
        ));
    }
    if item_table.is_empty() {
        return Err(battle_err!("native equipment: empty item table"));
    }
    let mut items = [0i64; NATIVE_INVENTORY_CELLS as usize];
    for (slot, marker) in items.iter_mut().enumerate() {
        let cell = NATIVE_INVENTORY_OFFSET as usize + slot * 2;
        if record[cell] as i64 & NATIVE_EQUIPPED_MASK == 0 {
            continue;
        }
        let item = record[cell + 1] as i64;
        match native_item_effect_row_offset(item) {
            Ok(row) if row + 8 <= item_table.len() as i64 => {}
            _ => return Err(battle_err!("native equipment: item {} row unavailable", go_sharp_x(item))),
        }
        // zero means unselected; item 0 is represented by 1
        *marker = item + 1;
    }
    let base_dx = native_signed_word(record, 0x3e) as i32;
    let mut totals =
        [native_signed_word(record, 0x37) as i32, native_signed_word(record, 0x39) as i32, base_dx, base_dx];
    if runtime_modifiers && record[0x24] != 0 {
        totals[2] = totals[2].wrapping_add(15);
        totals[3] = totals[3].wrapping_add(15);
    }
    for marker in items {
        if marker == 0 {
            continue;
        }
        let row = native_item_effect_row_offset(marker - 1).unwrap_or_default();
        // The native row accesses are non-contiguous in this order.
        for (i, field) in [1, 5, 3, 7].into_iter().enumerate() {
            totals[i] = totals[i].wrapping_add(native_signed_word(item_table, row + field) as i32);
        }
    }
    if runtime_modifiers && record[0x22] != 0 {
        totals[0] = native_scale115_toward_zero(totals[0]);
    }
    if runtime_modifiers && record[0x23] != 0 {
        totals[1] = native_scale115_toward_zero(totals[1]);
    }
    Ok(totals)
}

/// ApplyNativeEquipmentRecalc reproduces the proven arithmetic tail of
/// 0x1145a. record is one native 0x50-byte persistent record; itemTable starts
/// at the relocated 0x602ad row table. The eight inventory cells are raw
/// [flag,item] pairs at +0x0a. Only flag bit 0x40 selects a row. Cross-checking
/// all 215 known raw rows against item.json identifies row +1/+3/+5/+7 as the
/// AP/HIT/DP/EV equipment words. Native accumulation order is AP/DP/HIT/EV into
/// record +0x48/+0x4a/+0x4c/+0x4e. All word writes wrap at 16 bits.
///
/// The operation is fail-closed and atomic: malformed record/table input is
/// rejected before any destination byte is changed.
///
/// Go：`native_equipment.go:107`，`func ApplyNativeEquipmentRecalc(record, itemTable []byte) error`
pub fn apply_native_equipment_recalc(record: &mut [u8], item_table: &[u8]) -> Result<()> {
    let totals = native_equipment_totals(record, item_table, false)?;
    for (i, off) in [0x48, 0x4a, 0x4c, 0x4e].into_iter().enumerate() {
        native_put_word(record, off, totals[i] as i16);
    }
    Ok(())
}

/// ApplyNativeRuntimeEquipmentRecalc reproduces 0x1b750 for one 0x50-byte
/// runtime record rooted at [0x53a45]. It differs from persistent 0x1145a:
/// +0x22/+0x23 apply the exact x87 1.15 toward-zero transform to AP/DP, while
/// +0x24 adds 15 to both HIT/EV bases before equipped item words are added.
/// The operation validates every selected item row before changing the record.
///
/// Go：`native_equipment.go:123`，`func ApplyNativeRuntimeEquipmentRecalc(record, itemTable []byte) error`
pub fn apply_native_runtime_equipment_recalc(record: &mut [u8], item_table: &[u8]) -> Result<()> {
    let totals = native_equipment_totals(record, item_table, true)?;
    for (i, off) in [0x48, 0x4a, 0x4c, 0x4e].into_iter().enumerate() {
        native_put_word(record, off, totals[i] as i16);
    }
    Ok(())
}

#[cfg(test)]
mod native_equipment_test;
