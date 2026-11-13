//! 移植自 `remake/internal/battle/native_equipment_test.go`。

use super::*;
use crate::test_support::put16;

const RECORD: usize = NATIVE_EQUIPMENT_RECORD_SIZE as usize;
const ROW: usize = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;
const MASK: u8 = NATIVE_EQUIPPED_MASK as u8;

fn totals(record: &[u8]) -> [i16; 4] {
    [
        native_signed_word(record, 0x48),
        native_signed_word(record, 0x4a),
        native_signed_word(record, 0x4c),
        native_signed_word(record, 0x4e),
    ]
}

/// Go：`TestApplyNativeEquipmentRecalcUsesRawFlagsRowsAndWordWrap`
#[test]
fn apply_native_equipment_recalc_uses_raw_flags_rows_and_word_wrap() {
    let mut record = vec![0u8; RECORD];
    put16(&mut record, 0x37, 10i16 as u16);
    put16(&mut record, 0x39, 0xfffe);
    put16(&mut record, 0x3e, 7i16 as u16);
    // slot 0 selected, slot 1 has an item but no equipped bit, slot 2 selected.
    (record[0x0a], record[0x0b]) = (MASK, 1);
    (record[0x0c], record[0x0d]) = (0, 2);
    (record[0x0e], record[0x0f]) = (MASK, 2);
    let mut table = vec![0u8; 0x100 * ROW];
    let mut set = |item: i64, field: usize, value: i16| {
        let row = native_item_effect_row_offset(item).unwrap() as usize;
        put16(&mut table, row + field, value as u16);
    };
    set(1, 1, 5);
    set(1, 5, -3);
    set(1, 3, 2);
    set(1, 7, 1);
    set(2, 1, 0x7fff);
    set(2, 5, 1);
    set(2, 3, -2);
    set(2, 7, 4);
    apply_native_equipment_recalc(&mut record, &table).unwrap();
    let want_ap = 0x7fffi16.wrapping_add(15);
    assert_eq!(totals(&record), [want_ap, -4, 7, 12], "raw totals");
}

/// Go：`TestApplyNativeEquipmentRecalcRejectsMissingRowAtomically`
#[test]
fn apply_native_equipment_recalc_rejects_missing_row_atomically() {
    let mut record = vec![0u8; RECORD];
    (record[0x0a], record[0x0b]) = (MASK, 0xff);
    let before = record.clone();
    assert!(apply_native_equipment_recalc(&mut record, &[0u8; ROW]).is_err(), "missing item row accepted");
    assert_eq!(record, before, "failed raw recalc mutated record");
}

/// Go：`TestApplyNativeEquipmentRecalcIgnoresUnequippedItems`
#[test]
fn apply_native_equipment_recalc_ignores_unequipped_items() {
    let mut record = vec![0u8; RECORD];
    record[0x0b] = 0xff; // no flag bit: table need not contain item 0xff row
    apply_native_equipment_recalc(&mut record, &[0u8; ROW]).unwrap();
    assert!(
        native_signed_word(&record, 0x48) == 0 && native_signed_word(&record, 0x4a) == 0,
        "unequipped item changed raw totals"
    );
}

/// Go：`TestApplyNativeRuntimeEquipmentRecalcUsesTransientModifiers`
#[test]
fn apply_native_runtime_equipment_recalc_uses_transient_modifiers() {
    let mut record = vec![0u8; RECORD];
    put16(&mut record, 0x37, 20);
    put16(&mut record, 0x39, 0xffec);
    put16(&mut record, 0x3e, 7);
    (record[0x22], record[0x23], record[0x24]) = (1, 1, 1);
    (record[0x0a], record[0x0b]) = (MASK, 1);
    let mut table = vec![0u8; 2 * ROW];
    let row = native_item_effect_row_offset(1).unwrap() as usize;
    put16(&mut table, row + 1, 3);
    put16(&mut table, row + 5, 4);
    put16(&mut table, row + 3, 5);
    put16(&mut table, row + 7, 6);
    apply_native_runtime_equipment_recalc(&mut record, &table).unwrap();
    // The exact double 1.15 is slightly below the decimal value. x87 extended
    // precision plus toward-zero rounding therefore maps 23 to 26 and -16 to
    // -18 here; a host binary64 shortcut would hide this boundary behavior.
    assert_eq!(totals(&record), [26, -18, 27, 28], "runtime raw totals");
}

/// Go：`TestNativeScale115PreservesX87TowardZeroBoundary`
#[test]
fn native_scale115_preserves_x87_toward_zero_boundary() {
    for (value, want) in [(20, 22), (-20, -22), (100, 114), (-100, -114), (0, 0)] {
        assert_eq!(native_scale115_toward_zero(value), want, "native scale({value})");
    }
}

/// Go：`TestApplyNativeRuntimeEquipmentRecalcRejectsMissingRowAtomically`
#[test]
fn apply_native_runtime_equipment_recalc_rejects_missing_row_atomically() {
    let mut record = vec![0u8; RECORD];
    (record[0x0a], record[0x0b]) = (MASK, 0xff);
    (record[0x22], record[0x23], record[0x24]) = (1, 1, 1);
    let before = record.clone();
    assert!(
        apply_native_runtime_equipment_recalc(&mut record, &[0u8; ROW]).is_err(),
        "missing runtime item row accepted"
    );
    assert_eq!(record, before, "failed runtime recalc mutated record");
}
